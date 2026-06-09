<div align="center">

# TORUS

**A Layer 7 Reverse Proxy & Edge API Gateway built in Go.**

_High-throughput traffic routing, active health checking, round-robin load balancing — engineered with Go's standard library and zero-copy I/O._

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

</div>

---

## The Objective

Torus is a Layer 7 Reverse Proxy and Edge API Gateway rewritten from the ground up in Go. Originally prototyped in Node.js/TypeScript, the project was migrated to Go to leverage goroutine-based concurrency, `httputil.ReverseProxy` for zero-copy stream piping, and the Go standard library's native networking stack. Torus demonstrates how production-grade traffic routing, active health probing, and load balancing operate at the systems level — without heavyweight frameworks or dependency trees.

> The original Node.js/TypeScript implementation is preserved in the [`node/`](node/) directory for reference and benchmark comparison.

---

## Features

| Feature                        | Description                                                                                                                                                                                                                                           |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Reverse Proxying**           | Forwards client requests to upstream backends using Go's `httputil.ReverseProxy`, leveraging kernel-level `sendfile`/`splice` syscalls for zero-copy data transfer.                                                                                   |
| **Longest-Prefix Routing**     | Resolves incoming URIs to upstream services using a longest-prefix-match algorithm with path segment boundary enforcement (prevents `/api-status` from matching `/api`).                                                                              |
| **Round-Robin Load Balancing** | Distributes traffic evenly across healthy backends using an atomic counter (`sync/atomic`). The `LoadBalancer` interface is pluggable for future strategies.                                                                                          |
| **Active Health Checking**     | Periodically probes every backend via HTTP `GET` requests with configurable intervals and timeouts. Unhealthy servers are automatically skipped during routing. Health probers are self-healing — they auto-restart on panic with a 2-second backoff. |
| **Header Injection**           | Injects `X-Forwarded-For`, `X-Forwarded-Proto`, `X-Forwarded-Host`, and `X-Request-ID` (auto-generated UUID if absent) into every proxied request.                                                                                                    |
| **Tuned Connection Pooling**   | Custom `http.Transport` with 10,000 max idle connections, 2,000 per host — engineered for sustained high-throughput proxying.                                                                                                                         |
| **Goroutine Concurrency**      | Single-process, goroutine-per-connection model. No cluster hacks, no IPC overhead. Thousands of concurrent connections in one process with ~2KB per goroutine.                                                                                        |

---

## Architecture

```
                         ┌──────────────────────────────────────────────┐
                         │              TORUS PROXY (Single Process)    │
                         │                                              │
   torus.yaml ─parse──▶  │  Config Parser ──▶ Router ──▶ Service Layer  │
                         │                                              │
   Client HTTP ────────▶ │  net/http Server                             │
   Requests              │    ├── Longest-Prefix Router                 │
                         │    ├── Service (Load Balancer interface)     │
                         │    └── httputil.ReverseProxy (zero-copy)     │
                         │                                              │
                         │  Health Probers (goroutines)                 │
                         │    ├── HTTP GET /health per backend          │
                         │    ├── 5s interval, 2s timeout               │
                         │    └── auto-restart on panic                 │
                         └─────────────┬────────────────────────────────┘
                                       │
                         ┌─────────────▼──────────────────────────┐
                         │           UPSTREAM BACKENDS             │
                         │  ┌───────────┐   ┌────────────┐        │
                         │  │ api_backend│  │auth_backend │  ...   │
                         │  │ :3001,:3002│  │   :3003     │        │
                         │  └───────────┘   └────────────┘        │
                         └────────────────────────────────────────┘
```

> For a deeper dive, see [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) and [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md).

---

## Quick Start

### Prerequisites

- **Go** ≥ 1.22
- One or more backend HTTP servers running (for the proxy to forward to)

### 1. Clone & Build

```bash
git clone https://github.com/Ashish-Barmaiya/torus-proxy.git
cd torus-proxy
go build -o torus-proxy ./cmd/torus/
```

### 2. Configure Routes

Edit `torus.yaml` to define your routes and upstream backends:

```yaml
server:
  port: 8080

routes:
  - path: /api
    upstream: api_backend
  - path: /auth
    upstream: auth_backend
  - path: /
    upstream: default_backend

upstreams:
  - name: api_backend
    servers:
      - host: 127.0.0.1
        port: 3001
      - host: 127.0.0.1
        port: 3002

  - name: auth_backend
    servers:
      - host: 127.0.0.1
        port: 3003

  - name: default_backend
    servers:
      - host: 127.0.0.1
        port: 3001
```

### 3. Run

```bash
./torus-proxy
```

Torus will start accepting HTTP traffic on `:8080` and begin health-checking all configured backends.

---

## Benchmarks

Tested on a constrained **Intel i3-1115G4 (2C/4T), 8 GB RAM** with `wrk` and native Go mock backends — full results in [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md).

| Scenario                              | Throughput         | Avg Latency | Max Tail Latency |
| ------------------------------------- | ------------------ | ----------- | ---------------- |
| **Full Production Proxying** (`/api`)  | **17,865 req/sec** | 6.12 ms     | 45.32 ms         |
| **Short-Circuit Memory Path** (`/api2`)| **98,565 req/sec** | 10.21 ms    | 86.63 ms         |

> **~10.8× throughput improvement** over the original Node.js implementation on identical hardware. See [BENCHMARKS.md](docs/BENCHMARKS.md) for the full methodology and historical comparison.

---

## Project Structure

```
torus-proxy/
├── cmd/
│   └── torus/
│       └── main.go                # Entry point — config, health probes, server boot
├── internal/
│   ├── config/                    # (planned) YAML config parser
│   ├── health/
│   │   ├── checker.go             # Health prober goroutine (periodic, self-healing)
│   │   ├── http.go                # HTTP GET health check implementation
│   │   └── health_test.go
│   ├── loadbalancer/
│   │   ├── balancer.go            # LoadBalancer interface
│   │   ├── round_robin.go         # Round-robin with atomic index, skips unhealthy
│   │   └── round_robin_test.go
│   ├── middleware/                # (planned) Middleware chain
│   ├── observability/             # (planned) Prometheus metrics, structured logging
│   ├── proxy/
│   │   ├── server.go              # HTTP server, request handler, timeout config
│   │   └── server_test.go
│   ├── routing/
│   │   ├── router.go              # Longest-prefix-match route resolver
│   │   └── router_test.go
│   ├── service/
│   │   └── service.go             # Service layer — binds routes to load balancers
│   ├── transport/
│   │   └── http.go                # Transport layer — delegates to ReverseProxy
│   └── upstream/
│       └── backend.go             # Backend model (health state, reverse proxy, headers)
├── node/                          # Archived Node.js/TypeScript implementation
├── docs/
│   ├── ARCHITECTURE.md            # Detailed architecture specification
│   └── BENCHMARKS.md              # Performance benchmarks (Go vs Node.js)
├── torus.yaml                     # Proxy configuration
├── mock_backend.go                # Minimal Go mock backend for benchmarking
├── go.mod
├── go.sum
└── .github/
    └── workflows/
        └── ci.yml                 # Go CI — tests with race detector
```

---

## Running Tests

```bash
go test -v -race ./...
```

Tests cover the routing engine, round-robin load balancer (distribution, ordering, unhealthy-skip), health checker, and proxy server.

---

## Tech Stack

| Layer           | Technology                                         |
| --------------- | -------------------------------------------------- |
| Language        | Go 1.22+                                           |
| Reverse Proxy   | `net/http/httputil.ReverseProxy` (stdlib)          |
| HTTP Server     | `net/http` (stdlib)                                |
| Health Probes   | HTTP `GET` with `net/http.Client` (stdlib)         |
| Load Balancing  | Custom round-robin with `sync/atomic` (stdlib)     |
| Request Tracing | `github.com/google/uuid` (X-Request-ID generation) |
| Configuration   | YAML (`torus.yaml`)                                |
| Testing         | `testing` package (stdlib) + `-race` detector      |
| CI              | GitHub Actions (Go, race-enabled tests)            |

---

## Roadmap

The following features are planned or in progress:

- [ ] **YAML Config Parser** — Dynamic config loading from `torus.yaml` (currently hardcoded in `main.go`)
- [ ] **TLS Termination** — HTTPS decryption at the proxy edge via `crypto/tls`
- [ ] **Distributed Rate Limiting** — Redis-backed atomic Token Bucket (Lua script)
- [ ] **Hot Reload** — `fsnotify` file watcher with atomic router swap (`atomic.Pointer[Router]`)
- [ ] **Prometheus Metrics** — `/metrics` endpoint with request counters, latency histograms
- [ ] **Structured Logging** — `log/slog` with JSON/text handler toggle
- [ ] **WebSocket Proxying** — HTTP Upgrade → raw TCP bidirectional pipe via `io.Copy`
- [ ] **Graceful Shutdown** — `signal.NotifyContext` + `http.Server.Shutdown` with connection draining
- [ ] **Middleware Chain** — Composable middleware pipeline (rate limit → auth → route)

---

## License

This project is licensed under the [MIT License](LICENSE).
