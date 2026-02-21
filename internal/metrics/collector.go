// Package metrics polls Docker container stats and exposes them for TUI and Prometheus.
package metrics

import (
	"context"
	"sync"
	"time"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/logger"
	"github.com/f9-o/orbit/internal/orchestrator"
)

// PollInterval is how often metrics are collected.
const PollInterval = 2 * time.Second

// Snapshot holds the most recent metrics for all services on a node.
type Snapshot struct {
	mu   sync.RWMutex
	data v1.Metrics
}

// Get returns a copy of the current metrics snapshot.
func (s *Snapshot) Get() v1.Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

// set atomically updates the snapshot.
func (s *Snapshot) set(m v1.Metrics) {
	s.mu.Lock()
	s.data = m
	s.mu.Unlock()
}

// Collector polls Docker stats continuously and publishes to a Snapshot.
type Collector struct {
	docker    *orchestrator.Client
	node      string
	snapshots map[string]*Snapshot // service name → snapshot
	mu        sync.RWMutex
	log       *logger.Logger
}

// NewCollector constructs a Collector for a given Docker node.
func NewCollector(docker *orchestrator.Client, node string, log *logger.Logger) *Collector {
	return &Collector{
		docker:    docker,
		node:      node,
		snapshots: make(map[string]*Snapshot),
		log:       log,
	}
}

// GetSnapshot returns the Snapshot for a service, creating it if needed.
func (c *Collector) GetSnapshot(service string) *Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.snapshots[service]; !ok {
		c.snapshots[service] = &Snapshot{}
	}
	return c.snapshots[service]
}

// Run starts the collection loop. Blocks until ctx is cancelled.
func (c *Collector) Run(ctx context.Context) {
	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collect(ctx)
		}
	}
}

func (c *Collector) collect(ctx context.Context) {
	containers, err := c.docker.ListContainers(ctx, "")
	if err != nil {
		c.log.Debug("metrics collect: list containers", "err", err)
		return
	}

	for _, ctr := range containers {
		serviceName := ctr.Labels["orbit.service"]
		if serviceName == "" {
			continue
		}

		stats, err := c.docker.ContainerStats(ctx, ctr.ID)
		if err != nil {
			c.log.Debug("metrics collect: stats", "container", ctr.ID[:12], "err", err)
			continue
		}

		snap := c.GetSnapshot(serviceName)
		snap.set(v1.Metrics{
			Timestamp: time.Now().UTC(),
			Node:      c.node,
			Services: map[string]v1.ServiceMetrics{
				serviceName: stats,
			},
		})
	}
}

// AllMetrics returns a combined Metrics snapshot across all known services.
func (c *Collector) AllMetrics() v1.Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	m := v1.Metrics{
		Timestamp: time.Now().UTC(),
		Node:      c.node,
		Services:  make(map[string]v1.ServiceMetrics),
	}
	for name, snap := range c.snapshots {
		data := snap.Get()
		if svc, ok := data.Services[name]; ok {
			m.Services[name] = svc
		}
	}
	return m
}
