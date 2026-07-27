# Runtime Architecture

Torus implements configuration reload through immutable runtime generations.

Rather than modifying routing tables, services, or backend state in place, every configuration change produces an entirely new runtime. Once successfully constructed, the new runtime atomically replaces the active runtime while the previous generation continues serving any requests already using it.

This design enables zero-downtime configuration reload while maintaining a simple concurrency model and minimizing synchronization on the request path.

---

# Motivation

A configuration reload affects nearly every request-processing component inside the proxy.

A reload may change:

- routing rules
- services
- backend pools
- load balancers
- reverse proxies
- health checkers
- TLS configuration

Updating these components individually would require coordinating multiple shared data structures across many concurrent goroutines.

Such an approach introduces several challenges:

- partially updated state
- inconsistent routing behavior
- complex synchronization
- increased risk of data races
- blocking request processing during reload

Instead of mutating shared state, Torus treats the entire request-processing pipeline as a single immutable runtime.

---

# Design Goals

The runtime architecture was designed to satisfy the following objectives.

- Zero-downtime configuration reload.
- No interruption of in-flight requests.
- Consistent view of runtime state throughout request execution.
- Minimal synchronization overhead.
- Predictable runtime lifecycle.
- Straightforward reasoning about concurrency.

---

# Runtime Generation

A runtime generation represents a complete snapshot of the proxy configuration.

Each runtime contains all objects required to process requests, including:

- Router
- Services
- Load balancers
- Backend pools
- Reverse proxies
- Health checker manager
- Configuration metadata

After construction, a runtime is never modified.

Every request executes entirely against one runtime generation.

---

# Runtime Construction

Configuration reload follows a build-before-publish model.

```
Configuration File
        │
        ▼
Configuration Parser
        │
        ▼
Runtime Builder
        │
        ▼
Complete Runtime
```

The runtime builder constructs every component before the runtime becomes visible to request-handling goroutines.

If construction fails for any reason, the existing runtime remains active.

---

# Runtime Publication

Once the new runtime has been successfully constructed, it replaces the active runtime through a single atomic pointer update.

```
Runtime Generation N
          │
          ▼
atomic.Pointer
          ▲
          │
Runtime Generation N+1
```

Publishing a runtime is therefore an atomic operation.

Requests arriving before publication use the previous runtime.

Requests arriving afterward use the new runtime.

No request can observe a partially updated runtime.

---

# Request Lifecycle

Each request acquires a reference to the active runtime before request processing begins.

```
 Incoming Request
        │
        ▼
 Acquire Runtime
        │
        ▼
     Routing
        │
        ▼
Service Selection
        │
        ▼
  Load Balancer
        │
        ▼
  Reverse Proxy
        │
        ▼
     Backend
        │
        ▼
 Release Runtime
```

The acquired runtime remains valid for the entire lifetime of the request.

Even if a configuration reload occurs during execution, the request continues using the same runtime generation.

---

# Runtime Lifetime Management

Runtime publication does not immediately destroy the previous runtime.

Existing requests may still hold references to it.

Each runtime therefore maintains an internal reference count.

```
Acquire()
    │
Reference Count++
    │

  .....

Request Completes

  .....

Release()
    │
Reference Count--
```

A runtime is eligible for retirement only after:

- it has been replaced by a newer generation
- its reference count reaches zero

This guarantees that no request can observe released resources.

---

# Concurrency Model

The runtime architecture relies on three synchronization mechanisms.

## atomic.Pointer

The currently active runtime is published through an atomic pointer.

Runtime replacement therefore requires only a single atomic operation.

---

## sync.RWMutex

A narrow synchronization window exists between:

1. loading the active runtime pointer
2. acquiring a request reference

Without coordination, a runtime could be retired after the pointer is loaded but before the request increments the reference count.

A `sync.RWMutex` protects this transition.

Request processing briefly acquires a read lock while loading and acquiring the runtime.

Configuration reload acquires the write lock only during runtime publication and retirement.

The lock is intentionally held for a very small critical section, minimizing impact on request throughput.

---

## Reference Counting

Reference counting ensures that runtime retirement is deferred until all active requests complete.

This mechanism allows configuration reload and request execution to proceed concurrently without invalidating active runtime state.

---

# Zero-Downtime Reload

Configuration reload proceeds through four phases.

```
Parse Configuration
        │
        ▼
   Build Runtime
        │
        ▼
Atomic Publication
        │
        ▼
Graceful Retirement
```

If runtime construction fails, no publication occurs.

The currently active runtime continues serving traffic without interruption.

---

# Graceful Shutdown

Shutdown operates independently from runtime replacement.

The shutdown sequence is:

1. Stop accepting new connections.
2. Mark the proxy as not ready.
3. Allow active requests to complete.
4. Release remaining runtime references.
5. Shut down background components.

This ensures that in-flight requests complete successfully while preventing new traffic from entering the proxy.

---

# Testing Strategy

The runtime architecture is validated through multiple layers of testing.

## Unit Tests

Validate individual runtime components in isolation.

---

## Component Tests

Verify routing, load balancing, health checking, and runtime behavior.

---

## Integration Tests

Exercise the complete production startup path.

The integration suite verifies:

- functional runtime reload
- concurrent runtime replacement
- graceful shutdown
- request draining

---

## Race Detection

The entire test suite is executed with Go's race detector enabled.

Particular attention is given to concurrent runtime replacement while the proxy is actively serving requests.

---

# Summary

The runtime architecture is based on one central principle:

> Never modify the runtime currently serving requests.

Instead, Torus constructs a complete replacement runtime, publishes it atomically, and gracefully retires the previous generation after all active requests have completed.

This approach provides predictable concurrent behavior, simplifies reasoning about synchronization, and enables zero-downtime configuration reload while keeping the request path efficient.
