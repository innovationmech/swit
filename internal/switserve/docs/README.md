# SWIT Server OpenAPI Documentation

## 概述

本目录包含 SWIT Server 的 OpenAPI/Swagger 文档。这些文档是通过 [swaggo/swag](https://github.com/swaggo/swag) 工具自动生成的。

## 文档文件

- `docs.go` - Go 代码格式的文档（用于集成到应用程序中）
- `swagger.json` - JSON 格式的 OpenAPI 规范
- `swagger.yaml` - YAML 格式的 OpenAPI 规范

## 访问 Swagger UI

当 SWIT Server 运行时，你可以通过以下 URL 访问 Swagger UI：

```
http://localhost:8080/swagger/index.html
```

## 更新文档

当你修改了 API 接口或添加了新的端点时，需要重新生成文档：

```bash
make swagger
```

## 格式化 Swagger 注释

如果需要格式化代码中的 Swagger 注释：

```bash
make swagger-fmt
```

## API 端点

### 公开端点（无需认证）
- `GET /health` - 健康检查
- `POST /users/create` - 创建新用户
- `POST /stop` - 停止服务器（管理员功能）

### 需要认证的端点
- `GET /users/username/{username}` - 根据用户名获取用户信息
- `GET /users/email/{email}` - 根据邮箱获取用户信息
- `DELETE /users/{id}` - 删除用户

### 内部 API
- `POST /internal/validate-user` - 验证用户凭据（供认证服务使用）

## 认证

需要认证的端点使用 Bearer Token 认证。在请求头中添加：

```
Authorization: Bearer <your-jwt-token>
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
// CreateUser creates a new user.
// @Summary Create a new user
// @Description Create a new user with username, email and password
// @Tags users
// @Accept json
// @Produce json
// @Param user body model.CreateUserRequest true "User information"
// @Success 201 {object} map[string]interface{} "User created successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/create [post]
func (uc *UserController) CreateUser(c *gin.Context) {
    // ...
}
``` 