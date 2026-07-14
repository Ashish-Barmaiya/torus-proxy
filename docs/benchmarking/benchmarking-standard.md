# Torus Benchmarking Standard

**Version:** 1.0
**Status:** Active
**Last Updated:** 2026-07-05

---

# 1. Purpose

This document defines the official benchmarking philosophy, standards, repository organization, automation architecture, naming conventions, dataset management, and lifecycle for all performance investigations conducted on Torus.

The purpose of this standard is to ensure that every published benchmark is:

- Reproducible
- Fair
- Statistically meaningful
- Comparable across time
- Properly documented
- Scientifically defensible

Every benchmark report in this repository MUST conform to this standard unless explicitly stated otherwise.

---

# 2. Philosophy

Benchmarking exists to answer engineering questions.

It does **not** exist to produce the largest throughput number or to make Torus appear faster than competing software.

Every benchmark must answer a single engineering hypothesis.

Good example:

> Does TLS termination introduce measurable CPU overhead while preserving low latency under persistent connections?

Bad example:

> Benchmark TLS.

---

# 3. Core Principles

Every benchmark published by Torus must satisfy the following principles.

## Principle 1 — Reproducibility

Another engineer should be able to reproduce every published benchmark using the repository contents together with the corresponding published benchmark dataset.

This includes:

- benchmark automation
- benchmark configuration
- benchmark dataset
- commands
- topology
- hardware profile
- software versions
- benchmark tool versions

---

## Principle 2 — Controlled Variables

Only one independent variable should change between two comparable benchmark runs.

Example:

Correct:

HTTP

↓

HTTPS

Everything else remains identical.

Incorrect:

HTTP

↓

HTTPS

↓

New Go version

↓

Different hardware

↓

Different backend

---

## Principle 3 — Historical Integrity

Published benchmark reports are immutable.

Once a benchmark report is published:

- Results are never modified.
- Historical numbers are never overwritten.
- Methodology is never retroactively changed.

If improvements are made:

- create a new benchmark report
- or publish a new methodology version

---

## Principle 4 — Transparency

Every benchmark must include:

- exact commands
- benchmark duration
- hardware profile
- environment profile
- methodology version
- published benchmark dataset
- analysis

---

## Principle 5 — Measurement Before Optimization

Optimization work must always begin with measurement.

The workflow is:

Measure

↓

Identify bottleneck

↓

Hypothesis

↓

Implement optimization

↓

Measure again

Never optimize without first identifying the bottleneck.

---

# 4. Benchmark Lifecycle

Every benchmark follows the same lifecycle.

Problem

↓

Question

↓

Hypothesis

↓

Experiment Design

↓

Benchmark Execution

↓

Raw Dataset Generation

↓

Statistical Analysis

↓

Automated Report

↓

Engineering Interpretation

↓

Published Benchmark Report


Benchmark reports are expected to document every stage.

---

# 5. Repository Structure

docs/

    benchmarking/
    ├── benchmarking-standard.md
    ├── methodology.md
    ├── statistics.md
    ├── benchmark-matrix.md
    ├── benchmark-tooling.md
    ├── benchmark-automation.md
    ├── datasets.md
    ├── report-template.md
    ├── profiles/
    ├── analysis/
    ├── scripts/
    ├── reports/
    └── datasets/

---

# 6. Benchmark Identifiers

Every benchmark receives a permanent identifier.

Format:

B001

B002

B003

...

Identifiers are never reused.

Identifiers are never renumbered.

---

# 7. Methodology Versioning

Benchmark methodology evolves over time.

Methodology changes are versioned.

Example:

Methodology v1.0

↓

Methodology v1.1

↓

Methodology v2.0

Benchmark reports always reference the methodology version they use.

Historical reports continue referencing the methodology available when they were created.

---

# 8. Hardware Profiles

Hardware specifications are defined separately.

Every benchmark references a hardware profile.

Example:

Hardware Profile H001

Hardware Profile H002

Hardware profiles are immutable once published.

---

# 9. Environment Profiles

Hardware and topology are independent.

Environment profiles describe:

- benchmark topology
- client location
- proxy location
- backend location
- network characteristics

Benchmark reports reference both:

Hardware Profile

+

Environment Profile

---

# 10. Benchmark Categories

Every benchmark belongs to one or more categories.

- Throughput
- Latency
- Resource Usage
- Scalability
- Reliability
- Regression
- Comparative

Categories are defined in Benchmark Matrix.

---

# 11. Data Storage

Benchmark reports summarize the engineering findings.

Supporting benchmark datasets are generated automatically by the benchmarking framework and published separately as release artifacts.

A typical dataset contains:

- metadata.json
- summary.json
- report-auto.md
- raw benchmark outputs
- parsed benchmark outputs
- generated plots
- monitoring data

Published datasets are immutable and are referenced by the corresponding benchmark report.

---

# 12. Public Reporting Policy

All benchmark reports are public.

All benchmark methodology is public.

All benchmark commands are public.

Published benchmark datasets are distributed alongside benchmark reports as release artifacts.

Private engineering notes are excluded.

---

# 13. Engineering Ethics

Benchmark reports must never:

- hide failed experiments
- cherry-pick best runs
- manipulate graphs
- omit unfavorable comparisons
- claim unsupported conclusions

Interpretations must remain proportional to the collected evidence.

---

# 14. Scope

This standard applies to every benchmark published for Torus, including:

- microbenchmarks
- feature benchmarks
- regression benchmarks
- comparative benchmarks
- optimization studies

No benchmark report should deviate from this standard without explicitly documenting the reason.

This standard applies equally to manually authored benchmark reports and the automated benchmarking framework that generates supporting datasets and preliminary reports.
