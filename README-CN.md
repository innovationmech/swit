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
2. **swit-auth** - 认证服务（端口 9001）
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
**HTTP 端口**: 9000  
**gRPC 端口**: 10000 (HTTP 端口 + 1000)  
**协议**: gRPC + HTTP  
**实现的方法**:
- `SayHello` - 简单问候功能 ✅

**HTTP 端点**:
- `POST /v1/greeter/hello` - 发送问候请求

**gRPC 端点**:
- `greeter.v1.GreeterService/SayHello` - gRPC 问候服务

### 2. Auth 服务 (HTTP REST)
**端口**: 9001  
**协议**: HTTP REST + gRPC  
**主要功能**:
- 用户登录/登出
- JWT Token 管理
- Token 刷新和验证

### 3. User 服务 (HTTP REST)
**HTTP 端口**: 9000  
**gRPC 端口**: 10000 (HTTP 端口 + 1000)  
**协议**: HTTP REST + gRPC  
**主要功能**:
- 用户注册
- 用户信息管理
- 用户查询和删除

## 环境要求

- Go 1.24+
- MySQL 8.0+
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
CREATE DATABASE user_service_db;
CREATE DATABASE auth_service_db;

# 导入数据库结构
mysql -u root -p user_service_db < scripts/sql/user_service_db.sql
mysql -u root -p auth_service_db < scripts/sql/auth_service_db.sql
```

### 5. 构建和运行
```bash
# 构建所有服务（开发模式）
make build

# 或快速构建（跳过质量检查）
make build-dev

# 运行认证服务
./bin/swit-auth

# 运行用户服务
./bin/swit-serve
```

## API 开发

### 工具链设置
```bash
# 设置 protobuf 开发环境
make proto-setup

# 设置 swagger 开发环境
make swagger-setup
```

### 日常开发命令

#### Protobuf 开发
```bash
# 完整的 protobuf 工作流（推荐）
make proto

# 快速开发模式（跳过依赖）
make proto-dev

# 高级 proto 操作
make proto-advanced OPERATION=format
make proto-advanced OPERATION=lint
make proto-advanced OPERATION=breaking
make proto-advanced OPERATION=clean
make proto-advanced OPERATION=docs
```

#### Swagger 文档
```bash
# 生成 swagger 文档（推荐）
make swagger

# 快速开发模式（跳过格式化）
make swagger-dev

# 高级 swagger 操作
make swagger-advanced OPERATION=format
make swagger-advanced OPERATION=switserve
make swagger-advanced OPERATION=switauth
make swagger-advanced OPERATION=clean
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
  port: 9000

database:
  host: 127.0.0.1
  port: 3306
  username: root
  password: root
  dbname: user_service_db

serviceDiscovery:
  address: "localhost:8500"
```

### 认证服务配置 (switauth.yaml)
```yaml
server:
  port: 9001
  grpcPort: 50051

database:
  host: 127.0.0.1
  port: 3306
  username: root
  password: root
  dbname: auth_service_db

serviceDiscovery:
  address: "localhost:8500"
```

## Docker 部署

### 构建镜像
```bash
make docker
```

### 运行容器
```bash
# 运行用户服务
docker run -d -p 9000:9000 -p 10000:10000 --name swit-serve swit-serve:latest

# 运行认证服务
docker run -d -p 9001:9001 --name swit-auth swit-auth:latest
```

### 使用 Docker Compose
```bash
docker-compose up -d
```

## 测试

### 运行所有测试
```bash
make test
```

### 快速开发测试
```bash
make test-dev
```

### 测试覆盖率
```bash
make test-coverage
```

### 高级测试
```bash
# 运行特定类型的测试
make test-advanced TYPE=unit
make test-advanced TYPE=race
make test-advanced TYPE=bench

# 运行特定包的测试
make test-advanced TYPE=unit PACKAGE=internal
make test-advanced TYPE=unit PACKAGE=pkg
```

## 开发环境

### 设置开发环境
```bash
# 完整开发环境设置（推荐）
make setup-dev

# 快速设置（最小必要组件）
make setup-quick
```

### 开发工具

#### 代码质量
```bash
# 标准质量检查（推荐用于CI/CD）
make quality

# 快速质量检查（开发时使用）
make quality-dev

# 设置质量检查工具
make quality-setup
```

#### 代码格式化和检查
```bash
# 格式化代码
make format

# 检查代码
make vet

# 代码规范检查
make lint

# 安全扫描
make security
```

#### 依赖管理
```bash
# 整理Go模块依赖
make tidy
```

### 构建命令

#### 标准构建
```bash
# 构建所有服务（开发模式）
make build

# 快速构建（跳过质量检查）
make build-dev

# 发布构建（所有平台）
make build-release
```

#### 高级构建
```bash
# 为特定平台构建特定服务
make build-advanced SERVICE=swit-serve PLATFORM=linux/amd64
make build-advanced SERVICE=swit-auth PLATFORM=darwin/arm64
```

### 清理

```bash
# 标准清理（所有生成文件）
make clean

# 快速清理（仅构建输出）
make clean-dev

# 深度清理（重置环境）
make clean-setup

# 高级清理（特定类型）
make clean-advanced TYPE=build
make clean-advanced TYPE=proto
make clean-advanced TYPE=swagger
```

### CI/CD 和版权管理

#### CI 流水线
```bash
# 运行 CI 流水线（自动化测试和质量检查）
make ci
```

#### 版权管理
```bash
# 检查并修复版权声明
make copyright

# 仅检查版权声明
make copyright-check

# 为新项目设置版权
make copyright-setup
```

### Docker 开发

```bash
# 标准 Docker 构建（生产环境）
make docker

# 快速 Docker 构建（开发时使用缓存）
make docker-dev

# 设置 Docker 开发环境
make docker-setup

# 高级 Docker 操作
make docker-advanced OPERATION=build COMPONENT=images SERVICE=auth
```

## Makefile 命令参考

项目使用了完善的 Makefile 系统，命令组织清晰。以下是快速参考：

### 核心开发命令
```bash
make all              # 完整构建流水线 (proto + swagger + tidy + copyright + build)
make setup-dev        # 设置完整开发环境
make setup-quick      # 快速设置（最小组件）
make ci               # 运行 CI 流水线
```

### 构建命令
```bash
make build            # 标准构建（开发模式）
make build-dev        # 快速构建（跳过质量检查）
make build-release    # 发布构建（所有平台）
make build-advanced   # 高级构建（需要 SERVICE 和 PLATFORM 参数）
```

### 测试命令
```bash
make test             # 运行所有测试（包含依赖生成）
make test-dev         # 快速开发测试
make test-coverage    # 生成覆盖率报告
make test-advanced    # 高级测试（需要 TYPE 和 PACKAGE 参数）
```

### 质量检查命令
```bash
make quality          # 标准质量检查（CI/CD）
make quality-dev      # 快速质量检查（开发）
make quality-setup    # 设置质量检查工具
make tidy             # 整理 Go 模块
make format           # 格式化代码
make vet              # 代码检查
make lint             # 代码规范检查
make security         # 安全扫描
```

### API 开发命令
```bash
make proto            # 生成 protobuf 代码
make proto-dev        # 快速 proto 生成
make proto-setup      # 设置 protobuf 工具
make swagger          # 生成 swagger 文档
make swagger-dev      # 快速 swagger 生成
make swagger-setup    # 设置 swagger 工具
```

### 清理命令
```bash
make clean            # 标准清理（所有生成文件）
make clean-dev        # 快速清理（仅构建输出）
make clean-setup      # 深度清理（重置环境）
make clean-advanced   # 高级清理（需要 TYPE 参数）
```

### Docker 命令
```bash
make docker           # 标准 Docker 构建
make docker-dev       # 快速 Docker 构建（使用缓存）
make docker-setup     # 设置 Docker 开发环境
make docker-advanced  # 高级 Docker 操作
```

### 版权管理命令
```bash
make copyright        # 检查并修复版权声明
make copyright-check  # 仅检查版权声明
make copyright-setup  # 为新项目设置版权
```

### 帮助命令
```bash
make help             # 显示所有可用命令及说明
```

如需了解详细的命令选项和参数，请运行 `make help` 或参考 `scripts/mk/` 目录下的具体 `.mk` 文件。


## 相关文档

- [开发指南](DEVELOPMENT-CN.md)
- [API 文档](api/docs/README.md)

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件