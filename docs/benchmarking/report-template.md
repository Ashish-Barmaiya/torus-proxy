# Benchmark Report Template

> This document defines the standard format for every benchmark report published by Torus.
>
> All benchmark reports must follow this structure unless deviations are explicitly documented.

---

# Benchmark Information

| Field | Value |
|--------|-------|
| Benchmark ID | BXXX |
| Title | |
| Date | |
| Author | |
| Torus Version | |
| Git Commit | |
| Methodology Version | |
| Hardware Profile | |
| Environment Profile | |
| Software Baseline | |
| Benchmark Category | |
| Benchmark Tool(s) | |

---

# 1. Objective

Clearly state what this benchmark measures.

Example:

> Measure the throughput and latency overhead introduced by TLS termination compared with plain HTTP.

---

# 2. Background

Describe why this benchmark matters.

Include:

- Engineering motivation
- Related benchmark reports
- Relevant implementation details
- Expected practical impact

---

# 3. Hypothesis

State the engineering hypothesis **before** running the benchmark.

Example:

> TLS termination will reduce maximum throughput due to cryptographic operations while having only a small impact on persistent HTTPS latency.

---

# 4. Experimental Variables

## Independent Variable

Exactly one intentionally changed variable.

Examples:

- HTTP → HTTPS
- Old parser → New parser
- Go allocator → Custom allocator

---

## Dependent Variables

Metrics being measured.

Examples:

- Throughput
- Latency
- CPU utilization
- Memory usage
- GC behaviour
- Allocations
- Context switches

---

## Controlled Variables

Everything intentionally held constant.

Examples:

- Hardware
- OS
- Kernel
- Go version
- Torus version
- Backend
- Payload
- Benchmark duration
- Concurrency
- Tool versions

---

# 5. Experimental Setup

Describe the benchmark topology.

Example:

Client

↓

Torus

↓

Backend

Include architecture diagrams where useful.

---

# 6. System Configuration

Reference:

- Hardware Profile
- Environment Profile
- Software Baseline

Document any deviations from the referenced profiles.

---

# 7. Benchmark Configuration

Document everything specific to this experiment.

Include:

- Torus configuration
- Backend configuration
- TLS configuration
- Benchmark tool configuration
- Runtime flags
- Environment variables
- Kernel tuning (if modified)

---

# 8. Benchmark Procedure

Describe the exact execution procedure.

Example:

1. Start backend
2. Start Torus
3. Wait for initialization
4. Warm-up traffic
5. Execute benchmark
6. Repeat N runs
7. Collect raw data
8. Calculate statistics

Include all benchmark commands.

---

# 9. Raw Results

Include unmodified outputs.

Examples:

- wrk
- vegeta
- perf
- pidstat
- pprof
- vmstat
- ss

Raw outputs must never be edited.

---

# 10. Statistical Summary

Summarize the collected runs.

Report:

- Sample size (N)
- Mean
- Median
- Minimum
- Maximum
- Standard deviation
- Coefficient of variation
- 95% confidence interval (when applicable)

---

# 11. Primary Performance Metrics

## Throughput

Report:

- Requests/sec
- Transfer/sec
- Throughput graphs
- Throughput comparison

---

## Latency

Report:

- Mean
- Median
- p90
- p95
- p99
- p99.9
- Maximum

Include:

- Histogram
- Box plot
- Percentile curve

---

# 12. Supporting Performance Metrics

Report:

- CPU utilization
- Memory usage
- RSS
- Virtual memory
- Allocations
- GC statistics
- Goroutines
- File descriptors
- Context switches
- Network throughput
- System load

These metrics explain *why* primary performance changed.

---

# 13. Scalability Analysis (If Applicable)

Evaluate behaviour while varying:

- Concurrency
- Connections
- Request rate
- Payload size
- Number of routes
- Number of upstreams

Include scaling graphs where appropriate.

---

# 14. Reliability Analysis (If Applicable)

Measure:

- Successful requests
- Failed requests
- Error rate
- Connection drops
- Recovery time
- Health-check behaviour
- Graceful shutdown
- Hot reload behaviour
- Failover latency

---

# 15. Regression Analysis

Compare against:

- Previous benchmark report
- Previous Torus version
- Performance baseline
- Other proxies (when applicable)

Clearly indicate:

- Improvement
- Regression
- No statistically meaningful change

---

# 16. Analysis

Interpret the results.

Discuss:

- Why the observed behaviour occurred
- Whether the hypothesis was supported
- Engineering trade-offs
- Surprising observations
- Bottlenecks revealed

Avoid unsupported speculation.

---

# 17. Threats to Validity

Discuss experimental limitations.

Examples:

- Localhost testing
- Single benchmark client
- Limited hardware
- Thermal throttling
- Virtualization
- Small payloads
- Background processes
- Short benchmark duration

---

# 18. Conclusion

Summarize the benchmark.

State:

- Whether the hypothesis was supported
- Key quantitative findings
- Engineering implications

---

# 19. Future Work (optional)

List follow-up investigations.

Examples:

- Additional workloads
- Larger hardware
- Multi-node testing
- Alternative implementations
- Further optimizations

---

# 20. References (if applicable)

List all external references.

Examples:

- RFCs
- Books
- Research papers
- Articles
- Documentation
- GitHub issues

---

# 21. Reproducibility

Provide everything required to reproduce the benchmark.

Include:

- Benchmark scripts
- Configuration files
- Dataset locations
- Raw output directory
- Plots
- Git commit
- Exact commands
- Required tool versions

---

# Appendix A — Benchmark Commands

Include every executed command.

Examples:

- wrk
- vegeta
- perf
- pidstat
- pprof
- vmstat
- ss

---

# Appendix B — Benchmark Data

Reference all benchmark artifacts.

Examples:

```
datasets/BXXX/
raw/
plots/
profiles/
summary.csv
results.json
```

---

# Appendix C — Visual Artifacts

Include:

- Architecture diagrams
- Benchmark topology
- Flamegraphs
- CPU profiles
- Heap profiles
- Latency histograms
- Box plots
- Time-series graphs
- Screenshots (if applicable)
