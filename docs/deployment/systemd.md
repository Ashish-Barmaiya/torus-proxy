# Linux/systemd Deployment

Torus can run as a native Linux systemd service.

The repository includes an interactive installer that:

1. validates the supplied Torus configuration
2. builds Torus from the source checkout
3. creates the `torus` system user and group when necessary
4. installs the binary
5. installs the supplied configuration
6. validates and installs TLS material when TLS is configured
7. installs the systemd service
8. enables and starts the service
9. verifies that the configured listener becomes active.

## Supported platform

This deployment model targets:

```text
Linux + systemd
```

It is not a macOS or Windows service installer.

Docker Compose provides the cross-platform container deployment path.

## Prerequisites

The systemd installer requires:

- Linux
- systemd
- Go
- `curl`
- `ss`
- OpenSSL when TLS is configured

The installer builds Torus locally, so Go is required for the current v0.6.0 source-based installer.

## Configuration

The installer is configuration-driven.

It does not ask the user to select `HTTP` or `HTTPS`.

Instead, the user supplies the Torus configuration:

```bash
sudo ./deployments/systemd/install.sh
```

The installer asks:

```text
Configuration file path:
```

The configuration determines:

- listener address
- whether TLS is enabled
- service definitions
- routes
- health checks
- observability

Create or customize the configuration using the [Configuration Guide](../configuration/README.md).

## TLS configuration

If the supplied configuration contains a `tls` section, the installer asks for the source certificate and private-key files.

For a systemd deployment, the installed configuration should reference:

```yaml
tls:
  cert_file: "/etc/torus/tls/cert.pem"
  key_file: "/etc/torus/tls/key.pem"
  min_version: "1.2"
```

The source files supplied to the installer can be located elsewhere:

```text
TLS certificate source path: /path/to/cert.pem
TLS private key source path: /path/to/key.pem
```

The installer validates:

- the certificate
- the private key
- certificate/private-key matching

The validated files are installed under:

```text
/etc/torus/tls/
```

Do not use development certificates as production credentials.

## Installation

From the repository root:

```bash
sudo ./deployments/systemd/install.sh
```

Provide the configuration path when prompted.

For HTTPS deployments, provide the source certificate and private key paths when prompted.

The installer builds Torus itself. A separate `go build` command is not required.

## Installed layout

A successful installation creates:

```text
/usr/local/bin/torus

/etc/torus/
├── torus.yaml
├── torus.yaml.bak
└── tls/
    ├── cert.pem
    └── key.pem

/etc/systemd/system/
└── torus.service
```

The TLS directory and backup file are present only when applicable.

The `torus` system user and group are used to run the service.

## File ownership and permissions

The installer uses the following ownership model:

```text
Binary:
  root:root
  0755

Configuration:
  root:torus
  0640

TLS certificate:
  root:torus
  0640

TLS private key:
  root:torus
  0640

Torus directory:
  root:torus
  0750
```

This keeps deployment-owned configuration and TLS material readable by the Torus service while restricting general access.

## Service management

Check the service:

```bash
sudo systemctl status torus
```

View logs:

```bash
sudo journalctl -u torus
```

Follow logs:

```bash
sudo journalctl -u torus -f
```

Restart:

```bash
sudo systemctl restart torus
```

Check whether the service is active:

```bash
sudo systemctl is-active torus
```

Check whether it starts automatically at boot:

```bash
sudo systemctl is-enabled torus
```

## Verification

The installer verifies that the configured listener becomes active.

The installer derives the listener from:

```yaml
server:
  addr: ":8080"
```

or any other valid configured address.

After installation, verify the service independently:

```bash
sudo systemctl status torus --no-pager
```

Then send a request to the configured listener.

HTTP example:

```bash
curl http://localhost:8080/readyz
```

HTTPS example:

```bash
curl -k https://localhost:8443/readyz
```

Use the actual address configured in `torus.yaml`.

## Observability

Torus can expose Prometheus metrics when enabled in its configuration:

```yaml
observability:
  enabled: true
```

The systemd installer does not install or manage Prometheus or Grafana. They are separate components of the deployment.

A production-oriented Linux VM can run Prometheus and Grafana independently, either on the same host or on separate monitoring infrastructure.

### **Verify Torus metrics**

Check the metrics endpoint on the configured Torus listener:

curl http://127.0.0.1:8080/metrics

For HTTPS:

curl \-k https://127.0.0.1:8443/metrics

Use the actual listener configured in `/etc/torus/torus.yaml`.

If observability is disabled:

observability:
 enabled: false

the `/metrics` endpoint is not available.

### **Prometheus**

Configure Prometheus to scrape the Torus metrics endpoint.

The Prometheus target must be reachable from the Prometheus deployment and must use the actual address and port of the Torus service.

After configuring Prometheus, verify:

Prometheus → Status → Targets

The Torus target should report:

UP

### **Grafana**

Grafana can use Prometheus as its data source.

After Prometheus begins scraping Torus, verify that the configured dashboards receive data.

For production deployments, Prometheus and Grafana should be operated independently from the Torus systemd service.

### **Operational separation**

```text
systemd
  │
  └── Torus

Prometheus
  │
  └── scrapes Torus

Grafana
  │
  └── queries Prometheus
```

A failure in Prometheus or Grafana does not require reinstalling Torus. Diagnose and manage the monitoring stack separately.

## Configuration changes

The systemd installer is a deployment tool. It is not the normal mechanism for every configuration change.

Torus supports configuration reloads. Configuration changes should use the configuration reload mechanism when appropriate.

See [Configuration Reload](../configuration/configuration-reload.md).

Re-run the installer when the deployment itself needs to be changed, such as installing a new binary, replacing deployment-managed TLS material, or intentionally re-deploying a configuration.

## Configuration backup

When an existing `/etc/torus/torus.yaml` is replaced, the installer preserves the previous configuration as:

```text
/etc/torus/torus.yaml.bak
```

The backup represents the immediately previous installed configuration.

## Failure behavior

The installer separates installation failures from runtime failures.

### Preflight failure

Examples:

- invalid configuration
- invalid certificate
- invalid private key
- certificate/private-key mismatch

No installation mutation occurs.

Fix the input and run the installer again.

### Installation or activation failure

If installation has started and activation fails, the installer attempts to restore the previous deployment state.

For a fresh installation, newly created deployment artifacts are removed when rollback is possible.

For an existing installation, the previous binary, configuration, TLS files, and service definition are restored where applicable.

### Runtime failure after successful installation

Once the installation has completed successfully, the installer is no longer responsible for runtime supervision.

The systemd service uses process restart policy to recover from process failure.

Inspect:

```bash
sudo systemctl status torus
sudo journalctl -u torus
```

Do not reinstall Torus simply because the running process crashed.

## Upgrading or redeploying

To intentionally redeploy Torus, run the installer again:

```bash
sudo ./deployments/systemd/install.sh
```

Provide the desired configuration and TLS material.

The installer treats the operation as a new deployment activation, preserving the existing configuration before replacement and rolling back if activation fails.

## Uninstalling

The v0.6.0 installation can be removed manually through systemd and the installed filesystem paths.

Stop and disable the service:

```bash
sudo systemctl stop torus
sudo systemctl disable torus
```

Remove the service unit:

```bash
sudo rm -f /etc/systemd/system/torus.service
sudo systemctl daemon-reload
```

Remove the binary:

```bash
sudo rm -f /usr/local/bin/torus
```

Review and remove the deployment directory only when its configuration and TLS material are no longer needed:

```bash
sudo rm -rf /etc/torus
```

If the dedicated system account is no longer needed:

```bash
sudo userdel torus
sudo groupdel torus
```

## Related documentation

- [Configuration](../configuration/README.md)
- [Native Deployment](./native.md)
- [Docker Compose Deployment](./docker.md)
- [Production Deployment](./production.md)
