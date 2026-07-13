#!/usr/bin/env python3

"""
Plots for wrk benchmark results.
"""

from __future__ import annotations

import json
from pathlib import Path

from matplotlib.pylab import std
import matplotlib.pyplot as plt
import statistics

from numpy import std

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
    output.parent.mkdir(parents=True, exist_ok=True)
    fig.tight_layout()
    fig.savefig(output, dpi=200)
    plt.close(fig)


def _load_runs(wrk_directory: Path):

    runs = []

    for run in sorted(wrk_directory.glob("run-*")):

        parsed = run / "parsed.json"

        if not parsed.exists():
            continue

        with open(parsed) as f:
            data = json.load(f)

        if "wrk" in data:
            runs.append(data)

    return runs


def throughput_boxplot(wrk_directory: Path, output: Path):
    """
    Throughput distribution across all wrk runs.
    """

    runs = _load_runs(wrk_directory)

    values = [
        run["wrk"]["requests_per_sec"]
        for run in runs
    ]

    if not values:
        return

    fig, ax = plt.subplots(figsize=(6, 5))

    ax.boxplot(
        values,
        patch_artist=True,
        showmeans=True,
    )

    ax.set_title("wrk Throughput")
    ax.set_ylabel("Requests / second")
    ax.grid(axis="y", alpha=0.3)
    _annotate_boxplot(ax, values)

    _save(fig, output)


def latency_boxplot(wrk_directory: Path, output: Path):
    """
    Mean latency distribution across all wrk runs.
    """

    runs = _load_runs(wrk_directory)

    values = [
        run["wrk"]["latency_avg_ms"]
        for run in runs
    ]

    if not values:
        return

    fig, ax = plt.subplots(figsize=(6, 5))

    ax.boxplot(
        values,
        patch_artist=True,
        showmeans=True,
    )

    ax.set_title("wrk Mean Latency")
    ax.set_ylabel("Latency (ms)")
    ax.grid(axis="y", alpha=0.3)
    _annotate_boxplot(ax, values, " ms")

    _save(fig, output)


def transfer_boxplot(wrk_directory: Path, output: Path):
    """
    Transfer/sec distribution across all wrk runs.
    """

    runs = _load_runs(wrk_directory)

    values = [
        run["wrk"]["transfer_mb_per_sec"]
        for run in runs
    ]

    if not values:
        return

    fig, ax = plt.subplots(figsize=(6, 5))

    ax.boxplot(
        values,
        patch_artist=True,
        showmeans=True,
    )

    ax.set_title("wrk Transfer Rate")
    ax.set_ylabel("Transfer (MB/s)")
    ax.grid(axis="y", alpha=0.3)
    _annotate_boxplot(ax, values, " MB/s")

    _save(fig, output)


def throughput_errorbar(
    summary: dict,
    output: Path,
):
    """
    Mean throughput across benchmark runs with ±1 standard deviation.
    """

    stats = summary["wrk"]["requests_per_sec"]

    mean = stats["mean"]
    std = stats["stddev"]

    fig, ax = plt.subplots(figsize=(6, 5))

    ax.errorbar(
        x=[1],
        y=[mean],
        yerr=[std],
        fmt="o",
        markersize=8,
        capsize=8,
        linewidth=2,
    )

    ax.annotate(
    f"{mean:.0f}",
    (1, mean),
    xytext=(8, 0),
    textcoords="offset points",
    va="center",
    )

    ax.annotate(
    f"+{std:.0f}",
    (1, mean + std),
    xytext=(8, 0),
    textcoords="offset points",
    fontsize=8,
    )
    ax.annotate(
    f"-{std:.0f}",
    (1, mean - std),
    xytext=(8, 0),
    textcoords="offset points",
    fontsize=8,
    )

    ax.set_xlim(0.5, 1.5)
    ax.set_xticks([1])
    ax.set_xticklabels(["wrk"])

    ax.set_ylabel("Requests / second")
    ax.set_title("wrk Throughput (Mean ± Std Dev)")

    ax.grid(axis="y", alpha=0.3)

    # Give a sensible margin instead of starting from zero.
    lower = max(0, mean - std * 3)
    upper = mean + std * 3

    ax.set_ylim(lower, upper)

    _save(fig, output)


def generate(summary: dict, dataset_directory: Path):
    """
    Generate every wrk plot.
    """

    wrk_directory = dataset_directory / "raw" / "wrk"

    plots_directory = dataset_directory / "plots"

    throughput_boxplot(
        wrk_directory,
        plots_directory / "wrk-throughput-boxplot.png",
    )

    throughput_errorbar(
        summary,
        plots_directory / "wrk-throughput-errorbar.png",
    )

    latency_boxplot(
        wrk_directory,
        plots_directory / "wrk-latency-boxplot.png",
    )

    transfer_boxplot(
        wrk_directory,
        plots_directory / "wrk-transfer-boxplot.png",
    )
