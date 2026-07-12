#!/usr/bin/env bash

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${SCRIPT_DIR}/common.sh"

usage() {
    echo "Usage:"
    echo "  analyze.sh <dataset-directory>"
    exit 1
}

[[ $# -eq 1 ]] || usage

DATASET="$1"

require_directory "${DATASET}"

print_header

log_info "Running benchmark analysis..."

log_info "Computing statistics and generating plots..."

python3 \
    "$(analysis_dir)/analyze.py" \
    "${DATASET}"

log_info "Generating report..."

python3 \
    "$(analysis_dir)/report_generator.py" \
    "${DATASET}"

echo

log_success "Analysis complete."

echo
echo "Generated artifacts:"
echo "  ${DATASET}/summary.json"
echo "  ${DATASET}/report-auto.md"
echo "  ${DATASET}/plots/"
