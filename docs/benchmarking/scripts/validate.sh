#!/usr/bin/env bash

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${SCRIPT_DIR}/common.sh"

print_header

log_info "Validating benchmark environment..."
echo

require_git_repo

echo "Operating System"
log_success "$(os_name)"

echo
echo "Kernel"
log_success "$(kernel_version)"

echo
echo "Commands"

require_command git
require_command go
require_command python3
require_command jq
require_command yq

require_command wrk
require_command vegeta

require_command pidstat
require_command vmstat
require_command ss

require_command perf

echo
echo "Repository"

validate_repository_structure
log_success "Benchmark directory structure"

if git_dirty; then
    log_warning "Repository contains uncommitted changes."
else
    log_success "Working tree is clean."
fi

echo
echo "System"

log_success "Hostname           : $(hostname_name)"
log_success "CPU                : $(cpu_model)"
log_success "Logical CPUs       : $(logical_cpus)"
log_success "Memory             : $(memory_total)"
log_success "Available Disk     : $(disk_available)"

echo
echo "Tool Versions"

log_success "$(tool_version go)"
log_success "$(tool_version git)"
log_success "$(tool_version python3)"
log_success "$(tool_version jq)"
log_success "$(tool_version yq)"
log_success "$(tool_version wrk)"
log_success "$(tool_version vegeta)"
log_success "$(tool_version perf)"

echo
log_success "Environment validation passed."
