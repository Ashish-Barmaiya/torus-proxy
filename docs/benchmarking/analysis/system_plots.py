#!/usr/bin/env python3

"""
System resource plots for benchmark reports.

This module generates plots from pidstat and vmstat collected during
benchmark execution.

Architecture:
    dataset/
        raw/
            wrk/
                run-001/
                    parsed.json
                ...
            vegeta/
                run-001/
                    parsed.json
                ...

The module discovers runs automatically.
"""

from __future__ import annotations

import json
import statistics
from pathlib import Path

import matplotlib.pyplot as plt


# Utilities

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
    """
    Save figure.
    """

    output.parent.mkdir(parents=True, exist_ok=True)

    fig.tight_layout()

    fig.savefig(
        output,
        dpi=200,
    )

    plt.close(fig)


def _load_runs(tool_directory: Path):
    """
    Load parsed.json from every benchmark run.
    """

    runs = []

    for run in sorted(tool_directory.glob("run-*")):

        parsed = run / "parsed.json"

        if not parsed.exists():
            continue

        with open(parsed) as f:
            runs.append(json.load(f))

    return runs


def _pad(series: list[float], length: int):
    """
    Pad a time series using the final value.

    This lets shorter runs align with longer runs.
    """

    if not series:
        return [0.0] * length

    if len(series) >= length:
        return series[:length]

    return series + [series[-1]] * (length - len(series))


def _aggregate(series_list: list[list[float]]):
    """
    Aggregate multiple time series.

    Returns:
        mean
        minimum
        maximum
    """

    if not series_list:
        return [], [], []

    max_len = max(len(s) for s in series_list)

    padded = [
        _pad(s, max_len)
        for s in series_list
    ]

    means = []
    mins = []
    maxs = []

    for i in range(max_len):

        values = [
            run[i]
            for run in padded
        ]

        means.append(
            statistics.mean(values)
        )

        mins.append(
            min(values)
        )

        maxs.append(
            max(values)
        )

    return means, mins, maxs


def _plot_band(
    mean,
    minimum,
    maximum,
    title,
    ylabel,
    output,
):
    """
    Plot the mean time series.

    The minimum/maximum envelope is intentionally omitted because it
    becomes visually noisy with a small number of benchmark runs.
    """

    if not mean:
        return

    fig, ax = plt.subplots(
        figsize=(10, 5)
    )

    x = list(
        range(len(mean))
    )

    ax.plot(
    x,
    mean,
    linewidth=2,
    )
    peak = max(mean)
    peak_idx = mean.index(peak)

    low = min(mean)
    low_idx = mean.index(low)

    ax.scatter([peak_idx], [peak], s=35)
    ax.scatter([low_idx], [low], s=35)

    ax.annotate(
    f"Max {peak:.1f}",
    (peak_idx, peak),
    xytext=(0, 10),
    textcoords="offset points",
    ha="center",
    fontsize=8,
    )

    ax.annotate(
    f"Min {low:.1f}",
    (low_idx, low),
    xytext=(0, -15),
    textcoords="offset points",
    ha="center",
    fontsize=8,
    )

    ax.set_title(title)

    ax.set_xlabel("Sample")

    ax.set_ylabel(ylabel)

    ax.grid(
        axis="y",
        alpha=0.30,
    )

    _save(
        fig,
        output,
    )


def _boxplot(
    values,
    title,
    ylabel,
    output,
    unit="",
):
    """
    Generic box plot.
    """

    if not values:
        return

    fig, ax = plt.subplots(
        figsize=(6, 5)
    )

    ax.boxplot(
        values,
        showmeans=True,
        patch_artist=True,
    )

    ax.set_title(title)

    ax.set_ylabel(
        ylabel
    )

    ax.grid(
        axis="y",
        alpha=0.30,
    )

    _annotate_boxplot(ax, values, unit)

    _save(
        fig,
        output,
    )



# CPU

def cpu_timeseries(
    tool_directory: Path,
    output: Path,
):
    """
    CPU utilisation over time.

    Mean with min/max envelope.
    """

    runs = _load_runs(
        tool_directory
    )

    cpu = []

    for run in runs:

        samples = [
            sample["cpu"]
            for sample in run.get(
                "pidstat",
                [],
            )
        ]

        if samples:
            cpu.append(
                samples
            )

    if not cpu:
        return

    mean, minimum, maximum = _aggregate(
        cpu
    )

    _plot_band(
        mean,
        minimum,
        maximum,
        "CPU Utilisation Over Time",
        "CPU %",
        output,
    )


# CPU Distribution

def cpu_distribution(
    tool_directory: Path,
    output: Path,
):
    """
    CPU utilisation distribution across all samples.
    """

    runs = _load_runs(tool_directory)

    values = []

    for run in runs:

        for sample in run.get("pidstat", []):

            values.append(
                sample["cpu"]
            )

    _boxplot(
        values,
        "CPU Utilisation Distribution",
        "CPU %",
        output,
        "%"
    )


# Memory

def memory_timeseries(
    tool_directory: Path,
    output: Path,
):
    """
    RSS memory over time.
    """

    runs = _load_runs(
        tool_directory
    )

    rss = []

    for run in runs:

        samples = [
            sample["rss"] /1024
            for sample in run.get(
                "pidstat",
                [],
            )
        ]

        if samples:
            rss.append(
                samples
            )

    if not rss:
        return

    mean, minimum, maximum = _aggregate(
        rss
    )

    _plot_band(
        mean,
        minimum,
        maximum,
        "Resident Memory Over Time",
        "RSS (MB)",
        output,
    )


def memory_distribution(
    tool_directory: Path,
    output: Path,
):
    """
    RSS memory distribution.
    """

    runs = _load_runs(tool_directory)

    values = []

    for run in runs:

        for sample in run.get("pidstat", []):

            values.append(
                sample["rss"] /1024
            )

    _boxplot(
        values,
        "Resident Memory Distribution",
        "RSS (MB)",
        output,
        " MB"
    )


# VMStat Helpers

def _vmstat_series(
    tool_directory: Path,
    key: str,
):
    """
    Extract one vmstat metric from every run.

    Returns:
        list[list[float]]
    """

    runs = _load_runs(
        tool_directory
    )

    output = []

    for run in runs:

        samples = [
            sample[key]
            for sample in run.get(
                "vmstat",
                [],
            )
        ]

        if samples:
            output.append(
                samples
            )

    return output


# VMStat

def cpu_idle_timeseries(
    tool_directory: Path,
    output: Path,
):
    """
    CPU idle percentage over time.
    """

    values = _vmstat_series(
        tool_directory,
        "cpu_idle",
    )

    if not values:
        return

    mean, minimum, maximum = _aggregate(
        values
    )

    _plot_band(
        mean,
        minimum,
        maximum,
        "CPU Idle Time",
        "Idle %",
        output,
    )


def run_queue_timeseries(
    tool_directory: Path,
    output: Path,
):
    """
    Runnable processes over time.
    """

    values = _vmstat_series(
        tool_directory,
        "r",
    )

    if not values:
        return

    mean, minimum, maximum = _aggregate(
        values
    )

    _plot_band(
        mean,
        minimum,
        maximum,
        "Run Queue Length",
        "Processes",
        output,
    )


# Context Switches

def context_switch_timeseries(
    tool_directory: Path,
    output: Path,
):
    """
    Context switches over time.
    """

    values = _vmstat_series(
        tool_directory,
        "context_switches",
    )

    if not values:
        return

    mean, minimum, maximum = _aggregate(
        values
    )

    _plot_band(
        mean,
        minimum,
        maximum,
        "Context Switches",
        "Switches / second",
        output,
    )


# Interrupts

def interrupts_timeseries(
    tool_directory: Path,
    output: Path,
):
    """
    Hardware/software interrupts over time.
    """

    values = _vmstat_series(
        tool_directory,
        "interrupts",
    )

    if not values:
        return

    mean, minimum, maximum = _aggregate(
        values
    )

    _plot_band(
        mean,
        minimum,
        maximum,
        "Interrupt Rate",
        "Interrupts / second",
        output,
    )


# CPU User Time

def cpu_user_timeseries(
    tool_directory: Path,
    output: Path,
):
    """
    CPU user utilisation over time.
    """

    values = _vmstat_series(
        tool_directory,
        "cpu_user",
    )

    if not values:
        return

    mean, minimum, maximum = _aggregate(
        values
    )

    _plot_band(
        mean,
        minimum,
        maximum,
        "CPU User Time",
        "User %",
        output,
    )


# CPU System Time

def cpu_system_timeseries(
    tool_directory: Path,
    output: Path,
):
    """
    CPU system utilisation over time.
    """

    values = _vmstat_series(
        tool_directory,
        "cpu_system",
    )

    if not values:
        return

    mean, minimum, maximum = _aggregate(
        values
    )

    _plot_band(
        mean,
        minimum,
        maximum,
        "CPU System Time",
        "System %",
        output,
    )


# CPU Wait Time

def cpu_wait_timeseries(
    tool_directory: Path,
    output: Path,
):
    """
    CPU I/O wait over time.
    """

    values = _vmstat_series(
        tool_directory,
        "cpu_wait",
    )

    if not values:
        return

    mean, minimum, maximum = _aggregate(
        values
    )

    _plot_band(
        mean,
        minimum,
        maximum,
        "CPU I/O Wait",
        "Wait %",
        output,
    )


# Disk Read

def disk_read_timeseries(
    tool_directory: Path,
    output: Path,
):
    """
    Disk read throughput.
    """

    values = _vmstat_series(
        tool_directory,
        "bi",
    )

    if not values:
        return

    mean, minimum, maximum = _aggregate(
        values
    )

    _plot_band(
        mean,
        minimum,
        maximum,
        "Disk Read",
        "Blocks / second",
        output,
    )


# Disk Write

def disk_write_timeseries(
    tool_directory: Path,
    output: Path,
):
    """
    Disk write throughput.
    """

    values = _vmstat_series(
        tool_directory,
        "bo",
    )

    if not values:
        return

    mean, minimum, maximum = _aggregate(
        values
    )

    _plot_band(
        mean,
        minimum,
        maximum,
        "Disk Write",
        "Blocks / second",
        output,
    )


# Tool Generator

def _generate_for_tool(
    dataset_directory: Path,
    tool: str,
):
    """
    Generate every system plot for one benchmark tool.
    """

    tool_directory = (
        dataset_directory
        / "raw"
        / tool
    )

    if not tool_directory.exists():
        return

    plots_directory = (
        dataset_directory
        / "plots"
        / "system"
        / tool
    )

    plots_directory.mkdir(
        parents=True,
        exist_ok=True,
    )

    cpu_timeseries(
        tool_directory,
        plots_directory / "cpu-timeseries.png",
    )

    cpu_distribution(
        tool_directory,
        plots_directory / "cpu-distribution.png",
    )

    memory_timeseries(
        tool_directory,
        plots_directory / "memory-timeseries.png",
    )

    memory_distribution(
        tool_directory,
        plots_directory / "memory-distribution.png",
    )

    cpu_idle_timeseries(
        tool_directory,
        plots_directory / "cpu-idle-timeseries.png",
    )

    run_queue_timeseries(
        tool_directory,
        plots_directory / "run-queue-timeseries.png",
    )

    context_switch_timeseries(
        tool_directory,
        plots_directory / "context-switch-timeseries.png",
    )

    interrupts_timeseries(
        tool_directory,
        plots_directory / "interrupts-timeseries.png",
    )

    cpu_user_timeseries(
        tool_directory,
        plots_directory / "cpu-user-timeseries.png",
    )

    cpu_system_timeseries(
        tool_directory,
        plots_directory / "cpu-system-timeseries.png",
    )

    cpu_wait_timeseries(
        tool_directory,
        plots_directory / "cpu-wait-timeseries.png",
    )

    disk_read_timeseries(
        tool_directory,
        plots_directory / "disk-read-timeseries.png",
    )

    disk_write_timeseries(
        tool_directory,
        plots_directory / "disk-write-timeseries.png",
    )


# Public API

def generate(
    dataset_directory: Path,
):
    """
    Generate every system plot found in the benchmark dataset.

    The function automatically discovers benchmark tools under:

        dataset/raw/

    so future tools (oha, fortio, bombardier, etc.) require no
    changes to this file.
    """

    raw_directory = (
        dataset_directory
        / "raw"
    )

    if not raw_directory.exists():
        return

    for tool_directory in sorted(
        raw_directory.iterdir()
    ):

        if not tool_directory.is_dir():
            continue

        _generate_for_tool(
            dataset_directory,
            tool_directory.name,
        )
