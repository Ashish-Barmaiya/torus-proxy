# Health

The `health` section configures active health checks for upstream backends.

Torus periodically probes configured upstreams and tracks their health. Backend health is used during load balancing so unhealthy upstreams are excluded from normal request selection.

```yaml
health:
  interval_ms: 5000
  timeout_ms: 2000
  path: /health
```

## Configuration

| Field | Type | Required | Description |
|---|---|---:|---|
| `interval_ms` | integer | Yes | Interval between health checks, in milliseconds. |
| `timeout_ms` | integer | Yes | Maximum time allowed for an individual health-check request, in milliseconds. |
| `path` | string | Yes | HTTP path requested when checking an upstream. |

## `interval_ms`

Defines how frequently Torus checks backend health.

```yaml
health:
  interval_ms: 5000
```

A value of `5000` configures a five-second health-check interval.

## `timeout_ms`

Defines the timeout for an individual health-check request.

```yaml
health:
  timeout_ms: 2000
```

A value of `2000` allows up to two seconds for the health-check request.

## `path`

Defines the HTTP path requested from each upstream during a health check.

```yaml
health:
  path: /health
```

For an upstream:

```text
http://localhost:3001
```

Torus checks the configured health path on that upstream.

## Backend health and load balancing

```text
Health Checker
      │
      ├── Backend A → healthy
      ├── Backend B → unhealthy
      └── Backend C → healthy
                    │
                    ▼
                Load Balancer
                    │
             ┌──────┴──────┐
             ▼             ▼
         Backend A      Backend C
```

An unhealthy backend is excluded from normal load-balancing decisions.

When the backend becomes healthy again, it can become eligible for requests again.

## Example

```yaml
health:
  interval_ms: 5000
  timeout_ms: 2000
  path: /health
```

with:

```yaml
services:
  - name: api
    upstreams:
      - http://localhost:3001
      - http://localhost:3002
```

## Validation

Health configuration is validated before Torus constructs a runtime.

Invalid health configuration prevents the corresponding configuration from becoming active.

During configuration reload, the existing runtime remains active if the new health configuration cannot be accepted.

## Operational considerations

The interval and timeout should reflect the behavior of the backend services.

A very short interval can increase health-check traffic.

A very long interval can delay detection of an unhealthy backend.

The timeout should be long enough for a healthy backend to respond reliably but short enough to detect unavailable backends promptly.

## Related documentation

- [Services](./services.md)
- [Routing](./routing.md)
- [Configuration Reload](./configuration-reload.md)
- [Configuration Reference](./reference.md)
