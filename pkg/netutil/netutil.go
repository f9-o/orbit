// Package netutil provides network utility helpers used across Orbit.
package netutil

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"time"
)

var (
	// serviceNameRegex enforces DNS-label-safe service names.
	serviceNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{0,62}$`)

	// domainRegex provides a basic domain name sanity check.
	domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
)

// IsValidServiceName returns true if name is a DNS-label-safe service name.
func IsValidServiceName(name string) bool {
	return serviceNameRegex.MatchString(name)
}

// IsValidDomain returns true if domain passes basic format validation.
func IsValidDomain(domain string) bool {
	return domainRegex.MatchString(domain)
}

// IsValidPort returns true if port is in the user-space range (1024–65535).
func IsValidPort(port int) bool {
	return port >= 1024 && port <= 65535
}

// ProbeTCP dials host:port and returns nil if successful within the timeout.
func ProbeTCP(ctx context.Context, host string, port int, timeout time.Duration) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp probe to %s failed: %w", addr, err)
	}
	conn.Close()
	return nil
}

// FreePort finds an available TCP port on localhost.
func FreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// ResolveHost resolves a hostname and returns its first IP address string.
func ResolveHost(host string) (string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("no addresses resolved for %q", host)
	}
	return addrs[0], nil
}

// SplitHostPort wraps net.SplitHostPort with a default port fallback.
func SplitHostPort(addr string, defaultPort int) (host string, port string, err error) {
	host, port, err = net.SplitHostPort(addr)
	if err != nil {
		// No port in addr — treat entire string as host
		return addr, fmt.Sprintf("%d", defaultPort), nil
	}
	return host, port, nil
}
