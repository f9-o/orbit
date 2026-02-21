// Package remote: heartbeat engine — per-node goroutines maintaining live connectivity state.
package remote

import (
	"context"
	"sync"
	"time"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/logger"
)

// HeartbeatInterval is how often each node is probed.
const HeartbeatInterval = 30 * time.Second

// HeartbeatTimeout is the max time allowed for a single probe.
const HeartbeatTimeout = 10 * time.Second

// NodeEvent is emitted on the event channel when a node's status changes.
type NodeEvent struct {
	Node   string
	Status v1.NodeStatus
}

// Engine runs one goroutine per node to maintain heartbeat state.
type Engine struct {
	pool     *Pool
	registry *Registry
	events   chan NodeEvent // external consumers (TUI) read from this
	log      *logger.Logger

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

// NewEngine creates a heartbeat Engine.
// The events channel is buffered; consumers should drain it promptly.
func NewEngine(pool *Pool, registry *Registry, log *logger.Logger) *Engine {
	return &Engine{
		pool:     pool,
		registry: registry,
		events:   make(chan NodeEvent, 64),
		log:      log,
		cancels:  make(map[string]context.CancelFunc),
	}
}

// Events returns the channel on which NodeEvents are published.
func (e *Engine) Events() <-chan NodeEvent {
	return e.events
}

// Watch starts a heartbeat goroutine for the named node (idempotent).
func (e *Engine) Watch(node v1.NodeInfo) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.cancels[node.Spec.Name]; ok {
		return // already watching
	}

	ctx, cancel := context.WithCancel(context.Background())
	e.cancels[node.Spec.Name] = cancel
	go e.watchLoop(ctx, node)
	e.log.Info("heartbeat started", "node", node.Spec.Name)
}

// Unwatch stops the heartbeat goroutine for a node.
func (e *Engine) Unwatch(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if cancel, ok := e.cancels[name]; ok {
		cancel()
		delete(e.cancels, name)
	}
}

// StopAll stops all heartbeat goroutines.
func (e *Engine) StopAll() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for name, cancel := range e.cancels {
		cancel()
		delete(e.cancels, name)
		e.log.Info("heartbeat stopped", "node", name)
	}
}

// watchLoop is the per-node heartbeat goroutine.
func (e *Engine) watchLoop(ctx context.Context, node v1.NodeInfo) {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	failCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			probeCtx, cancel := context.WithTimeout(ctx, HeartbeatTimeout)
			_, _, err := e.pool.Run(probeCtx, node, "echo __orbit_hb__")
			cancel()

			if err != nil {
				failCount++
				e.log.Debug("heartbeat miss", "node", node.Spec.Name, "fail_count", failCount)

				status := v1.NodeDegraded
				if failCount >= 3 {
					status = v1.NodeOffline
				}

				if uerr := e.registry.MarkOffline(node.Spec.Name, failCount); uerr != nil {
					e.log.Warn("heartbeat: state update failed", "err", uerr)
				}

				// Emit event on status transition
				e.emit(NodeEvent{Node: node.Spec.Name, Status: status})
			} else {
				if failCount > 0 {
					// Recovery from degraded state
					e.log.Info("node recovered", "node", node.Spec.Name)
					e.emit(NodeEvent{Node: node.Spec.Name, Status: v1.NodeOnline})
				}
				failCount = 0
				if uerr := e.registry.MarkOnline(node.Spec.Name); uerr != nil {
					e.log.Warn("heartbeat: state update failed", "err", uerr)
				}
			}
		}
	}
}

// emit sends a NodeEvent without blocking (drops if channel full).
func (e *Engine) emit(ev NodeEvent) {
	select {
	case e.events <- ev:
	default:
		e.log.Debug("heartbeat event channel full, dropping event", "node", ev.Node)
	}
}
