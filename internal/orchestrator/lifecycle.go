// Package orchestrator: service lifecycle — up and down operations.
package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/logger"
	"github.com/f9-o/orbit/internal/core/state"
)

// LifecycleManager handles 'orbit up' and 'orbit down' for a set of services.
type LifecycleManager struct {
	docker *Client
	state  *state.DB
	log    *logger.Logger
}

// NewLifecycleManager constructs a LifecycleManager.
func NewLifecycleManager(docker *Client, db *state.DB, log *logger.Logger) *LifecycleManager {
	return &LifecycleManager{docker: docker, state: db, log: log}
}

// Up ensures all services in specs are running.
// Existing containers with the same name are skipped unless forceRecreate is true.
func (m *LifecycleManager) Up(ctx context.Context, specs []v1.ServiceSpec, node string, forceRecreate bool) error {
	for _, spec := range specs {
		if err := m.upOne(ctx, spec, node, forceRecreate); err != nil {
			return fmt.Errorf("up %q: %w", spec.Name, err)
		}
	}
	return nil
}

func (m *LifecycleManager) upOne(ctx context.Context, spec v1.ServiceSpec, node string, forceRecreate bool) error {
	existing, err := m.state.GetServiceState(node, spec.Name)
	if err != nil {
		return err
	}

	if existing != nil && existing.ContainerID != "" && !forceRecreate {
		// Verify the container is actually running
		info, inspectErr := m.docker.InspectContainer(ctx, existing.ContainerID)
		if inspectErr == nil && info.State.Running {
			m.log.Info("service already running, skipping", "service", spec.Name)
			return nil
		}
	}

	// If forceRecreate or container is not running, stop + remove existing
	if existing != nil && existing.ContainerID != "" {
		_ = m.docker.StopContainer(ctx, existing.ContainerID, true)
	}

	// Add orbit labels
	if spec.Labels == nil {
		spec.Labels = map[string]string{}
	}
	spec.Labels["orbit.service"] = spec.Name
	spec.Labels["orbit.node"] = node
	spec.Labels["orbit.started"] = time.Now().UTC().Format(time.RFC3339)

	id, err := m.docker.RunContainer(ctx, spec, spec.Name)
	if err != nil {
		return err
	}

	return m.state.PutServiceState(v1.ServiceState{
		Name:        spec.Name,
		ContainerID: id,
		Image:       spec.Image,
		Status:      v1.StatusUnknown,
		Node:        node,
		StartedAt:   time.Now().UTC(),
	})
}

// Down stops and removes the specified services (or all if names is empty).
// If removeVolumes is true, named volumes are also removed.
func (m *LifecycleManager) Down(ctx context.Context, node string, names []string, removeVolumes bool) error {
	states, err := m.state.ListServiceStates(node)
	if err != nil {
		return err
	}

	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}

	for _, s := range states {
		if len(names) > 0 && !nameSet[s.Name] {
			continue
		}
		m.log.Info("stopping service", "service", s.Name, "id", s.ContainerID[:12])
		if err := m.docker.StopContainer(ctx, s.ContainerID, true); err != nil {
			m.log.Warn("stop failed", "service", s.Name, "err", err)
		}
	}
	return nil
}
