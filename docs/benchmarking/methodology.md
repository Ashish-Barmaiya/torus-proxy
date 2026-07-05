# Torus Benchmark Methodology

**Version:** 1.0
**Status:** Active
**Last Updated:** 2026-07-05

---

# 1. Purpose

This document defines the official benchmarking methodology used throughout the Torus project.

Its purpose is to ensure that benchmark results are:

- Reproducible
- Fair
- Statistically meaningful
- Comparable across time
- Independent of accidental environmental factors

All benchmark reports reference the methodology version used during testing.

---

# 2. Benchmark Goals

Every benchmark must answer a clearly defined engineering question.

Examples:

- What is the throughput cost of TLS termination?
- Does hot reload introduce latency spikes?
- How much CPU overhead does Prometheus instrumentation introduce?
- Does allocation-free header parsing reduce tail latency?

Benchmarks must never be performed simply to obtain larger numbers.

---

# 3. Benchmark Lifecycle

Every benchmark follows the same process.

1. Define objective
2. Define hypothesis
3. Select benchmark category
4. Select benchmark tools
5. Prepare environment
6. Warm up the system
7. Execute benchmark
8. Collect raw data
9. Perform statistical analysis
10. Draw conclusions
11. Document limitations

---

# 4. Controlled Variables

Only one independent variable should change between two comparable benchmark runs.

Everything else must remain constant.

Examples of controlled variables:

- Hardware
- Operating System
- Kernel Version
- Go Version
- Compiler Flags
- Benchmark Tool Version
- Backend Implementation
- Network Topology
- Benchmark Duration
- Concurrency
- Payload Size

---

# 5. Warm-Up Procedure

Benchmarking must never begin immediately after starting the proxy.

Before collecting measurements:

1. Start all services.
2. Wait for initialization.
3. Generate warm-up traffic.
4. Verify stable CPU and memory usage.
5. Begin measurements.

Default warm-up duration:

30 seconds

Exceptions must be documented.

---

# 6. Benchmark Duration

Default benchmark duration:

30 seconds

Longer durations are recommended for:

- long-running stability tests
- memory leak detection
- GC behaviour
- resource monitoring

Shorter durations require justification.

---

# 7. Number of Iterations

Every benchmark must be repeated multiple times.

Default:

10 runs

Recommended:

20 runs

The reported results must summarize all runs.

Never publish only the single best run.

---

# 8. Randomization

When comparing two implementations:

Avoid:

AAAAAAAAAA

BBBBBBBBBB

Instead:

ABABABABAB

or

BABABABABA

This minimizes bias introduced by:

- thermal throttling
- background OS activity
- CPU frequency scaling
- cache warming

---

# 9. Benchmark Topology

Every benchmark report must clearly define:

Benchmark Client

↓

Proxy

↓

Backend

Include:

- machine placement
- operating systems
- network connections
- local vs remote execution

Topology diagrams are encouraged.

---

# 10. Benchmark Categories

Every benchmark belongs to one or more categories:

- Throughput
- Latency
- Resource Usage
- Scalability
- Reliability
- Regression
- Comparative

Definitions are provided in Benchmark Matrix.

---

# 11. Resource Monitoring

During benchmarking, the following resources should be monitored whenever applicable.

CPU Utilization

Memory Usage (RSS)

Virtual Memory

Goroutine Count

File Descriptor Count

Context Switches

Network Throughput

GC Activity

System Load

The exact monitoring tools are defined in Tooling.

---

# 12. Profiling

Whenever a benchmark indicates a measurable regression or improvement, profiling should be performed.

Examples:

CPU Profile

Heap Profile

Allocation Profile

Block Profile

Mutex Profile

Profiles become part of the benchmark dataset.

---

# 13. Raw Data

Every benchmark stores its raw outputs.

Examples:

wrk output

wrk2 output

JSON summaries

CPU profiles

Memory profiles

System statistics

Raw data must not be edited.

---

# 14. Result Interpretation

Benchmark reports should distinguish between:

Observed Result

↓

Interpretation

↓

Hypothesis

↓

Conclusion

Interpretations must never overstate the evidence.

Correlation does not imply causation.

---

# 15. Threats to Validity

Every benchmark report must discuss possible limitations.

Examples:

- Localhost networking
- Small dataset
- Single benchmark client
- CPU thermal throttling
- Background processes
- Virtualized hardware
- Limited concurrency

Understanding limitations is part of the benchmark.

---

# 16. Benchmark Evolution

Methodology may evolve.

When methodology changes:

- publish a new methodology version
- keep previous versions available
- never modify historical reports

Historical benchmarks remain valid within the methodology used when they were executed.

---

# 17. Deviations

Any deviation from this methodology must be explicitly documented inside the benchmark report.

Examples:

- shorter benchmark duration
- fewer iterations
- different warm-up
- alternative benchmarking tool

Every deviation requires justification.

---

# 18. Reproducibility Checklist

Before publishing a benchmark report, verify that the following are included:

- Objective
- Hypothesis
- Hardware Profile
- Environment Profile
- Methodology Version
- Tool Versions
- Commands Used
- Configuration Files
- Raw Results
- Statistical Summary
- Analysis
- Limitations

Only after this checklist is complete should the benchmark report be considered complete.
