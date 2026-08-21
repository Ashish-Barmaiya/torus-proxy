# Configuration Reference

This document provides the complete top-level configuration reference.

For conceptual explanations and examples, see the individual configuration pages.

## Complete structure

```yaml
apiVersion: v2

server:
  addr: ":8080"

health:
  interval_ms: 5000
  timeout_ms: 2000
  path: /health

observability:
  enabled: true

tls:
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
  min_version: "1.2"

services:
  - name: api
    upstreams:
      - http://localhost:3001
      - http://localhost:3002

routes:
  - path: /api
    service: api
```

## `apiVersion`

| Property | Value |
|---|---|
| Type | string |
| Required | Yes |
| Current version | `v2` |

## `server`

Configures the client-facing listener.

### `server.addr`

| Property | Value |
|---|---|
| Type | string |
| Required | Yes |

Example:

```yaml
server:
  addr: ":8080"
```

## `health`

Configures active upstream health checks.

### `health.interval_ms`

| Property | Value |
|---|---|
| Type | integer |
| Required | Yes |

Interval between health checks.

### `health.timeout_ms`

| Property | Value |
|---|---|
| Type | integer |
| Required | Yes |

Timeout for an individual health-check request.

### `health.path`

| Property | Value |
|---|---|
| Type | string |
| Required | Yes |

Path requested on upstream backends during health checks.

Example:

```yaml
health:
  interval_ms: 5000
  timeout_ms: 2000
  path: /health
```

## `observability`

Controls Prometheus metrics.

### `observability.enabled`

| Property | Value |
|---|---|
| Type | boolean |
| Required | No |

Example:

```yaml
observability:
  enabled: true
```

## `tls`

Optional TLS termination configuration.

### `tls.cert_file`

| Property | Value |
|---|---|
| Type | string |
| Required when `tls` is present | Yes |

Path to the X.509 certificate.

### `tls.key_file`

| Property | Value |
|---|---|
| Type | string |
| Required when `tls` is present | Yes |

Path to the corresponding private key.

### `tls.min_version`

| Property | Value |
|---|---|
| Type | string |
| Required | No |

Supported values:

```text
1.2
1.3
```

## `services`

Defines logical services and their backend pools.

### `services[].name`

| Property | Value |
|---|---|
| Type | string |
| Required | Yes |

Unique logical service name.

### `services[].upstreams`

| Property | Value |
|---|---|
| Type | list of strings |
| Required | Yes |
| Minimum | 1 |

Each upstream must use a supported HTTP or HTTPS URL and specify a valid port.

## `routes`

Defines mappings from request path prefixes to logical services.

### `routes[].path`

| Property | Value |
|---|---|
| Type | string |
| Required | Yes |

Path prefix matched against requests.

### `routes[].service`

| Property | Value |
|---|---|
| Type | string |
| Required | Yes |

Name of a configured service.

## Configuration validation

Torus validates the configuration before constructing a runtime.

Validation includes relationships between sections, including:

```text
routes → services → upstreams
```

An invalid configuration is rejected.

During configuration reload, the currently active runtime remains in use when the new configuration cannot be accepted.

## Deployment-specific paths

Configuration values such as certificate paths depend on the deployment model.

For local execution:

```yaml
tls:
  cert_file: "configs/cert.pem"
  key_file: "configs/key.pem"
```

For the systemd deployment:

```yaml
tls:
  cert_file: "/etc/torus/tls/cert.pem"
  key_file: "/etc/torus/tls/key.pem"
```

See the deployment documentation for deployment-specific requirements.

## Related documentation

- [Configuration Overview](./README.md)
- [API Version](./api-version.md)
- [Server](./server.md)
- [Services](./services.md)
- [Routing](./routing.md)
- [Health](./health.md)
- [TLS](./tls.md)
- [Observability](./observability.md)
- [Configuration Reload](./configuration-reload.md)
