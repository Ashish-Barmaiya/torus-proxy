# API Version

Every Torus configuration begins with an API version declaration.

```yaml
apiVersion: v2
```

The API version identifies the configuration schema used by the file.

Torus checks the declared version before constructing the runtime. A configuration that declares an unsupported version is rejected.

## Configuration

| Field | Type | Required | Description |
|---|---|---|---|
| `apiVersion` | string | Yes | Identifies the configuration schema version. |

## Why version the configuration?

Configuration formats evolve as Torus evolves.

Future releases may introduce new fields, change existing behavior, remove deprecated fields, or change validation rules.

An explicit schema version allows Torus to determine how a configuration should be interpreted.

## Supported versions

| Version | Status |
|---|---|
| `v2` | Supported |

Configurations using an unsupported version are rejected.

## Validation

API version validation occurs before Torus constructs a runtime.

```text
Read Configuration
        ↓
Check apiVersion
        ↓
Supported?---+
   │         │
  Yes        No
   ↓         ↓
Continue   Validation Error
```

During startup, an invalid version prevents Torus from starting.

During configuration reload, an invalid version prevents the new configuration from becoming active. The existing runtime remains in service.

## Best practices

- Always specify `apiVersion` explicitly.
- Use the version supported by the installed Torus release.
- Review migration guidance before changing the configuration version after a Torus upgrade.
- Do not assume a configuration written for a newer schema is compatible with an older Torus release.

## Related documentation

- [Configuration Overview](./README.md)
- [Server](./server.md)
- [Configuration Reload](./configuration-reload.md)
- [Configuration Reference](./reference.md)
