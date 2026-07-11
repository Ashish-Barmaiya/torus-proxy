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

log_info "Running statistical analysis..."

python3 "$(analysis_dir)/analyze.py" "${DATASET}"

log_success "Analysis complete."
