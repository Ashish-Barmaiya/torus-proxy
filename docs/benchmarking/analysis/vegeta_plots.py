#!/usr/bin/env python3

"""
Plot generation for Vegeta benchmark results.

Unlike wrk, Vegeta is a fixed-rate benchmark.

Plots generated:

- Mean latency boxplot
- Latency percentile curve
- Aggregate latency histogram
"""

from __future__ import annotations

import json
from pathlib import Path

import matplotlib.pyplot as plt
import statistics


# Helpers

def _annotate_boxplot(ax, values, unit=""):
    if not values:
        return

    mean = statistics.mean(values)
    median = statistics.median(values)

    x = 1

    # Mean on LEFT
    ax.text(
        x - 0.22,
        mean,
        f"Mean\n{mean:.3f}{unit}",
        color="green",
        fontsize=9,
        ha="right",
        va="center",
    )

    # Median on RIGHT
    ax.text(
        x + 0.22,
        median,
        f"Median\n{median:.3f}{unit}",
        color="tab:orange",
        fontsize=9,
        ha="left",
        va="center",
    )

    # Max on upper-right
    ax.text(
        x + 0.12,
        max(values),
        f"Max\n{max(values):.3f}{unit}",
        fontsize=9,
        ha="left",
        va="center",
    )

    # Min on lower-right
    ax.text(
        x + 0.12,
        min(values),
        f"Min\n{min(values):.3f}{unit}",
        fontsize=9,
        ha="left",
        va="center",
    )

def _save(fig, output: Path):

    output.parent.mkdir(
        parents=True,
        exist_ok=True,
    )

    fig.tight_layout()

    fig.savefig(
        output,
        dpi=200,
    )

    plt.close(fig)


def _load_runs(
    vegeta_directory: Path,
):
    """
    Load every parsed Vegeta run.
    """

    runs = []

    for parsed in sorted(
        vegeta_directory.glob("run-*/parsed.json")
    ):

        with open(parsed) as f:

            data = json.load(f)

        if "vegeta" in data:

            runs.append(data)

    return runs


# Mean Latency Distribution

def latency_boxplot(
    vegeta_directory: Path,
    output: Path,
):
    """
    Distribution of mean latency across all benchmark runs.
    """

    runs = _load_runs(
        vegeta_directory,
    )

    values = [
        run["vegeta"]["latency_mean_ms"]
        for run in runs
    ]

    if not values:
        return

    fig, ax = plt.subplots(
        figsize=(6, 5),
    )

    ax.boxplot(
        values,
        patch_artist=True,
        showmeans=True,
    )

    ax.set_title(
        "Vegeta Mean Latency Across Runs"
    )

    ax.set_ylabel(
        "Latency (ms)"
    )

    ax.grid(
        axis="y",
        alpha=0.30,
    )

    _annotate_boxplot(ax, values, " ms")

    _save(
        fig,
        output,
    )


# Latency Percentiles

def latency_percentiles(
    summary: dict,
    output: Path,
):
    """
    Plot the representative latency percentiles from the benchmark.

    This chart summarizes the latency distribution using the
    aggregated statistics computed across benchmark runs.
    """

    if "vegeta" not in summary:
        return

    stats = summary["vegeta"]

    percentiles = [
    "P50",
    "P95",
    "P99",
    "Max",
    ]

    values = [
    stats["latency_p50_ms"]["mean"],
    stats["latency_p95_ms"]["mean"],
    stats["latency_p99_ms"]["mean"],
    stats["latency_max_ms"]["mean"],
    ]

    fig, ax = plt.subplots(
        figsize=(7, 4),
    )

    ax.plot(
        percentiles,
        values,
        marker="o",
        linewidth=2,
    )

    for x, y in zip(percentiles, values):
        ax.annotate(
            f"{y:.3f}",
            (x, y),
            xytext=(0, 7),
            textcoords="offset points",
            ha="center",
            fontsize=9,
        )

    ax.set_title(
        "Vegeta Latency Percentiles"
    )

    ax.set_ylabel(
        "Latency (ms)"
    )

    ax.grid(
        alpha=0.30,
    )

    _save(
        fig,
        output,
    )


# Latency Histogram

def _load_histogram(
    histogram_file: Path,
):
    """
    Parse Vegeta histogram.txt.

    Expected format:

    Bucket         #     %        Histogram
    [0s,    2ms]   1000  100.00%  #######
    [2ms,   4ms]   0     0.00%
    ...
    """

    buckets = []
    counts = []

    if not histogram_file.exists():
        return buckets, counts

    for line in histogram_file.read_text().splitlines():

        line = line.strip()

        if (
            not line
            or line.startswith("Bucket")
        ):
            continue

        parts = line.split()

        if len(parts) < 3:
            continue

        bucket = f"{parts[0]} {parts[1]}"

        try:
            count = int(parts[2])
        except ValueError:
            continue

        buckets.append(bucket)
        counts.append(count)

    return buckets, counts


def latency_histogram(
    vegeta_directory: Path,
    output: Path,
):
    """
    Aggregate latency histogram across every benchmark run.
    """

    aggregate = {}

    for run_directory in sorted(
        vegeta_directory.glob("run-*")
    ):

        histogram = (
            run_directory
            / "histogram.txt"
        )

        buckets, counts = _load_histogram(
            histogram,
        )

        for bucket, count in zip(
            buckets,
            counts,
        ):

            aggregate[bucket] = (
                aggregate.get(bucket, 0)
                + count
            )

    if not aggregate:
        return

    labels = list(
        aggregate.keys()
    )

    values = list(
        aggregate.values()
    )

    fig, ax = plt.subplots(
        figsize=(9, 5),
    )

    ax.bar(
        labels,
        values,
    )

    ax.set_title(
        "Aggregate Latency Histogram"
    )

    ax.set_xlabel(
        "Latency Bucket"
    )

    ax.set_ylabel(
        "Requests"
    )

    plt.xticks(
        rotation=30,
        ha="right",
    )

    ax.grid(
        axis="y",
        alpha=0.30,
    )

    _save(
        fig,
        output,
    )


# Public API

def generate(
    summary: dict,
    dataset_directory: Path,
):
    """
    Generate every Vegeta plot.
    """

    if "vegeta" not in summary:
        return

    vegeta_directory = (
        dataset_directory
        / "raw"
        / "vegeta"
    )

    if not vegeta_directory.exists():
        return

    output_directory = (
        dataset_directory
        / "plots"
        / "vegeta"
    )

    output_directory.mkdir(
        parents=True,
        exist_ok=True,
    )

    print(
        "  [INFO] Vegeta: latency boxplot..."
    )

    latency_boxplot(
        vegeta_directory,
        output_directory / "latency-boxplot.png",
    )

    print(
        "  [INFO] Vegeta: percentile plot..."
    )

    latency_percentiles(
        summary,
        output_directory / "latency-percentiles.png",
    )

    print(
        "  [INFO] Vegeta: latency histogram..."
    )

    latency_histogram(
        vegeta_directory,
        output_directory / "latency-histogram.png",
    )
