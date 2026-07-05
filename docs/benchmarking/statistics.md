# Torus Benchmark Statistical Methodology

**Version:** 1.0
**Status:** Active
**Last Updated:** 2026-07-05

---

# 1. Purpose

This document defines the statistical methodology used to analyze benchmark results throughout the Torus project.

Its purpose is to ensure that benchmark conclusions are based on reproducible and statistically meaningful evidence rather than isolated benchmark runs.

All benchmark reports MUST follow this document unless explicitly stated otherwise.

---

# 2. Philosophy

A benchmark result is not a single number.

Every benchmark is treated as a sample from a performance distribution.

The objective is to estimate the true behaviour of the system while minimizing noise introduced by the operating system, scheduler, CPU frequency scaling, thermal effects and background processes.

---

# 3. Sample Size

Every benchmark consists of multiple independent runs.

Minimum:

10 runs

Recommended:

20 runs

Exceptional cases may use larger sample sizes.

Single-run benchmarks must never be published.

---

# 4. Warm-Up

Warm-up runs are never included in the statistical analysis.

Purpose:

- JIT effects (where applicable)
- cache warming
- connection establishment
- TCP slow start
- memory allocation stabilization
- runtime initialization

Default warm-up:

30 seconds

---

# 5. Reported Statistics

Every benchmark report must include the following statistics.

## Mean

Arithmetic average of all benchmark runs.

Represents expected performance.

---

## Median

Middle observation after sorting.

Represents typical performance.

Less sensitive to outliers than the mean.

---

## Minimum

Lowest observed value.

Useful for understanding worst observed throughput.

---

## Maximum

Highest observed value.

Useful for understanding best observed throughput.

---

## Standard Deviation

Measures variability between runs.

Low standard deviation indicates stable performance.

High standard deviation suggests unstable behaviour or noisy measurements.

---

## Coefficient of Variation (CV)

Formula

CV = StandardDeviation / Mean

Reported as a percentage.

Interpretation:

Excellent

< 2%

Good

2–5%

Acceptable

5–10%

Poor

>10%

Benchmarks with high CV should be investigated before publication.

---

# 6. Percentiles

Latency measurements must include:

p50

Median latency.

---

p90

Latency below which 90% of requests complete.

---

p95

Common production latency metric.

---

p99

Primary tail latency metric.

Mandatory.

---

p99.9

Recommended for high-performance investigations.

---

Maximum

Highest observed latency.

Useful for detecting pathological behaviour.

---

# 7. Confidence Intervals

Future methodology versions may report:

95% Confidence Interval

Confidence intervals estimate uncertainty in the reported mean.

Current benchmark reports are not required to include confidence intervals.

---

# 8. Outliers

Outliers must never be removed automatically.

An outlier may only be excluded if a documented external cause exists.

Examples:

- benchmark machine rebooted
- benchmark interrupted
- power management event
- unrelated background workload
- benchmark tool failure

Excluded observations must be documented.

---

# 9. Comparing Two Systems

When comparing two implementations:

Always compare:

Mean

Median

Standard Deviation

Percentiles

Resource Usage

Never compare only the maximum throughput.

---

# 10. Throughput Analysis

Throughput reports should include:

Mean RPS

Median RPS

Minimum

Maximum

Standard Deviation

Coefficient of Variation

Graphs are strongly encouraged.

---

# 11. Latency Analysis

Latency reports should include:

p50

p90

p95

p99

p99.9 (recommended)

Maximum

Latency distributions should be visualized whenever practical.

---

# 12. Resource Analysis

Resource usage should be summarized using:

Average CPU

Peak CPU

Average Memory

Peak Memory

Average Goroutines

Peak Goroutines

Average File Descriptors

Peak File Descriptors

---

# 13. Relative Change

Whenever two benchmark results are compared:

Report both:

Absolute Difference

and

Percentage Difference

Example

HTTP

17,500 RPS

HTTPS

16,300 RPS

Absolute Difference

−1,200 RPS

Percentage Difference

−6.86%

Percentages alone should never be reported.

---

# 14. Benchmark Stability

A benchmark should not be considered reliable solely because it is fast.

Stability is equally important.

Indicators of stability include:

Low standard deviation

Low coefficient of variation

Stable latency percentiles

Stable CPU utilisation

Stable memory usage

---

# 15. Visualization Standards

Recommended charts:

Throughput

Bar Chart

---

Performance Evolution

Line Chart

---

Latency Distribution

Histogram

---

Latency Comparison

Box Plot

---

Resource Usage

Time Series

---

Historical Performance

Line Chart

---

Scatter plots should only be used when investigating correlations.

Pie charts should never be used.

---

# 16. Interpretation Guidelines

Observed data should always be separated from interpretation.

Correct:

Observed

↓

Possible Explanation

↓

Supporting Evidence

↓

Conclusion

Incorrect:

Observation

↓

Assumption

↓

Conclusion

Benchmark reports must avoid unsupported causal claims.

---

# 17. Statistical Limitations

Benchmark statistics describe the tested environment only.

Results obtained on:

- different hardware
- different kernels
- different Go versions
- different benchmark tools
- different network topologies

may differ significantly.

Benchmark conclusions should always acknowledge this limitation.

---

# 18. Publication Checklist

Before publishing a benchmark report verify that:

✓ Sample size documented

✓ Warm-up documented

✓ Mean reported

✓ Median reported

✓ Standard deviation reported

✓ Coefficient of variation reported

✓ Percentiles reported (latency)

✓ Absolute and percentage differences reported

✓ Resource usage summarized

✓ Graphs included

✓ Outliers documented

✓ Limitations discussed

Only then should benchmark results be considered complete.
