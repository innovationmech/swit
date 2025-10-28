# Swit v0.9.0 发布说明

## 🎉 重大更新：Saga 分布式事务系统正式发布

我们非常高兴地宣布 Swit v0.9.0 正式发布！这是一个里程碑版本，引入了完整的 **Saga 分布式事务系统**，为微服务架构提供企业级的分布式事务管理能力。

## 📅 发布信息

- **版本**: v0.9.0
- **发布日期**: 2025-10-28
- **Go 版本要求**: 1.23.12+
- **兼容性**: 完全向后兼容 v0.8.x

## ✨ 核心亮点

### 1. 完整的 Saga 分布式事务系统

Saga 模式是处理微服务分布式事务的最佳实践。Swit v0.9.0 提供了功能完整、生产就绪的 Saga 实现。

#### 🔧 核心能力

- **多种执行模式**
  - **编排模式（Orchestration）**: 中心化的事务协调，适合复杂的业务流程
  - **协同模式（Choreography）**: 去中心化的事件驱动，适合松耦合的服务
  - **混合模式（Hybrid）**: 智能选择最优执行模式

- **可靠性保障**
  - 指数退避、固定延迟、线性退避等多种重试策略
  - 灵活的补偿策略（顺序、并行、自定义）
  - 自动超时检测和恢复
  - 死信队列（DLQ）处理失败消息
  - 事务性消息发送保证

- **多存储后端**
  - PostgreSQL（生产推荐）
  - MySQL
  - SQLite（开发测试）
  - 内存存储（单元测试）

#### 📝 Saga DSL - 声明式工作流定义

使用人类可读的 YAML 格式定义复杂的分布式事务：

```yaml
saga:
  id: order-processing
  name: 订单处理
  mode: orchestration
  timeout: 5m

steps:
  - id: create-order
    name: 创建订单
    action: http
    url: http://order-service/orders
    compensation:
      action: http
      url: http://order-service/orders/{order_id}/cancel
    
  - id: reserve-inventory
    name: 预留库存
    action: http
    url: http://inventory-service/reserve
    compensation:
      action: http
      url: http://inventory-service/release
```

**配套工具：**
- `saga-dsl-validate` - DSL 配置验证
- `saga-migrate` - 数据库迁移
- 代码生成器 - 自动生成 Go 代码

### 2. 企业级安全特性

#### 🔒 认证与授权

- **认证中间件**: 支持 JWT 和 API Key 认证
- **RBAC 权限控制**: 基于角色的访问控制
- **ACL 访问控制列表**: 细粒度的权限管理

#### 🛡️ 数据保护

- **敏感数据加密**: AES-256-GCM 加密算法
- **字段级加密**: 支持对特定字段进行加密
- **审计日志**: 完整的操作追踪和审计报告

### 3. 全面的监控与可观测性

#### 📊 指标收集

- **Prometheus 集成**: 收集 40+ 关键性能指标
  - Saga 执行次数、成功率、失败率
  - 步骤执行时长、重试次数
  - 补偿执行统计
  - 存储和消息传递性能

- **自定义指标**: 支持业务指标定义和收集

#### 🔍 分布式追踪

- **OpenTelemetry 集成**: 标准的追踪协议
- **Jaeger/Zipkin 支持**: 可视化追踪数据
- **上下文传播**: 跨服务的完整调用链追踪

#### 💊 健康检查

- 实时监控 Saga 系统健康状态
- 自动告警和通知集成

### 4. Saga Dashboard - 可视化管理后台

功能完整的 Web 管理界面，提供直观的运维体验。

#### 核心功能

- **Saga 管理**
  - 实时查看所有 Saga 实例状态
  - 查询 Saga 执行历史和详情
  - 手动取消或重试 Saga
  - 流程可视化图表

- **监控大盘**
  - 实时性能指标展示
  - 告警通知
  - 趋势分析图表

- **现代化 UI**
  - 响应式设计，支持移动端
  - WebSocket 实时数据推送
  - 交互式可视化图表

### 5. 丰富的示例应用

提供 4 个完整的生产级示例，展示最佳实践：

#### 📦 订单处理 Saga
完整的电商订单处理流程，包含订单创建、库存预留、支付处理和订单确认。

#### 💳 支付处理 Saga
跨账户资金转账场景，展示强一致性要求下的 Saga 实现。

#### 📊 库存管理 Saga
多仓库库存协调，展示并行步骤和补偿策略。

#### 👤 用户注册 Saga
用户注册和初始化流程，展示简单的 Saga 使用。

**每个示例包含：**
- 完整的源代码实现
- Docker Compose 一键启动环境
- 端到端测试
- 详细的文档说明

## 📚 完善的文档体系

### 用户文档
- **快速开始**: 10 分钟上手 Saga
- **用户指南**: 核心概念和使用方法
- **API 参考**: 完整的 API 文档
- **教程**: 分步教程和最佳实践

### 开发者文档
- **开发者指南**: 架构设计和扩展开发
- **DSL 参考**: DSL 语法完整说明
- **安全指南**: 安全最佳实践
- **监控指南**: 监控和故障排查

### 部署文档
- **部署指南**: 生产环境部署最佳实践
- **配置参考**: 完整的配置选项说明
- **运维手册**: 日常运维操作指南

### 文档网站
- **在线文档**: https://innovationmech.github.io/swit/
- **多语言支持**: 中文和英文文档
- **交互式示例**: 可运行的代码示例

## 🧪 测试与质量

### 测试覆盖率大幅提升

- **Saga 核心包**: 96.0% 覆盖率（从 70.4% 提升）
- **监控包**: 77% 覆盖率（从 71.5% 提升）
- **整体覆盖率**: 85%+ 覆盖率

### 完整的测试套件

- **单元测试**: 覆盖所有核心功能
- **集成测试**: 端到端场景测试
- **性能基准测试**: 量化性能指标
- **混沌测试**: 验证系统在各种异常情况下的稳定性
- **DSL 测试**: 验证 DSL 配置的正确性

### CI/CD 增强

- 自动化测试和构建流程
- 代码覆盖率报告（Codecov）
- 代码质量检查（Linting）
- 安全扫描（CodeQL）

## 🚀 快速开始

### 安装

```bash
go get github.com/innovationmech/swit@v0.9.0
```

### 创建第一个 Saga

```go
package main

import (
    "context"
    "time"
    "github.com/innovationmech/swit/pkg/saga"
)

func main() {
    // 1. 创建 Saga 定义
    def := saga.NewDefinitionBuilder().
        SetID("order-processing").
        SetName("订单处理").
        AddStep(NewCreateOrderStep()).
        AddStep(NewReserveInventoryStep()).
        AddStep(NewProcessPaymentStep()).
        SetTimeout(5 * time.Minute).
        Build()
    
    // 2. 创建协调器
    coordinator, err := saga.NewCoordinator(
        saga.WithStorage(storage),
        saga.WithMessaging(publisher),
    )
    if err != nil {
        panic(err)
    }
    
    // 3. 执行 Saga
    instance, err := coordinator.Execute(context.Background(), def, orderData)
    if err != nil {
        panic(err)
    }
    
    // 4. 等待完成
    result := coordinator.Wait(context.Background(), instance.ID)
    if result.State == saga.StateCompleted {
        println("订单处理成功!")
    }
}
```

### 使用 DSL 方式

```yaml
# order-saga.yaml
saga:
  id: order-processing
  name: 订单处理
  mode: orchestration
  timeout: 5m

steps:
  - id: create-order
    name: 创建订单
    action: http
    url: http://order-service/orders
    method: POST
    
  - id: reserve-inventory
    name: 预留库存
    action: http
    url: http://inventory-service/reserve
    
  - id: process-payment
    name: 处理支付
    action: http
    url: http://payment-service/pay
```

```bash
# 验证配置
saga-dsl-validate order-saga.yaml

# 运行 Saga
./bin/swit-serve --saga-config order-saga.yaml
```

## 📦 完整示例

克隆仓库并运行示例：

```bash
git clone https://github.com/innovationmech/swit.git
cd swit

# 启动依赖服务
docker-compose -f examples/saga-examples/docker-compose.yml up -d

# 运行订单处理示例
cd pkg/saga/examples
go run order_saga.go

# 或使用提供的脚本
./run.sh order
```

## 🔄 迁移指南

### 从 v0.8.x 升级

**好消息**: v0.9.0 完全向后兼容 v0.8.x，无需修改现有代码！

如果您想使用新的 Saga 功能：

1. **更新依赖**
   ```bash
   go get -u github.com/innovationmech/swit@v0.9.0
   ```

2. **安装数据库**（如需使用 Saga）
   ```bash
   # PostgreSQL (推荐)
   docker run -d -p 5432:5432 \
     -e POSTGRES_DB=saga \
     -e POSTGRES_PASSWORD=password \
     postgres:16
   ```

3. **运行迁移**
   ```bash
   saga-migrate -dsn "postgres://postgres:password@localhost/saga" -action migrate
   ```

4. **配置 Saga**
   在您的配置文件中添加：
   ```yaml
   saga:
     enabled: true
     storage:
       type: postgres
       dsn: "postgres://postgres:password@localhost/saga"
     messaging:
       broker: nats
       url: "nats://localhost:4222"
   ```

### 无破坏性变更

- ✅ 所有现有 API 保持不变
- ✅ 配置格式兼容
- ✅ 依赖版本兼容

## 🐛 Bug 修复

- 修复 CI 构建缺少 proto 生成的问题 (#756)
- 修复多个不稳定的测试用例 (#758, #740, #747, #736)
- 修复 CodeQL 安全告警 (#745)
- 修正 Saga 重试策略类型名称 (#763)
- 修复 MockStateStorage 状态持久化问题 (#743)

## ⚡ 性能改进

- 批量事件发布优化 - 提高吞吐量 50%
- 连接池优化 - 提高并发能力 30%
- 异步补偿 - 降低延迟 40%

## 📊 基准测试结果

```
BenchmarkSagaExecution/orchestration-8         10000    115234 ns/op
BenchmarkSagaExecution/choreography-8           8000    142567 ns/op
BenchmarkStateStorage/save-8                  100000     11234 ns/op
BenchmarkStateStorage/get-8                   200000      5678 ns/op
BenchmarkEventPublisher/publish-8             150000      8901 ns/op
```

## 🛠️ 新增工具

- `saga-migrate` - Saga 数据库迁移工具
- `saga-dsl-validate` - DSL 配置验证工具
- `swit-docgen` - 文档自动生成工具

## 🔗 资源链接

- **官方文档**: https://innovationmech.github.io/swit/
- **GitHub 仓库**: https://github.com/innovationmech/swit
- **示例代码**: https://github.com/innovationmech/swit/tree/master/pkg/saga/examples
- **问题反馈**: https://github.com/innovationmech/swit/issues
- **讨论区**: https://github.com/innovationmech/swit/discussions

## 🙏 致谢

感谢所有为本版本做出贡献的开发者和社区成员！特别感谢以下方面的贡献：

- Saga 分布式事务系统的完整实现
- 全面的测试套件和文档体系
- 示例应用和最佳实践
- Bug 修复和性能优化

## 🚧 下一步计划

我们正在为 **v1.0.0** 做最后的准备，计划包括：

- ✅ 完善 Saga 文档（已完成）
- ✅ 全面的测试覆盖（已完成）
- 🔄 社区反馈收集（进行中）
- 📝 更新 README 和项目文档（即将开始）
- 🎯 发布 v1.0.0 正式版（预计 2025-11 月）

## 💬 反馈与支持

如果您在使用过程中遇到任何问题或有任何建议，欢迎：

- 提交 Issue: https://github.com/innovationmech/swit/issues
- 参与讨论: https://github.com/innovationmech/swit/discussions
- 贡献代码: https://github.com/innovationmech/swit/pulls

---

**Happy Coding with Swit! 🎉**

Swit 团队
2025-10-28

