// Package v1 defines the public data types shared across all Orbit layers.
package v1

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// Status enumerations
// ─────────────────────────────────────────────────────────────────────────────

// ServiceStatus represents the health state of a running service.
type ServiceStatus string

const (
	StatusHealthy   ServiceStatus = "healthy"
	StatusDegraded  ServiceStatus = "degraded"
	StatusUnhealthy ServiceStatus = "unhealthy"
	StatusUnknown   ServiceStatus = "unknown"
)

// NodeStatus represents the connectivity state of a remote node.
type NodeStatus string

const (
	NodeOnline   NodeStatus = "online"
	NodeOffline  NodeStatus = "offline"
	NodeDegraded NodeStatus = "degraded"
)

// ─────────────────────────────────────────────────────────────────────────────
// Specification types (derived from orbit.yaml)
// ─────────────────────────────────────────────────────────────────────────────

// ServiceSpec is the declarative definition of a service from orbit.yaml.
type ServiceSpec struct {
	Name          string            `yaml:"name"           mapstructure:"name"`
	Image         string            `yaml:"image"          mapstructure:"image"`
	Ports         []string          `yaml:"ports"          mapstructure:"ports"`
	Environment   map[string]string `yaml:"environment"    mapstructure:"environment"`
	Labels        map[string]string `yaml:"labels"         mapstructure:"labels"`
	Volumes       []string          `yaml:"volumes"        mapstructure:"volumes"`
	Networks      []string          `yaml:"networks"       mapstructure:"networks"`
	User          string            `yaml:"user"           mapstructure:"user"`
	RestartPolicy string            `yaml:"restart"        mapstructure:"restart"`
	HealthCheck   *HealthCheckSpec  `yaml:"health_check"   mapstructure:"health_check"`
	Proxy         *ProxySpec        `yaml:"proxy"          mapstructure:"proxy"`
	Deploy        *DeploySpec       `yaml:"deploy"         mapstructure:"deploy"`
}

// HealthCheckSpec configures how Orbit probes service liveness.
type HealthCheckSpec struct {
	Type         string        `yaml:"type"          mapstructure:"type"` // tcp | http | cmd
	URL          string        `yaml:"url"           mapstructure:"url"`
	Port         int           `yaml:"port"          mapstructure:"port"`
	Command      string        `yaml:"command"       mapstructure:"command"`
	Timeout      time.Duration `yaml:"timeout"       mapstructure:"timeout"`
	Interval     time.Duration `yaml:"interval"      mapstructure:"interval"`
	Retries      int           `yaml:"retries"       mapstructure:"retries"`
	ExpectedCode int           `yaml:"expected_code" mapstructure:"expected_code"`
}

// ProxySpec controls NGINX reverse proxy generation for a service.
type ProxySpec struct {
	Domain  string `yaml:"domain"  mapstructure:"domain"`
	SSL     bool   `yaml:"ssl"     mapstructure:"ssl"`
	Port    int    `yaml:"port"    mapstructure:"port"`    // listen port on proxy
	Backend int    `yaml:"backend" mapstructure:"backend"` // container port to proxy to
}

// DeploySpec controls rolling deploy behaviour.
type DeploySpec struct {
	Replicas          int           `yaml:"replicas"           mapstructure:"replicas"`
	Strategy          string        `yaml:"strategy"           mapstructure:"strategy"` // rolling | blue-green
	MaxSurge          int           `yaml:"max_surge"          mapstructure:"max_surge"`
	RollbackOnFailure bool          `yaml:"rollback_on_failure" mapstructure:"rollback_on_failure"`
	ReadinessDelay    time.Duration `yaml:"readiness_delay"    mapstructure:"readiness_delay"`
}

// NodeSpec is the declarative definition of a remote node.
type NodeSpec struct {
	Name   string   `yaml:"name"   mapstructure:"name"`
	Host   string   `yaml:"host"   mapstructure:"host"`
	User   string   `yaml:"user"   mapstructure:"user"`
	Key    string   `yaml:"key"    mapstructure:"key"`
	Port   int      `yaml:"port"   mapstructure:"port"`
	Groups []string `yaml:"groups" mapstructure:"groups"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Runtime state types (persisted in BoltDB)
// ─────────────────────────────────────────────────────────────────────────────

// NodeInfo is the persisted runtime record for a registered node.
type NodeInfo struct {
	Spec           NodeSpec   `json:"spec"`
	Status         NodeStatus `json:"status"`
	LastSeen       time.Time  `json:"last_seen"`
	KeyFingerprint string     `json:"key_fingerprint"`
	HostKey        string     `json:"host_key"`  // base64-encoded known host line
	HostKeyKnown   bool       `json:"host_key_known"`
	FailCount      int        `json:"fail_count"`
}

// ServiceState is the runtime state of a deployed service instance.
type ServiceState struct {
	Name        string        `json:"name"`
	ContainerID string        `json:"container_id"`
	Image       string        `json:"image"`
	Status      ServiceStatus `json:"status"`
	CPU         float64       `json:"cpu"`
	MemBytes    int64         `json:"mem_bytes"`
	Replicas    int           `json:"replicas"`
	Node        string        `json:"node"`
	StartedAt   time.Time     `json:"started_at"`
	Ports       []string      `json:"ports"`
}

// DeploymentRecord is an immutable audit record of a deployment action.
type DeploymentRecord struct {
	ID          string    `json:"id"`
	Service     string    `json:"service"`
	Node        string    `json:"node"`
	FromImage   string    `json:"from_image"`
	ToImage     string    `json:"to_image"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Result      string    `json:"result"` // success | failure | rolledback
	DurationMS  int64     `json:"duration_ms"`
	Error       string    `json:"error,omitempty"`
}

// Metrics is a point-in-time snapshot of resource utilisation across services.
type Metrics struct {
	Timestamp time.Time                 `json:"timestamp"`
	Node      string                    `json:"node"`
	Services  map[string]ServiceMetrics `json:"services"`
}

// ServiceMetrics holds per-container resource stats.
type ServiceMetrics struct {
	CPUPercent float64 `json:"cpu_percent"`
	MemBytes   int64   `json:"mem_bytes"`
	MemLimit   int64   `json:"mem_limit"`
	NetRxBytes int64   `json:"net_rx_bytes"`
	NetTxBytes int64   `json:"net_tx_bytes"`
	PIDs       int     `json:"pids"`
}
