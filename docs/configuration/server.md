# Server

The `server` section defines the network address on which Torus accepts incoming client connections.

```yaml
server:
  addr: ":8080"
```

Every Torus configuration must define a valid server address.

## Configuration

| Field | Type | Required | Description |
|---|---|---|---|
| `addr` | string | Yes | Network address on which Torus listens for client connections. |

## Listening address

The address follows Go's standard `host:port` format.

To listen on all available interfaces:

```yaml
server:
  addr: ":8080"
```

To bind to a specific interface:

```yaml
server:
  addr: "127.0.0.1:8080"
```

To use a different port:

```yaml
server:
  addr: ":18080"
```

The listening port is independent of TLS. TLS is enabled separately through the `tls` section.

## Network binding

A binding such as:

```yaml
server:
  addr: ":8080"
```

does not restrict the listener to a specific local interface.

A binding such as:

```yaml
server:
  addr: "127.0.0.1:8080"
```

restricts the listener to the local loopback interface.

The appropriate binding depends on the deployment architecture.

## Validation

The server address is validated while Torus loads the configuration.

An empty address is invalid:

```yaml
server:
  addr: ""
```

If validation fails during startup, Torus does not construct the runtime.

If validation fails during configuration reload, the currently active runtime remains unchanged.

## Examples

### HTTP

```yaml
server:
  addr: ":8080"
```

### HTTPS

```yaml
server:
  addr: ":8443"

tls:
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
  min_version: "1.2"
```

### Local-only listener

```yaml
server:
  addr: "127.0.0.1:8080"
```

### Custom listener

```yaml
server:
  addr: ":18080"
```

## Best practices

- Bind to a specific interface when the proxy should not be exposed on every interface.
- Use a dedicated port appropriate for the deployment environment.
- Treat the listening address as deployment-specific configuration.
- When changing the address, update clients, monitoring, firewall rules, and deployment configuration accordingly.

## Related documentation

- [TLS](./tls.md)
- [Health](./health.md)
- [Configuration Reload](./configuration-reload.md)
- [Configuration Reference](./reference.md)
