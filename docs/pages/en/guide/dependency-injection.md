# Dependency Injection

This guide covers the dependency injection system in Swit, which provides factory-based dependency management with lifecycle support, singleton and transient patterns, and thread-safe operations.

## Overview

The Swit dependency injection system is designed around the `BusinessDependencyContainer` interface, providing flexible dependency creation, lifecycle management, and resource cleanup. It supports both singleton and transient dependency patterns with factory-based creation.

## Core Interfaces

### BusinessDependencyContainer

The base interface for dependency access:

```go
type BusinessDependencyContainer interface {
    Close() error
    GetService(name string) (interface{}, error)
}
```

### BusinessDependencyRegistry

Extended interface for registration and lifecycle management:

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

Factory function for creating dependencies:

```go
type DependencyFactory func(container BusinessDependencyContainer) (interface{}, error)
```

## Basic Usage

### Creating a Dependency Container

```go
import "github.com/innovationmech/swit/pkg/server"

// Create a new dependency container
container := server.NewBusinessDependencyContainerBuilder().Build()

// Or create with dependencies
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

### Registering Dependencies

```go
// Register singleton dependency (created once, reused)
err := container.RegisterSingleton("database", func(c server.BusinessDependencyContainer) (interface{}, error) {
    return sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname")
})

// Register transient dependency (created each time)
err = container.RegisterTransient("user-repository", func(c server.BusinessDependencyContainer) (interface{}, error) {
    db, err := c.GetService("database")
    if err != nil {
        return nil, err
    }
    return &UserRepository{DB: db.(*sql.DB)}, nil
})

// Register existing instance
err = container.RegisterInstance("config", &Config{
    DatabaseURL: "mysql://localhost:3306/db",
    RedisURL:    "redis://localhost:6379",
})
```

### Retrieving Dependencies

```go
// Get a dependency
db, err := container.GetService("database")
if err != nil {
    return fmt.Errorf("failed to get database: %w", err)
}

// Type assert to use
database := db.(*sql.DB)
users, err := database.Query("SELECT * FROM users")
```

## Dependency Patterns

### Singleton Pattern

Singletons are created once and reused throughout the application lifecycle:

```go
// Database connection (singleton - expensive to create)
container.RegisterSingleton("database", func(c server.BusinessDependencyContainer) (interface{}, error) {
    config, err := c.GetService("config")
    if err != nil {
        return nil, err
    }
    
    cfg := config.(*Config)
    db, err := sql.Open("mysql", cfg.DatabaseURL)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(25)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    // Test connection
    if err := db.Ping(); err != nil {
        db.Close()
        return nil, fmt.Errorf("database ping failed: %w", err)
    }
    
    return db, nil
})

// Redis client (singleton)
container.RegisterSingleton("redis", func(c server.BusinessDependencyContainer) (interface{}, error) {
    config, err := c.GetService("config")
    if err != nil {
        return nil, err
    }
    
    cfg := config.(*Config)
    client := redis.NewClient(&redis.Options{
        Addr:         cfg.RedisURL,
        Password:     cfg.RedisPassword,
        DB:           0,
        PoolSize:     10,
        MinIdleConns: 5,
    })
    
    // Test connection
    _, err = client.Ping(context.Background()).Result()
    if err != nil {
        client.Close()
        return nil, fmt.Errorf("redis ping failed: %w", err)
    }
    
    return client, nil
})
```

### Transient Pattern

Transients are created fresh each time they're requested:

```go
// Repository (transient - stateful, request-scoped)
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

// Service layer (transient - request-scoped)
container.RegisterTransient("user-service", func(c server.BusinessDependencyContainer) (interface{}, error) {
    repo, err := c.GetService("user-repository")
    if err != nil {
        return nil, err
    }
    
    validator, err := c.GetService("validator")
    if err != nil {
        return nil, err
    }
    
    return &UserService{
        Repository: repo.(*UserRepository),
        Validator:  validator.(*Validator),
    }, nil
})
```

### Instance Registration

Register pre-created instances:

```go
// Configuration (instance - already created)
config := &Config{
    DatabaseURL:   os.Getenv("DATABASE_URL"),
    RedisURL:      os.Getenv("REDIS_URL"),
    JWTSecret:     os.Getenv("JWT_SECRET"),
    ServerPort:    os.Getenv("SERVER_PORT"),
}
container.RegisterInstance("config", config)

// Logger (instance - configured externally)
logger, _ := zap.NewProduction()
container.RegisterInstance("logger", logger)
```

## Advanced Patterns

### Dependency Builder Pattern

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

func (b *DependencyBuilder) WithRedis() *DependencyBuilder {
    b.container.RegisterSingleton("redis", func(c server.BusinessDependencyContainer) (interface{}, error) {
        config, err := c.GetService("config")
        if err != nil {
            return nil, err
        }
        
        cfg := config.(*Config)
        client := redis.NewClient(&redis.Options{
            Addr: cfg.RedisURL,
        })
        
        return client, nil
    })
    return b
}

func (b *DependencyBuilder) WithRepositories() *DependencyBuilder {
    b.container.RegisterTransient("user-repository", func(c server.BusinessDependencyContainer) (interface{}, error) {
        db, err := c.GetService("database")
        if err != nil {
            return nil, err
        }
        return &UserRepository{DB: db.(*sql.DB)}, nil
    })
    
    b.container.RegisterTransient("order-repository", func(c server.BusinessDependencyContainer) (interface{}, error) {
        db, err := c.GetService("database")
        if err != nil {
            return nil, err
        }
        return &OrderRepository{DB: db.(*sql.DB)}, nil
    })
    
    return b
}

func (b *DependencyBuilder) WithServices() *DependencyBuilder {
    b.container.RegisterTransient("user-service", func(c server.BusinessDependencyContainer) (interface{}, error) {
        repo, err := c.GetService("user-repository")
        if err != nil {
            return nil, err
        }
        return &UserService{Repository: repo.(*UserRepository)}, nil
    })
    
    return b
}

func (b *DependencyBuilder) Build() server.BusinessDependencyRegistry {
    return b.container
}

// Usage
container := NewDependencyBuilder().
    WithConfig(config).
    WithDatabase().
    WithRedis().
    WithRepositories().
    WithServices().
    Build()
```

### Factory with Configuration

```go
type DatabaseFactory struct {
    config *DatabaseConfig
}

func NewDatabaseFactory(config *DatabaseConfig) *DatabaseFactory {
    return &DatabaseFactory{config: config}
}

func (f *DatabaseFactory) Create(c server.BusinessDependencyContainer) (interface{}, error) {
    db, err := sql.Open(f.config.Driver, f.config.ConnectionString)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(f.config.MaxOpenConns)
    db.SetMaxIdleConns(f.config.MaxIdleConns)
    db.SetConnMaxLifetime(f.config.MaxLifetime)
    
    // Test connection
    ctx, cancel := context.WithTimeout(context.Background(), f.config.Timeout)
    defer cancel()
    
    if err := db.PingContext(ctx); err != nil {
        db.Close()
        return nil, fmt.Errorf("database ping failed: %w", err)
    }
    
    return db, nil
}

// Register factory
dbConfig := &DatabaseConfig{
    Driver:           "mysql",
    ConnectionString: "user:pass@tcp(localhost:3306)/db",
    MaxOpenConns:     25,
    MaxIdleConns:     25,
    MaxLifetime:      5 * time.Minute,
    Timeout:          30 * time.Second,
}

factory := NewDatabaseFactory(dbConfig)
container.RegisterSingleton("database", factory.Create)
```

### Conditional Dependencies

```go
// Register different implementations based on environment
environment := os.Getenv("ENVIRONMENT")

if environment == "production" {
    // Production database
    container.RegisterSingleton("database", func(c server.BusinessDependencyContainer) (interface{}, error) {
        return sql.Open("mysql", productionConnectionString)
    })
    
    // Production cache
    container.RegisterSingleton("cache", func(c server.BusinessDependencyContainer) (interface{}, error) {
        return redis.NewClient(&redis.Options{
            Addr:     "redis-cluster.prod.com:6379",
            Password: "prod-password",
        }), nil
    })
} else {
    // Development/test database
    container.RegisterSingleton("database", func(c server.BusinessDependencyContainer) (interface{}, error) {
        return sql.Open("sqlite3", ":memory:")
    })
    
    // In-memory cache
    container.RegisterSingleton("cache", func(c server.BusinessDependencyContainer) (interface{}, error) {
        return NewInMemoryCache(), nil
    })
}
```

## Lifecycle Management

### Container Initialization

```go
// Initialize all dependencies
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := container.Initialize(ctx); err != nil {
    log.Fatalf("Failed to initialize dependencies: %v", err)
}

// Check if initialized
if !container.IsInitialized() {
    log.Fatal("Container not properly initialized")
}
```

### Graceful Cleanup

```go
// Proper cleanup with defer
defer func() {
    if err := container.Close(); err != nil {
        log.Printf("Error closing dependencies: %v", err)
    }
}()

// Check if closed
if container.IsClosed() {
    log.Println("Container has been closed")
}
```

### Custom Cleanup Logic

```go
// Register dependency with cleanup logic
container.RegisterSingleton("database", func(c server.BusinessDependencyContainer) (interface{}, error) {
    db, err := sql.Open("mysql", connectionString)
    if err != nil {
        return nil, err
    }
    
    // Wrap with cleanup logic
    return &DatabaseWrapper{
        DB: db,
        onClose: func() error {
            log.Println("Closing database connections...")
            return db.Close()
        },
    }, nil
})

type DatabaseWrapper struct {
    *sql.DB
    onClose func() error
}

func (w *DatabaseWrapper) Close() error {
    if w.onClose != nil {
        return w.onClose()
    }
    return w.DB.Close()
}
```

## Integration with Server Framework

### Using Dependencies in Services

```go
type UserService struct {
    repository *UserRepository
    cache      *redis.Client
    logger     *zap.Logger
}

func NewUserService(container server.BusinessDependencyContainer) (*UserService, error) {
    repo, err := container.GetService("user-repository")
    if err != nil {
        return nil, fmt.Errorf("failed to get user repository: %w", err)
    }
    
    cache, err := container.GetService("redis")
    if err != nil {
        return nil, fmt.Errorf("failed to get redis client: %w", err)
    }
    
    logger, err := container.GetService("logger")
    if err != nil {
        return nil, fmt.Errorf("failed to get logger: %w", err)
    }
    
    return &UserService{
        repository: repo.(*UserRepository),
        cache:      cache.(*redis.Client),
        logger:     logger.(*zap.Logger),
    }, nil
}

func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    // Try cache first
    cached, err := s.cache.Get(ctx, fmt.Sprintf("user:%s", id)).Result()
    if err == nil {
        var user User
        if err := json.Unmarshal([]byte(cached), &user); err == nil {
            return &user, nil
        }
    }
    
    // Get from repository
    user, err := s.repository.GetByID(ctx, id)
    if err != nil {
        s.logger.Error("Failed to get user from repository", 
            zap.String("userID", id),
            zap.Error(err))
        return nil, err
    }
    
    // Cache the result
    if userData, err := json.Marshal(user); err == nil {
        s.cache.Set(ctx, fmt.Sprintf("user:%s", id), userData, time.Hour)
    }
    
    return user, nil
}
```

### Service Registrar with Dependencies

```go
type ServiceRegistrar struct {
    dependencies server.BusinessDependencyContainer
}

func NewServiceRegistrar(deps server.BusinessDependencyContainer) *ServiceRegistrar {
    return &ServiceRegistrar{dependencies: deps}
}

func (r *ServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Create user service with dependencies
    userService, err := NewUserService(r.dependencies)
    if err != nil {
        return fmt.Errorf("failed to create user service: %w", err)
    }
    
    // Register HTTP handler
    if err := registry.RegisterBusinessHTTPHandler(userService); err != nil {
        return fmt.Errorf("failed to register user HTTP handler: %w", err)
    }
    
    // Create auth service with dependencies
    authService, err := NewAuthService(r.dependencies)
    if err != nil {
        return fmt.Errorf("failed to create auth service: %w", err)
    }
    
    // Register gRPC service
    if err := registry.RegisterBusinessGRPCService(authService); err != nil {
        return fmt.Errorf("failed to register auth gRPC service: %w", err)
    }
    
    return nil
}
```

## Testing with Dependency Injection

### Mock Dependencies for Testing

```go
func TestUserService(t *testing.T) {
    // Create test container
    container := server.NewSimpleBusinessDependencyContainer()
    
    // Register mock dependencies
    mockDB := &MockDatabase{}
    mockCache := &MockRedisClient{}
    mockLogger := zap.NewNop()
    
    container.RegisterInstance("database", mockDB)
    container.RegisterInstance("redis", mockCache)
    container.RegisterInstance("logger", mockLogger)
    
    // Register test repository
    container.RegisterTransient("user-repository", func(c server.BusinessDependencyContainer) (interface{}, error) {
        db, _ := c.GetService("database")
        return &UserRepository{
            DB: db.(*MockDatabase),
        }, nil
    })
    
    // Create service with test dependencies
    userService, err := NewUserService(container)
    require.NoError(t, err)
    
    // Test service methods
    ctx := context.Background()
    user, err := userService.GetUser(ctx, "test-id")
    assert.NoError(t, err)
    assert.NotNil(t, user)
    
    // Verify mock interactions
    assert.True(t, mockDB.GetByIDCalled)
    assert.Equal(t, "test-id", mockDB.LastRequestedID)
}

type MockDatabase struct {
    GetByIDCalled    bool
    LastRequestedID  string
}

func (m *MockDatabase) GetByID(ctx context.Context, id string) (*User, error) {
    m.GetByIDCalled = true
    m.LastRequestedID = id
    return &User{ID: id, Name: "Test User"}, nil
}
```

### Test Container Builder

```go
func NewTestContainer() server.BusinessDependencyRegistry {
    container := server.NewSimpleBusinessDependencyContainer()
    
    // Test configuration
    config := &Config{
        DatabaseURL: "sqlite3::memory:",
        RedisURL:    "redis://localhost:6379",
        Environment: "test",
    }
    container.RegisterInstance("config", config)
    
    // Test logger
    logger := zap.NewNop()
    container.RegisterInstance("logger", logger)
    
    // Mock database
    container.RegisterSingleton("database", func(c server.BusinessDependencyContainer) (interface{}, error) {
        return &MockDatabase{}, nil
    })
    
    // Mock cache
    container.RegisterSingleton("redis", func(c server.BusinessDependencyContainer) (interface{}, error) {
        return &MockRedisClient{}, nil
    })
    
    return container
}
```

## Best Practices

### Dependency Organization

1. **Layered Dependencies** - Organize dependencies by layer (config, infrastructure, repositories, services)
2. **Interface Dependencies** - Depend on interfaces rather than concrete types
3. **Lifecycle Awareness** - Choose singleton vs transient based on lifecycle requirements
4. **Resource Management** - Always implement proper cleanup for resources
5. **Error Handling** - Handle dependency creation errors gracefully

### Factory Design

1. **Validation** - Validate dependencies and configuration in factories
2. **Error Context** - Provide meaningful error messages with context
3. **Resource Testing** - Test resource connectivity during creation
4. **Configuration Driven** - Make factories configurable through dependency injection
5. **Cleanup Registration** - Register cleanup functions for complex resources

### Testing Strategy

1. **Mock Everything** - Create mock implementations for all external dependencies
2. **Test Isolation** - Each test should have isolated dependency containers
3. **Factory Testing** - Test dependency factories separately
4. **Integration Testing** - Test with real dependencies in integration tests
5. **Cleanup Testing** - Verify proper resource cleanup in tests

### Performance Considerations

1. **Singleton for Expensive** - Use singletons for expensive-to-create resources
2. **Transient for Stateful** - Use transients for stateful or request-scoped objects
3. **Lazy Loading** - Consider lazy loading for optional dependencies
4. **Connection Pooling** - Configure appropriate connection pools for database/cache
5. **Resource Limits** - Set appropriate limits on pooled resources

This dependency injection guide covers all aspects of the DI system in Swit, from basic usage to advanced patterns, testing strategies, and best practices for production systems.