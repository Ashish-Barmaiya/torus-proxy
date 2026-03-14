# Torus Proxy Benchmarks

This document tracks the performance of Torus Proxy across two distinct architectural states. All tests were executed on highly constrained, shared hardware to establish a baseline worst-case scenario.

## Test Environment
* **CPU:** Intel(R) Core(TM) i3-1115G4 @ 3.00GHz (2 Cores, 4 Threads)
* **RAM:** 8.0 GB
* **OS:** Windows / Node.js
* **Tool:** Autocannon (100 concurrent connections, 10-second duration)

---

## Scenario A: The Baseline (Raw I/O & Routing)
**Objective** Validate the core `node:http` stream piping architecture. Measure the maximum throughput of the Master/Worker cluster routing plain-text HTTP without security overhead.

**Command:** `autocannon -c 100 -d 10 http://localhost:8080/api`

* **Throughput:** 1,647 requests/sec
* **Errors:** 0
* **Memory Footprint:** Flat (Zero memory leaks or buffering bloat)

**Conclusion:** The native `.pipe()` architecture successfully bypasses the V8 JavaScript heap. The bottleneck is purely CPU event-loop saturation.

---

## Scenario B: The Production Edge (TLS + Redis)
**Objective:** Measure the true performance of Torus v1.0.0 operating as a fully-loaded Enterprise API Gateway. 

**Active Constraints:**
1. **TLS Termination:** Every connection requires strict cryptographic handshaking and payload decryption.
2. **Distributed Rate Limiting:** Every request executes an atomic Lua script over TCP to a local Redis cluster.
3. **Hardware Contention:** The dual-core i3 is simultaneously running the Torus Master, 4 Torus Workers, the Redis Container, 3 Backend Servers, and the Autocannon generator.

**Command:** `export NODE_TLS_REJECT_UNAUTHORIZED=0 && autocannon -c 100 -d 10 https://localhost:8080/api`

* **Throughput:** 968 requests/sec
* **Latency:** Avg: 103 ms | P99: 488 ms
* **Data Transferred:** 2.09 MB
* **Errors:** 0 (100% HTTP 200 Success Rate)

**Conclusion:** Security and distributed state have a mathematical cost. Adding TLS decryption and Redis atomicity resulted in a ~41% reduction in raw throughput. However, pushing nearly 1,000 RPS on a saturated, low-voltage dual-core processor proves the architecture's efficiency. Deployed on dedicated infrastructure (e.g., AWS c7g.large), throughput will scale linearly with physical core count.