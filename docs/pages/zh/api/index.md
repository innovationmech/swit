# API 文档

欢迎使用 Swit 框架 API 文档。以下是可用的服务和接口说明。

## 服务列表

### [SWIT Server API](./switserve.md)

This is the SWIT server API documentation.

- **接口数量**: 5
- **版本**: 1.0
- **基础URL**: http://localhost:8080/

### [SWIT Auth API](./switauth.md)

This is the SWIT authentication service API documentation.

- **接口数量**: 5
- **版本**: 1.0
- **基础URL**: http://localhost:8090/

## 通用信息

### 认证

所有 API 接口都需要适当的身份验证。请参考各个服务的具体认证要求。

### 错误处理

API 使用标准 HTTP 状态码来表示请求的成功或失败状态。

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求错误 |
| 401 | 未授权 |
| 403 | 禁止访问 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

