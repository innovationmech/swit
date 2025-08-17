---
title: API 文档
description: Swit 框架 API 接口文档
---

# API 文档

欢迎使用 Swit 框架 API 文档。以下是可用的服务和接口说明。

## 服务列表

<div class="service-grid">

<div class="service-card">

### [SWIT Server API](./switserve.md)

This is the SWIT server API documentation.

<div class="service-stats">

- **端点数量**: 5
- **版本**: 1.0
- **基础URL**: `http://localhost:8080/`

</div>

[查看文档 →](./switserve.md)

</div>

<div class="service-card">

### [SWIT Auth API](./switauth.md)

This is the SWIT authentication service API documentation.

<div class="service-stats">

- **端点数量**: 5
- **版本**: 1.0
- **基础URL**: `http://localhost:8090/`

</div>

[查看文档 →](./switauth.md)

</div>

</div>

## 快速链接

- [完整 API 参考](./complete.md)
- [概览](./overview.md)
- [认证服务](./switauth.md)
- [用户管理服务](./switserve.md)

## 通用信息

### 认证

所有 API 接口都需要适当的身份验证。大多数接口使用 Bearer Token 认证：

```http
Authorization: Bearer <your_access_token>
```

### 请求格式

- **Content-Type**: `application/json`
- **Accept**: `application/json`
- **编码**: UTF-8

### 错误处理

API 使用标准 HTTP 状态码来表示请求的成功或失败状态：

| 状态码 | 说明 | 示例 |
|--------|------|------|
| 200 | 请求成功 | 数据获取成功 |
| 201 | 创建成功 | 用户创建成功 |
| 400 | 请求错误 | 参数格式错误 |
| 401 | 未授权 | Token 无效或过期 |
| 403 | 禁止访问 | 权限不足 |
| 404 | 资源不存在 | 用户不存在 |
| 429 | 请求过多 | 触发速率限制 |
| 500 | 服务器内部错误 | 系统异常 |

### 响应格式

所有响应都采用统一的 JSON 格式：

```json
{
  "status": "success|error",
  "data": {},
  "message": "响应消息",
  "timestamp": "2023-12-01T12:00:00Z"
}
```

<style>
.service-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1rem;
  margin: 1rem 0;
}

.service-card {
  border: 1px solid var(--vp-c-border);
  border-radius: 8px;
  padding: 1.5rem;
  background: var(--vp-c-bg-soft);
}

.service-card h3 {
  margin-top: 0;
  color: var(--vp-c-brand-1);
}

.service-stats {
  margin: 1rem 0;
  font-size: 0.9em;
}

.service-card > p:last-child {
  margin-bottom: 0;
  text-align: right;
  font-weight: 500;
}

.service-card a {
  text-decoration: none;
}

.service-card a:hover {
  text-decoration: underline;
}
</style>

