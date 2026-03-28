package state

import (
	"encoding/json"
	"time"

	"go.etcd.io/bbolt"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/pkg/encryption"
	"github.com/f9-o/orbit/pkg/errs"
)

// Bucket names
var (
	bucketNodes       = []byte("nodes")
	bucketServices    = []byte("services")
	bucketDeployments = []byte("deployments")
)

// DB wraps a BoltDB instance with typed accessor methods and encryption handling.
type DB struct {
	bolt   *bbolt.DB
	crypto *encryption.Engine
}

// Open opens (or creates) the state database at the given path.
// It initializes the encryption engine which is required to securely store data.
func Open(path string) (*DB, error) {
	cryptoEngine, err := encryption.NewEngine()
	if err != nil {
		return nil, errs.Wrap(err, errs.ErrInternal, "state.Open.InitCrypto")
	}

	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, errs.New(errs.ErrStateRead, "state.Open", err).WithAdvice("Ensure you have file permissions and no other process holds the DB lock.")
	}

	// Ensure all buckets exist
	err = db.Update(func(tx *bbolt.Tx) error {
		for _, b := range [][]byte{bucketNodes, bucketServices, bucketDeployments} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return errs.New(errs.ErrStateWrite, "state.InitBuckets", err)
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &DB{bolt: db, crypto: cryptoEngine}, nil
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
	err := db.putJSON(bucketNodes, info.Spec.Name, info)
	if err != nil {
		return errs.Wrap(err, errs.ErrStateWrite, "state.PutNode").WithNode(info.Spec.Name)
	}
	return nil
}

// GetNode retrieves a NodeInfo by name. Returns nil, nil if not found.
func (db *DB) GetNode(name string) (*v1.NodeInfo, error) {
	var info v1.NodeInfo
	found, err := db.getJSON(bucketNodes, name, &info)
	if err != nil {
		return nil, errs.Wrap(err, errs.ErrStateRead, "state.GetNode").WithNode(name)
	}
	if !found {
		return nil, nil
	}
	return &info, nil
}

// DeleteNode removes a node record.
func (db *DB) DeleteNode(name string) error {
	err := db.bolt.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketNodes).Delete([]byte(name))
	})
	if err != nil {
		return errs.New(errs.ErrStateWrite, "state.DeleteNode", err).WithNode(name)
	}
	return nil
}

// ListNodes returns all registered nodes.
func (db *DB) ListNodes() ([]v1.NodeInfo, error) {
	var nodes []v1.NodeInfo
	err := db.bolt.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketNodes).ForEach(func(k, v []byte) error {
			var info v1.NodeInfo
			data, err := db.crypto.Decrypt(v)
			if err != nil {
				return errs.New(errs.ErrStateRead, "state.ListNodes.Decrypt", err).WithNode(string(k))
			}
			if err := json.Unmarshal(data, &info); err != nil {
				return errs.New(errs.ErrStateRead, "state.ListNodes.Unmarshal", err).WithNode(string(k))
			}
			nodes = append(nodes, info)
			return nil
		})
	})
	if err != nil {
		return nil, errs.Wrap(err, errs.ErrStateRead, "state.ListNodes")
	}
	return nodes, nil
}

// UpdateNodeStatus updates only the status, last_seen, and fail_count fields.
func (db *DB) UpdateNodeStatus(name string, status v1.NodeStatus, failCount int) error {
	info, err := db.GetNode(name)
	if err != nil {
		return err
	}
	if info == nil {
		return errs.Newf(errs.ErrNodeNotFound, "state.UpdateNodeStatus", "node %q not found", name).WithNode(name)
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
	err := db.putJSON(bucketServices, key, state)
	if err != nil {
		return errs.Wrap(err, errs.ErrStateWrite, "state.PutServiceState").WithNode(key)
	}
	return nil
}

// GetServiceState retrieves a ServiceState. Returns nil, nil if not found.
func (db *DB) GetServiceState(node, name string) (*v1.ServiceState, error) {
	var s v1.ServiceState
	key := node + "/" + name
	found, err := db.getJSON(bucketServices, key, &s)
	if err != nil {
		return nil, errs.Wrap(err, errs.ErrStateRead, "state.GetServiceState").WithNode(key)
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
			data, err := db.crypto.Decrypt(v)
			if err != nil {
				return errs.New(errs.ErrStateRead, "state.ListServiceStates.Decrypt", err).WithNode(string(k))
			}
			if err := json.Unmarshal(data, &s); err != nil {
				return errs.New(errs.ErrStateRead, "state.ListServiceStates.Unmarshal", err).WithNode(string(k))
			}
			if node == "" || s.Node == node {
				states = append(states, s)
			}
			return nil
		})
	})
	if err != nil {
		return nil, errs.Wrap(err, errs.ErrStateRead, "state.ListServiceStates")
	}
	return states, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Deployment history
// ─────────────────────────────────────────────────────────────────────────────

// PutDeployment appends a deployment record to the history.
func (db *DB) PutDeployment(rec v1.DeploymentRecord) error {
	err := db.putJSON(bucketDeployments, rec.ID, rec)
	if err != nil {
		return errs.Wrap(err, errs.ErrStateWrite, "state.PutDeployment").WithNode(rec.ID)
	}
	return nil
}

// ListDeployments returns all deployment records for a given service name.
// Pass empty string to return all deployments.
func (db *DB) ListDeployments(service string) ([]v1.DeploymentRecord, error) {
	var recs []v1.DeploymentRecord
	err := db.bolt.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketDeployments).ForEach(func(k, v []byte) error {
			var r v1.DeploymentRecord
			data, err := db.crypto.Decrypt(v)
			if err != nil {
				return errs.New(errs.ErrStateRead, "state.ListDeployments.Decrypt", err).WithNode(string(k))
			}
			if err := json.Unmarshal(data, &r); err != nil {
				return errs.New(errs.ErrStateRead, "state.ListDeployments.Unmarshal", err).WithNode(string(k))
			}
			if service == "" || r.Service == service {
				recs = append(recs, r)
			}
			return nil
		})
	})
	if err != nil {
		return nil, errs.Wrap(err, errs.ErrStateRead, "state.ListDeployments")
	}
	return recs, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Generic helpers
// ─────────────────────────────────────────────────────────────────────────────

func (db *DB) putJSON(bucket []byte, key string, val any) error {
	data, err := json.Marshal(val)
	if err != nil {
		return errs.New(errs.ErrStateWrite, "state.putJSON.Marshal", err)
	}
	
	encryptedData, err := db.crypto.Encrypt(data)
	if err != nil {
		return errs.New(errs.ErrStateWrite, "state.putJSON.Encrypt", err)
	}

	return db.bolt.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucket).Put([]byte(key), encryptedData)
	})
}

func (db *DB) getJSON(bucket []byte, key string, out any) (bool, error) {
	var found bool
	err := db.bolt.View(func(tx *bbolt.Tx) error {
		encryptedData := tx.Bucket(bucket).Get([]byte(key))
		if encryptedData == nil {
			return nil
		}
		found = true
		
		data, err := db.crypto.Decrypt(encryptedData)
		if err != nil {
			return errs.New(errs.ErrStateRead, "state.getJSON.Decrypt", err)
		}

		if err := json.Unmarshal(data, out); err != nil {
			return errs.New(errs.ErrStateRead, "state.getJSON.Unmarshal", err)
		}
		return nil
	})
	return found, err
}
