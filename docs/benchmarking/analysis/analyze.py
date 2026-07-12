#!/usr/bin/env python3

"""
Analyze a benchmark dataset.

Pipeline:

1. Parse raw benchmark outputs.
2. Compute statistical summaries.
3. Generate plots.
4. Write summary.json.
"""

from __future__ import annotations

import argparse
import json
import subprocess
from pathlib import Path

import metrics
import system_plots
import vegeta_plots
import wrk_plots


# Parsing

def parse_runs(dataset_directory: Path):
    """
    Parse every benchmark run into parsed.json.
    """

    parser = Path(__file__).parent / "parser.py"

    raw_directory = dataset_directory / "raw"

    if not raw_directory.exists():
        return

    for tool_directory in sorted(raw_directory.iterdir()):

        if not tool_directory.is_dir():
            continue

        for run_directory in sorted(tool_directory.glob("run-*")):

            subprocess.run(
                [
                    "python3",
                    str(parser),
                    "--run-dir",
                    str(run_directory),
                ],
                check=True,
            )


# Plot Discovery

def collect_plot_paths(dataset_directory: Path):
    """
    Discover every generated plot automatically.

    The returned dictionary mirrors the directory structure
    under plots/.
    """

    plots = {}

    plots_directory = dataset_directory / "plots"

    if not plots_directory.exists():
        return plots

    for image in sorted(plots_directory.rglob("*.png")):

        relative = image.relative_to(plots_directory)

        current = plots

        parts = list(relative.parts)

        for part in parts[:-1]:

            key = part.replace("-", "_")

            current = current.setdefault(key, {})

        current[parts[-1].replace(".png", "")] = str(relative)

    return plots


# Summary

def build_summary(dataset_directory: Path):
    """
    Build benchmark summary from parsed runs.
    """

    metadata_file = dataset_directory / "metadata.json"

    with open(metadata_file) as f:
        metadata = json.load(f)

    summary = {

        "metadata": {
            "id": metadata["benchmark"]["id"],
            "title": metadata["benchmark"]["title"],
            "timestamp": metadata["system"]["timestamp"],
            "hostname": metadata["system"]["hostname"],
            "git_branch": metadata["git"]["branch"],
            "git_commit": metadata["git"]["commit"],
        },

        "workload": {
            "protocol": metadata["benchmark"]["protocol"],
            "url": metadata["benchmark"]["url"],
            "threads": metadata["benchmark"]["threads"],
            "connections": metadata["benchmark"]["connections"],
            "duration": metadata["benchmark"]["duration"],
            "warmup": metadata["benchmark"]["warmup"],
            "iterations": metadata["benchmark"]["iterations"],
        },

        "environment": {
            "cpu": metadata["system"]["cpu"],
            "logical_cpus": metadata["system"]["logical_cpus"],
            "memory": metadata["system"]["memory"],
            "os": metadata["system"]["os"],
            "kernel": metadata["system"]["kernel"],
        },

        "wrk": {},
        "vegeta": {},
        "plots": {},
    }

    # wrk

    wrk_directory = dataset_directory / "raw" / "wrk"

    if wrk_directory.exists():

        runs = []

        for parsed in sorted(
            wrk_directory.glob("run-*/parsed.json")
        ):

            with open(parsed) as f:
                runs.append(json.load(f))

        if runs:
            summary["wrk"] = metrics.summarize_wrk(runs)


    # vegeta

    vegeta_directory = dataset_directory / "raw" / "vegeta"

    if vegeta_directory.exists():

        runs = []

        for parsed in sorted(
            vegeta_directory.glob("run-*/parsed.json")
        ):

            with open(parsed) as f:
                runs.append(json.load(f))

        if runs:
            summary["vegeta"] = metrics.summarize_vegeta(runs)

    return summary


# Plot Generation

def generate_plots(
    summary: dict,
    dataset_directory: Path,
):
    """
    Generate every benchmark plot.
    """

    if summary.get("wrk"):

        wrk_plots.generate(
            summary,
            dataset_directory,
        )

    if summary.get("vegeta"):

        vegeta_plots.generate(
            summary,
            dataset_directory,
        )

    system_plots.generate(
        dataset_directory,
    )


# Summary Output

def save_summary(
    summary: dict,
    dataset_directory: Path,
):
    """
    Write summary.json.
    """

    # Plots now exist, so discover them.
    summary["plots"] = collect_plot_paths(
        dataset_directory,
    )

    with open(
        dataset_directory / "summary.json",
        "w",
    ) as f:

        json.dump(
            summary,
            f,
            indent=4,
        )


# Analysis Pipeline

def analyze(
    dataset_directory: Path,
):
    """
    Complete benchmark analysis pipeline.
    """

    print(
        "[INFO] Parsing benchmark runs..."
    )

    parse_runs(
        dataset_directory,
    )

    print(
        "[INFO] Computing statistics..."
    )

    summary = build_summary(
        dataset_directory,
    )

    print(
        "[INFO] Generating plots..."
    )

    generate_plots(
        summary,
        dataset_directory,
    )

    print(
        "[INFO] Writing summary..."
    )

    save_summary(
        summary,
        dataset_directory,
    )

    print(
        "[ OK ] Analysis complete."
    )

# CLI

def main():

    parser = argparse.ArgumentParser(
        description="Analyze a benchmark dataset."
    )

    parser.add_argument(
        "dataset",
        type=Path,
        help="Path to benchmark dataset directory.",
    )

    args = parser.parse_args()

    dataset_directory = args.dataset.resolve()

    if not dataset_directory.exists():

        parser.error(
            f"Dataset does not exist: {dataset_directory}"
        )

    analyze(
        dataset_directory,
    )


if __name__ == "__main__":
    main()
