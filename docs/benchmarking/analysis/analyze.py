#!/usr/bin/env python3

"""
Benchmark analysis entrypoint.

Pipeline:

raw benchmark outputs
        ↓
parser.py
        ↓
parsed.json
        ↓
statistics.py
        ↓
summary.json
        ↓
plots.py
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

from parser import (
    parse_pidstat,
    parse_vegeta,
    parse_vmstat,
    parse_wrk,
)

from statistics import (
    summarize_vegeta,
    summarize_wrk,
)

from plots import (
    cpu_plot,
    latency_distribution,
    latency_plot,
    memory_plot,
    throughput_plot,
)


def load_runs(tool_dir: Path):
    runs = []

    for run_dir in sorted(tool_dir.glob("run-*")):

        data = {}

        if (run_dir / "wrk.txt").exists():
            data["wrk"] = parse_wrk(run_dir / "wrk.txt")

        if (run_dir / "vegeta.json").exists():
            data["vegeta"] = parse_vegeta(run_dir / "vegeta.json")

        if (run_dir / "pidstat.txt").exists():
            data["pidstat"] = parse_pidstat(run_dir / "pidstat.txt")

        if (run_dir / "vmstat.txt").exists():
            data["vmstat"] = parse_vmstat(run_dir / "vmstat.txt")

        runs.append(data)

    return runs


def pidstat_cpu(runs):

    values = []

    for run in runs:

        if "pidstat" not in run:
            continue

        for sample in run["pidstat"]:
            values.append(sample["cpu"])

    return values


def pidstat_memory(runs):

    values = []

    for run in runs:

        if "pidstat" not in run:
            continue

        for sample in run["pidstat"]:
            values.append(sample["rss"])

    return values


def vegeta_latency(runs):

    values = []

    for run in runs:

        if "vegeta" not in run:
            continue

        values.append(run["vegeta"]["latency_mean_ms"])

    return values


def analyze(dataset: Path):

    raw = dataset / "raw"
    plots = dataset / "plots"

    plots.mkdir(exist_ok=True)

    summary = {}

    wrk_dir = raw / "wrk"

    if wrk_dir.exists():

        runs = load_runs(wrk_dir)

        summary["wrk"] = summarize_wrk(runs)

        throughput_plot(
            summary,
            plots / "wrk-throughput.png",
        )

        latency_plot(
            summary,
            plots / "wrk-latency.png",
        )

        cpu_plot(
            pidstat_cpu(runs),
            plots / "wrk-cpu.png",
        )

        memory_plot(
            pidstat_memory(runs),
            plots / "wrk-memory.png",
        )

    vegeta_dir = raw / "vegeta"

    if vegeta_dir.exists():

        runs = load_runs(vegeta_dir)

        summary["vegeta"] = summarize_vegeta(runs)

        latency_plot(
            summary,
            plots / "vegeta-latency.png",
        )

        cpu_plot(
            pidstat_cpu(runs),
            plots / "vegeta-cpu.png",
        )

        memory_plot(
            pidstat_memory(runs),
            plots / "vegeta-memory.png",
        )

        latency_distribution(
            vegeta_latency(runs),
            plots / "vegeta-histogram.png",
        )

    with open(dataset / "summary.json", "w") as fp:
        json.dump(summary, fp, indent=4)


def main():

    parser = argparse.ArgumentParser()

    parser.add_argument(
        "dataset",
        type=Path,
        help="Benchmark dataset directory",
    )

    args = parser.parse_args()

    analyze(args.dataset)


if __name__ == "__main__":
    main()
