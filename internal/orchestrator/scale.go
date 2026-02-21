// Package orchestrator: service scaling.
package orchestrator

import (
	"context"
	"fmt"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/logger"
	"github.com/f9-o/orbit/internal/core/state"
)

// Scaler manages replica counts for services.
type Scaler struct {
	docker *Client
	state  *state.DB
	log    *logger.Logger
}

// NewScaler constructs a Scaler.
func NewScaler(docker *Client, db *state.DB, log *logger.Logger) *Scaler {
	return &Scaler{docker: docker, state: db, log: log}
}

// Scale adjusts the running replica count for a service to target.
// This implementation uses a simple container-per-replica model with indexed names.
func (s *Scaler) Scale(ctx context.Context, spec v1.ServiceSpec, node string, target int) error {
	if target < 0 {
		return fmt.Errorf("replica count must be >= 0")
	}

	current, err := s.state.ListServiceStates(node)
	if err != nil {
		return err
	}

	// Count existing replicas for this service
	var running []v1.ServiceState
	for _, ss := range current {
		if ss.Name == spec.Name {
			running = append(running, ss)
		}
	}

	currentCount := len(running)
	s.log.Info("scale", "service", spec.Name, "current", currentCount, "target", target)

	if currentCount == target {
		s.log.Info("already at target replica count", "service", spec.Name)
		return nil
	}

	// Scale up: start additional containers
	for i := currentCount; i < target; i++ {
		name := fmt.Sprintf("%s-%d", spec.Name, i+1)
		if spec.Labels == nil {
			spec.Labels = map[string]string{}
		}
		spec.Labels["orbit.service"] = spec.Name
		spec.Labels["orbit.replica"] = fmt.Sprintf("%d", i+1)

		id, err := s.docker.RunContainer(ctx, spec, name)
		if err != nil {
			return fmt.Errorf("scale up replica %d: %w", i+1, err)
		}
		s.log.Info("replica started", "name", name, "id", id[:12])
	}

	// Scale down: stop excess containers (from the end)
	for i := currentCount - 1; i >= target; i-- {
		ss := running[i]
		s.log.Info("stopping excess replica", "name", ss.Name, "id", ss.ContainerID[:12])
		if err := s.docker.StopContainer(ctx, ss.ContainerID, true); err != nil {
			s.log.Warn("scale down: stop failed", "err", err)
		}
	}

	return nil
}
