# Dependency Injection Refactoring Documentation

## Overview

This document describes the dependency injection refactoring implemented in the switserve service to improve code maintainability, testability, and adherence to SOLID principles.

## Problem Analysis

### Original Issues

1. **Duplicate Dependency Creation**
   - `ServiceRegistrar` and `Controller` both created separate `UserSrv` instances
   - Resource waste and potential inconsistency

2. **Hard-coded Dependencies**
   - Direct calls to `db.GetDB()` throughout the codebase
   - Tight coupling between layers

3. **Mixed Responsibilities**
   - `ServiceRegistrar` handled both service creation and route registration
   - Violation of Single Responsibility Principle

4. **SOLID Principle Violations**
   - **SRP**: Components had multiple responsibilities
   - **DIP**: High-level modules depended on low-level implementations
   - **OCP**: Adding new dependencies required modifying existing code

## Solution: Manual Dependency Injection

### Why Manual DI (Not DI Container)?

Following Go community best practices:
- **Simplicity**: Go favors explicit over implicit
- **Compile-time Safety**: All dependencies resolved at compile time
- **Community Standard**: Most Go projects use manual DI
- **Debugging**: Clear dependency flow, easier to debug

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Server Initialization                    │
├─────────────────────────────────────────────────────────────┤
│ 1. NewDependencies() creates all dependencies               │
│ 2. NewServer() initializes server with dependencies         │
│ 3. registerServices() injects dependencies to registrars    │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                  Dependencies Manager                       │
├─────────────────────────────────────────────────────────────┤
│ • Database connections                                      │
│ • Repository instances                                      │
│ • Service instances                                         │
│ • Resource lifecycle management                             │
└─────────────────────────────────────────────────────────────┘
                                │
                ┌───────────────┼───────────────┐
                ▼               ▼               ▼
    ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
    │  ServiceRegistrar│ │   Controller    │ │    Service      │
    │                 │ │                 │ │                 │
    │ • Route setup   │ │ • HTTP handling │ │ • Business      │
    │ • Transport mgmt│ │ • Request/Resp  │ │   logic         │
    └─────────────────┘ └─────────────────┘ └─────────────────┘
```

## Implementation Details

### 1. Dependencies Manager

**File**: `internal/switserve/deps/deps.go`

```go
type Dependencies struct {
    // Infrastructure
    DB *gorm.DB
    
    // Repository layer
    UserRepo repository.UserRepository
    
    // Service layer
    UserSrv userv1.UserSrv
}

func NewDependencies() (*Dependencies, error) {
    // 1. Initialize infrastructure
    db := db.GetDB()
    
    // 2. Initialize Repository layer
    userRepo := repository.NewUserRepository(db)
    
    // 3. Initialize Service layer
    userSrv, err := userv1.NewUserSrv(userv1.WithUserRepository(userRepo))
    if err != nil {
        return nil, err
    }
    
    return &Dependencies{
        DB: db,
        UserRepo: userRepo,
        UserSrv: userSrv,
    }, nil
}
```

**Key Features**:
- Single source of truth for all dependencies
- Clear initialization order (Infrastructure → Repository → Service)
- Error handling and validation
- Resource cleanup with `Close()` method

### 2. Controller Layer Refactoring

**File**: `internal/switserve/handler/http/user/v1/user.go`

**Before**:
```go
func NewUserController() *Controller {
    userSrv, err := v1.NewUserSrv(
        v1.WithUserRepository(repository.NewUserRepository(db.GetDB())),
    )
    if err != nil {
        logger.Logger.Error("failed to create user service", zap.Error(err))
    }
    return &Controller{userSrv: userSrv}
}
```

**After**:
```go
func NewUserController(userSrv v1.UserSrv) *Controller {
    return &Controller{userSrv: userSrv}
}
```

**Benefits**:
- Dependency injection through constructor
- No more hard-coded dependency creation
- Easily testable with mock services

### 3. ServiceRegistrar Layer Refactoring

**File**: `internal/switserve/service/user/registrar.go`

**Before**:
```go
func NewServiceRegistrar() *ServiceRegistrar {
    userSrv, err := v1.NewUserSrv(...)  // Duplicate creation
    controller := v2.NewUserController()  // Also creates UserSrv
    return &ServiceRegistrar{controller: controller, userSrv: userSrv}
}
```

**After**:
```go
func NewServiceRegistrar(userSrv v1.UserSrv) *ServiceRegistrar {
    controller := v2.NewUserController(userSrv)  // Inject dependency
    return &ServiceRegistrar{
        controller: controller,
        userSrv: userSrv,
    }
}
```

**Benefits**:
- Single UserSrv instance shared between registrar and controller
- Clear dependency flow
- Separated concerns: registrar handles routing, not service creation

### 4. Server Startup Flow

**File**: `internal/switserve/server.go`

```go
func NewServer() (*Server, error) {
    // 1. Initialize dependencies first
    dependencies, err := deps.NewDependencies()
    if err != nil {
        return nil, fmt.Errorf("failed to initialize dependencies: %v", err)
    }
    
    server := &Server{
        transportManager: transport.NewManager(),
        serviceRegistry:  transport.NewServiceRegistry(),
        deps:             dependencies,
    }
    
    // 2. Register services with dependencies
    server.registerServices()
    
    return server, nil
}

func (s *Server) registerServices() {
    // Inject dependencies to registrars
    userRegistrar := user.NewServiceRegistrar(s.deps.UserSrv)
    s.serviceRegistry.Register(userRegistrar)
}
```

**Key Improvements**:
- Dependencies created once at server startup
- Clear separation between dependency creation and service registration
- Graceful shutdown with resource cleanup

## Testing Strategy

### Mock Services for Unit Testing

```go
type MockUserSrv struct {
    mock.Mock
}

func (m *MockUserSrv) CreateUser(ctx context.Context, user *model.User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}

// Test example
func TestServiceRegistrar(t *testing.T) {
    mockUserSrv := &MockUserSrv{}
    registrar := NewServiceRegistrar(mockUserSrv)
    
    // Test without database dependencies
    assert.NotNil(t, registrar)
    assert.Equal(t, mockUserSrv, registrar.userSrv)
}
```

**Benefits**:
- Easy unit testing without database
- Isolated component testing
- Predictable test behavior

## Benefits Achieved

### 1. Improved Code Quality

- **✅ Single Responsibility**: Each component has one clear purpose
- **✅ Dependency Inversion**: High-level modules don't depend on low-level details
- **✅ Open/Closed**: Easy to extend without modifying existing code

### 2. Enhanced Maintainability

- **Clear Dependencies**: All dependencies visible in one place
- **Reduced Coupling**: Components loosely coupled through interfaces
- **Easier Debugging**: Explicit dependency flow

### 3. Better Testability

- **Unit Testing**: Components easily tested in isolation
- **Mock Injection**: Dependencies easily mocked for testing
- **Predictable Behavior**: No hidden dependencies

### 4. Improved Performance

- **Single Instances**: No duplicate service creation
- **Resource Management**: Proper lifecycle management
- **Memory Efficiency**: Shared dependencies

## Migration Guide

### For Existing Services

1. **Create Dependencies Structure**:
   ```go
   // Add new service to Dependencies struct
   type Dependencies struct {
       // ... existing fields
       NewSrv newservice.Service
   }
   ```

2. **Update NewDependencies**:
   ```go
   func NewDependencies() (*Dependencies, error) {
       // ... existing initialization
       newSrv := newservice.NewService(dependency)
       return &Dependencies{
           // ... existing fields
           NewSrv: newSrv,
       }
   }
   ```

3. **Refactor Service Constructors**:
   ```go
   func NewServiceRegistrar(newSrv newservice.Service) *ServiceRegistrar {
       // Use injected dependency
   }
   ```

4. **Update Server Registration**:
   ```go
   func (s *Server) registerServices() {
       // ... existing registrations
       newRegistrar := newservice.NewServiceRegistrar(s.deps.NewSrv)
       s.serviceRegistry.Register(newRegistrar)
   }
   ```

### Backward Compatibility

Legacy methods are maintained with `Deprecated` markers:
- `NewUserControllerLegacy()` - maintains old behavior
- `NewServiceRegistrarLegacy()` - maintains old behavior

These should be removed in future versions after migration is complete.

## Future Considerations

### Scaling to Complex Dependencies

For projects with complex dependency graphs, consider:

1. **Google Wire**: Compile-time dependency injection
2. **Factory Pattern**: Group related dependencies
3. **Builder Pattern**: Complex object construction

### When to Consider DI Containers

Consider DI containers when:
- Service count > 20
- Complex circular dependencies
- Dynamic service discovery needs
- Team prefers framework-based approach

## Conclusion

This refactoring successfully addresses the identified architectural issues while maintaining Go's simplicity principles. The solution provides:

- **Clear Architecture**: Explicit dependency management
- **Easy Testing**: Injectable dependencies
- **Good Performance**: No duplicate instances
- **Future-Proof**: Extensible design

The manual dependency injection approach aligns with Go community best practices and provides a solid foundation for future development.