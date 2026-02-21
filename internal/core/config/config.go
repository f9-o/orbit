// Package config provides the Orbit configuration loader.
// Config is loaded by merging orbit.yaml → ~/.orbit/config.yaml → ORBIT_* env vars.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"

	v1 "github.com/f9-o/orbit/api/v1"
)

// sensitiveKeyRegex matches config keys that should be redacted in log output.
var sensitiveKeyRegex = regexp.MustCompile(`(?i)(password|token|secret|key|passphrase)`)

// Defaults contains factory-default values applied before any config file is loaded.
var Defaults = map[string]any{
	"project.environment": "development",
	"log.level":           "info",
	"log.format":          "text",
	"metrics.enabled":     false,
	"metrics.port":        9091,
	"proxy.backend":       "nginx",
	"ssl.acme_url":        "https://acme-v02.api.letsencrypt.org/directory",
}

// ─────────────────────────────────────────────────────────────────────────────
// Config types
// ─────────────────────────────────────────────────────────────────────────────

// Config is the fully-decoded project configuration.
type Config struct {
	Version  string           `mapstructure:"version"`
	Project  ProjectConfig    `mapstructure:"project"`
	Nodes    []v1.NodeSpec    `mapstructure:"nodes"`
	Services []v1.ServiceSpec `mapstructure:"services"`
	Metrics  MetricsConfig    `mapstructure:"metrics"`
	Proxy    ProxyConfig      `mapstructure:"proxy"`
	SSL      SSLConfig        `mapstructure:"ssl"`
	Log      LogConfig        `mapstructure:"log"`
}

// ProjectConfig holds project-level metadata.
type ProjectConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
}

// MetricsConfig controls the optional Prometheus /metrics endpoint.
type MetricsConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

// ProxyConfig holds reverse proxy settings.
type ProxyConfig struct {
	Backend    string `mapstructure:"backend"`     // nginx | caddy
	ConfigPath string `mapstructure:"config_path"` // output config file path
}

// SSLConfig holds ACME configuration.
type SSLConfig struct {
	AcmeURL   string        `mapstructure:"acme_url"`
	Email     string        `mapstructure:"email"`
	CertDir   string        `mapstructure:"cert_dir"`
	RenewDays int           `mapstructure:"renew_days"` // renew if expiry < N days
	Timeout   time.Duration `mapstructure:"timeout"`
}

// LogConfig controls logging behaviour.
type LogConfig struct {
	Level  string `mapstructure:"level"` // debug | info | warn | error
	File   string `mapstructure:"file"`
	Format string `mapstructure:"format"` // json | text
}

// ─────────────────────────────────────────────────────────────────────────────
// Loader
// ─────────────────────────────────────────────────────────────────────────────

// Load discovers and loads the configuration, walking up directories to find
// orbit.yaml, then merging it with the global config and environment variables.
func Load(explicitPath string) (*Config, error) {
	v := viper.New()

	// Apply defaults
	for k, val := range Defaults {
		v.SetDefault(k, val)
	}

	// Environment variable binding: ORBIT_LOG_LEVEL → log.level
	v.SetEnvPrefix("ORBIT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Load global config (~/.orbit/config.yaml) if it exists
	globalCfg := filepath.Join(orbitHome(), "config.yaml")
	if _, err := os.Stat(globalCfg); err == nil {
		v.SetConfigFile(globalCfg)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read global config: %w", err)
		}
	}

	// Load project config
	if explicitPath != "" {
		v.SetConfigFile(explicitPath)
	} else {
		path, err := discoverProjectConfig()
		if err == nil {
			v.SetConfigFile(path)
		}
	}

	if v.ConfigFileUsed() != "" || explicitPath != "" {
		if err := v.MergeInConfig(); err != nil && explicitPath != "" {
			return nil, fmt.Errorf("read project config %q: %w", explicitPath, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Resolve env variable placeholders in string values
	expandEnvInConfig(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return &cfg, nil
}

// ServiceByName returns the ServiceSpec with the given name, or nil.
func (c *Config) ServiceByName(name string) *v1.ServiceSpec {
	for i := range c.Services {
		if c.Services[i].Name == name {
			return &c.Services[i]
		}
	}
	return nil
}

// NodeByName returns the NodeSpec with the given name, or nil.
func (c *Config) NodeByName(name string) *v1.NodeSpec {
	for i := range c.Nodes {
		if c.Nodes[i].Name == name {
			return &c.Nodes[i]
		}
	}
	return nil
}

// IsSensitiveKey returns true if key matches a known sensitive pattern.
func IsSensitiveKey(key string) bool {
	return sensitiveKeyRegex.MatchString(key)
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

// discoverProjectConfig walks up from the CWD looking for orbit.yaml.
func discoverProjectConfig() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, "orbit.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("orbit.yaml not found (searched up from %s)", func() string { d, _ := os.Getwd(); return d }())
}

// expandEnvInConfig resolves ${VAR} placeholders in sensitive string fields.
func expandEnvInConfig(cfg *Config) {
	for i := range cfg.Services {
		for k, v := range cfg.Services[i].Environment {
			cfg.Services[i].Environment[k] = os.ExpandEnv(v)
		}
	}
	cfg.SSL.Email = os.ExpandEnv(cfg.SSL.Email)
}

// validate performs semantic validation on the loaded config.
func validate(cfg *Config) error {
	seen := map[string]bool{}
	for _, svc := range cfg.Services {
		if svc.Name == "" {
			return fmt.Errorf("service with empty name is not allowed")
		}
		if seen[svc.Name] {
			return fmt.Errorf("duplicate service name: %q", svc.Name)
		}
		seen[svc.Name] = true
		if svc.Image == "" {
			return fmt.Errorf("service %q: image is required", svc.Name)
		}
	}
	return nil
}

// OrbitHome returns the Orbit home directory (~/.orbit).
func orbitHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orbit"
	}
	return filepath.Join(home, ".orbit")
}

// OrbitHome is the exported variant for use by other packages.
func OrbitHome() string {
	return orbitHome()
}

// DefaultConfigTemplate is the content written by `orbit init`.
const DefaultConfigTemplate = `# orbit.yaml — Project manifest
# See: https://github.com/f9-o/orbit/docs/cli-reference.md
version: "1"

project:
  name: my-app
  environment: production

# nodes:
#   - name: prod-01
#     host: 192.168.1.10
#     user: deploy
#     key: ~/.ssh/orbit_ed25519
#     port: 22

services:
  - name: web
    image: nginx:alpine
    ports:
      - "80:80"
    restart: unless-stopped
    health_check:
      type: http
      url: http://localhost:80/
      timeout: 5s
      interval: 10s
      retries: 3
    deploy:
      replicas: 1
      strategy: rolling
      rollback_on_failure: true
`
