# Docker Compose Deployment

The repository provides Docker Compose deployments for running Torus and its supporting development stack together.

This deployment model is intended for local development, demonstrations, and evaluation.

## What the full stack contains

The full HTTP deployment contains:

```text
Torus
Backend A
Backend B
Prometheus
Grafana
```

The HTTPS deployment contains the same services plus TLS certificate and private-key mounts.

The backend services in these Compose files are mock backends intended for development and evaluation.

## Repository layout

Docker-specific deployment assets live under:

```text
docker/
├── compose.yml
├── compose.full.yml
├── compose.full-https.yml
├── torus/
├── certs/
├── prometheus/
└── grafana/
```

The generic Torus configuration examples under `configs/` are separate from these deployment-specific Docker assets.

When using the full Docker deployment, customize the files under `docker/` that the Compose files mount into the containers.

## Prerequisites

- Docker
- Docker Compose support through `docker compose`

## Full HTTP stack

### Start

From the repository root:

```bash
docker compose -f docker/compose.full.yml up --build -d
```

The `--build` flag builds the Torus and mock-backend images from the repository source before starting the stack.

No separate `go build` or `docker build` command is required.

### Verify

Check the containers:

```bash
docker compose -f docker/compose.full.yml ps
```

Check Torus readiness:

```bash
curl http://localhost:8080/readyz
```

Check request routing:

```bash
curl http://localhost:8080/api/hello
```

The response should be served by one of the mock backends.

### Logs

View the Torus logs:

```bash
docker compose -f docker/compose.full.yml logs torus
```

Follow the logs:

```bash
docker compose -f docker/compose.full.yml logs -f torus
```

### Stop

```bash
docker compose -f docker/compose.full.yml down
```

Named Prometheus and Grafana volumes are retained unless explicitly removed.

## Full HTTPS stack

### Configuration

The HTTPS Compose deployment uses:

```text
docker/compose.full-https.yml
docker/torus/torus-https.yaml
docker/certs/cert.pem
docker/certs/key.pem
```

The Compose file mounts the certificate and private key into the Torus container.

Customize the Docker deployment assets when using a different certificate, listener, or backend topology.

### Start

```bash
docker compose -f docker/compose.full-https.yml up --build -d
```

### Verify

```bash
docker compose -f docker/compose.full-https.yml ps
```

Then:

```bash
curl -k https://localhost:8443/readyz
curl -k https://localhost:8443/api/hello
```

Use the actual listener configured by `docker/torus/torus-https.yaml`.

### Logs

```bash
docker compose -f docker/compose.full-https.yml logs torus
```

Follow:

```bash
docker compose -f docker/compose.full-https.yml logs -f torus
```

### Stop

```bash
docker compose -f docker/compose.full-https.yml down
```

## Docker configuration

The Compose files reference deployment-specific configuration under `docker/`.

For example, the full HTTP stack mounts:

```text
docker/torus/torus.yaml
```

as the Torus configuration inside the container.

The HTTPS deployment mounts:

```text
docker/torus/torus-https.yaml
```

and the TLS files under:

```text
docker/certs/
```

Customize these files when changing the Docker deployment.

For configuration semantics, see the [Configuration Guide](../configuration/README.md).

## Observability

The full stack includes Prometheus and Grafana.

Prometheus is exposed on:

```text
http://localhost:9090
```

Grafana is exposed on:

```text
http://localhost:3000
```

The repository's Grafana configuration and dashboards are under:

```text
docker/grafana/
```

The repository's Prometheus configurations are under:

```text
docker/prometheus/
```

### Verify the observability stack

Verify that Torus exposes metrics:

```bash
curl http://localhost:8080/metrics
```

For HTTPS:

```bash
curl -k https://localhost:8443/metrics
```

Then verify Prometheus:

```bash
http://localhost:9090
```

Navigate to:

```text
Status → Targets
```

The Torus target should report UP.

Finally, open Grafana:

```bash
http://localhost:3000
```

After generating traffic through Torus, verify that the provisioned dashboards contain current Prometheus data.

## Development limitations

The repository's full Docker Compose configurations are not production-ready by default.

They include development-oriented assets such as:

- mock backend services
- source-built images
- repository-managed development TLS material
- development Grafana credentials
- single-machine networking assumptions

For a production-oriented Linux VM deployment, see [Production Deployment](./production.md).

## Related documentation

- [Configuration](../configuration/README.md)
- [Native Deployment](./native.md)
- [Linux/systemd Deployment](./systemd.md)
- [Production Deployment](./production.md)
