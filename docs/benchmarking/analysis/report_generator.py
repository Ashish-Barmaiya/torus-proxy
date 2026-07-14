#!/usr/bin/env python3

"""
Generate an automatic benchmark report from summary.json.

The generated report is intended to be an engineering draft that
assists writing the final benchmark report.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path


# Markdown Writer

class Markdown:

    def __init__(self):
        self.lines: list[str] = []

    def write(self, text: str = ""):
        self.lines.append(text)

    def heading(self, title: str, level: int = 1):
        self.write(f'{"#" * level} {title}')
        self.write()

    def table(self, headers, rows):

        self.write("| " + " | ".join(headers) + " |")
        self.write("|" + "|".join(["---"] * len(headers)) + "|")

        for row in rows:
            self.write("| " + " | ".join(map(str, row)) + " |")

        self.write()

    def bullet(self, text: str):
        self.write(f"- {text}")

    def image(self, path: str, title: str | None = None):

        if title:
            self.write(f"**{title}**")
            self.write()

        self.write(f"![]({path})")
        self.write()

    def code(self, text: str):

        self.write("```")
        self.write(text.rstrip())
        self.write("```")
        self.write()

    def save(self, path: Path):

        path.write_text(
            "\n".join(self.lines)
        )


# Helpers

def load_summary(dataset_directory: Path):

    with open(dataset_directory / "summary.json") as f:
        return json.load(f)


def metric(value):

    if isinstance(value, float):
        return f"{value:.3f}"

    return str(value)


# Benchmark Information


def benchmark_information(md: Markdown, summary: dict):

    metadata = summary["metadata"]

    workload = summary["workload"]

    environment = summary["environment"]

    md.heading("Benchmark Information")

    md.table(
        ["Field", "Value"],
        [
            ["Benchmark ID", metadata["id"]],
            ["Title", metadata["title"]],
            ["Date", metadata["timestamp"]],
            ["Git Branch", metadata["git_branch"]],
            ["Git Commit", metadata["git_commit"]],
            ["Protocol", workload["protocol"]],
            ["URL", workload["url"]],
            ["Threads", workload["threads"]],
            ["Connections", workload["connections"]],
            ["Duration", workload["duration"]],
            ["Iterations", workload["iterations"]],
            ["CPU", environment["cpu"]],
            ["Logical CPUs", environment["logical_cpus"]],
            ["Memory", environment["memory"]],
            ["Operating System", environment["os"]],
            ["Kernel", environment["kernel"]],
        ],
    )


# Template Sections

def placeholder(
    md: Markdown,
    title: str,
    suggestion: str,
):

    md.heading(title)

    md.write("TODO")
    md.write()
    md.write(suggestion)
    md.write()

# Objective

def objective(md: Markdown, summary: dict):

    workload = summary["workload"]

    placeholder(
        md,
        "Objective",
        (
            f"Suggested objective:\n\n"
            f"Evaluate the {workload['protocol'].upper()} performance "
            f"of Torus using the configured workload."
        ),
    )


# Background

def background(md: Markdown):

    placeholder(
        md,
        "Background",
        (
            "Describe why this benchmark was performed.\n\n"
            "Include:\n"
            "- Engineering motivation\n"
            "- Related benchmark reports\n"
            "- Relevant implementation details\n"
            "- Expected practical impact"
        ),
    )


# Hypothesis

def hypothesis(md: Markdown):

    placeholder(
        md,
        "Hypothesis",
        (
            "State the engineering hypothesis before running the benchmark."
        ),
    )


# Experimental Variables

def experimental_variables(md: Markdown, summary: dict):

    workload = summary["workload"]

    md.heading("Experimental Variables")

    md.heading("Independent Variable", 2)

    md.write("TODO")
    md.write()

    md.heading("Dependent Variables", 2)

    for metric_name in [
        "Throughput",
        "Latency",
        "CPU utilisation",
        "Memory utilisation",
        "Context switches",
    ]:
        md.bullet(metric_name)

    md.write()

    md.heading("Controlled Variables", 2)

    controlled = [
        ("Protocol", workload["protocol"]),
        ("Threads", workload["threads"]),
        ("Connections", workload["connections"]),
        ("Duration", workload["duration"]),
        ("Iterations", workload["iterations"]),
    ]

    md.table(
        ["Variable", "Value"],
        controlled,
    )


# System Configuration

def system_configuration(md: Markdown, summary: dict):

    env = summary["environment"]

    md.heading("System Configuration")

    md.table(
        ["Component", "Value"],
        [
            ["CPU", env["cpu"]],
            ["Logical CPUs", env["logical_cpus"]],
            ["Memory", env["memory"]],
            ["Operating System", env["os"]],
            ["Kernel", env["kernel"]],
        ],
    )

    md.write(
        "Any deviation from the baseline profiles should be documented "
        "in the final report."
    )

    md.write()


# Benchmark Configuration

def benchmark_configuration(md: Markdown, summary: dict):

    workload = summary["workload"]

    md.heading("Benchmark Configuration")

    md.table(
        ["Parameter", "Value"],
        [
            ["Protocol", workload["protocol"]],
            ["Target URL", workload["url"]],
            ["Threads", workload["threads"]],
            ["Connections", workload["connections"]],
            ["Duration", workload["duration"]],
            ["Warm-up", workload["warmup"]],
            ["Iterations", workload["iterations"]],
        ],
    )


# Benchmark Procedure

def benchmark_procedure(md: Markdown):

    md.heading("Benchmark Procedure")

    steps = [
        "Start backend server.",
        "Start Torus.",
        "Wait for initialization.",
        "Execute warm-up traffic.",
        "Run configured benchmark iterations.",
        "Collect monitoring data.",
        "Generate statistics and plots.",
    ]

    for step in steps:
        md.bullet(step)

    md.write()

    md.write(
        "Exact benchmark commands are listed in Appendix A."
    )

    md.write()


# Statistical Summary

def _statistics_table(md: Markdown, title: str, stats: dict):

    md.heading(title, level=3)

    md.table(
        ["Statistic", "Value"],
        [
            ["Sample Size", metric(stats["count"])],
            ["Mean", metric(stats["mean"])],
            ["Median", metric(stats["median"])],
            ["Minimum", metric(stats["min"])],
            ["Maximum", metric(stats["max"])],
            ["Standard Deviation", metric(stats["stddev"])],
            [
                "Coefficient of Variation (%)",
                metric(stats["coefficient_of_variation"]),
            ],
            ["P50", metric(stats["p50"])],
            ["P90", metric(stats["p90"])],
            ["P95", metric(stats["p95"])],
            ["P99", metric(stats["p99"])],
        ],
    )


def statistical_summary(md: Markdown, summary: dict):

    md.heading("Statistical Summary")

    # wrk

    if summary["wrk"]:

        md.heading("wrk", level=2)

        _statistics_table(
            md,
            "Requests / Second",
            summary["wrk"]["requests_per_sec"],
        )

        _statistics_table(
            md,
            "Average Latency (ms)",
            summary["wrk"]["latency_avg_ms"],
        )

        _statistics_table(
            md,
            "Transfer Rate (MB/s)",
            summary["wrk"]["transfer_mb_per_sec"],
        )

    # vegeta

    if summary["vegeta"]:

        md.heading("vegeta", level=2)

        _statistics_table(
            md,
            "Request Rate",
            summary["vegeta"]["rate"],
        )

        _statistics_table(
            md,
            "Throughput",
            summary["vegeta"]["throughput"],
        )

        _statistics_table(
            md,
            "Success Ratio",
            summary["vegeta"]["success_ratio"],
        )

        _statistics_table(
            md,
            "Mean Latency (ms)",
            summary["vegeta"]["latency_mean_ms"],
        )

        _statistics_table(
            md,
            "P50 Latency (ms)",
            summary["vegeta"]["latency_p50_ms"],
        )

        _statistics_table(
            md,
            "P95 Latency (ms)",
            summary["vegeta"]["latency_p95_ms"],
        )

        _statistics_table(
            md,
            "P99 Latency (ms)",
            summary["vegeta"]["latency_p99_ms"],
        )

        _statistics_table(
            md,
            "Maximum Latency (ms)",
            summary["vegeta"]["latency_max_ms"],
        )

# Plot Helpers

def _render_plot_tree(
    md: Markdown,
    tree: dict,
    level: int = 2,
):
    """
    Recursively render the plot hierarchy stored in summary["plots"].
    """

    for key in sorted(tree):

        value = tree[key]

        if isinstance(value, dict):

            md.heading(
                key.replace("_", " ").title(),
                level=level,
            )

            _render_plot_tree(
                md,
                value,
                level + 1,
            )

        else:

            title = (
                Path(value)
                .stem
                .replace("-", " ")
                .title()
            )

            md.write(
                f"- **{title}** — `{value}`"
            )

    md.write()


# Primary Performance Metrics

def primary_metrics(
    md: Markdown,
    summary: dict,
):

    md.heading("Primary Performance Metrics")

    plots = summary.get("plots", {})

    # wrk

    wrk = plots.get("wrk", {})

    if wrk:

        md.heading("wrk", level=2)

        ordered = [
            "throughput_boxplot",
            "throughput_errorbar",
            "latency_boxplot",
            "transfer_boxplot",
        ]

        for name in ordered:

            if name not in wrk:
                continue

            md.bold(
                name.replace("_", " ").title()
            )

            md.write()

            md.image(
                wrk[name]
            )

            md.write()

    # vegeta

    vegeta = plots.get("vegeta", {})

    if vegeta:

        md.heading("vegeta", level=2)

        ordered = [
            "latency_boxplot",
            "latency_histogram",
            "latency_percentiles",
        ]

        for name in ordered:

            if name not in vegeta:
                continue

            md.bold(
                name.replace("_", " ").title()
            )

            md.write()

            md.image(
                vegeta[name]
            )

            md.write()


# Supporting Performance Metrics

def supporting_metrics(md: Markdown, summary: dict):

    md.heading(
        "Supporting Performance Metrics"
    )

    plots = summary.get(
        "plots",
        {},
    )

    system = plots.get(
        "system",
        {},
    )

    if not system:

        md.write(
            "No supporting system metrics available."
        )

        md.write()

        return

    for tool, tree in system.items():

        md.heading(
            tool,
            level=2,
        )

        _render_plot_tree(
            md,
            tree,
        )


# Threats to Validity

def threats_to_validity(md: Markdown):

    md.heading("Threats to Validity")

    md.bullet("Benchmark executed on a specific hardware configuration.")
    md.bullet("Background operating-system activity may influence results.")
    md.bullet("Localhost networking does not represent distributed deployments.")
    md.bullet("Thermal throttling may affect long-running benchmarks.")
    md.bullet("Interpret results within the benchmark scope.")

    md.write()


# Conclusion

def conclusion(md: Markdown):

    placeholder(
        md,
        "Conclusion",
        (
            "Summarize the benchmark outcome.\n\n"
            "Suggested discussion:\n"
            "- Was the hypothesis supported?\n"
            "- Key performance findings\n"
            "- Engineering implications"
        ),
    )


# Future Work

def future_work(md: Markdown):

    placeholder(
        md,
        "Future Work",
        (
            "Possible follow-up investigations:\n\n"
            "- Larger workloads\n"
            "- Multi-node deployment\n"
            "- Different hardware\n"
            "- Additional benchmark tools\n"
            "- Performance optimizations"
        ),
    )


# Reproducibility

def reproducibility(md: Markdown, summary: dict):

    md.heading("Reproducibility")

    metadata = summary["metadata"]
    workload = summary["workload"]

    md.table(
        ["Item", "Value"],
        [
            ["Benchmark ID", metadata["id"]],
            ["Git Commit", metadata["git_commit"]],
            ["Git Branch", metadata["git_branch"]],
            ["Protocol", workload["protocol"]],
            ["Threads", workload["threads"]],
            ["Connections", workload["connections"]],
            ["Duration", workload["duration"]],
            ["Iterations", workload["iterations"]],
        ],
    )


# Appendix A

def appendix_a(md: Markdown):

    md.heading("Appendix A — Benchmark Commands")

    md.write(
        "The benchmark was executed using the benchmark automation "
        "framework. Refer to the raw dataset for exact command outputs."
    )

    md.write()


# Appendix B

def appendix_b(md: Markdown):

    md.heading("Appendix B — Benchmark Data")

    md.code(
        """raw/
plots/
summary.json
metadata.json"""
    )


# Appendix C

def _appendix_plot_index(
    md: Markdown,
    node: dict,
    level: int = 2,
):
    """
    Recursively list every generated plot without embedding it.
    """

    for key, value in sorted(node.items()):

        title = key.replace(
            "_",
            " ",
        ).title()

        if isinstance(value, str):

            md.bullet(f"{title} — `{value}`")

        else:

            md.heading(
                title,
                level=level,
            )

            _appendix_plot_index(
                md,
                value,
                level + 1,
            )

def appendix_c(md: Markdown, summary: dict):

    md.heading("Appendix C — Visual Artifacts")

    plots = summary.get("plots", {})

    if not plots:
        md.write("No plots generated.")
        md.write()
        return

    md.write(
        "The following visual artifacts were generated during analysis."
    )
    md.write()

    _render_plot_tree(md, plots)


# Report Generation

def generate_report(dataset_directory: Path):

    summary = load_summary(dataset_directory)

    md = Markdown()

    benchmark_information(md, summary)

    objective(md, summary)

    background(md)

    hypothesis(md)

    experimental_variables(md, summary)

    system_configuration(md, summary)

    benchmark_configuration(md, summary)

    benchmark_procedure(md)

    statistical_summary(md, summary)

    primary_metrics(md, summary)

    supporting_metrics(md, summary)

    threats_to_validity(md)

    conclusion(md)

    future_work(md)

    reproducibility(md, summary)

    appendix_a(md)

    appendix_b(md)

    appendix_c(md, summary)

    md.save(
        dataset_directory / "report-auto.md"
    )


# CLI

def main():

    parser = argparse.ArgumentParser()

    parser.add_argument(
        "dataset",
        type=Path,
    )

    args = parser.parse_args()

    generate_report(
        args.dataset,
    )


if __name__ == "__main__":
    main()
