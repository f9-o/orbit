// Package state manages Orbit's persistent state using BoltDB.
// All writes are transactional; reads use read-only transactions to minimise contention.
package state

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	v1 "github.com/f9-o/orbit/api/v1"
)

// Bucket names
var (
	bucketNodes       = []byte("nodes")
	bucketServices    = []byte("services")
	bucketDeployments = []byte("deployments")
)

// DB wraps a BoltDB instance with typed accessor methods.
type DB struct {
	bolt *bbolt.DB
}

// Open opens (or creates) the state database at the given path.
func Open(path string) (*DB, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open state db %q: %w", path, err)
	}

	// Ensure all buckets exist
	err = db.Update(func(tx *bbolt.Tx) error {
		for _, b := range [][]byte{bucketNodes, bucketServices, bucketDeployments} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return fmt.Errorf("create bucket %q: %w", b, err)
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("init buckets: %w", err)
	}

	return &DB{bolt: db}, nil
}

// Close closes the underlying BoltDB file.
func (db *DB) Close() error {
	return db.bolt.Close()
}

// ─────────────────────────────────────────────────────────────────────────────
// Node operations
// ─────────────────────────────────────────────────────────────────────────────

// PutNode upserts a NodeInfo record.
func (db *DB) PutNode(info v1.NodeInfo) error {
	return db.putJSON(bucketNodes, info.Spec.Name, info)
}

// GetNode retrieves a NodeInfo by name. Returns nil, nil if not found.
func (db *DB) GetNode(name string) (*v1.NodeInfo, error) {
	var info v1.NodeInfo
	found, err := db.getJSON(bucketNodes, name, &info)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &info, nil
}

// DeleteNode removes a node record.
func (db *DB) DeleteNode(name string) error {
	return db.bolt.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketNodes).Delete([]byte(name))
	})
}

// ListNodes returns all registered nodes.
func (db *DB) ListNodes() ([]v1.NodeInfo, error) {
	var nodes []v1.NodeInfo
	err := db.bolt.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketNodes).ForEach(func(k, v []byte) error {
			var info v1.NodeInfo
			if err := json.Unmarshal(v, &info); err != nil {
				return fmt.Errorf("unmarshal node %q: %w", k, err)
			}
			nodes = append(nodes, info)
			return nil
		})
	})
	return nodes, err
}

// UpdateNodeStatus updates only the status, last_seen, and fail_count fields.
func (db *DB) UpdateNodeStatus(name string, status v1.NodeStatus, failCount int) error {
	info, err := db.GetNode(name)
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("node %q not found", name)
	}
	info.Status = status
	info.LastSeen = time.Now().UTC()
	info.FailCount = failCount
	return db.PutNode(*info)
}

// ─────────────────────────────────────────────────────────────────────────────
// Service state operations
// ─────────────────────────────────────────────────────────────────────────────

// PutServiceState upserts a ServiceState record.
func (db *DB) PutServiceState(state v1.ServiceState) error {
	key := state.Node + "/" + state.Name
	return db.putJSON(bucketServices, key, state)
}

// GetServiceState retrieves a ServiceState. Returns nil, nil if not found.
func (db *DB) GetServiceState(node, name string) (*v1.ServiceState, error) {
	var s v1.ServiceState
	key := node + "/" + name
	found, err := db.getJSON(bucketServices, key, &s)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &s, nil
}

// ListServiceStates returns all service states, optionally filtered by node.
func (db *DB) ListServiceStates(node string) ([]v1.ServiceState, error) {
	var states []v1.ServiceState
	err := db.bolt.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketServices).ForEach(func(k, v []byte) error {
			var s v1.ServiceState
			if err := json.Unmarshal(v, &s); err != nil {
				return err
			}
			if node == "" || s.Node == node {
				states = append(states, s)
			}
			return nil
		})
	})
	return states, err
}

// ─────────────────────────────────────────────────────────────────────────────
// Deployment history
// ─────────────────────────────────────────────────────────────────────────────

// PutDeployment appends a deployment record to the history.
func (db *DB) PutDeployment(rec v1.DeploymentRecord) error {
	return db.putJSON(bucketDeployments, rec.ID, rec)
}

// ListDeployments returns all deployment records for a given service name.
// Pass empty string to return all deployments.
func (db *DB) ListDeployments(service string) ([]v1.DeploymentRecord, error) {
	var recs []v1.DeploymentRecord
	err := db.bolt.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketDeployments).ForEach(func(k, v []byte) error {
			var r v1.DeploymentRecord
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			if service == "" || r.Service == service {
				recs = append(recs, r)
			}
			return nil
		})
	})
	return recs, err
}

// ─────────────────────────────────────────────────────────────────────────────
// Generic helpers
// ─────────────────────────────────────────────────────────────────────────────

func (db *DB) putJSON(bucket []byte, key string, val any) error {
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return db.bolt.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucket).Put([]byte(key), data)
	})
}

func (db *DB) getJSON(bucket []byte, key string, out any) (bool, error) {
	var found bool
	err := db.bolt.View(func(tx *bbolt.Tx) error {
		data := tx.Bucket(bucket).Get([]byte(key))
		if data == nil {
			return nil
		}
		found = true
		return json.Unmarshal(data, out)
	})
	return found, err
}
