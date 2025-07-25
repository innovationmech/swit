# SWIT 服务API文档导航

本目录包含 SWIT 项目中各个微服务的API文档和使用指南。所有服务都基于统一的 ServiceHandler 架构模式，提供完整的服务生命周期管理。

## 服务架构

```mermaid
graph TB
    A[客户端应用] --> B[API网关]
    B --> C[SwitAuth ServiceHandler]
    B --> D[SwitServe ServiceHandler]
    
    C --> E[Auth Service Layer]
    D --> F[User Service Layer]
    
    E --> G[Auth Database]
    F --> H[User Database]
    
    I[EnhancedServiceRegistry] --> C
    I --> D
    
    C -.-> J[JWT Token验证]
    D -.-> J
    
    K[Health Monitoring] --> C
    K --> D
```

### ServiceHandler 架构特性
- **统一接口**: 所有服务实现 ServiceHandler 接口
- **生命周期管理**: 支持初始化、健康检查、优雅关闭
- **服务注册**: 使用 EnhancedServiceRegistry 管理服务
- **接口分离**: 业务接口与实现完全分离
- **类型安全**: 共享 types 包确保类型一致性

## 可用服务

### 🔐 SwitAuth - 认证授权服务
- **功能**: 用户登录、Token管理、权限验证
- **架构**: 基于 ServiceHandler 模式，支持完整生命周期管理
- **端口**: 9001
- **文档**: [详细文档](./switauth/README.md)
- **API**: http://localhost:9001/swagger/index.html

**主要端点**:
- `POST /auth/login` - 用户登录
- `POST /auth/refresh` - 刷新Token
- `GET /auth/validate` - 验证Token
- `POST /auth/logout` - 用户登出

### 👥 SwitServe - 用户管理服务
- **功能**: 用户CRUD操作、用户信息管理
- **架构**: 基于 ServiceHandler 模式，实现接口分离和依赖注入优化
- **端口**: 9000  
- **文档**: [详细文档](./switserve/README.md)
- **API**: http://localhost:9000/swagger/index.html

**主要端点**:
- `POST /users/create` - 创建用户
- `GET /users/username/{username}` - 获取用户信息
- `GET /users/email/{email}` - 通过邮箱获取用户
- `DELETE /users/{id}` - 删除用户

## 跨服务通信

### 认证流程
1. 客户端调用 SwitAuth 进行登录认证
2. SwitAuth 返回 JWT Token
3. 客户端使用 Token 调用 SwitServe API
4. SwitServe 调用 SwitAuth 验证 Token

### 内部API
- `POST /internal/validate-user` - SwitServe内部用户验证接口

## 开发工具

### 生成API文档
```bash
# 生成 SwitServe 文档
make swagger-switserve

# 生成 SwitAuth 文档  
make swagger-switauth

# 生成所有服务文档
make swagger
```

### 测试API
```bash
# 健康检查
curl http://localhost:9000/health
curl http://localhost:9001/health

# 查看API版本
curl http://localhost:9000/version
curl http://localhost:9001/version
```

## 环境配置

各服务的配置文件：
- **SwitServe**: `switblog.yaml`
- **SwitAuth**: `switauth.yaml`
- **通用配置**: `swit.yaml`