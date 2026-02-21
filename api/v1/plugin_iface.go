// Package plugin_iface defines the Orbit plugin contract (PluginV1).
// All external plugins must implement this interface and export an "OrbitPlugin" symbol.
package v1

// PluginAPIVersion is the current plugin API version.
// Checked at plugin load time to prevent incompatible plugins from loading.
const PluginAPIVersion = "v1"

// HookFunc is a function invoked at a named lifecycle point.
type HookFunc func(ctx HookContext) error

// HookContext carries contextual data passed to plugin hooks.
type HookContext struct {
	Service   *ServiceSpec
	Node      *NodeSpec
	ImageFrom string
	ImageTo   string
	DryRun    bool
	// Metadata is a free-form map for passing extension data between hooks.
	Metadata map[string]string
}

// PluginV1 is the interface every Orbit plugin must implement.
// Exported symbol name in the .so file must be "OrbitPlugin" of type PluginV1.
type PluginV1 interface {
	// Name returns the human-readable plugin identifier.
	Name() string

	// APIVersion must return exactly PluginAPIVersion.
	// A mismatch causes the plugin to be rejected at load time.
	APIVersion() string

	// Init is called once after the plugin is loaded.
	// Return an error to abort loading.
	Init(cfg map[string]string) error

	// Hooks returns the named hooks this plugin subscribes to.
	// Supported hook names:
	//   OnPreDeploy, OnPostDeploy, OnPreScale, OnPostScale,
	//   OnNodeConnect, OnNodeDisconnect, OnSSLRenew
	Hooks() map[string]HookFunc

	// Shutdown is called when Orbit exits cleanly.
	Shutdown() error
}
