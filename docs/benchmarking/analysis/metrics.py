#!/usr/bin/env python3

"""
Statistical utilities for benchmark analysis.
"""

from __future__ import annotations

import statistics
from typing import Iterable


def mean(values: Iterable[float]) -> float:
    values = list(values)
    return statistics.mean(values) if values else 0.0


def median(values: Iterable[float]) -> float:
    values = list(values)
    return statistics.median(values) if values else 0.0


def minimum(values: Iterable[float]) -> float:
    values = list(values)
    return min(values) if values else 0.0


def maximum(values: Iterable[float]) -> float:
    values = list(values)
    return max(values) if values else 0.0


def stddev(values: Iterable[float]) -> float:
    values = list(values)

    if len(values) < 2:
        return 0.0

    return statistics.stdev(values)


def coefficient_of_variation(values: Iterable[float]) -> float:
    values = list(values)

    if not values:
        return 0.0

    avg = mean(values)

    if avg == 0:
        return 0.0

    return (stddev(values) / avg) * 100


def percentile(values: Iterable[float], p: float) -> float:
    """
    Linear interpolation percentile.

    p = 50, 95, 99 ...
    """

    values = sorted(values)

    if not values:
        return 0.0

    if len(values) == 1:
        return values[0]

    rank = (len(values) - 1) * (p / 100)

    lower = int(rank)
    upper = min(lower + 1, len(values) - 1)

    weight = rank - lower

    return values[lower] * (1 - weight) + values[upper] * weight


def summarize(values: Iterable[float]) -> dict:
    """
    Return all standard benchmark statistics.
    """

    values = list(values)

    return {
        "count": len(values),
        "mean": mean(values),
        "median": median(values),
        "min": minimum(values),
        "max": maximum(values),
        "stddev": stddev(values),
        "coefficient_of_variation": coefficient_of_variation(values),
        "p50": percentile(values, 50),
        "p90": percentile(values, 90),
        "p95": percentile(values, 95),
        "p99": percentile(values, 99),
    }


def summarize_wrk(runs: list[dict]) -> dict:
    """
    Summarize all wrk runs.
    """

    valid_runs = [run for run in runs if "wrk" in run]

    return {
        "requests_per_sec": summarize(
            run["wrk"]["requests_per_sec"]
            for run in valid_runs
        ),
        "latency_avg_ms": summarize(
            run["wrk"]["latency_avg_ms"]
            for run in valid_runs
        ),
        "transfer_mb_per_sec": summarize(
            run["wrk"]["transfer_mb_per_sec"]
            for run in valid_runs
        ),
    }


def summarize_vegeta(runs: list[dict]) -> dict:
    """
    Summarize all Vegeta runs.
    """

    valid_runs = [run for run in runs if "vegeta" in run]

    return {
        "rate": summarize(
            run["vegeta"]["rate"]
            for run in valid_runs
        ),

        "latency_mean_ms": summarize(
            run["vegeta"]["latency_mean_ms"]
            for run in valid_runs
        ),

        "latency_p50_ms": summarize(
            run["vegeta"]["latency_p50_ms"]
            for run in valid_runs
        ),

        "latency_p95_ms": summarize(
            run["vegeta"]["latency_p95_ms"]
            for run in valid_runs
        ),

        "latency_p99_ms": summarize(
            run["vegeta"]["latency_p99_ms"]
            for run in valid_runs
        ),

        "latency_max_ms": summarize(
            run["vegeta"]["latency_max_ms"]
            for run in valid_runs
        ),

        "success_ratio": summarize(
            run["vegeta"]["success_ratio"]
            for run in valid_runs
        ),

        "throughput": summarize(
            run["vegeta"]["throughput"]
            for run in valid_runs
        ),
    }

