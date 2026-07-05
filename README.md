<div align="center">

# Torus Proxy

A Go-based Layer 7 reverse proxy and edge API gateway.

Torus routes HTTP traffic to upstream services, performs health-aware load balancing, and can terminate TLS while keeping the implementation dependency-light and focused on the standard library.

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

</div>

---

## What the project does

The current implementation is a working Go proxy with the following capabilities:

- Reverse proxying with Go's standard-library `net/http/httputil.ReverseProxy`
- Longest-prefix route matching with path-segment boundaries
- Round-robin load balancing across configured upstreams
- Active health probing for each backend with configurable interval and timeout
- Automatic header injection for forwarded client information and request tracing
- Optional TLS termination from YAML config
- Structured request logging and a readiness endpoint
- Graceful shutdown on SIGINT/SIGTERM

The original Node.js/TypeScript prototype remains in the [node/](node/) directory as a reference and benchmark comparison.

---

## Current architecture

The proxy startup flow is:

1. Load YAML config from `torus.yaml`
2. Create backends and start health check probers for each upstream
3. Build the router and bind routes to services
4. Start the HTTP server and serve requests through the reverse proxy pipeline

The main runtime pieces are:

- [cmd/torus/main.go](cmd/torus/main.go) for startup, signal handling, and lifecycle coordination
- [internal/config/config.go](internal/config/config.go) for configuration parsing
- [internal/routing/router.go](internal/routing/router.go) for route resolution
- [internal/service/service.go](internal/service/service.go) and [internal/loadbalancer/round_robin.go](internal/loadbalancer/round_robin.go) for service and balancing logic
- [internal/proxy/server.go](internal/proxy/server.go) for the HTTP server and request handling
- [internal/health](internal/health) for health probing
- [internal/upstream/backend.go](internal/upstream/backend.go) for backend and proxy setup

---

## Quick start

### Prerequisites

- Go 1.26.1 or newer
- One or more backend HTTP services to proxy to
- Optional: TLS certificate and key files if you want to serve HTTPS

### 1. Build the binary

```bash
git clone https://github.com/Ashish-Barmaiya/torus-proxy.git
cd torus-proxy
go build -o torus ./cmd/torus
```

### 2. Configure routes

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

### 3. Run the proxy

```bash
./torus
```

The server listens on the configured address and exposes a readiness endpoint at `/readyz`.

---

## Example behavior

A request such as:

```bash
curl http://localhost:8080/api/hello
```

will be routed to the configured upstreams using the service's round-robin selection while preserving request headers and tracing metadata.

---

## Testing

Run the test suite with:

```bash
go test ./...
```

The repository also includes race-enabled checks in CI and supports:

- routing behavior
- backend selection and failover
- health-check status changes
- proxy request handling and error responses

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
├── node/                 # Reference Node.js/TypeScript implementation
├── docs/
├── torus.yaml
├── mock_backend.go
└── go.mod
```

---

## License

This project is licensed under the [MIT License](LICENSE).
