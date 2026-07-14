# Torus Benchmark Datasets

**Version:** 1.0
**Status:** Active

---

# 1. Purpose

This document defines how benchmark datasets are generated, stored, published, and referenced throughout the Torus project.

Benchmark datasets contain the complete evidence supporting every published benchmark report.

The repository stores benchmark reports and automation code, while benchmark datasets are published separately.

---

# 2. What is a Dataset?

A benchmark dataset is the complete collection of artifacts generated during a benchmark execution.

Typical contents include:

- benchmark metadata
- raw benchmark outputs
- parsed benchmark results
- statistical summaries
- generated plots
- monitoring data
- automated report

A dataset contains sufficient information to independently inspect the benchmark results.

---

# 3. Dataset Structure

Example

```text
datasets/

benchmark-002-http/

├── metadata.json
├── summary.json
├── report-auto.md
│
├── raw/
│   ├── wrk/
│   └── vegeta/
│
└── plots/
```

Additional profiling artifacts may be included for future benchmarks.

Examples:

- pprof profiles
- perf recordings
- flamegraphs
- heap profiles

---

# 4. Repository Policy

Benchmark datasets are **not committed** to the Git repository.

Reasons include:

- large file size
- binary benchmark artifacts
- generated files
- repository growth over time

The repository contains only:

- benchmark automation
- methodology
- benchmark reports
- dataset schema

---

# 5. Dataset Publication

Published benchmark reports reference an accompanying benchmark dataset.

Datasets are distributed separately as release assets.

Each published dataset corresponds to exactly one benchmark report.

This separation keeps the source repository lightweight while preserving complete reproducibility.

---

# 6. Dataset Lifecycle

Every benchmark execution creates a new dataset.

```
Benchmark Execution
        │
        ▼
Raw Outputs
        │
        ▼
Statistical Analysis
        │
        ▼
Plots
        │
        ▼
Automated Report
        │
        ▼
Dataset
```

Datasets are generated automatically by the benchmarking framework.

---

# 7. Immutability

Published benchmark datasets are immutable.

After publication:

- raw outputs are never modified
- generated summaries are never edited
- plots are never regenerated using different inputs

If benchmark methodology changes, a new dataset is generated.

Historical datasets remain unchanged.

---

# 8. Naming

Datasets use the benchmark identifier.

Examples

```
benchmark-001-http

benchmark-002-https

benchmark-003-routing
```

Identifiers are never reused.

---

# 9. Relationship to Benchmark Reports

Each benchmark report references one benchmark dataset.

```
Benchmark Report
        │
        ▼
Benchmark Dataset
        │
        ▼
Raw Evidence
```

The report contains engineering interpretation.

The dataset contains the supporting evidence.

---

# 10. Reproducibility

A published dataset should contain sufficient information to reproduce the benchmark analysis.

This includes:

- benchmark configuration
- hardware information
- software versions
- Git commit
- raw benchmark outputs
- parsed results
- statistical summaries
- generated plots

Running the analysis pipeline on the dataset should reproduce the published summary and plots.
