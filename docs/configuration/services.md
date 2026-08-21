# Services

A service is the logical boundary between request routing and backend servers.

Routes reference services. Services own the upstream backend pool that can receive traffic.

```text
Route
  ↓
Service
  ↓
Upstream Pool
  ├── Backend A
  ├── Backend B
  └── Backend C
```

## Service definition

Services are declared under the top-level `services` section.

```yaml
services:
  - name: api
    upstreams:
      - http://localhost:3001
      - http://localhost:3002
```

Each service contains:

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Unique logical name referenced by routes. |
| `upstreams` | Yes | Backend URLs belonging to the service. |

A service must contain at least one upstream.

## `name`

The `name` field identifies the logical service.

```yaml
services:
  - name: api
    upstreams:
      - http://localhost:3001
```

Routes reference this name:

```yaml
routes:
  - path: /api
    service: api
```

Service names should be unique, descriptive, stable, and meaningful within the application architecture.

## `upstreams`

The `upstreams` field defines the backend servers belonging to a service.

```yaml
services:
  - name: api
    upstreams:
      - http://localhost:3001
      - http://localhost:3002
      - http://localhost:3003
```

These backends form one upstream pool.

Torus selects a backend from the pool when forwarding a request.

## Upstream URL format

Upstream URLs use HTTP or HTTPS URLs and must contain a hostname and explicit port.

Valid examples:

```yaml
upstreams:
  - http://localhost:3001
  - http://10.0.0.12:8080
  - https://backend.example.com:8443
```

The following is invalid because the port is missing:

```yaml
upstreams:
  - http://localhost
```

The following is invalid because the scheme is unsupported:

```yaml
upstreams:
  - ftp://localhost:3001
```

Torus supports the `http` and `https` schemes for upstreams.

## Multiple upstreams

A service can contain multiple upstreams:

```yaml
services:
  - name: api
    upstreams:
      - http://localhost:3001
      - http://localhost:3002
      - http://localhost:3003
```

The current load-balancing strategy is round-robin.

```text
Request 1 → :3001
Request 2 → :3002
Request 3 → :3003
Request 4 → :3001
...
```

Only healthy backends are eligible for selection.

## Health-aware backend selection

Torus continuously tracks the health of configured upstreams.

```text
Service: api

├── :3001  ✓ healthy
├── :3002  ✗ unhealthy
└── :3003  ✓ healthy
```

An unhealthy backend is excluded from normal backend selection.

When a later health check succeeds, the backend becomes eligible again.

See [Health](./health.md).

## Connecting services to routes

A service becomes reachable through a route.

```yaml
services:
  - name: api
    upstreams:
      - http://localhost:3001
      - http://localhost:3002

routes:
  - path: /api
    service: api
```

A request such as `/api/users` follows:

```text
/api/users
    ↓
/api route
    ↓
api service
    ↓
healthy upstream
    ↓
backend
```

## Reusing a service

Multiple routes can reference the same service.

```yaml
services:
  - name: api
    upstreams:
      - http://localhost:3001
      - http://localhost:3002

routes:
  - path: /api
    service: api

  - path: /v1
    service: api
```

Both routes use the same service and therefore the same upstream pool.

## Validation

Torus validates services before constructing a runtime.

A valid service must:

- define a non-empty name
- use a unique name
- define at least one upstream
- contain valid upstream URLs
- use a supported URL scheme
- use a valid upstream port

A route referencing an undefined service is also invalid.

## Invalid examples

### Empty service name

```yaml
services:
  - name: ""
    upstreams:
      - http://localhost:3001
```

### Duplicate service names

```yaml
services:
  - name: api
    upstreams:
      - http://localhost:3001

  - name: api
    upstreams:
      - http://localhost:3002
```

### No upstreams

```yaml
services:
  - name: api
    upstreams: []
```

### Unknown service reference

```yaml
routes:
  - path: /api
    service: payments
```

when `payments` is not defined under `services`.

Invalid configuration is rejected before it becomes part of the active runtime.

## Complete example

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

services:
  - name: api
    upstreams:
      - http://localhost:3001
      - http://localhost:3002

routes:
  - path: /api
    service: api
```

## Design rationale

The service abstraction separates routing from backend management.

```text
Request
   ↓
Router
   ↓
Service
   ↓
Load Balancer
   ↓
Healthy Backend
   ↓
Reverse Proxy
```

A route determines which service should handle a request.

A service determines which backend pool can handle that request.

## Related documentation

- [Routing](./routing.md)
- [Health](./health.md)
- [Configuration Reload](./configuration-reload.md)
- [Configuration Reference](./reference.md)
