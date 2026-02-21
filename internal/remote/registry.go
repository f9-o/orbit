// Package remote: node registry — CRUD operations backed by BoltDB via the state package.
package remote

import (
	"fmt"
	"time"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/state"
)

// Registry wraps state.DB for node-specific operations.
type Registry struct {
	db *state.DB
}

// NewRegistry constructs a Registry.
func NewRegistry(db *state.DB) *Registry {
	return &Registry{db: db}
}

// Add registers a new node. Returns an error if the name is already taken.
func (r *Registry) Add(node v1.NodeInfo) error {
	existing, err := r.db.GetNode(node.Spec.Name)
	if err != nil {
		return fmt.Errorf("registry add: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("node %q already registered; use 'orbit nodes rm' first", node.Spec.Name)
	}
	node.Status = v1.NodeOffline
	node.LastSeen = time.Now().UTC()
	return r.db.PutNode(node)
}

// Remove deletes a node from the registry.
func (r *Registry) Remove(name string) error {
	existing, err := r.db.GetNode(name)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("node %q not found", name)
	}
	return r.db.DeleteNode(name)
}

// Get returns the NodeInfo for name, or an error if not found.
func (r *Registry) Get(name string) (v1.NodeInfo, error) {
	info, err := r.db.GetNode(name)
	if err != nil {
		return v1.NodeInfo{}, err
	}
	if info == nil {
		return v1.NodeInfo{}, fmt.Errorf("node %q not registered", name)
	}
	return *info, nil
}

// List returns all registered nodes.
func (r *Registry) List() ([]v1.NodeInfo, error) {
	return r.db.ListNodes()
}

// Trust records the host key fingerprint for a node, enabling strict verification.
func (r *Registry) Trust(name, fingerprint, encodedHostKey string) error {
	info, err := r.Get(name)
	if err != nil {
		return err
	}
	info.KeyFingerprint = fingerprint
	info.HostKey = encodedHostKey
	info.HostKeyKnown = true
	return r.db.PutNode(info)
}

// MarkOnline updates a node's status to Online and resets its fail count.
func (r *Registry) MarkOnline(name string) error {
	return r.db.UpdateNodeStatus(name, v1.NodeOnline, 0)
}

// MarkOffline increments the fail count and marks the node Offline if threshold is reached.
func (r *Registry) MarkOffline(name string, failCount int) error {
	status := v1.NodeDegraded
	if failCount >= 3 {
		status = v1.NodeOffline
	}
	return r.db.UpdateNodeStatus(name, status, failCount)
}
