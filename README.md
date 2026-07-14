<div align="center">

# Torus Proxy

**A Layer 7 Reverse Proxy & Edge API Gateway built in Go.**

High-performance traffic routing, health-aware load balancing, and production-oriented infrastructure engineering built with Go's standard library.

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

</div>

---

## Overview

Torus is a Layer 7 reverse proxy and edge API gateway written entirely in Go.

The project began as a Node.js implementation before being rewritten in Go to explore systems programming, networking internals, and high-performance infrastructure software.

The long-term objective is not to compete directly with established production proxies, but to build a reverse proxy as a systems engineering project—implementing, benchmarking, and documenting the techniques used in modern networking infrastructure.

---

## Current Features

- Reverse proxying with Go's standard-library `net/http/httputil.ReverseProxy`
- Longest-prefix route matching with path-segment boundaries
- Round-robin load balancing across configured upstreams
- Active health probing for each backend with configurable interval and timeout
- Automatic header injection for forwarded client information and request tracing
- Optional TLS termination from YAML config
- Structured request logging
- Readiness endpoint (`/readyz`)
- Graceful shutdown
- Comprehensive unit tests
- Automated benchmarking and statistical analysis framework

The original Node.js/TypeScript prototype remains in the [node/](node/) directory as a historical reference.

---

## Engineering Principles

Torus is developed with a strong emphasis on engineering discipline.

Major architectural decisions are documented through Architecture Decision Records (ADRs), while performance-sensitive changes are validated through reproducible benchmark reports.

This repository intentionally treats documentation, benchmarking, and implementation as equally important parts of the engineering process.

---

## Architecture

```
                Client

                   │

                   ▼

             net/http Server

                   │

                   ▼

           Longest Prefix Router

                   │

                   ▼

              Service Layer

                   │

                   ▼

         Round-Robin Load Balancer

                   │

                   ▼

          httputil.ReverseProxy

                   │

                   ▼

            Upstream Backend
```

---

## Performance & Benchmarking

Performance engineering is a core part of Torus.

Every significant architectural change is evaluated using the project's automated benchmarking framework before being documented in a benchmark report.

The benchmarking framework includes:

- Automated benchmark execution
- Statistical analysis
- Automated report generation
- Standardized benchmark methodology
- Hardware profiles
- Environment profiles
- Software baselines
- Historical benchmark reports

Published benchmark datasets are distributed separately as GitHub Release assets to keep the repository lightweight while preserving reproducibility.

Current benchmarks reports:

- [**Benchmark-001** — Node.js to Go Performance Evaluation](/docs/benchmarking/reports/Benchmark-001-nodejs-to-go-performance-evaluation.md)

See:

```
docs/
└── benchmarking/
```

---

## [Engineering Documentation](/docs)

Torus maintains engineering documentation beyond source code.

```
docs/

├── benchmarking/
│   ├── benchmarking standard
│   ├── methodology
│   ├── statistics
│   ├── benchmark automation
│   ├── benchmark reports
│   └── benchmark datasets
│
└── engineering/
    ├── architecture
    ├── design notes
    └── architecture decision records
```

Architecture Decision Records (ADRs) document major architectural decisions together with their rationale and consequences.

---

## Quick start

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

### 3. Configure

The repository includes a sample configuration in [torus.yaml](torus.yaml). A minimal example looks like this:

```yaml
server:
  addr: ":8080"

health:
  interval_ms: 5000
  timeout_ms: 2000
  path: /health

routes:
  - path: /api
    upstream:
      - "http://localhost:3001"
      - "http://localhost:3002"
```

TLS can be enabled by adding a `tls` section with certificate paths and a minimum version:

```yaml
tls:
  cert_file: "cert.pem"
  key_file: "key.pem"
  min_version: "1.2"
```

### 4. Run the proxy

```bash
./torus
```

The proxy listens on the configured address and exposes:

```bash
/readyz
```

for readiness checks

---

## Example behavior

A request such as:

```bash
curl http://localhost:8080/api/hello
```

The request is:

- matched using longest-prefix routing
- load-balanced using round robin
- enriched with forwarding headers
- forwarded to a healthy backend
- proxied through Go's standard library reverse proxy

---

## Testing

Run the test suite with:

```bash
go test ./...
```

Race detector:

```bash
go test -race ./...
```

Current test coverage includes:

- routing
- load balancing
- backend health
- proxy behaviour
- graceful error handling

---

## Roadmap

Planned work includes:

- Hot configuration reload
- Prometheus metrics
- Comparative benchmarking
- Allocation reduction
- Allocation-free HTTP parsing
- Advanced networking optimizations
- Kernel-assisted networking research

---

## Project layout

```text
torus-proxy/
├── cmd/
│   └── torus/
│       └── main.go
├── internal/
│   ├── config/
│   ├── health/
│   ├── loadbalancer/
│   ├── middleware/
│   ├── proxy/
│   ├── routing/
│   ├── service/
│   ├── transport/
│   └── upstream/
|
├── node/                 # Reference Node.js/TypeScript implementation
├── docs/
|   ├── benchmarking/
│   └── engineering/
|
├── torus.yaml
├── mock_backend.go
└── go.mod
```

---

## [Documentation](/docs/)

| Document | Description |
|----------|-------------|
| [`docs/engineering/ARCHITECTURE.md`](./docs/engineering/ARCHITECTURE.md) | System architecture |
| [`docs/benchmarking/`](./docs/benchmarking/) | Benchmark reports, methodology, tooling and statistical framework |
| [`docs/engineering/decision-records/`](./docs/engineering/decision-records/) | Architecture Decision Records (ADRs) |

---

## License

This project is licensed under the [MIT License](LICENSE).
