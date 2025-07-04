# OpenAPI Integration for SWIT Project

## 概述

本文档描述了如何为 SWIT 项目的微服务添加 OpenAPI/Swagger 支持的实现过程。

## 服务实现状态

- ✅ **SwitServe** - 用户管理服务 (完整的OpenAPI支持)
- ✅ **SwitAuth** - 认证授权服务 (完整的OpenAPI支持)

## SwitServe OpenAPI 实现

### 1. 添加了 Swagger 依赖

在项目中添加了以下依赖：
- `github.com/swaggo/swag` - Swagger 文档生成工具
- `github.com/swaggo/gin-swagger` - Gin 框架的 Swagger 中间件
- `github.com/swaggo/files` - Swagger UI 静态文件

### 2. 为 API 端点添加了 Swagger 注释

为以下端点添加了完整的 Swagger 注释：

#### 用户管理 API
- `POST /users/create` - 创建新用户
- `GET /users/username/{username}` - 根据用户名获取用户
- `GET /users/email/{email}` - 根据邮箱获取用户
- `DELETE /users/{id}` - 删除用户

#### 内部 API
- `POST /internal/validate-user` - 验证用户凭据

#### 系统 API
- `GET /health` - 健康检查
- `POST /stop` - 停止服务器

### 3. 集成 Swagger UI

在路由配置中添加了 Swagger UI 端点：
```go
s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

### 4. 更新了 Makefile

添加了以下 Make 命令：
- `make swagger-switserve` - 生成/更新 SwitServe Swagger 文档
- `make swagger-fmt-switserve` - 格式化 SwitServe Swagger 注释

### 5. 生成的文档文件

文档生成在 `internal/switserve/docs/` 目录下：
- `docs.go` - Go 代码格式的文档
- `swagger.json` - JSON 格式的 OpenAPI 规范
- `swagger.yaml` - YAML 格式的 OpenAPI 规范

## SwitAuth OpenAPI 实现

### 1. 认证服务端点

为以下认证相关端点添加了完整的 Swagger 注释：

#### 认证 API
- `POST /auth/login` - 用户登录
- `POST /auth/logout` - 用户退出登录
- `POST /auth/refresh` - 刷新访问令牌
- `GET /auth/validate` - 验证访问令牌

#### 系统 API
- `GET /health` - 健康检查

### 2. 安全认证配置

配置了 Bearer Token 认证：
```go
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
```

### 3. 请求/响应模型

定义了完整的数据模型：
- `LoginRequest/LoginResponse`
- `RefreshTokenRequest/RefreshTokenResponse`
- `ValidateTokenResponse`
- `LogoutResponse`
- `ErrorResponse`

### 4. Makefile 集成

添加了以下命令：
- `make swagger-switauth` - 生成/更新 SwitAuth Swagger 文档
- `make swagger-fmt-switauth` - 格式化 SwitAuth Swagger 注释
- `make swagger` - 生成所有服务的文档

## 使用说明

### 生成文档

当修改了 API 或添加新端点后，运行以下命令重新生成文档：

```bash
# 生成所有服务文档
make swagger

# 生成特定服务文档
make swagger-switserve
make swagger-switauth
```

### 访问 Swagger UI

启动对应服务后，通过以下 URL 访问 Swagger UI：

**SwitServe**:
```
http://localhost:9000/swagger/index.html
```

**SwitAuth**:
```
http://localhost:8090/swagger/index.html
```

### 添加新的 API 文档

为新的 API 端点添加文档时，使用以下格式：

```go
// FunctionName does something
// @Summary 简短描述
// @Description 详细描述
// @Tags 标签分组
// @Accept json
// @Produce json
// @Param 参数名 参数位置 参数类型 是否必须 "参数描述"
// @Success 状态码 {类型} 返回类型 "描述"
// @Failure 状态码 {类型} 返回类型 "描述"
// @Security BearerAuth
// @Router /路径 [方法]
func FunctionName(c *gin.Context) {
    // 实现
}
```

## 注意事项

1. 确保在 main 文件中添加了基本的 Swagger 信息注释
2. 对于需要认证的端点，使用 `@Security BearerAuth` 标记
3. 为请求和响应模型添加示例值，提高文档的可读性
4. 定期运行 `make swagger` 更新文档

## 项目结构变化

```
internal/switserve/
├── docs/                  # SwitServe OpenAPI 文档目录
│   ├── docs.go           # 生成的文档代码
│   ├── swagger.json      # JSON 格式文档
│   ├── swagger.yaml      # YAML 格式文档
│   └── README.md         # 文档说明

internal/switauth/
├── docs/                  # SwitAuth OpenAPI 文档目录
│   ├── docs.go           # 生成的文档代码
│   ├── swagger.json      # JSON 格式文档
│   ├── swagger.yaml      # YAML 格式文档
│   └── README.md         # 文档说明
```

## 后续改进建议

1. 为响应模型创建专门的结构体，而不是使用 `map[string]interface{}`
2. 添加更多的示例和详细的错误响应说明
3. 考虑添加 API 版本控制
4. 添加请求/响应的验证规则说明
5. 集成 API 测试工具，如 Postman 集合生成
6. 添加 API 性能监控和指标收集 