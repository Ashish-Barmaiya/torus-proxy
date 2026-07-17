# Torus Proxy — Architecture Specification

Torus Proxy is a Layer 7 Reverse Proxy and Edge API Gateway built in Go. Originally prototyped in Node.js/TypeScript, the project was rewritten in Go to eliminate the single-threaded event loop bottleneck, leverage goroutine-based concurrency, and use the standard library's optimized networking primitives.

---

## 1. System Topology (Single-Process, Goroutine-Per-Connection)

Unlike the Node.js version, which required `node:cluster` to fork one OS process per CPU core (4 processes × ~50MB = ~200MB RAM), the Go implementation runs as a **single OS process**. Each incoming connection spawns a lightweight goroutine (~2KB stack), managed by Go's M:N scheduler.

* **Single `net/http.Server`:** Listens on the configured address (`:8080`). Go's `net/http` internally uses goroutines to handle each connection, so no manual worker management is needed.
* **No IPC, no cluster module, no multi-process coordination.** Hot reload becomes a simple atomic pointer swap instead of a broadcast + re-parse across N workers.
* **Memory footprint:** ~25MB average under benchmark load (10× less than the Node.js cluster).

---

## 2. The Reverse Proxy Pipeline

Torus uses Go's `net/http/httputil.ReverseProxy` (stdlib) to forward requests to upstream backends.

### Request Flow

```
Client → net/http.Server → Router (longest-prefix match)
       → Service (round-robin load balancer)
       → httputil.ReverseProxy.ServeHTTP()
       → Backend (with custom Rewrite + Transport)
```

### Custom Rewrite Function

Each `Backend` configures a `Rewrite` function on its `ReverseProxy` that:

1. Sets `URL.Scheme` and `URL.Host` to the target backend.
2. Extracts the client IP and injects/appends `X-Forwarded-For`.
3. Sets `X-Forwarded-Proto` (`http` or `https` based on TLS state).
4. Sets `X-Forwarded-Host` from the original `Host` header.
5. Generates a `X-Request-ID` (UUID v4 via `github.com/google/uuid`) if none is present.

### Tuned Connection Pool

Each backend's `httputil.ReverseProxy` uses a custom `http.Transport`:

| Parameter | Value | Purpose |
|---|---|---|
| `MaxIdleConns` | 10,000 | Total idle connections across all hosts |
| `MaxIdleConnsPerHost` | 2,000 | Per-backend idle connection cap |
| `IdleConnTimeout` | 90s | Reclaim stale connections |
| `DialContext.Timeout` | 30s | TCP dial timeout |
| `DialContext.KeepAlive` | 30s | TCP keepalive interval |
| `TLSHandshakeTimeout` | 10s | TLS negotiation deadline |
| `ResponseHeaderTimeout` | 5s | Time to wait for response headers |
| `ExpectContinueTimeout` | 1s | Wait before sending the request body after receiving `100-continue` |

This ensures sustained high throughput under load without exhausting file descriptors or leaking connections.

---

## 3. The Routing Engine

Torus uses a **Longest-Prefix Match** algorithm with **segment boundary enforcement**.

```go
// Only matches if the path is an exact match OR the next character is '/'
if len(path) == len(routeKey) || path[len(routeKey)] == '/' {
    // match
}
```

This prevents false positives: `/api-status` does **not** match a route for `/api`, because `-` is not a path separator. This is stricter than the original Node.js implementation, which used a simple `startsWith()` check.

---

## 4. Load Balancing

The `LoadBalancer` interface decouples the balancing strategy from the routing layer:

```go
type LoadBalancer interface {
    Next() *upstream.Backend
}
```

### Round-Robin Implementation

* Uses `sync/atomic.Uint64` for the index — **lock-free, goroutine-safe**.
* Automatically **skips unhealthy backends**: iterates through the pool up to `len(backends)` times, returning `nil` only if every backend is down.
* The `Service` layer binds a set of backends to a `RoundRobin` instance and exposes `NextBackend()` to the router.

---

## 5. Active Health Checking

Passive health checking (waiting for a user request to hit a dead server and fail) is unacceptable. Torus employs **active probing**.

### How It Works

1. For each backend, a dedicated goroutine runs a `StartProber` loop.
2. Every 5 seconds, the prober sends an HTTP `GET` to the backend's `/health` endpoint with a 2-second timeout.
3. If the probe succeeds (`200 OK`), the backend is marked healthy via `atomic.Bool`.
4. If the probe fails (timeout, connection refused, non-200 status), the backend is marked unhealthy and immediately skipped by the round-robin balancer.

### Self-Healing Probers

The health prober goroutine includes a **double-recovery mechanism**:

* **Primary recovery:** If the prober panics, it logs a crash alert, waits 2 seconds, and recursively restarts itself.
* **Secondary recovery:** If the recovery process itself panics, it logs a critical system fault and halts gracefully (no infinite crash loop).

This ensures that a bug in the health check logic cannot crash the entire proxy process.

---

## 6. Server Timeouts

The `net/http.Server` is configured with defensive timeouts to guard against slow clients and resource exhaustion:

| Timeout | Value | Purpose |
|---|---|---|
| `ReadTimeout` | 5s | Maximum time to read the entire request (headers + body) |
| `WriteTimeout` | 10s | Maximum time to write the response |
| `IdleTimeout` | 120s | Maximum time a keep-alive connection can remain idle |

---

## 7. What Changed from Node.js

| Aspect | Node.js (archived in `node/`) | Go (current) |
|---|---|---|
| Concurrency model | `node:cluster` — 4 OS processes, IPC broadcasts | Goroutines — 1 process, shared memory |
| Stream piping | `stream.pipe()` via libuv event loop | `httputil.ReverseProxy` with potential `splice(2)` |
| Health checks | Raw TCP sockets (`node:net`) | HTTP GET with `net/http.Client` |
| Load balancing | `RoundRobinStrategy` class | `RoundRobin` struct with `sync/atomic` |
| Memory footprint | ~200MB (4 workers) | ~25MB average under benchmark load |
| External dependencies | 8+ npm packages | 1 (`github.com/google/uuid`) |
| HTTP throughput | 1,647 req/s | **15,617 req/s** |
| HTTPS throughput | n/a | **14,595 req/s** |
