# Changelog

所有对该项目的重要更改都将记录在此文件中。

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [未发布]

### 重大特性 🎉

#### Saga 分布式事务系统

Swit 框架现已完整支持 Saga 分布式事务模式，提供企业级的分布式事务管理能力。

- **核心引擎**
  - ✅ Saga 协调器（Coordinator）- 管理事务生命周期
  - ✅ 状态管理器（State Manager）- 持久化和查询 Saga 状态
  - ✅ 事件发布/订阅系统 - 可靠的事件消息传递
  - ✅ 多存储后端支持 - PostgreSQL、MySQL、SQLite、内存存储

- **执行模式**
  - ✅ 编排模式（Orchestration）- 中心化事务协调
  - ✅ 协同模式（Choreography）- 去中心化事件驱动
  - ✅ 混合模式（Hybrid）- 智能模式选择

- **高级特性**
  - ✅ 灵活的重试策略（指数退避、固定延迟、线性退避）
  - ✅ 多种补偿策略（顺序、并行、自定义）
  - ✅ 超时控制和自动恢复
  - ✅ 死信队列（DLQ）处理
  - ✅ 事务性消息发送
  - ✅ 批量事件发布

#### Saga DSL（领域特定语言）

提供声明式的 YAML 配置方式定义 Saga 工作流，简化开发流程。

- **DSL 功能**
  - ✅ YAML 语法定义 - 人类可读的配置格式
  - ✅ DSL 解析器 - 将 YAML 转换为 Saga 定义
  - ✅ 配置验证器 - 编译时和运行时验证
  - ✅ 代码生成器 - 自动生成 Go 代码
  - ✅ DSL 测试工具 - 验证 DSL 配置正确性

- **命令行工具**
  - ✅ `saga-dsl-validate` - DSL 配置验证工具
  - ✅ `saga-migrate` - 数据库迁移工具

#### Saga 安全系统

企业级安全特性确保分布式事务的安全性和合规性。

- **认证与授权**
  - ✅ Saga 认证中间件 - JWT/API Key 支持
  - ✅ RBAC 权限控制 - 基于角色的访问控制
  - ✅ ACL 访问控制列表 - 细粒度权限管理

- **数据保护**
  - ✅ 敏感数据加密 - AES-256-GCM 加密
  - ✅ 字段级加密 - 支持特定字段加密
  - ✅ 审计日志系统 - 完整的操作追踪
  - ✅ 操作审计报告 - 生成审计报告

#### Saga 监控与可观测性

全面的监控和追踪能力，确保生产环境的可靠运行。

- **指标收集**
  - ✅ Prometheus 集成 - 收集关键性能指标
  - ✅ 自定义指标 - 支持业务指标定义
  - ✅ 指标仪表盘 - 可视化监控数据

- **分布式追踪**
  - ✅ OpenTelemetry 集成 - 标准追踪协议
  - ✅ Jaeger/Zipkin 支持 - 追踪数据可视化
  - ✅ 上下文传播 - 跨服务追踪

- **健康检查**
  - ✅ 健康检查接口 - 监控 Saga 系统状态
  - ✅ 告警集成 - 自动告警通知

#### Saga Dashboard（管理后台）

功能完整的 Web 管理界面，提供直观的 Saga 管理体验。

- **核心功能**
  - ✅ Saga 列表和详情查询 - 查看所有 Saga 实例
  - ✅ 实时状态监控 - 实时查看执行状态
  - ✅ 操作控制 API - 取消、重试 Saga
  - ✅ 流程可视化 - 图形化显示执行流程
  - ✅ 指标数据展示 - 性能和健康指标

- **前端界面**
  - ✅ 现代化 Web UI - 响应式设计
  - ✅ 实时数据推送 - WebSocket 实时更新
  - ✅ 交互式图表 - 数据可视化

#### Saga 示例应用

提供 4 个完整的业务场景示例，展示最佳实践。

- **示例应用**
  - ✅ 订单处理 Saga - 电商订单完整流程
  - ✅ 支付处理 Saga - 跨账户资金转账
  - ✅ 库存管理 Saga - 多仓库库存协调
  - ✅ 用户注册 Saga - 用户注册和初始化

- **配套工具**
  - ✅ Docker Compose 配置 - 一键启动环境
  - ✅ 运行脚本 - 简化示例运行
  - ✅ 端到端测试 - 验证示例正确性

### 测试与质量 🧪

- **测试套件**
  - ✅ 完整的单元测试套件 - 核心包覆盖率 >90%
  - ✅ 集成测试套件 - 端到端场景测试
  - ✅ 性能基准测试 - 性能回归检测
  - ✅ 混沌测试套件 - 验证系统稳定性
  - ✅ 测试数据 Fixtures - 标准化测试数据

- **测试工具**
  - ✅ Mock 测试工具 - 简化测试编写
  - ✅ DSL 测试工具 - DSL 配置测试

- **CI/CD 集成**
  - ✅ GitHub Actions 工作流 - 自动化测试和构建
  - ✅ 代码覆盖率报告 - Codecov 集成
  - ✅ 代码质量检查 - Linting 和代码审查

### 文档 📚

#### Saga 文档体系

- **用户文档**
  - ✅ Saga 用户指南 - 快速开始和核心概念
  - ✅ Saga API 参考 - 完整的 API 文档
  - ✅ Saga 教程 - 分步教程和最佳实践
  - ✅ 部署指南 - 生产环境部署指南

- **开发者文档**
  - ✅ Saga 开发者指南 - 架构设计和扩展开发
  - ✅ DSL 参考文档 - DSL 语法和使用
  - ✅ 安全指南 - 安全最佳实践
  - ✅ 监控指南 - 监控和故障排查

- **测试文档**
  - ✅ 测试指南 - 测试策略和方法
  - ✅ 测试覆盖率报告 - 详细的覆盖率分析

- **示例文档**
  - ✅ 示例应用文档 - 每个示例的详细说明
  - ✅ 架构分析 - 示例架构设计

#### 文档网站

- ✅ VitePress 文档站点 - 现代化文档体验
- ✅ 多语言支持 - 中英文文档
- ✅ API 自动生成 - 保持文档同步
- ✅ 交互式示例 - 可运行的代码示例

### 修复 🐛

- **测试稳定性**
  - 修复 `TestRealtimePusher_MultipleClients` 不稳定测试 (#758)
  - 修复混沌测试中步骤名称不匹配问题 (#740)
  - 修复 `TestMessageLoss_EventOrdering` 不稳定性 (#747)
  - 修复集成测试跳过问题 (#736)

- **CI/CD 修复**
  - 修复 CI 构建缺少 proto 生成的问题 (#756)
  - 修复 CodeQL 安全告警 (#745)
  - 修复测试工作流失败问题 (#745)
  - 修复覆盖率测试跳过不稳定测试 (#745)

- **Saga 修复**
  - 修正重试策略类型名称 (#763)
  - 修复 MockStateStorage 状态持久化问题 (#743)

### 改进 ⚡

- **代码质量**
  - 重构 CI 和测试工作流，消除重复 (#745)
  - 清理测试文件中的空白字符 (#740, #744, #762)
  - 标准化 mock 服务函数格式 (#738)
  - 调整覆盖率阈值为更现实的目标 (#745)

- **测试覆盖率提升**
  - Saga base 包：70.4% → 96.0% (#735)
  - Saga monitoring 包：71.5% → 77% (#734)
  - State/storage 包：大幅提升 (#733)
  - DSL Parser：补充边界测试和错误处理 (#737)

- **依赖更新**
  - 升级 vite 从 5.4.20 到 5.4.21 (#645)
  - 升级 playwright 和 @playwright/test (#644)

### 工具和命令 🛠️

- **新增命令行工具**
  - `saga-migrate` - Saga 数据库迁移工具
  - `saga-dsl-validate` - DSL 配置验证工具
  - `swit-docgen` - 文档生成工具

- **Makefile 增强**
  - 新增 `make saga-test` - Saga 相关测试
  - 新增 `make saga-docs` - 生成 Saga 文档
  - 改进 `make quality` - 更全面的质量检查

### 部署与运维 🚀

- **容器化支持**
  - Docker Compose 配置 - 完整的本地开发环境
  - Helm Charts - Kubernetes 部署支持
  - 多阶段构建 - 优化镜像大小

- **配置管理**
  - Saga 配置示例 - 各种场景的配置模板
  - 环境变量支持 - 灵活的配置方式
  - 配置验证 - 启动时验证配置正确性

### 性能优化 📈

- **Saga 性能**
  - 批量事件发布 - 提高吞吐量
  - 异步补偿 - 降低延迟
  - 连接池优化 - 提高并发能力

- **基准测试**
  - 性能基准测试套件 - 量化性能指标
  - 压力测试 - 验证系统极限

## 版本说明

### 版本号更新

- 当前版本：**v0.9.0** (准备中)
- 目标版本：**v1.0.0** (即将发布)

### Go 版本要求

- **最低版本**: Go 1.23.12+
- **推荐版本**: Go 1.23.12 或更高

### 依赖要求

#### 必需依赖
- `google.golang.org/grpc` - v1.69.0+
- `google.golang.org/protobuf` - v1.36.6+
- `github.com/gin-gonic/gin` - v1.9.1+
- `go.uber.org/zap` - v1.27.0+
- `github.com/spf13/viper` - v1.18.2+
- `github.com/spf13/cobra` - v1.7.0+

#### Saga 相关依赖
- `gorm.io/gorm` - v1.30.0+ (状态存储)
- `github.com/lib/pq` - v1.10.9+ (PostgreSQL)
- `github.com/go-sql-driver/mysql` - v1.8.1+ (MySQL)
- `github.com/mattn/go-sqlite3` - v1.14.32+ (SQLite)
- `github.com/nats-io/nats.go` - v1.31.0+ (消息传递)
- `github.com/streadway/amqp` - v1.1.0+ (RabbitMQ)

#### 监控和追踪依赖
- `go.opentelemetry.io/otel` - v1.38.0+
- `go.opentelemetry.io/otel/sdk` - v1.32.0+
- `github.com/prometheus/client_golang` - v1.23.0+

#### 可选依赖
- `github.com/redis/go-redis/v9` - v9.14.0+ (缓存)
- `github.com/hashicorp/consul/api` - v1.29.4+ (服务发现)
- `github.com/getsentry/sentry-go` - v0.35.1+ (错误追踪)

## 迁移指南

### 从 v0.8.x 升级到 v0.9.0

#### 新增功能

如果您想使用 Saga 分布式事务功能：

1. **安装数据库**
   ```bash
   # PostgreSQL (推荐)
   brew install postgresql
   # 或使用 Docker
   docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=password postgres:16
   ```

2. **运行数据库迁移**
   ```bash
   saga-migrate -dsn "postgres://user:password@localhost/saga" -action migrate
   ```

3. **配置 Saga**
   ```yaml
   saga:
     enabled: true
     storage:
       type: postgres
       dsn: "postgres://user:password@localhost/saga"
     messaging:
       broker: nats
       url: "nats://localhost:4222"
   ```

4. **使用 Saga**
   ```go
   import "github.com/innovationmech/swit/pkg/saga"
   
   // 创建 Saga 定义
   def := saga.NewSagaDefinition(...)
   
   // 创建协调器
   coordinator := saga.NewCoordinator(...)
   
   // 执行 Saga
   instance, err := coordinator.Execute(ctx, def, data)
   ```

#### 破坏性变更

**无破坏性变更** - v0.9.0 完全向后兼容 v0.8.x。

#### 已废弃的功能

**无** - 所有现有功能继续保持支持。

## 贡献者

感谢所有为本版本做出贡献的开发者！

特别感谢 Saga 分布式事务系统的开发和文档完善工作。

## 相关链接

- **项目主页**: https://github.com/innovationmech/swit
- **完整文档**: https://innovationmech.github.io/swit/
- **问题反馈**: https://github.com/innovationmech/swit/issues
- **讨论区**: https://github.com/innovationmech/swit/discussions

---

格式约定：
- 🎉 重大特性
- ✨ 新功能
- 🐛 Bug 修复
- 📚 文档更新
- ⚡ 性能改进
- 🧪 测试相关
- 🛠️ 工具和命令
- 🚀 部署和运维
- ♻️ 代码重构
- 🔒 安全相关

本文档遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/) 规范。

