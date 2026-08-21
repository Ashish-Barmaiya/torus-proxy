# Production Deployment

For a Linux virtual machine, the recommended production-oriented deployment model in v0.6.0 is to run Torus as a native systemd service.

The repository's Docker Compose configurations are intended for development and evaluation. They include mock backends and development-oriented assets.

## Reference architecture

A simple single-VM deployment can look like:

```text
                         Internet
                            │
                          HTTPS
                            │
                            ▼
                   ┌─────────────────┐
                   │      Torus      │
                   │     systemd     │
                   │    :8443/:443   │
                   └────────┬────────┘
                            │
                 ┌──────────┴──────────┐
                 ▼                     ▼
             Backend A             Backend B


                   Observability
                         │
                ┌────────┴────────┐
                ▼                 ▼
           Prometheus          Grafana
```

Torus and the backend services are independent components.

Prometheus and Grafana may be co-located on the same VM for a small deployment or managed separately as the environment grows.

## What is production-specific?

A production deployment should use:

- real backend services
- a configuration created for the target environment
- production TLS certificates and private keys
- appropriate network exposure and firewall rules
- an operational monitoring strategy
- a certificate renewal process
- a deployment and rollback procedure

Do not use repository development certificates, mock backends, or default development credentials in production.

## 1. Provision the Linux VM

Provision a supported Linux system with systemd.

The VM should have:

- sufficient CPU and memory for the expected traffic
- network connectivity to the backend services
- network exposure only where required
- system time synchronized correctly
- a supported Go installation for the current source-based installer

## 2. Install prerequisites

The v0.6.0 systemd installer requires Go to build Torus.

Verify:

```bash
go version
```

Verify systemd:

```bash
systemctl --version
```

The installer also uses:

```text
curl
ss
```

and OpenSSL for HTTPS deployments.

## 3. Obtain Torus

Clone the repository:

```bash
git clone https://github.com/Ashish-Barmaiya/torus-proxy.git
cd torus-proxy
```

The current v0.6.0 systemd installer builds Torus from this checkout.

> A future release can use pre-built release binaries instead of compiling Torus on the target machine.

## 4. Create the production configuration

Create a Torus configuration appropriate for the production environment.

Example:

```yaml
apiVersion: v2

server:
  addr: ":8443"

tls:
  cert_file: "/etc/torus/tls/cert.pem"
  key_file: "/etc/torus/tls/key.pem"
  min_version: "1.2"

health:
  interval_ms: 5000
  timeout_ms: 2000
  path: /health

observability:
  enabled: true

services:
  - name: api
    upstreams:
      - http://10.0.1.21:3001
      - http://10.0.1.22:3001

routes:
  - path: /api
    service: api
```

Replace the backend addresses, listener, routes, and other values with the actual environment.

See the [Configuration Guide](../configuration/README.md).

## 5. Prepare TLS

For HTTPS, use a certificate and private key issued for the production hostname.

Do not commit private keys to the repository.

The systemd installer expects the configuration to reference its managed locations:

```yaml
tls:
  cert_file: "/etc/torus/tls/cert.pem"
  key_file: "/etc/torus/tls/key.pem"
```

The installer asks for the source certificate and private-key paths and installs them into those locations.

Plan certificate renewal and rotation independently from Torus installation.

## 6. Install Torus

From the repository root:

```bash
sudo ./deployments/systemd/install.sh
```

Provide the production configuration file.

For HTTPS, provide the production certificate and private-key source paths.

The installer:

```text
Validate configuration and TLS
        ↓
Build Torus
        ↓
Back up existing deployment if present
        ↓
Install binary/configuration/TLS
        ↓
Install systemd service
        ↓
Enable service
        ↓
Start service
        ↓
Verify configured listener
```

## 7. Verify the deployment

Check systemd:

```bash
sudo systemctl status torus --no-pager
```

Check readiness using the configured listener.

HTTP example:

```bash
curl http://127.0.0.1:8080/readyz
```

HTTPS example:

```bash
curl -k https://127.0.0.1:8443/readyz
```

In a real environment, verify the production hostname and certificate behavior separately.

Then verify application traffic:

```bash
curl -k https://example.com/api/hello
```

Use a real production route appropriate for the deployed service rather than the development `/api/hello` example when deploying an actual application.

## 8. Backend services

The production backend services should be deployed and managed independently of Torus.

Do not use:

```text
docker/compose.full.yml
```

as the backend production architecture unless you intentionally build a separate production Compose configuration.

Torus should point to the real upstream services through the `services` section of the production configuration.

## 9. Networking

Expose only the ports required by the deployment.

For example:

```text
Internet
   │
   │ HTTPS
   ▼
Torus
   │
   ├── Backend network
   └── Observability network
```

If Torus binds to:

```yaml
server:
  addr: ":8443"
```

ensure the host firewall and any external security group allow the required traffic.

Backend ports generally do not need to be exposed publicly when they can remain on a private network.

## 10. Observability

Observability should be treated as a separate operational subsystem from Torus itself.

Torus exposes Prometheus-compatible metrics when enabled:

```yaml
observability:
  enabled: true
```

A production deployment can place Prometheus and Grafana on the same VM as Torus for a small installation or operate them on separate monitoring infrastructure.

A typical architecture is:

```text
                   ┌─────────────┐
                   │    Torus    │
                   │   systemd   │
                   └──────┬──────┘
                          │
                      /metrics
                          │
                          ▼
                   ┌─────────────┐
                   │ Prometheus  │
                   └──────┬──────┘
                          │
                          ▼
                   ┌─────────────┐
                   │   Grafana   │
                   └─────────────┘
```

### **Verify the Torus metrics endpoint**

Verify that Torus exposes metrics on its configured listener.

For HTTP:

curl http://127.0.0.1:8080/metrics

For HTTPS:

curl \-k https://127.0.0.1:8443/metrics

Use the actual production listener configured in `/etc/torus/torus.yaml`.

The response should contain Prometheus exposition-format metrics.

### **Verify Prometheus**

Configure Prometheus to scrape the Torus metrics endpoint.

Then verify the target:

Prometheus → Status → Targets

The Torus target should report:

UP

If the target is `DOWN`, check:

* the configured Torus listener address and port;
* network connectivity between Prometheus and Torus;
* firewall rules;
* whether `observability.enabled` is `true`;
* whether `/metrics` is reachable from the Prometheus host.

### **Verify Grafana**

Configure Grafana to use Prometheus as its data source.

After Prometheus begins collecting Torus metrics, generate application traffic and verify that the dashboards show current data.

The exact Grafana deployment is environment-specific. The repository's Docker Grafana configuration is primarily intended for development and evaluation rather than as a universal production monitoring configuration.

### **Monitoring architecture**

For a small single-VM deployment:

```text
VM
├── Torus
│   └── systemd
├── Prometheus
└── Grafana
```

For a larger environment:

```text
Application VM(s)
└── Torus
   └── systemd

Monitoring infrastructure
├── Prometheus
└── Grafana
```

Separating monitoring infrastructure from the application host generally provides stronger isolation and allows the monitoring system to remain available when an application VM is unhealthy.

### **Operational separation**

Prometheus and Grafana are not managed by the Torus systemd installer.

Do not reinstall Torus to troubleshoot a Prometheus or Grafana problem.

Manage the monitoring stack independently from the Torus service.


## 11. Logs and service management

Use systemd for operational management:

```bash
sudo systemctl status torus
```

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

Check boot persistence:

```bash
sudo systemctl is-enabled torus
```

## 12. Runtime failures

After a successful installation, do not rerun the installer merely because the Torus process crashes.

The systemd service is responsible for process supervision.

```text
Torus process failure
        ↓
systemd detects failure
        ↓
Restart policy
        ↓
Torus starts again
```

Inspect:

```bash
sudo systemctl status torus
sudo journalctl -u torus
```

Only use the installer again when intentionally redeploying the application or changing deployment-managed artifacts.

## 13. Configuration changes

Torus supports configuration reloads.

For ordinary configuration changes, prefer the configuration reload mechanism rather than reinstalling Torus.

See:

```text
docs/configuration/configuration-reload.md
```

A configuration change should be validated before it replaces the active runtime.

## 14. Re-deployment and rollback

When a new binary or deployment configuration is intentionally installed:

```bash
sudo ./deployments/systemd/install.sh
```

The installer preserves the previous configuration and attempts to restore the previous deployment state if activation fails.

Operationally:

```text
Existing deployment
        ↓
New deployment
        ↓
Activation----+
   │          │
Success     Failure
   │          │
   ▼          ▼
 New state  Rollback
```

The installer creates a configuration backup at:

```text
/etc/torus/torus.yaml.bak
```

The current v0.6.0 installer is a source-based deployment mechanism rather than a prebuilt release distribution mechanism.

## 15. Production TLS lifecycle

Certificate installation and certificate renewal are separate concerns.

The installer can place certificate and key files in the systemd-managed TLS directory, but a production environment still needs a process for:

- certificate issuance
- renewal
- rotation
- verification
- rollback of certificate changes

A certificate renewal mechanism should integrate with Torus configuration reload or an explicit service restart strategy appropriate to the deployment.

## 16. Production limitations of v0.6.0

The v0.6.0 systemd deployment is production-oriented but still source-based.

The installation path is:

```text
Git checkout
    ↓
Go toolchain
    ↓
Build Torus
    ↓
Install binary
    ↓
systemd
```

A future production distribution can replace the build step with a verified pre-built release binary:

```text
Release artifact
    ↓
Verify
    ↓
Install binary
    ↓
systemd
```

The deployment model itself does not need to change.

## 17. Recommended production checklist

Before exposing Torus to production traffic, verify:

- [ ] Production configuration created and reviewed
- [ ] Production backend services reachable
- [ ] Production TLS certificate installed
- [ ] Private key protected
- [ ] Production listener and firewall configured
- [ ] DNS configured
- [ ] `/readyz` verified
- [ ] Application request path verified
- [ ] Prometheus or equivalent monitoring configured
- [ ] Logs accessible
- [ ] systemd enabled at boot
- [ ] Certificate renewal process established
- [ ] Deployment and rollback procedure tested

## Related documentation

- [Configuration](../configuration/README.md)
- [Linux/systemd Deployment](./systemd.md)
- [Docker Compose Deployment](./docker.md)
- [Native Deployment](./native.md)
