// Package commands provides the shared context type and all CLI subcommands.
package commands

import (
	"context"

	"github.com/f9-o/orbit/internal/core/config"
	"github.com/f9-o/orbit/internal/core/logger"
	"github.com/f9-o/orbit/internal/core/state"
)

// contextKey is the key type for values stored in a command context.
type contextKey string

const runtimeContextKey contextKey = "orbit.runtime"

// GlobalFlags holds the parsed global flags for use by subcommands.
type GlobalFlags struct {
	Node       string
	Debug      bool
	JSONOutput bool
	DryRun     bool
}

// Runtime is the shared dependency bundle injected into each subcommand via context.
type Runtime struct {
	Config *config.Config
	Log    *logger.Logger
	State  *state.DB
	Flags  GlobalFlags
}

// NewContext returns a new context carrying the Runtime.
func NewContext(parent context.Context, rt *Runtime) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, runtimeContextKey, rt)
}

// FromContext extracts the Runtime from ctx. Panics if not present (programming error).
func FromContext(ctx context.Context) *Runtime {
	rt, ok := ctx.Value(runtimeContextKey).(*Runtime)
	if !ok || rt == nil {
		panic("orbit: Runtime not found in context — missing PersistentPreRunE?")
	}
	return rt
}
