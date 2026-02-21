// Package sshutil provides reusable SSH client helpers for Orbit's remote layer.
package sshutil

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// DefaultPort is the standard SSH port.
const DefaultPort = 22

// ConnectTimeout is the default dial timeout for SSH connections.
const ConnectTimeout = 15 * time.Second

// KeepAliveInterval is how often a keepalive packet is sent to the server.
const KeepAliveInterval = 15 * time.Second

// ClientConfig builds an ssh.ClientConfig from a private key file.
// If knownHostsFile is non-empty, strict host key verification is enabled.
func ClientConfig(user, keyPath, knownHostsFile string) (*ssh.ClientConfig, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key %q: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	cfg := &ssh.ClientConfig{
		User:    user,
		Auth:    []ssh.AuthMethod{ssh.PublicKeys(signer)},
		Timeout: ConnectTimeout,
	}

	if knownHostsFile != "" {
		hostKeyCallback, err := knownhosts.New(knownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("load known_hosts %q: %w", knownHostsFile, err)
		}
		cfg.HostKeyCallback = hostKeyCallback
	} else {
		// Warn: insecure — only used for first-trust scenarios
		cfg.HostKeyCallback = ssh.InsecureIgnoreHostKey() //nolint:gosec
	}

	return cfg, nil
}

// Dial establishes an SSH connection to addr (host:port) using cfg.
func Dial(addr string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %q: %w", addr, err)
	}
	return client, nil
}

// RunCommand executes a shell command on the remote host and returns its combined output.
func RunCommand(client *ssh.Client, cmd string) (string, int, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", -1, fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return string(out), exitErr.ExitStatus(), err
		}
		return string(out), -1, err
	}
	return string(out), 0, nil
}

// FingerprintMD5 computes the legacy MD5 fingerprint of an SSH public key.
func FingerprintMD5(key ssh.PublicKey) string {
	sum := md5.Sum(key.Marshal()) //nolint:gosec
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(parts, ":")
}

// EncodeHostKey serialises an ssh.PublicKey to a base64 known_hosts-style line.
func EncodeHostKey(host string, key ssh.PublicKey) string {
	return fmt.Sprintf("%s %s %s",
		host,
		key.Type(),
		base64.StdEncoding.EncodeToString(key.Marshal()),
	)
}

// GatherHostKey dials addr and retrieves the server's host key without authentication.
// Used during `orbit nodes add` to record the host fingerprint before full trust.
func GatherHostKey(addr string, timeout time.Duration) (ssh.PublicKey, error) {
	var capturedKey ssh.PublicKey

	cfg := &ssh.ClientConfig{
		User: "orbit-probe",
		Auth: []ssh.AuthMethod{ssh.Password("")},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			capturedKey = key
			return nil // intentionally accept to capture key
		},
		Timeout: timeout,
	}

	// The connection will fail (auth), but we capture the key beforehand.
	conn, err := ssh.Dial("tcp", addr, cfg)
	if conn != nil {
		conn.Close()
	}
	if capturedKey == nil {
		return nil, fmt.Errorf("could not capture host key from %s: %w", addr, err)
	}
	return capturedKey, nil
}
