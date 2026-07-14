# Torus Benchmarking Framework

This directory contains the complete benchmarking framework used throughout the Torus project.

Torus publishes the methodology, statistical analysis, benchmark environment, tooling, datasets, and engineering rationale behind every published benchmark.

The objective is to ensure that every reported result is:

- Reproducible
- Statistically meaningful
- Fair
- Transparent
- Comparable across time

The framework includes benchmark automation, statistical analysis, visualization, report generation, and engineering documentation.

---

# Philosophy

Benchmarking exists to answer engineering questions.

Every benchmark begins with a hypothesis, follows a standardized methodology, and concludes with a documented analysis supported by raw data.

Performance claims made by Torus should always be traceable back to an individual benchmark report.

---

# Directory Structure

```text
benchmarking/
│
├── README.md
├── benchmark-automation.md
├── benchmark-matrix.md
├── benchmark-tooling.md
├── benchmarking-standard.md
├── datasets.md
├── methodology.md
├── report-template.md
├── statistics.md
│
├── profiles/
|   ├── environment.md
|   ├── hardware.md
|   └── software.md
|
├── reports/
├── datasets/
├── analysis/
├── scripts/
├── scenarios/
└── templates/
```

---

# Documentation

| Document | Purpose |
|----------|---------|
| [`benchmarking-standard.md` ](benchmarking-standard.md)| Engineering principles and lifecycle governing all benchmarks |
| [`methodology.md`](methodology.md) | Experimental design and execution methodology |
| [`statistics.md`](statistics.md) | Statistical analysis methodology and interpretation |
| [`benchmark-matrix.md`](benchmark-matrix.md) | Performance characteristics evaluated throughout the project |
| [`benchmark-tooling.md`](benchmark-tooling.md) | Benchmarking, profiling and monitoring tools |
| [`benchmark-automation.md`](benchmark-automation.md) | Architecture of the automated benchmarking pipeline |
| [`datasets.md`](datasets.md) | Dataset structure, publication policy and lifecycle |
| [`report-template.md`](report-template.md) | Standard template used for published benchmark reports |

---

# Benchmark Workflow

Every benchmark follows the same engineering workflow.

```text
Engineering Question
        │
        ▼
Hypothesis
        │
        ▼
Experiment Design
        │
        ▼
Benchmark Execution
        │
        ▼
Raw Data Collection
        │
        ▼
Statistical Analysis
        │
        ▼
Engineering Interpretation
        │
        ▼
Published Benchmark Report
```

---

# Automation Pipeline

Benchmark execution is fully automated.

```text
benchmark.sh
      │
      ▼
validate.sh
      │
      ▼
collect.sh
      │
      ▼
wrk / vegeta
      │
      ▼
monitor.sh
      │
      ▼
Raw Dataset
      │
      ▼
parser.py
      │
      ▼
metrics.py
      │
      ▼
summary.json
      │
      ▼
Plot Generation
      │
      ▼
report-auto.md
      │
      ▼
Final Benchmark Report
```

---

# Benchmark Reports

Published benchmark reports are stored in

```text
reports/
```

Each report documents:

- Objective
- Background
- Hypothesis
- Experimental setup
- Statistical summary
- Performance analysis
- Threats to validity
- Conclusion
- Reproducibility

Benchmark reports contain engineering interpretation rather than raw benchmark data.

---

# Benchmark Datasets

Each benchmark execution produces a self-contained dataset.

Typical contents include:

- benchmark metadata
- raw benchmark outputs
- parsed results
- statistical summaries
- generated plots
- automated report

Datasets are generated automatically by the benchmarking framework.

Published benchmark datasets are distributed separately from the repository as release assets to keep the repository lightweight while preserving reproducibility.

---

# Design Principles

The benchmarking framework follows several core principles:

- Measure before optimizing.
- Change only one independent variable at a time.
- Preserve raw benchmark outputs.
- Keep statistical analysis reproducible.
- Separate observed results from engineering interpretation.
- Maintain historical benchmark integrity.

---

# Current Status

The benchmarking infrastructure is complete and actively used for evaluating Torus.

Future work focuses primarily on publishing benchmark reports for new features, optimizations, regressions, scalability studies, and comparative evaluations.
