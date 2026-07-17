#!/usr/bin/env python3

import argparse
import json
import re
from pathlib import Path


def parse_wrk(path: Path) -> dict:
    """
    Parse a wrk output file.
    """

    text = path.read_text()

    result = {}

    patterns = {
        "latency_avg_ms": r"Latency\s+([\d.]+)(ms|us|s)",
        "requests_per_sec": r"Requests/sec:\s+([\d.]+)",
        "transfer_mb_per_sec": r"Transfer/sec:\s+([\d.]+)(KB|MB|GB)",
        "total_requests": r"(\d+)\s+requests in",
        "socket_errors": r"Socket errors: connect (\d+), read (\d+), write (\d+), timeout (\d+)",
        "non_success_responses": r"Non-2xx or 3xx responses:\s+(\d+)",
    }

    for key, pattern in patterns.items():
        match = re.search(pattern, text)

        if not match:
            continue

        if key == "socket_errors":
            result[key] = {
                "connect": int(match.group(1)),
                "read": int(match.group(2)),
                "write": int(match.group(3)),
                "timeout": int(match.group(4)),
            }

        elif key == "transfer_mb_per_sec":
            value = float(match.group(1))
            unit = match.group(2)

            if unit == "KB":
                value /= 1024
            elif unit == "GB":
                value *= 1024

            result[key] = value

        elif len(match.groups()) == 1:
            try:
                result[key] = float(match.group(1))
            except ValueError:
                result[key] = match.group(1)

        else:
            value = float(match.group(1))
            unit = match.group(2)

            if unit == "us":
                value /= 1000
            elif unit == "s":
                value *= 1000

            result[key] = value

    return result


def parse_vegeta(path: Path) -> dict:
    """
    Parse a Vegeta JSON report.

    All latency values are normalized to milliseconds.
    """

    data = json.loads(path.read_text())

    latencies = data["latencies"]

    return {

        # Request statistics

        "requests": data["requests"],
        "rate": data["rate"],
        "throughput": data["throughput"],
        "success_ratio": data["success"],

        # Latency statistics (ms)

        "latency_mean_ms": latencies["mean"] / 1_000_000,
        "latency_p50_ms": latencies["50th"] / 1_000_000,
        "latency_p95_ms": latencies["95th"] / 1_000_000,
        "latency_p99_ms": latencies["99th"] / 1_000_000,
        "latency_max_ms": latencies["max"] / 1_000_000,

        # Network statistics

        "bytes_in_total": data["bytes_in"]["total"],
        "bytes_in_mean": data["bytes_in"]["mean"],

        "bytes_out_total": data["bytes_out"]["total"],
        "bytes_out_mean": data["bytes_out"]["mean"],

        # Response statistics

        "status_codes": data.get("status_codes", {}),
        "errors": data.get("errors", []),
    }


def parse_pidstat(path: Path):
    rows = []

    for line in path.read_text().splitlines():

        if not line.strip():
            continue

        if line.startswith("#"):
            continue

        if line.startswith("Linux"):
            continue

        if line.startswith("Average"):
            continue

        cols = line.split()

        if len(cols) < 17:
            continue

        try:
            rows.append(
                {
                    "time": f"{cols[0]} {cols[1]}",
                    "pid": int(cols[3]),
                    "cpu": float(cols[8]),
                    "rss": int(cols[13]),
                    "mem": float(cols[14]),
                    "command": cols[-1],
                }
            )
        except Exception:
            pass

    return rows


def parse_vmstat(path: Path):
    rows = []

    for line in path.read_text().splitlines():

        if (
            not line.strip()
            or line.startswith("procs")
            or line.startswith("r ")
        ):
            continue

        cols = line.split()

        if len(cols) != 17:
            continue

        rows.append(
            {
                "r": int(cols[0]),
                "b": int(cols[1]),
                "free": int(cols[3]),
                "cache": int(cols[5]),
                "bi": int(cols[8]),
                "bo": int(cols[9]),
                "interrupts": int(cols[10]),
                "context_switches": int(cols[11]),
                "cpu_user": int(cols[12]),
                "cpu_system": int(cols[13]),
                "cpu_idle": int(cols[14]),
                "cpu_wait": int(cols[15]),
            }
        )

    return rows


def main():

    parser = argparse.ArgumentParser()

    parser.add_argument("--run-dir", required=True)

    args = parser.parse_args()

    run = Path(args.run_dir)

    output = {}

    if (run / "wrk.txt").exists():
        output["wrk"] = parse_wrk(run / "wrk.txt")

    if (run / "vegeta.json").exists():
        output["vegeta"] = parse_vegeta(run / "vegeta.json")

    if (run / "pidstat.txt").exists():
        output["pidstat"] = parse_pidstat(run / "pidstat.txt")

    if (run / "vmstat.txt").exists():
        output["vmstat"] = parse_vmstat(run / "vmstat.txt")

    with open(run / "parsed.json", "w") as f:
        json.dump(output, f, indent=2)


if __name__ == "__main__":
    main()
