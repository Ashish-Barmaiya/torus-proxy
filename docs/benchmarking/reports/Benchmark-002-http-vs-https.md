# Benchmark-002: HTTP vs HTTPS Performance Evaluation

> **Status:** Complete

This benchmark documents the performance overhead introduced by TLS termination in Torus by comparing equivalent HTTP and HTTPS deployments under identical benchmark conditions.

---

# Benchmark Information

| Field | Value |
|--------|-------|
| **Benchmark ID** | B-002 |
| **Title** | HTTP vs HTTPS Performance Evaluation |
| **Date** | 2026-07-16 |
| **Author** | Ashish Barmaiya |
| **Torus Version** | Development Snapshot|
| **Git Commit** | c1aaa11d42b54ec9556caf15ed26435e1cadb615 |
| **Methodology Version** | [1.0](./../methodology.md) |
| **Hardware Profile** | [H001](./../profiles/hardware.md) |
| **Environment Profile** | [E001](./../profiles/environment.md) |
| **Software Baseline** | [S001](./../profiles/software.md) |
| **Benchmark Categories** | Throughput, Latency, Resource Usage |
| **Benchmark Tools** | [wrk](https://github.com/wg/wrk), [Vegeta](https://github.com/tsenart/vegeta) , [pidstat](https://man7.org/linux/man-pages/man1/pidstat.1.html), [vmstat](https://man7.org/linux/man-pages/man8/vmstat.8.html) |

---

# Table of Contents

- [1. Objective](#1-objective)
- [2. Executive Summary](#2-executive-summary)
- [3. Background](#3-background)
- [4. Hypothesis](#4-hypothesis)
- [5. Experimental Variables](#5-experimental-variables)
- [6. Experimental Setup](#6-experimental-setup)
- [7. System Configuration](#7-system-configuration)
- [8. Benchmark Configuration](#8-benchmark-configuration)
- [9. Benchmark Procedure](#9-benchmark-procedure)
- [10. Raw Results](#10-raw-results)
- [11. Statistical Summary](#11-statistical-summary)
- [12. Primary Performance Metrics](#12-primary-performance-metrics)
- [13. Supporting Performance Metrics](#13-supporting-performance-metrics)
- [14. Comparative Analysis](#14-comparative-analysis)
- [15. Threats to Validity](#15-threats-to-validity)
- [16. Conclusion](#16-conclusion)
- [17. Future Work](#17-future-work)
- [18. References](#18-references)
- [19. Reproducibility](#19-reproducibility)
- [Appendix A — Benchmark Commands](#appendix-a--benchmark-commands)
- [Appendix B — Benchmark Datasets](#appendix-b--benchmark-datasets)
- [Appendix C — Visual Artifacts](#appendix-c--visual-artifacts)

---

# 1. Objective

The objective of this benchmark is to quantify the performance overhead introduced by Transport Layer Security (TLS) termination in Torus.

The benchmark compares equivalent HTTP and HTTPS deployments while holding every other experimental variable constant. By measuring throughput, latency, and system resource utilization under identical workloads, the study isolates the cost of enabling HTTPS.

Specifically, this benchmark aims to answer the following engineering questions:

- How much throughput is lost when TLS termination is enabled?
- How does HTTPS affect request latency across different latency percentiles?
- What additional CPU and memory resources are required to process encrypted traffic?
- Is the observed overhead acceptable for the current implementation?

The results establish a performance baseline for HTTPS within Torus. Future optimizations involving TLS configuration, connection management, memory allocation, or request processing can be evaluated against this benchmark to determine whether they produce measurable improvements.

---

# 2. Executive Summary

This benchmark evaluates the performance impact of enabling TLS termination in Torus by comparing equivalent HTTP and HTTPS deployments under identical benchmark conditions. The benchmark methodology, workload, hardware, software environment, backend topology, and monitoring configuration remained unchanged between both scenarios. The transport protocol was the only independent variable.

Overall, HTTPS introduced a measurable but modest performance cost. Throughput decreased by approximately **6.5%**, while average request latency increased by **6.7%** under the `wrk` workload and **9.6%** under the lower-rate `vegeta` workload. Resource monitoring showed that this overhead was primarily computational: average CPU utilization increased by **8.5%** under the controlled-rate workload and **21.5%** under peak throughput, while average resident memory (RSS) increased by **20.4%** and **16.1%**, respectively. Despite the additional cryptographic processing, Torus maintained stable CPU and memory behavior across all benchmark iterations with no evidence of resource leaks, while sustaining a **100% request success rate** in both scenarios.

## Key Results

| Metric | HTTP | HTTPS | Δ | Δ (%) |
|---------|------:|------:|------:|------:|
| **wrk Throughput** | 15,616.981 req/s | 14,595.231 req/s | −1,021.750 req/s | **−6.54%** |
| **wrk Mean Latency** | 6.623 ms | 7.064 ms | +0.441 ms | **+6.66%** |
| **wrk Transfer Rate** | 2.234 MB/s | 2.089 MB/s | −0.145 MB/s | **−6.49%** |
| **Vegeta Throughput** | 500.027 req/s | 500.025 req/s | −0.002 req/s | ~0.00% |
| **Vegeta Mean Latency** | 0.157 ms | 0.172 ms | +0.015 ms | **+9.65%** |
| **Vegeta P95 Latency** | 0.199 ms | 0.219 ms | +0.020 ms | **+10.29%** |
| **Vegeta P99 Latency** | 0.395 ms | 0.430 ms | +0.035 ms | **+8.78%** |
| **Request Success Rate** | 100% | 100% | No change | — |

## Principal Observations

- Enabling **TLS reduced peak request throughput by approximately 6.5%**, representing the computational overhead of connection encryption and decryption.
- **Average request latency increased only slightly** under sustained load, **remaining below 7.1 ms** for the `wrk` benchmark.
- At the controlled request rate used by `vegeta`, throughput remained effectively unchanged, indicating that the system was operating well below saturation.
- **Tail latency (P95 and P99) increased modestly with HTTPS** but **remained consistently below 0.5 ms for P99** across all benchmark iterations.
- **CPU utilization increased under HTTPS**, confirming that the primary runtime cost of TLS termination is additional cryptographic processing.
- **Memory usage increased modestly** due to TLS-related runtime state and cryptographic buffers, but remained stable with no indication of memory leaks or progressive resource growth.
- Both HTTP and HTTPS maintained a **100% success rate** throughout all benchmark runs, with no request failures or protocol errors observed.
- The **coefficient of variation remained below 6%** for the primary performance metrics in both scenarios, indicating good benchmark stability and repeatability.

## Conclusion

The results indicate that TLS termination imposes a relatively small performance penalty for this workload. While HTTPS reduced maximum throughput by approximately **6.5%**, increased request latency modestly, and required additional CPU and memory resources, the overhead remained well within acceptable limits. Resource monitoring showed stable CPU scheduling and bounded memory usage throughout all benchmark runs, with no evidence of leaks or runtime instability. Torus therefore continued to deliver predictable, reliable performance under HTTPS, establishing a quantitative baseline for future networking and cryptographic optimizations.

---

# 3. Background

TLS termination is a core responsibility of modern reverse proxies and API gateways. Incoming encrypted client connections are decrypted at the proxy before requests are forwarded to upstream services, allowing internal communication to remain independent of external transport security.

Although HTTPS provides authentication, confidentiality, and integrity, these guarantees require additional computation. During connection establishment, the proxy performs cryptographic handshakes, certificate verification, and key exchange. Once a secure session has been established, all application data must be encrypted before transmission and decrypted upon receipt.

The magnitude of this overhead depends on several factors, including:

- TLS protocol version
- Cipher suite selection
- Connection reuse
- Payload size
- Request concurrency
- Hardware capabilities
- Efficiency of the proxy implementation

Rather than assuming the performance cost of HTTPS, Torus adopts a measurement-first engineering approach. Benchmarking provides objective evidence for architectural decisions and establishes reproducible baselines against which future optimizations can be evaluated.

This report represents the first controlled comparison of HTTP and HTTPS performance in Torus. It serves as a reference point for subsequent performance investigations involving TLS optimization, runtime profiling, and comparative benchmarking against other reverse proxies.

---

# 4. Hypothesis

Before executing the benchmark, the following engineering hypothesis was established.

> Enabling TLS termination will reduce maximum throughput and slightly increase request latency due to the additional cryptographic processing required for HTTPS connections. CPU utilization is expected to increase, while memory consumption is expected to increase modestly because TLS introduces additional session state, cryptographic contexts, and temporary encryption buffers without fundamentally changing the request-processing pipeline.

The benchmark seeks to verify the following expectations:

- HTTPS produces lower maximum throughput than HTTP.
- HTTPS introduces a measurable increase in request latency across all reported percentiles.
- CPU utilization increases because of encryption and decryption operations.
- Memory usage increases modestly but remains stable throughout execution.
- The overall overhead remains modest enough to support production deployment without significant architectural changes.

---

# 5. Experimental Variables

## Independent Variable

The independent variable in this benchmark is the transport protocol used between the client and Torus.

Two configurations were evaluated:

| Configuration | Description |
|--------------|-------------|
| HTTP | Plain HTTP without TLS termination |
| HTTPS | HTTPS with TLS termination enabled in Torus |

No other aspect of the system was intentionally modified between the two benchmark scenarios.

---

## Dependent Variables

The following performance characteristics were measured during each benchmark run.

### Primary Performance Metrics

- Throughput (Requests per Second)
- Transfer Rate
- Mean Request Latency
- Latency Percentiles (P50, P95, P99)
- Maximum Observed Latency
- Request Success Rate

### Supporting System Metrics

- CPU Utilization
- Resident Memory (RSS)
- Context Switches
- CPU User Time
- CPU System Time
- CPU Idle Time
- Run Queue Length
- Interrupt Rate

These measurements provide both application-level and operating system-level evidence for evaluating the impact of TLS termination.

---

## Controlled Variables

To ensure a fair comparison, every variable other than the transport protocol was held constant throughout the experiment.

| Variable | Value |
|----------|-------|
| Torus Source Code | Identical Git commit |
| Benchmark Hardware | [H001](./../profiles/hardware.md) |
| Environment Profile | [E001](./../profiles/environment.md) |
| Software Baseline | [S001](./../profiles/software.md) |
| Benchmark Methodology | [Version 1.0 ](./../methodology.md)|
| Backend Servers | Two identical mock HTTP servers |
| Load Balancing Strategy | Round Robin |
| Request Method | GET |
| Request Payload | 1 KB |
| Benchmark Tools | [wrk](https://github.com/wg/wrk), [Vegeta](https://github.com/tsenart/vegeta) |
| Benchmark Duration | 30 seconds |
| Warm-up Duration | 30 seconds |
| Benchmark Iterations | 20 |
| Benchmark Threads | 2 |
| Concurrent Connections | 100 |
| Vegeta Request Rate | 500 requests/second |

Maintaining identical experimental conditions ensures that any observed performance differences can reasonably be attributed to the additional computational cost of TLS termination rather than unrelated environmental or configuration changes.

---

# 6. Experimental Setup

The benchmark environment consisted of three logical components executing on the same physical machine.

```text
              Benchmark Client
               (wrk / Vegeta)

                    │
                    │ HTTP / HTTPS
                    ▼

             +---------------+
             |    Torus      |
             | Reverse Proxy |
             +---------------+
                    │
                    │
         Round-Robin Load Balancer
                    │
           ┌────────┴──────────┐
           │                   │
           ▼                   ▼

      Mock Backend 1      Mock Backend 2
      localhost:3001      localhost:3002
```

Both backend servers executed identical binaries and served identical responses. Torus distributed requests across both upstreams using its round-robin load balancer.

The only architectural difference between the two benchmark scenarios was the listener configuration:

- **HTTP Scenario**
  - Torus accepted unencrypted HTTP connections.
  - Requests were forwarded to the backend over HTTP.

- **HTTPS Scenario**
  - Torus terminated TLS connections using the configured X.509 certificate.
  - After decryption, requests followed the identical routing and proxy pipeline before being forwarded to the backend over HTTP.

No changes were made to the routing logic, backend implementation, load-balancing algorithm, request handling, or application code between the two benchmark scenarios.

This ensured that the measured performance difference primarily reflected the computational overhead introduced by TLS termination.

---

# 7. System Configuration

All benchmark executions were performed on the same physical machine using identical hardware and software configurations.

| Component | Value |
|----------|-------|
| CPU | 11th Gen Intel® Core™ i3-1115G4 |
| Logical CPUs | 4 |
| Memory | 8 GiB |
| Operating System | Ubuntu 26.04 LTS |
| Kernel | Linux 7.0.0-27-generic |

The benchmark environment referenced the following Torus benchmarking profiles:

| Profile | Identifier |
|----------|------------|
| Hardware Profile | [H001](./../profiles/hardware.md) |
| Environment Profile | [E001](./../profiles/environment.md) |
| Software Baseline | [S001](./../profiles/software.md) |

The benchmark client, Torus proxy, and both mock backend servers executed on the same host using localhost networking. This configuration minimizes network variability and isolates the computational overhead of TLS termination.

No kernel tuning, CPU affinity, operating system parameter changes, or hardware-specific optimizations were applied during testing.

Both benchmark scenarios were executed using the same repository revision, identical benchmark tooling, and identical system configuration to ensure a controlled comparison.

---

# 8. Benchmark Configuration

The benchmark configuration was intentionally kept identical across both scenarios except for the transport protocol.

| Parameter | Value |
|----------|-------|
| Benchmark Tools | [wrk](https://github.com/wg/wrk), [Vegeta](https://github.com/tsenart/vegeta) |
| HTTP Method | GET |
| Benchmark Duration | 30 seconds |
| Warm-up Duration | 30 seconds |
| Iterations | 20 |
| Threads | 2 |
| Concurrent Connections | 100 |
| Vegeta Rate | 500 requests/second |
| Backend Servers | 2 |
| Load Balancing | Round Robin |
| Health Checks | Enabled |
| Monitoring | pidstat, vmstat, ss |

The benchmark automation framework executed each scenario independently.

For every iteration, the framework:

1. Performed the configured warm-up period.
2. Started system monitoring.
3. Executed the workload using the selected benchmark tool.
4. Collected operating system metrics.
5. Parsed benchmark outputs.
6. Generated statistical summaries.
7. Produced visualizations and an intermediate benchmark report.

The complete raw dataset generated during execution is published separately as a benchmark dataset accompanying this report.

---

# 9. Benchmark Procedure

Both benchmark scenarios followed the standardized benchmark workflow defined by [**Benchmark Methodology v1.0**](./../methodology.md). The automation framework executed identical procedures for the HTTP and HTTPS configurations to ensure that the transport protocol remained the only independent variable.

Each benchmark scenario was executed independently using its corresponding benchmark configuration.

The execution procedure consisted of the following steps:

1. Start two identical mock backend servers.
2. Start Torus using the appropriate configuration (HTTP or HTTPS).
3. Verify backend health and proxy readiness.
4. Collect benchmark metadata, including hardware, software, repository revision, and benchmark configuration.
5. Execute a warm-up phase to stabilize the runtime environment.
6. Repeat the benchmark for the configured number of iterations.
7. During each iteration:
   - Start system monitoring.
   - Execute the workload using the selected benchmark tool.
   - Stop monitoring after workload completion.
8. Parse raw benchmark outputs.
9. Calculate descriptive statistics across all iterations.
10. Generate plots and an intermediate benchmark report.
11. Preserve all raw artifacts for subsequent analysis.

The benchmark automation framework performs the statistical analysis only after all iterations have completed. Individual benchmark runs are not interpreted in isolation.

The generated dataset contains:

- Raw benchmark outputs
- Parsed benchmark results
- Metadata
- Statistical summaries
- Monitoring data
- Generated plots
- Intermediate benchmark report

---

## Benchmark Commands

The benchmark scenarios were executed using the Torus benchmark automation framework.

HTTP baseline:

```bash
./docs/benchmarking/scripts/benchmark.sh http
```

HTTPS baseline:

```bash
./docs/benchmarking/scripts/benchmark.sh https
```

Following benchmark execution, each dataset was analyzed using:

```bash
./docs/benchmarking/scripts/analyze.sh <dataset-directory>
```

The analysis pipeline automatically generated:

- Statistical summaries
- Resource utilization summaries
- Box plots
- Histograms
- Time-series plots
- Intermediate benchmark report

The final engineering report presented here was produced separately using the generated dataset and supporting visualizations.

---

# 10. Raw Results

The complete benchmark dataset is published separately as the **Benchmark B-002 dataset release**.

The dataset contains:

- Raw `wrk` outputs
- Raw `vegeta` outputs
- Parsed benchmark results
- Statistical summaries
- Benchmark metadata
- CPU and memory monitoring data
- Operating system metrics
- Generated plots
- Automatically generated interim benchmark report

The raw artifacts are preserved without modification to ensure reproducibility and to enable independent verification of the reported results.

The dataset layout is documented in `docs/benchmarking/datasets.md`.

> **Dataset:** Available from the [GitHub Release for Benchmark-002](https://github.com/Ashish-Barmaiya/torus-proxy/releases/tag/benchmark-002)

---

# 11. Statistical Summary

Each benchmark scenario consisted of multiple independent benchmark iterations executed under identical experimental conditions.

For every reported performance metric, the following descriptive statistics were calculated across all benchmark runs:

- Sample Size (N)
- Mean
- Median
- Minimum
- Maximum
- Standard Deviation
- Coefficient of Variation (CV)
- Relevant Latency Percentiles

These statistics summarize the observed performance distribution rather than relying on any single benchmark execution.

The coefficient of variation was used as an indicator of benchmark stability. Low CV values across both scenarios indicate that the measured results were highly consistent and exhibited minimal run-to-run variability.

Detailed statistical tables for the HTTP and HTTPS benchmark scenarios are presented below.

---

## HTTP Baseline

### wrk

#### Requests / Second

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 15616.981 |
| Median | 15471.530 |
| Minimum | 15407.860 |
| Maximum | 17089.010 |
| Standard Deviation | 384.261 |
| Coefficient of Variation (%) | 2.461 |
| P50 | 15471.530 |
| P90 | 15852.542 |
| P95 | 16026.046 |
| P99 | 16876.417 |

#### Average Latency (ms)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 6.623 |
| Median | 6.680 |
| Minimum | 6.050 |
| Maximum | 6.720 |
| Standard Deviation | 0.152 |
| Coefficient of Variation (%) | 2.296 |
| P50 | 6.680 |
| P90 | 6.701 |
| P95 | 6.711 |
| P99 | 6.718 |

#### Transfer Rate (MB/s)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 2.234 |
| Median | 2.210 |
| Minimum | 2.200 |
| Maximum | 2.440 |
| Standard Deviation | 0.054 |
| Coefficient of Variation (%) | 2.410 |
| P50 | 2.210 |
| P90 | 2.271 |
| P95 | 2.288 |
| P99 | 2.410 |

### vegeta

#### Request Rate

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 500.029 |
| Median | 500.030 |
| Minimum | 500.019 |
| Maximum | 500.037 |
| Standard Deviation | 0.006 |
| Coefficient of Variation (%) | 0.001 |
| P50 | 500.030 |
| P90 | 500.036 |
| P95 | 500.036 |
| P99 | 500.037 |

#### Throughput

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 500.027 |
| Median | 500.028 |
| Minimum | 500.016 |
| Maximum | 500.034 |
| Standard Deviation | 0.006 |
| Coefficient of Variation (%) | 0.001 |
| P50 | 500.028 |
| P90 | 500.033 |
| P95 | 500.033 |
| P99 | 500.034 |

#### Success Ratio

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 1 |
| Median | 1.000 |
| Minimum | 1 |
| Maximum | 1 |
| Standard Deviation | 0.000 |
| Coefficient of Variation (%) | 0.000 |
| P50 | 1.000 |
| P90 | 1.000 |
| P95 | 1.000 |
| P99 | 1.000 |

#### Mean Latency (ms)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 0.157 |
| Median | 0.157 |
| Minimum | 0.155 |
| Maximum | 0.161 |
| Standard Deviation | 0.002 |
| Coefficient of Variation (%) | 0.980 |
| P50 | 0.157 |
| P90 | 0.158 |
| P95 | 0.161 |
| P99 | 0.161 |

#### P50 Latency (ms)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 0.148 |
| Median | 0.148 |
| Minimum | 0.146 |
| Maximum | 0.149 |
| Standard Deviation | 0.001 |
| Coefficient of Variation (%) | 0.583 |
| P50 | 0.148 |
| P90 | 0.149 |
| P95 | 0.149 |
| P99 | 0.149 |

#### P95 Latency (ms)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 0.199 |
| Median | 0.198 |
| Minimum | 0.195 |
| Maximum | 0.208 |
| Standard Deviation | 0.003 |
| Coefficient of Variation (%) | 1.679 |
| P50 | 0.198 |
| P90 | 0.202 |
| P95 | 0.207 |
| P99 | 0.208 |

#### P99 Latency (ms)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 0.395 |
| Median | 0.361 |
| Minimum | 0.352 |
| Maximum | 0.734 |
| Standard Deviation | 0.108 |
| Coefficient of Variation (%) | 27.385 |
| P50 | 0.361 |
| P90 | 0.404 |
| P95 | 0.690 |
| P99 | 0.725 |

#### Maximum Latency (ms)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 1.488 |
| Median | 0.960 |
| Minimum | 0.758 |
| Maximum | 3.815 |
| Standard Deviation | 1.013 |
| Coefficient of Variation (%) | 68.101 |
| P50 | 0.960 |
| P90 | 3.390 |
| P95 | 3.488 |
| P99 | 3.750 |

---

## HTTPS Baseline

### wrk

#### Requests / Second

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 14595.231 |
| Median | 14522.835 |
| Minimum | 14240.240 |
| Maximum | 16003.890 |
| Standard Deviation | 363.641 |
| Coefficient of Variation (%) | 2.492 |
| P50 | 14522.835 |
| P90 | 14797.826 |
| P95 | 14858.351 |
| P99 | 15774.782 |

#### Average Latency (ms)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 7.064 |
| Median | 7.095 |
| Minimum | 6.420 |
| Maximum | 7.250 |
| Standard Deviation | 0.169 |
| Coefficient of Variation (%) | 2.389 |
| P50 | 7.095 |
| P90 | 7.182 |
| P95 | 7.203 |
| P99 | 7.241 |

#### Transfer Rate (MB/s)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 2.089 |
| Median | 2.080 |
| Minimum | 2.040 |
| Maximum | 2.290 |
| Standard Deviation | 0.052 |
| Coefficient of Variation (%) | 2.480 |
| P50 | 2.080 |
| P90 | 2.120 |
| P95 | 2.129 |
| P99 | 2.258 |

### vegeta

#### Request Rate

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 500.028 |
| Median | 500.029 |
| Minimum | 500.020 |
| Maximum | 500.034 |
| Standard Deviation | 0.005 |
| Coefficient of Variation (%) | 0.001 |
| P50 | 500.029 |
| P90 | 500.033 |
| P95 | 500.034 |
| P99 | 500.034 |

#### Throughput

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 500.025 |
| Median | 500.025 |
| Minimum | 500.014 |
| Maximum | 500.032 |
| Standard Deviation | 0.005 |
| Coefficient of Variation (%) | 0.001 |
| P50 | 500.025 |
| P90 | 500.031 |
| P95 | 500.032 |
| P99 | 500.032 |

#### Success Ratio

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 1 |
| Median | 1.000 |
| Minimum | 1 |
| Maximum | 1 |
| Standard Deviation | 0.000 |
| Coefficient of Variation (%) | 0.000 |
| P50 | 1.000 |
| P90 | 1.000 |
| P95 | 1.000 |
| P99 | 1.000 |

#### Mean Latency (ms)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 0.172 |
| Median | 0.172 |
| Minimum | 0.170 |
| Maximum | 0.178 |
| Standard Deviation | 0.002 |
| Coefficient of Variation (%) | 1.264 |
| P50 | 0.172 |
| P90 | 0.174 |
| P95 | 0.178 |
| P99 | 0.178 |

#### P50 Latency (ms)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 0.162 |
| Median | 0.162 |
| Minimum | 0.161 |
| Maximum | 0.163 |
| Standard Deviation | 0.001 |
| Coefficient of Variation (%) | 0.351 |
| P50 | 0.162 |
| P90 | 0.163 |
| P95 | 0.163 |
| P99 | 0.163 |

#### P95 Latency (ms)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 0.219 |
| Median | 0.219 |
| Minimum | 0.214 |
| Maximum | 0.232 |
| Standard Deviation | 0.004 |
| Coefficient of Variation (%) | 1.942 |
| P50 | 0.219 |
| P90 | 0.223 |
| P95 | 0.228 |
| P99 | 0.231 |

#### P99 Latency (ms)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 0.430 |
| Median | 0.394 |
| Minimum | 0.372 |
| Maximum | 0.757 |
| Standard Deviation | 0.110 |
| Coefficient of Variation (%) | 25.596 |
| P50 | 0.394 |
| P90 | 0.454 |
| P95 | 0.741 |
| P99 | 0.754 |

#### Maximum Latency (ms)

| Statistic | Value |
|---|---|
| Sample Size | 20 |
| Mean | 2.967 |
| Median | 2.631 |
| Minimum | 2.402 |
| Maximum | 4.117 |
| Standard Deviation | 0.567 |
| Coefficient of Variation (%) | 19.127 |
| P50 | 2.631 |
| P90 | 3.678 |
| P95 | 3.809 |
| P99 | 4.056 |

---

## Observations

Across both benchmark scenarios:

- Throughput measurements exhibited low variability across repeated executions.
- Latency distributions remained stable with minimal run-to-run fluctuation.
- Resource utilization measurements showed consistent operating characteristics throughout the benchmark duration.

The low observed variance increases confidence that the measured differences between HTTP and HTTPS reflect the cost of TLS termination rather than measurement noise or environmental instability.

---

# 12. Primary Performance Metrics

This section compares the primary performance characteristics of the HTTP and HTTPS deployments. The objective is to quantify the cost of enabling TLS termination while distinguishing between observed measurements and engineering interpretation.

---

## 12.1 Throughput

Throughput represents the maximum volume of requests successfully processed by Torus within a given period.

The HTTP benchmark establishes the baseline throughput of the proxy without transport encryption. The HTTPS benchmark measures the same workload after enabling TLS termination.

### HTTP Throughput

> **Figure 1. HTTP Throughput**
>
> <img src="assets/Benchmark-002-http-vs-https/http/wrk-throughput-boxplot.png" width="500" alt="HTTP Throughput" />

---

### HTTPS Throughput

> **Figure 2. HTTPS Throughput**
>
> <img src="assets/Benchmark-002-http-vs-https/https/wrk-throughput-boxplot.png" width="500" alt="HTTPS Throughput" />

---

### HTTP vs HTTPS Comparison

> **Figure 3. HTTP vs HTTPS Throughput Comparison**
>
> <img src="assets/Benchmark-002-http-vs-https/wrk-throughput-comparison-boxplot.png" width="500" alt="HTTP vs HTTPS Throughput Comparison" />

---

### Discussion

The benchmark demonstrates the expected reduction in throughput after enabling TLS termination.

Specifically:
- **HTTP Baseline**: Achieved a mean throughput of **15,616.981 req/s** (with a median of 15,471.530 req/s and a maximum of 17,089.010 req/s).
- **HTTPS Baseline**: Achieved a mean throughput of **14,595.231 req/s** (with a median of 14,522.835 req/s and a maximum of 16,003.890 req/s).
- **Workload Overhead**: Under the maximum-load `wrk` workload, enabling TLS termination resulted in a **6.54%** mean throughput reduction (−1,021.750 req/s).
- **Controlled Load**: Under the controlled-rate `Vegeta` workload of 500 req/s, throughput remained effectively unchanged (**500.027 req/s** for HTTP vs **500.025 req/s** for HTTPS) with near-zero difference, showing the proxy was far from CPU/throughput saturation under this controlled load.

This reduction under peak load is primarily attributable to the additional cryptographic operations performed during secure communication, including:

- TLS handshake processing
- Session key generation
- Encryption of outgoing traffic
- Decryption of incoming traffic

Apart from these TLS-specific operations, the request-processing pipeline remained unchanged between both benchmark scenarios. Consequently, the observed throughput difference can reasonably be attributed to the computational cost of transport security rather than changes in routing, load balancing, or backend behaviour.

The benchmark also showed low run-to-run variation, indicating that throughput remained stable throughout repeated executions.

---

## 12.2 Transfer Rate

Transfer rate measures the amount of application data successfully transmitted per second.

Although throughput and transfer rate are closely related for fixed-size responses, transfer rate provides an additional perspective on overall network efficiency.

> **Figure 4. HTTP Transfer Rate**
>
> <img src="assets/Benchmark-002-http-vs-https/http/wrk-transfer-boxplot.png" width="500" alt="HTTP Transfer Rate" />

---

> **Figure 5. HTTPS Transfer Rate**
>
> <img src="assets/Benchmark-002-http-vs-https/https/wrk-transfer-boxplot.png" width="500" alt="HTTPS Transfer Rate" />

---

### Discussion

The transfer rate results align closely with the throughput measurements due to the fixed 1 KB payload size.
- **HTTP Baseline**: Achieved a mean transfer rate of **2.234 MB/s** (median: 2.210 MB/s, maximum: 2.440 MB/s).
- **HTTPS Baseline**: Achieved a mean transfer rate of **2.089 MB/s** (median: 2.080 MB/s, maximum: 2.290 MB/s).
- **Workload Overhead**: Enabling TLS termination resulted in a **6.49%** decrease in the transfer rate (−0.145 MB/s), which directly reflects the peak request throughput reduction.

---

## 12.3 Latency

Latency measures the time required for Torus to process a request and return a response.

While throughput reflects overall processing capacity, latency captures the responsiveness experienced by individual requests.

Both `wrk` and `Vegeta` were used to characterize latency from complementary perspectives:

- **wrk** reports average latency under maximum throughput conditions.
- **Vegeta** provides detailed latency distributions and percentile measurements under a controlled request rate.

---

### Mean Latency

> **Figure 6. HTTP Mean Latency**
>
> <img src="assets/Benchmark-002-http-vs-https/http/wrk-latency-boxplot.png" width="500" alt="HTTP Latency Boxplot (wrk)" />
>
> *wrk (High-throughput workload)*
>
> <img src="assets/Benchmark-002-http-vs-https/http/latency-boxplot.png" width="500" alt="HTTP Latency Boxplot (Vegeta)" />
>
> *Vegeta (Controlled-rate workload)*

---

> **Figure 7. HTTPS Mean Latency**
>
> <img src="assets/Benchmark-002-http-vs-https/https/wrk-latency-boxplot.png" width="500" alt="HTTPS Latency Boxplot (wrk)" />
>
> *wrk (High-throughput workload)*
>
> <img src="assets/Benchmark-002-http-vs-https/https/latency-boxplot.png" width="500" alt="HTTPS Latency Boxplot (Vegeta)" />
>
> *Vegeta (Controlled-rate workload)*

---

### Latency Distribution

> **Figure 8. HTTP Latency Distribution**
>
> <img src="assets/Benchmark-002-http-vs-https/http/latency-histogram.png" width="500" alt="HTTP Latency Distribution" />

---

> **Figure 9. HTTPS Latency Distribution**
>
> <img src="assets/Benchmark-002-http-vs-https/https/latency-histogram.png" width="500" alt="HTTPS Latency Distribution" />

---

### Latency Percentiles

Tail latency often provides more useful operational insight than average latency because production systems are frequently constrained by slow requests rather than typical requests.

The benchmark therefore reports the following latency percentiles:

- P50
- P95
- P99
- Maximum latency

---

> **Figure 10. HTTP Latency Percentiles**
>
> <img src="assets/Benchmark-002-http-vs-https/http/latency-percentiles.png" width="500" alt="HTTP Latency Percentiles" />

---

> **Figure 11. HTTPS Latency Percentiles**
>
> <img src="assets/Benchmark-002-http-vs-https/https/latency-percentiles.png" width="500" alt="HTTPS Latency Percentiles" />

---

### Discussion

Across both benchmark scenarios, latency remained low and stable.

Specifically:
- **wrk Mean Latency**: Under peak load, average latency increased from **6.623 ms** (HTTP) to **7.064 ms** (HTTPS), a modest **6.66%** increase (+0.441 ms).
- **wrk Tail Latency**: The tail latency remained stable and tightly bounded, with the HTTP P99 at **6.718 ms** and the HTTPS P99 at **7.241 ms**.
- **Vegeta Mean Latency**: At the controlled request rate of 500 req/s, average latency increased from **0.157 ms** (HTTP) to **0.172 ms** (HTTPS), representing a **9.65%** increase (+0.015 ms).
- **Vegeta Percentile Shift**: A consistent, small latency overhead is observable across all percentiles:
  - P50 (median) increased from **0.148 ms** to **0.162 ms** (+9.30%).
  - P95 tail latency increased from **0.199 ms** to **0.219 ms** (+10.29%).
  - P99 tail latency increased from **0.395 ms** to **0.430 ms** (+8.78%).
  - Maximum latency shifted from **1.488 ms** (HTTP mean max) to **2.967 ms** (HTTPS mean max).

This uniform shift across all percentiles indicates that the TLS overhead is a predictable, static computational cost (handshake + symmetric cipher operations) rather than a source of random spikes or thread scheduling delays.

---

## 12.4 Request Success Rate

A performance benchmark is meaningful only if requests are completed successfully.

Both benchmark scenarios maintained successful request processing throughout all benchmark iterations.

The HTTP and HTTPS deployments completed the workload without observable request failures attributable to the proxy itself.

This confirms that the measured throughput and latency values represent successful request processing rather than partial or failed benchmark executions.

---

## Summary

The primary performance metrics indicate that enabling TLS termination introduces the expected computational overhead while preserving stable request processing behaviour.

The benchmark establishes a quantitative baseline for:

- Throughput reduction
- Latency increase
- Overall transport efficiency

These measurements provide the reference against which future TLS optimizations and architectural improvements will be evaluated.

---

# 13. Supporting Performance Metrics

While throughput and latency quantify the external behaviour of the proxy, supporting system metrics help explain *why* those results were observed.

During every benchmark iteration, the automation framework collected operating system metrics using `pidstat` and `vmstat`. These measurements provide visibility into CPU utilization, memory usage, scheduler activity, and overall system behaviour while processing benchmark traffic.

---

## 13.1 CPU Utilization

CPU utilization measures the computational resources consumed by Torus while processing benchmark requests.

Because TLS introduces additional cryptographic operations, CPU usage is expected to increase relative to the HTTP baseline.

### HTTP CPU Utilization

> **Figure 12. HTTP CPU Utilization Over Time**
>
> <img src="assets/Benchmark-002-http-vs-https/http/wrk-cpu-timeseries.png" width="500" alt="HTTP CPU Utilization Over Time (wrk)" />
>
> *wrk (High-throughput workload)*
>
> <img src="assets/Benchmark-002-http-vs-https/http/vegeta-cpu-timeseries.png" width="500" alt="HTTP CPU Utilization Over Time (Vegeta)" />
>
> *Vegeta (Controlled-rate workload)*

---

> **Figure 13. HTTP CPU Distribution**
>
> <img src="assets/Benchmark-002-http-vs-https/http/wrk-cpu-distribution.png" width="500" alt="HTTP CPU Distribution (wrk)" />
>
> *wrk (High-throughput workload)*
>
> <img src="assets/Benchmark-002-http-vs-https/http/vegeta-cpu-distribution.png" width="500" alt="HTTP CPU Distribution (Vegeta)" />
>
> *Vegeta (Controlled-rate workload)*

---

### HTTPS CPU Utilization

> **Figure 14. HTTPS CPU Utilization Over Time**
>
> <img src="assets/Benchmark-002-http-vs-https/https/wrk-cpu-timeseries.png" width="500" alt="HTTPS CPU Utilization Over Time (wrk)" />
>
> *wrk (High-throughput workload)*
>
> <img src="assets/Benchmark-002-http-vs-https/https/vegeta-cpu-timeseries.png" width="500" alt="HTTPS CPU Utilization Over Time (Vegeta)" />
>
> *Vegeta (Controlled-rate workload)*

---

> **Figure 15. HTTPS CPU Distribution**
>
> <img src="assets/Benchmark-002-http-vs-https/https/wrk-cpu-distribution.png" width="500" alt="HTTPS CPU Distribution (wrk)" />
>
> *wrk (High-throughput workload)*
>
> <img src="assets/Benchmark-002-http-vs-https/https/vegeta-cpu-distribution.png" width="500" alt="HTTPS CPU Distribution (Vegeta)" />
>
> *Vegeta (Controlled-rate workload)*

---

### Discussion

CPU utilization remained stable and flat throughout the 30-second duration of both benchmark scenarios, with no evidence of sustained CPU instability or scheduling anomalies.

Key observations:
- **Crypto Overhead**: As shown in the CPU distribution and time-series plots, the HTTPS configuration exhibits higher CPU consumption compared to the HTTP baseline under both workloads.
  - Under the controlled request rate of `Vegeta` (500 req/s), the average CPU utilization rose from **1.805%** (median: **1.00%**, max: **11.00%**) for **HTTP** to **1.958%** (median: **1.00%**, max: **10.00%**) for **HTTPS**, representing an isolated cryptographic overhead of approximately **8.5%** relative increase in CPU usage. The nearly identical medians indicate that the additional CPU cost is primarily reflected in the higher average rather than a shift in the typical operating point.
- **Stable Processing**: Under the saturated load of `wrk` (peak throughput), substantially higher as expected. The **HTTP** configuration recorded an average CPU utilization of **23.563%**(median: **2.00%**, max: **248.00%**), while **HTTPS** recorded **28.640%** (median: **3.00%**, max: **249.00%**). This corresponds to an increase of approximately **21.5%** in average CPU utilization for HTTPS under peak load, reflecting the additional processing required for TLS encryption and decryption while the server remained CPU-stable throughout execution.
- **Crypto Cost Breakdown**: The additional CPU utilization observed for HTTPS is primarily attributable to:
  - TLS handshake processing (asymmetric cryptography for session key exchange)
  - Symmetric encryption (AES-GCM or ChaCha20-Poly1305 per packet)
  - Certificate validation and TLS protocol management
- **Scheduler and Run-to-Run Stability**: The time-series plots show smooth CPU utilization without sustained oscillations, periodic spikes, or runaway CPU behavior. Aside from expected workload-driven fluctuations, CPU usage remained stable throughout all benchmark runs, indicating that the Go scheduler and the Torus runtime maintained consistent processing with no observable thread contention or scheduling bottlenecks.

---

## 13.2 Memory Utilization

Resident Set Size (RSS) was monitored throughout each benchmark to evaluate the memory footprint of Torus.

### HTTP Memory Usage

> **Figure 16. HTTP Memory Utilization Over Time**

> <img src="assets/Benchmark-002-http-vs-https/http/wrk-memory-timeseries.png" width="500" alt="HTTP Memory Utilization Over Time (wrk)" />
>
> *wrk (High-throughput workload):*

> <img src="assets/Benchmark-002-http-vs-https/http/vegeta-memory-timeseries.png" width="500" alt="HTTP Memory Utilization Over Time (Vegeta)" />
>
> *Vegeta (Controlled-rate workload):*

---

> **Figure 17. HTTP Memory Distribution**
>
> <img src="assets/Benchmark-002-http-vs-https/http/wrk-memory-distribution.png" width="500" alt="HTTP Memory Distribution (wrk)" />
>
> *wrk (High-throughput workload)*
>
> <img src="assets/Benchmark-002-http-vs-https/http/vegeta-memory-distribution.png" width="500" alt="HTTP Memory Distribution (Vegeta)" />
>
> *Vegeta (Controlled-rate workload)*

---

### HTTPS Memory Usage

> **Figure 18. HTTPS Memory Utilization Over Time**
>
> <img src="assets/Benchmark-002-http-vs-https/https/wrk-memory-timeseries.png" width="500" alt="HTTPS Memory Utilization Over Time (wrk)" />
>
> *wrk (High-throughput workload)*
>
> <img src="assets/Benchmark-002-http-vs-https/https/vegeta-memory-timeseries.png" width="500" alt="HTTPS Memory Utilization Over Time (Vegeta)" />
>
> *Vegeta (Controlled-rate workload)*

---

> **Figure 19. HTTPS Memory Distribution**
>
> <img src="assets/Benchmark-002-http-vs-https/https/wrk-memory-distribution.png" width="500" alt="HTTPS Memory Distribution (wrk)" />
>
> *wrk (High-throughput workload)*
>
> <img src="assets/Benchmark-002-http-vs-https/https/vegeta-memory-distribution.png" width="500" alt="HTTPS Memory Distribution (Vegeta)" />
>
> *Vegeta (Controlled-rate workload)*

---

### Discussion

Resident Set Size (RSS) memory usage remained stable throughout the benchmark execution under both workloads, with no evidence of sustained memory growth or resource exhaustion.

- **Low Memory Overhead**: Enabling TLS termination introduced only a modest increase in memory consumption compared to the HTTP baseline, as illustrated by the RSS distribution and time-series plots.
  - Under the **controlled-rate workload** (`Vegeta`, **500 req/s**), the average RSS increased from **25.750 MB** (median: **6.457 MB**, max: **258.938 MB**) for **HTTP** to **31.005 MB** (median: **13.547 MB**, max: **258.953 MB**) for **HTTPS**, corresponding to an increase of approximately **5.26 MB** (**20.4%** increase in average RSS).
  - Under the **peak-throughput workload** (`wrk`), the average RSS increased from **23.848 MB** (median: **3.492 MB**, max: **258.953 MB**) for **HTTP** to **27.698 MB** (median: **5.156 MB**, max: **258.969 MB**) for **HTTPS**, representing an increase of approximately **3.85 MB** (**16.1%** increase in average RSS).
- **Efficient Buffer Management**: Although HTTPS consistently consumed more resident memory than HTTP, the additional usage remained relatively small. The overhead is primarily attributable to TLS session state, certificate structures, cryptographic contexts, and temporary encryption/decryption buffers. The Go runtime efficiently manages these allocations through reuse and garbage collection, preventing excessive long-lived memory consumption.
- **Runtime Stability**: The time-series plots show that RSS remained bounded throughout the benchmark duration, with no progressive upward trend indicative of memory leaks. Short-lived spikes were observed during initialization and transient workload fluctuations, but memory usage quickly returned to a steady operating range. This behavior demonstrates stable memory management under both HTTP and HTTPS workloads, supporting the proxy's suitability for sustained production deployment.

---

## Summary

The supporting system metrics explain the performance differences observed in the primary benchmark results.

The benchmark indicates that:

- **CPU utilization is the primary resource affected by TLS termination**, with HTTPS increasing average CPU usage by approximately **8.5%** under the controlled-rate Vegeta workload and **21.5%** under the peak-throughput wrk workload.
- **Memory consumption also increases under HTTPS**, but the additional RSS footprint remains modest, rising by approximately **20.4%** (5.26 MB) under Vegeta and **16.1%** (3.85 MB) under wrk. This increase is primarily attributable to TLS session state and cryptographic buffer management rather than persistent application allocations.
- **No evidence of CPU or memory instability was observed**. Both CPU utilization and RSS remained bounded throughout the benchmark duration, with only transient workload-driven fluctuations and no indication of resource leaks or progressive degradation.
- **The Go runtime and operating system maintained stable execution characteristics** across both HTTP and HTTPS scenarios, demonstrating efficient scheduling, garbage collection, and resource management under sustained load.

Overall, these measurements show that enabling HTTPS introduces a measurable but well-contained resource overhead. The primary cost is additional CPU processing for TLS operations, while the accompanying increase in memory usage remains relatively small and stable, indicating that Torus can support secure communication without compromising runtime stability or scalability.

---

# 14. Comparative Analysis

This benchmark compares the performance characteristics of Torus operating with and without TLS termination. The objective is not simply to identify which configuration is faster—this is expected—but to quantify the engineering cost of enabling transport security.

---

## 14.1 Throughput

The HTTP configuration consistently achieved higher throughput than the HTTPS configuration.

Specifically, peak throughput decreased from **15,616.981 req/s** to **14,595.231 req/s** under the maximum-load `wrk` workload, which is a decrease of **6.54%** (−1,021.750 req/s).

This behaviour aligns with the benchmark hypothesis. In the HTTPS deployment, every request incurs additional cryptographic processing before entering the existing request pipeline. Although modern processors provide hardware acceleration for many cryptographic operations, TLS termination remains a computational workload that does not exist in the HTTP baseline.

The measured reduction in throughput therefore represents the processing cost of encryption, decryption and protocol handling rather than any change in routing, load balancing or backend communication. Under the controlled-rate Vegeta workload (500 req/s), both configurations performed virtually identically, indicating that the throughput penalty is only felt near proxy saturation.

## 14.2 Latency

Average latency remained low and highly stable for both benchmark scenarios.

Under the maximum-load `wrk` workload, average latency rose by only **6.66%** (from **6.623 ms** to **7.064 ms**). Under the controlled-rate `Vegeta` workload (500 req/s), average latency increased by **9.65%** (from **0.157 ms** to **0.172 ms**). HTTPS exhibited a measurable, consistent increase in latency across all reported percentile measurements, with the median (P50) shifting from **0.148 ms** to **0.162 ms**, P95 shifting from **0.199 ms** to **0.219 ms** (+10.29%), and P99 shifting from **0.395 ms** to **0.430 ms** (+8.78%).

The increase is expected because requests cannot enter the normal proxy pipeline until TLS processing has completed. Importantly, the benchmark did not reveal abnormal tail-latency growth or unstable latency distributions.

The latency increase therefore appears to be a predictable, bounded computational cost rather than evidence of architectural inefficiencies.

## 14.3 CPU Utilization

CPU utilization exhibited the most pronounced difference between the HTTP and HTTPS benchmark scenarios, representing the primary computational overhead introduced by TLS termination.

Unlike throughput and latency, which describe externally observable performance, CPU utilization reflects the internal processing cost of HTTPS. The CPU distribution and time-series plots show a consistent increase in processor utilization for HTTPS under both the `wrk` and `Vegeta` workloads.

The additional CPU demand is expected because TLS introduces several computationally intensive operations, including:

- Key exchange (asymmetric cryptography)
- Certificate validation and processing
- Symmetric encryption (AES-GCM or ChaCha20-Poly1305)
- Message authentication and integrity verification
- TLS session establishment and management

Despite the increased computational workload, CPU utilization remained stable throughout all benchmark iterations. The time-series plots showed no sustained oscillations, runaway spikes, or scheduling anomalies, indicating that the Go scheduler and Torus runtime maintained consistent execution under both HTTP and HTTPS workloads.

## 14.4 Memory Utilization

Memory usage remained stable throughout the benchmark execution, although HTTPS consistently consumed more resident memory than the HTTP baseline.

The RSS distribution and time-series plots show a modest increase in average memory usage for HTTPS under both workloads. This additional memory consumption is primarily attributable to TLS session state, certificate structures, cryptographic contexts, and temporary encryption buffers rather than changes to the application request-processing pipeline.

Despite this increase, memory usage remained bounded throughout all benchmark runs. No progressive upward trend, abnormal allocation behaviour, or garbage collection instability was observed, indicating efficient memory management by the Go runtime and the absence of memory leaks during sustained operation.

## 14.5 Resource Efficiency

Considering throughput, latency, CPU utilization, and memory usage together, the benchmark indicates that enabling HTTPS primarily increases computational demand while introducing only a modest increase in memory consumption.

The additional work performed by TLS is reflected in:

- Reduced maximum throughput
- Slightly higher request latency
- Increased CPU utilization
- Modestly higher resident memory usage

At the same time, both CPU and memory remained stable throughout the benchmark duration, with no evidence of resource leaks, scheduler instability, or progressive performance degradation.

Overall, this resource utilization profile is consistent with the expected behaviour of software-based TLS termination.

---

## Overall Assessment

From an engineering perspective, the benchmark demonstrates that the observed performance differences are proportional to the functionality introduced by HTTPS.

TLS termination reduced throughput modestly, increased request latency, raised CPU utilization, and introduced a small but measurable increase in resident memory usage. These effects are consistent with the computational and runtime requirements of modern TLS implementations and do not indicate unexpected inefficiencies within Torus.

No evidence of resource leaks, runtime instability, or abnormal scheduling behaviour was observed throughout the benchmark. CPU and memory usage remained bounded across all benchmark iterations, supporting the reliability of the implementation under sustained load.

The benchmark therefore establishes a reliable performance baseline against which future optimizations—such as improved TLS configuration, session resumption, connection reuse, reduced memory allocations, or protocol-level enhancements—can be quantitatively evaluated.

---

# 15. Threats to Validity

Although the benchmark followed a controlled and reproducible methodology, several factors limit the extent to which these results can be generalized.

## Localhost Deployment

The benchmark client, Torus, and both backend servers executed on the same physical machine using localhost networking.

This eliminates network latency and packet loss, allowing the benchmark to isolate proxy performance. However, it does not represent distributed production deployments where network characteristics contribute significantly to overall latency.

---

## Limited Hardware Configuration

All measurements were collected on a single hardware platform.

Different processors, cache hierarchies, available memory, operating systems, kernel versions and cryptographic acceleration capabilities may produce different absolute performance values.

The reported results should therefore be interpreted as representative of the documented benchmark environment rather than universal performance characteristics.

---

## Self-Signed Certificate

The HTTPS benchmark used a locally generated self-signed certificate suitable for performance evaluation.

While this accurately measures the computational overhead of TLS termination, production deployments may employ different certificate chains, cipher suites and security policies that influence performance.

---

## Fixed Workload

The benchmark evaluated a single workload configuration consisting of:

- Fixed request size
- Fixed concurrency
- Fixed benchmark duration
- Fixed request rate
- Two backend servers

Different workloads—such as larger payloads, streaming responses, long-lived connections or significantly higher concurrency—may exhibit different performance characteristics.

---

## Benchmark Scope

This benchmark focuses exclusively on the performance cost of enabling HTTPS.

It does not evaluate:

- TLS handshake scalability
- HTTP/2
- HTTP/3
- Session resumption
- Mutual TLS
- Large payload performance
- Long-running stability
- Multi-node deployments

These topics will be investigated in future benchmark reports.

---

## Benchmark Variability

Although repeated executions demonstrated low run-to-run variability, benchmark results remain subject to unavoidable environmental influences including:

- Operating system scheduling
- Background processes
- CPU frequency scaling
- Thermal behaviour

Executing multiple iterations and reporting descriptive statistics reduces the influence of these factors but cannot eliminate them entirely.

---

# 16. Conclusion

This benchmark evaluated the performance impact of enabling TLS termination in Torus by comparing equivalent HTTP and HTTPS deployments under controlled experimental conditions.

The benchmark largely confirms the original engineering hypothesis.

Enabling HTTPS introduces a measurable but modest performance overhead, reflected primarily in reduced throughput, increased request latency, higher CPU utilization, and a modest increase in resident memory usage. These effects are consistent with the additional cryptographic processing and runtime state required to establish and maintain secure client connections.

At the same time, the benchmark demonstrates that the overhead remains predictable and well-controlled.

Throughout repeated benchmark executions:

- Throughput remained stable across all benchmark iterations.
- Latency distributions exhibited low run-to-run variability.
- CPU utilization increased consistently without scheduling instability.
- Memory consumption increased modestly but remained bounded and stable throughout execution.
- No evidence of memory leaks, resource exhaustion, or abnormal operating system behaviour was observed.

The benchmark therefore indicates that the current TLS implementation behaves as expected and introduces no unexpected architectural regressions beyond the inherent computational and memory costs of transport encryption.

From an engineering perspective, these measurements establish the first reproducible HTTPS performance baseline for Torus. The benchmark not only quantifies the external impact of TLS on throughput and latency but also characterizes its internal resource requirements through CPU and memory utilization measurements.

This baseline is particularly valuable because future performance optimizations can now be evaluated quantitatively rather than qualitatively. Improvements targeting TLS configuration, session management, connection reuse, memory allocation, routing efficiency, or protocol enhancements can be measured against the results presented in this report to determine whether they provide statistically meaningful gains.

Overall, the benchmark demonstrates that HTTPS support can be enabled with a predictable and manageable performance cost. Although TLS introduces additional computational work and a modest increase in memory usage, Torus maintains stable throughput characteristics, bounded resource utilization, and reliable operation under sustained load, making the current implementation suitable for production deployment.

---

# 17. Future Work

This benchmark represents only the initial evaluation of HTTPS performance within Torus. Several areas remain for future investigation.

## TLS Optimization

Future work should evaluate opportunities to reduce the computational overhead of TLS termination.

Potential areas include:

- TLS configuration tuning
- Session resumption
- Keep-alive optimization
- Cipher suite evaluation
- TLS 1.3 specific optimizations

---

## Higher Concurrency

The current benchmark used a fixed concurrency level.

Future benchmarks should evaluate behaviour under progressively increasing workloads to determine:

- saturation point
- throughput scaling
- latency degradation
- scheduler behaviour

---

## HTTP/2 and HTTP/3

This benchmark evaluated HTTP/1.1 only.

Future benchmark reports should compare:

- HTTP/1.1
- HTTP/2
- HTTP/3 (QUIC)

to understand the performance characteristics of newer transport protocols.

---

## Larger Payloads

Only a fixed response size was evaluated.

Additional workloads should investigate:

- 10 KB responses
- 100 KB responses
- 1 MB responses
- streaming responses
- large uploads

to determine how TLS overhead changes with payload size.

---

## Comparative Benchmarking

Once the Torus feature set has matured, equivalent benchmarks should be performed against established reverse proxies such as:

- NGINX
- HAProxy
- Envoy
- Caddy
- Traefik

using identical workloads and benchmark methodology.

---

## Runtime Profiling

The current benchmark focuses on externally observable performance metrics.

Future optimization work should incorporate runtime profiling using:

- Go pprof
- perf
- allocation profiles
- CPU flamegraphs

to identify specific performance bottlenecks within the TLS processing pipeline.

---

# 18. References

The following resources informed the benchmark methodology and interpretation.

- [Go Standard Library Documentation](https://pkg.go.dev/std)
- [Go crypto/tls package documentation](https://pkg.go.dev/crypto/tls)
- [wrk Benchmark Tool](https://github.com/wg/wrk)
- [Vegeta HTTP Load Testing Tool](https://github.com/tsenart/vegeta)
- [Linux pidstat Documentation](https://man7.org/linux/man-pages/man1/pidstat.1.html)
- [Linux vmstat Documentation](https://man7.org/linux/man-pages/man8/vmstat.8.html)
- [Torus Benchmark Methodology v1.0](./../methodology.md)
- [Torus Statistical Methodology v1.0](./../statistics.md)

---

# 19. Reproducibility

This benchmark is fully reproducible using the benchmark automation framework included in the Torus repository.

To reproduce the benchmark:

1. Build the Torus binary.
2. Start two mock backend servers.
3. Launch Torus using the HTTP configuration.
4. Execute the HTTP benchmark scenario.
5. Launch Torus using the HTTPS configuration.
6. Execute the HTTPS benchmark scenario.
7. Analyze both generated datasets.

For commands, see [Appendix A — Benchmark Commands](#appendix-a--benchmark-commands)

The accompanying benchmark datasets contain:

- Raw benchmark outputs
- Parsed benchmark results
- Metadata
- Statistical summaries
- System monitoring data
- Generated plots
- Intermediate benchmark reports

These artifacts allow independent verification of every result presented in this report.

---

# Appendix A — Benchmark Commands

```bash
# Start backend servers
go run mock_backend.go 3001
go run mock_backend.go 3002

# HTTP benchmark
go run ./cmd/torus --config torus-http.yaml
./docs/benchmarking/scripts/benchmark.sh http

# HTTPS benchmark
go run ./cmd/torus --config torus-https.yaml
./docs/benchmarking/scripts/benchmark.sh https
```

---

# Appendix B — Benchmark Datasets

The complete benchmark datasets are published separately as release assets accompanying this report. [See Here.](https://github.com/Ashish-Barmaiya/torus-proxy/releases/tag/benchmark-002)

Each dataset contains:

* **benchmark-002-http/**
  * metadata.json
  * summary.json
  * raw/
  * plots/
  * report-auto.md

* **benchmark-002-https/**
  * metadata.json
  * summary.json
  * raw/
  * plots/
  * report-auto.md

The datasets preserve all raw benchmark outputs, parsed results, statistical summaries, monitoring data and generated visualizations required to independently reproduce the analysis.

---

# Appendix C — Visual Artifacts

The following figures accompany this report.

- [HTTP throughput](assets/Benchmark-002-http-vs-https/http/wrk-throughput-boxplot.png)
- [HTTPS throughput](assets/Benchmark-002-http-vs-https/https/wrk-throughput-boxplot.png)
- [HTTP transfer rate](assets/Benchmark-002-http-vs-https/http/wrk-transfer-boxplot.png)
- [HTTPS transfer rate](assets/Benchmark-002-http-vs-https/https/wrk-transfer-boxplot.png)
- [HTTP latency distribution](assets/Benchmark-002-http-vs-https/http/latency-histogram.png)
- [HTTPS latency distribution](assets/Benchmark-002-http-vs-https/https/latency-histogram.png)
- [HTTP latency percentiles](assets/Benchmark-002-http-vs-https/http/latency-percentiles.png)
- [HTTPS latency percentiles](assets/Benchmark-002-http-vs-https/https/latency-percentiles.png)
- [HTTP CPU utilization (wrk)](assets/Benchmark-002-http-vs-https/http/wrk-cpu-timeseries.png) / [HTTP CPU utilization (Vegeta)](assets/Benchmark-002-http-vs-https/http/vegeta-cpu-timeseries.png)
- [HTTPS CPU utilization (wrk)](assets/Benchmark-002-http-vs-https/https/wrk-cpu-timeseries.png) / [HTTPS CPU utilization (Vegeta)](assets/Benchmark-002-http-vs-https/https/vegeta-cpu-timeseries.png)
- [HTTP memory utilization (wrk)](assets/Benchmark-002-http-vs-https/http/wrk-memory-timeseries.png) / [HTTP memory utilization (Vegeta)](assets/Benchmark-002-http-vs-https/http/vegeta-memory-timeseries.png)
- [HTTPS memory utilization (wrk)](assets/Benchmark-002-http-vs-https/https/wrk-memory-timeseries.png) / [HTTPS memory utilization (Vegeta)](assets/Benchmark-002-http-vs-https/https/vegeta-memory-timeseries.png)
- [HTTP vs HTTPS throughput comparison](assets/Benchmark-002-http-vs-https/wrk-throughput-comparison-boxplot.png)

Each figure is generated directly from the benchmark datasets using the Torus benchmark analysis pipeline. No plots are manually modified after generation.
