---
title: Deployment-Specific Configuration Examples
outline: deep
---

# Deployment-Specific Configuration Examples

This guide provides production-oriented configuration examples for running Swit services across different environments, with a focus on Docker and Kubernetes. It also shows environment-specific overrides and security/performance best practices.

Useful reference files:

- Example configs: `examples/deployment-config-examples.yaml`
- Templates: `examples/deployment-templates/`
- Service configs: `swit.yaml`, `switauth.yaml`
- K8s templates: `deployments/k8s/*.yaml`
- Docker compose: `deployments/docker/docker-compose.yml`

## Environment Strategy

We recommend a layered configuration approach:

1. Base defaults checked into VCS (`swit.yaml` / `switauth.yaml`)
2. Per-environment overlays (`swit.dev.yaml`, `swit.prod.yaml`)
3. Secrets via environment variables or secret managers

See consolidated examples in `examples/deployment-config-examples.yaml`.

## Docker Examples

### 1) Container-Friendly Settings

```yaml
# swit.docker.yaml (excerpt)
server:
  http:
    enabled: true
    port: "9000"

messaging:
  enabled: true
  default_broker: "local"
  connection:
    timeout: "20s"
    retry_interval: "3s"
  security:
    enable_encryption: false

tracing:
  enabled: true
  exporter:
    type: "jaeger"
    endpoint: "http://jaeger:14268/api/traces" # container name on the same network
```

### 2) Docker Compose Service

```yaml
# docker-compose.override.yml (service layer)
services:
  swit-serve:
    image: swit-serve:latest
    ports:
      - "9000:9000"
    environment:
      SWIT_TRACING_EXPORTER_ENDPOINT: http://jaeger:14268/api/traces
      SWIT_MESSAGING_ENABLED: "true"
    volumes:
      - ./swit.docker.yaml:/app/config/swit.yaml:ro
    depends_on:
      - jaeger
```

#### 2.1) Production-leaning Compose Override

See `examples/deployment-templates/docker-compose.prod.override.yml` for:

- Resource hints (CPU/memory limits/reservations)
- File-based secrets via `_FILE` environment variables
- Jaeger collector endpoint for production networks

### 3) Security & Performance Tips

- TLS termination at ingress/reverse-proxy (Caddy/NGINX/Traefik)
- Restrict CORS origins; avoid wildcards in production
- Set CPU/memory limits in orchestrator (Compose profiles or K8s)
- Use `GODEBUG=madvdontneed=1` for memory reuse under pressure

## Kubernetes Examples

### 1) ConfigMap and Secret

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: swit-config
data:
  swit.yaml: |
    server:
      http:
        enabled: true
        port: "9000"
    messaging:
      enabled: true
      default_broker: "local"
---
apiVersion: v1
kind: Secret
metadata:
  name: swit-secrets
type: Opaque
stringData:
  SWIT_DB_PASSWORD: "change-me"
```

### 2) Deployment with Probes and Resources

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: swit-serve
spec:
  replicas: 3
  selector:
    matchLabels:
      app: swit-serve
  template:
    metadata:
      labels:
        app: swit-serve
    spec:
      containers:
        - name: swit-serve
          image: swit-serve:latest
          ports:
            - containerPort: 9000
          envFrom:
            - secretRef: { name: swit-secrets }
          volumeMounts:
            - name: config
              mountPath: /app/config
          readinessProbe:
            httpGet: { path: /health, port: 9000 }
            initialDelaySeconds: 10
            periodSeconds: 10
          livenessProbe:
            httpGet: { path: /health, port: 9000 }
            initialDelaySeconds: 30
            periodSeconds: 15
          resources:
            requests:
              cpu: "100m"
              memory: "128Mi"
            limits:
              cpu: "500m"
              memory: "512Mi"
      volumes:
        - name: config
          configMap:
            name: swit-config
```

### 3) Production Checklist

- Configure PodDisruptionBudget and HPA
- Enable TLS (mTLS if applicable) and service mesh policies if used
- Lock down network policies per namespace
- Define SLOs and alerts (readiness latency, error ratio)

## Environment Overrides

Example overlay pairs included in `examples/deployment-config-examples.yaml`:

- `swit.dev.yaml` vs `swit.prod.yaml`
- `switauth.dev.yaml` vs `switauth.prod.yaml`

Key differences typically include:

- Log verbosity, sampling rates, and tracing exporters
- Broker endpoints (localhost/docker network vs managed services)
- TLS and authentication settings
- Connection pools and timeouts

## Where to Go Next

- Detailed config reference: `/en/guide/configuration-reference`
- Capabilities matrix: `/en/reference/capabilities`
- Adapter switching guide: `/en/guide/messaging-adapter-switching`


