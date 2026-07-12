#!/usr/bin/env python3

"""
Plot generation for benchmark reports.
"""

from __future__ import annotations

from pathlib import Path

import matplotlib.pyplot as plt


def _save(fig, output: Path):
    output.parent.mkdir(parents=True, exist_ok=True)
    fig.tight_layout()
    fig.savefig(output, dpi=200)
    plt.close(fig)


# Throughput

def throughput_plot(summary: dict, output: Path):

    fig, ax = plt.subplots(figsize=(8, 4))

    tools = []
    values = []

    if "wrk" in summary:
        tools.append("wrk")
        values.append(summary["wrk"]["requests_per_sec"]["mean"])

    if "vegeta" in summary:
        tools.append("vegeta")
        values.append(summary["vegeta"]["rate"]["mean"])

    ax.bar(tools, values)

    ax.set_title("Average Throughput")
    ax.set_ylabel("Requests / second")

    _save(fig, output)


# Latency

def latency_plot(summary: dict, output: Path):

    fig, ax = plt.subplots(figsize=(8, 4))

    labels = []
    values = []

    if "wrk" in summary:
        labels.append("wrk")
        values.append(summary["wrk"]["latency_avg_ms"]["mean"])

    if "vegeta" in summary:
        labels.append("vegeta")
        values.append(summary["vegeta"]["latency_mean_ms"]["mean"])

    ax.bar(labels, values)

    ax.set_title("Average Latency")
    ax.set_ylabel("Milliseconds")

    _save(fig, output)


# CPU

def cpu_plot(cpu_samples: list[float], output: Path):

    fig, ax = plt.subplots(figsize=(10, 4))

    ax.plot(range(len(cpu_samples)), cpu_samples)

    ax.set_title("CPU Usage")
    ax.set_xlabel("Sample")
    ax.set_ylabel("CPU %")

    _save(fig, output)


# Memory

def memory_plot(memory_samples: list[float], output: Path):

    fig, ax = plt.subplots(figsize=(10, 4))

    ax.plot(range(len(memory_samples)), memory_samples)

    ax.set_title("Memory Usage")
    ax.set_xlabel("Sample")
    ax.set_ylabel("RSS (KB)")

    _save(fig, output)


# Latency Distribution

def latency_distribution(samples: list[float], output: Path):

    fig, ax = plt.subplots(figsize=(8, 4))

    ax.hist(samples, bins=25)

    ax.set_title("Latency Distribution")
    ax.set_xlabel("Milliseconds")
    ax.set_ylabel("Frequency")

    _save(fig, output)
