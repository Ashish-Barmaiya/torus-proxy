#!/usr/bin/env bash

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${SCRIPT_DIR}/common.sh"

usage() {
    echo "Usage:"
    echo "  collect.sh <scenario> <dataset-directory>"
    exit 1
}

[[ $# -eq 2 ]] || usage

SCENARIO="$1"
DATASET_DIR="$2"

require_scenario "${SCENARIO}"
require_directory "${DATASET_DIR}"

log_info "Collecting benchmark metadata..."

cat > "${DATASET_DIR}/metadata.json" <<EOF
{
  "benchmark": {
    "id": "$(benchmark_id "${SCENARIO}")",
    "title": "$(benchmark_title "${SCENARIO}")",
    "protocol": "$(benchmark_protocol "${SCENARIO}")",
    "url": "$(benchmark_url "${SCENARIO}")",
    "threads": $(benchmark_threads "${SCENARIO}"),
    "connections": $(benchmark_connections "${SCENARIO}"),
    "duration": "$(benchmark_duration "${SCENARIO}")",
    "warmup": "$(benchmark_warmup "${SCENARIO}")",
    "iterations": $(benchmark_iterations "${SCENARIO}")
  },

  "environment": {
    "hardware_profile": "$(hardware_profile "${SCENARIO}")",
    "environment_profile": "$(environment_profile "${SCENARIO}")",
    "software_profile": "$(software_profile "${SCENARIO}")",
    "methodology_version": "$(methodology_version "${SCENARIO}")"
  },

  "system": {
    "timestamp": "$(timestamp)",
    "hostname": "$(hostname_name)",
    "os": "$(os_name)",
    "kernel": "$(kernel_version)",
    "cpu": "$(cpu_model)",
    "logical_cpus": $(logical_cpus),
    "memory": "$(memory_total)"
  },

  "git": {
    "branch": "$(git_branch)",
    "commit": "$(git_commit)",
    "dirty": $(git_dirty && echo true || echo false)
  },

  "tools": {
    "go": "$(tool_version go)",
    "wrk": "$(tool_version wrk)",
    "wrk2": "$(tool_version wrk2)",
    "perf": "$(tool_version perf)",
    "pidstat": "$(tool_version pidstat)",
    "vmstat": "$(tool_version vmstat)",
    "jq": "$(tool_version jq)",
    "yq": "$(tool_version yq)"
  }
}
EOF

log_success "Metadata written."

echo
echo "Metadata:"
echo "  ${DATASET_DIR}/metadata.json"
