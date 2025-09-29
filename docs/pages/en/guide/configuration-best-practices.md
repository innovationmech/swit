---
title: Configuration Best Practices
outline: deep
---

# Configuration Best Practices

This guide outlines recommended practices for managing configuration in Swit-based services. It complements the generated reference and focuses on security, performance, maintainability, inheritance/overrides, environment variable mapping, and validation/testing.

## Principles

- Keep configuration declarative, versioned, and environment-specific
- Prefer secure defaults, validate early, and fail fast on critical misconfigurations
- Separate secrets from configs; support `_FILE` indirection for secrets
- Provide safe overrides via hierarchy and environment variables

## Configuration Layers and Precedence

Low → High precedence:

1. Defaults (code)
2. Base file (e.g., `swit.yaml`)
3. Environment file (e.g., `swit.production.yaml`)
4. Override file (e.g., `swit.override.yaml`)
5. Environment variables (`SWIT_...`)

See also: `/en/guide/configuration-reference` for exact keys and env mapping.

## Environment Variable Mapping

- Prefix: `SWIT_`
- Mapping rule: config key `a.b.c` → env `SWIT_A_B_C`
- Dynamic segments: `messaging.brokers.<name>.endpoints` → `SWIT_MESSAGING_BROKERS_<NAME>_ENDPOINTS`
- Secret indirection: `VAR_FILE` pattern is supported to load value from a file

Example:

```bash
export SWIT_SERVICE_NAME="my-service"
export SWIT_HTTP_PORT="8080"
export SWIT_GRPC_TLS_ENABLED="true"
export SWIT_MESSAGING_BROKERS_KAFKA_TYPE="kafka"
export SWIT_MESSAGING_BROKERS_KAFKA_ENDPOINTS="broker-1:9092,broker-2:9092"
```

## Inheritance and Overrides

Use base + per-environment files and keep `swit.override.yaml` for local/operator overrides that are not committed.

```yaml
# swit.yaml (base)
service_name: my-service
http:
  enabled: true
  port: "8080"

# swit.production.yaml
http:
  read_timeout: "30s"
  write_timeout: "30s"

# swit.override.yaml (local)
http:
  port: "9090"
```

## Validation and Testing

- Validate on startup; prefer clear error messages. Common checks include:
  - Required fields present (e.g., `service_name`, `http.port` when enabled)
  - Value ranges and enums (e.g., `discovery.failure_mode`)
  - Security invariants (e.g., no CORS `*` with credentials)
- Add table-driven tests for config parsing/validation

Example validation snippet:

```go
cfg := server.NewServerConfig()
// ... load values ...
if err := cfg.Validate(); err != nil {
    log.Fatalf("configuration invalid: %v", err)
}
```

## Security Best Practices

- Do not commit secrets; use env vars or secret files (`*_FILE`)
- Enable TLS in production; verify certificates and set `server_name`
- Avoid `skip_verify: true` except in controlled testing
- For CORS, never use wildcard origins when `allow_credentials: true`

## Performance and Reliability

- Tune transport timeouts and message sizes to match workloads
- For messaging:
  - Set `endpoints`, `type` per broker
  - Configure retry/backoff and pooling for producers/consumers
  - Enable monitoring/metrics and health checks

## Observability and Operations

- Enable metrics (`prometheus`) and tracing (`tracing`) where applicable
- Use health check endpoints and readiness probes
- Prefer immutable config in containers; inject overrides via env/ConfigMap/Secrets

## Deployment Checklists

- Security
  - [ ] TLS enabled with valid certs in production
  - [ ] No wildcard CORS with credentials
  - [ ] Secrets sourced from env/secret files only

- Reliability
  - [ ] Reasonable timeouts and retry policies
  - [ ] Health checks enabled and scraped

- Configuration Hygiene
  - [ ] Environment-specific files are used
  - [ ] Overrides documented and auditable
  - [ ] Startup validation passes locally and in CI

## Examples

Minimal production-ready HTTP section:

```yaml
http:
  enabled: true
  address: ":8080"
  read_timeout: "30s"
  write_timeout: "30s"
  middleware:
    enable_cors: true
    cors:
      allow_origins:
        - "https://app.example.com"
      allow_credentials: true
```

Kafka broker example:

```yaml
messaging:
  default_broker: kafka
  brokers:
    kafka:
      type: kafka
      endpoints: ["broker-1:9092", "broker-2:9092"]
      retry:
        max_attempts: 5
        initial_delay: "1s"
      tls:
        enabled: true
        ca_file: "/etc/ssl/certs/ca.crt"
```

## See Also

- Configuration Guide: [/en/guide/configuration](/en/guide/configuration)
- Generated Reference: [/en/guide/configuration-reference](/en/guide/configuration-reference)
- Deployment Examples: [/en/guide/deployment-examples](/en/guide/deployment-examples)


