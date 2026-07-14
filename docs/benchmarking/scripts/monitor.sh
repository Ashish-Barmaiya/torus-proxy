#!/usr/bin/env bash

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${SCRIPT_DIR}/common.sh"

usage() {
    echo "Usage:"
    echo "  monitor.sh start <run-directory>"
    echo "  monitor.sh stop"
    exit 1
}

[[ $# -ge 1 ]] || usage

ACTION="$1"

PID_FILE="/tmp/torus-benchmark-monitor.pids"

start_monitors() {

    [[ $# -eq 1 ]] || die "Run directory required."

    local RUN_DIR="$1"

    ensure_directory "${RUN_DIR}"

    : > "${PID_FILE}"

    log_info "Starting monitoring..."

    pidstat -durh 1 > "${RUN_DIR}/pidstat.txt" &
    echo $! >> "${PID_FILE}"

    vmstat 1 > "${RUN_DIR}/vmstat.txt" &
    echo $! >> "${PID_FILE}"

    ss -tanp > "${RUN_DIR}/ss.txt" &
    echo $! >> "${PID_FILE}"

    perf stat \
        -o "${RUN_DIR}/perf.txt" \
        sleep 999999 &
    echo $! >> "${PID_FILE}"

    log_success "Monitoring started."
}

stop_monitors() {

    [[ -f "${PID_FILE}" ]] || {
        log_warning "No running monitors."
        return
    }

    log_info "Stopping monitoring..."

    while read -r pid; do
        kill -INT "${pid}" 2>/dev/null || true
        wait "${pid}" 2>/dev/null || true
    done < "${PID_FILE}"

    rm -f "${PID_FILE}"

    log_success "Monitoring stopped."
}

case "${ACTION}" in

    start)

        [[ $# -eq 2 ]] || usage
        start_monitors "$2"
        ;;

    stop)

        stop_monitors
        ;;

    *)

        usage
        ;;

esac
