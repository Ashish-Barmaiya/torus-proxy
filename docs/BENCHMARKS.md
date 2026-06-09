# Torus Proxy — Benchmarks

This document tracks the performance of Torus Proxy across its evolution from Node.js to Go. All tests were executed on identical, constrained hardware to provide an apples-to-apples comparison.

## Test Environment

- **CPU:** Intel(R) Core(TM) i3-1115G4 @ 3.00GHz (2 Cores, 4 Threads)
- **RAM:** 8.0 GB
- **OS:** Windows (WSL2)

---

## Latest: Go Implementation — wrk Benchmarks (June 2026)

### Methodology Improvements

The following changes were made to eliminate measurement noise and maximize benchmark fidelity:

1. **Replaced Node.js mock backends with a native Go backend** (`mock_backend.go`) — removes the V8 runtime and event-loop overhead from the critical path. The mock backend is a minimal `net/http` server that returns `"OK"` on every request, ensuring the upstream is never the bottleneck.
2. **Replaced Autocannon with [wrk](https://github.com/wg/wrk)** — `wrk` is a C-based HTTP benchmarking tool with a multi-threaded, epoll-driven architecture. It generates significantly higher request throughput with lower client-side overhead compared to Node.js-based Autocannon.
3. **Process isolation via dedicated WSL tabs** — the mock backend, Torus proxy, and `wrk` load generator each ran in separate WSL terminal sessions, preventing OS-level scheduling contention between producer and consumer processes.

### Test A: Full Production Proxying (`/api`)

- **Objective:** Benchmark Torus under real production conditions — full reverse-proxy lifecycle including request parsing, longest-prefix routing, round-robin load balancing, downstream TCP connection, backend response, and zero-copy stream piping back to the client.
- **Behavior:** Torus forwards every request to live Go mock backends on `:3001` and `:3002`, which respond with `200 OK`.

| Metric                     | Value                               |
| :------------------------- | :---------------------------------- |
| **Total Throughput**       | **17,865.81 req/sec**               |
| **Average Latency**        | 6.12 ms                             |
| **Maximum Tail Latency**   | 45.32 ms                            |
| **Data Transfer Rate**     | 2.01 MB/sec                         |
| **Total Requests Handled** | 178,892 requests (in 10.01s)        |

- **Analysis:** With native Go backends replacing Node.js servers, the full proxy lifecycle now achieves **17,865 req/sec** — a **~2.95× improvement** over the previous autocannon-measured 6,044 req/sec baseline. This confirms that the prior bottleneck was not Torus itself, but Node.js backend latency and Autocannon's single-threaded JavaScript load generator.

### Test B: Short-Circuit Memory Path (`/api2`)

- **Objective:** Isolate the raw routing and HTTP processing speed of Torus by hitting a path with no live backends. Torus catches the downstream connection failure instantly and returns an error response directly from memory — no network I/O roundtrip.
- **Behavior:** Torus returns a non-2xx failure boundary from memory without opening any downstream TCP connection.

| Metric                     | Value                               |
| :------------------------- | :---------------------------------- |
| **Total Throughput**       | **98,565.91 req/sec**               |
| **Average Latency**        | 10.21 ms                            |
| **Maximum Tail Latency**   | 86.63 ms                            |
| **Data Transfer Rate**     | 15.70 MB/sec                        |
| **Total Requests Handled** | 994,967 requests (in 10.09s)        |

- **Analysis:** With network I/O eliminated entirely, Torus's pure routing engine peaks at **~98.5k req/sec** — a **~6.3× improvement** over the previous autocannon-measured 15,644 req/sec baseline. This demonstrates that the Go HTTP router and memory-path error handling are extremely efficient, and that the prior Autocannon client was the primary throughput bottleneck in no-backend scenarios.

---

## Go vs Node.js — Historical Comparison

| Metric                | Node.js (Autocannon)                | Go + wrk (Latest)           | Improvement        |
| --------------------- | ----------------------------------- | --------------------------- | ------------------ |
| **Avg Throughput**    | 1,647 req/s                         | **17,865 req/s**            | **~10.8×**         |
| **Avg Latency**       | —                                   | 6.12 ms                     | —                  |
| **Max Tail Latency**  | —                                   | 45.32 ms                    | —                  |
| **Concurrency Model** | 4 OS processes (`node:cluster`)     | 1 process (goroutines)      | 10× less memory    |
| **Memory Footprint**  | ~200 MB                             | ~20 MB                      | **~10×**           |

> **Note:** The Node.js throughput (1,647 req/s) was measured with Autocannon against live Node.js backends. The Go throughput (17,865 req/s) was measured with `wrk` against live Go mock backends. Both represent full end-to-end production proxying on identical hardware.

---

## Archived: Go Implementation — Autocannon Benchmarks

The following benchmarks were captured using [Autocannon](https://github.com/mcollina/autocannon) (100 concurrent connections, 10-second duration) with Node.js mock backends. They are preserved for historical reference.

**Command:** `autocannon -c 100 -d 10 http://localhost:8080/api`

### Test Configuration A: Short-Circuit Routing Overhead (No Live Backends)

- **Objective:** Isolate the execution speed of Torus's core routing engine, HTTP parsing layer, and memory allocation hot path by intentionally withholding downstream backends.
- **Behavior:** Torus catches downstream network connection errors instantly and returns non-2xx failure boundaries directly from memory.

| Stat        | 2.5% | 50%  | 97.5% | 99%   | Avg     | Stdev   | Max   |
| :---------- | :--- | :--- | :---- | :---- | :------ | :------ | :---- |
| **Latency** | 4 ms | 5 ms | 10 ms | 14 ms | 5.94 ms | 3.06 ms | 78 ms |

| Stat          | 1%      | 2.5%    | 50%     | 97.5%  | Avg      | Stdev    | Min     |
| :------------ | :------ | :------ | :------ | :----- | :------- | :------- | :------ |
| **Req/Sec**   | 11,799  | 11,799  | 15,343  | 18,207 | 15,644.8 | 1,938.27 | 11,798  |
| **Bytes/Sec** | 2.21 MB | 2.21 MB | 2.87 MB | 3.4 MB | 2.93 MB  | 362 kB   | 2.21 MB |

- **Metrics Snapshot:** ~157k total requests in 10.08s | 29.3 MB data read.
- **Analysis:** With network I/O roundtrips eliminated, this run captures the absolute processing limit of the Go runtime engine on a dual-core constraint, peaking at **15,644 req/sec**.

### Test Configuration B: End-to-End Production Routing (Live Node.js Backends)

- **Objective:** Benchmark the true performance of Torus under real production conditions where requests execute the full proxy lifecycle.
- **Behavior:** Torus parses the request, load-balances across active servers, opens a downstream TCP connection, waits for the backend response, and pipes successful `200 OK` bytes back to the client.

| Stat        | 2.5% | 50%   | 97.5% | 99%   | Avg      | Stdev   | Max    |
| :---------- | :--- | :---- | :---- | :---- | :------- | :------ | :----- |
| **Latency** | 1 ms | 13 ms | 47 ms | 63 ms | 16.06 ms | 15.5 ms | 294 ms |

| Stat          | 1%     | 2.5%   | 50%    | 97.5%  | Avg     | Stdev    | Min    |
| :------------ | :----- | :----- | :----- | :----- | :------ | :------- | :----- |
| **Req/Sec**   | 3,245  | 3,245  | 6,383  | 7,491  | 6,044.6 | 1,363.41 | 3,245  |
| **Bytes/Sec** | 344 kB | 344 kB | 677 kB | 795 kB | 641 kB  | 145 kB   | 344 kB |

- **Metrics Snapshot:** ~61k total requests in 10.09s | 6.41 MB data read (100% `2xx OK` success rate).
- **Analysis:** When bound by network I/O and downstream application latency, Torus normalizes at **6,044 req/sec**. This represents a true **+267% performance increase** over the optimized Node.js cluster baseline (1,647 req/sec) on identical physical hardware.

---

## Archived: Node.js Implementation (`node/` directory)

The following benchmarks were captured with the original Node.js/TypeScript implementation, which used `node:cluster` (4 worker processes), `stream.pipe()` for proxying, and the V8 JavaScript runtime.

### Scenario A: The Baseline (Raw I/O & Routing)

**Objective:** Validate the core `node:http` stream piping architecture. Measure the maximum throughput of the Master/Worker cluster routing plain-text HTTP without security overhead.

**Command:** `autocannon -c 100 -d 10 http://localhost:8080/api`

- **Throughput:** 1,647 requests/sec
- **Errors:** 0
- **Memory Footprint:** Flat (Zero memory leaks or buffering bloat)

**Conclusion:** The native `.pipe()` architecture successfully bypasses the V8 JavaScript heap. The bottleneck is purely CPU event-loop saturation.

---

### Scenario B: The Production Edge (TLS + Redis)

**Objective:** Measure the true performance of the Node.js version operating as a fully-loaded API Gateway.

**Active Constraints:**

1. **TLS Termination:** Every connection requires strict cryptographic handshaking and payload decryption.
2. **Distributed Rate Limiting:** Every request executes an atomic Lua script over TCP to a local Redis cluster.
3. **Hardware Contention:** The dual-core i3 is simultaneously running the Torus Master, 4 Torus Workers, the Redis Container, 3 Backend Servers, and the Autocannon generator.

**Command:** `export NODE_TLS_REJECT_UNAUTHORIZED=0 && autocannon -c 100 -d 10 https://localhost:8080/api`

- **Throughput:** 968 requests/sec
- **Latency:** Avg: 103 ms | P99: 488 ms
- **Data Transferred:** 2.09 MB
- **Errors:** 0 (100% HTTP 200 Success Rate)

**Conclusion:** Security and distributed state have a mathematical cost. Adding TLS decryption and Redis atomicity resulted in a ~41% reduction in raw throughput.
