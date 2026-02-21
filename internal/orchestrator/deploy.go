// Package orchestrator: rolling deploy algorithm with automatic rollback.
package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/logger"
	"github.com/f9-o/orbit/internal/core/state"
	"github.com/f9-o/orbit/internal/health"
	"github.com/f9-o/orbit/pkg/errs"
)

// DeployOptions holds per-deploy overrides.
type DeployOptions struct {
	Tag     string        // image tag override
	Timeout time.Duration // health check timeout per replica
	DryRun  bool
}

// DefaultDeployTimeout is used when no timeout is specified.
const DefaultDeployTimeout = 120 * time.Second

// Deployer orchestrates rolling updates for a single service.
type Deployer struct {
	docker  *Client
	state   *state.DB
	checker *health.Checker
	log     *logger.Logger
}

// NewDeployer constructs a Deployer.
func NewDeployer(docker *Client, db *state.DB, checker *health.Checker, log *logger.Logger) *Deployer {
	return &Deployer{
		docker:  docker,
		state:   db,
		checker: checker,
		log:     log,
	}
}

// Deploy performs a rolling update for spec on the given node.
// If RollbackOnFailure is set and a health check fails, the old container is restarted.
func (d *Deployer) Deploy(ctx context.Context, spec v1.ServiceSpec, node string, opts DeployOptions) error {
	image := spec.Image
	if opts.Tag != "" {
		if idx := lastColonIdx(image); idx != -1 {
			image = image[:idx+1] + opts.Tag
		} else {
			image = image + ":" + opts.Tag
		}
	}

	timeout := DefaultDeployTimeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}
	if spec.Deploy != nil && spec.HealthCheck != nil && spec.HealthCheck.Timeout > 0 {
		timeout = spec.HealthCheck.Timeout * time.Duration(spec.HealthCheck.Retries+2)
	}

	d.log.Info("deploy.start",
		"service", spec.Name, "node", node,
		"image", image, "dry_run", opts.DryRun,
	)

	if opts.DryRun {
		d.log.Info("deploy.dryrun — no changes made", "service", spec.Name)
		return nil
	}

	// Get existing container state
	existing, err := d.state.GetServiceState(node, spec.Name)
	if err != nil {
		return errs.Wrap(err, errs.ErrStateRead, "deploy.getstate")
	}

	// 1. Pull new image
	if err := d.docker.PullImage(ctx, image); err != nil {
		return errs.New(errs.ErrDockerPull, "deploy.pull", err).
			WithNode(node).
			WithAdvice("Check your registry credentials and image name")
	}

	// 2. Start new container with a unique temporary name
	newName := fmt.Sprintf("%s-new-%d", spec.Name, time.Now().Unix())
	newSpec := spec
	newSpec.Image = image
	if newSpec.Labels == nil {
		newSpec.Labels = map[string]string{}
	}
	newSpec.Labels["orbit.service"] = spec.Name
	newSpec.Labels["orbit.node"] = node

	newID, err := d.docker.RunContainer(ctx, newSpec, newName)
	if err != nil {
		return errs.New(errs.ErrDockerRun, "deploy.run", err).WithNode(node)
	}

	// 3. Wait for health check to pass
	if spec.HealthCheck != nil {
		d.log.Info("deploy.healthcheck", "service", spec.Name, "timeout", timeout)

		hctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		if err := d.checker.WaitHealthy(hctx, spec, newID); err != nil {
			d.log.Warn("deploy.healthcheck.failed", "service", spec.Name, "err", err)

			// Stop the new (failed) container
			_ = d.docker.StopContainer(ctx, newID, true)

			// Rollback: restart old image if enabled
			if existing != nil && spec.Deploy != nil && spec.Deploy.RollbackOnFailure {
				d.log.Warn("deploy.rollback", "service", spec.Name, "old_container", existing.ContainerID[:12])
				rollbackSpec := spec
				rollbackSpec.Image = existing.Image
				if _, rollErr := d.docker.RunContainer(ctx, rollbackSpec, spec.Name); rollErr != nil {
					d.log.Warn("deploy.rollback.failed", "err", rollErr)
				}
			}

			return errs.New(errs.ErrServiceHealthFail, "deploy.healthcheck", err).
				WithNode(node).
				WithAdvice(fmt.Sprintf("New container failed health check. Run: orbit logs %s", spec.Name))
		}
	}

	// 4. Stop old container
	if existing != nil && existing.ContainerID != "" {
		d.log.Info("deploy.stop_old", "id", existing.ContainerID[:12])
		if err := d.docker.StopContainer(ctx, existing.ContainerID, true); err != nil {
			d.log.Warn("deploy.stop_old.failed", "err", err)
		}
	}

	// 5. Rename new container to canonical name
	if err := d.docker.docker.ContainerRename(ctx, newID, spec.Name); err != nil {
		d.log.Warn("deploy.rename.failed", "err", err)
	}

	// 6. Persist state
	newState := v1.ServiceState{
		Name:        spec.Name,
		ContainerID: newID,
		Image:       image,
		Status:      v1.StatusHealthy,
		Node:        node,
		StartedAt:   time.Now().UTC(),
	}
	if err := d.state.PutServiceState(newState); err != nil {
		d.log.Warn("deploy.state_persist.failed", "err", err)
	}

	d.log.Info("deploy.complete", "service", spec.Name, "image", image)
	return nil
}

// lastColonIdx finds the last colon in a string (for tag parsing).
func lastColonIdx(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}
