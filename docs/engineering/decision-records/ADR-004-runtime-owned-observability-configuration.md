# ADR-004: Runtime-Owned Observability Configuration

**Status:** Accepted

**Decision Date:** 2026-08-02

**Recorded:** 2026-08-02

**Authors:** Ashish Barmaiya

---

# Context

Torus follows an immutable runtime model in which every configuration reload constructs a new runtime generation and publishes it through a single atomic pointer swap.

Each runtime owns all configuration required for request processing, including routing tables, services, load balancers, backend pools, health checkers, and TLS configuration. Once published, a runtime generation is never modified.

The initial implementation of the observability subsystem diverged from this architecture.

Whether metrics were collected was controlled by a package-level mutable flag inside the `observability` package. During configuration reloads, the reload manager updated this global flag before constructing the next runtime generation.

At the same time, request-handling goroutines continuously consulted the same flag while recording HTTP, backend, and runtime metrics.

This introduced shared mutable state outside the runtime lifecycle.

The design produced several undesirable properties:

- concurrent reads and writes against shared configuration
- race detector failures during reload testing
- shared observability state between independent runtime generations
- inconsistent architectural ownership compared to the rest of the runtime

Although the implementation functioned correctly during normal execution, it violated the immutable runtime model established by [ADR-003](./ADR-003-use-immutable-runtime-generations-for-configuration-reload).

A new ownership model for observability configuration was therefore required.

---

# Decision

Make observability configuration an immutable property of each runtime generation.

Each runtime now owns whether observability is enabled through its own immutable configuration.

Instrumentation decisions are made using the runtime acquired by the current request rather than consulting shared global state.

The Prometheus registry, collectors, metric definitions, and `/metrics` handler remain process-wide infrastructure and are not recreated during configuration reloads.

Only the decision of whether instrumentation is active belongs to the runtime.

No mutable global observability configuration exists after publication.

---

# Alternatives Considered

## Option 1 — Global Mutable Configuration

### Description

Maintain observability state as package-level mutable configuration.

Configuration reloads update a global `enabled` flag while request goroutines consult the same value during metric collection.

### Advantages

- Simple implementation.
- Minimal additional runtime state.
- Centralized configuration.

### Disadvantages

- Shared mutable state across runtime generations.
- Concurrent reads and writes introduce data races.
- Runtime behavior may change while older runtime generations are still serving requests.
- Test isolation becomes more difficult.
- Violates the immutable runtime model established by ADR-003.

---

## Option 2 — Runtime-Owned Observability Configuration (Accepted)

### Description

Store observability configuration as immutable runtime state.

Each runtime generation determines whether instrumentation is active for requests executing against that runtime.

The Prometheus registry and collectors remain process-wide.

### Advantages

- Eliminates shared mutable configuration.
- Preserves immutable runtime ownership.
- Removes race conditions caused by configuration reload.
- Existing runtime generations remain internally consistent throughout their lifetime.
- Minimal implementation complexity.
- Preserves the existing Prometheus integration.

### Disadvantages

- Instrumentation call sites perform an explicit conditional check before recording metrics.
- Observability decisions become distributed across instrumentation points rather than centralized inside the observability package.

---

## Option 3 — Runtime-Owned Recorder Interface

### Description

Associate each runtime with an observability recorder implementation.

Instrumentation sites would invoke recorder methods without checking configuration directly.

Possible implementations include:

- Prometheus recorder
- No-op recorder

### Advantages

- Eliminates conditional logic from instrumentation call sites.
- Provides a clean dependency-injection model.
- Supports multiple telemetry backends without modifying request handling code.
- Naturally extends to future observability implementations.

### Disadvantages

- Introduces additional abstraction.
- Requires a relatively large recorder interface covering all instrumentation operations.
- Adds implementation complexity beyond current project requirements.

### Decision

Deferred.

The additional abstraction is not currently justified because Torus supports only a single observability backend (Prometheus). The runtime-owned configuration adopted by this ADR satisfies current requirements while preserving the option to introduce a recorder abstraction if future telemetry backends require it.

---

# Decision Drivers

The decision was primarily influenced by the following architectural requirements.

| Driver | Importance |
|---------|------------|
| Immutable runtime ownership | Critical |
| Thread safety | Critical |
| Zero-downtime configuration reload | Critical |
| Runtime consistency | High |
| Test isolation | High |
| Implementation simplicity | Medium |
| Future extensibility | Medium |

---

# Rationale

ADR-003 established immutable runtime generations as the fundamental concurrency model used throughout Torus.

Every request executes against a single runtime generation that remains unchanged until the request completes.

The original observability implementation was the only subsystem whose behavior could change independently of the runtime lifecycle through mutation of global configuration.

Moving observability configuration into the runtime restores architectural consistency across the system.

Requests now determine whether instrumentation is active by consulting the runtime generation they acquired at the beginning of request processing.

Older runtime generations continue using their own observability configuration until retirement, while newly published runtimes immediately begin using their own configuration.

The Prometheus registry and collectors remain process-wide because they represent shared infrastructure rather than runtime configuration. Separating configuration ownership from metrics infrastructure preserves the existing Prometheus integration while eliminating mutable global state.

Although a runtime-owned recorder abstraction offers greater long-term flexibility, the current runtime-owned configuration provides a substantially simpler implementation while satisfying all present requirements.

---

# Consequences

## Positive

- Eliminates mutable global observability configuration.
- Removes race conditions caused by concurrent configuration updates.
- Restores consistency with the immutable runtime architecture.
- Improves test isolation.
- Simplifies reasoning about runtime ownership.
- Preserves existing Prometheus infrastructure.

---

## Negative

- Instrumentation call sites contain explicit configuration checks.
- Observability logic is no longer centralized behind a single package-level decision point.

---

## Trade-offs

This decision favors architectural consistency and correctness over introducing an additional abstraction layer.

The implementation retains simple runtime-owned configuration while deferring a recorder-based dependency injection model until multiple telemetry backends or more advanced observability requirements justify the additional complexity.

---

# Validation

The implementation was validated through:

- Successful execution of the complete test suite.
- Successful execution of the complete test suite with Go's race detector enabled.
- Repeated shuffled race-detector execution (`go test -race -count=100 -shuffle=on ./...`).
- Integration tests covering configuration reload.
- Integration tests covering concurrent request processing during runtime replacement.

These tests demonstrate that observability configuration no longer introduces observable data races while preserving runtime correctness throughout configuration reloads.

---

# Related Documents

## Architecture

- [`docs/engineering/ARCHITECTURE.md`](../ARCHITECTURE.md)

## Architecture Decision Records

- [`ADR-003: Use Immutable Runtime Generations for Configuration Reload`](./ADR-003-use-immutable-runtime-generations-for-configuration-reload)

## Source

- `internal/runtime/`
- `internal/observability/`
- `internal/proxy/`
- `internal/upstream/`
- `internal/reload/`

---

# Notes

This decision extends the immutable runtime ownership model introduced by ADR-003 to observability configuration.

Future observability enhancements should preserve this ownership model.

If Torus later supports multiple telemetry backends, the runtime-owned configuration adopted by this ADR may evolve into a runtime-owned recorder abstraction without changing the underlying immutable runtime architecture.
