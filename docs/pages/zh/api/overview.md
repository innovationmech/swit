# API 概览

Swit 框架提供了丰富的 API 用于构建微服务应用。本文档提供了框架核心 API 的完整参考。

## 核心 API

### 服务器 API

服务器 API 是框架的核心，提供了服务生命周期管理、配置和依赖注入功能。

- [`BusinessServerCore`](/zh/api/server#businessservercore) - 主服务器接口
- [`ServerConfig`](/zh/api/server#serverconfig) - 服务器配置
- [`BusinessServiceRegistrar`](/zh/api/server#businessserviceregistrar) - 服务注册接口

### 传输层 API

传输层 API 管理 HTTP 和 gRPC 通信。

- [`TransportCoordinator`](/zh/api/transport#transportcoordinator) - 传输协调器
- [`NetworkTransport`](/zh/api/transport#networktransport) - 网络传输接口
- [`MultiTransportRegistry`](/zh/api/transport#multitransportregistry) - 多传输注册表

### 中间件 API

中间件 API 提供了请求处理的扩展点。

- [`Middleware`](/zh/api/middleware#middleware) - 中间件接口
- [`AuthMiddleware`](/zh/api/middleware#authmiddleware) - 认证中间件
- [`LoggingMiddleware`](/zh/api/middleware#loggingmiddleware) - 日志中间件

## 服务 API

### Swit Serve API

用户管理服务的完整 REST API。

- [用户管理](/zh/api/swit-serve#users)
- [角色权限](/zh/api/swit-serve#roles)
- [组织管理](/zh/api/swit-serve#organizations)

### Swit Auth API

认证和授权服务 API。

- [JWT 认证](/zh/api/swit-auth#jwt)
- [OAuth 2.0](/zh/api/swit-auth#oauth)
- [权限管理](/zh/api/swit-auth#permissions)

## API 约定

### 请求格式

所有 API 请求应使用 JSON 格式：

```json
{
  "field1": "value1",
  "field2": "value2"
}
```

### 响应格式

标准响应格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 错误处理

错误响应格式：

```json
{
  "code": 400,
  "message": "Invalid request",
  "error": {
    "field": "username",
    "reason": "required"
  }
}
```

## 版本控制

API 使用 URL 路径版本控制：

- `/api/v1/` - 版本 1
- `/api/v2/` - 版本 2（未来）

## 认证

大多数 API 需要认证。支持以下认证方式：

- **Bearer Token**: `Authorization: Bearer <token>`
- **API Key**: `X-API-Key: <key>`

## 速率限制

API 请求受到速率限制保护：

- 默认限制：每分钟 100 个请求
- 认证用户：每分钟 1000 个请求

## 更多信息

- [完整 API 参考](/zh/api/server)
- [OpenAPI 规范](https://github.com/innovationmech/swit/blob/main/api/openapi.yaml)
- [Postman 集合](https://github.com/innovationmech/swit/blob/main/api/postman.json)