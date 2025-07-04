# SWIT 项目文档

欢迎来到 SWIT（Simple Web Interface Toolkit）项目文档中心。

## 项目概述

SWIT 是一个现代化的微服务框架，提供用户认证、内容管理等核心功能。

## 快速开始

- [快速开始指南](./quick-start-example.md) - 5分钟内运行项目
- [开发环境设置](../DEVELOPMENT.md)

## 架构文档

- [路由注册系统](./route-registry-guide.md) - 了解路由注册机制
- [OpenAPI 集成](./openapi-integration.md) - API文档生成和集成

## 服务文档

### 📊 API文档快速入口
- **[统一API文档汇总](./generated/README.md)** - 🔗 所有服务的API文档统一访问入口

### 微服务架构

| 服务 | 功能描述 | API文档 | 默认端口 |
|------|----------|---------|----------|
| **switserve** | 用户管理、内容服务 | [Swagger UI](http://localhost:9000/swagger/index.html) | 9000 |
| **switauth** | 认证授权服务 | [Swagger UI](http://localhost:8080/swagger/index.html) | 8080 |

### 详细文档导航

#### 📋 API 规范文档
- [**API文档汇总**](./generated/README.md) - 所有服务的生成文档统一入口
- [SwitServe 生成文档](../internal/switserve/docs/) - Swagger自动生成的API规范
- [SwitAuth 生成文档](../internal/switauth/docs/) - 认证服务API规范（待完善）

#### 📖 使用指南
- [SwitServe API 指南](./services/switserve/README.md) - 用户管理和内容服务使用指南
- [SwitAuth API 指南](./services/switauth/README.md) - 认证授权服务使用指南
- [服务API导航](./services/README.md) - 微服务架构和跨服务通信说明

## 开发指南

- [API 开发规范](./development/api-guidelines.md)
- [代码贡献指南](./development/contributing.md)
- [测试指南](./development/testing.md)

## 部署和运维

- [部署指南](./deployment.md)
- [监控和日志](./monitoring.md)
- [故障排查](./troubleshooting.md)

## API 访问

### 开发环境
- **SwitServe API**: http://localhost:9000
- **SwitAuth API**: http://localhost:8080

### API 文档界面
- **SwitServe Swagger UI**: http://localhost:9000/swagger/index.html
- **SwitAuth Swagger UI**: http://localhost:8080/swagger/index.html

## 相关链接

- [项目仓库](https://github.com/innovationmech/swit)
- [问题反馈](https://github.com/innovationmech/swit/issues)
- [讨论区](https://github.com/innovationmech/swit/discussions) 