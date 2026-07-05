# Torus Benchmark Tooling

**Version:** 1.0
**Status:** Active

---

# 1. Purpose

This document defines the official benchmarking and profiling tools used throughout the Torus project.

The purpose is to standardize data collection so that benchmark results remain reproducible and comparable across time.

Every benchmark report should reference the tools used.

---

# 2. Tool Categories

Benchmark tooling is divided into six categories.

| Category | Purpose |
|----------|---------|
| Load Generation | Generate benchmark traffic |
| Runtime Profiling | Analyze Go runtime |
| System Monitoring | Observe operating system resources |
| Kernel Profiling | Measure kernel and CPU behaviour |
| Network Inspection | Inspect socket and network state |
| Visualization | Analyze benchmark data |

---

# 3. Load Generation

## wrk

Purpose

Measure maximum throughput.

Use Cases

- Peak RPS
- Throughput scaling
- Reverse proxy throughput
- Comparative benchmarks

Metrics

- Requests/sec
- Transfer/sec
- Latency summary

---

## wrk2

Purpose

Generate traffic at a constant request rate.

Use Cases

- Latency studies
- Tail latency
- Saturation testing
- Performance regressions

Metrics

- p50
- p95
- p99
- p99.9
- Maximum latency

Preferred for latency investigations.

---

## Vegeta

Purpose

Flexible HTTP workload generation.

Use Cases

- Custom traffic profiles
- Burst workloads
- Replay workloads

Future use.

---

# 4. Go Runtime Profiling

## pprof

Purpose

Analyze Go runtime.

Profiles

- CPU
- Heap
- Allocations
- Goroutines
- Block
- Mutex

Required whenever a benchmark investigates an optimization.

---

## Go Benchmark Harness

Command

go test -bench

Purpose

Microbenchmarks.

Metrics

- ns/op
- allocs/op
- bytes/op

---

## Runtime Metrics

runtime/metrics

Purpose

Collect Go runtime statistics during execution.

Future integration.

---

# 5. System Monitoring

## pidstat

Measure

- CPU
- Memory
- Context Switches
- Threads

Primary resource monitoring tool.

---

## vmstat

Measure

- System Load
- Memory
- Swap
- CPU
- Context Switches

---

## htop

Interactive monitoring.

Used during benchmark validation.

Not considered benchmark evidence.

---

## free

Memory usage verification.

---

## uptime

System load verification.

---

# 6. Linux Kernel Profiling

## perf

Purpose

CPU profiling.

Measure

- CPU cycles
- Instructions
- Cache misses
- Branch misses
- Context switches

Required for advanced optimization studies.

---

## perf stat

Primary command for hardware counters.

---

## perf record

Generate CPU flamegraphs.

Future optimization work.

---

# 7. Network Inspection

## ss

Measure

- Active connections
- TCP state
- Socket usage

Preferred over netstat.

---

## ip

Inspect network interfaces.

---

## ethtool

Inspect NIC capabilities.

Future cloud benchmarking.

---

# 8. Process Inspection

## ps

Verify process state.

---

## lsof

Measure

Open files.

Open sockets.

Useful for FD investigations.

---

# 9. Visualization

Benchmark reports should include visualizations whenever practical.

Preferred tools

- Grafana
- Python (matplotlib)
- Vega-Lite
- Website charts

Recommended charts

- Line Chart
- Histogram
- Box Plot
- Time Series
- Bar Chart

---

# 10. Tool Selection Guide

Throughput

- wrk
- pidstat

Latency

- wrk2
- pidstat

Resource Usage

- pidstat
- vmstat
- pprof

Optimization

- pprof
- perf
- go test -bench

Socket Investigation

- ss
- lsof

Comparative Benchmark

- wrk
- wrk2
- pidstat
- perf

---

# 11. Required Tools by Benchmark Type

Feature Benchmark

Required

- wrk
- wrk2
- pidstat

Recommended

- pprof

---

Optimization Benchmark

Required

- wrk
- wrk2
- pidstat
- pprof
- perf

---

Comparative Benchmark

Required

- wrk
- wrk2
- pidstat
- perf

---

Long Running Stability

Required

- pidstat
- vmstat
- ss
- lsof

---

# 12. Future Tooling

Future investigations may include

- eBPF
- bpftool
- bpftrace
- flamegraph
- Intel VTune
- Valgrind (for C experiments)
- tcpdump
- Wireshark

These are outside the scope of Methodology v1.
