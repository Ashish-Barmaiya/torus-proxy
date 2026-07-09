# ADR-002: Use `atomic.Bool` for Backend Health State

**Status:** Accepted

**Decision Date:** 2026-06-07

**Recorded:** 2026-07-09

**Authors:** Ashish Barmaiya

---

# Context

Each backend managed by Torus maintains a health state indicating whether it is eligible to receive traffic.

This state is accessed from multiple goroutines concurrently.

The primary access patterns are:

- Health checker goroutines periodically update the state.
- Every incoming request reads the state during backend selection.
- Multiple concurrent requests may read the value simultaneously.

Because backend health directly influences request routing, the implementation must be:

- thread-safe
- low-overhead
- simple to reason about

A synchronization strategy was therefore required.

---

# Decision

Represent backend health using Go's `atomic.Bool`.

```go
type Backend struct {
    URL     string
    Proxy   *httputil.ReverseProxy
    healthy atomic.Bool
}
```

Health state updates are performed using `Store()` and routing decisions read the state using `Load()`.

---

# Alternatives Considered

## Option 1 — `sync.RWMutex`

### Advantages

- Familiar synchronization primitive.
- Suitable for protecting multiple related fields.
- Flexible for future complex state.

### Disadvantages

- Every read requires lock acquisition.
- Introduces additional synchronization overhead.
- Can increase contention under high request concurrency.
- More complex than necessary for a single boolean value.

---

## Option 2 — `atomic.Bool`

### Advantages

- Lock-free reads and writes.
- Minimal synchronization overhead.
- Well suited for independent scalar state.
- Simple implementation.
- Scales efficiently under read-heavy workloads.

### Disadvantages

- Applicable only to individual atomic values.
- Cannot coordinate updates across multiple fields.
- Requires understanding of atomic memory operations.

---

# Decision Drivers

The decision was primarily influenced by the following characteristics of the workload.

| Driver | Importance |
|---------|------------|
| Read performance | Critical |
| Low synchronization overhead | High |
| Simplicity | High |
| Thread safety | Critical |
| Scalability | High |

---

# Rationale

The backend health flag represents a single independent piece of state.

It does not participate in larger transactions and does not require coordination with other fields.

The access pattern is heavily skewed toward reads.

For every health check update, the value may be read thousands of times by concurrent requests.

Under these conditions, a lock-based synchronization primitive would provide no practical benefit while introducing unnecessary synchronization overhead.

`atomic.Bool` performs atomic loads and stores without requiring mutex acquisition, making it well suited for this workload.

The resulting implementation is both simpler and more efficient.

---

# Consequences

## Positive

- Lock-free access to backend health state.
- Lower synchronization overhead.
- Simpler implementation.
- Better scalability under high request concurrency.
- Clear expression of intent that the field represents independent atomic state.

---

## Negative

- Limited to a single value.
- Cannot be extended to protect multiple related fields.
- Requires familiarity with Go's atomic primitives.

---

## Trade-offs

This decision optimizes a specific synchronization problem rather than providing a general synchronization mechanism.

If backend state evolves into a more complex structure requiring coordinated updates across multiple fields, a different synchronization strategy may become necessary.

---

# Validation

The decision was supported by investigation into:

- Go's `sync/atomic` package
- Go memory model
- CPU atomic instructions
- Lock contention characteristics
- Read-heavy synchronization patterns

The current workload strongly favors atomic operations because backend health is updated infrequently but read on every routing decision.

---

# Related Documents

## Architecture

- [`docs/engineering/ARCHITECTURE.md`](../ARCHITECTURE.md)

## Source

- [`internal/upstream/backend.go`](../../../internal/upstream/backend.go)

## Planned Article

- Why Torus Uses `atomic.Bool` Instead of `sync.RWMutex`
