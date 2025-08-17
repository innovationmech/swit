# 示例和教程

Swit 框架提供了丰富的示例代码，帮助您理解不同使用场景和最佳实践。从简单的 HTTP 服务到完整的微服务架构，这些示例涵盖了框架的各个方面。

## 入门示例

### 🚀 简单 HTTP 服务
最基础的框架使用示例，演示如何创建一个 HTTP 微服务。

- **路径**: `examples/simple-http-service/`
- **特性**: RESTful API、健康检查、优雅关闭
- **适用于**: 框架入门、HTTP 单一服务

```bash
cd examples/simple-http-service
go run main.go
```

**API 端点:**
- `GET /api/v1/hello?name=<name>` - 问候接口
- `GET /api/v1/status` - 服务状态
- `POST /api/v1/echo` - 回声接口
- `GET /health` - 健康检查

### 📡 gRPC 服务示例
展示如何使用框架创建 gRPC 服务，包括 Protocol Buffers 集成。

- **路径**: `examples/grpc-service/`
- **特性**: gRPC 服务器、流式传输支持、Protocol Buffer 定义
- **适用于**: gRPC 专用服务、服务间通信

```bash
cd examples/grpc-service
go run main.go
```

### 🏆 全功能服务示例
完整展示框架所有功能的综合示例。

- **路径**: `examples/full-featured-service/`
- **特性**: HTTP + gRPC、依赖注入、服务发现、中间件
- **适用于**: 生产环境模式、框架评估

```bash
cd examples/full-featured-service
go run main.go
```

## 参考实现

### 👥 用户管理服务 (switserve)
完整的用户管理微服务，展示框架在实际项目中的应用。

- **路径**: `internal/switserve/`
- **端口**: HTTP: 9000, gRPC: 10000
- **功能**:
  - 用户 CRUD 操作（HTTP REST + gRPC）
  - 问候服务（支持流式传输）
  - 通知系统集成
  - 数据库集成（GORM）
  - 中间件堆栈演示

```bash
# 构建和运行
make build
./bin/swit-serve

# 或者直接运行
cd internal/switserve
go run main.go
```

**API 示例:**
```bash
# 健康检查
curl http://localhost:9000/health

# 用户操作
curl http://localhost:9000/api/v1/users
curl -X POST http://localhost:9000/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "张三", "email": "zhangsan@example.com"}'
```

### 🔐 身份验证服务 (switauth)
JWT 身份验证微服务，展示安全模式和令牌管理。

- **路径**: `internal/switauth/`
- **端口**: HTTP: 9001, gRPC: 50051
- **功能**:
  - 用户登录/登出（HTTP + gRPC）
  - JWT 令牌生成和验证
  - 令牌刷新和撤销
  - 密码重置工作流
  - Redis 会话管理

```bash
# 构建和运行
make build
./bin/swit-auth

# 或者直接运行
cd internal/switauth
go run main.go
```

**API 示例:**
```bash
# 用户登录
curl -X POST http://localhost:9001/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password123"}'

# 令牌验证
curl -X POST http://localhost:9001/api/v1/auth/verify \
  -H "Authorization: Bearer <token>"
```

### 🛠️ 命令行工具 (switctl)
框架集成的命令行管理工具。

- **路径**: `internal/switctl/`
- **功能**:
  - 健康检查命令
  - 服务管理操作
  - 版本信息和诊断

```bash
# 构建和运行
make build
./bin/switctl --help

# 健康检查
./bin/switctl health check

# 版本信息
./bin/switctl version
```

## 使用模式演示

### 1. 服务注册模式

#### HTTP 服务注册
```go
type MyHTTPHandler struct{}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    
    // 注册路由
    v1 := ginRouter.Group("/api/v1")
    v1.GET("/users", h.listUsers)
    v1.POST("/users", h.createUser)
    v1.GET("/users/:id", h.getUser)
    
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return "user-service"
}
```

#### gRPC 服务注册
```go
type MyGRPCService struct{
    // 服务实现
}

func (s *MyGRPCService) RegisterGRPC(server interface{}) error {
    grpcServer := server.(*grpc.Server)
    userpb.RegisterUserServiceServer(grpcServer, s)
    return nil
}

func (s *MyGRPCService) GetServiceName() string {
    return "user-grpc-service"
}
```

### 2. 配置管理模式

#### 环境配置
```go
type ServiceConfig struct {
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
    JWT      JWTConfig      `mapstructure:"jwt"`
}

func (c *ServiceConfig) Validate() error {
    if c.Database.Host == "" {
        return errors.New("database host is required")
    }
    return nil
}

func (c *ServiceConfig) SetDefaults() {
    if c.Database.Port == 0 {
        c.Database.Port = 3306
    }
}
```

#### YAML 配置文件
```yaml
service_name: "my-service"
http:
  enabled: true
  port: "8080"
grpc:
  enabled: true
  port: "9080"
database:
  host: "localhost"
  port: 3306
  name: "myapp"
redis:
  address: "localhost:6379"
```

### 3. 依赖注入模式

#### 依赖注册
```go
type MyDependencyContainer struct {
    db    *gorm.DB
    redis *redis.Client
}

func (c *MyDependencyContainer) Initialize(ctx context.Context) error {
    // 初始化数据库连接
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        return err
    }
    c.db = db
    
    // 初始化 Redis 客户端
    c.redis = redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    return nil
}

func (c *MyDependencyContainer) GetService(name string) (interface{}, error) {
    switch name {
    case "database":
        return c.db, nil
    case "redis":
        return c.redis, nil
    default:
        return nil, fmt.Errorf("service %s not found", name)
    }
}
```

### 4. 健康检查模式

#### 数据库健康检查
```go
type DatabaseHealthCheck struct {
    db *gorm.DB
}

func (h *DatabaseHealthCheck) Check(ctx context.Context) error {
    var result int
    return h.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error
}

func (h *DatabaseHealthCheck) GetServiceName() string {
    return "database"
}
```

#### Redis 健康检查
```go
type RedisHealthCheck struct {
    redis *redis.Client
}

func (h *RedisHealthCheck) Check(ctx context.Context) error {
    return h.redis.Ping(ctx).Err()
}

func (h *RedisHealthCheck) GetServiceName() string {
    return "redis"
}
```

### 5. 中间件集成模式

#### 身份验证中间件
```go
func JWTAuthMiddleware(secret string) gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        tokenString := c.GetHeader("Authorization")
        if tokenString == "" {
            c.JSON(401, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }
        
        // 验证 JWT 令牌
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return []byte(secret), nil
        })
        
        if err != nil || !token.Valid {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }
        
        c.Next()
    })
}
```

#### CORS 中间件
```go
func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}
```

## 测试策略

### 单元测试示例
```go
func TestUserService_CreateUser(t *testing.T) {
    // 设置测试
    service := &UserService{
        db: setupTestDB(),
    }
    
    // 测试用例
    user := &User{
        Name:  "张三",
        Email: "zhangsan@example.com",
    }
    
    // 执行
    result, err := service.CreateUser(context.Background(), user)
    
    // 断言
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.NotZero(t, result.ID)
}
```

### 集成测试示例
```go
func TestHTTPEndpoints(t *testing.T) {
    // 启动测试服务器
    config := &server.ServerConfig{
        HTTP: server.HTTPConfig{
            Port:     "0", // 随机端口
            Enabled:  true,
            TestMode: true,
        },
    }
    
    service := &MyService{}
    srv, _ := server.NewBusinessServerCore(config, service, nil)
    
    go srv.Start(context.Background())
    defer srv.Shutdown()
    
    // 等待服务器启动
    time.Sleep(100 * time.Millisecond)
    
    // 测试 API
    addr := srv.GetHTTPAddress()
    resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
    
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

## 性能优化实践

### 数据库连接池优化
```go
func setupDatabase() *gorm.DB {
    db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    
    sqlDB, _ := db.DB()
    
    // 连接池设置
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)
    
    return db
}
```

### Redis 连接池优化
```go
func setupRedis() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:         "localhost:6379",
        PoolSize:     10,
        MinIdleConns: 5,
        PoolTimeout:  30 * time.Second,
    })
}
```

## 部署示例

### Docker 部署
```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o myservice .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/myservice .
COPY config.yaml .

CMD ["./myservice"]
```

### Kubernetes 部署
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-service
  template:
    metadata:
      labels:
        app: my-service
    spec:
      containers:
      - name: my-service
        image: my-service:latest
        ports:
        - containerPort: 8080
        env:
        - name: HTTP_PORT
          value: "8080"
        - name: DATABASE_HOST
          value: "mysql-service"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
```

## 下一步

选择适合您需求的示例开始探索：

- 🚀 [快速开始](/zh/guide/getting-started) - 创建您的第一个服务
- 📖 [深入指南](/zh/guide/) - 学习框架的各个方面
- 🔗 [API 参考](/zh/api/) - 查看详细的 API 文档
- 🤝 [社区贡献](/zh/community/) - 参与项目开发