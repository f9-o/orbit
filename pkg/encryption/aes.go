package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"github.com/f9-o/orbit/pkg/errs"
)

const (
	// EnvSecretKey is the environment variable for providing the master key.
	EnvSecretKey = "ORBIT_SECRET_KEY"
	// KeyFilename is the default key file generated if no environment variable is given.
	KeyFilename = ".master.key"
)

// ErrEncryption is the custom error code for encryption failures.
const ErrEncryption errs.ErrorCode = "ERR-CRYPTO-001"

// Engine handles AES-256-GCM encryption and decryption.
type Engine struct {
	aead cipher.AEAD
}

// NewEngine initializes the secure encryption engine.
// It loads a 32-byte master key from ORBIT_SECRET_KEY environment variable,
// or reads/generates a safe key in ~/.orbit/.master.key.
func NewEngine() (*Engine, error) {
	key, err := loadOrGenerateKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errs.New(ErrEncryption, "encryption.InitCipher", err).
			WithAdvice("Ensure the master key is exactly 32 bytes for AES-256.")
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errs.New(ErrEncryption, "encryption.InitGCM", err)
	}

	return &Engine{aead: aead}, nil
}

func loadOrGenerateKey() ([]byte, error) {
	// 1. Check Env
	envKey := os.Getenv(EnvSecretKey)
	if envKey != "" {
		key, err := hex.DecodeString(envKey)
		if err == nil && len(key) == 32 {
			return key, nil
		}
		if len(envKey) == 32 {
			return []byte(envKey), nil
		}
		return nil, errs.Newf(ErrEncryption, "encryption.LoadEnvKey", "invalid ORBIT_SECRET_KEY length").
			WithAdvice("ORBIT_SECRET_KEY must be a 32-byte raw string or a 64-character hex string.")
	}

	// 2. Check ~/.orbit/.master.key
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, errs.New(ErrEncryption, "encryption.UserHomeDir", err).
			WithAdvice("Unable to determine user home directory to store master key.")
	}

	orbitDir := filepath.Join(homeDir, ".orbit")
	if err := os.MkdirAll(orbitDir, 0700); err != nil {
		return nil, errs.New(ErrEncryption, "encryption.Mkdir", err)
	}

	keyPath := filepath.Join(orbitDir, KeyFilename)
	data, err := os.ReadFile(keyPath)
	if err == nil {
		if len(data) == 32 {
			return data, nil
		}
		if len(data) == 64 {
			key, err := hex.DecodeString(string(data))
			if err == nil {
				return key, nil
			}
		}
		return nil, errs.Newf(ErrEncryption, "encryption.LoadFileKey", "invalid key length in %s", keyPath).
			WithAdvice("Delete the corrupted key file to allow Orbit to generate a new one, but note this will invalidate existing encrypted state.")
	}
	if !os.IsNotExist(err) {
		return nil, errs.New(ErrEncryption, "encryption.ReadFile", err)
	}

	// 3. Generate new key securely
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, errs.New(ErrEncryption, "encryption.Generate", err).
			WithAdvice("System entropy is severely depleted.")
	}

	// Store as hex string for safe reading
	if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(key)), 0600); err != nil {
		return nil, errs.New(ErrEncryption, "encryption.WriteKey", err)
	}

	return key, nil
}

// Encrypt encrypts the given plaintext using AES-256-GCM.
func (e *Engine) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errs.New(ErrEncryption, "encryption.GenerateNonce", err)
	}
	ciphertext := e.aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts the ciphertext using AES-256-GCM.
func (e *Engine) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errs.Newf(ErrEncryption, "encryption.Decrypt", "ciphertext smaller than nonce size").
			WithAdvice("The stored data may be corrupted or not encrypted properly.")
	}
	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := e.aead.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, errs.New(ErrEncryption, "encryption.DecryptData", err).
			WithAdvice("Failed to decrypt data. The master key might have changed or data is corrupted.")
	}
	return plaintext, nil
}
