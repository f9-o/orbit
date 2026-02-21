// Package health: TCP and command probe implementations.
package health

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"time"
)

// CheckTCP dials host:port and returns nil if the connection succeeds.
func CheckTCP(ctx context.Context, host string, port int, timeout time.Duration) error {
	if port == 0 {
		return fmt.Errorf("tcp health check: port is required")
	}
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	dialer := &net.Dialer{Timeout: timeout}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp dial %q: %w", addr, err)
	}
	conn.Close()
	return nil
}

// CheckCmd runs a shell command locally and returns nil if it exits 0.
func CheckCmd(ctx context.Context, command string, timeout time.Duration) error {
	if command == "" {
		return fmt.Errorf("cmd health check: command is required")
	}
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute via shell to support pipes and compound commands
	cmd := exec.CommandContext(ctx, "sh", "-c", command) //nolint:gosec
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cmd probe %q exited non-zero: %w (output: %s)", command, err, string(out))
	}
	return nil
}
