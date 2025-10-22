# Saga Monitoring Example

这是一个完整的 Saga 监控系统示例，展示了如何使用 Prometheus、Grafana、Jaeger 和 Alertmanager 监控分布式 Saga 事务。

## 📋 功能特性

- ✅ **Prometheus 指标收集** - 收集 Saga 执行指标
- ✅ **Grafana 可视化** - 预配置的仪表板
- ✅ **Jaeger 分布式追踪** - 追踪 Saga 执行流程
- ✅ **Alertmanager 告警** - 基于规则的告警系统
- ✅ **完整的监控栈** - 一键启动所有组件
- ✅ **示例应用** - 模拟 Saga 工作负载

## 🚀 快速开始

### 前置条件

- Docker 和 Docker Compose
- Go 1.23+ (如果要本地运行)

### 启动监控栈

```bash
# 进入示例目录
cd examples/saga-monitoring

# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f saga-service
```

### 访问监控界面

启动后，可以访问以下界面：

| 服务 | URL | 用户名/密码 | 说明 |
|------|-----|------------|------|
| 示例应用 | http://localhost:8080 | - | Saga 监控示例仪表板 |
| Prometheus | http://localhost:9090 | - | 指标查询和浏览 |
| Grafana | http://localhost:3000 | admin/admin | 可视化仪表板 |
| Jaeger UI | http://localhost:16686 | - | 分布式追踪界面 |
| Alertmanager | http://localhost:9093 | - | 告警管理 |
| RabbitMQ | http://localhost:15672 | guest/guest | 消息队列管理 |

### 查看指标

访问 Prometheus 指标端点：

```bash
curl http://localhost:8080/metrics
```

关键指标：

```
# Saga 启动总数
saga_monitoring_saga_started_total

# Saga 完成总数
saga_monitoring_saga_completed_total

# Saga 失败总数（按失败原因分类）
saga_monitoring_saga_failed_total{reason="timeout"}

# Saga 执行时长分布
saga_monitoring_saga_duration_seconds_bucket

# 活跃 Saga 数量
saga_monitoring_active_sagas
```

## 📊 Grafana 仪表板

预配置的仪表板包含以下面板：

1. **Saga 执行速率** - 显示启动、完成和失败的速率
2. **成功率** - 实时 Saga 成功率（带阈值指示）
3. **活跃 Saga** - 当前正在执行的 Saga 数量
4. **执行时长** - P50、P95、P99 百分位数
5. **失败原因分布** - 按原因分类的失败统计
6. **补偿速率** - 补偿操作的执行和失败率
7. **资源使用** - 内存和 Goroutine 数量
8. **消息系统指标** - 发布和消费统计
9. **存储延迟** - 数据库操作延迟

### 导入仪表板

仪表板会自动导入。如需手动导入：

1. 访问 Grafana (http://localhost:3000)
2. 登录（admin/admin）
3. 导航到 Dashboards → Import
4. 上传 `grafana-dashboard.json`

## 🔔 告警规则

配置了以下告警规则（在 `alert-rules.yml` 中）：

### 关键告警

- **HighSagaFailureRate** - Saga 失败率超过 10%
- **SlowSagaExecution** - P95 执行时长超过 60 秒
- **TooManyActiveSagas** - 活跃 Saga 超过 1000 个
- **SagaServiceDown** - Saga 服务不可用
- **CompensationFailures** - 补偿操作失败

### 告警通知

告警配置在 `alertmanager.yml` 中。支持多种通知方式：

- Email
- Slack
- PagerDuty
- Webhook

要启用通知，编辑 `alertmanager.yml` 并配置相应的接收器。

## 🔍 分布式追踪

### 查看追踪

1. 访问 Jaeger UI (http://localhost:16686)
2. 在 "Service" 下拉菜单选择 `saga-coordinator`
3. 点击 "Find Traces" 查看追踪

### 追踪配置

追踪配置在 `tracing-config.yaml` 中，支持：

- OpenTelemetry
- Jaeger
- Zipkin
- OTLP (云服务提供商)

关键配置项：

```yaml
opentelemetry:
  sampling:
    strategy: "parent_based"
    ratio: 1.0  # 采样率：1.0 = 100%
    
jaeger:
  endpoint: "http://jaeger-collector:14268/api/traces"
```

## 📈 性能指标

### PromQL 查询示例

**Saga 成功率：**
```promql
(
  rate(saga_monitoring_saga_completed_total[5m]) 
  / 
  rate(saga_monitoring_saga_started_total[5m])
) * 100
```

**P95 执行时长：**
```promql
histogram_quantile(0.95, 
  rate(saga_monitoring_saga_duration_seconds_bucket[5m])
)
```

**按失败原因分组的失败率：**
```promql
sum by (reason) (
  rate(saga_monitoring_saga_failed_total[5m])
)
```

**每秒吞吐量：**
```promql
rate(saga_monitoring_saga_completed_total[1m])
```

## 🔧 配置说明

### Prometheus 配置

`prometheus.yml` 包含：

- 抓取配置（scrape configs）
- 告警规则文件
- 存储配置（30天保留期，10GB 限制）
- 服务发现配置

### 追踪配置

`tracing-config.yaml` 包含：

- 采样策略
- 导出器配置（Jaeger/Zipkin/OTLP）
- 批处理配置
- 资源属性

### 告警配置

`alert-rules.yml` 包含：

- Saga 执行健康告警
- 失败分析告警
- 补偿告警
- 系统资源告警
- SLO 告警

## 🛠️ 本地开发

### 运行示例应用

```bash
# 安装依赖
go mod download

# 运行应用
go run main.go

# 或使用环境变量
HTTP_PORT=8080 go run main.go
```

### 构建 Docker 镜像

```bash
docker build -t saga-monitoring-example .
```

## 📁 文件结构

```
examples/saga-monitoring/
├── main.go                    # 示例应用程序
├── go.mod                     # Go 模块文件
├── Dockerfile                 # Docker 构建文件
├── docker-compose.yml         # Docker Compose 配置
├── prometheus.yml             # Prometheus 配置
├── alert-rules.yml            # 告警规则
├── alertmanager.yml           # Alertmanager 配置
├── tracing-config.yaml        # 分布式追踪配置
├── grafana-dashboard.json     # Grafana 仪表板
├── grafana/
│   ├── provisioning/
│   │   ├── datasources/       # 数据源配置
│   │   └── dashboards/        # 仪表板配置
│   └── dashboards/            # 仪表板 JSON 文件
└── README.md                  # 本文档
```

## 🧪 测试告警

### 手动触发告警

可以通过修改应用程序代码来测试告警：

1. **测试高失败率告警**：修改 `executeSaga` 中的成功率
   ```go
   if rand.Float64() < 0.50 {  // 降低到 50% 成功率
   ```

2. **测试慢执行告警**：增加执行时间
   ```go
   executionTime := time.Duration(rand.Intn(60000)+30000) * time.Millisecond
   ```

3. **测试活跃 Saga 告警**：增加 Saga 创建频率
   ```go
   numSagas := rand.Intn(20) + 10  // 每次创建 10-30 个 Saga
   ```

重新构建并部署后，查看 Alertmanager 界面确认告警触发。

## 🔒 安全建议

- 在生产环境中修改默认密码
- 使用 TLS/SSL 加密通信
- 配置适当的身份验证
- 限制网络访问
- 定期更新镜像版本

## 📚 相关文档

- [Saga 监控指南](../../docs/saga-monitoring-guide.md)
- [Prometheus 文档](https://prometheus.io/docs/)
- [Grafana 文档](https://grafana.com/docs/)
- [Jaeger 文档](https://www.jaegertracing.io/docs/)
- [OpenTelemetry 文档](https://opentelemetry.io/docs/)

## 🐛 故障排查

### Prometheus 无法抓取指标

```bash
# 检查服务是否运行
docker-compose ps

# 检查网络连接
docker-compose exec prometheus wget -O- http://saga-service:8080/metrics

# 查看 Prometheus 日志
docker-compose logs prometheus
```

### Grafana 无法连接 Prometheus

```bash
# 测试 Prometheus 连接
docker-compose exec grafana wget -O- http://prometheus:9090/api/v1/query?query=up

# 检查数据源配置
docker-compose exec grafana cat /etc/grafana/provisioning/datasources/prometheus.yml
```

### Jaeger 没有追踪数据

```bash
# 检查 Jaeger 日志
docker-compose logs jaeger

# 验证应用配置
echo $JAEGER_ENDPOINT

# 测试追踪端点
curl -X POST http://localhost:14268/api/traces \
  -H "Content-Type: application/json" \
  -d '{"data": []}'
```

## 🤝 贡献

欢迎提交问题和改进建议！

## 📄 许可证

本示例遵循项目主许可证。

