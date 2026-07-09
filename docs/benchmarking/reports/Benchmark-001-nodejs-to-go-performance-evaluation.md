# Benchmark-001: Node.js to Go Performance Evaluation

> **Status:** Complete

This benchmark documents the performance impact of rewriting Torus from Node.js to Go. It serves as the historical baseline for all future benchmark reports and establishes the initial performance characteristics of the Go implementation.

---

# Benchmark Information

| Field | Value |
|--------|-------|
| Benchmark ID | B-001 |
| Title | Node.js to Go Performance Evaluation |
| Status | Complete |
| Date | June 2026 |
| Author | Ashish Barmaiya |
| Torus Version | Pre-v1.0 |
| Methodology Version | v1.0 (retrospectively structured) |
| Hardware Profile | [H001](../profiles/hardware.md) |
| Environment Profile | [E001](../profiles/environment.md) |
| Software Baseline | [S001](../profiles/software.md) |
| Benchmark Category | Throughput, Latency, Resource Usage |
| Related ADR | ADR-001 – Rewrite Torus from Node.js to Go |

---

# 1. Objective

Evaluate whether rewriting Torus from Node.js to Go resulted in measurable improvements in performance, resource efficiency, and engineering scalability while preserving equivalent reverse proxy functionality.

This benchmark establishes the historical performance baseline for the Go implementation and provides the reference point for all future performance investigations.

---

# 2. Background

Torus was originally developed in Node.js and TypeScript as an educational reverse proxy and API gateway.

While the Node.js implementation successfully demonstrated production concepts such as reverse proxying, load balancing, clustering, graceful shutdown and rate limiting, several architectural limitations became apparent during development.

The most significant issues were:

- increasing complexity around stream lifecycle management
- limited visibility into low-level networking behaviour
- higher runtime overhead
- growing implementation complexity for future systems-oriented features

The long-term roadmap for Torus includes advanced topics such as custom networking paths, zero-copy optimizations, kernel integration and allocation-free request processing. Achieving these goals required significantly more control over memory management and networking primitives than the Node.js runtime provides.

For these reasons, Torus was rewritten in Go.

---

# 3. Hypothesis

The rewrite to Go is expected to:

- significantly improve end-to-end throughput
- reduce runtime overhead
- lower memory consumption
- simplify the concurrency model
- provide a stronger foundation for future systems-level optimizations

---

# 4. Experimental Variables

## Independent Variable

Implementation runtime.

- Node.js implementation
- Go implementation

---

## Dependent Variables

The following characteristics are evaluated throughout this benchmark:

- Throughput
- Latency
- Memory footprint
- Concurrency model

---

## Controlled Variables

Where practical, the following remained constant:

- identical physical hardware
- local benchmark topology
- equivalent proxy functionality
- identical routing configuration
- similar benchmark duration
- identical concurrency levels

Some methodology improvements were intentionally introduced during the Go implementation. These are discussed separately because they improve measurement fidelity rather than changing Torus itself.

---

# 5. Historical Evolution

This benchmark represents the culmination of three successive performance investigations.

## Stage 1 — Original Node.js Implementation

The original implementation established the baseline behaviour of Torus using the Node.js runtime, `node:cluster`, and `stream.pipe()`.

---

## Stage 2 — Initial Go Implementation

After the rewrite, Torus was benchmarked using Autocannon while retaining the original Node.js mock backends.

This provided an initial comparison against the Node.js implementation but still contained external bottlenecks introduced by the benchmark client and upstream services.

---

## Stage 3 — Refined Go Benchmarking

The benchmark methodology was refined to better isolate Torus itself.

Three major improvements were introduced:

1. Node.js mock backends were replaced with minimal Go mock backends.

2. Autocannon was replaced with `wrk`, reducing benchmark client overhead.

3. Benchmark client, proxy and backend were executed in separate terminal sessions to reduce operating-system scheduling contention.

These refinements significantly reduced measurement noise and form the basis of the final benchmark results presented in this report.

---

# 6. Test Environment

This benchmark references the following standardized profiles:

| Profile | Identifier |
|----------|------------|
| Hardware | [H001](../profiles/hardware.md) |
| Environment | [E001](../profiles/environment.md) |
| Software Baseline | [S001](../profiles/software.md) |

Detailed specifications are documented separately.

---

# 7. Benchmark Procedure

Each benchmark followed the same high-level process.

1. Start all backend services.
2. Start Torus.
3. Verify successful route registration.
4. Allow the system to stabilize.
5. Execute benchmark workload.
6. Record raw benchmark output.
7. Repeat where required.
8. Compare results with historical baselines.

Methodology evolved throughout the project to improve measurement accuracy, but every published result represents the best available methodology at the time it was recorded.

---

# 8. Historical Benchmarks

This section presents the historical performance evolution of Torus.

Rather than comparing only the initial Node.js implementation against the latest Go implementation, intermediate benchmark stages are also preserved. These intermediate results illustrate how both the implementation and the benchmarking methodology evolved throughout the project.

---

# 8.1 Original Node.js Implementation

The original implementation was written in Node.js and TypeScript using `node:cluster`, `stream.pipe()` for proxying, and the V8 JavaScript runtime.

It served as the functional prototype from which the Go implementation was later developed.

---

## Scenario A — Baseline Reverse Proxy

### Objective

Measure the maximum throughput of the original Node.js implementation while acting as a reverse proxy under normal operating conditions.

The benchmark focused on validating the architecture rather than maximizing performance.

---

### Configuration

- Runtime: Node.js
- Concurrency Model: `node:cluster` (4 worker processes)
- Benchmark Tool: Autocannon
- Duration: 10 seconds
- Concurrency: 100 connections

---

### Results

| Metric | Value |
|--------|------:|
| Throughput | **1,647 req/sec** |
| Errors | 0 |
| Memory Footprint | ~200 MB |

---

### Analysis

The Node.js implementation successfully demonstrated that the overall proxy architecture was correct.

The benchmark also showed that the proxy remained stable under sustained load with no observable memory leaks.

However, throughput was ultimately constrained by the JavaScript runtime and event-loop scheduling. This benchmark became the primary performance baseline against which the Go rewrite would later be evaluated.

---

## Scenario B — Production Edge

### Objective

Evaluate the performance impact of enabling production-oriented features.

Unlike the baseline benchmark, this configuration enabled additional responsibilities commonly performed by an API gateway.

---

### Active Features

- TLS termination
- Redis-backed distributed rate limiting
- Four worker processes
- Multiple backend services
- Local Redis container

These services executed simultaneously on the same dual-core development machine.

---

### Results

| Metric | Value |
|--------|------:|
| Throughput | **968 req/sec** |
| Average Latency | **103 ms** |
| P99 Latency | **488 ms** |
| Errors | 0 |

---

### Analysis

Adding TLS and distributed rate limiting reduced throughput by approximately **41%** compared with the baseline proxy benchmark.

This result was expected. Every request now required cryptographic processing together with an additional Redis round trip before reaching the backend.

Although performance decreased, the implementation maintained a **100% success rate**, demonstrating functional correctness under significantly higher processing overhead.

---

# 8.2 Initial Go Implementation

The first Go implementation focused on validating the rewrite rather than maximizing absolute performance.

At this stage, the benchmarking methodology still reused the original Node.js mock backends together with Autocannon.

While these results already demonstrated significant improvements over the Node.js implementation, they were later superseded by a more rigorous methodology using native Go backends and `wrk`.

They are preserved here to document the project's performance evolution.

---

## Experiment A — Short-Circuit Routing

### Objective

Measure the maximum processing capability of Torus while minimizing downstream overhead.

No live backend servers were available.

Requests immediately failed at the downstream connection stage, allowing the benchmark to primarily measure routing, request parsing and in-memory response generation.

---

### Configuration

- Benchmark Tool: Autocannon
- Duration: 10 seconds
- Concurrency: 100 connections
- Backend: unavailable

---

### Results

#### Latency

| Metric | Value |
|--------|------:|
| Average | 5.94 ms |
| Maximum | 78 ms |

#### Throughput

| Metric | Value |
|--------|------:|
| Average | **15,644.8 req/sec** |
| Peak | 18,207 req/sec |

---

### Analysis

With network latency largely removed from the critical path, the benchmark highlighted the efficiency of the Go implementation itself.

Even at this early stage, Torus processed approximately **9.5×** more requests than the original Node.js baseline.

The benchmark also suggested that the benchmark client itself was beginning to approach its own practical limits.

---

## Experiment B — End-to-End Production Routing

### Objective

Measure real-world reverse proxy performance using live backend services.

Each request exercised the complete proxy pipeline:

- HTTP request parsing
- Route lookup
- Load balancing
- Backend connection
- Response forwarding

---

### Configuration

- Benchmark Tool: Autocannon
- Duration: 10 seconds
- Concurrency: 100 connections
- Live Node.js mock backends

---

### Results

#### Latency

| Metric | Value |
|--------|------:|
| Average | 16.06 ms |
| Maximum | 294 ms |

#### Throughput

| Metric | Value |
|--------|------:|
| Average | **6,044.6 req/sec** |
| Peak | 7,491 req/sec |

---

### Analysis

The initial Go implementation achieved approximately **3.7×** the throughput of the original Node.js implementation while performing the same end-to-end proxy workflow.

However, these measurements also revealed two significant external bottlenecks:

- the Node.js mock backends
- Autocannon itself

These observations motivated the methodology refinements introduced in the final benchmark stage.

---

# 9. Final Go Benchmarks

Following the observations made during the initial Go benchmarks, the benchmark methodology was refined to better isolate the performance of Torus itself.

Three major improvements were introduced.

1. **Native Go mock backends**

   The original Node.js mock services were replaced with minimal Go HTTP servers (`mock_backend.go`). This eliminated JavaScript runtime overhead from the upstream services.

2. **wrk replaced Autocannon**

   Autocannon was replaced with **wrk**, a high-performance C-based benchmarking tool capable of generating significantly higher request rates with lower client-side overhead.

3. **Process Isolation**

   The benchmark client, Torus proxy and backend services were executed in separate terminal sessions to reduce operating-system scheduling contention and improve measurement consistency.

These refinements form the official benchmark methodology for the remainder of the project.

---

# Experiment A — Full Production Proxying

## Objective

Measure the performance of Torus while executing the complete reverse proxy pipeline under realistic operating conditions.

Every request performs:

- HTTP request parsing
- Route lookup
- Longest-prefix matching
- Round-robin load balancing
- Backend TCP connection
- Response forwarding
- Header injection
- Zero-copy stream forwarding

This benchmark represents the primary production performance metric for Torus.

---

## Configuration

Benchmark Tool

- wrk

Duration

- 10 seconds

Connections

- 100

Threads

- 2

Backend

- Native Go mock servers

---

## Results

| Metric | Value |
|--------|------:|
| Throughput | **17,865.81 req/sec** |
| Average Latency | **6.12 ms** |
| Maximum Tail Latency | **45.32 ms** |
| Transfer Rate | **2.01 MB/sec** |
| Total Requests | **178,892** |

---

## Analysis

Compared with the previous Autocannon benchmark, throughput increased from **6,044 req/sec** to **17,865 req/sec**, representing an improvement of approximately **2.95×**.

This improvement is not solely attributable to Torus.

The refined benchmark methodology eliminated two major external bottlenecks:

- JavaScript mock backends
- Benchmark client limitations

The resulting measurement more accurately represents the performance characteristics of Torus itself.

---

# Experiment B — Short-Circuit Memory Path

## Objective

Measure the maximum processing capability of Torus when downstream network I/O is removed from the critical execution path.

Instead of forwarding requests to live upstream servers, Torus immediately returns an in-memory failure response after detecting an unavailable backend.

This isolates the cost of:

- HTTP parsing
- Routing
- Load balancing
- Request handling
- Response generation

---

## Configuration

Benchmark Tool

- wrk

Duration

- 10 seconds

Connections

- 100

Threads

- 2

Backend

- No live backend

---

## Results

| Metric | Value |
|--------|------:|
| Throughput | **98,565.91 req/sec** |
| Average Latency | **10.21 ms** |
| Maximum Tail Latency | **86.63 ms** |
| Transfer Rate | **15.70 MB/sec** |
| Total Requests | **994,967** |

---

## Analysis

Removing downstream network communication increased throughput to approximately **98.5k requests/sec**, representing a **6.3×** improvement over the previous Autocannon measurement.

This benchmark demonstrates that the routing engine itself introduces very little overhead.

The dominant performance cost of a production reverse proxy lies in downstream networking rather than request dispatch.

---

# 10. Comparative Summary

The following table summarizes the evolution of Torus throughout the rewrite.

| Metric | Node.js | Initial Go | Final Go |
|--------|--------:|-----------:|----------:|
| Production Throughput | 1,647 req/sec | 6,044 req/sec | **17,865 req/sec** |
| Routing Throughput | — | 15,644 req/sec | **98,565 req/sec** |
| Average Latency | 103 ms* | 16.06 ms | **6.12 ms** |
| Memory Footprint | ~200 MB | ~20 MB | ~20 MB |
| Concurrency Model | node:cluster | Goroutines | Goroutines |

\*Production configuration with TLS and Redis enabled.

---

## Overall Improvement

| Characteristic | Improvement |
|---------------|------------:|
| End-to-End Throughput | **10.8×** |
| Memory Consumption | **~10× reduction** |
| Runtime Model | Multi-process → Single-process goroutines |

---

# 11. Statistical Summary

Benchmark-001 predates the establishment of Torus' standardized benchmarking methodology.

The benchmark data presented in this report was collected over multiple stages of development using different benchmark tools, backend implementations, and experimental setups. Consequently, the results should be interpreted as historical engineering measurements rather than a controlled statistical experiment.

Formal statistical analysis—including repeated benchmark runs, confidence intervals, coefficient of variation, and regression testing—was intentionally not performed for this benchmark because comparisons across the historical stages would not be statistically meaningful.

Instead, this report focuses on documenting the engineering evolution of Torus and establishing the historical performance baseline for future benchmark reports.

Beginning with **Benchmark-002**, all benchmarks will follow the standardized methodology defined in the Torus Benchmarking Framework, including repeated executions, statistical analysis, and reproducibility requirements.

---
