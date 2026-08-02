# ADR-003: Use Immutable Runtime Generations for Configuration Reload

**Status:** Accepted

**Decision Date:** 2026-07-24

**Recorded:** 2026-07-27

**Authors:** Ashish Barmaiya

---

# Context

Torus supports configuration-driven routing, load balancing, health checking, and TLS termination.

As the project evolved, configuration reload became a required capability. The proxy needed to apply configuration changes without interrupting active traffic or requiring a process restart.

A configuration reload affects multiple interconnected components, including:

- routing tables
- services
- load balancers
- backend pools
- reverse proxies
- health checkers
- TLS configuration

These components are accessed concurrently by many request-handling goroutines.

Updating them individually would expose the system to partially updated state, inconsistent request routing, and data races. Any synchronization strategy also needed to preserve high request throughput without introducing unnecessary contention on the request path.

A runtime management strategy was therefore required.

---

# Decision

Represent the running proxy as an immutable runtime generation.

Each configuration reload constructs an entirely new runtime from the updated configuration.

Once the new runtime has been successfully built and validated, it replaces the current runtime through a single atomic pointer swap.

The previous runtime remains available for in-flight requests until all active references have been released, after which it is gracefully retired.

No runtime is modified after publication.

---

# Alternatives Considered

## Option 1 — Mutate Existing Runtime In Place

### Advantages

- Lower temporary memory usage.
- No duplicate runtime during reload.
- Simpler object lifecycle.

### Disadvantages

- Complex synchronization across multiple shared structures.
- Risk of partially updated state.
- Increased likelihood of data races.
- More difficult to reason about under concurrent request execution.

---

## Option 2 — Global Reload Lock

### Advantages

- Straightforward implementation.
- Guarantees consistency during reload.

### Disadvantages

- Blocks request processing during configuration updates.
- Increased request latency during reload.
- Reduced scalability under concurrent traffic.
- Reload duration directly impacts client requests.

---

## Option 3 — Immutable Runtime Generations

### Advantages

- Requests always observe a consistent runtime.
- Zero-downtime configuration reload.
- Simple concurrency model.
- Minimal synchronization on the request path.
- Clear runtime lifecycle.

### Disadvantages

- Temporary increase in memory usage during reload.
- Additional implementation complexity for runtime lifetime management.
- Requires explicit retirement of previous runtime generations.

---

# Decision Drivers

The decision was primarily influenced by the following characteristics of the workload.

| Driver | Importance |
|---------|------------|
| Zero-downtime reload | Critical |
| Thread safety | Critical |
| Request throughput | Critical |
| Architectural simplicity | High |
| Predictable runtime behavior | High |

---

# Rationale

Treating the runtime as an immutable object significantly simplifies concurrent request processing.

Each request acquires a reference to a single runtime generation before routing begins and continues using that same generation until the request completes.

Configuration reloads never modify data structures currently being used by active requests. Instead, they construct an entirely new runtime and publish it atomically.

This approach eliminates the possibility of requests observing partially updated routing state or inconsistent backend configuration.

Although the design temporarily increases memory usage during reload, reload operations are expected to occur infrequently compared to request processing. The additional memory overhead is therefore considered an acceptable trade-off for improved correctness, simpler synchronization, and predictable behavior.

---

# Consequences

## Positive

- Zero-downtime configuration reload.
- Requests always execute against a consistent runtime generation.
- Simplified concurrency model.
- Reduced risk of data races.
- Clear separation between runtime construction and runtime execution.
- Easier testing of reload behavior.

---

## Negative

- Additional memory required while two runtime generations coexist.
- Runtime lifetime management becomes more complex.
- Reload implementation requires reference counting and graceful retirement.

---

## Trade-offs

This decision favors correctness, maintainability, and predictable concurrent behavior over minimizing temporary memory usage.

The implementation introduces additional runtime management infrastructure but substantially reduces the complexity of synchronization during request processing.

---

# Validation

The implementation was validated through:

- Functional integration tests verifying runtime replacement.
- Concurrent integration tests performing repeated reloads under active traffic.
- Graceful shutdown tests verifying interaction with in-flight requests.
- Successful execution of the complete test suite with Go's race detector enabled.

These tests demonstrate that runtime replacement occurs without interrupting active requests while remaining free of observable data races.

---

# Related Documents

## Architecture

- [`docs/engineering/ARCHITECTURE.md`](../ARCHITECTURE.md)

## Engineering Documentation

- [`docs/engineering/RUNTIME_ARCHITECTURE.md`](../RUNTIME_ARCHITECTURE.md)

## Source

- `internal/runtime/`s
- `internal/reload/`
- `internal/proxy/`

---

# Notes

This decision establishes immutable runtime generations as the foundation for configuration management in Torus.

Future features involving runtime state, including service discovery, dynamic configuration sources, and runtime optimization, should preserve this model by constructing and publishing new runtime generations rather than mutating the active runtime in place.
