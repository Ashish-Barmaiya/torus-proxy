# Torus Benchmark Matrix

**Version:** 1.0
**Status:** Active

---

# 1. Purpose

This document defines every performance characteristic evaluated throughout the lifetime of Torus.

Rather than benchmarking individual features in isolation, Torus benchmarks engineering characteristics.

Every benchmark report references one or more benchmark categories defined here.

---

# 2. Benchmark Categories

| Category | Purpose |
|----------|---------|
| Throughput | Maximum request processing capability |
| Latency | Response time characteristics |
| Resource Usage | CPU, memory and operating system resources |
| Scalability | Behaviour under increasing workload |
| Reliability | Behaviour during failures |
| Regression | Detect performance changes across releases |
| Comparative | Compare Torus against existing proxies |
| Correctness Under Load | Verify correctness while the proxy is under stress |

---

# 3. Throughput

## Objective

Measure how much useful work Torus can perform.

---

## Metrics

- Requests/sec (RPS)
- Responses/sec
- Transfer/sec
- Requests per CPU %
- Requests per MB of Memory
- Throughput Scaling

---

## Test Matrix

### Protocol

- HTTP
- HTTPS
- HTTP Keep-Alive
- New TCP Connection Per Request
- HTTP/2
- HTTP/3 (Future)

---

### Payload Size

- Empty Response
- 1 KB
- 10 KB
- 100 KB
- 1 MB

---

### Request Types

- GET
- POST
- PUT
- DELETE

---

### Workload

- Static Response
- Reverse Proxy
- Backend Computation
- Streaming Response
- WebSocket
- Mixed Traffic

---

### Route Count

- 1
- 10
- 100
- 1000
- 10000 (Future)

---

### Backend Count

- 1
- 2
- 3
- 5
- 10

---

# 4. Latency

## Objective

Measure response time characteristics under different operating conditions.

---

## Metrics

Mandatory

- Mean
- Median
- p90
- p95
- p99
- Maximum

Recommended

- p99.9

---

## Test Matrix

Measure latency under:

- Idle
- Moderate Load
- Near Saturation
- Saturation

Measure latency during:

- TLS Handshake
- Backend Timeout
- Health Recovery
- Hot Reload
- Graceful Shutdown
- Rate Limiting
- Retry
- Circuit Breaker (Future)

Measure:

- End-to-End Request Latency
- Backend Latency
- Proxy Processing Overhead

---

# 5. Resource Usage

## Objective

Measure resource efficiency.

---

## CPU

- Average CPU
- Peak CPU
- User CPU
- System CPU

---

## Memory

- RSS
- Virtual Memory
- Heap
- Stack

---

## Go Runtime

- Heap Allocations
- Heap Objects
- Allocations/op
- Bytes/op
- GC Cycles
- GC Pause Time

---

## Operating System

- File Descriptors
- Context Switches
- Threads
- Goroutines
- Network Throughput

---

# 6. Scalability

## Objective

Measure how Torus scales with increasing workload.

---

## Connections

- 10
- 50
- 100
- 250
- 500
- 1000
- 5000
- 10000 (Future)

---

## CPU Scaling

Measure with:

- 1 Core
- 2 Cores
- 4 Cores
- 8 Cores (Future)

---

## GOMAXPROCS Scaling

Measure:

- 1
- 2
- 4
- 8

---

## Backend Scaling

Measure with:

- 1 Backend
- 2 Backends
- 3 Backends
- 5 Backends
- 10 Backends

---

## Route Scaling

Measure with:

- 1 Route
- 10 Routes
- 100 Routes
- 1000 Routes
- 10000 Routes (Future)

---

## Payload Scaling

Measure:

- Small
- Medium
- Large

---

# 7. Reliability

## Objective

Evaluate behaviour during failures.

---

## Backend Failure

Measure:

- Detection Time
- Recovery Time
- Failed Requests
- Successful Requests

---

## Graceful Shutdown

Measure:

- Dropped Requests
- Drain Duration
- Shutdown Duration

---

## Hot Reload

Measure:

- Reload Duration
- Latency Spike
- Dropped Connections

---

## Resource Exhaustion (Future)

Measure:

- Memory Pressure
- FD Pressure
- Goroutine Growth

---

## Long Running Stability

Run:

- 12 Hours
- 24 Hours

Measure:

- Memory Drift
- CPU Drift
- FD Drift
- GC Behaviour
- Goroutine Growth

---

# 8. Regression

Every release should compare against:

- Previous Release
- Previous Benchmark
- Last Stable Version

Regression Metrics:

- Throughput
- Latency
- CPU
- Memory
- GC
- Allocations

---

# 9. Comparative

Compare Torus against:

- NGINX
- HAProxy
- Envoy
- Caddy
- Traefik

Comparison Rules:

- Same Hardware
- Same Benchmark Client
- Same Backend
- Same Payload
- Same Concurrency
- Same Duration
- Same Topology
- Equivalent Configuration

---

# 10. Correctness Under Load

## Objective

Ensure Torus remains functionally correct while processing traffic under load.

Performance improvements are meaningless if correctness is compromised.

---

## Request Correctness

Verify:

- Every request reaches the intended backend.
- No request is routed incorrectly.

---

## Response Correctness

Verify:

- Status codes remain unchanged.
- Response headers remain correct.
- Response body remains unchanged.

---

## Header Integrity

Verify:

- Proxy headers are correctly injected.
- Existing headers are preserved.
- No duplicate headers appear.
- Header ordering (where relevant) remains valid.

---

## Request Body Integrity

Verify:

- Request body hash matches.
- No truncation.
- No corruption.

---

## Response Body Integrity

Verify:

- Response body hash matches.
- Streaming responses complete successfully.
- No corruption.

---

## Ordering

Verify:

- No duplicated requests.
- No missing requests.
- No unexpected retries.

---

## Connection Behaviour

Verify:

- Keep-Alive functions correctly.
- Connections close gracefully.
- No unexpected resets.
- No leaked connections.

---

## Protocol Correctness

Verify:

- HTTP semantics remain valid.
- WebSocket upgrades succeed.
- Streaming responses remain uninterrupted.

---

# 11. Feature Benchmark Checklist

Every new feature should evaluate whether it affects:

- □ Throughput
- □ Latency
- □ CPU
- □ Memory
- □ GC
- □ Scalability
- □ Reliability
- □ Correctness Under Load

If applicable, a benchmark report should be created.

---

# 12. Optimization Benchmark Checklist

Every optimization must compare:

Before

↓

After

Measure:

- □ Throughput
- □ Latency
- □ CPU
- □ Memory
- □ GC
- □ Allocations
- □ System Calls (if applicable)
- □ Context Switches (if applicable)
- □ Tail Latency
- □ Correctness Under Load

---

# 13. Benchmark Priority

## Priority 1 (Mandatory)

- Throughput
- Latency
- CPU
- Memory
- Correctness Under Load

---

## Priority 2 (Recommended)

- Scalability
- Reliability
- Regression

---

## Priority 3 (Future)

- Comparative Benchmarks
- Energy Consumption
- NUMA
- Cache Misses
- Branch Prediction
- Instructions Per Cycle (IPC)
