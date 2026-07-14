# Torus Benchmark Automation

**Version:** 2.0
**Status:** Active

---

# 1. Purpose

This document describes the automated benchmarking framework used throughout the Torus project.

The framework standardizes benchmark execution, data collection, statistical analysis, plot generation, and report generation.

The primary objectives are:

- reproducibility
- consistency
- repeatability
- minimal manual work
- historical traceability

Every benchmark follows the same automated pipeline.

---

# 2. Architecture

```
                benchmark.sh
                     │
                     ▼
                 validate.sh
                     │
                     ▼
                 collect.sh
                     │
                     ▼
           wrk / vegeta execution
                     │
                     ▼
                 monitor.sh
                     │
                     ▼
              raw benchmark data
                     │
                     ▼
                 analyze.sh
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
       wrk_plots.py / vegeta_plots.py
                     │
                     ▼
                system_plots.py
                     │
                     ▼
              report_generator.py
                     │
                     ▼
                report-auto.md
                     │
                     ▼
          Final benchmark report (manual)
```

---

# 3. Repository Structure

```
benchmarking/

├── scripts/
│   ├── analyze.sh
│   ├── benchmark.sh
│   ├── collect.sh
│   ├── common.sh
│   ├── monitor.sh
│   └── validate.sh
│
├── analysis/
│   ├── analyze.py
│   ├── metrics.py
│   ├── parser.py
│   ├── report_generator.py
│   ├── system_plots.py
│   ├── vegeta_plots.py
│   └── wrk_plots.py
│
├── datasets/
├── reports/
└── templates/
```

---

# 4. Benchmark Execution

Benchmarks are executed through a single entry point.

```bash
./benchmark.sh <benchmark-scenario>
```

Example

```bash
./benchmark.sh http
```

The benchmark scenario defines:

- workload
- protocol
- benchmark tools
- concurrency
- duration
- iterations
- target URL

The automation framework reads the scenario configuration and executes the entire pipeline automatically.

---

# 5. Script Responsibilities

## benchmark.sh

Primary entry point.

Responsibilities:

- validate benchmark scenario
- prepare dataset directories
- collect benchmark metadata
- execute benchmark tools
- coordinate monitoring
- invoke analysis pipeline

---

## validate.sh

Performs pre-flight validation.

Examples:

- required tools installed
- benchmark configuration valid
- target endpoint reachable
- required directories exist

Benchmark execution stops immediately if validation fails.

---

## collect.sh

Collects benchmark metadata before execution.

Examples:

- timestamp
- hostname
- CPU information
- memory
- operating system
- kernel
- Git branch
- Git commit
- benchmark configuration

The collected information is written to

```
metadata.json
```

---

## monitor.sh

Collects operating-system metrics while benchmarks are running.

Current metrics include:

- pidstat
- vmstat

Outputs are stored alongside each benchmark run.

---

## analyze.sh

Runs the analysis pipeline after benchmark execution.

Responsibilities:

- parse benchmark outputs
- compute statistics
- generate plots
- generate automated report

---

## common.sh

Shared helper functions used by every script.

Examples:

- logging
- scenario lookup
- directory helpers
- benchmark configuration
- error handling

---

# 6. Analysis Pipeline

The analysis stage consists of several independent modules.

## parser.py

Parses raw benchmark outputs into a normalized JSON representation.

Supported inputs include:

- wrk
- Vegeta
- pidstat
- vmstat

Each benchmark run produces

```
parsed.json
```

---

## metrics.py

Computes descriptive statistics across benchmark runs.

Current metrics include:

- mean
- median
- minimum
- maximum
- standard deviation
- coefficient of variation
- p50
- p90
- p95
- p99

---

## analyze.py

Coordinates statistical analysis.

Responsibilities:

- parse every benchmark run
- aggregate statistics
- produce summary.json
- invoke plot generation

---

## wrk_plots.py

Generates throughput-oriented plots.

Current plots include:

- throughput boxplot
- throughput error bar
- latency boxplot
- transfer rate boxplot

---

## vegeta_plots.py

Generates latency-oriented plots.

Current plots include:

- latency boxplot
- latency histogram
- latency percentile curve

---

## system_plots.py

Generates operating-system resource visualizations.

Current plots include:

- CPU utilisation
- CPU distribution
- memory utilisation
- memory distribution
- context switches
- interrupts
- run queue
- disk activity

---

## report_generator.py

Generates an automated benchmark report.

The generated report is intentionally concise.

Its purpose is to provide:

- benchmark metadata
- statistical summaries
- generated plots
- experiment configuration

It serves as the starting point for writing the final engineering benchmark report.

---

# 7. Generated Dataset

Each benchmark execution produces a self-contained dataset.

Example

```
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

Raw outputs remain unchanged.

Generated files are derived entirely from the raw data.

---

# 8. Generated Artifacts

Every benchmark produces:

- metadata.json
- parsed benchmark outputs
- summary.json
- statistical plots
- report-auto.md

These artifacts collectively provide everything required to inspect, reproduce, and interpret the benchmark.

---

# 9. Reproducibility

The automation framework records:

- Git commit
- Git branch
- benchmark configuration
- hardware information
- operating system
- kernel version
- workload parameters

These records ensure that benchmark results remain reproducible.

---

# 10. Design Principles

The benchmarking framework follows several design principles.

### Single Entry Point

Every benchmark is executed through the same interface.

### Separation of Responsibilities

Benchmark execution, parsing, analysis, plotting, and report generation remain independent modules.

### Immutable Raw Data

Raw benchmark outputs are never modified.

Derived artifacts are regenerated from raw data whenever necessary.

### Extensibility

New benchmark tools can be integrated by implementing:

- parser support
- statistical summarization
- plot generation

without modifying the rest of the pipeline.
