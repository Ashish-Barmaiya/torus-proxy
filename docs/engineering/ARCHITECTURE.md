# Torus Proxy — Architecture Specification

Torus Proxy is a Layer 7 reverse proxy and edge API gateway written in Go. The current implementation is the Go rewrite of an earlier Node.js/TypeScript prototype and is organized around a single-process runtime model with explicit health-aware routing and hot-reload support.

---

## 1. System topology

The proxy runs as a single OS process and uses Go's standard library networking stack rather than a multi-process worker model.

- The main entrypoint in [cmd/torus/main.go](cmd/torus/main.go) wires together the server, runtime manager, and config watcher.
- The HTTP server is created once by [internal/proxy/server.go](internal/proxy/server.go) and serves all traffic through a single shared runtime pointer.
- Each incoming request is handled in its own goroutine by the standard library HTTP server, which keeps the request path concurrent without manual worker management.
- The Node.js prototype remains under [node/](node/) as a historical reference; the active implementation is the Go runtime described here.

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

              Round-Robin
                   │
                   ▼

              ReverseProxy
                   │
                   ▼

                Backend
                   │
                   ▼

        Release Runtime Generation
```

The handler in [internal/proxy/server.go](internal/proxy/server.go) performs three key steps for each request:

1. It acquires a runtime reference from the currently published runtime generation.
2. It resolves the matching route and picks the next healthy backend.
3. It forwards the request through the backend's reverse proxy.

If no route matches, the handler returns `404 Not Found`. If a route exists but no healthy backend is available, it returns `503 Service Unavailable`.

---

## 3. Runtime model

Torus uses immutable runtime generations rather than mutating routing tables in place.

The runtime struct in [internal/runtime/runtime.go](internal/runtime/runtime.go) contains:

- a router
- a TLS configuration
- a cancellation context for runtime shutdown
- a wait group for in-flight request tracking
- a generation identifier

The runtime is published through an `atomic.Pointer` in [internal/proxy/server.go](internal/proxy/server.go). New requests always operate against a single runtime generation from start to finish.

This design prevents partially applied config changes and keeps the request path safe while a reload is taking place.

---

## 4. Routing engine

Routing is implemented in [internal/routing/router.go](internal/routing/router.go).

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

- [internal/service/service.go](internal/service/service.go) wraps a load balancer inside a service object.
- The current implementation uses a round-robin load balancer in [internal/loadbalancer/round_robin.go](internal/loadbalancer/round_robin.go).

### Round-robin behavior

The balancer uses an `atomic.Uint64` index to choose the next backend in a goroutine-safe way. It scans the configured backend list and skips unhealthy backends. If all backends are unhealthy, it returns `nil`.

This keeps backend selection simple and deterministic while still allowing runtime health state to influence routing decisions.

---

## 6. Upstream proxying

Each backend is represented by [internal/upstream/backend.go](internal/upstream/backend.go).

### Reverse proxy setup

Each backend creates a `httputil.ReverseProxy` with a custom `http.Transport` tuned for sustained concurrency:

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

---

## 7. Health checking

Health checking is active rather than purely passive.

The runtime builder in [internal/runtime/builder.go](internal/runtime/builder.go) creates one health prober per backend. Each prober uses the HTTP checker defined in [internal/health/http.go](internal/health/http.go) to issue a GET request to the configured health path.

### Health loop

The prober loop in [internal/health/checker.go](internal/health/checker.go):

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

## 8. Configuration and TLS

Configuration is loaded from YAML through [internal/config/config.go](internal/config/config.go).

The config schema includes:

- `server.addr`
- `health.interval_ms`
- `health.timeout_ms`
- `health.path`
- `routes` with one or more upstreams each
- optional `tls.cert_file`, `tls.key_file`, and `tls.min_version`

Validation ensures that:

- the server address is present
- health interval and timeout are valid
- the health path starts with `/`
- each route has a non-empty path and at least one upstream
- TLS settings are present and valid when TLS is enabled

When TLS is configured, the server wraps the listener with `tls.NewListener` so the proxy can terminate TLS directly.

---

## 9. Hot reload and configuration watcher

Hot reload is handled by the reload manager and the config watcher.

- [internal/reload/manager.go](internal/reload/manager.go) builds a fresh runtime from the latest config.
- [internal/configwatcher/watcher.go](internal/configwatcher/watcher.go) watches the config file's directory using `fsnotify`.
- Changes are debounced before a reload is triggered so a burst of writes does not cause repeated rebuilds.

### Reload sequence

1. The watcher detects a config change.
2. The reload manager parses and validates the YAML.
3. A new runtime is built from the updated configuration.
4. The proxy server swaps the runtime pointer.
5. The previous runtime is retired only after active requests have released their references.

This design supports zero-downtime reloads while keeping the runtime immutable across generations.

---

## 10. Server lifecycle and readiness

The server exposes a readiness endpoint at `/readyz` in [internal/proxy/server.go](internal/proxy/server.go).

- `200 OK` is returned once the server has fully started.
- `503 Service Unavailable` is returned before startup completes or during shutdown.

The shutdown path drains requests with a grace period and then forces cancellation if the graceful window expires.

---

## 11. Concurrency model

The current implementation relies on a small set of synchronization primitives:

- `atomic.Pointer` for runtime publication
- `sync.RWMutex` to protect the runtime swap window without blocking normal reads for long
- `sync.WaitGroup` to track in-flight requests during runtime retirement
- per-backend and per-request goroutines for health checks and request handling

This gives the proxy a safe and predictable concurrency model without introducing a multi-process coordination layer.

---

## 12. Current design direction

The present architecture favors the following properties:

- simplicity over feature breadth
- explicit, testable runtime boundaries
- safe configuration reloads
- strong separation between routing, balancing, health checking, and proxying
- use of Go's standard library rather than a larger framework

The repository's benchmarking and engineering documentation in [docs/benchmarking](docs/benchmarking) and [docs/engineering](docs/engineering) continue to document performance and design decisions as the code evolves.
