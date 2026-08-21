# Torus Configuration

Torus is configured through a declarative YAML configuration file.

The configuration defines the desired state of the proxy, including its listening address, request routing, backend services, health checks, TLS termination, and observability settings.

Torus reads, parses, and validates the configuration before constructing a runtime from it. During normal request processing, the active runtime is used rather than repeatedly reading the configuration file.

## Configuration lifecycle

```text
Configuration File
        │
        ▼
     Read YAML
        │
        ▼
       Parse
        │
        ▼
      Validate
        │
        ▼
   Build Runtime
        │
        ▼
   Serve Requests
```

If the configuration is invalid, Torus does not construct a new runtime from it.

During a configuration reload, the existing runtime remains active when the new configuration cannot be validated or used to construct a valid runtime.

## Configuration structure

A Torus configuration consists of the following top-level sections:

| Section | Required | Purpose |
|---|---:|---|
| `apiVersion` | Yes | Identifies the configuration schema version. |
| `server` | Yes | Configures the address on which Torus listens for client connections. |
| `services` | Yes | Defines logical services and their upstream backend pools. |
| `routes` | Yes | Maps request path prefixes to logical services. |
| `health` | Yes | Configures active health checks for upstream backends. |
| `tls` | No | Enables TLS termination and configures certificates and TLS behavior. |
| `observability` | No | Enables or disables Prometheus metrics. |

## Creating a configuration

Generic Torus configuration examples are provided under:

```text
configs/
```

Docker Compose deployments use deployment-specific configuration and supporting assets under:

```text
docker/
```

The repository examples are intended for development, testing, and evaluation. Production deployments should use a configuration customized for the actual deployment environment.

## Configuration sections

- [API Version](./api-version.md)
- [Server](./server.md)
- [Services](./services.md)
- [Routing](./routing.md)
- [Health](./health.md)
- [TLS](./tls.md)
- [Observability](./observability.md)
- [Configuration Reload](./configuration-reload.md)
- [Configuration Reference](./reference.md)

## Configuration and deployment

Configuration and deployment are separate concerns.

Configuration defines:

```text
What should Torus do?
```

Deployment defines:

```text
How should Torus be installed and run?
```

Deployment-specific filesystem paths and operational procedures are documented separately in the deployment documentation.
