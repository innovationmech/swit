# Swit

Swit 是一个基于 Go 语言开发的微服务架构后端系统，采用模块化设计，提供用户管理、身份认证和服务发现等功能。项目使用 Gin 框架处理 HTTP 请求，GORM 进行数据持久化，并支持 gRPC 协议进行服务间通信。

## 核心特性

- **微服务架构**: 采用模块化设计，支持独立部署和扩展
- **身份认证**: 基于 JWT 的完整认证系统，支持 token 刷新
- **用户管理**: 完整的用户 CRUD 操作和权限管理
- **服务发现**: 集成 Consul 进行服务注册与发现
- **数据库支持**: 使用 MySQL 进行数据持久化
- **双协议支持**: 同时支持 HTTP REST API 和 gRPC 协议
- **现代化 API**: 使用 Buf 工具链管理 gRPC API，支持版本化和自动文档生成
- **健康检查**: 内置健康检查端点用于监控服务状态
- **Docker 支持**: 提供容器化部署方案
- **OpenAPI 文档**: 集成 Swagger UI，提供交互式 API 文档

## 系统架构

项目包含以下主要组件：

1. **swit-serve** - 主要用户服务（端口 9000）
2. **swit-auth** - 认证服务（端口 8090）
3. **switctl** - 命令行控制工具

## API 现代化架构

项目已完成 API 现代化迁移，采用 Buf 工具链管理 gRPC API：

```
api/
├── buf.yaml              # Buf 主配置文件
├── buf.gen.yaml          # 代码生成配置
├── buf.lock              # 依赖锁文件
├── proto/                # Protocol Buffer 定义
│   └── swit/
│       └── v1/          # API 版本 1
│           └── greeter/ # 问候服务
├── gen/                 # 生成的代码
│   ├── go/             # Go 代码
│   └── openapiv2/      # OpenAPI 文档
└── docs/               # 文档
    ├── README.md       # API 使用文档
    └── MIGRATION.md    # 迁移指南
```

### API 设计原则

- **版本化**: 所有 API 都有明确的版本号（v1, v2, ...）
- **模块化**: 按服务域组织 proto 文件
- **双协议**: 同时支持 gRPC 和 HTTP/REST
- **自动化**: 使用 Buf 工具链自动生成代码和文档

## 当前实现的服务

### 1. Greeter 服务 (gRPC)
**端口**: 9000  
**协议**: gRPC + HTTP  
**实现的方法**:
- `SayHello` - 简单问候功能 ✅

**HTTP 端点**:
- `POST /v1/greeter/hello` - 发送问候请求

### 2. Auth 服务 (HTTP REST)
**端口**: 8090  
**协议**: HTTP REST  
**主要功能**:
- 用户登录/登出
- JWT Token 管理
- Token 刷新和验证

### 3. User 服务 (HTTP REST)
**端口**: 9000  
**协议**: HTTP REST  
**主要功能**:
- 用户注册
- 用户信息管理
- 用户查询和删除

## 环境要求

- Go 1.24+
- MySQL 8.0+
- Redis 6.0+
- Consul 1.12+
- Buf CLI 1.0+ (用于 API 开发)

## 快速开始

### 1. 克隆仓库
```bash
git clone https://github.com/innovationmech/swit.git
cd swit
```

### 2. 安装依赖
```bash
go mod download
```

### 3. 环境配置
```bash
# 复制配置文件
cp swit.yaml.example swit.yaml
cp switauth.yaml.example switauth.yaml

# 编辑配置文件，填入数据库等信息
```

### 4. 数据库初始化
```bash
# 创建数据库
mysql -u root -p
CREATE DATABASE swit_db;
CREATE DATABASE swit_auth_db;

# 导入数据库结构
mysql -u root -p swit_db < scripts/sql/user_service_db.sql
mysql -u root -p swit_auth_db < scripts/sql/auth_service_db.sql
```

### 5. 构建和运行
```bash
# 构建所有服务
make build

# 运行认证服务
./bin/swit-auth

# 运行用户服务
./bin/swit-serve
```

## API 开发

### 工具链设置
```bash
# 安装 Buf CLI
make buf-install

# 设置开发环境
make proto-setup
```

### 日常开发命令
```bash
# 完整的 protobuf 工作流
make proto

# 仅生成代码
make proto-generate

# 检查 proto 文件
make proto-lint

# 格式化 proto 文件
make proto-format

# 检查破坏性变更
make proto-breaking

# 清理生成的代码
make proto-clean

# 查看 OpenAPI 文档
make proto-docs
```

### API 开发工作流

1. **修改 proto 文件**
   ```bash
   # 编辑 proto 文件
   vim api/proto/swit/v1/greeter/greeter.proto
   ```

2. **生成代码**
   ```bash
   make proto-generate
   ```

3. **实现服务**
   ```bash
   # 在相应的服务文件中实现 gRPC 方法
   vim internal/switserve/service/greeter.go
   ```

4. **测试和验证**
   ```bash
   make proto-lint
   make build
   make test
   ```

## 配置说明

### 主服务配置 (swit.yaml)
```yaml
server:
  host: "0.0.0.0"
  port: 9000
  grpc_port: 9001

database:
  host: "localhost"
  port: 3306
  name: "swit_db"
  username: "root"
  password: "password"

consul:
  address: "127.0.0.1:8500"
  datacenter: "dc1"
```

### 认证服务配置 (switauth.yaml)
```yaml
server:
  host: "0.0.0.0"
  port: 8090

database:
  host: "localhost"
  port: 3306
  name: "swit_auth_db"
  username: "root"
  password: "password"

jwt:
  secret: "your-secret-key"
  access_token_expiry: "15m"
  refresh_token_expiry: "7d"
```

## Docker 部署

### 构建镜像
```bash
make docker-build
```

### 运行容器
```bash
# 运行用户服务
docker run -d -p 9000:9000 -p 9001:9001 --name swit-serve swit-serve:latest

# 运行认证服务
docker run -d -p 8090:8090 --name swit-auth swit-auth:latest
```

### 使用 Docker Compose
```bash
docker-compose up -d
```

## 测试

### 运行单元测试
```bash
make test
```

### 运行集成测试
```bash
make test-integration
```

### 测试覆盖率
```bash
make test-coverage
```

## 开发工具

### 代码格式化
```bash
make fmt
```

### 代码检查
```bash
make lint
```

### 依赖检查
```bash
make deps
```

## 从旧版本升级

请参考 [API 迁移指南](api/MIGRATION.md) 了解详细的升级步骤。

主要变更：
- 新的 API 目录结构
- 使用 Buf 工具链管理 gRPC API
- 更新的包名和导入路径
- 新增的 HTTP/REST 端点支持

## 相关文档

- [开发指南](DEVELOPMENT-CN.md)
- [API 文档](api/docs/README.md)
- [API 迁移指南](api/MIGRATION.md)
- [路由注册指南](docs/route-registry-guide.md)
- [OpenAPI 集成](docs/openapi-integration.md)

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件 