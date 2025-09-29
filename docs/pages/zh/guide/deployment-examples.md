---
title: 部署场景配置示例
outline: deep
---

# 部署场景配置示例

本文档提供面向生产的 Swit 服务部署配置示例，覆盖 Docker 与 Kubernetes，并展示按环境覆盖、以及安全与性能的最佳实践。

相关参考：

- 示例配置：`examples/deployment-config-examples.yaml`
- 服务配置：`swit.yaml`、`switauth.yaml`
- K8s 模板：`deployments/k8s/*.yaml`
- Docker 组合：`deployments/docker/docker-compose.yml`

## 环境分层策略

推荐采用分层配置：

1. 基础默认（纳入版本库）：`swit.yaml` / `switauth.yaml`
2. 按环境覆盖：`swit.dev.yaml`、`swit.prod.yaml`
3. 机密通过环境变量或密钥管理器注入

综合示例见 `examples/deployment-config-examples.yaml`。

## Docker 示例

### 1) 容器友好配置

```yaml
# swit.docker.yaml（片段）
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
    endpoint: "http://jaeger:14268/api/traces" # 同网段容器名
```

### 2) Docker Compose 服务

```yaml
# docker-compose.override.yml（服务层）
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

### 3) 安全与性能要点

- 在入口或反向代理（Caddy/NGINX/Traefik）终止 TLS
- 生产环境限制 CORS 源；避免通配符
- 在编排层设定 CPU/内存限制（Compose profiles 或 K8s）
- 在压力场景使用 `GODEBUG=madvdontneed=1` 改善内存重用

## Kubernetes 示例

### 1) ConfigMap 与 Secret

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

### 2) 带探针与资源配额的 Deployment

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

### 3) 生产清单

- 配置 PodDisruptionBudget 与 HPA
- 若使用网格，开启 TLS/mTLS 与策略
- 使用每命名空间的网络策略限制流量
- 定义 SLO 与告警（就绪延迟、错误比率）

## 环境覆盖

`examples/deployment-config-examples.yaml` 中包含以下覆盖示例：

- `swit.dev.yaml` vs `swit.prod.yaml`
- `switauth.dev.yaml` vs `switauth.prod.yaml`

典型差异：

- 日志级别、采样率、追踪导出器
- Broker 端点（本地/容器网络 vs 托管服务）
- TLS 与认证配置
- 连接池与超时

## 下一步

- 配置参考：`/zh/guide/configuration-reference`
- 能力矩阵：`/zh/reference/capabilities`
- 适配器切换：`/zh/guide/messaging-adapter-switching`


