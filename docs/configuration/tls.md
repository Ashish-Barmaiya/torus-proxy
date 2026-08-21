# TLS

Torus can terminate HTTPS connections by loading an X.509 certificate and private key during startup.

TLS is optional.

If the `tls` section is omitted, Torus serves plain HTTP.

If the `tls` section is present, Torus enables TLS termination on the configured `server.addr`.

```yaml
server:
  addr: ":8443"

tls:
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
  min_version: "1.2"
```

## TLS termination

Torus performs TLS termination.

```text
Client
   │
   │ HTTPS
   ▼
 Torus
   │
   │ HTTP or HTTPS
   ▼
Upstream Backend
```

TLS between Torus and an upstream backend is independent of client-facing TLS and is determined by the upstream URL.

## Configuration

| Field | Type | Required | Description |
|---|---|---:|---|
| `cert_file` | string | Yes | Path to the X.509 certificate. |
| `key_file` | string | Yes | Path to the corresponding private key. |
| `min_version` | string | No | Minimum accepted TLS version. |

## Certificate and private key

Torus requires both:

```yaml
tls:
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
```

The certificate and private key must form a valid pair.

If either file cannot be loaded, or the pair is invalid, the configuration cannot be used to construct the runtime.

## Minimum TLS version

The current configuration model supports:

| Value | Meaning |
|---|---|
| `"1.2"` | TLS 1.2 and newer |
| `"1.3"` | TLS 1.3 only |

Example:

```yaml
tls:
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
  min_version: "1.3"
```

Unsupported values are rejected during configuration validation.

## HTTP configuration

To run Torus without TLS, omit the `tls` section.

```yaml
server:
  addr: ":8080"
```

## HTTPS configuration

```yaml
server:
  addr: ":8443"

tls:
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
  min_version: "1.2"
```

The port itself does not enable TLS. The presence of the `tls` section does.

## Certificate paths and deployment

Certificate paths are configuration values, but their appropriate value depends on the deployment model.

For local execution, a configuration may reference repository files:

```yaml
tls:
  cert_file: "configs/cert.pem"
  key_file: "configs/key.pem"
```

For the systemd deployment, the installed configuration uses installer-managed locations:

```yaml
tls:
  cert_file: "/etc/torus/tls/cert.pem"
  key_file: "/etc/torus/tls/key.pem"
```

The systemd installer asks for the source certificate and key files separately and installs them into the managed locations.

## Development certificates

A self-signed certificate can be generated for local testing:

```bash
openssl req   -x509   -newkey rsa:4096   -nodes   -keyout configs/key.pem   -out configs/cert.pem   -days 365
```

Self-signed certificates are suitable for development and testing only.

Do not use repository development certificates as production credentials.

## Production recommendations

For production deployments:

- use a certificate issued by a trusted Certificate Authority;
- keep private keys outside source control;
- restrict access to private-key files;
- use the strongest TLS version compatible with required clients;
- establish a certificate renewal and rotation process.

## Validation

Torus validates the TLS configuration before constructing a runtime.

If TLS configuration is invalid during startup, Torus does not start successfully.

If TLS configuration becomes invalid during reload, the existing runtime remains active.

## Related documentation

- [Server](./server.md)
- [Configuration Reload](./configuration-reload.md)
- [Configuration Reference](./reference.md)
