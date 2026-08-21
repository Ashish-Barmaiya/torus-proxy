# Configuration Reload

Torus supports configuration reloads without requiring a process restart.

The configuration file is monitored for changes. When a change is detected, Torus reads and validates the new configuration and attempts to construct a new runtime generation.

A valid new generation replaces the active runtime.

```text
Configuration Change
        ↓
    Read YAML
        ↓
       Parse
        ↓
      Validate
        ↓
 Build New Runtime
        ↓
 Atomic Runtime Swap
        ↓
New Requests → New Runtime
```

## Runtime generations

Each accepted configuration produces a runtime generation.

A runtime generation contains the state required to serve requests, including routing, services, upstream pools, backend health state, observability configuration, and other runtime-owned state.

The active runtime is immutable after construction.

## Atomic replacement

When a new configuration is accepted:

```text
Current Runtime
      │
      │ atomic replacement
      ▼
New Runtime
```

Requests that begin after the replacement use the new runtime.

Existing requests continue according to the runtime's lifecycle rules.

## Invalid configuration reload

A changed configuration is not automatically accepted.

Torus must be able to parse, validate, and construct a valid runtime from the new configuration.

If any step fails:

```text
Configuration Change
        ↓
    Parse / Validate
        ↓
      Failure
        ↓
 Keep Existing Runtime
```

For example, an invalid upstream URL can cause the new runtime build to fail while the previously active runtime continues serving requests.

## Reload versus restart

### Configuration reload

```text
Edit configuration
      ↓
Torus detects change
      ↓
New runtime constructed
      ↓
Atomic replacement
```

### Process restart

```text
systemctl restart torus
      ↓
Torus process stops
      ↓
Process starts again
      ↓
Initial runtime constructed
```

Configuration changes should use Torus's reload mechanism when possible rather than restarting the process unnecessarily.

## Configuration and deployment

The deployment mechanism determines where the configuration file is located.

For example, the systemd deployment installs the active configuration at:

```text
/etc/torus/torus.yaml
```

Torus then monitors and reloads that configuration according to its runtime lifecycle.

## Failure isolation

The intended behavior is:

```text
Active Runtime
      │
      ├── New configuration valid
      │         ↓
      │   New Runtime
      │         ↓
      │   Atomic Swap
      │
      └── New configuration invalid
                ↓
         Keep Active Runtime
```

This prevents an invalid configuration from replacing known-good runtime state.

## Related documentation

- [Configuration Overview](./README.md)
- [Server](./server.md)
- [Services](./services.md)
- [Routing](./routing.md)
- [Health](./health.md)
- [TLS](./tls.md)
- [Observability](./observability.md)
- [Runtime Lifecycle](./../engineering/architecture/runtime-lifecycle.md)
