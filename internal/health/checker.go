// Package health provides multi-protocol health check probes for Orbit services.
package health

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/logger"
)

// DefaultInterval is used when spec.HealthCheck.Interval is zero.
const DefaultInterval = 5 * time.Second

// DefaultTimeout is used when spec.HealthCheck.Timeout is zero.
const DefaultTimeout = 5 * time.Second

// DefaultRetries is used when spec.HealthCheck.Retries is zero.
const DefaultRetries = 3

// Checker dispatches health probes for a ServiceSpec.
type Checker struct {
	log *logger.Logger
}

// NewChecker constructs a Checker.
func NewChecker(log *logger.Logger) *Checker {
	return &Checker{log: log}
}

// Check performs a single health probe for spec and returns nil if healthy.
func (c *Checker) Check(ctx context.Context, spec v1.ServiceSpec, containerID string) error {
	hc := spec.HealthCheck
	if hc == nil {
		return nil // No health check configured — assume healthy
	}

	switch hc.Type {
	case "http":
		return CheckHTTP(ctx, hc.URL, hc.ExpectedCode, hc.Timeout)
	case "tcp":
		host := "localhost"
		return CheckTCP(ctx, host, hc.Port, hc.Timeout)
	case "cmd":
		return CheckCmd(ctx, hc.Command, hc.Timeout)
	default:
		return fmt.Errorf("unknown health check type %q", hc.Type)
	}
}

// WaitHealthy polls the health check until it passes or ctx is cancelled.
// Uses exponential backoff up to the configured interval.
func (c *Checker) WaitHealthy(ctx context.Context, spec v1.ServiceSpec, containerID string) error {
	hc := spec.HealthCheck
	if hc == nil {
		return nil
	}

	interval := hc.Interval
	if interval == 0 {
		interval = DefaultInterval
	}
	retries := hc.Retries
	if retries == 0 {
		retries = DefaultRetries
	}

	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if attempt > 0 {
			// Wait interval between attempts
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}

		lastErr = c.Check(ctx, spec, containerID)
		if lastErr == nil {
			c.log.Info("health check passed", "service", spec.Name, "attempt", attempt+1)
			return nil
		}

		c.log.Debug("health check attempt failed",
			"service", spec.Name,
			"attempt", attempt+1,
			"of", retries+1,
			"err", lastErr,
		)
	}

	return fmt.Errorf("health check failed after %d attempts: %w", retries+1, lastErr)
}

// Probe performs a one-off health check for a service and returns the ServiceStatus.
func (c *Checker) Probe(ctx context.Context, spec v1.ServiceSpec, containerID string) v1.ServiceStatus {
	if err := c.Check(ctx, spec, containerID); err != nil {
		c.log.Debug("health probe unhealthy", "service", spec.Name, "err", err)
		return v1.StatusUnhealthy
	}
	return v1.StatusHealthy
}
