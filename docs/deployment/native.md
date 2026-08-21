# Native Deployment

Native deployment runs Torus directly as a host process rather than through a container or systemd.

This is the primary deployment model for local development, debugging, experimentation, and benchmarking.

## Architecture

```text
Host
  │
  ├── Torus
  │
  ├── Backend services
  │
  └── Optional observability stack
        ├── Prometheus
        └── Grafana
```

Torus and backend services run independently. Prometheus and Grafana can be run through the repository's Docker configuration.

## Prerequisites

- Go 1.26.1 or newer
- One or more backend HTTP services
- Docker, if the local Prometheus/Grafana stack is used
- TLS certificate and key files if HTTPS is required

## 1. Clone

```bash
git clone https://github.com/Ashish-Barmaiya/torus-proxy.git
cd torus-proxy
```

## 2. Build Torus

```bash
go build -o torus ./cmd/torus
```

For local testing, the repository includes a mock backend.

Start one backend:

```bash
go run ./cmd/mock-backend 3001
```

Start a second backend in another terminal:

```bash
go run ./cmd/mock-backend 3002
```

## 3. Configure Torus

Create or customize a configuration using the [Configuration Guide](../configuration/README.md).

The repository provides example configurations under:

```text
configs/
```

For example:

```text
configs/torus-http.yaml
configs/torus-https.yaml
```

The configuration determines the listener, TLS behavior, services, routes, health checks, and observability.

## 4. Start Torus

Pass the configuration explicitly:

```bash
./torus -config configs/torus-http.yaml
```

For HTTPS:

```bash
./torus -config configs/torus-https.yaml
```

## 5. Optional observability stack

The repository includes a Docker-based Prometheus and Grafana setup.

```bash
docker compose -f docker/compose.yml up -d
```

The observability stack is independent of the Torus process.

This starts:

| Service | Port | Purpose |
| ----- | ----- | ----- |
| Prometheus | `9090` | Scrapes and stores Torus metrics |
| Grafana | `3000` | Visualizes Prometheus data |

The Prometheus configuration under `docker/prometheus/prometheus.yml` must target the address and port on which the native Torus process is listening.

For example, if Torus listens on:

server:
 addr: ":8080"

the Prometheus target should point to the corresponding host address and port.

### **Verify the observability stack**

First verify that Torus exposes metrics:

curl http://localhost:8080/metrics

The endpoint should return Prometheus exposition-format metrics.

Next open Prometheus:

http://localhost:9090

Navigate to:

Status → Targets

The Torus target should be shown as `UP`.

Then open Grafana:

http://localhost:3000

Use the credentials configured by the Docker deployment.

Generate a few requests through Torus and verify that the provisioned dashboards begin showing data.

### **Stop the observability stack**

```bash
docker compose -f docker/compose.yml down
```

## 6. Verify

Check readiness:

```bash
curl http://localhost:8080/readyz
```

Send a proxied request:

```bash
curl http://localhost:8080/api/hello
```

For HTTPS:

```bash
curl -k https://localhost:8443/readyz
curl -k https://localhost:8443/api/hello
```

Use the actual listener configured in your YAML.

## 7. Stop

Stop the foreground Torus process with `Ctrl-C`.

Stop the optional observability stack with:

```bash
docker compose -f docker/compose.yml down
```

## Intended use

Native deployment is appropriate for:

- development
- debugging
- configuration experiments
- performance benchmarking
- learning Torus behavior

It is not the recommended mechanism for keeping Torus automatically running across host reboots or supervising a long-lived Linux VM service.

For Linux VM operation, use [systemd](./systemd.md).
