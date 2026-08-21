<div align="center">

# Torus Proxy

**A Layer 7 Reverse Proxy & Edge API Gateway built in Go.**

A systems-engineering project exploring networking, concurrency, runtime lifecycle, observability, and performance through a Layer 7 reverse proxy built in Go.

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/Ashish-Barmaiya/torus-proxy)](https://github.com/Ashish-Barmaiya/torus-proxy/releases)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go CI](https://github.com/Ashish-Barmaiya/torus-proxy/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Ashish-Barmaiya/torus-proxy/actions/workflows/ci.yml)

</div>

---

## Overview

Torus is a Layer 7 reverse proxy and edge API gateway written entirely in Go.

The project began as a Node.js implementation before being rewritten in Go to explore systems programming, networking internals, and high-performance infrastructure software.

The long-term objective is not to compete directly with established production proxies, but to build a reverse proxy as a systems engineering project — implementing, benchmarking, and documenting the techniques used in modern networking infrastructure.

---

## Current Features

### Request Management

- Layer 7 reverse proxy built on Go's standard library (`net/http/httputil.ReverseProxy`)
- HTTP/1.1 reverse proxying
- Longest-prefix routing with path-segment boundary matching
- Round-robin load balancing across healthy upstreams
- Active backend health checking
- Automatic forwarding and request tracing headers

### Runtime

- Zero-downtime configuration hot reload
- Atomic runtime replacement with in-flight request and worker draining
- Graceful shutdown with request draining

### Security & Observability

- Optional TLS termination from YAML configuration
- Prometheus metrics for HTTP traffic, upstreams, and runtime state
- Structured request logging
- Readiness endpoint (`/readyz`)

### Engineering

- Comprehensive unit, component, and integration tests
- Automated benchmarking, statistical analysis, and reproducible benchmark reports

> The original Node.js/TypeScript implementation is retained in the [`node/`](node/) directory as a historical reference.

---

## Current Limitations

- HTTP/2 and HTTP/3 are not currently supported.
- Single-process architecture; no multi-process hot restart or binary hot upgrade.
- Torus is a systems-engineering and experimentation project, not a direct replacement for established production proxies.

---

## Quick Start

The Quick Start covers **native local development**. For Docker Compose or Linux/systemd deployment, see the [Deployment documentation](./docs/deployment/README.md).

### Prerequisites

- Go 1.26.1 or newer
- One or more backend HTTP services
- Optional: TLS certificate and key files for HTTPS
- Docker, if you want the local Prometheus/Grafana stack

### 1. Clone

```bash
git clone https://github.com/Ashish-Barmaiya/torus-proxy.git
cd torus-proxy
```

### 2. Build the binary

```bash
go build -o torus ./cmd/torus
```

For local testing, the repository includes a mock backend:

```bash
go run ./cmd/mock-backend 3001
```

In a second terminal:

```bash
go run ./cmd/mock-backend 3002
```

### 3. Configure

Torus is configured through a declarative YAML file.

Generic development configurations are provided under [`configs/`](configs/).

For example:

- [`configs/torus-http.yaml`](configs/torus-http.yaml) — HTTP development configuration
- [`configs/torus-https.yaml`](configs/torus-https.yaml) — HTTPS development configuration

A minimal HTTP configuration looks like:

```yaml
apiVersion: v2

server:
  addr: ":8080"

health:
  interval_ms: 5000
  timeout_ms: 2000
  path: /health

observability:
  enabled: true

services:
  - name: api
    upstreams:
      - http://localhost:3001
      - http://localhost:3002

routes:
  - path: /api
    service: api
```

See the [Configuration Guide](./docs/configuration/README.md) for the complete configuration model and field reference.

### 4. Run Torus

```bash
./torus -config configs/torus-http.yaml
```

Torus listens on the address configured by `server.addr` and exposes:

- `/readyz` — readiness endpoint
- `/metrics` — Prometheus metrics when observability is enabled

### 5. Optional local observability

Start the repository's local Prometheus and Grafana stack:

```bash
docker compose -f docker/compose.yml up -d
```

Verify metrics:

```bash
curl http://localhost:8080/metrics
```

Prometheus:

```text
http://localhost:9090
```

Grafana:

```text
http://localhost:3000
```

See the [Native Deployment](./docs/deployment/native.md) guide for the complete local verification flow.

---

## Configuration

Configuration and deployment are intentionally documented separately.

The [Configuration Guide](./docs/configuration/README.md) covers:

- API versioning
- server/listener configuration
- services and upstream backends
- routing
- health checks
- TLS termination
- Prometheus observability
- configuration reloads
- complete configuration reference

Generic Torus configuration examples are located under:

```text
configs/
```

Docker Compose uses deployment-specific configuration and supporting assets under:

```text
docker/
```

The Docker files remain with the Docker deployment because the Compose files directly mount those assets.

Repository-provided configurations and certificates are development/evaluation examples. Production deployments should use environment-specific configuration and credentials.

---

## Deployment

Torus supports three deployment models:

| Deployment | Intended use |
|---|---|
| Native | Local development, debugging, experimentation, and benchmarking |
| Docker Compose | Reproducible full-stack development and evaluation |
| Linux/systemd | Linux VM deployment and production-oriented operation |

### Native

Runs Torus directly as a host process.

[Native Deployment](./docs/deployment/native.md)

### Docker Compose

Runs Torus, mock backends, Prometheus, and Grafana together.

HTTP:

```bash
docker compose -f docker/compose.full.yml up --build -d
```

HTTPS:

```bash
docker compose -f docker/compose.full-https.yml up --build -d
```

This is a full-stack development/evaluation deployment, not a production architecture.

[Docker Compose Deployment](./docs/deployment/docker.md)

### Linux/systemd

Runs Torus as a native Linux systemd service.

```bash
sudo ./deployments/systemd/install.sh
```

The installer is configuration-driven. It validates the supplied configuration, detects the configured listener and TLS requirements, validates TLS material when needed, builds Torus, installs the deployment, starts systemd, verifies the configured listener, and rolls back the previous installation if activation fails.

[Linux/systemd Deployment](./docs/deployment/systemd.md)

### Production

For Linux VMs, systemd is the recommended production-oriented deployment model in v0.6.0.

[Production Deployment](./docs/deployment/production.md)

---

## Verification

For local HTTP:

```bash
curl http://localhost:8080/readyz
curl http://localhost:8080/api/hello
```

For local HTTPS:

```bash
curl -k https://localhost:8443/readyz
curl -k https://localhost:8443/api/hello
```

Use the actual listener configured in your Torus configuration.

For deployment-specific verification, see:

- [Native Deployment](./docs/deployment/native.md)
- [Docker Compose Deployment](./docs/deployment/docker.md)
- [Linux/systemd Deployment](./docs/deployment/systemd.md)
- [Production Deployment](./docs/deployment/production.md)

---

## Architecture

```
                                                    +--------------------------------+
                         Client                     |    Runtime Reload Pipeline     |
                            │                       |                                |
                            ▼                       |         Config Watcher         |
                     net/http Server                |              │                 |
                            │                       |              ▼                 |
                            |                       |   Build New Runtime Generation |
                            |                       |              │                 |
                            |                       |              ▼                 |
                            |                       |         Atomic Pointer         |
                            |                       +--------------------------------+
                            ▼                                       │
              +---------------------------+   Swap Runtime Pointer  │
              |   Atomic Runtime Pointer  | <───────────────────────+
              +---------------------------+
                            │
                            ▼
      +----------------------------------------------------+
      |            Immutable Runtime Generation            |
      |----------------------------------------------------+
      |        │               │                 |         |
      |        │               │                 |         |
      |        ▼               ▼                 ▼         |
      |     Router        Health State     Observability   |
      |        │               |               Config      |
      |        ▼               |                           |
      |    Services            |                           |
      |        │               |                           |
      |        ▼               ▼                           |
      | Load balancer----Health Workers                    |
      |        │               |                           |
      |        ▼               |                           |
      | Reverse Proxy          |                           |
      +----------------------------------------------------+
               |               |
               |               |
               ▼               |
         Backend Pool <--------+

  +—————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————+
  |  ** Request Path                            ** Reload Path                            ** Key Properties                     |
  |                                                                                                                             |
  |  Client —> Server —> Runtime —>             Config change —> Build new runtime —>     * Zero-downtime configuration reloads |
  |  Router —> Service —> Load Balancer —>      Atomic pointer swap —> New runtime        * Immutable runtime snapshots         |
  |  Reverse Proxy —>  Backend Pool             serves subsequent requests                * Atomic Pointer swaps                |
  |                                                                                       * Request isolation from reloads      |
  |                                                                                       * Graceful shutdown                   |
  |                                                                                                                             |
  +—————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————+
```

---

## Performance & Benchmarking

Performance engineering is a core part of Torus.

Every significant architectural change is evaluated using a standardized benchmarking framework before being documented in a published benchmark report.

The benchmarking framework includes:

- Automated benchmark execution
- Statistical analysis and summary generation
- Performance visualization
- Automated draft report generation
- Standardized benchmark methodology
- Reproducible benchmark scenarios
- Published benchmark datasets
- Historical benchmark reports

Each published benchmark includes a detailed engineering report, supporting visualizations, and a downloadable dataset distributed as a GitHub Release asset to preserve reproducibility while keeping the repository lightweight.

### Published Benchmark Reports

- [**Benchmark-001** — Node.js to Go Performance Evaluation](/docs/benchmarking/reports/Benchmark-001-nodejs-to-go-performance-evaluation.md)
- [**Benchmark-002** — HTTP vs HTTPS Performance Evaluation](./docs/benchmarking/reports/Benchmark-002-http-vs-https.md)
- [**Benchmark-003** — Observability Overhead Evaluation](./docs/benchmarking/reports/Benchmark-003-observability-overhead.md)

Additional benchmarking methodology, tooling, and reports are available in [`docs/benchmarking/`](./docs/benchmarking/).

---

## Architecture Decision Records

Torus documents major architectural decisions using **Architecture Decision Records (ADRs)**.

Each ADR captures the engineering context, alternatives considered, rationale, trade-offs, and long-term consequences behind significant design decisions. Rather than documenting only *what* was implemented, ADRs explain *why* a particular approach was selected.

Current ADRs include:

- [**ADR-001** — Rewrite Torus from Nodejs to Go](./docs/engineering/decision-records/ADR-001-rewrite-torus-from-nodejs-to-go.md)
- [**ADR-002** — Use Atomic Bool for Backend Health](./docs/engineering/decision-records/ADR-002-use-atomic-bool-for-backend-health.md)
- [**ADR-003** — Use Immutable Runtime Generations for Configuration Reload](./docs/engineering/decision-records/ADR-003-use-immutable-runtime-generations-for-configuration-reload.md)
- [**ADR-004** — Runtime-Owned Observability Configuration](./docs/engineering/decision-records/ADR-004-runtime-owned-observability-configuration.md)

Additional decision records are available in [`docs/engineering/decision-records/`](./docs/engineering/decision-records/).

---

## Documentation

Torus documentation is organized by engineering concern.

| Documentation | Description |
|---|---|
| [`docs/configuration/`](./docs/configuration/) | Configuration concepts, examples, validation, configuration reloads, and complete reference |
| [`docs/deployment/`](./docs/deployment/) | Native, Docker Compose, Linux/systemd, and production deployment guides |
| [`docs/benchmarking/`](./docs/benchmarking/) | Benchmark reports, methodology, automation framework, datasets, and statistical analysis |
| [`docs/engineering/ARCHITECTURE.md`](./docs/engineering/ARCHITECTURE.md) | System architecture overview |
| [`docs/engineering/architecture/runtime-lifecycle.md`](./docs/engineering/architecture/runtime-lifecycle.md) | Runtime lifecycle architecture |
| [`docs/engineering/refactor/`](./docs/engineering/refactor/) | Detailed engineering refactor documentation |
| [`docs/engineering/decision-records/`](./docs/engineering/decision-records/) | Architecture Decision Records documenting major engineering decisions |

Documentation is maintained alongside the source code so configuration semantics, deployment procedures, architectural decisions, refactors, and performance evaluations remain reproducible and versioned with the implementation.

---

## Project layout

```text
torus-proxy/

├── cmd/
│   ├── torus/
│   │   └── main.go
│   └── mock-backend/
│       └── main.go
│
├── configs/                     # Generic and development Torus configurations
│   ├── cert.pem
│   ├── key.pem
│   ├── torus-http*.yaml
│   ├── torus-https.yaml
│   └── torus-systemd-*.yaml
│
├── deployments/
│   └── systemd/                 # Linux/systemd deployment assets and installer
│       ├── install.sh
│       └── torus.service
│
├── docker/                      # Docker Compose deployments and observability assets
│   ├── compose.yml
│   ├── compose.full.yml
│   ├── compose.full-https.yml
│   ├── torus/
│   ├── certs/
│   ├── prometheus/
│   └── grafana/
│
├── docs/
│   ├── configuration/           # Configuration documentation
│   ├── deployment/              # Deployment documentation
│   ├── benchmarking/            # Benchmarking framework and reports
│   └── engineering/
│       ├── architecture/        # Architecture documentation
│       ├── decision-records/    # Architecture Decision Records
│       └── refactor/            # Refactor documentation
│
├── integration/                 # Runtime, reload, shutdown, and observability integration tests
├── internal/                    # Core proxy implementation
├── node/                        # Reference Node.js/TypeScript implementation
├── Dockerfile.mock
├── dockerfile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```
---

## Testing

Run the complete test suite:

```bash
go test ./...
```

Run the test suite with Go's race detector:

```bash
go test -race ./...
```
Run the race-enabled test suite repeatedly with randomized test order:
```bash
go test -race -count=50 -shuffle=on ./...
```

The repeated race-enabled run is useful for exposing concurrency defects that may not reproduce during a single test execution. Increasing the iteration count to 100 can provide additional confidence when investigating intermittent failures or race conditions:

```bash
go test -race -count=100 -shuffle=on ./...
```

The test suite includes unit, component, and integration tests covering:

- Configuration parsing and validation
- Longest-prefix route matching
- Round-robin load balancing
- Backend health management
- Reverse proxy request forwarding
- Runtime hot reload
- Graceful shutdown
- Prometheus metrics and observability

Integration tests exercise the complete production startup path, request forwarding pipeline, runtime reload mechanism, observability surface, and graceful shutdown behaviour.

---

## License

This project is licensed under the [MIT License](LICENSE).
