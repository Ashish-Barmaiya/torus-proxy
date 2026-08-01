<div align="center">

# Torus Proxy

**A Layer 7 Reverse Proxy & Edge API Gateway built in Go.**

High-performance traffic routing, health-aware load balancing, zero-downtime runtime reloading, and built-in observability implemented using Go's standard library.

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

### Traffic Management

- Layer 7 reverse proxy built on Go's standard library (`net/http/httputil.ReverseProxy`)
- Longest-prefix routing with path-segment boundary matching
- Round-robin load balancing across healthy upstreams
- Active backend health checking
- Automatic forwarding and request tracing headers

### Runtime

- Zero-downtime configuration hot reload
- Atomic runtime replacement with reference-counted retirement
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

## Quick Start

### Prerequisites

- Go 1.26.1 or newer
- One or more backend HTTP services to proxy to
- Optional: TLS certificate and key files if you want to serve HTTPS

### 1. Clone

```bash
git clone https://github.com/Ashish-Barmaiya/torus-proxy.git

cd torus-proxy
```

### 2. Build the binary

```bash
go build -o torus ./cmd/torus
```

For local testing, a simple mock backend is included:

```bash
go run ./cmd/mock-backend 3001
```

In a second terminal:

```bash
go run ./cmd/mock-backend 3002
```

### 3. Configure

The repository includes sample configuration files in [configs/](configs/):

- [configs/torus-http.yaml](configs/torus-http.yaml) for an HTTP-only proxy
- [configs/torus-https.yaml](configs/torus-https.yaml) for TLS termination

A minimal HTTP example looks like this:

```yaml
apiVersion: v1

server:
  addr: ":8080"

health:
  interval_ms: 5000
  timeout_ms: 2000
  path: /health

observability:
  enabled: true # When enabled, Torus exposes Prometheus metrics at /metrics.

routes:
  - path: /api
    upstream:
      - "http://localhost:3001"
      - "http://localhost:3002"
```

The HTTPS sample adds a `tls` section with certificate paths and a minimum version:

```yaml
server:
  addr: ":8443"

tls:
  cert_file: "cert.pem"
  key_file: "key.pem"
  min_version: "1.2"
```

When starting the proxy, pass the config file with `-config` if you are not using the default file name:

```bash
./torus -config configs/torus-http.yaml
```

### 4. Run the proxy

```bash
./torus -config configs/torus-http.yaml
```

The proxy listens on the configured address and exposes:

- `/readyz` — readiness endpoint
- `/metrics` — Prometheus metrics (when observability is enabled)

To launch the local observability stack with Prometheus and Grafana:

```bash
docker compose -f docker/compose.yml up -d
```

---

## Verify

A request such as:

```bash
curl http://localhost:8080/readyz
```

```bash
curl http://localhost:8080/api/hello
```

---

## Architecture

```
                                                    +--------------------------------+
                         Client                     |    Runtime Reload Pipeline     |
                            │                       |                                |
                            ▼                       |         Config Watcher         |
                     net/http Server                |              │                 |
                            │                       |              ▼                 |
                            |                       |    Build New Runtime Snapshot  |
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
      +---------------------------------------+
      |       Immutable Runtime Snapshot      |
      |---------------------------------------+
      |        │                   │          |
      |        │                   │          |
      |        ▼                   ▼          |
      |     Router            Health State    |
      |        │                   |          |
      |        ▼                   ▼          |
      |    Services          Health Workers   |
      |        │                   |          |
      |        ▼                   |          |
      | Reverse Proxy              |          |
      +---------------------------------------+
               |                   |
               |                   |
               ▼                   |
         Backend Pool <------------+

  +—————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————+
  |  ** Request Path                            ** Reload Path                            ** Key Properties                     |
  |                                                                                                                             |
  |  Client —> Server —> Runtime —>             Config change —> Build new rutime —>      * Zero-downtime configuration reloads |
  |  Router —> Service —> Reverse Proxy —>      Atomic pointer swap —> New runtime        * Immutable runtime snapshots         |
  |  Backend Pool                               serves subsequent requests                * Atomic Pointer swaps                |
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

## Documentation

Torus is accompanied by extensive engineering documentation covering architecture, benchmarking, and design decisions.

| Documentation | Description |
|--------------|-------------|
| [`docs/benchmarking/`](./docs/benchmarking/) | Benchmark reports, methodology, automation framework, datasets, and statistical analysis |
| [`docs/engineering/ARCHITECTURE.md`](./docs/engineering/ARCHITECTURE.md) | System architecture and runtime design |
| [`docs/engineering/decision-records/`](./docs/engineering/decision-records/) | Architecture Decision Records (ADRs) documenting major engineering decisions |

The documentation is maintained alongside the source code to ensure that architectural decisions, performance evaluations, and implementation details remain reproducible and easy to understand.s.

---

The request is:

- matched using longest-prefix routing
- load-balanced using round robin
- enriched with forwarding headers
- forwarded to a healthy backend
- proxied through Go's standard library reverse proxy

---

## Project layout

```text
torus-proxy/
├── cmd/
│   ├── torus/
│   │   └── main.go
│   └── mock-backend/
|
├── configs/             # sample YAML configs for HTTP and HTTPS
├── docker/              # Prometheus and Grafana compose setup
├── docs/                # architecture, ADRs, and benchmarking docs
│   ├── benchmarking/          # Benchmarking framework and reports
│   └── engineering/           # Architecture and ADRs
|
├── integration/         # integration tests for runtime, reload, shutdown, and observability
├── internal/            # core proxy implementation
├── node/                # reference Node.js/TypeScript implementation
├── Dockerfile.mock
├── dockerfile
├── go.mod
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
