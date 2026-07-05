# Torus Benchmark Automation

**Version:** 1.0
**Status:** Active

---

# 1. Purpose

This document defines how benchmark execution, data collection, storage, and presentation are automated throughout the Torus project.

Automation ensures that benchmark reports remain reproducible, consistent, and easy to regenerate.

---

# 2. Repository Structure

benchmarking/

├── scripts/

├── datasets/

├── reports/

├── raw/

└── analysis/

---

# 3. Benchmark IDs

Every benchmark receives a permanent identifier.

Example

B001

B002

B003

Identifiers are never reused.

---

# 4. Benchmark Scripts

Each benchmark should have a dedicated script.

Examples

benchmark_tls.sh

benchmark_parser.sh

benchmark_reload.sh

benchmark_compare_nginx.sh

Scripts should:

- execute benchmarks
- collect system statistics
- save outputs
- produce machine-readable summaries

---

# 5. Dataset Layout

datasets/

B003/

results.json

summary.csv

wrk-output.txt

wrk2-output.txt

cpu-profile.pb.gz

heap-profile.pb.gz

pidstat.txt

vmstat.txt

---

# 6. JSON Result Format

Each benchmark produces a JSON summary.

Example

{
  "benchmark": "B003",
  "title": "TLS Termination",
  "version": "0.3.0",
  "hardware": "H001",
  "environment": "E001",
  "throughput": {
    "mean": 17865,
    "median": 17820,
    "stddev": 121
  },
  "latency": {
    "p50": 5.8,
    "p95": 8.9,
    "p99": 12.7
  },
  "cpu": {
    "average": 82.3
  }
}

---

# 7. CSV Summary

summary.csv

contains one row per benchmark execution.

Useful for plotting.

---

# 8. Raw Outputs

Raw outputs are preserved.

Never edited.

Never overwritten.

---

# 9. Graph Generation

Graphs should be generated from JSON or CSV.

Never manually edited.

Preferred charts

- Throughput
- Latency
- Resource usage
- Historical trends

---

# 10. Website Integration

The Torus website should read benchmark JSON directly.

Website responsibilities

- render charts
- render tables
- compare benchmark versions

Website should not contain benchmark logic.

The repository remains the source of truth.

---

# 11. Make Targets

Future benchmark commands.

Examples

make benchmark-tls

make benchmark-hotreload

make benchmark-parser

make benchmark-compare

Each command should:

- execute benchmark
- collect metrics
- store outputs
- generate summary

---

# 12. Reproducibility

Running the same benchmark twice using the same repository revision, hardware profile, environment profile, and methodology version should produce statistically similar results.

---

# 13. Historical Preservation

Historical datasets are immutable.

New benchmark results generate new datasets.

Historical results are never overwritten.

---

# 14. Future Automation

Future work

- CI benchmark pipeline
- Oracle VM automation
- Benchmark dashboard
- Automatic regression detection
- Performance history timeline
- Grafana integration
