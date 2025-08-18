# 依赖注入

本指南涵盖 Swit 中的依赖注入系统，它提供基于工厂的依赖管理，具有生命周期支持、单例和瞬态模式以及线程安全操作。

## 概述

Swit 依赖注入系统围绕 `BusinessDependencyContainer` 接口设计，提供灵活的依赖创建、生命周期管理和资源清理。它支持基于工厂创建的单例和瞬态依赖模式。

## 核心接口

### BusinessDependencyContainer

依赖访问的基础接口：

```go
type BusinessDependencyContainer interface {
    Close() error
    GetService(name string) (interface{}, error)
}
```

### BusinessDependencyRegistry

用于注册和生命周期管理的扩展接口：

```go
type BusinessDependencyRegistry interface {
    BusinessDependencyContainer
    Initialize(ctx context.Context) error
    RegisterSingleton(name string, factory DependencyFactory) error
    RegisterTransient(name string, factory DependencyFactory) error
    RegisterInstance(name string, instance interface{}) error
    GetDependencyNames() []string
    IsInitialized() bool
    IsClosed() bool
}
```

### DependencyFactory

创建依赖的工厂函数：

```go
type DependencyFactory func(container BusinessDependencyContainer) (interface{}, error)
```

## 基本用法

### 创建依赖容器

```go
import "github.com/innovationmech/swit/pkg/server"

// 创建新的依赖容器
container := server.NewBusinessDependencyContainerBuilder().Build()

// 或使用依赖创建
container := server.NewBusinessDependencyContainerBuilder().
    AddSingleton("database", func(c server.BusinessDependencyContainer) (interface{}, error) {
        return sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname")
    }).
    AddSingleton("redis", func(c server.BusinessDependencyContainer) (interface{}, error) {
        return redis.NewClient(&redis.Options{
            Addr: "localhost:6379",
        }), nil
    }).
    Build()
```

### 注册依赖

```go
// 注册单例依赖（创建一次，重复使用）
err := container.RegisterSingleton("database", func(c server.BusinessDependencyContainer) (interface{}, error) {
    return sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname")
})

// 注册瞬态依赖（每次创建）
err = container.RegisterTransient("user-repository", func(c server.BusinessDependencyContainer) (interface{}, error) {
    db, err := c.GetService("database")
    if err != nil {
        return nil, err
    }
    return &UserRepository{DB: db.(*sql.DB)}, nil
})

// 注册现有实例
err = container.RegisterInstance("config", &Config{
    DatabaseURL: "mysql://localhost:3306/db",
    RedisURL:    "redis://localhost:6379",
})
```

### 检索依赖

```go
// 获取依赖
db, err := container.GetService("database")
if err != nil {
    return fmt.Errorf("获取数据库失败: %w", err)
}

// 类型断言使用
database := db.(*sql.DB)
users, err := database.Query("SELECT * FROM users")
```

## 依赖模式

### 单例模式

单例在应用程序生命周期中创建一次并重复使用：

```go
// 数据库连接（单例 - 创建成本高）
container.RegisterSingleton("database", func(c server.BusinessDependencyContainer) (interface{}, error) {
    config, err := c.GetService("config")
    if err != nil {
        return nil, err
    }
    
    cfg := config.(*Config)
    db, err := sql.Open("mysql", cfg.DatabaseURL)
    if err != nil {
        return nil, fmt.Errorf("打开数据库失败: %w", err)
    }
    
    // 配置连接池
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(25)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    // 测试连接
    if err := db.Ping(); err != nil {
        db.Close()
        return nil, fmt.Errorf("数据库 ping 失败: %w", err)
    }
    
    return db, nil
})
```

### 瞬态模式

瞬态每次请求时都会新创建：

```go
// 存储库（瞬态 - 有状态，请求范围）
container.RegisterTransient("user-repository", func(c server.BusinessDependencyContainer) (interface{}, error) {
    db, err := c.GetService("database")
    if err != nil {
        return nil, err
    }
    
    cache, err := c.GetService("redis")
    if err != nil {
        return nil, err
    }
    
    logger, err := c.GetService("logger")
    if err != nil {
        return nil, err
    }
    
    return &UserRepository{
        DB:     db.(*sql.DB),
        Cache:  cache.(*redis.Client),
        Logger: logger.(*zap.Logger),
    }, nil
})
```

### 实例注册

注册预创建的实例：

```go
// 配置（实例 - 已创建）
config := &Config{
    DatabaseURL:   os.Getenv("DATABASE_URL"),
    RedisURL:      os.Getenv("REDIS_URL"),
    JWTSecret:     os.Getenv("JWT_SECRET"),
    ServerPort:    os.Getenv("SERVER_PORT"),
}
container.RegisterInstance("config", config)

// 日志器（实例 - 外部配置）
logger, _ := zap.NewProduction()
container.RegisterInstance("logger", logger)
```

## 高级模式

### 依赖构建器模式

```go
type DependencyBuilder struct {
    container server.BusinessDependencyRegistry
}

func NewDependencyBuilder() *DependencyBuilder {
    return &DependencyBuilder{
        container: server.NewSimpleBusinessDependencyContainer(),
    }
}

func (b *DependencyBuilder) WithConfig(config *Config) *DependencyBuilder {
    b.container.RegisterInstance("config", config)
    return b
}

func (b *DependencyBuilder) WithDatabase() *DependencyBuilder {
    b.container.RegisterSingleton("database", func(c server.BusinessDependencyContainer) (interface{}, error) {
        config, err := c.GetService("config")
        if err != nil {
            return nil, err
        }
        
        cfg := config.(*Config)
        db, err := sql.Open("mysql", cfg.DatabaseURL)
        if err != nil {
            return nil, err
        }
        
        return db, nil
    })
    return b
}

func (b *DependencyBuilder) Build() server.BusinessDependencyRegistry {
    return b.container
}

// 使用
container := NewDependencyBuilder().
    WithConfig(config).
    WithDatabase().
    WithRedis().
    WithRepositories().
    WithServices().
    Build()
```

## 生命周期管理

### 容器初始化

```go
// 初始化所有依赖
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := container.Initialize(ctx); err != nil {
    log.Fatalf("初始化依赖失败: %v", err)
}

// 检查是否已初始化
if !container.IsInitialized() {
    log.Fatal("容器未正确初始化")
}
```

### 优雅清理

```go
// 使用 defer 进行适当清理
defer func() {
    if err := container.Close(); err != nil {
        log.Printf("关闭依赖时出错: %v", err)
    }
}()

// 检查是否已关闭
if container.IsClosed() {
    log.Println("容器已关闭")
}
```

## 与服务器框架集成

### 在服务中使用依赖

```go
type UserService struct {
    repository *UserRepository
    cache      *redis.Client
    logger     *zap.Logger
}

func NewUserService(container server.BusinessDependencyContainer) (*UserService, error) {
    repo, err := container.GetService("user-repository")
    if err != nil {
        return nil, fmt.Errorf("获取用户存储库失败: %w", err)
    }
    
    cache, err := container.GetService("redis")
    if err != nil {
        return nil, fmt.Errorf("获取 redis 客户端失败: %w", err)
    }
    
    logger, err := container.GetService("logger")
    if err != nil {
        return nil, fmt.Errorf("获取日志器失败: %w", err)
    }
    
    return &UserService{
        repository: repo.(*UserRepository),
        cache:      cache.(*redis.Client),
        logger:     logger.(*zap.Logger),
    }, nil
}

func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    // 首先尝试缓存
    cached, err := s.cache.Get(ctx, fmt.Sprintf("user:%s", id)).Result()
    if err == nil {
        var user User
        if err := json.Unmarshal([]byte(cached), &user); err == nil {
            return &user, nil
        }
    }
    
    // 从存储库获取
    user, err := s.repository.GetByID(ctx, id)
    if err != nil {
        s.logger.Error("从存储库获取用户失败", 
            zap.String("userID", id),
            zap.Error(err))
        return nil, err
    }
    
    // 缓存结果
    if userData, err := json.Marshal(user); err == nil {
        s.cache.Set(ctx, fmt.Sprintf("user:%s", id), userData, time.Hour)
    }
    
    return user, nil
}
```

## 依赖注入测试

### 测试的模拟依赖

```go
func TestUserService(t *testing.T) {
    // 创建测试容器
    container := server.NewSimpleBusinessDependencyContainer()
    
    // 注册模拟依赖
    mockDB := &MockDatabase{}
    mockCache := &MockRedisClient{}
    mockLogger := zap.NewNop()
    
    container.RegisterInstance("database", mockDB)
    container.RegisterInstance("redis", mockCache)
    container.RegisterInstance("logger", mockLogger)
    
    // 注册测试存储库
    container.RegisterTransient("user-repository", func(c server.BusinessDependencyContainer) (interface{}, error) {
        db, _ := c.GetService("database")
        return &UserRepository{
            DB: db.(*MockDatabase),
        }, nil
    })
    
    // 使用测试依赖创建服务
    userService, err := NewUserService(container)
    require.NoError(t, err)
    
    // 测试服务方法
    ctx := context.Background()
    user, err := userService.GetUser(ctx, "test-id")
    assert.NoError(t, err)
    assert.NotNil(t, user)
    
    // 验证模拟交互
    assert.True(t, mockDB.GetByIDCalled)
    assert.Equal(t, "test-id", mockDB.LastRequestedID)
}
```

## 最佳实践

### 依赖组织

1. **分层依赖** - 按层组织依赖（配置、基础设施、存储库、服务）
2. **接口依赖** - 依赖接口而不是具体类型
3. **生命周期感知** - 根据生命周期要求选择单例与瞬态
4. **资源管理** - 始终为资源实现适当的清理
5. **错误处理** - 优雅处理依赖创建错误

### 工厂设计

1. **验证** - 在工厂中验证依赖和配置
2. **错误上下文** - 提供带有上下文的有意义的错误消息
3. **资源测试** - 在创建期间测试资源连接性
4. **配置驱动** - 通过依赖注入使工厂可配置
5. **清理注册** - 为复杂资源注册清理函数

这个依赖注入指南涵盖了 Swit 中 DI 系统的所有方面，从基本用法到高级模式、测试策略和生产系统的最佳实践。