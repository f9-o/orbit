# Changelog

All notable changes to Orbit are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Planned for v0.2

- ACME/Let's Encrypt automated SSL certificate issuance and renewal
- Prometheus `/metrics` endpoint with per-service counters
- `orbit ps` command with rich formatted output
- Secrets management (AES-256-GCM, encrypted at rest)
- Blue/green deploy strategy

---

## [0.1.0] — 2025-02

### Added

- **Core CLI** — `init`, `up`, `down`, `deploy`, `logs`, `scale`, `monitor`, `ui`, `nodes`, `ssl`, `version`
- **Rolling deploy** — health-check-gated rolling updates with automatic rollback on failure
- **Health probes** — HTTP, TCP dial, and shell command probes with retry loop and backoff
- **Multi-node SSH** — persistent multiplexed SSH connections with keepalive and heartbeat engine
- **Node registry** — BoltDB-backed node CRUD with host key trust and online/offline tracking
- **Interactive TUI** — Bubble Tea dashboard with service table, log viewport, and real-time metrics
- **Metrics collector** — async Docker stats polling with per-service CPU/memory/network snapshots
- **NGINX config generator** — template-based server blocks with optional SSL support
- **Plugin host** — Go plugin API (`api/v1` plugin interface) for extending Orbit commands
- **Structured logger** — `log/slog`-based with JSON/text format, file output, and TUI sink
- **State persistence** — BoltDB state manager for services, nodes, and deployment audit log
- **GitHub Actions CI** — test, lint (golangci-lint), cross-platform build matrix (5 targets)
- **Release pipeline** — cosign binary signing, SHA256 checksums, automated GitHub Release

[Unreleased]: https://github.com/f9-o/orbit/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/f9-o/orbit/releases/tag/v0.1.0
