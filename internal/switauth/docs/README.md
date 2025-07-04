# SWIT Auth OpenAPI Documentation

## 概述

本目录包含 SWIT Auth 认证服务的 OpenAPI/Swagger 文档。这些文档是通过 [swaggo/swag](https://github.com/swaggo/swag) 工具自动生成的。

## 文档文件

- `docs.go` - Go 代码格式的文档（用于集成到应用程序中）
- `swagger.json` - JSON 格式的 OpenAPI 规范
- `swagger.yaml` - YAML 格式的 OpenAPI 规范

## 访问 Swagger UI

当 SWIT Auth 服务运行时，你可以通过以下 URL 访问 Swagger UI：

```
http://localhost:8090/swagger/index.html
```

## 更新文档

当你修改了 API 接口或添加了新的端点时，需要重新生成文档：

```bash
make swagger-auth
```

## 格式化 Swagger 注释

如果需要格式化代码中的 Swagger 注释：

```bash
make swagger-fmt-auth
```

## API 端点

### 公开端点（无需认证）
- `GET /health` - 健康检查
- `POST /auth/login` - 用户登录
- `POST /auth/refresh` - 刷新访问令牌

### 需要认证的端点
- `GET /auth/validate` - 验证访问令牌
- `POST /auth/logout` - 用户退出登录

## 认证

需要认证的端点使用 Bearer Token 认证。在请求头中添加：

```
Authorization: Bearer <your-jwt-token>
```

## API 使用示例

### 用户登录

```bash
curl -X POST http://localhost:8090/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "password": "password123"
  }'
```

响应：
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### 验证令牌

```bash
curl -X GET http://localhost:8090/auth/validate \
  -H "Authorization: Bearer <your-access-token>"
```

响应：
```json
{
  "message": "Token is valid",
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 刷新令牌

```bash
curl -X POST http://localhost:8090/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "<your-refresh-token>"
  }'
```

响应：
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### 用户退出

```bash
curl -X POST http://localhost:8090/auth/logout \
  -H "Authorization: Bearer <your-access-token>"
```

响应：
```json
{
  "message": "Logged out successfully"
}
```

## 开发提示

1. 在编写新的 API 端点时，请确保添加完整的 Swagger 注释
2. 使用 `@Summary` 提供简短描述
3. 使用 `@Description` 提供详细说明
4. 使用 `@Tags` 对 API 进行分组
5. 使用 `@Security` 标记需要认证的端点
6. 为请求和响应提供示例值

## 示例 Swagger 注释

```go
// Login authenticates a user and returns access and refresh tokens
// @Summary User login
// @Description Authenticate a user with username and password, returns access and refresh tokens
// @Tags authentication
// @Accept json
// @Produce json
// @Param login body model.LoginRequest true "User login credentials"
// @Success 200 {object} model.LoginResponse "Login successful"
// @Failure 400 {object} model.ErrorResponse "Bad request"
// @Failure 401 {object} model.ErrorResponse "Invalid credentials"
// @Router /auth/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
    // implementation
}
```

## 错误处理

所有 API 端点都遵循统一的错误响应格式：

```json
{
  "error": "错误描述信息"
}
```

常见的 HTTP 状态码：
- `200` - 成功
- `400` - 请求参数错误
- `401` - 未授权或认证失败
- `500` - 服务器内部错误 