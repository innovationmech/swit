# Swit 微服务平台 Helm Chart

这是一个完整的 Helm Chart，用于部署 Swit 微服务平台到 Kubernetes 集群。

## 📋 目录

- [架构概述](#架构概述)
- [前置要求](#前置要求)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [Charts 目录说明](#charts-目录说明)
- [环境配置](#环境配置)
- [部署指南](#部署指南)
- [故障排除](#故障排除)
- [生产环境最佳实践](#生产环境最佳实践)

## 🏗️ 架构概述

Swit 平台包含以下核心组件：

- **swit-auth**: 认证服务 (端口 9001)
- **swit-serve**: 主要 API 服务 (端口 9000)  
- **MySQL**: 数据存储 (端口 3306)
- **Consul**: 服务发现 (端口 8500)

## 📁 Charts 目录说明

`charts/` 目录是 Helm Chart 的依赖管理目录，用于存放子 Chart（Subcharts）和外部依赖。

### 目录用途

#### 1. **自动依赖下载**
当在 `Chart.yaml` 中定义依赖项时，运行 `helm dependency update` 会将依赖下载到此目录：

```yaml
# Chart.yaml
dependencies:
  - name: mysql
    version: "9.14.4"
    repository: "https://charts.bitnami.com/bitnami"
  - name: redis
    version: "17.3.0" 
    repository: "https://charts.bitnami.com/bitnami"
```

执行命令后的目录结构：
```
charts/
├── mysql-9.14.4.tgz          # 下载的 MySQL Chart
├── redis-17.3.0.tgz          # 下载的 Redis Chart
└── Chart.lock                # 依赖锁定文件
```

#### 2. **本地子 Chart**
你可以在此目录创建本地的子 Chart：

```
charts/
├── monitoring/               # 本地监控 Chart
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
│       ├── prometheus.yaml
│       └── grafana.yaml
├── logging/                  # 本地日志 Chart
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
│       └── elasticsearch.yaml
└── backup/                   # 本地备份 Chart
    ├── Chart.yaml
    ├── values.yaml
    └── templates/
        └── cronjob.yaml
```

### 依赖管理命令

```bash
# 下载依赖到 charts/ 目录
helm dependency update

# 构建依赖（如果有本地 chart）
helm dependency build

# 列出依赖
helm dependency list

# 清理下载的依赖
rm -rf charts/*.tgz Chart.lock
```

### 实际应用场景

#### 场景 1：使用外部数据库 Chart
```yaml
# Chart.yaml
dependencies:
  - name: mysql
    version: "9.14.4"
    repository: "https://charts.bitnami.com/bitnami"
    condition: mysql.external.enabled

# values.yaml
mysql:
  external:
    enabled: false  # 默认使用内置 MySQL
  # 当 enabled: true 时，使用 Bitnami MySQL Chart
```

#### 场景 2：微服务拆分
```yaml
# Chart.yaml - 主 Chart
dependencies:
  - name: user-service
    version: "1.0.0"
    repository: "file://./charts/user-service"
  - name: auth-service  
    version: "1.0.0"
    repository: "file://./charts/auth-service"
  - name: api-gateway
    version: "1.0.0"
    repository: "file://./charts/api-gateway"
```

#### 场景 3：环境特定依赖
```yaml
# Chart.yaml
dependencies:
  - name: prometheus
    version: "15.5.0"
    repository: "https://prometheus-community.github.io/helm-charts"
    condition: monitoring.enabled
    tags:
      - monitoring
  - name: jaeger
    version: "0.71.2"
    repository: "https://jaegertracing.github.io/helm-charts"
    condition: tracing.enabled
    tags:
      - observability
```

### 最佳实践

1. **版本锁定**：始终指定具体版本号，避免使用 `^` 或 `~`
2. **条件控制**：使用 `condition` 字段控制依赖是否安装
3. **别名使用**：当需要同一个 Chart 的多个实例时使用 `alias`
4. **本地开发**：在 charts/ 目录创建本地子 Chart 进行开发

### 注意事项

1. **依赖顺序**：Helm 按依赖声明顺序安装，考虑依赖关系
2. **命名冲突**：子 Chart 资源名称可能冲突，使用命名空间或前缀
3. **值传递**：父 Chart 可以通过 values.yaml 向子 Chart 传递配置
4. **钩子继承**：子 Chart 的 hooks 会在父 Chart 中执行

## 🚀 快速开始

### 前置要求

- Kubernetes 1.19+
- Helm 3.2.0+
- Docker 20.10+
- kubectl 配置正确

### 安装步骤

1. **克隆项目**
   ```bash
   git clone <repository-url>
   cd swit
   ```

2. **构建镜像**
   ```bash
   ./deployments/helm/deploy.sh build
   ```

3. **部署平台**
   ```bash
   ./deployments/helm/deploy.sh install
   ```

4. **检查状态**
   ```bash
   ./deployments/helm/deploy.sh status
   ```

## ⚙️ 配置选项

### 全局配置

```yaml
global:
  imageRegistry: ""          # 镜像仓库前缀
  imagePullSecrets: []       # 镜像拉取密钥
  storageClass: "standard"   # 存储类
```

### MySQL 配置

```yaml
mysql:
  enabled: true
  image:
    repository: mysql
    tag: "8.0"
    pullPolicy: IfNotPresent
  
  auth:
    rootPassword: "root"
    username: "root"
    password: "root"
  
  persistence:
    enabled: true
    size: 10Gi
    storageClass: ""
    accessModes:
      - ReadWriteOnce
  
  resources:
    requests:
      memory: "256Mi"
      cpu: "250m"
    limits:
      memory: "512Mi"
      cpu: "500m"
```

### Consul 配置

```yaml
consul:
  enabled: true
  image:
    repository: consul
    tag: "1.15"
    pullPolicy: IfNotPresent
  
  persistence:
    enabled: true
    size: 1Gi
  
  ui:
    enabled: true
    service:
      type: NodePort
      nodePort: 30850
```

### 认证服务配置

```yaml
switAuth:
  enabled: true
  replicaCount: 2
  image:
    repository: swit-auth
    tag: "latest"
    pullPolicy: Never
  
  service:
    type: ClusterIP
    port: 9001
  
  external:
    enabled: true
    service:
      type: NodePort
      nodePort: 30901
  
  resources:
    requests:
      memory: "128Mi"
      cpu: "100m"
    limits:
      memory: "256Mi"
      cpu: "200m"
```

### 主要服务配置

```yaml
switServe:
  enabled: true
  replicaCount: 3
  image:
    repository: swit-serve
    tag: "latest"
    pullPolicy: Never
  
  service:
    type: ClusterIP
    port: 9000
  
  external:
    enabled: true
    service:
      type: NodePort
      nodePort: 30900
```

## 📖 部署指南

### 1. 标准部署

```bash
# 使用默认配置部署
./deployments/helm/deploy.sh install
```

### 2. 自定义配置部署

```bash
# 创建自定义配置文件
cp deployments/helm/values.yaml my-values.yaml

# 编辑配置
vim my-values.yaml

# 使用自定义配置部署
./deployments/helm/deploy.sh -f my-values.yaml install
```

### 3. 生产环境部署

```bash
# 部署到生产命名空间
./deployments/helm/deploy.sh -n production -f production-values.yaml install
```

### 4. 开发环境部署

```bash
# 开发环境（禁用持久化存储）
cat > dev-values.yaml << EOF
mysql:
  persistence:
    enabled: false

consul:
  persistence:
    enabled: false

switAuth:
  replicaCount: 1

switServe:
  replicaCount: 1
EOF

./deployments/helm/deploy.sh -f dev-values.yaml install
```

## 🛠️ 管理操作

### 部署管理

```bash
# 安装
./deployments/helm/deploy.sh install

# 升级
./deployments/helm/deploy.sh upgrade

# 卸载
./deployments/helm/deploy.sh uninstall

# 查看状态
./deployments/helm/deploy.sh status
```

### 服务管理

```bash
# 查看日志
./deployments/helm/deploy.sh logs swit-auth      # 认证服务日志
./deployments/helm/deploy.sh logs swit-serve     # 主要服务日志
./deployments/helm/deploy.sh logs mysql          # 数据库日志
./deployments/helm/deploy.sh logs consul         # Consul 日志

# 重启服务
./deployments/helm/deploy.sh restart swit-auth   # 重启认证服务
./deployments/helm/deploy.sh restart swit-serve  # 重启主要服务

# 扩容服务
kubectl scale deployment swit-auth -n swit --replicas=5
kubectl scale deployment swit-serve -n swit --replicas=10
```

### 开发调试

```bash
# 端口转发到本地
kubectl port-forward -n swit svc/swit-serve-service 9000:9000
kubectl port-forward -n swit svc/swit-auth-service 9001:9001
kubectl port-forward -n swit svc/swit-consul-service 8500:8500

# 进入容器
kubectl exec -n swit -it deployment/swit-auth -- /bin/bash
kubectl exec -n swit -it deployment/swit-serve -- /bin/bash

# 查看配置
kubectl get configmap -n swit
kubectl describe configmap swit-config -n swit
```

## 🔧 故障排除

### 常见问题

#### 1. Pod 无法启动

```bash
# 查看 Pod 状态
kubectl get pods -n swit

# 查看 Pod 详情
kubectl describe pod <pod-name> -n swit

# 查看日志
kubectl logs <pod-name> -n swit --previous
```

#### 2. 镜像拉取失败

```bash
# 检查镜像是否存在
docker images | grep swit

# 重新构建镜像
./deployments/helm/deploy.sh build

# 更新部署
./deployments/helm/deploy.sh upgrade
```

#### 3. 服务无法访问

```bash
# 检查服务状态
kubectl get svc -n swit

# 检查端点
kubectl get endpoints -n swit

# 检查网络策略
kubectl get networkpolicy -n swit
```

#### 4. 数据库连接失败

```bash
# 检查 MySQL 状态
kubectl logs deployment/swit-mysql -n swit

# 检查密钥
kubectl get secret swit-secret -n swit -o yaml

# 测试连接
kubectl run mysql-client --image=mysql:8.0 -n swit -it --rm --restart=Never -- \
  mysql -h swit-mysql-service -u root -p
```

#### 5. 持久化存储问题

```bash
# 检查 PVC 状态
kubectl get pvc -n swit

# 检查存储类
kubectl get storageclass

# 查看存储详情
kubectl describe pvc mysql-pvc -n swit
```

### 日志查看

```bash
# 查看所有组件状态
kubectl get all -n swit

# 查看事件
kubectl get events -n swit --sort-by='.lastTimestamp'

# 查看资源使用情况
kubectl top pods -n swit
kubectl top nodes
```

## 🔄 升级指南

### 版本升级

```bash
# 备份当前配置
helm get values swit -n swit > backup-values.yaml

# 升级到新版本
./deployments/helm/deploy.sh upgrade

# 检查升级状态
kubectl rollout status deployment/swit-auth -n swit
kubectl rollout status deployment/swit-serve -n swit
```

### 回滚操作

```bash
# 查看发布历史
helm history swit -n swit

# 回滚到指定版本
helm rollback swit 1 -n swit

# 回滚到上一个版本
helm rollback swit -n swit
```

### 配置更新

```bash
# 更新配置文件
vim values.yaml

# 应用新配置
./deployments/helm/deploy.sh upgrade

# 验证更新
./deployments/helm/deploy.sh status
```

## 🔒 安全配置

本 Helm Chart 提供了全面的安全配置选项，支持 TLS/mTLS、Secret 管理、ServiceAccount RBAC 和 NetworkPolicy。

### 1. 密钥管理

```yaml
secret:
  database:
    password: "your-secure-password"
    rootPassword: "your-root-password"
  jwt:
    secret: "base64-encoded-jwt-secret"
  
  # TLS 证书配置
  tls:
    createSelfSigned: false  # 仅用于开发/测试
    validityDays: 365
    subject:
      organization: "Swit"
      organizationalUnit: "Engineering"
    san:
      dnsNames:
        - "localhost"
        - "*.swit.svc.cluster.local"
      ipAddresses:
        - "127.0.0.1"
  
  # 数据加密配置
  encryption:
    enabled: false
    key: ""  # 32 字节 base64 编码
    rotationDays: 90
  
  # 外部 Secret 管理（HashiCorp Vault、AWS Secrets Manager 等）
  externalSecrets:
    enabled: false
    backend: "vault"
    vault:
      address: "https://vault.example.com"
      path: "secret/data/swit"
      role: "swit-role"
```

### 2. TLS/mTLS 配置

```yaml
security:
  enabled: true
  
  # TLS 配置
  tls:
    enabled: true
    source: "cert-manager"  # cert-manager, external, self-signed
    duration: "8760h"       # 1 年
    renewBefore: "720h"     # 30 天
    issuer:
      name: "letsencrypt-prod"
      kind: "ClusterIssuer"
    existingSecret: ""      # 使用已有的 Secret
    minVersion: "TLS12"
    cipherSuites:
      - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
      - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
  
  # mTLS 配置（服务间通信）
  mtls:
    enabled: true
    clientAuthMode: "require"  # require, optional, none
    caSecret: ""
    clientCertSecret: ""
```

### 3. OAuth2/OIDC 配置

```yaml
security:
  oauth2:
    enabled: true
    issuerUrl: "https://auth.example.com"
    clientId: "swit-client"
    clientSecretRef: "oauth2-client-secret"
    audience:
      - "swit-api"
    scopes:
      - "openid"
      - "profile"
      - "email"
```

### 4. 速率限制

```yaml
security:
  rateLimit:
    enabled: true
    requestsPerSecond: 100
    burst: 200
    scope: "ip"  # ip, user, global
```

### 5. 网络安全 (NetworkPolicy)

```yaml
networkPolicy:
  enabled: true
  policyTypes:
    - Ingress
    - Egress
  
  ingress:
    allowSameNamespace: true
    allowIngressController: true
    ingressControllerNamespace:
      matchLabels:
        name: ingress-nginx
    customRules: []
  
  egress:
    allowDNS: true
    allowDatabase: true
    allowServiceDiscovery: true
    allowExternalHTTPS: false
    customRules: []
```

### 6. Pod 安全

```yaml
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 10001
  runAsGroup: 10001
  fsGroup: 10001
  seccompProfile:
    type: RuntimeDefault

securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 10001
  capabilities:
    drop:
      - ALL
    add:
      - NET_BIND_SERVICE
```

### 7. ServiceAccount RBAC 配置

```yaml
serviceAccount:
  create: true
  automount: false  # 生产环境建议禁用自动挂载
  annotations:
    # AWS EKS
    eks.amazonaws.com/role-arn: "arn:aws:iam::123456789012:role/SwitServiceRole"
    # GCP GKE
    # iam.gke.io/gcp-service-account: "swit@project.iam.gserviceaccount.com"
  name: "swit-service-account"
  rbac:
    create: true
    rules:
      - apiGroups: [""]
        resources: ["configmaps"]
        verbs: ["get", "list", "watch"]
```

### 8. 安全加固（生产环境）

```yaml
securityHardening:
  enabled: true
  
  # 容器镜像安全
  image:
    verifySignature: false
    allowedRegistries:
      - "docker.io"
      - "gcr.io"
    disallowLatestTag: true
  
  # 运行时安全
  runtime:
    appArmor:
      enabled: false
      profile: "runtime/default"
    seccomp:
      enabled: false
      profile: "RuntimeDefault"
  
  # 敏感数据保护
  sensitiveData:
    maskInLogs: true
    sensitiveFields:
      - "password"
      - "secret"
      - "token"
```

### 安全配置快速参考

| 配置项 | 默认值 | 生产环境建议 |
|--------|--------|--------------|
| `security.enabled` | `false` | `true` |
| `security.tls.enabled` | `false` | `true` |
| `security.mtls.enabled` | `false` | `true` |
| `networkPolicy.enabled` | `false` | `true` |
| `serviceAccount.automount` | `true` | `false` |
| `podSecurityContext.runAsNonRoot` | `false` | `true` |
| `securityContext.allowPrivilegeEscalation` | 未设置 | `false` |
| `securityContext.readOnlyRootFilesystem` | 未设置 | `true` |

## 📊 监控和指标

### Prometheus 集成

```yaml
# 启用 ServiceMonitor
monitoring:
  enabled: true
  prometheus:
    serviceMonitor:
      enabled: true
      interval: 30s
      path: /metrics
```

### 健康检查

```bash
# 检查服务健康状态
curl http://localhost:9000/health    # 主要服务
curl http://localhost:9001/health    # 认证服务
curl http://localhost:8500/v1/status/leader  # Consul
```

## 🌐 生产环境最佳实践

### 1. 资源配置

```yaml
# 生产环境资源配置
switServe:
  replicaCount: 5
  resources:
    requests:
      memory: "256Mi"
      cpu: "200m"
    limits:
      memory: "512Mi"
      cpu: "500m"

switAuth:
  replicaCount: 3
  resources:
    requests:
      memory: "256Mi"
      cpu: "200m"
    limits:
      memory: "512Mi"
      cpu: "500m"
```

### 2. 自动扩容

```yaml
autoscaling:
  switAuth:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
  
  switServe:
    enabled: true
    minReplicas: 3
    maxReplicas: 20
    targetCPUUtilizationPercentage: 70
```

### 3. 高可用配置

```yaml
# Pod 反亲和性
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app.kubernetes.io/name
            operator: In
            values:
            - swit
        topologyKey: kubernetes.io/hostname
```

### 4. 备份策略

```bash
# 数据库备份
kubectl exec -n swit deployment/swit-mysql -- \
  mysqldump -u root -p<password> --all-databases > backup.sql

# 配置备份
helm get values swit -n swit > swit-values-backup.yaml
```

## 📞 支持和贡献

- **问题报告**: [GitHub Issues](https://github.com/innovationmech/swit/issues)
- **功能请求**: [GitHub Discussions](https://github.com/innovationmech/swit/discussions)
- **文档**: [项目文档](https://github.com/innovationmech/swit/docs)

## 📄 许可证

本项目基于 MIT 许可证开源。详情请参见 [LICENSE](../../LICENSE) 文件。 