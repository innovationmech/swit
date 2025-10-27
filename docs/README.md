# SWIT 项目文档

欢迎来到 SWIT（Simple Web Interface Toolkit）项目文档中心。

## 项目概述

SWIT 是一个现代化的微服务框架，提供用户认证、内容管理等核心功能。

## 快速开始

- [开发环境设置](../DEVELOPMENT.md)

## 架构文档

### 🏗️ 核心架构
- [服务开发指南](./service-development-guide.md) - **ServiceHandler 模式开发指南**
- [服务架构分析](./service-architecture-analysis.md) - ServiceHandler 架构深度分析
- [Kafka 处理-生产事务模式](./architecture/kafka-process-and-produce.md) - Kafka 事务流程与 Exactly-Once 实践


## 服务文档

### 📊 API文档快速入口
- **[统一API文档汇总](./generated/)** - 🔗 所有服务的API文档统一访问入口

### 微服务架构

| 服务 | 功能描述 | API文档 | 默认端口 |
|------|----------|---------|----------|
| **switserve** | 用户管理、内容服务 | [Swagger UI](http://localhost:9000/swagger/index.html) | 9000 |
| **switauth** | 认证授权服务 | [Swagger UI](http://localhost:9001/swagger/index.html) | 9001 |

### 详细文档导航

#### 📋 API 规范文档
- [**API文档汇总**](./generated/) - 所有服务的生成文档统一入口
- [SwitServe 生成文档](./generated/switserve/) - Swagger自动生成的API规范
- [SwitAuth 生成文档](./generated/switauth/) - 认证服务API规范

#### 📖 使用指南
- [SwitServe API 指南](./services/switserve/README.md) - 用户管理和内容服务使用指南
- [SwitAuth API 指南](./services/switauth/README.md) - 认证授权服务使用指南
- [服务API导航](./services/README.md) - 微服务架构和跨服务通信说明

## 开发指南

- [开发环境设置](../DEVELOPMENT-CN.md) - 中文开发指南
- [Development Guide](../DEVELOPMENT.md) - 英文开发指南

## Saga 分布式事务

### 📚 用户文档
- [Saga 用户指南](./saga-user-guide.md) - Saga 使用指南和最佳实践
- [Saga DSL 指南](./saga-dsl-guide.md) - DSL 定义和使用
- [Saga DSL 参考](./saga-dsl-reference.md) - DSL 语法参考
- [Saga 监控指南](./saga-monitoring-guide.md) - 监控和告警配置
- [Saga 安全指南](./saga-security-guide.md) - 安全配置和最佳实践
- [Saga Dashboard 指南](./saga-dashboard-guide.md) - Dashboard 使用指南

### 🧪 测试文档
- **[Saga 测试指南](./saga-testing-guide.md)** - 全面的测试指南，包括单元测试、集成测试、性能测试和混沌测试
- **[测试覆盖率报告](./saga-test-coverage.md)** - 详细的覆盖率统计、分析和改进建议 (73.6%)
- **[性能基准报告](./saga-performance-benchmarks.md)** - 吞吐量、延迟、资源使用和优化建议
- [测试工具包文档](../pkg/saga/testing/README.md) - Mock、构建器、断言和 Fixtures

### 🔧 技术文档
- [Saga 数据库迁移](./saga-database-migrations.md) - 数据库迁移指南
- [Saga 数据库设计](./saga-database-schema-design.md) - 数据库架构设计
- [Saga 恢复监控](./saga-recovery-monitoring.md) - 故障恢复和监控
- [Saga 敏感数据保护](./saga-sensitive-data-protection.md) - 数据保护策略
- [Saga Dashboard API](./saga-dashboard-api.md) - Dashboard API 规范

## API 访问

### 开发环境
- **SwitServe API**: http://localhost:9000
- **SwitAuth API**: http://localhost:9001

### API 文档界面
- **SwitServe Swagger UI**: http://localhost:9000/swagger/index.html
- **SwitAuth Swagger UI**: http://localhost:9001/swagger/index.html

## 相关链接

- [项目仓库](https://github.com/innovationmech/swit)
- [问题反馈](https://github.com/innovationmech/swit/issues)
- [讨论区](https://github.com/innovationmech/swit/discussions)
