// Package orchestrator wraps the Docker Engine API for Orbit container operations.
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	networktypes "github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/logger"
)

// Client wraps the Docker API client with Orbit-specific helpers.
type Client struct {
	docker *dockerclient.Client
	log    *logger.Logger
}

// NewClient creates a new Docker API client.
func NewClient(host string, log *logger.Logger) (*Client, error) {
	opts := []dockerclient.Opt{
		dockerclient.WithAPIVersionNegotiation(),
	}
	if host != "" {
		opts = append(opts, dockerclient.WithHost(host))
	} else {
		opts = append(opts, dockerclient.FromEnv)
	}

	dc, err := dockerclient.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &Client{docker: dc, log: log}, nil
}

// Ping verifies Docker daemon connectivity.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.docker.Ping(ctx)
	return err
}

// Close releases the Docker API client resources.
func (c *Client) Close() error {
	return c.docker.Close()
}

// PullImage pulls the specified image and streams progress to the logger.
func (c *Client) PullImage(ctx context.Context, img string) error {
	c.log.Info("pulling image", "image", img)
	rc, err := c.docker.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("image pull %q: %w", img, err)
	}
	defer rc.Close()

	dec := json.NewDecoder(rc)
	for {
		var msg struct {
			Status   string `json:"status"`
			Progress string `json:"progress"`
			Error    string `json:"error"`
		}
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if msg.Error != "" {
			return fmt.Errorf("image pull error: %s", msg.Error)
		}
		if msg.Status != "" {
			c.log.Debug("pull", "status", msg.Status, "progress", msg.Progress)
		}
	}
	return nil
}

// RunContainer creates and starts a container according to spec.
func (c *Client) RunContainer(ctx context.Context, spec v1.ServiceSpec, name string) (string, error) {
	// Build port bindings
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	for _, p := range spec.Ports {
		parts := strings.SplitN(p, ":", 2)
		if len(parts) != 2 {
			continue
		}
		hostPort, containerPortStr := parts[0], parts[1]
		containerPort := nat.Port(containerPortStr + "/tcp")
		exposedPorts[containerPort] = struct{}{}
		portBindings[containerPort] = []nat.PortBinding{{HostPort: hostPort}}
	}

	// Environment slice
	envSlice := make([]string, 0, len(spec.Environment))
	for k, v := range spec.Environment {
		envSlice = append(envSlice, k+"="+v)
	}

	// Restart policy name
	restartPolicyName := containertypes.RestartPolicyMode("unless-stopped")
	if spec.RestartPolicy != "" {
		restartPolicyName = containertypes.RestartPolicyMode(spec.RestartPolicy)
	}

	containerCfg := &containertypes.Config{
		Image:        spec.Image,
		Env:          envSlice,
		Labels:       spec.Labels,
		ExposedPorts: exposedPorts,
	}
	if spec.User != "" {
		containerCfg.User = spec.User
	}

	hostCfg := &containertypes.HostConfig{
		PortBindings:  portBindings,
		Binds:         spec.Volumes,
		RestartPolicy: containertypes.RestartPolicy{Name: restartPolicyName},
	}

	netCfg := &networktypes.NetworkingConfig{}

	resp, err := c.docker.ContainerCreate(ctx, containerCfg, hostCfg, netCfg, nil, name)
	if err != nil {
		return "", fmt.Errorf("container create %q: %w", name, err)
	}

	if err := c.docker.ContainerStart(ctx, resp.ID, containertypes.StartOptions{}); err != nil {
		_ = c.docker.ContainerRemove(ctx, resp.ID, containertypes.RemoveOptions{Force: true})
		return "", fmt.Errorf("container start %q: %w", resp.ID[:12], err)
	}

	c.log.Info("container started", "name", name, "id", resp.ID[:12])
	return resp.ID, nil
}

// StopContainer gracefully stops a container and optionally removes it.
func (c *Client) StopContainer(ctx context.Context, idOrName string, remove bool) error {
	timeout := 10
	stopOpts := containertypes.StopOptions{Timeout: &timeout}

	if err := c.docker.ContainerStop(ctx, idOrName, stopOpts); err != nil {
		return fmt.Errorf("container stop %q: %w", idOrName, err)
	}
	c.log.Info("container stopped", "id", idOrName)

	if remove {
		if err := c.docker.ContainerRemove(ctx, idOrName, containertypes.RemoveOptions{}); err != nil {
			return fmt.Errorf("container remove %q: %w", idOrName, err)
		}
		c.log.Info("container removed", "id", idOrName)
	}
	return nil
}

// InspectContainer returns full container JSON for the given id/name.
func (c *Client) InspectContainer(ctx context.Context, idOrName string) (types.ContainerJSON, error) {
	return c.docker.ContainerInspect(ctx, idOrName)
}

// ListContainers returns running containers matching Orbit labels.
func (c *Client) ListContainers(ctx context.Context, serviceFilter string) ([]types.Container, error) {
	f := filters.NewArgs()
	f.Add("label", "orbit.service")
	if serviceFilter != "" {
		f.Add("label", "orbit.service="+serviceFilter)
	}
	return c.docker.ContainerList(ctx, containertypes.ListOptions{
		Filters: f,
	})
}

// StreamLogs streams container logs to the provided writer.
func (c *Client) StreamLogs(ctx context.Context, idOrName string, follow bool, since time.Duration, w io.Writer) error {
	sinceStr := ""
	if since > 0 {
		sinceStr = fmt.Sprintf("%ds", int(since.Seconds()))
	}
	rc, err := c.docker.ContainerLogs(ctx, idOrName, containertypes.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Timestamps: true,
		Since:      sinceStr,
	})
	if err != nil {
		return fmt.Errorf("logs %q: %w", idOrName, err)
	}
	defer rc.Close()
	_, err = io.Copy(w, rc)
	return err
}

// ContainerStats returns a single stats snapshot for the container.
func (c *Client) ContainerStats(ctx context.Context, idOrName string) (v1.ServiceMetrics, error) {
	resp, err := c.docker.ContainerStatsOneShot(ctx, idOrName)
	if err != nil {
		return v1.ServiceMetrics{}, fmt.Errorf("stats %q: %w", idOrName, err)
	}
	defer resp.Body.Close()

	var raw types.StatsJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return v1.ServiceMetrics{}, err
	}

	// CPU percent calculation
	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(raw.CPUStats.SystemUsage - raw.PreCPUStats.SystemUsage)
	numCPU := float64(len(raw.CPUStats.CPUUsage.PercpuUsage))
	cpuPercent := 0.0
	if sysDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / sysDelta) * numCPU * 100.0
	}

	netStats := raw.Networks["eth0"]
	return v1.ServiceMetrics{
		CPUPercent: cpuPercent,
		MemBytes:   int64(raw.MemoryStats.Usage),
		MemLimit:   int64(raw.MemoryStats.Limit),
		NetRxBytes: int64(netStats.RxBytes),
		NetTxBytes: int64(netStats.TxBytes),
		PIDs:       int(raw.PidsStats.Current),
	}, nil
}
