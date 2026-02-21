<div align="center">

# ◉ Orbit

**A developer-first container orchestrator for self-hosted infrastructure.**

Deploy, scale, and monitor Docker services across multiple servers — from a single beautiful terminal UI or a clean CLI.

[![CI](https://github.com/f9-o/orbit/actions/workflows/ci.yml/badge.svg)](https://github.com/f9-o/orbit/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blueviolet)](LICENSE)
[![Release](https://img.shields.io/github/v/release/f9-o/orbit?color=56E0C8)](https://github.com/f9-o/orbit/releases)

![Orbit TUI Screenshot](docs/assets/orbit-tui.png)

</div>

---

## Why Orbit?

Most container orchestrators are either too heavy (Kubernetes) or too simple (plain Docker Compose). **Orbit** sits in the sweet spot:

- **Zero infrastructure** — no control plane, no etcd cluster, no agents to install
- **Single binary** — one `orbit` binary, drop it on any server
- **Real-time TUI** — live CPU, memory, and network metrics in your terminal
- **Multi-node** — manage containers across multiple SSH servers from one config file
- **Rolling deploys** — health-check-gated rolling updates with automatic rollback

---

## Features

| Feature                                      | Status      |
| -------------------------------------------- | ----------- |
| Docker container lifecycle (up/down/restart) | ✅          |
| Rolling deploy with automatic rollback       | ✅          |
| Health checks (HTTP · TCP · shell command)   | ✅          |
| Real-time metrics (CPU · memory · network)   | ✅          |
| Multi-node SSH management                    | ✅          |
| NGINX reverse proxy auto-configuration       | ✅          |
| Interactive Bubble Tea TUI dashboard         | ✅          |
| Plugin system (Go plugin API)                | ✅          |
| GitHub Actions CI + release pipeline         | ✅          |
| SSL/TLS via ACME (Let's Encrypt)             | 🔜 **v0.2** |
| Prometheus metrics endpoint                  | 🔜 **v0.2** |
| Web UI                                       | 🔜 **v0.3** |

---

## Quick Start

### Install

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/f9-o/orbit/main/install.sh | bash

# Or download a release binary directly
# https://github.com/f9-o/orbit/releases
```

### Build from source

```bash
git clone https://github.com/f9-o/orbit.git
cd orbit
make build        # → dist/orbit
make install      # → GOPATH/bin/orbit
```

**Requirements:** Go 1.22+, Docker Engine running locally

---

## Usage

### 1. Initialize a project

```bash
orbit init
```

This scaffolds an `orbit.yaml` in the current directory.

### 2. Configure your services

```yaml
# orbit.yaml
version: "1"

project:
  name: my-app
  environment: production

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
    proxy:
      domain: myapp.example.com
      ssl: true
      port: 443
      backend: 80
```

### 3. Start everything

```bash
orbit up
```

### 4. Open the TUI dashboard

```bash
orbit ui
```

### 5. Rolling deploy

```bash
orbit deploy web --tag v1.2.0
```

---

## CLI Reference

```
orbit [command]

Commands:
  init      Scaffold a new orbit.yaml
  up        Start all services
  down      Stop and remove services
  deploy    Rolling update a service
  logs      Stream service container logs
  scale     Adjust service replica count
  monitor   Real-time metrics dashboard (text)
  ui        Launch the interactive TUI
  nodes     Manage remote SSH nodes
  ssl       Manage SSL certificates
  version   Print version information

Flags:
  -c, --config string   Path to orbit.yaml (default: auto-discover)
  -n, --node string     Target node name (default: local)
  --debug               Enable debug logging
```

---

## Remote Nodes

Orbit can manage Docker containers on remote servers over SSH:

```yaml
# orbit.yaml
nodes:
  - name: prod-01
    host: 192.168.1.10
    user: deploy
    key: ~/.ssh/orbit_ed25519
    port: 22
  - name: prod-02
    host: 192.168.1.11
    user: deploy
    key: ~/.ssh/orbit_ed25519
```

```bash
# Add a node to trusted registry
orbit nodes add prod-01 --host 192.168.1.10 --user deploy --key ~/.ssh/orbit_ed25519

# List all nodes with status
orbit nodes ls

# Test connectivity
orbit nodes test prod-01
```

---

## Architecture

```
orbit/
├── cmd/orbit/          # Binary entrypoint
├── api/v1/             # Shared types (ServiceSpec, NodeSpec, etc.)
├── internal/
│   ├── cli/            # Cobra commands
│   ├── tui/            # Bubble Tea TUI (dashboard, components)
│   ├── core/           # Config loader, logger, BoltDB state manager
│   ├── orchestrator/   # Docker API wrapper, deploy, lifecycle, scale
│   ├── health/         # Health check probes (HTTP, TCP, cmd)
│   ├── metrics/        # Container stats collector
│   ├── proxy/nginx/    # NGINX config generator
│   └── remote/         # SSH pool, node registry, heartbeat
└── pkg/
    ├── errs/           # Structured error types with codes
    ├── sshutil/        # SSH client helpers
    └── netutil/        # Network utilities
```

State is stored in `~/.orbit/state.db` (BoltDB — a single embedded file, no server).

---

## Configuration Reference

| Key                   | Type   | Default       | Description                             |
| --------------------- | ------ | ------------- | --------------------------------------- |
| `version`             | string | —             | Config schema version (currently `"1"`) |
| `project.name`        | string | —             | Project name                            |
| `project.environment` | string | `development` | Environment tag                         |
| `log.level`           | string | `info`        | `debug\|info\|warn\|error`              |
| `log.format`          | string | `text`        | `text\|json`                            |
| `metrics.enabled`     | bool   | `false`       | Enable Prometheus endpoint              |
| `metrics.port`        | int    | `9091`        | Prometheus listen port                  |
| `proxy.backend`       | string | `nginx`       | Proxy backend (`nginx\|caddy`)          |

Full reference: [docs/configuration.md](docs/configuration.md)

---

## Development

```bash
# Run all tests
make test

# Run with race detector
make test ARGS="-race"

# Build for all platforms
make release

# Lint
make lint
```

---

## Roadmap

### v0.1 — Current

- Core CLI (`init`, `up`, `down`, `deploy`, `logs`, `scale`, `nodes`, `monitor`, `ui`)
- Rolling deploy with health-check-gated rollback
- Multi-node SSH management with heartbeat engine
- Interactive Bubble Tea TUI

### v0.2 — ~6 months

- ACME/Let's Encrypt SSL automation
- Prometheus `/metrics` endpoint
- `orbit ps` (process status with nice formatting)
- Secrets management (encrypted at rest)
- Blue/green deploy strategy

### v0.3

- Web UI (React + WebSocket)
- `orbit logs --tail` with search/filter
- Alerting (webhook, Slack, PagerDuty)
- Cluster-aware scheduling

---

## Contributing

Contributions are very welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

---

## License

MIT © 2026 Orbit Contributors. See [LICENSE](LICENSE).
