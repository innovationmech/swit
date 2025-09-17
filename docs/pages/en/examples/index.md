# Examples and Tutorials

The Swit framework provides rich example code to help you understand different use cases and best practices. From simple HTTP services to complete microservice architectures, these examples cover all aspects of the framework.

## Getting Started Examples

### 🚀 Simple HTTP Service
The most basic framework usage example, demonstrating how to create an HTTP microservice.

- **Path**: `examples/simple-http-service/`
- **Features**: RESTful API, health checks, graceful shutdown
- **Best For**: Framework introduction, HTTP-only services

```bash
cd examples/simple-http-service
go run main.go
```

**API Endpoints:**
- `GET /api/v1/hello?name=<name>` - Greeting endpoint
- `GET /api/v1/status` - Service status
- `POST /api/v1/echo` - Echo endpoint
- `GET /health` - Health check

### 📡 gRPC Service Example
Shows how to create gRPC services with the framework, including Protocol Buffers integration.

- **Path**: `examples/grpc-service/`
- **Features**: gRPC server, streaming support, Protocol Buffer definitions
- **Best For**: gRPC-focused services, inter-service communication

```bash
cd examples/grpc-service
go run main.go
```

### 🔄 Adapter Switching Example
Demonstrates config-driven broker selection and how to run `PlanBrokerSwitch` before migrating adapters.

- **Path**: `examples/messaging/adapter-switch/`
- **Features**: Multiple broker configs, migration planning, environment overrides
- **Best For**: Cross-broker migrations, rollout rehearsals

```bash
cd examples/messaging/adapter-switch
go run .
```

### 🏆 Full-Featured Service Example
Complete showcase of all framework features.

- **Path**: `examples/full-featured-service/`
- **Features**: HTTP + gRPC, dependency injection, service discovery, middleware
- **Best For**: Production-ready patterns, framework evaluation

```bash
cd examples/full-featured-service
go run main.go
```

### 📊 Sentry Monitoring Example
Demonstrates comprehensive error monitoring and performance tracking with Sentry integration.

- **Path**: `examples/sentry-example-service/`
- **Features**: Error capture, performance monitoring, custom context, panic recovery
- **Best For**: Production monitoring, error tracking, performance analysis

```bash
cd examples/sentry-example-service
export SENTRY_DSN="your-sentry-dsn"
go run main.go
```

**Test Endpoints:**
- `GET /api/v1/error/500` - Generate server error
- `GET /api/v1/slow` - Performance monitoring test
- `GET /api/v1/panic` - Panic recovery test
- `POST /api/v1/error/custom` - Custom error with context

[→ Detailed Sentry Example Guide](/en/examples/sentry-service)

## Reference Implementations

### 👥 User Management Service (switserve)
Complete user management microservice showcasing real-world framework application.

- **Path**: `internal/switserve/`
- **Ports**: HTTP: 9000, gRPC: 10000
- **Features**:
  - User CRUD operations (HTTP REST + gRPC)
  - Greeter service with streaming support
  - Notification system integration
  - Database integration (GORM)
  - Middleware stack demonstration

```bash
# Build and run
make build
./bin/swit-serve

# Or run directly
cd internal/switserve
go run main.go
```

**API Examples:**
```bash
# Health check
curl http://localhost:9000/health

# User operations
curl http://localhost:9000/api/v1/users
curl -X POST http://localhost:9000/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "john@example.com"}'
```

### 🔐 Authentication Service (switauth)
JWT authentication microservice showcasing security patterns and token management.

- **Path**: `internal/switauth/`
- **Ports**: HTTP: 9001, gRPC: 50051
- **Features**:
  - User login/logout (HTTP + gRPC)
  - JWT token generation and validation
  - Token refresh and revocation
  - Password reset workflows
  - Redis session management

```bash
# Build and run
make build
./bin/swit-auth

# Or run directly
cd internal/switauth
go run main.go
```

**API Examples:**
```bash
# User login
curl -X POST http://localhost:9001/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password123"}'

# Token verification
curl -X POST http://localhost:9001/api/v1/auth/verify \
  -H "Authorization: Bearer <token>"
```

### 🛠️ Command-Line Tool (switctl)
Framework-integrated command-line administration tool.

- **Path**: `internal/switctl/`
- **Features**:
  - Health check commands
  - Service management operations
  - Version information and diagnostics

```bash
# Build and run
make build
./bin/switctl --help

# Health check
./bin/switctl health check

# Version information
./bin/switctl version
```

## Usage Pattern Demonstrations

### 1. Service Registration Patterns

#### HTTP Service Registration
```go
type MyHTTPHandler struct{}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    
    // Register routes
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

#### gRPC Service Registration
```go
type MyGRPCService struct{
    // Service implementation
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

### 2. Configuration Management Patterns

#### Environment Configuration
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

#### YAML Configuration File
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

### 3. Dependency Injection Patterns

#### Dependency Registration
```go
type MyDependencyContainer struct {
    db    *gorm.DB
    redis *redis.Client
}

func (c *MyDependencyContainer) Initialize(ctx context.Context) error {
    // Initialize database connection
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        return err
    }
    c.db = db
    
    // Initialize Redis client
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

### 4. Health Check Patterns

#### Database Health Check
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

#### Redis Health Check
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

### 5. Middleware Integration Patterns

#### Authentication Middleware
```go
func JWTAuthMiddleware(secret string) gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        tokenString := c.GetHeader("Authorization")
        if tokenString == "" {
            c.JSON(401, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }
        
        // Validate JWT token
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

#### CORS Middleware
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

## Testing Strategies

### Unit Testing Example
```go
func TestUserService_CreateUser(t *testing.T) {
    // Setup test
    service := &UserService{
        db: setupTestDB(),
    }
    
    // Test case
    user := &User{
        Name:  "John Doe",
        Email: "john@example.com",
    }
    
    // Execute
    result, err := service.CreateUser(context.Background(), user)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.NotZero(t, result.ID)
}
```

### Integration Testing Example
```go
func TestHTTPEndpoints(t *testing.T) {
    // Start test server
    config := &server.ServerConfig{
        HTTP: server.HTTPConfig{
            Port:     "0", // Random port
            Enabled:  true,
            TestMode: true,
        },
    }
    
    service := &MyService{}
    srv, _ := server.NewBusinessServerCore(config, service, nil)
    
    go srv.Start(context.Background())
    defer srv.Shutdown()
    
    // Wait for server startup
    time.Sleep(100 * time.Millisecond)
    
    // Test API
    addr := srv.GetHTTPAddress()
    resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
    
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

## Performance Optimization Practices

### Database Connection Pool Optimization
```go
func setupDatabase() *gorm.DB {
    db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    
    sqlDB, _ := db.DB()
    
    // Connection pool settings
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)
    
    return db
}
```

### Redis Connection Pool Optimization
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

## Deployment Examples

### Docker Deployment
```dockerfile
FROM golang:1.23.12-alpine AS builder

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

### Kubernetes Deployment
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

## Next Steps

Choose the example that fits your needs to start exploring:

- 🚀 [Quick Start](/en/guide/getting-started) - Create your first service
- 📖 [In-Depth Guide](/en/guide/) - Learn all aspects of the framework
- 🔗 [API Reference](/en/api/) - View detailed API documentation
- 🤝 [Community Contribution](/en/community/) - Participate in project development
