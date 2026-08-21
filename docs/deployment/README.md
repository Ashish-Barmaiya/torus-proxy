# Deployment

Torus supports three deployment models.

| Deployment | Intended use |
|---|---|
| Native | Local development, debugging, benchmarking, and direct execution |
| Docker Compose | Reproducible full-stack development and evaluation |
| Linux/systemd | Linux VM deployment and production-oriented operation |

Configuration is documented separately in the [Configuration Guide](../configuration/README.md).

## Deployment models

### Native

Torus runs directly as a host process.

```text
Torus
  │
  ├── Backend services → independently managed
  │
  └── Prometheus/Grafana → optional Docker deployment
```

Use this model when developing Torus, experimenting with configuration, debugging behavior, or running benchmarks.

See [Native Deployment](./native.md).

### Docker Compose

Torus, mock backends, Prometheus, and Grafana can run together as a full stack.

```text
Docker Compose
  ├── Torus
  ├── Backend A
  ├── Backend B
  ├── Prometheus
  └── Grafana
```

The repository provides HTTP and HTTPS full-stack Compose configurations.

This deployment is intended for development, demonstrations, and evaluation. The repository's mock backends, development credentials, and development certificates are not a production environment.

See [Docker Compose Deployment](./docker.md).

### Linux/systemd

Torus can run as a native Linux systemd service.

```text
Linux VM
  │
  ├── Torus → systemd
  │
  ├── Backend services → independently managed
  │
  └── Prometheus/Grafana → separately managed
```

The repository includes an interactive installer that builds Torus from the source checkout, installs the supplied configuration, optionally installs TLS material, configures the systemd service, and verifies the configured listener.

See [Linux/systemd Deployment](./systemd.md).

## Choosing a deployment

Use Native when:

- developing Torus
- debugging
- experimenting with configuration
- running benchmarks

Use Docker Compose when:

- you want the entire local stack in one command
- you want isolated Torus and mock backends
- you want a reproducible development/evaluation environment

Use systemd when:

- deploying Torus on a Linux VM
- running Torus as a long-lived host service
- managing Torus through the native Linux service manager

For a production-oriented Linux VM deployment, see [Production Deployment](./production.md).

## Configuration

Deployment does not define what Torus should do. The supplied Torus configuration does.

See the [Configuration Guide](../configuration/README.md) for:

- creating and customizing configuration files
- server and listener configuration
- services and upstreams
- routing
- health checks
- TLS
- observability
- configuration reloads
- the configuration reference

Repository-provided generic examples live under:

```text
configs/
```

Docker-specific deployment configuration and supporting assets live under:

```text
docker/
```

## Deployment and runtime responsibility

Deployment tools install and activate Torus. They do not become the runtime supervisor.

For the systemd deployment:

```text
install.sh
    ↓
systemd
    ↓
Torus
```

Their responsibilities are separate:

| Component | Responsibility |
|---|---|
| `install.sh` | Install and activate the deployment |
| systemd | Supervise the Torus process |
| Torus | Manage application runtime, reloads, and graceful shutdown |

## Failure handling

The systemd installer separates installation failures from runtime failures.

### Preflight failure

Invalid configuration, invalid TLS material, or another validation failure occurs before installation changes are made.

The installer exits without mutating the existing deployment.

### Installation or activation failure

If installation has started and activation fails, the installer attempts to roll back to the previous deployment state.

### Runtime failure after installation

Once installation has succeeded, runtime supervision belongs to systemd and Torus.

The installer is not rerun merely because the Torus process later crashes. systemd is responsible for process restart according to the service policy.

## Production

For a Linux VM, the systemd deployment is the reference production-oriented deployment model in v0.6.0.

The current installer builds Torus from the source checkout. A future release can replace that source build step with installation of a prebuilt release binary without changing the overall systemd deployment model.

See [Production Deployment](./production.md).
