# Routing

Routing determines which logical service handles an incoming request.

A route maps a request path to a service:

```yaml
routes:
  - path: /api
    service: api
```

Torus uses longest-prefix matching to select the most specific matching route.

Routing determines the target service. Individual backend selection is handled separately by the service's upstream pool.

## Route structure

| Field | Required | Description |
|---|---|---|
| `path` | Yes | URL path prefix matched against incoming requests. |
| `service` | Yes | Name of the service that handles matching requests. |

Example:

```yaml
routes:
  - path: /api
    service: api
```

The referenced service must exist in `services`.

## Longest-prefix matching

When multiple routes match a request, Torus selects the longest valid matching prefix.

```yaml
routes:
  - path: /
    service: web

  - path: /api
    service: api

  - path: /api/v1
    service: api-v1
```

For `/api/v1/users`, `/api/v1` is the longest matching prefix.

| Request | Route | Service |
|---|---|---|
| `/` | `/` | `web` |
| `/about` | `/` | `web` |
| `/api` | `/api` | `api` |
| `/api/users` | `/api` | `api` |
| `/api/v1` | `/api/v1` | `api-v1` |
| `/api/v1/users` | `/api/v1` | `api-v1` |

## Path boundaries

Route matching respects path-segment boundaries.

Given:

```yaml
routes:
  - path: /api
    service: api
```

the route matches:

```text
/api
/api/
/api/users
/api/v1/users
```

It does not match:

```text
/apiv1
/api-test
/apix
```

## Multiple routes

```yaml
routes:
  - path: /api
    service: api

  - path: /admin
    service: admin

  - path: /static
    service: static
```

Each request is evaluated against the configured routes.

## Root route

A route at `/` can be used as a catch-all:

```yaml
routes:
  - path: /
    service: web
```

More specific routes take precedence.

```text
/                  → web
/about             → web
/api/users         → api
/admin/settings    → admin
```

## Service separation

Routing selects a service, not an individual backend.

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

## Validation

Torus validates routing before constructing a runtime.

A route must:

- define a non-empty path
- define a service
- use a unique path
- reference an existing service

During configuration reload, the existing runtime remains active if the new routing configuration fails validation.

## Invalid examples

### Empty path

```yaml
routes:
  - path: ""
    service: api
```

### Duplicate paths

```yaml
routes:
  - path: /api
    service: api

  - path: /api
    service: admin
```

### Unknown service

```yaml
routes:
  - path: /api
    service: payments
```

when `payments` does not exist under `services`.

## No matching route

If no configured route matches an incoming request, Torus does not select a backend and the request is rejected with:

```text
404 Not Found
```

## Related documentation

- [Services](./services.md)
- [Health](./health.md)
- [Configuration Reload](./configuration-reload.md)
- [Configuration Reference](./reference.md)
