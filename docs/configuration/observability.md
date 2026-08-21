# Observability

The `observability` section controls whether Torus exposes Prometheus metrics.

```yaml
observability:
  enabled: true
```

When enabled, Torus exposes:

```text
/metrics
```

which can be scraped by Prometheus.

## Configuration

| Field | Type | Required | Description |
|---|---|---:|---|
| `enabled` | boolean | No | Enables or disables Prometheus metrics. |

## Enabling observability

```yaml
observability:
  enabled: true
```

When enabled, Torus exposes the Prometheus metrics endpoint on the configured Torus listener.

## Disabling observability

```yaml
observability:
  enabled: false
```

When disabled, Torus does not register the `/metrics` endpoint.

Normal proxy operation, health checks, and logging continue independently.

## Prometheus integration

A Prometheus configuration can scrape Torus:

```yaml
scrape_configs:
  - job_name: torus
    static_configs:
      - targets:
          - host.docker.internal:8080
```

The target must match the listener used by the Torus deployment.

## Local Prometheus and Grafana

The repository includes Docker-based Prometheus and Grafana configuration under:

```text
docker/
```

These files provide a convenient local observability stack.

The monitoring stack is separate from the Torus configuration itself.

## Verifying metrics

When observability is enabled:

```bash
curl http://localhost:8080/metrics
```

or use the address and port configured for Torus.

The endpoint should return Prometheus exposition-format metrics.

## Observability pipeline

```text
Torus
  │
  │ /metrics
  ▼
Prometheus
  │
  ▼
Grafana
```

Prometheus collects metrics from Torus, while Grafana visualizes the collected data.

## Configuration and deployment

The `observability` section controls the metrics endpoint exposed by Torus.

It does not install or configure Prometheus or Grafana.

Those are deployment concerns documented separately.

## Related documentation

- [Server](./server.md)
- [Configuration Reload](./configuration-reload.md)
- [Configuration Reference](./reference.md)
- [Deployment](../deployment/README.md)
