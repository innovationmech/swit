# Swit Kubernetes 部署

这个目录包含了完整的 Swit 项目 Kubernetes 部署配置，支持在 Kubernetes 集群中部署包括数据库、服务发现和应用服务在内的完整微服务平台。

## 🚀 快速开始

### 前置要求

- Kubernetes 集群 (>= 1.20)
- kubectl 命令行工具
- Docker (用于构建镜像)
- 存储类支持 (默认使用 `standard` 存储类)

### 一键部署

```bash
# 进入 k8s 部署目录
cd deployments/k8s

# 一键部署所有服务
./deploy.sh
```

### 手动部署

```bash
# 进入 k8s 部署目录
cd deployments/k8s

# 按顺序部署各个组件
kubectl apply -f namespace.yaml
kubectl apply -f storage.yaml
kubectl apply -f secret.yaml
kubectl apply -f configmap.yaml
kubectl apply -f mysql.yaml
kubectl apply -f consul.yaml
kubectl apply -f swit-auth.yaml
kubectl apply -f swit-serve.yaml
kubectl apply -f ingress.yaml
```

## 📋 部署架构

### 核心服务

| 服务名 | 部署名称 | 副本数 | 描述 |
|--------|----------|--------|------|
| swit-serve | swit-serve | 3 | 主要应用服务 |
| swit-auth | swit-auth | 2 | 认证服务 |

### 基础设施服务

| 服务名 | 部署名称 | 副本数 | 描述 |
|--------|----------|--------|------|
| MySQL | mysql | 1 | 数据库服务 |
| Consul | consul | 1 | 服务发现和配置中心 |

### 网络配置

| 服务类型 | 内部端口 | 外部端口 | 描述 |
|----------|----------|----------|------|
| swit-serve | 9000 | 30900 | 主要服务 |
| swit-auth | 9001 | 30901 | 认证服务 |
| consul | 8500 | 30850 | Consul UI |
| mysql | 3306 | - | 数据库（仅内部访问）|

## 🌐 访问地址

部署完成后，您可以通过以下地址访问服务：

- **主要服务**: http://\<NODE_IP\>:30900
- **认证服务**: http://\<NODE_IP\>:30901
- **Consul UI**: http://\<NODE_IP\>:30850

其中 `<NODE_IP>` 是您的 Kubernetes 节点 IP 地址。

### 获取节点 IP

```bash
# 获取节点 IP
kubectl get nodes -o wide

# 或者使用脚本自动获取
./deploy.sh status
```

## 🛠️ 管理命令

### 使用部署脚本（推荐）

```bash
# 部署所有服务
./deploy.sh deploy

# 查看部署状态
./deploy.sh status

# 查看服务日志
./deploy.sh logs swit-auth

# 重启服务
./deploy.sh restart swit-serve

# 构建镜像
./deploy.sh build

# 删除部署
./deploy.sh delete

# 查看帮助
./deploy.sh help
```

### 使用 kubectl

```bash
# 查看所有 Pod
kubectl get pods -n swit

# 查看服务状态
kubectl get services -n swit

# 查看部署状态
kubectl get deployments -n swit

# 查看 Pod 日志
kubectl logs -f <pod-name> -n swit

# 进入 Pod
kubectl exec -it <pod-name> -n swit -- sh

# 查看存储状态
kubectl get pvc -n swit

# 查看 Ingress 状态
kubectl get ingress -n swit
```

## 🗂️ 配置文件说明

| 文件名 | 描述 |
|--------|------|
| `namespace.yaml` | 命名空间定义 |
| `storage.yaml` | 持久化存储配置 |
| `secret.yaml` | 密钥配置（数据库密码、JWT 密钥等）|
| `configmap.yaml` | 配置映射（应用配置、数据库连接等）|
| `mysql.yaml` | MySQL 数据库部署和服务 |
| `consul.yaml` | Consul 服务发现部署和服务 |
| `swit-auth.yaml` | 认证服务部署和服务 |
| `swit-serve.yaml` | 主要服务部署和服务 |
| `ingress.yaml` | Ingress 路由配置 |

## 🔧 配置详情

### 环境变量配置

所有服务都通过 ConfigMap 和 Secret 进行配置：

#### ConfigMap (swit-config)
- `DATABASE_HOST`: 数据库主机地址
- `DATABASE_PORT`: 数据库端口
- `SERVICE_DISCOVERY_ADDRESS`: Consul 地址
- `AUTH_SERVICE_URL`: 认证服务地址

#### Secret (swit-secret)
- `DATABASE_PASSWORD`: 数据库密码
- `MYSQL_ROOT_PASSWORD`: MySQL root 密码
- `JWT_SECRET`: JWT 签名密钥

### 存储配置

- **MySQL**: 10GB 持久化存储
- **Consul**: 1GB 持久化存储
- 使用动态存储分配（StorageClass: `standard`）

### 资源限制

| 服务 | CPU 请求 | 内存请求 | CPU 限制 | 内存限制 |
|------|----------|----------|----------|----------|
| mysql | 250m | 256Mi | 500m | 512Mi |
| consul | 100m | 128Mi | 200m | 256Mi |
| swit-auth | 100m | 128Mi | 200m | 256Mi |
| swit-serve | 100m | 128Mi | 200m | 256Mi |

## 🔍 健康检查

所有服务都配置了健康检查：

### Liveness Probe
- 检查服务是否正在运行
- 失败时会重启 Pod

### Readiness Probe
- 检查服务是否就绪接收流量
- 失败时会从服务负载均衡中移除

### 初始化容器 (Init Containers)
- 确保依赖服务（MySQL、Consul）在应用启动前就绪
- 防止启动顺序问题

## 🐛 故障排除

### 常见问题

1. **Pod 启动失败**
   ```bash
   # 查看 Pod 状态
   kubectl get pods -n swit
   
   # 查看 Pod 详细信息
   kubectl describe pod <pod-name> -n swit
   
   # 查看 Pod 日志
   kubectl logs <pod-name> -n swit
   ```

2. **镜像拉取失败**
   ```bash
   # 检查镜像是否存在
   docker images | grep swit
   
   # 重新构建镜像
   ./deploy.sh build
   ```

3. **服务无法访问**
   ```bash
   # 检查服务状态
   kubectl get services -n swit
   
   # 检查端点
   kubectl get endpoints -n swit
   
   # 检查网络策略
   kubectl get networkpolicies -n swit
   ```

4. **存储问题**
   ```bash
   # 查看 PVC 状态
   kubectl get pvc -n swit
   
   # 查看存储类
   kubectl get storageclass
   
   # 查看持久化卷
   kubectl get pv
   ```

5. **配置问题**
   ```bash
   # 查看 ConfigMap
   kubectl get configmap -n swit -o yaml
   
   # 查看 Secret
   kubectl get secret -n swit
   ```

### 重置环境

如果遇到无法解决的问题，可以完全重置环境：

```bash
# 删除整个部署
./deploy.sh delete

# 重新部署
./deploy.sh deploy
```

## 📈 扩容和监控

### 手动扩容

```bash
# 扩容认证服务到 3 个副本
kubectl scale deployment swit-auth --replicas=3 -n swit

# 扩容主要服务到 5 个副本
kubectl scale deployment swit-serve --replicas=5 -n swit
```

### 自动扩容（HPA）

可以配置 Horizontal Pod Autoscaler 来自动扩容：

```bash
# 为主要服务配置自动扩容
kubectl autoscale deployment swit-serve --cpu-percent=70 --min=2 --max=10 -n swit

# 为认证服务配置自动扩容
kubectl autoscale deployment swit-auth --cpu-percent=70 --min=1 --max=5 -n swit
```

### 监控

推荐安装以下监控工具：

- **Prometheus**: 指标收集
- **Grafana**: 可视化监控
- **Jaeger**: 分布式追踪
- **ELK Stack**: 日志聚合

## 🔐 安全配置

### 网络安全

- 所有内部通信都在 Kubernetes 集群网络内
- 外部访问仅通过 NodePort 或 Ingress
- 可以配置 NetworkPolicy 进一步限制网络访问

### 数据安全

- 数据库密码存储在 Kubernetes Secret 中
- JWT 密钥使用 Secret 管理
- 所有配置数据与代码分离

### RBAC

建议配置 Role-Based Access Control：

```bash
# 创建服务账户
kubectl create serviceaccount swit-sa -n swit

# 绑定适当的角色
kubectl create rolebinding swit-rb --clusterrole=view --serviceaccount=swit:swit-sa -n swit
```

## 📝 开发说明

### 本地开发

1. **修改代码后重新部署**:
   ```bash
   # 构建新镜像
   ./deploy.sh build
   
   # 重启相关服务
   ./deploy.sh restart swit-auth
   ./deploy.sh restart swit-serve
   ```

2. **调试模式**:
   可以修改 Deployment 配置添加调试参数或挂载本地代码目录。

### CI/CD 集成

可以将部署脚本集成到 CI/CD 流水线中：

```yaml
# GitLab CI 示例
deploy_k8s:
  stage: deploy
  script:
    - cd deployments/k8s
    - ./deploy.sh build
    - ./deploy.sh deploy
  only:
    - master
```

## 🤝 贡献

如果您发现问题或有改进建议，请提交 Issue 或 Pull Request。

## 📄 许可证

请参考项目根目录的 LICENSE 文件。 