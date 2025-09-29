---
title: 配置最佳实践
outline: deep
---

# 配置最佳实践

本文档给出在 Swit 框架中进行配置管理的推荐实践，覆盖安全、性能、可维护性，分层继承/覆盖、环境变量映射，以及验证与测试建议。配合自动生成的配置参考一起使用效果最佳。

## 原则

- 配置应声明式、可版本化、按环境隔离
- 采用安全默认值；在启动时尽早校验；关键配置错误快速失败
- 机密与配置分离；支持通过 `*_FILE` 从文件注入机密
- 通过分层与环境变量提供安全的覆盖途径

## 分层与优先级

低 → 高 优先级：

1. 代码默认值（Defaults）
2. 基础文件（例如 `swit.yaml`）
3. 环境文件（例如 `swit.production.yaml`）
4. 覆盖文件（例如 `swit.override.yaml`）
5. 环境变量（`SWIT_...`）

详细键位与映射见：`/zh/guide/configuration-reference`。

## 环境变量映射

- 前缀：`SWIT_`
- 规则：配置键 `a.b.c` → 环境变量 `SWIT_A_B_C`
- 动态段：如 `messaging.brokers.<name>.endpoints` → `SWIT_MESSAGING_BROKERS_<NAME>_ENDPOINTS`
- 机密文件：支持 `VAR_FILE` 形式从文件读取值

示例：

```bash
export SWIT_SERVICE_NAME="my-service"
export SWIT_HTTP_PORT="8080"
export SWIT_GRPC_TLS_ENABLED="true"
export SWIT_MESSAGING_BROKERS_KAFKA_TYPE="kafka"
export SWIT_MESSAGING_BROKERS_KAFKA_ENDPOINTS="broker-1:9092,broker-2:9092"
```

## 继承与覆盖

使用基础 + 环境文件，并保留 `swit.override.yaml` 作为本地/运维覆盖（不入库）。

```yaml
# swit.yaml（基础）
service_name: my-service
http:
  enabled: true
  port: "8080"

# swit.production.yaml
http:
  read_timeout: "30s"
  write_timeout: "30s"

# swit.override.yaml（本地）
http:
  port: "9090"
```

## 校验与测试

- 启动期校验，输出清晰错误信息。常见检查：
  - 必填项（如启用 HTTP 时必须设置 `http.port`）
  - 数值范围与枚举（如 `discovery.failure_mode`）
  - 安全不变式（如 CORS 不允许 `*` 与 `allow_credentials: true` 同时出现）
- 为解析/校验编写表驱动测试

示例：

```go
cfg := server.NewServerConfig()
// ... 加载配置 ...
if err := cfg.Validate(); err != nil {
    log.Fatalf("配置无效: %v", err)
}
```

## 安全最佳实践

- 不提交机密；使用环境变量或机密文件（`*_FILE`）
- 生产环境启用 TLS；正确设置并校验 `server_name`
- 除受控测试外，避免使用 `skip_verify: true`
- CORS 在 `allow_credentials: true` 时禁止通配符来源

## 性能与可靠性

- 合理设置传输超时、消息大小等参数
- 消息系统：
  - 明确设置 `type` 与 `endpoints`
  - 配置重试/退避与连接池
  - 启用监控与健康检查

## 可观测性与运维

- 启用 metrics（`prometheus`）与 tracing（`tracing`）
- 配置健康检查与就绪探针
- 容器中优先使用不可变配置；通过 Env/ConfigMap/Secret 注入覆盖

## 部署清单

- 安全
  - [ ] 生产启用 TLS 且使用有效证书
  - [ ] 未出现通配符 CORS + 凭据
  - [ ] 机密仅来自 Env/Secret 文件

- 可靠性
  - [ ] 合理的超时与重试策略
  - [ ] 健康检查已启用并被采集

- 配置卫生
  - [ ] 按环境划分配置文件
  - [ ] 覆盖路径可追踪且有文档
  - [ ] 启动校验在本地与 CI 均通过

## 示例

最小化生产可用的 HTTP 片段：

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

Kafka broker 示例：

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

## 相关链接

- 配置指南：[/zh/guide/configuration](/zh/guide/configuration)
- 自动生成参考：[/zh/guide/configuration-reference](/zh/guide/configuration-reference)
- 部署示例：[/zh/guide/deployment-examples](/zh/guide/deployment-examples)


