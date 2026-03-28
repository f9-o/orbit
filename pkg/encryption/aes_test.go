package encryption_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/f9-o/orbit/pkg/encryption"
)

func TestEngineEncryptDecrypt(t *testing.T) {
	// Set up isolated env for testing
	os.Setenv(encryption.EnvSecretKey, "12345678901234567890123456789012") // valid 32-byte key
	defer os.Unsetenv(encryption.EnvSecretKey)

	eng, err := encryption.NewEngine()
	if err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	plaintext := []byte("secret military grade payload")
	
	ciphertext, err := eng.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if bytes.Equal(plaintext, ciphertext) {
		t.Fatal("Ciphertext must not be identical to plaintext!")
	}

	decrypted, err := eng.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Expected '%s', got '%s'", string(plaintext), string(decrypted))
	}
}

func TestEngineAutoGenerateKey(t *testing.T) {
	tmpHome := t.TempDir()
	os.Setenv("USERPROFILE", tmpHome) // Windows home dir
	os.Setenv("HOME", tmpHome)        // Unix home dir
	os.Unsetenv(encryption.EnvSecretKey)
	
	eng, err := encryption.NewEngine()
	if err != nil {
		t.Fatalf("Failed to generate and load key via temp home file: %v", err)
	}

	keyPath := filepath.Join(tmpHome, ".orbit", encryption.KeyFilename)
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Fatal("Expected key file to be generated at: ", keyPath)
	}

	plaintext := []byte("another payload")
	ciphertext, err := eng.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Auto-gen encrypt failed: %v", err)
	}

	decrypted, err := eng.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Auto-gen decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("Failed decrypt sequence with auto-generated key")
	}
}
