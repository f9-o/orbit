// Package plugin implements the Orbit plugin host.
// Plugins are loaded from ~/.orbit/plugins/ as Go shared objects (.so files).
// Each .so must export an "OrbitPlugin" symbol implementing api/v1.PluginV1.
package plugin

import (
	"context"
	"fmt"
	"path/filepath"
	"plugin"
	"sync"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/logger"
)

// Host manages plugin lifecycle and hook dispatch.
type Host struct {
	mu      sync.RWMutex
	plugins map[string]v1.PluginV1   // name → plugin
	hooks   map[string][]v1.HookFunc // hookName → ordered list
	log     *logger.Logger
}

// NewHost creates and returns an empty plugin host.
func NewHost(log *logger.Logger) *Host {
	return &Host{
		plugins: make(map[string]v1.PluginV1),
		hooks:   make(map[string][]v1.HookFunc),
		log:     log,
	}
}

// LoadDir scans dir for *.so files and attempts to load each as an Orbit plugin.
// Load failures are logged and skipped — they never abort the host startup.
func (h *Host) LoadDir(dir string) error {
	matches, err := filepath.Glob(filepath.Join(dir, "*.so"))
	if err != nil {
		return fmt.Errorf("glob plugins: %w", err)
	}

	for _, path := range matches {
		if err := h.loadPlugin(path); err != nil {
			h.log.Warn("plugin load failed, skipping",
				"path", path,
				"err", err,
			)
		}
	}
	return nil
}

// loadPlugin opens a single .so file and registers its hooks.
func (h *Host) loadPlugin(path string) (retErr error) {
	// Recover from plugin panics so a bad .so never crashes Orbit
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("plugin panicked during load: %v", r)
		}
	}()

	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("open shared object: %w", err)
	}

	sym, err := p.Lookup("OrbitPlugin")
	if err != nil {
		return fmt.Errorf("symbol OrbitPlugin not found: %w", err)
	}

	impl, ok := sym.(v1.PluginV1)
	if !ok {
		return fmt.Errorf("OrbitPlugin does not implement PluginV1")
	}

	if impl.APIVersion() != v1.PluginAPIVersion {
		return fmt.Errorf("API version mismatch: plugin=%q, host=%q",
			impl.APIVersion(), v1.PluginAPIVersion)
	}

	if err := impl.Init(nil); err != nil {
		return fmt.Errorf("plugin Init() failed: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	name := impl.Name()
	h.plugins[name] = impl

	for hookName, fn := range impl.Hooks() {
		h.hooks[hookName] = append(h.hooks[hookName], fn)
	}

	h.log.Info("plugin loaded", "name", name, "api_version", impl.APIVersion())
	return nil
}

// Fire dispatches a named hook to all registered plugins.
// Plugin errors are logged but do not prevent subsequent plugins from running.
// The context may be used to cancel long-running hook implementations.
func (h *Host) Fire(ctx context.Context, hookName string, hctx v1.HookContext) {
	h.mu.RLock()
	fns := h.hooks[hookName]
	h.mu.RUnlock()

	for _, fn := range fns {
		select {
		case <-ctx.Done():
			return
		default:
		}

		func(f v1.HookFunc) {
			defer func() {
				if r := recover(); r != nil {
					h.log.Error("plugin hook panicked",
						"hook", hookName,
						"panic", fmt.Sprintf("%v", r),
					)
				}
			}()
			if err := f(hctx); err != nil {
				h.log.Warn("plugin hook returned error",
					"hook", hookName,
					"err", err,
				)
			}
		}(fn)
	}
}

// Shutdown calls Shutdown() on every loaded plugin.
func (h *Host) Shutdown() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for name, p := range h.plugins {
		if err := p.Shutdown(); err != nil {
			h.log.Warn("plugin shutdown error", "name", name, "err", err)
		}
	}
}

// List returns the names of all loaded plugins.
func (h *Host) List() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	names := make([]string, 0, len(h.plugins))
	for name := range h.plugins {
		names = append(names, name)
	}
	return names
}
