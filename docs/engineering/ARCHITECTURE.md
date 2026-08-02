# Torus Proxy — Architecture Specification

Torus Proxy is a Layer 7 reverse proxy and edge API gateway written in Go. The current implementation is the Go rewrite of an earlier Node.js/TypeScript prototype and is organized around a single-process runtime model with explicit health-aware routing and hot-reload support.

---

## 1. System topology

The proxy runs as a single OS process and uses Go's standard library networking stack rather than a multi-process worker model.

- The main entrypoint in [cmd/torus/main.go](./../../cmd/torus/main.go) wires together the server, runtime manager, and config watcher.
- The HTTP server is created once by [internal/proxy/server.go](./../../internal/proxy/server.go) and serves all traffic through a single shared runtime pointer.
- Each incoming request is handled in its own goroutine by the standard library HTTP server, which keeps the request path concurrent without manual worker management.
- The Node.js prototype remains under [node/](./../../node/) as a historical reference; the active implementation is the Go runtime described here.

---

## 2. Request handling pipeline

Requests follow a straightforward path through the proxy runtime:

```text
                Client
                   │
                   ▼
             net/http Server
                   │
                   ▼
          Acquire Runtime Generation
                   │
                   ▼
                 Router
                   │
                   ▼
                Service
                   │
                   ▼
     Round-Robin Backend Selection
                   │
                   ▼
     Instrumented Reverse Proxy
                   │
                   ▼
                Backend
                   │
                   ▼
        Release Runtime Generation
```

The handler in [internal/proxy/server.go](./../../internal/proxy/server.go) performs three key steps for each request:

1. It acquires a runtime reference from the currently published runtime generation.
2. It resolves the matching route, selects the corresponding service, and obtains the next healthy backend.
3. It forwards the request through the backend's instrumented reverse proxy.

If no route matches, the handler returns `404 Not Found`. If a route exists but no healthy backend is available, it returns `503 Service Unavailable`.

Request instrumentation is performed transparently by the backend transport and does not alter routing, backend selection, or request forwarding behaviour.

---

## 3. Runtime model

Torus uses immutable runtime generations rather than mutating routing tables in place.

The runtime struct in [internal/runtime/runtime.go](./../../internal/runtime/runtime.go) contains:

- a router
- a TLS configuration
- a cancellation context for runtime shutdown
- a wait group for in-flight request tracking
- a generation identifier

The runtime is published through an `atomic.Pointer` in [internal/proxy/server.go](./../../internal/proxy/server.go). New requests always operate against a single runtime generation from start to finish.

This design prevents partially applied configuration changes and keeps the request path safe while a reload is taking place. Runtime state remains immutable after publication, while operational metrics are exposed independently through the observability subsystem.

---

## 4. Routing engine

Routing is implemented in [internal/routing/router.go](./../../internal/routing/router.go).

### Matching strategy

The router performs a longest-prefix match with segment-boundary enforcement. A route matches only when:

- the request path is exactly the route key, or
- the next character after the route key is `/`

That means `/api-status` does not match a route for `/api`, which prevents false positives from simple prefix matching.

### Route structure

The router stores a map of route key to service. Each configured route points to a service object that owns its own load balancer and backend pool.

---

## 5. Service and load balancing

The service layer sits between routing and upstream selection.

- [internal/service/service.go](./../../internal/service/service.go) wraps a load balancer inside a service object.
- The current implementation uses a round-robin load balancer in [internal/loadbalancer/round_robin.go](./../../internal/loadbalancer/round_robin.go).

### Round-robin behavior

The balancer uses an `atomic.Uint64` index to choose the next backend in a goroutine-safe way. It scans the configured backend list and skips unhealthy backends. If all backends are unhealthy, it returns `nil`.

This keeps backend selection simple and deterministic while still allowing runtime health state to influence routing decisions.

---

## 6. Upstream proxying

Each backend is represented by [internal/upstream/backend.go](./../../internal/upstream/backend.go).

### Reverse proxy setup

Each backend creates a `httputil.ReverseProxy` backed by an instrumented HTTP transport tuned for sustained concurrency.

The underlying transport is configured with:

- `MaxIdleConns`: 10,000
- `MaxIdleConnsPerHost`: 2,000
- `IdleConnTimeout`: 90s
- `DialContext.Timeout`: 30s
- `DialContext.KeepAlive`: 30s
- `TLSHandshakeTimeout`: 10s
- `ResponseHeaderTimeout`: 5s
- `ExpectContinueTimeout`: 1s

### Rewrite behavior

The proxy rewrite function performs the following work before forwarding:

1. Sets the outbound URL scheme and host to the selected upstream.
2. Appends or creates an `X-Forwarded-For` header from the client address.
3. Sets `X-Forwarded-Proto` to `http` or `https` based on whether the incoming request was TLS.
4. Copies the original `Host` into `X-Forwarded-Host`.
5. Preserves or creates an `X-Request-ID` value using `github.com/google/uuid`.

Failures in the reverse proxy path are converted into `502 Bad Gateway` responses by the backend error handler.

### Observability instrumentation

Each backend wraps its transport with an instrumented transport that records request metrics during normal request forwarding.

When observability is enabled, the transport records:

- total requests
- request duration
- response status codes
- backend-specific request metrics

Instrumentation is transparent to the request path and does not modify routing, load balancing, or proxy semantics. When observability is disabled, requests bypass metric collection entirely.

---

## 7. Health checking

Health checking is active rather than purely passive.

The runtime builder in [internal/runtime/builder.go](./../../internal/runtime/builder.go) creates one health prober per backend. Each prober uses the HTTP checker defined in [internal/health/http.go](./../../internal/health/http.go) to issue a GET request to the configured health path.

### Health loop

The prober loop in [internal/health/checker.go](./../../internal/health/checker.go):

- runs periodically using the configured interval
- uses the configured timeout per probe
- marks the backend healthy on success
- marks it unhealthy on failure

### Recovery behavior

The prober has a built-in recovery mechanism:

- a primary recovery handles panics inside the probe loop and restarts it
- a secondary recovery protects the recovery path itself from crashing the process entirely

This keeps a single health-check bug from taking down the whole proxy process.

---

## 8. Observability

Observability is implemented as an optional subsystem that can be enabled through configuration.

When enabled, Torus exposes a Prometheus-compatible `/metrics` endpoint and publishes runtime, process, backend, and HTTP request metrics suitable for scraping by Prometheus and visualization with Grafana.

The subsystem exports metrics covering:

- HTTP request throughput
- request latency histograms
- response status codes
- backend health status
- Go runtime statistics
- process metrics

Metric collection is passive and does not participate in routing decisions, backend selection, or health checking.

The runtime overhead introduced by this subsystem is evaluated in Benchmark-003, which demonstrates that enabling observability introduces only a small and predictable performance cost while preserving runtime stability.

---

## 9. Configuration and TLS

Configuration is loaded from YAML through [internal/config/config.go](./../../internal/config/config.go).

The config schema includes:

- `server.addr`
- `health.interval_ms`
- `health.timeout_ms`
- `health.path`
- `routes` with one or more upstreams each
- optional `tls.cert_file`, `tls.key_file`, and `tls.min_version`
- optional `observability.enabled`

Validation ensures that:

- the server address is present
- health interval and timeout are valid
- the health path starts with `/`
- each route has a non-empty path and at least one upstream
- TLS settings are present and valid when TLS is enabled

When TLS is configured, the server wraps the listener with `tls.NewListener` so the proxy can terminate TLS directly.

---

## 10. Hot reload and configuration watcher

Hot reload is handled by the reload manager and the config watcher.

- [internal/reload/manager.go](./../../internal/reload/manager.go) builds a fresh runtime from the latest config.
- [internal/configwatcher/watcher.go](./../../internal/configwatcher/watcher.go) watches the config file's directory using `fsnotify`.
- Changes are debounced before a reload is triggered so a burst of writes does not cause repeated rebuilds.

### Reload sequence

1. The watcher detects a config change.
2. The reload manager parses and validates the YAML.
3. A new runtime is built from the updated configuration.
4. The proxy server swaps the runtime pointer.
5. The previous runtime is retired only after active requests have released their references.

This design supports zero-downtime reloads while keeping the runtime immutable across generations.

---

## 11. Server lifecycle and readiness

The server exposes operational endpoints in [internal/proxy/server.go](./../../internal/proxy/server.go).

- `/readyz` reports proxy readiness.
- `/metrics` exposes Prometheus metrics when observability is enabled.

`/readyz` returns:

- `200 OK` once the server has fully started.
- `503 Service Unavailable` before startup completes or during shutdown.

The shutdown path drains requests with a grace period and then forces cancellation if the graceful window expires.

---

## 12. Concurrency model

The current implementation relies on a small set of synchronization primitives:

- `atomic.Pointer` for runtime publication
- `sync.RWMutex` to protect the runtime swap window without blocking normal reads for long
- `sync.WaitGroup` to track in-flight requests during runtime retirement
- per-backend and per-request goroutines for health checks and request handling

This gives the proxy a safe and predictable concurrency model without introducing a multi-process coordination layer.

---

## 13. Current design direction

The present architecture favors the following properties:

- simplicity over feature breadth
- immutable runtime generations
- zero-downtime configuration reloads
- explicit ownership of routing and backend state
- optional built-in observability
- measurable and reproducible performance engineering
- strong separation between routing, balancing, health checking, and proxying
- use of Go's standard library rather than a larger framework

The repository's benchmarking and engineering documentation in [docs/benchmarking](../benchmarking/) and [docs/engineering](./) document the architecture, performance characteristics, and design decisions as the project evolves.
