#!/usr/bin/env bash

# Torus Benchmark Framework
# Common Utility Functions
# Version : 1.0

set -Eeuo pipefail

# Colors
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly BOLD='\033[1m'
readonly RESET='\033[0m'

# Logging
log_info() {
    echo -e "${BLUE}[INFO]${RESET} $*"
}

log_success() {
    echo -e "${GREEN}[ OK ]${RESET} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${RESET} $*"
}

log_error() {
    echo -e "${RED}[FAIL]${RESET} $*" >&2
}

die() {
    log_error "$*"
    exit 1
}

divider() {
    printf '%*s\n' "${COLUMNS:-80}" '' | tr ' ' '='
}

print_header() {
    divider
    echo " Torus Benchmark Framework"
    divider
    echo
}

# Repository Helpers
repo_root() {
    git rev-parse --show-toplevel
}

benchmark_root() {
    echo "$(repo_root)/docs/benchmarking"
}

scenario_dir() {
    echo "$(benchmark_root)/scenarios"
}

dataset_dir() {
    echo "$(benchmark_root)/datasets"
}

script_dir() {
    echo "$(benchmark_root)/scripts"
}

analysis_dir() {
    echo "$(benchmark_root)/analysis"
}

report_dir() {
    echo "$(benchmark_root)/reports"
}

template_dir() {
    echo "$(benchmark_root)/templates"
}

# Benchmark Helpers
benchmark_dataset_dir() {
    local benchmark="$1"

    echo "$(dataset_dir)/${benchmark}"
}

raw_dir() {
    local benchmark="$1"

    echo "$(benchmark_dataset_dir "$benchmark")/raw"
}

plots_dir() {
    local benchmark="$1"

    echo "$(benchmark_dataset_dir "$benchmark")/plots"
}

run_dir() {
    local benchmark="$1"
    local run="$2"

    printf "%s/run-%03d\n" "$(raw_dir "$benchmark")" "$run"
}

# Scenario Helpers
scenario_file() {
    echo "$(scenario_dir)/$1.yaml"
}

require_scenario() {
    local scenario

    scenario="$(scenario_file "$1")"

    [[ -f "$scenario" ]] || die "Scenario '$1' does not exist."
}

scenario_value() {
    local scenario="$1"
    local key="$2"

    yq -r "$key" "$(scenario_file "$scenario")"

}

# Command Helpers
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

require_command() {
    local command="$1"

    if command_exists "$command"; then
        log_success "$command"
    else
        die "Required command not found: $command"
    fi
}

# Filesystem Helpers
require_directory() {
    [[ -d "$1" ]] || die "Directory not found: $1"
}

require_file() {
    [[ -f "$1" ]] || die "File not found: $1"
}

ensure_directory() {
    mkdir -p "$1"
}

# Git Helpers
require_git_repo() {
    git rev-parse --is-inside-work-tree >/dev/null 2>&1 ||
        die "Not inside a Git repository."
}

git_commit() {
    git rev-parse HEAD
}

git_branch() {
    git rev-parse --abbrev-ref HEAD
}

git_dirty() {
    [[ -n "$(git status --porcelain)" ]]
}

# Version Helpers
tool_version() {
    case "$1" in

        go)
            go version
            ;;

        python3)
            python3 --version
            ;;

        git)
            git --version
            ;;

        jq)
            jq --version
            ;;

        yq)
            yq --version
            ;;

        wrk)
            wrk --version 2>&1 | head -n 1
            ;;

        vegeta)
            vegeta version 2>&1 | head -n 1
            ;;

        perf)
            perf --version
            ;;

        pidstat)
            pidstat -V
            ;;

        vmstat)
            vmstat -V
            ;;

        ss)
            ss --version 2>/dev/null || echo "Unknown"
            ;;

        *)
            echo "Unknown"
            ;;

    esac
}

# System Information
os_name() {
    source /etc/os-release
    echo "$PRETTY_NAME"
}

kernel_version() {
    uname -r
}

hostname_name() {
    hostname
}

cpu_model() {
    lscpu | awk -F: '/Model name/ {
        gsub(/^[ \t]+/, "", $2)
        print $2
    }'
}

logical_cpus() {
    nproc
}

memory_total() {
    free -h | awk '/Mem:/ {print $2}'
}

disk_available() {
    df -h . | awk 'NR==2 {print $4}'
}

# Timestamp Helpers
timestamp() {
    date +"%Y-%m-%dT%H:%M:%S%z"
}

date_stamp() {
    date +"%Y-%m-%d"
}

time_stamp() {
    date +"%H:%M:%S"
}

# Benchmark Metadata Helpers
benchmark_id() {
    scenario_value "$1" ".id"
}

benchmark_title() {
    scenario_value "$1" ".title"
}

benchmark_protocol() {
    scenario_value "$1" ".benchmark.protocol"
}

benchmark_tools() {
    scenario_value "$1" ".benchmark.tools[]"
}

benchmark_iterations() {
    scenario_value "$1" ".workload.iterations"
}

benchmark_duration() {
    scenario_value "$1" ".workload.duration"
}

benchmark_warmup() {
    scenario_value "$1" ".workload.warmup"
}

benchmark_threads() {
    scenario_value "$1" ".workload.threads"
}

benchmark_connections() {
    scenario_value "$1" ".workload.connections"
}

benchmark_url() {
    scenario_value "$1" ".benchmark.url"
}

hardware_profile() {
    scenario_value "$1" ".environment.hardware_profile"
}

environment_profile() {
    scenario_value "$1" ".environment.environment_profile"
}

software_profile() {
    scenario_value "$1" ".environment.software_profile"
}

methodology_version() {
    scenario_value "$1" ".environment.methodology_version"
}

benchmark_rate() {
    scenario_value "$1" ".workload.rate"
}

# Validation Helpers
validate_repository_structure() {

    require_directory "$(benchmark_root)"
    require_directory "$(scenario_dir)"
    require_directory "$(script_dir)"
    require_directory "$(analysis_dir)"
    require_directory "$(dataset_dir)"
    require_directory "$(report_dir)"
    require_directory "$(template_dir)"
    require_directory "$(benchmark_root)/profiles"

}
