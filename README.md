<div align="center">

# <h1 style="font-size: 48px;">TORUS</h1>

**A Layer 7 Reverse Proxy & Edge API Gateway built natively on Node.js.**

*High-throughput traffic routing, TLS termination, redis-backed distributed rate limiting - engineered directly on Node.js core modules.*

[![Node.js](https://img.shields.io/badge/Node.js-22+-339933?logo=node.js&logoColor=white)](https://nodejs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-Strict-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D?logo=redis&logoColor=white)](https://redis.io/)
[![License](https://img.shields.io/badge/License-ISC-blue.svg)](LICENSE)

</div>

---

## The Objective

Torus is a Layer 7 Reverse Proxy and Edge API Gateway built entirely from scratch using Node.js core modules (`node:http`, `node:https`, `node:cluster`, `node:net`). It was engineered to explore the low-level mechanics of distributed systems, stream processing, and network security without relying on abstracted web frameworks like Express or Fastify. Torus demonstrates how production-grade routing and high-availability infrastructure actually operate at the OS and TCP level.

---

## Features

| Feature | Description |
|---|---|
|**Reverse Proxying** | Streams client requests to upstream backends using native `stream.pipe()`, keeping the V8 heap uninvolved in payload transfer. |
|**Longest-Prefix Routing** | Resolves incoming URIs to upstream clusters using a longest-prefix-match algorithm, parsed from `torus.yaml`. |
|**Round-Robin Load Balancing** | Distributes traffic evenly across healthy servers. The strategy pattern (`ILoadBalancingStrategy`) is pluggable for future algorithms. |
|**Active Health Checking** | Probes every backend with raw TCP socket connections (`node:net`) every 10 seconds. Unhealthy servers are pruned from rotation instantly. |
|**TLS Termination** | Decrypts HTTPS at the proxy edge, forwarding plain HTTP internally — saving backend CPU cycles. Injects `X-Forwarded-For` and `X-Forwarded-Proto` headers. |
|**Distributed Rate Limiting** | Atomic Token Bucket algorithm executed as a Lua script inside Redis, guaranteeing race-condition-free global rate limiting across all worker processes. |
|**Zero-Downtime Hot Reload** | The master process watches `torus.yaml` for changes. On a validated update, it broadcasts an IPC signal to all workers, which hot-swap their routing tables without dropping a single TCP connection. |
|**Multi-Core Clustering** | Forks one worker per CPU core. The OS kernel distributes incoming connections across workers. If a worker dies, the master automatically replaces it. |
|**Prometheus Metrics** | Exposes a `/metrics` endpoint with request counters, latency histograms, event loop lag, and V8 heap statistics via `prom-client`. |
|**Structured Logging** | Emits NDJSON logs via Pino (production) or pretty-printed colorized output (development). Log level is configurable via `LOG_LEVEL` env var. |

---

## Architecture

```
                          ┌─────────────────────────────────────────────┐
                          │              MASTER PROCESS                 │
                          │                                             │
    torus.yaml ──watch──▶ │  fs.watch() ──▶ validate ──▶ IPC broadcast │
                          └────────┬────────────┬───────────┬───────────┘
                                   │            │           │
                          ┌────────▼──┐  ┌──────▼───┐  ┌───▼────────┐
                          │ Worker 1  │  │ Worker 2 │  │ Worker N   │
                          │           │  │          │  │            │
    HTTPS ──────────────▶ │ TLS Term. │  │ TLS Term.│  │ TLS Term.  │
    Client                │ Rate Limit│  │ Rate Lim.│  │ Rate Limit │
    Requests              │ Router    │  │ Router   │  │ Router     │
                          │ Health ✓  │  │ Health ✓ │  │ Health ✓  │
                          └─────┬─────┘  └────┬─────┘  └─────┬──────┘
                                │             │              │
                          ┌─────▼─────────────▼──────────────▼──────┐
                          │           UPSTREAM BACKENDS             │
                          │  ┌───────────┐   ┌────────────┐         │
                          │  │ api_backend│  │auth_backend│  ...    │
                          │  │ :3001,:3002│  │   :3003    │         |
                          │  └───────────┘   └────────────┘         │
                          └─────────────────────────────────────────┘
                                           │
                          ┌────────────────▼────────────────┐
                          │        REDIS (Rate Limit)       │
                          │   Atomic Lua Token Bucket State │
                          └─────────────────────────────────┘
```

> For a deeper dive, see [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) and [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md).

---

## Quick Start

### Prerequisites

- **Node.js** ≥ 22
- **Redis** running on `127.0.0.1:6379` (for rate limiting)
- **OpenSSL** (to generate self-signed TLS certs for local development)

### 1. Clone & Install

```bash
git clone https://github.com/Ashish-Barmaiya/torus-proxy.git
cd torus-proxy
npm install
```

### 2. Generate TLS Certificates

```bash
mkdir -p certs
openssl req -x509 -newkey rsa:2048 -keyout certs/key.pem -out certs/cert.pem \
  -days 365 -nodes -subj "/CN=localhost"
```

### 3. Configure Routes

Edit `torus.yaml` to define your server port, routes, and upstream backends:

```yaml
server:
  port: 8080

routes:
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

### 4. Boot the Redis dependency

```bash
docker run -p 6379:6379 -d redis
```

### 5. Build & Run

```bash
npm run build
npm start
```

Torus will boot one worker per CPU core and begin accepting HTTPS traffic on the configured port.

---

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `NODE_ENV` | — | Set to `production` for raw NDJSON logs; otherwise pretty-prints. |
| `LOG_LEVEL` | `info` | Pino log level (`trace`, `debug`, `info`, `warn`, `error`, `fatal`). |

---

## Hot Reloading

Modify `torus.yaml` while Torus is running — **no restart needed**.

1. The master process detects the file change via `fs.watch()`.
2. It parses and **validates** the new config (invalid YAML is rejected, preserving the current state).
3. On success, it broadcasts a `RELOAD_CONFIG` IPC message to every worker.
4. Each worker atomically swaps its router and health checker — **zero connections are dropped**.

---

## Observability

### Prometheus Metrics

Scrape `https://localhost:8080/metrics` to access:

| Metric | Type | Description |
|---|---|---|
| `torus_http_requests_total` | Counter | Total requests processed, labeled by `method` and `status`. |
| `torus_http_request_duration_seconds` | Histogram | Request latency distribution (buckets from 5ms to 10s). |
| Default Node.js metrics | Gauge/Histogram | Event loop lag, V8 heap size, active handles, etc. |

### Logging

```jsonc
// Production (NODE_ENV=production) → NDJSON
{"level":30,"time":1710000000000,"pid":12345,"msg":"Worker listening for Secure HTTPS traffic","port":8080}

// Development → Pretty-printed
[18:07:40] INFO: Worker listening for Secure HTTPS traffic { port: 8080 }
```

---

## Benchmarks

Tested on a constrained **Intel i3-1115G4 (2C/4T), 8 GB RAM** — full results in [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md).

| Scenario | Throughput | Latency (P99) | Errors |
|---|---|---|---|
| Raw HTTP (no TLS, no Redis) | **1,647 req/s** | — | 0 |
| Full Production (TLS + Redis rate limiting) | **968 req/s** | 488 ms | 0 |

> On dedicated infrastructure, throughput scales linearly with physical core count.

---

## Project Structure

```
torus-proxy/
├── src/
│   ├── index.ts              # Entry point — cluster master/worker fork
│   ├── config/
│   │   ├── parser.ts         # YAML parser & router builder
│   │   └── __tests__/        # Unit tests for YAML parser     
│   ├── proxy/
│   │   └── server.ts         # HTTPS server, request handler, metrics endpoint
│   ├── routing/
│   │   ├── router.ts         # Longest-prefix-match route resolver
│   │   ├── pool.ts           # Backend pool with strategy-pattern load balancing
│   │   ├── roundRobin.ts     # Round-robin strategy implementation
│   │   ├── strategy.ts       # ILoadBalancingStrategy interface
│   │   ├── backend.ts        # BackendServer model (health state, connections)
│   │   ├── health.ts         # Active TCP health checker (node:net probes)
│   │   └── __tests__/        # Unit tests for router and round-robin
│   ├── security/
│   │   └── rateLimiter.ts    # Redis Lua Token Bucket rate limiter
│   └── utils/
│       ├── logger.ts         # Pino logger (JSON / pretty-print)
│       └── metrics.ts        # Prometheus registry, counters, histograms
├── certs/                    # TLS key + certificate (git-ignored)
├── docs/
│   ├── ARCHITECTURE.md       # Detailed architecture specification
│   └── BENCHMARKS.md         # Performance benchmarks
├── torus.yaml                # Proxy configuration
├── package.json
└── tsconfig.json
```

---

## Running Tests

```bash
npm run test
```

Tests use Jest with `ts-jest` and cover the routing engine and load-balancing strategies.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Runtime | Node.js (ES Modules) |
| Language | TypeScript (strict mode) |
| Clustering | `node:cluster` |
| TLS | `node:https` |
| Health Probes | `node:net` raw TCP sockets |
| Rate Limiting | Redis + Lua scripting |
| Metrics | Prometheus via `prom-client` |
| Logging | Pino + pino-pretty |
| Config | YAML (`yaml` package) |
| Testing | Jest + ts-jest |

---

## License

This project is licensed under the [ISC License](LICENSE).
