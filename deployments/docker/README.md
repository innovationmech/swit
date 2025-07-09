# Swit Docker 开发环境

这个目录包含了完整的 Swit 项目 Docker 开发环境，可以快速启动包括数据库、服务发现和应用服务在内的完整开发环境。

## 🚀 快速开始

### 前置要求

- Docker (>= 20.0)
- Docker Compose (>= 1.29)

### 一键启动

```bash
# 进入 docker 部署目录
cd deployments/docker

# 启动所有服务
./start.sh
```

### 手动启动

```bash
# 进入 docker 部署目录
cd deployments/docker

# 构建并启动所有服务
docker-compose up -d --build

# 查看服务状态
docker-compose ps
```

## 📋 服务组件

### 核心服务

| 服务名 | 容器名 | 端口 | 描述 |
|--------|---------|------|------|
| swit-serve | swit-serve | 9000 | 主要应用服务 |
| swit-auth | swit-auth | 9001 | 认证服务 |

### 基础设施服务

| 服务名 | 容器名 | 端口 | 描述 |
|--------|---------|------|------|
| mysql | swit-mysql | 3306 | MySQL 数据库 |
| consul | swit-consul | 8500, 8600 | 服务发现和配置中心 |

## 🌐 访问地址

- **Consul UI**: http://localhost:8500
- **认证服务**: http://localhost:9001
- **主要服务**: http://localhost:9000
- **MySQL 数据库**: localhost:3306 (用户名: root, 密码: root)

### 健康检查端点

- 认证服务: http://localhost:9001/health
- 主要服务: http://localhost:9000/health

## 🛠️ 管理命令

### 使用启动脚本（推荐）

```bash
# 启动服务
./start.sh start

# 停止服务
./start.sh stop

# 重启服务
./start.sh restart

# 查看服务状态
./start.sh status

# 查看所有服务日志
./start.sh logs

# 查看特定服务日志
./start.sh logs swit-auth

# 清理环境（删除所有数据）
./start.sh clean

# 查看帮助
./start.sh help
```

### 使用 docker-compose

```bash
# 启动服务
docker-compose up -d

# 停止服务
docker-compose down

# 重启特定服务
docker-compose restart swit-auth

# 查看日志
docker-compose logs -f swit-serve

# 查看服务状态
docker-compose ps

# 重新构建并启动
docker-compose up -d --build

# 停止并删除所有容器、网络、数据卷
docker-compose down -v
```

## 📊 数据库

### 数据库信息

- **主机**: localhost
- **端口**: 3306
- **用户名**: root
- **密码**: root

### 数据库列表

1. **auth_service_db** - 认证服务数据库
   - tokens 表：存储访问令牌和刷新令牌

2. **user_service_db** - 用户服务数据库
   - users 表：存储用户信息

### 数据库初始化

数据库会在 MySQL 容器首次启动时自动初始化，SQL 脚本位于：
- `../../scripts/sql/auth_service_db.sql`
- `../../scripts/sql/user_service_db.sql`

## 🔧 配置说明

### 环境变量

所有服务都可以通过环境变量进行配置：

#### MySQL
- `MYSQL_ROOT_PASSWORD`: root 用户密码

#### Swit-Auth
- `DATABASE_HOST`: 数据库主机
- `DATABASE_PORT`: 数据库端口
- `DATABASE_USERNAME`: 数据库用户名
- `DATABASE_PASSWORD`: 数据库密码
- `DATABASE_DBNAME`: 数据库名称
- `SERVER_PORT`: 服务端口
- `SERVICE_DISCOVERY_ADDRESS`: Consul 地址

#### Swit-Serve
- `DATABASE_HOST`: 数据库主机
- `DATABASE_PORT`: 数据库端口
- `DATABASE_USERNAME`: 数据库用户名
- `DATABASE_PASSWORD`: 数据库密码
- `DATABASE_DBNAME`: 数据库名称
- `SERVER_PORT`: 服务端口
- `SERVICE_DISCOVERY_ADDRESS`: Consul 地址
- `AUTH_SERVICE_URL`: 认证服务地址

### 配置文件

- `swit.yaml`: 主服务配置
- `switauth.yaml`: 认证服务配置

## 🐛 故障排除

### 常见问题

1. **端口冲突**
   ```bash
   # 检查端口占用
   lsof -i :3306
   lsof -i :8500
   lsof -i :9000
   lsof -i :9001
   ```

2. **服务启动失败**
   ```bash
   # 查看特定服务日志
   docker-compose logs swit-auth
   
   # 查看所有服务状态
   docker-compose ps
   ```

3. **数据库连接问题**
   ```bash
   # 检查 MySQL 容器状态
   docker-compose logs mysql
   
   # 手动连接数据库测试
   docker-compose exec mysql mysql -u root -proot
   ```

4. **服务发现问题**
   ```bash
   # 检查 Consul 状态
   docker-compose logs consul
   
   # 访问 Consul UI
   open http://localhost:8500
   ```

### 重置环境

如果遇到无法解决的问题，可以完全重置环境：

```bash
# 使用启动脚本清理
./start.sh clean

# 或手动清理
docker-compose down -v --remove-orphans
docker system prune -f
```

## 📝 开发说明

### 代码修改

当您修改代码后，需要重新构建对应的服务：

```bash
# 重新构建特定服务
docker-compose build swit-auth
docker-compose up -d swit-auth

# 重新构建所有服务
docker-compose up -d --build
```

### 调试模式

可以通过修改 docker-compose.yml 文件来启用调试模式或挂载本地代码目录。

### 性能监控

- 通过 Consul UI (http://localhost:8500) 监控服务状态
- 使用 `docker stats` 查看容器资源使用情况
- 通过健康检查端点监控服务健康状态

## 🤝 贡献

如果您发现问题或有改进建议，请提交 Issue 或 Pull Request。

## 📄 许可证

请参考项目根目录的 LICENSE 文件。 