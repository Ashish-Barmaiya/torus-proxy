#!/usr/bin/env python3

"""
Plots for wrk benchmark results.
"""

from __future__ import annotations

import json
from pathlib import Path

import matplotlib.pyplot as plt


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

    _save(fig, output)


def throughput_errorbar(summary: dict, output: Path):
    """
    Mean throughput with one standard deviation.
    """

    stats = summary["wrk"]["requests_per_sec"]

    fig, ax = plt.subplots(figsize=(5, 5))

    ax.bar(
        ["wrk"],
        [stats["mean"]],
        yerr=[stats["stddev"]],
        capsize=8,
    )

    ax.set_title("wrk Throughput")
    ax.set_ylabel("Requests / second")
    ax.grid(axis="y", alpha=0.3)

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
