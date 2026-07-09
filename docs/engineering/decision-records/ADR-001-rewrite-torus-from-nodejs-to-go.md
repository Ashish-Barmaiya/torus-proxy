# ADR-001: Rewrite Torus from Node.js to Go

**Status:** Accepted

**Decision Date:** Early April 2026

**Recorded:** 2026-07-09

**Authors:** Ashish Barmaiya

---

# Context

Torus was originally implemented in Node.js and TypeScript as a learning project to understand the internals of reverse proxies and API gateways.

Over time, the project expanded beyond basic request forwarding and began incorporating production-oriented capabilities such as active health checking, load balancing, TLS termination, structured logging, and graceful shutdown. The long-term roadmap also included low-level systems features such as allocation-free request processing, kernel-assisted networking, eBPF integration, and advanced performance optimizations.

As development progressed, several limitations of the Node.js implementation became apparent.

The most significant issues were:

- Increasing complexity around stream lifecycle management.
- Limited control over low-level networking primitives.
- Higher runtime overhead.
- Multi-process concurrency through `node:cluster`, introducing additional operational complexity.
- Reduced suitability for future systems-level experimentation.

Although these limitations were not blockers, they increasingly influenced architectural decisions and implementation complexity.

A decision was therefore required regarding the long-term implementation language for Torus.

---

# Decision

Rewrite Torus entirely in Go.

The Node.js implementation would be retained in the repository as a historical reference and performance baseline, but all future feature development would occur exclusively in the Go implementation.

---

# Alternatives Considered

## Option 1 — Continue Development in Node.js

### Advantages

- Existing codebase and tooling.
- Faster iteration for application-level features.
- Mature ecosystem.

### Disadvantages

- Limited visibility into low-level networking.
- Higher runtime overhead.
- Increasing implementation complexity for future roadmap items.
- Less suitable for systems-oriented experimentation.

---

## Option 2 — Rewrite in Rust

### Advantages

- Excellent runtime performance.
- Explicit memory management.
- Strong systems programming capabilities.
- Zero-cost abstractions.

### Disadvantages

- Significantly steeper learning curve.
- Longer development time.
- Higher implementation complexity for a solo project.
- Reduced iteration speed.

---

## Option 3 — Rewrite in Go

### Advantages

- Mature networking standard library.
- Lightweight concurrency through goroutines.
- Excellent profiling and benchmarking ecosystem.
- Widely adopted for infrastructure software.
- Simpler development model than Rust while still enabling systems programming.

### Disadvantages

- Garbage-collected runtime.
- Less control over memory than Rust.
- Certain low-level optimizations require working around runtime abstractions.

---

# Rationale

Go provided the best balance between engineering productivity and systems-level capability.

Its standard library includes mature networking primitives suitable for building production-grade infrastructure software while remaining significantly simpler than lower-level alternatives.

The language also provides:

- efficient concurrency primitives
- integrated CPU and memory profiling
- strong tooling
- cross-platform support
- a large ecosystem around cloud-native infrastructure

These characteristics aligned closely with the long-term goals of Torus.

While Rust may ultimately enable higher peak performance, its additional implementation complexity was not justified for the objectives of this project.

---

# Consequences

## Positive

- Simplified networking implementation.
- Lower runtime overhead.
- Significantly improved throughput.
- Reduced memory consumption.
- Simpler concurrency model.
- Better foundation for future systems-level research and optimization.

---

## Negative

- Complete rewrite required.
- Existing Node.js implementation entered maintenance mode.
- Temporary pause in feature development during migration.

---

## Trade-offs

The project traded short-term development effort for a stronger long-term architectural foundation.

The rewrite increased implementation effort but significantly expanded the range of future optimizations and research topics that Torus can realistically explore.

---

# Validation

The engineering impact of this decision is documented in:

[**Benchmark-001 — Node.js to Go Performance Evaluation**](../../benchmarking/reports/Benchmark-001-nodejs-to-go-performance-evaluation.md)

Key observations include:

- approximately **10.8×** improvement in end-to-end throughput
- approximately **10×** reduction in runtime memory consumption
- lower request latency
- simplified concurrency model

These results support the architectural decision to adopt Go as the long-term implementation language.

---

# Related Documents

## Benchmark Reports

- [Benchmark-001 — Node.js to Go Performance Evaluation](../../benchmarking/reports/Benchmark-001-nodejs-to-go-performance-evaluation.md)

## Architecture

- [ARCHITECTURE.md](../../ARCHITECTURE.md)

## Related Blog Posts

- [Why I Ripped `stream.pipe()` Out of My Node.js API Gateway](https://ashishbarmaiya.hashnode.dev/why-i-ripped-stream-pipe-out-of-my-node-js-api-gateway)
- [Surviving Node.js Clusters: Graceful Teardowns, Windows Quirks, and Black-Box Testing](https://ashishbarmaiya.hashnode.dev/surviving-node-js-clusters-graceful-teardowns-windows-quirks-and-black-box-testing)
- Why I Moved Torus from Node.js to Go *(planned)*

---

# Notes

This decision establishes Go as the implementation language for Torus.

Future architectural decisions assume Go as the project's long-term platform.
