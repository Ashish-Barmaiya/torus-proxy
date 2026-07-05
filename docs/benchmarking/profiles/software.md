# Software Baselines

**Version:** 1.0
**Status:** Active

---

# 1. Purpose

Software Baselines define the software environment used during benchmarking.

Each benchmark report references a Software Baseline to ensure experiments remain reproducible and comparable over time.

Software Baselines are immutable. Any benchmark-relevant software change requires a new baseline.

---

# 2. Naming Convention

S001

S002

S003

...

---

# 3. Active Baselines

---

# S001

## Name

Local Ubuntu Development Workstation

---

## Status

Active

---

## Purpose

Primary software environment for local feature development, profiling, and benchmarking.

---

## Operating System

Distribution

Ubuntu 26.04 LTS

Architecture

x86_64

Kernel

(Output of `uname -r`)

---

## Go Toolchain

Go Version

Go 1.26.x

Compiler

Official Go Toolchain

CGO

Enabled (default)

GOMAXPROCS

Default

GOGC

Default

---

## Benchmark Tools

| Tool | Purpose |
|------|---------|
| wrk | Peak throughput benchmarking |
| wrk2 | Constant-rate latency benchmarking |
| Go testing (`go test -bench`) | Microbenchmarks |
| perf | CPU profiling |
| pprof | Go CPU, heap and allocation profiling |
| pidstat | CPU, memory and context-switch monitoring |
| vmstat | System resource monitoring |
| ss | TCP socket inspection |
| lsof | File descriptor inspection |

---

## System Analysis Tools

| Tool | Purpose |
|------|---------|
| strace | System call tracing |
| bpftool | eBPF inspection |
| bpftrace | Dynamic kernel tracing |
| tcpdump | Packet capture |
| FlameGraph | CPU flamegraph visualization |

---

## Runtime Configuration

CPU Governor

Performance (when benchmarking)

Transparent Huge Pages

Default

Open File Limit

(Default unless benchmark specifies otherwise)

---

## TLS

OpenSSL

System OpenSSL

Certificate

2048-bit self-signed RSA (development)

---

## Notes

This baseline is used for all local benchmark reports unless explicitly overridden.

---

# 4. Baseline Evolution

Create a new Software Baseline whenever any of the following changes:

- Operating System
- Linux Kernel
- Go version
- Benchmark tool version
- Runtime tuning
- TLS implementation
- Significant system configuration

Existing baselines must never be modified.

---

# 5. Benchmark References

Every benchmark report must reference:

- Hardware Profile
- Environment Profile
- Software Baseline
- Methodology Version

Example

Hardware: H001

Environment: E001

Software: S001

Methodology: v1.0
