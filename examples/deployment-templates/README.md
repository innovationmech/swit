# Deployment Configuration Templates

This directory provides ready-to-use configuration templates for common deployment scenarios. Copy and adapt these files as needed.

Contents:

- `swit.dev.yaml`: Development overlay for `swit-serve`
- `swit.prod.yaml`: Production overlay with security/performance tuning for `swit-serve`
- `switauth.dev.yaml`: Development overlay for `swit-auth`
- `switauth.prod.yaml`: Production overlay with security settings for `swit-auth`
- `swit.docker.yaml`: Container-friendly base settings for running in Docker/Compose
- `docker-compose.override.yml`: Service-level override for local Compose
- `docker-compose.prod.override.yml`: Production-leaning Compose override with secrets and resource hints

How to use (Docker Compose):

1) Copy `swit.docker.yaml` next to the service binaries inside container and mount as read-only:
   - Example: `- ./swit.docker.yaml:/app/config/swit.yaml:ro`

2) Place `docker-compose.override.yml` or `docker-compose.prod.override.yml` alongside your base Compose file to override service configuration.

3) Secrets via files (recommended):
   - Compose secret → mount to `/run/secrets/...`
   - Point env var with `_FILE` suffix to that path, e.g., `SWIT_DB_PASSWORD_FILE=/run/secrets/db_password`

How to use (Kubernetes/Helm):

- Use these YAML overlays as references for `ConfigMap` content or Helm `values-*.yaml` files. See `deployments/helm/values-dev.yaml` and `deployments/helm/values-prod.yaml`.


