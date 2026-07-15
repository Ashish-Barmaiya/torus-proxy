#!/usr/bin/env bash

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${SCRIPT_DIR}/common.sh"

usage() {
    echo "Usage:"
    echo "  benchmark.sh <scenario>"
    exit 1
}

[[ $# -eq 1 ]] || usage

SCENARIO="$1"

require_scenario "${SCENARIO}"

BENCHMARK_ID="$(benchmark_id "${SCENARIO}")"
BENCHMARK_DIR="$(benchmark_dataset_dir "${BENCHMARK_ID}")"

TOOLS=($(benchmark_tools "${SCENARIO}"))
ITERATIONS="$(benchmark_iterations "${SCENARIO}")"
WARMUP="$(benchmark_warmup "${SCENARIO}")"

print_header

log_info "Running benchmark: ${BENCHMARK_ID}"

"${SCRIPT_DIR}/validate.sh"

ensure_directory "${BENCHMARK_DIR}"
ensure_directory "$(raw_dir "${BENCHMARK_ID}")"
ensure_directory "$(plots_dir "${BENCHMARK_ID}")"

"${SCRIPT_DIR}/collect.sh" "${SCENARIO}" "${BENCHMARK_DIR}"

for TOOL in "${TOOLS[@]}"; do

    log_info "Running ${TOOL}"

    TOOL_DIR="$(raw_dir "${BENCHMARK_ID}")/${TOOL}"
    ensure_directory "${TOOL_DIR}"

    log_info "Warm-up (${WARMUP})..."

    case "${TOOL}" in

        wrk)
            sleep "${WARMUP%s}"
            ;;

        vegeta)
            sleep "${WARMUP%s}"
            ;;

        *)
            die "Unsupported benchmark tool: ${TOOL}"
            ;;

    esac

    for ((RUN=1; RUN<=ITERATIONS; RUN++)); do

        RUN_DIR="${TOOL_DIR}/run-$(printf "%03d" "${RUN}")"

        ensure_directory "${RUN_DIR}"

        log_info "Run ${RUN}/${ITERATIONS}"

        "${SCRIPT_DIR}/monitor.sh" start "${RUN_DIR}"

        case "${TOOL}" in

            wrk)

                wrk \
                    -t "$(benchmark_threads "${SCENARIO}")" \
                    -c "$(benchmark_connections "${SCENARIO}")" \
                    -d "$(benchmark_duration "${SCENARIO}")" \
                    "$(benchmark_url "${SCENARIO}")" \
                    > "${RUN_DIR}/wrk.txt"

                ;;

                        vegeta)

                cat > "${RUN_DIR}/targets.txt" <<EOF
GET $(benchmark_url "${SCENARIO}")
EOF

                vegeta attack \
                    -insecure \
                    -rate="$(benchmark_rate "${SCENARIO}")" \
                    -duration="$(benchmark_duration "${SCENARIO}")" \
                    < "${RUN_DIR}/targets.txt" \
                    > "${RUN_DIR}/results.bin"

                vegeta report \
                    "${RUN_DIR}/results.bin" \
                    > "${RUN_DIR}/vegeta.txt"

                vegeta report \
                    -type=json \
                    "${RUN_DIR}/results.bin" \
                    > "${RUN_DIR}/vegeta.json"

                vegeta report \
                    -type='hist[0,2ms,4ms,8ms,16ms,32ms]' \
                    "${RUN_DIR}/results.bin" \
                    > "${RUN_DIR}/histogram.txt"

                ;;

        esac

        "${SCRIPT_DIR}/monitor.sh" stop

    done

done

"${SCRIPT_DIR}/analyze.sh" "${BENCHMARK_DIR}"

echo
log_success "Benchmark completed."
echo
echo "Dataset:"
echo "  ${BENCHMARK_DIR}"
