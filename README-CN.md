# Swit

Swit 是一个基于 Go 语言开发的微服务架构后端系统，采用模块化设计，提供用户管理、身份认证和服务发现等功能。项目使用 Gin 框架处理 HTTP 请求，GORM 进行数据持久化，并支持 gRPC 协议进行服务间通信。

## 核心特性

- **微服务架构**: 采用模块化设计，支持独立部署和扩展
- **身份认证**: 基于 JWT 的完整认证系统，支持 token 刷新
- **用户管理**: 完整的用户 CRUD 操作和权限管理
- **服务发现**: 集成 Consul 进行服务注册与发现
- **数据库支持**: 使用 MySQL 进行数据持久化
- **协议支持**: 同时支持 HTTP REST API 和 gRPC 协议
- **健康检查**: 内置健康检查端点用于监控服务状态
- **Docker 支持**: 提供容器化部署方案
- **OpenAPI 文档**: 集成 Swagger UI，提供交互式 API 文档

## 系统架构

项目包含以下主要组件：

1. **swit-serve** - 主要用户服务（端口 9000）
2. **swit-auth** - 认证服务（端口 9001）
3. **switctl** - 命令行控制工具

## 快速开始

### 环境要求

- Go 1.22 或更高版本
- MySQL 5.7+ 或 8.0+
- Consul（可选，用于服务发现）

### 安装步骤

1. 克隆项目：
   ```bash
   git clone https://github.com/innovationmech/swit.git
   cd swit
   ```

2. 初始化数据库：
   ```bash
   # 执行数据库脚本
   mysql -u root -p < scripts/sql/user_service_db.sql
   mysql -u root -p < scripts/sql/auth_service_db.sql
   ```

3. 配置服务：
   - 编辑 `swit.yaml` 配置主服务数据库和端口
   - 编辑 `switauth.yaml` 配置认证服务数据库和端口

### 构建项目

构建所有服务：
```bash
make build
```

或者分别构建：
```bash
make build-serve    # 构建主服务
make build-auth     # 构建认证服务
make build-ctl      # 构建控制工具
```

构建完成后，二进制文件将生成在 `_output/` 目录下。

### 运行服务

1. 启动认证服务：
   ```bash
   ./_output/swit-auth/swit-auth start
   ```

2. 启动主服务：
   ```bash
   ./_output/swit-serve/swit-serve serve
   ```

3. 使用控制工具检查服务状态：
   ```bash
   ./_output/switctl/switctl health
   ```

## API 接口

### OpenAPI/Swagger 文档

项目集成了 Swagger UI，提供交互式 API 文档。启动服务后，可以通过以下地址访问：

```
http://localhost:9000/swagger/index.html
```

### 认证服务 API (端口 9001)

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/auth/login` | 用户登录 |
| POST | `/auth/logout` | 用户登出 |
| POST | `/auth/refresh` | 刷新访问令牌 |
| GET | `/auth/validate` | 验证令牌有效性 |
| GET | `/health` | 健康检查 |

### 用户服务 API (端口 9000)

| 方法 | 路径 | 描述 | 需要认证 |
|------|------|------|----------|
| POST | `/api/v1/users/create` | 创建用户 | ✓ |
| GET | `/api/v1/users/username/:username` | 根据用户名获取用户 | ✓ |
| GET | `/api/v1/users/email/:email` | 根据邮箱获取用户 | ✓ |
| DELETE | `/api/v1/users/:id` | 删除用户 | ✓ |
| POST | `/internal/validate-user` | 验证用户凭证（内部接口） | ✗ |
| GET | `/health` | 健康检查 | ✗ |
| POST | `/stop` | 停止服务 | ✗ |

### gRPC 服务

项目还提供 gRPC 接口，定义在 `api/proto/greeter.proto` 中：
- `SayHello`: 简单的问候服务

## 配置说明

### 主服务配置 (swit.yaml)
```yaml
database:
  host: 127.0.0.1
  port: 3306
  username: root
  password: root
  dbname: user_service_db
server:
  port: 9000
serviceDiscovery:
  address: "localhost:8500"
```

### 认证服务配置 (switauth.yaml)
```yaml
database:
  host: 127.0.0.1
  port: 3306
  username: root
  password: root
  dbname: auth_service_db
server:
  port: 9001
serviceDiscovery:
  address: "localhost:8500"
```

## 开发指南

### 项目结构

```
swit/
├── cmd/                    # 应用入口点
│   ├── swit-serve/        # 主服务
│   ├── swit-auth/         # 认证服务
│   └── switctl/           # 控制工具
├── internal/              # 内部包
│   ├── switserve/         # 主服务内部逻辑
│   ├── switauth/          # 认证服务内部逻辑
│   └── switctl/           # 控制工具内部逻辑
├── api/                   # API 定义
│   └── proto/             # gRPC 协议定义
├── pkg/                   # 公共包
├── scripts/               # 脚本文件
│   └── sql/               # 数据库脚本
└── docs/                  # 文档
```

### 可用的 Make 命令

- `make tidy`: 整理 Go 模块依赖
- `make build`: 构建所有二进制文件
- `make clean`: 清理输出目录
- `make test`: 运行测试
- `make test-coverage`: 运行测试并生成覆盖率报告
- `make test-race`: 运行竞态条件检测测试
- `make ci`: 运行完整的 CI 流程（tidy, copyright, quality, test）
- `make image-serve`: 构建 Docker 镜像
- `make swagger`: 生成/更新 OpenAPI 文档
- `make swagger-install`: 安装 Swagger 文档生成工具
- `make swagger-fmt`: 格式化 Swagger 注释

### Docker 支持

构建 Docker 镜像：
```bash
make image-serve
```

运行容器：
```bash
docker run -d -p 9000:9000 -v ./swit.yaml:/root/swit.yaml swit-serve:master
```

## 数据库模式

### 用户表 (user_service_db.users)
- `id`: UUID 主键
- `username`: 用户名（唯一）
- `email`: 邮箱地址（唯一）
- `password_hash`: 加密后的密码
- `role`: 用户角色
- `is_active`: 是否激活
- `created_at`/`updated_at`: 时间戳

### 令牌表 (auth_service_db.tokens)
- `id`: UUID 主键
- `user_id`: 关联用户ID
- `access_token`: 访问令牌
- `refresh_token`: 刷新令牌
- `access_expires_at`: 访问令牌过期时间
- `refresh_expires_at`: 刷新令牌过期时间
- `is_valid`: 令牌是否有效

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

---

[English Documentation](README.md) 