# Charts 目录使用指南

这个文档详细说明了 Helm Chart 中 `charts/` 目录的用途和最佳实践。

## 📁 什么是 Charts 目录？

`charts/` 目录是 Helm Chart 的依赖管理目录，用于存放：

1. **子 Chart（Subcharts）**：本地创建的子组件
2. **外部依赖**：从 Chart 仓库下载的依赖包
3. **依赖锁定文件**：`Chart.lock` 文件记录依赖的确切版本

## 🏗️ 目录结构示例

```
charts/
├── monitoring/                # 本地子 Chart
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
│       ├── _helpers.tpl
│       └── prometheus.yaml
├── mysql-9.14.4.tgz          # 外部依赖包
├── redis-17.3.0.tgz          # 外部依赖包
└── Chart.lock                # 依赖锁定文件
```

## 🔧 依赖配置

### 1. 在 Chart.yaml 中定义依赖

```yaml
# Chart.yaml
dependencies:
  # 本地子 Chart
  - name: monitoring
    version: "0.1.0"
    repository: "file://./charts/monitoring"
    condition: monitoring.enabled
  
  # 外部 Chart
  - name: mysql
    version: "9.14.4"
    repository: "https://charts.bitnami.com/bitnami"
    condition: mysql.external.enabled
    alias: mysql-external
  
  # 条件性依赖
  - name: prometheus
    version: "15.5.0"
    repository: "https://prometheus-community.github.io/helm-charts"
    condition: monitoring.prometheus.enabled
    tags:
      - monitoring
```

### 2. 依赖项字段说明

| 字段 | 描述 | 示例 |
|------|------|------|
| `name` | Chart 名称 | `mysql` |
| `version` | Chart 版本 | `9.14.4` |
| `repository` | 仓库地址 | `https://charts.bitnami.com/bitnami` |
| `condition` | 启用条件 | `mysql.enabled` |
| `alias` | 别名（用于多实例） | `mysql-primary` |
| `tags` | 标签分组 | `["database"]` |

## 🔄 依赖管理命令

### 基本命令

```bash
# 列出依赖
helm dependency list

# 下载/更新依赖
helm dependency update

# 构建依赖（本地chart）
helm dependency build

# 清理下载的依赖
rm -rf charts/*.tgz Chart.lock
```

### 实际演示

```bash
# 1. 查看当前依赖状态
$ helm dependency list
NAME            VERSION REPOSITORY                      STATUS  
monitoring      0.1.0   file://./charts/monitoring      unpacked

# 2. 如果有外部依赖，更新依赖
$ helm dependency update
Getting updates for unmanaged Helm repositories...
...Successfully got an update from the "https://charts.bitnami.com/bitnami" chart repository
Saving 1 charts
Downloading mysql from repo https://charts.bitnami.com/bitnami
Deleting outdated charts

# 3. 验证依赖已下载
$ ls charts/
monitoring/  mysql-9.14.4.tgz  Chart.lock

# 4. 测试模板生成
$ helm template test . --set monitoring.enabled=true
```

## 💡 实际应用场景

### 场景 1：本地子 Chart（当前示例）

我们创建了一个本地监控子 Chart：

```
charts/monitoring/
├── Chart.yaml          # 监控组件元数据
├── values.yaml         # 默认配置
└── templates/
    ├── _helpers.tpl     # 模板帮助函数
    └── prometheus.yaml  # Prometheus 部署模板
```

**使用方式：**
```bash
# 启用监控
helm install swit . --set monitoring.enabled=true

# 查看监控组件模板
helm template test . --set monitoring.enabled=true --show-only charts/monitoring/templates/prometheus.yaml
```

### 场景 2：外部数据库依赖

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
  auth:
    rootPassword: "secretpassword"
    database: "myapp"
```

**使用外部 MySQL：**
```bash
helm install swit . --set mysql.external.enabled=true
```

### 场景 3：微服务架构拆分

```yaml
# Chart.yaml
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

每个微服务都有自己的子 Chart，独立配置和部署。

### 场景 4：环境特定依赖

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

**按标签安装：**
```bash
# 只安装监控组件
helm install swit . --set tags.monitoring=true

# 安装所有可观测性组件
helm install swit . --set tags.observability=true
```

## 🎯 最佳实践

### 1. 版本管理

```yaml
# ✅ 好的做法：使用精确版本
dependencies:
  - name: mysql
    version: "9.14.4"

# ❌ 避免：使用范围版本
dependencies:
  - name: mysql
    version: "^9.14.0"  # 可能导致不可预测的更新
```

### 2. 条件控制

```yaml
# 使用条件控制依赖安装
dependencies:
  - name: redis
    version: "17.3.0"
    repository: "https://charts.bitnami.com/bitnami"
    condition: redis.enabled

# values.yaml
redis:
  enabled: false  # 默认不安装
```

### 3. 别名使用

```yaml
# 同一个 Chart 的多个实例
dependencies:
  - name: mysql
    version: "9.14.4"
    repository: "https://charts.bitnami.com/bitnami"
    alias: mysql-primary
  - name: mysql
    version: "9.14.4"
    repository: "https://charts.bitnami.com/bitnami"
    alias: mysql-replica
```

### 4. 配置传递

```yaml
# values.yaml - 父 Chart 向子 Chart 传递配置
monitoring:
  prometheus:
    image: prom/prometheus:v2.45.0
    resources:
      limits:
        memory: 2Gi
    storage:
      size: 10Gi
```

### 5. 本地开发

```bash
# 创建新的子 Chart
mkdir -p charts/new-component
cd charts/new-component

# 创建基本结构
cat > Chart.yaml << EOF
apiVersion: v2
name: new-component
version: 0.1.0
EOF

mkdir -p templates
cat > values.yaml << EOF
enabled: true
EOF
```

## ⚠️ 常见问题和注意事项

### 1. 依赖顺序

Helm 按依赖声明顺序安装，确保依赖关系正确：

```yaml
dependencies:
  - name: mysql        # 先安装数据库
    version: "9.14.4"
  - name: my-app       # 再安装应用
    version: "1.0.0"
```

### 2. 命名冲突

子 Chart 资源名称可能冲突，使用命名空间或前缀：

```yaml
# 在模板中使用 Release.Name 避免冲突
metadata:
  name: {{ .Release.Name }}-{{ .Chart.Name }}-service
```

### 3. Chart.lock 管理

```bash
# 提交 Chart.lock 到版本控制
git add Chart.lock

# 团队成员使用相同版本
helm dependency build  # 而不是 update
```

### 4. 值传递机制

父 Chart 的 values.yaml 中的配置会传递给子 Chart：

```yaml
# 父 Chart values.yaml
monitoring:          # 这个键对应子 Chart 名称
  prometheus:         # 会传递给子 Chart
    enabled: true
    resources:
      memory: 1Gi
```

## 🔗 相关命令参考

```bash
# Chart 管理
helm create mychart                    # 创建新 Chart
helm lint .                           # 验证 Chart
helm template test .                  # 生成模板
helm install release .                # 安装 Chart
helm upgrade release .                # 升级 Chart

# 依赖管理
helm dependency list                  # 列出依赖
helm dependency update               # 更新依赖
helm dependency build                # 构建依赖

# 仓库管理
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
helm search repo mysql

# 调试
helm template . --debug              # 调试模板
helm install --dry-run --debug       # 模拟安装
helm get manifest release            # 查看已安装的清单
```

## 📚 扩展阅读

- [Helm Chart Dependencies](https://helm.sh/docs/chart_template_guide/subcharts_and_globals/)
- [Helm Dependency Commands](https://helm.sh/docs/helm/helm_dependency/)
- [Chart Best Practices](https://helm.sh/docs/chart_best_practices/)
- [Subcharts and Global Values](https://helm.sh/docs/chart_template_guide/subcharts_and_globals/) 