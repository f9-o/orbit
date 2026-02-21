// Package remote manages SSH connections to remote nodes.
// Each node gets a persistent, multiplexed SSH connection with keepalive.
package remote

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/logger"
	"github.com/f9-o/orbit/pkg/sshutil"
)

// DefaultSSHPort is the fallback SSH port when NodeSpec.Port is 0.
const DefaultSSHPort = 22

// connection holds a live SSH connection and its metadata.
type connection struct {
	client   *ssh.Client
	node     string
	lastUsed time.Time
	cancel   context.CancelFunc
}

// Pool manages persistent SSH connections to remote nodes.
type Pool struct {
	mu    sync.Mutex
	conns map[string]*connection // node name → connection
	log   *logger.Logger
}

// NewPool creates an empty connection pool.
func NewPool(log *logger.Logger) *Pool {
	return &Pool{
		conns: make(map[string]*connection),
		log:   log,
	}
}

// Connect establishes (or returns an existing) SSH connection for a node.
func (p *Pool) Connect(ctx context.Context, node v1.NodeInfo) (*ssh.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if c, ok := p.conns[node.Spec.Name]; ok {
		// Verify connection is still alive with a lightweight keepalive
		if _, _, err := c.client.Conn.SendRequest("keepalive@orbit", true, nil); err == nil {
			c.lastUsed = time.Now()
			return c.client, nil
		}
		// Connection dead — remove it and reconnect
		c.cancel()
		delete(p.conns, node.Spec.Name)
	}

	client, err := p.dial(node)
	if err != nil {
		return nil, err
	}

	connCtx, cancel := context.WithCancel(context.Background())
	conn := &connection{
		client:   client,
		node:     node.Spec.Name,
		lastUsed: time.Now(),
		cancel:   cancel,
	}
	p.conns[node.Spec.Name] = conn

	// Background keepalive goroutine
	go p.keepalive(connCtx, node.Spec.Name, client)

	p.log.Info("ssh connected", "node", node.Spec.Name, "host", node.Spec.Host)
	return client, nil
}

// dial opens a new SSH connection to node based on its spec.
func (p *Pool) dial(node v1.NodeInfo) (*ssh.Client, error) {
	keyPath := node.Spec.Key
	if keyPath == "" {
		return nil, fmt.Errorf("no SSH key configured for node %q", node.Spec.Name)
	}

	port := node.Spec.Port
	if port == 0 {
		port = DefaultSSHPort
	}
	addr := net.JoinHostPort(node.Spec.Host, fmt.Sprintf("%d", port))

	// Use InsecureIgnoreHostKey for initial connections; proper known_hosts for subsequent.
	cfg, err := sshutil.ClientConfig(node.Spec.User, keyPath, "")
	if err != nil {
		return nil, fmt.Errorf("ssh config for node %q: %w", node.Spec.Name, err)
	}

	// Override host key callback if node has a recorded host key
	if node.HostKeyKnown && node.HostKey != "" {
		cfg.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			got := sshutil.FingerprintMD5(key)
			expect := node.KeyFingerprint
			if got != expect {
				return fmt.Errorf("host key mismatch for %s: got %s, expected %s", hostname, got, expect)
			}
			return nil
		}
	}

	return sshutil.Dial(addr, cfg)
}

// Run executes a command on the named node and returns its combined output.
func (p *Pool) Run(ctx context.Context, node v1.NodeInfo, cmd string) (string, int, error) {
	client, err := p.Connect(ctx, node)
	if err != nil {
		return "", -1, err
	}
	return sshutil.RunCommand(client, cmd)
}

// Disconnect closes the connection for a named node.
func (p *Pool) Disconnect(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.conns[name]; ok {
		c.cancel()
		c.client.Close()
		delete(p.conns, name)
		p.log.Info("ssh disconnected", "node", name)
	}
}

// Close disconnects all managed connections.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for name, c := range p.conns {
		c.cancel()
		c.client.Close()
		delete(p.conns, name)
		p.log.Info("ssh connection closed", "node", name)
	}
}

// keepalive sends periodic keepalive packets to prevent session timeout.
func (p *Pool) keepalive(ctx context.Context, node string, client *ssh.Client) {
	ticker := time.NewTicker(sshutil.KeepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, _, err := client.Conn.SendRequest("keepalive@orbit", true, nil); err != nil {
				p.log.Warn("ssh keepalive failed, connection may be dead",
					"node", node, "err", err)
				return
			}
		}
	}
}
