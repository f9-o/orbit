package state_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/state"
	"github.com/f9-o/orbit/pkg/encryption"
)

func TestStateEncryptionAtRest(t *testing.T) {
	// Provide valid master key for init
	os.Setenv(encryption.EnvSecretKey, "12345678901234567890123456789012")
	defer os.Unsetenv(encryption.EnvSecretKey)

	dbPath := filepath.Join(t.TempDir(), "orbit_test.db")
	db, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	// Insert test data
	info := v1.NodeInfo{}
	info.Spec.Name = "worker-1"

	if err := db.PutNode(info); err != nil {
		t.Fatalf("PutNode failed: %v", err)
	}

	// Close to release locks for RAW file check
	db.Close() 

	// 1. Verify that raw file data is NOT stored as plaintext JSON
	rawDb, err := bbolt.Open(dbPath, 0600, &bbolt.Options{ReadOnly: true})
	if err != nil {
		t.Fatalf("Raw open failed: %v", err)
	}
	err = rawDb.View(func(tx *bbolt.Tx) error {
		val := tx.Bucket([]byte("nodes")).Get([]byte("worker-1"))
		if val == nil {
			t.Fatal("Data not found in raw bucket")
		}
		// JSON normally starts with '{', if it does, it implies it wasn't encrypted
		if len(val) > 0 && val[0] == '{' {
			t.Fatalf("SECURITY FLAW: Data is stored in plaintext JSON!")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	rawDb.Close()

	// 2. Open via the state package again to verify read decryption
	db, err = state.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	retrieved, err := db.GetNode("worker-1")
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected node, got nil")
	}

	if retrieved.Spec.Name != info.Spec.Name {
		t.Fatalf("Retrieved node mismatch. Expected %s, got %s", info.Spec.Name, retrieved.Spec.Name)
	}
}
