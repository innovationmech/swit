# Jaeger 配置目录

本目录包含 Jaeger 追踪后端的配置文件和部署说明。

## 目录结构

```bash
jaeger/
├── README.md              # Jaeger 配置说明（本文件）
├── config/               # Jaeger 配置文件
│   ├── jaeger.yml        # Jaeger All-in-One 配置
│   ├── sampling.json     # 采样策略配置
│   └── storage.yml       # 存储配置（生产环境用）
└── docker-compose.jaeger.yml  # 独立的 Jaeger 部署配置
```

## 配置说明

### 开发环境配置

开发环境使用 Jaeger All-in-One 模式，包含所有组件：
- Jaeger Agent（接收追踪数据）
- Jaeger Collector（处理追踪数据）
- Jaeger Query（查询服务）
- Jaeger UI（Web 界面）
- 内存存储（临时存储）

### 生产环境配置

生产环境建议使用独立部署模式：
- 分离的 Collector 集群
- 外部存储后端（Elasticsearch/Cassandra）
- 负载均衡配置
- 监控和告警设置

## 端口说明

| 端口 | 协议 | 用途 |
|------|------|------|
| 16686 | HTTP | Jaeger UI Web 界面 |
| 14268 | HTTP | Jaeger Collector (HTTP) |
| 14250 | gRPC | Jaeger Collector (gRPC) |
| 6831 | UDP | Jaeger Agent (Thrift compact) |
| 6832 | UDP | Jaeger Agent (Thrift binary) |
| 5778 | HTTP | Jaeger Agent 配置端口 |

## 环境变量配置

### 基础配置
```bash
# 存储类型
SPAN_STORAGE_TYPE=memory  # 开发环境
# SPAN_STORAGE_TYPE=elasticsearch  # 生产环境

# Collector 配置
COLLECTOR_ZIPKIN_HOST_PORT=:9411
COLLECTOR_OTLP_ENABLED=true

# UI 配置
QUERY_BASE_PATH=/
```

### Elasticsearch 存储配置（生产环境）
```bash
ES_SERVER_URLS=http://elasticsearch:9200
ES_USERNAME=elastic
ES_PASSWORD=changeme
ES_INDEX_PREFIX=jaeger
```

## 采样策略

采样策略控制追踪数据的收集比例，平衡性能和可观测性：

- **开发环境**: 100% 采样（所有请求都追踪）
- **测试环境**: 50% 采样
- **生产环境**: 1-10% 采样（根据流量调整）

## 使用示例

### 启动 Jaeger
```bash
# 使用 Docker Compose（推荐）
docker-compose up -d jaeger

# 或者使用 Docker 直接启动
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:1.42
```

### 访问 Jaeger UI
```bash
# 本地访问
open http://localhost:16686

# 或通过命令行查询
curl http://localhost:16686/api/services
```

### 发送测试数据
```bash
# 创建测试追踪
curl -X POST http://localhost:14268/api/traces \
  -H "Content-Type: application/json" \
  -d @test-trace.json
```

## 监控和维护

### 健康检查
- Jaeger UI 健康检查：`http://localhost:16686`
- Collector 健康检查：`http://localhost:14269`

### 日志查看
```bash
# 查看 Jaeger 容器日志
docker logs jaeger

# 实时监控日志
docker logs -f jaeger
```

### 数据清理
内存存储会在容器重启时自动清理。生产环境需要定期清理旧数据。

## 故障排查

### 常见问题

1. **UI 无法访问**
   - 检查端口 16686 是否被占用
   - 确认 Docker 容器正常运行

2. **无追踪数据**
   - 检查服务配置的 Jaeger endpoint
   - 确认防火墙设置允许 14268 端口

3. **性能问题**
   - 调整采样率
   - 检查存储后端性能
   - 优化 Collector 配置

### 调试命令
```bash
# 检查 Jaeger 服务状态
curl http://localhost:14269

# 查看配置
curl http://localhost:5778/sampling

# 测试连接
telnet localhost 14268
```

## 相关链接

- [Jaeger 官方文档](https://www.jaegertracing.io/docs/)
- [OpenTelemetry 集成指南](https://opentelemetry.io/docs/instrumentation/go/)
- [Docker 部署指南](https://www.jaegertracing.io/docs/1.42/deployment/)
