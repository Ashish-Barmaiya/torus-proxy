#!/usr/bin/env bash

set -Eeuo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

readonly SERVICE_SOURCE="${SCRIPT_DIR}/torus.service"

readonly SERVICE_NAME="torus.service"
readonly SERVICE_DEST="/etc/systemd/system/${SERVICE_NAME}"

readonly TORUS_BINARY="/usr/local/bin/torus"

readonly TORUS_HOME="/etc/torus"
readonly TORUS_CONFIG="${TORUS_HOME}/torus.yaml"
readonly TORUS_CONFIG_BACKUP="${TORUS_HOME}/torus.yaml.bak"

readonly TORUS_TLS_DIR="${TORUS_HOME}/tls"
readonly TORUS_TLS_CERT="${TORUS_TLS_DIR}/cert.pem"
readonly TORUS_TLS_KEY="${TORUS_TLS_DIR}/key.pem"

readonly TORUS_USER="torus"
readonly TORUS_GROUP="torus"

GO_BIN=""
TMP_DIR=""

CONFIG_SOURCE=""
TLS_CERT_SOURCE=""
TLS_KEY_SOURCE=""

SERVER_ADDR=""
TLS_ENABLED="false"
CONFIG_TLS_CERT_FILE=""
CONFIG_TLS_KEY_FILE=""

BACKUP_DIR=""

MUTATION_STARTED=0
INSTALL_SUCCEEDED=0

EXISTING_BINARY=0
EXISTING_CONFIG=0
EXISTING_SERVICE=0
EXISTING_TLS_CERT=0
EXISTING_TLS_KEY=0

SERVICE_WAS_ACTIVE=0
SERVICE_WAS_ENABLED=0

CREATED_USER=0
CREATED_GROUP=0
CREATED_HOME=0
CREATED_TLS_DIR=0

log() {
    printf '[INFO] %s\n' "$1"
}

error() {
    printf '[ERROR] %s\n' "$1" >&2
    exit 1
}

cleanup_tmp() {
    if [[ -n "${TMP_DIR}" && -d "${TMP_DIR}" ]]; then
        rm -rf "${TMP_DIR}"
    fi
}

rollback() {
    if [[ "${MUTATION_STARTED}" -ne 1 || "${INSTALL_SUCCEEDED}" -eq 1 ]]; then
        return 0
    fi

    echo
    printf '[ERROR] Installation failed. Rolling back changes...\n' >&2

    # Prevent a broken newly-installed service from continuing to restart.
    systemctl stop "${SERVICE_NAME}" >/dev/null 2>&1 || true

    # Restore binary.
    if [[ "${EXISTING_BINARY}" -eq 1 && -f "${BACKUP_DIR}/torus" ]]; then
        install \
            -o root \
            -g root \
            -m 0755 \
            "${BACKUP_DIR}/torus" \
            "${TORUS_BINARY}"
    else
        rm -f "${TORUS_BINARY}"
    fi

    # Restore configuration.
    if [[ "${EXISTING_CONFIG}" -eq 1 && -f "${BACKUP_DIR}/torus.yaml" ]]; then
        install \
            -o root \
            -g "${TORUS_GROUP}" \
            -m 0640 \
            "${BACKUP_DIR}/torus.yaml" \
            "${TORUS_CONFIG}"
    else
        rm -f "${TORUS_CONFIG}"
    fi

    # Restore TLS certificate.
    if [[ "${EXISTING_TLS_CERT}" -eq 1 && -f "${BACKUP_DIR}/cert.pem" ]]; then
        install \
            -o root \
            -g "${TORUS_GROUP}" \
            -m 0640 \
            "${BACKUP_DIR}/cert.pem" \
            "${TORUS_TLS_CERT}"
    else
        rm -f "${TORUS_TLS_CERT}"
    fi

    # Restore TLS private key.
    if [[ "${EXISTING_TLS_KEY}" -eq 1 && -f "${BACKUP_DIR}/key.pem" ]]; then
        install \
            -o root \
            -g "${TORUS_GROUP}" \
            -m 0640 \
            "${BACKUP_DIR}/key.pem" \
            "${TORUS_TLS_KEY}"
    else
        rm -f "${TORUS_TLS_KEY}"
    fi

    # Restore systemd unit.
    if [[ "${EXISTING_SERVICE}" -eq 1 && -f "${BACKUP_DIR}/torus.service" ]]; then
        install \
            -o root \
            -g root \
            -m 0644 \
            "${BACKUP_DIR}/torus.service" \
            "${SERVICE_DEST}"
    else
        rm -f "${SERVICE_DEST}"
    fi

    systemctl daemon-reload >/dev/null 2>&1 || true

    # Restore previous service state.
    if [[ "${EXISTING_SERVICE}" -eq 1 ]]; then
        if [[ "${SERVICE_WAS_ENABLED}" -eq 1 ]]; then
            systemctl enable "${SERVICE_NAME}" >/dev/null 2>&1 || true
        else
            systemctl disable "${SERVICE_NAME}" >/dev/null 2>&1 || true
        fi

        if [[ "${SERVICE_WAS_ACTIVE}" -eq 1 ]]; then
            systemctl restart "${SERVICE_NAME}" >/dev/null 2>&1 || true
        fi
    else
        systemctl disable "${SERVICE_NAME}" >/dev/null 2>&1 || true
    fi

    # Remove directories created by this installation.
    if [[ "${CREATED_TLS_DIR}" -eq 1 ]]; then
        rmdir "${TORUS_TLS_DIR}" >/dev/null 2>&1 || true
    fi

    if [[ "${CREATED_HOME}" -eq 1 ]]; then
        rmdir "${TORUS_HOME}" >/dev/null 2>&1 || true
    fi

    # Remove user/group only if this installer created them.
    if [[ "${CREATED_USER}" -eq 1 ]]; then
        userdel "${TORUS_USER}" >/dev/null 2>&1 || true
    fi

    if [[ "${CREATED_GROUP}" -eq 1 ]]; then
        groupdel "${TORUS_GROUP}" >/dev/null 2>&1 || true
    fi

    printf '[ERROR] Rollback completed.\n' >&2
}

on_exit() {
    local exit_code=$?

    if [[ "${exit_code}" -ne 0 ]]; then
        rollback
    fi

    cleanup_tmp
    exit "${exit_code}"
}

trap on_exit EXIT

require_root() {
    [[ "${EUID}" -eq 0 ]] \
        || error "This installer must be run as root. Use sudo."
}

check_platform() {
    log "Checking platform..."

    [[ "$(uname -s)" == "Linux" ]] \
        || error "Torus systemd deployment requires Linux."

    command -v systemctl >/dev/null 2>&1 \
        || error "systemctl was not found."

    systemctl --version >/dev/null 2>&1 \
        || error "systemd is not available."

    command -v curl >/dev/null 2>&1 \
        || error "curl is required."

    command -v ss >/dev/null 2>&1 \
        || error "ss is required to verify the Torus listening socket."

    log "Platform and systemd: OK"
}

find_go() {
    log "Checking Go..."

    local candidates=(
        "$(command -v go 2>/dev/null || true)"
        "/usr/local/go/bin/go"
        "/usr/bin/go"
        "/snap/bin/go"
    )

    for candidate in "${candidates[@]}"; do
        if [[ -n "${candidate}" && -x "${candidate}" ]]; then
            GO_BIN="${candidate}"
            break
        fi
    done

    [[ -n "${GO_BIN}" ]] \
        || error "Go is required to build Torus. Install Go and run the installer again."

    log "Go found: ${GO_BIN}"
}

select_config() {
    echo
    echo "Torus Proxy — systemd installer"
    echo
    echo "Provide the Torus configuration file to install."
    echo
    echo "The configuration determines:"
    echo "  - listener address"
    echo "  - TLS configuration"
    echo "  - health checks"
    echo "  - observability"
    echo "  - services"
    echo "  - routes"
    echo

    while true; do
        read -r -p "Configuration file path: " CONFIG_SOURCE

        [[ -n "${CONFIG_SOURCE}" ]] || {
            echo "Configuration path cannot be empty."
            continue
        }

        if [[ -f "${CONFIG_SOURCE}" ]]; then
            CONFIG_SOURCE="$(realpath -e "${CONFIG_SOURCE}")"
            break
        fi

        echo "Configuration file not found: ${CONFIG_SOURCE}"
    done

    log "Configuration: ${CONFIG_SOURCE}"
}

build_config_inspector() {
    local source_dir="${TMP_DIR}/config-inspector-src"
    local binary="${TMP_DIR}/config-inspector"

    mkdir -p "${source_dir}"

    cat > "${source_dir}/main.go" <<'EOF'
package main

import (
	"fmt"
	"net"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`

	TLS *struct {
		CertFile   string `yaml:"cert_file"`
		KeyFile    string `yaml:"key_file"`
		MinVersion string `yaml:"min_version"`
	} `yaml:"tls"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: config-inspector CONFIG")
		os.Exit(2)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read config: %v\n", err)
		os.Exit(1)
	}

	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "parse config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Server.Addr == "" {
		fmt.Fprintln(os.Stderr, "server.addr must not be empty")
		os.Exit(1)
	}

	host, port, err := net.SplitHostPort(cfg.Server.Addr)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"invalid server.addr %q: %v\n",
			cfg.Server.Addr,
			err,
		)
		os.Exit(1)
	}

	// Go represents :8080 as an empty host.
	if host == "" {
		host = "127.0.0.1"
	}

	if port == "" {
		fmt.Fprintln(os.Stderr, "server.addr port must not be empty")
		os.Exit(1)
	}

	tlsEnabled := false
	certFile := ""
	keyFile := ""

	if cfg.TLS != nil {
		tlsEnabled = true
		certFile = cfg.TLS.CertFile
		keyFile = cfg.TLS.KeyFile

		if certFile == "" {
			fmt.Fprintln(os.Stderr, "tls.cert_file must not be empty")
			os.Exit(1)
		}

		if keyFile == "" {
			fmt.Fprintln(os.Stderr, "tls.key_file must not be empty")
			os.Exit(1)
		}
	}

	fmt.Printf("server_addr=%s\n", cfg.Server.Addr)
	fmt.Printf("host=%s\n", host)
	fmt.Printf("port=%s\n", port)
	fmt.Printf("tls_enabled=%t\n", tlsEnabled)
	fmt.Printf("tls_cert_file=%s\n", certFile)
	fmt.Printf("tls_key_file=%s\n", keyFile)
}
EOF

    (
        cd "${PROJECT_ROOT}"

        "${GO_BIN}" build \
            -o "${binary}" \
            "${source_dir}/main.go"
    )

    [[ -x "${binary}" ]] \
        || error "Configuration inspector build completed but binary was not produced."
}

inspect_config() {
    log "Validating Torus configuration..."

    local metadata

    metadata="$(
        "${TMP_DIR}/config-inspector" "${CONFIG_SOURCE}"
    )" || error "Invalid Torus configuration."

    while IFS='=' read -r key value; do
        case "${key}" in
            server_addr)
                SERVER_ADDR="${value}"
                ;;
            host)
                SERVER_HOST="${value}"
                ;;
            port)
                SERVER_PORT="${value}"
                ;;
            tls_enabled)
                TLS_ENABLED="${value}"
                ;;
            tls_cert_file)
                CONFIG_TLS_CERT_FILE="${value}"
                ;;
            tls_key_file)
                CONFIG_TLS_KEY_FILE="${value}"
                ;;
        esac
    done <<< "${metadata}"

    log "Configuration: OK"
    log "Server address: ${SERVER_ADDR}"

    if [[ "${TLS_ENABLED}" == "true" ]]; then
        log "TLS configuration: enabled"
    else
        log "TLS configuration: disabled"
    fi
}

select_tls_files() {
    [[ "${TLS_ENABLED}" == "true" ]] || return 0

    echo
    echo "TLS is enabled by the supplied configuration."
    echo "Configuration certificate path: ${CONFIG_TLS_CERT_FILE}"
    echo "Configuration private-key path: ${CONFIG_TLS_KEY_FILE}"
    echo
    echo "The installer will install:"
    echo "  ${TORUS_TLS_CERT}"
    echo "  ${TORUS_TLS_KEY}"
    echo

    while true; do
        read -r -p "TLS certificate source path: " TLS_CERT_SOURCE

        [[ -n "${TLS_CERT_SOURCE}" ]] || {
            echo "Certificate path cannot be empty."
            continue
        }

        if [[ -f "${TLS_CERT_SOURCE}" ]]; then
            TLS_CERT_SOURCE="$(realpath -e "${TLS_CERT_SOURCE}")"
            break
        fi

        echo "Certificate file not found: ${TLS_CERT_SOURCE}"
    done

    while true; do
        read -r -p "TLS private key source path: " TLS_KEY_SOURCE

        [[ -n "${TLS_KEY_SOURCE}" ]] || {
            echo "Private key path cannot be empty."
            continue
        }

        if [[ -f "${TLS_KEY_SOURCE}" ]]; then
            TLS_KEY_SOURCE="$(realpath -e "${TLS_KEY_SOURCE}")"
            break
        fi

        echo "Private key file not found: ${TLS_KEY_SOURCE}"
    done
}

validate_tls() {
    [[ "${TLS_ENABLED}" == "true" ]] || return 0

    command -v openssl >/dev/null 2>&1 \
        || error "openssl is required for TLS installation."

    log "Validating TLS certificate..."

    openssl x509 \
        -in "${TLS_CERT_SOURCE}" \
        -noout >/dev/null 2>&1 \
        || error "Invalid TLS certificate: ${TLS_CERT_SOURCE}"

    log "TLS certificate: OK"

    log "Validating TLS private key..."

    openssl pkey \
        -in "${TLS_KEY_SOURCE}" \
        -noout >/dev/null 2>&1 \
        || error "Invalid or unsupported TLS private key: ${TLS_KEY_SOURCE}"

    log "TLS private key: OK"

    log "Checking certificate/private key match..."

    local cert_public_key
    local key_public_key

    cert_public_key="$(
        openssl x509 \
            -in "${TLS_CERT_SOURCE}" \
            -pubkey \
            -noout |
        openssl pkey \
            -pubin \
            -outform pem |
        sha256sum
    )"

    key_public_key="$(
        openssl pkey \
            -in "${TLS_KEY_SOURCE}" \
            -pubout \
            -outform pem |
        sha256sum
    )"

    [[ "${cert_public_key}" == "${key_public_key}" ]] \
        || error "TLS certificate and private key do not match."

    log "TLS certificate/private key match: OK"

    # The systemd deployment config must reference the paths
    # managed by the installer.
    [[ "${CONFIG_TLS_CERT_FILE}" == "${TORUS_TLS_CERT}" ]] \
        || error \
            "Configuration tls.cert_file must be '${TORUS_TLS_CERT}' for systemd deployment."

    [[ "${CONFIG_TLS_KEY_FILE}" == "${TORUS_TLS_KEY}" ]] \
        || error \
            "Configuration tls.key_file must be '${TORUS_TLS_KEY}' for systemd deployment."
}

validate_inputs() {
    [[ -f "${SERVICE_SOURCE}" ]] \
        || error "systemd service file not found: ${SERVICE_SOURCE}"

    validate_tls

    log "Configuration and deployment inputs: OK"
}

build_torus() {
    log "Building Torus..."

    mkdir -p "${TMP_DIR}/build"

    (
        cd "${PROJECT_ROOT}"

        "${GO_BIN}" build \
            -o "${TMP_DIR}/build/torus" \
            ./cmd/torus
    )

    [[ -x "${TMP_DIR}/build/torus" ]] \
        || error "Torus build completed but binary was not produced."

    log "Torus build: OK"
}

prepare_backup() {
    BACKUP_DIR="${TMP_DIR}/backup"
    mkdir -p "${BACKUP_DIR}"

    if [[ -f "${TORUS_BINARY}" ]]; then
        cp -a "${TORUS_BINARY}" "${BACKUP_DIR}/torus"
        EXISTING_BINARY=1
    fi

    if [[ -f "${TORUS_CONFIG}" ]]; then
        cp -a "${TORUS_CONFIG}" "${BACKUP_DIR}/torus.yaml"
        EXISTING_CONFIG=1
    fi

    if [[ -f "${SERVICE_DEST}" ]]; then
        cp -a "${SERVICE_DEST}" "${BACKUP_DIR}/torus.service"
        EXISTING_SERVICE=1
    fi

    if [[ -f "${TORUS_TLS_CERT}" ]]; then
        cp -a "${TORUS_TLS_CERT}" "${BACKUP_DIR}/cert.pem"
        EXISTING_TLS_CERT=1
    fi

    if [[ -f "${TORUS_TLS_KEY}" ]]; then
        cp -a "${TORUS_TLS_KEY}" "${BACKUP_DIR}/key.pem"
        EXISTING_TLS_KEY=1
    fi

    if [[ "${EXISTING_SERVICE}" -eq 1 ]]; then
        if systemctl is-active --quiet "${SERVICE_NAME}"; then
            SERVICE_WAS_ACTIVE=1
        fi

        if systemctl is-enabled --quiet "${SERVICE_NAME}" 2>/dev/null; then
            SERVICE_WAS_ENABLED=1
        fi
    fi
}

create_user_group() {
    if getent group "${TORUS_GROUP}" >/dev/null 2>&1; then
        log "Group '${TORUS_GROUP}' already exists."
    else
        log "Creating group '${TORUS_GROUP}'..."
        groupadd --system "${TORUS_GROUP}"
        CREATED_GROUP=1
    fi

    if id "${TORUS_USER}" >/dev/null 2>&1; then
        log "User '${TORUS_USER}' already exists."
    else
        log "Creating user '${TORUS_USER}'..."

        useradd \
            --system \
            --gid "${TORUS_GROUP}" \
            --no-create-home \
            --shell /usr/sbin/nologin \
            "${TORUS_USER}"

        CREATED_USER=1
    fi
}

prepare_directories() {
    log "Creating Torus directories..."

    if [[ ! -d "${TORUS_HOME}" ]]; then
        install -d \
            -o root \
            -g "${TORUS_GROUP}" \
            -m 0750 \
            "${TORUS_HOME}"

        CREATED_HOME=1
    else
        chown root:"${TORUS_GROUP}" "${TORUS_HOME}"
        chmod 0750 "${TORUS_HOME}"
    fi

    if [[ "${TLS_ENABLED}" == "true" ]]; then
        if [[ ! -d "${TORUS_TLS_DIR}" ]]; then
            install -d \
                -o root \
                -g "${TORUS_GROUP}" \
                -m 0750 \
                "${TORUS_TLS_DIR}"

            CREATED_TLS_DIR=1
        else
            chown root:"${TORUS_GROUP}" "${TORUS_TLS_DIR}"
            chmod 0750 "${TORUS_TLS_DIR}"
        fi
    fi
}

backup_visible_config() {
    if [[ "${EXISTING_CONFIG}" -eq 1 ]]; then
        log "Backing up existing configuration..."

        install \
            -o root \
            -g "${TORUS_GROUP}" \
            -m 0640 \
            "${TORUS_CONFIG}" \
            "${TORUS_CONFIG_BACKUP}"

        log "Existing configuration backed up to ${TORUS_CONFIG_BACKUP}"
    fi
}

install_files() {
    log "Installing Torus binary..."

    install \
        -o root \
        -g root \
        -m 0755 \
        "${TMP_DIR}/build/torus" \
        "${TORUS_BINARY}"

    backup_visible_config

    log "Installing Torus configuration..."

    install \
        -o root \
        -g "${TORUS_GROUP}" \
        -m 0640 \
        "${CONFIG_SOURCE}" \
        "${TORUS_CONFIG}"

    if [[ "${TLS_ENABLED}" == "true" ]]; then
        log "Installing TLS certificate..."

        install \
            -o root \
            -g "${TORUS_GROUP}" \
            -m 0640 \
            "${TLS_CERT_SOURCE}" \
            "${TORUS_TLS_CERT}"

        log "Installing TLS private key..."

        install \
            -o root \
            -g "${TORUS_GROUP}" \
            -m 0640 \
            "${TLS_KEY_SOURCE}" \
            "${TORUS_TLS_KEY}"
    fi

    log "Installing systemd service..."

    install \
        -o root \
        -g root \
        -m 0644 \
        "${SERVICE_SOURCE}" \
        "${SERVICE_DEST}"
}

activate_service() {
    log "Reloading systemd..."
    systemctl daemon-reload

    log "Enabling ${SERVICE_NAME}..."
    systemctl enable "${SERVICE_NAME}" >/dev/null

    log "Starting ${SERVICE_NAME}..."
    systemctl restart "${SERVICE_NAME}"
}

wait_for_service() {
    local attempts=0
    local max_attempts=15

    while (( attempts < max_attempts )); do
        if systemctl is-active --quiet "${SERVICE_NAME}"; then
            return 0
        fi

        sleep 1
        ((attempts += 1))
    done

    return 1
}

wait_for_listener() {
    local attempts=0
    local max_attempts=15

    while (( attempts < max_attempts )); do
        if ss -H -ltn "sport = :${SERVER_PORT}" 2>/dev/null | grep -q .; then
            return 0
        fi

        sleep 1
        ((attempts += 1))
    done

    return 1
}

verify_service() {
    log "Verifying ${SERVICE_NAME}..."

    if ! wait_for_service; then
        echo
        echo "Torus service status:"
        systemctl --no-pager --full status "${SERVICE_NAME}" || true

        echo
        echo "Recent Torus logs:"
        journalctl -u "${SERVICE_NAME}" -n 30 --no-pager -l || true

        error "Torus service failed to become active."
    fi

    log "Torus service is running."

    log "Checking configured listener ${SERVER_ADDR}..."

    if ! wait_for_listener; then
        echo
        echo "Torus service status:"
        systemctl --no-pager --full status "${SERVICE_NAME}" || true

        echo
        echo "Recent Torus logs:"
        journalctl -u "${SERVICE_NAME}" -n 30 --no-pager -l || true

        error "Torus is running but the configured listener is not active: ${SERVER_ADDR}"
    fi

    log "Torus listener: OK"
}

print_success() {
    echo
    log "Torus systemd installation completed successfully."
    echo
    echo "Binary:        ${TORUS_BINARY}"
    echo "Configuration: ${TORUS_CONFIG}"
    echo "Service:       ${SERVICE_DEST}"

    if [[ "${TLS_ENABLED}" == "true" ]]; then
        echo "TLS directory: ${TORUS_TLS_DIR}"
    fi

    echo
    echo "Configured listener:"
    echo "  ${SERVER_ADDR}"
    echo
    echo "Useful commands:"
    echo "  systemctl status torus"
    echo "  journalctl -u torus"
    echo "  systemctl restart torus"
    echo
}

main() {
    require_root
    check_platform
    find_go

    TMP_DIR="$(mktemp -d)"

    build_config_inspector
    select_config
    inspect_config
    select_tls_files
    validate_inputs

    build_torus

    prepare_backup

    MUTATION_STARTED=1

    create_user_group
    prepare_directories
    install_files
    activate_service
    verify_service

    INSTALL_SUCCEEDED=1

    print_success
}

main "$@"
