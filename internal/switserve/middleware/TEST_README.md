# Middleware 测试文档

## 测试覆盖范围

本目录包含了 switserve 中间件的完整测试套件，覆盖了以下组件：

### 1. GlobalMiddlewareRegistrar 测试 (`registrar_test.go`)

#### 基础功能测试
- ✅ `TestNewGlobalMiddlewareRegistrar` - 测试构造函数
- ✅ `TestGlobalMiddlewareRegistrar_GetName` - 测试名称获取
- ✅ `TestGlobalMiddlewareRegistrar_GetPriority` - 测试优先级获取

#### 中间件注册测试
- ✅ `TestGlobalMiddlewareRegistrar_RegisterMiddleware` - 测试中间件注册功能
  - 成功注册测试
  - 返回值验证
- ✅ `TestGlobalMiddlewareRegistrar_RegisterMiddleware_NilRouter` - 测试空路由器处理
- ✅ `TestGlobalMiddlewareRegistrar_MiddlewareOrder` - 测试中间件执行顺序
- ✅ `TestGlobalMiddlewareRegistrar_MultipleRegistrations` - 测试多次注册

#### 配置验证测试
- ✅ `TestGlobalMiddlewareRegistrar_ConfigurationValidation` - 测试配置验证
- ✅ `TestGlobalMiddlewareRegistrar_TimeoutConfiguration` - 测试超时配置
- ✅ `TestGlobalMiddlewareRegistrar_Interface` - 测试接口实现
- ✅ `TestGlobalMiddlewareRegistrar_Consistency` - 测试实例一致性

#### 性能测试
- ✅ `BenchmarkGlobalMiddlewareRegistrar_Creation` - 创建性能基准测试

### 2. 超时中间件测试 (`timeout_test.go`)

#### 核心超时功能测试
- ✅ `TestTimeoutMiddleware` - 基本超时中间件测试
  - 请求在超时时间内完成
  - 请求超过超时时间
- ✅ `TestContextTimeoutMiddleware` - Context超时中间件测试
  - 成功传递超时上下文
  - 超时上下文触发

#### 配置和定制测试
- ✅ `TestTimeoutWithConfig` - 配置化超时中间件测试
  - 自定义超时配置
  - 路径跳过功能
  - 默认配置使用
- ✅ `TestDefaultTimeoutConfig` - 默认配置测试
- ✅ `TestTimeoutRegistrar` - 超时注册器测试
- ✅ `TestTimeoutWithCustomHandler` - 自定义处理器测试

#### 性能基准测试
- ✅ `BenchmarkTimeoutMiddleware` - 基本超时中间件性能测试
- ✅ `BenchmarkContextTimeoutMiddleware` - Context超时中间件性能测试

## 测试统计

- **总测试用例**: 24个
- **子测试用例**: 10个
- **基准测试**: 3个
- **测试覆盖率**: 63.5%

## 测试策略

### 1. 单元测试策略
- **隔离测试**: 每个测试用例都使用独立的 Gin 引擎实例
- **结构化测试**: 使用表驱动测试模式进行多场景验证
- **边界测试**: 测试空值、超时、错误条件等边界情况

### 2. 避免的依赖问题
由于 `pkg/middleware` 中的 Logger 和 CORS 中间件依赖外部包（如 zap logger），测试中采用了以下策略：
- **结构验证**: 验证中间件是否正确注册到路由器
- **接口测试**: 测试注册器接口的正确实现
- **配置测试**: 验证超时配置是否正确应用

### 3. 性能测试
- **创建性能**: 测试对象创建的性能开销
- **注册性能**: 测试中间件注册的性能开销
- **执行性能**: 测试中间件执行的性能开销

## 运行测试

### 运行所有测试
```bash
go test ./internal/switserve/middleware -v
```

### 运行特定测试
```bash
# 只运行 GlobalMiddlewareRegistrar 测试
go test ./internal/switserve/middleware -run TestGlobalMiddlewareRegistrar -v

# 只运行超时中间件测试
go test ./internal/switserve/middleware -run TestTimeout -v
```

### 检查测试覆盖率
```bash
go test ./internal/switserve/middleware -cover
```

### 运行性能基准测试
```bash
go test ./internal/switserve/middleware -bench=. -benchmem
```

## 测试最佳实践

### 1. 测试命名
- 使用描述性的测试名称
- 子测试使用蛇形命名法（snake_case）
- 基准测试以 `Benchmark` 开头

### 2. 测试结构
- 使用 `gin.SetMode(gin.TestMode)` 避免输出干扰
- 使用 `require` 进行必须通过的断言
- 使用 `assert` 进行一般性断言

### 3. 资源管理
- 每个测试用例使用独立的资源
- 避免测试之间的相互依赖
- 正确清理测试资源

## 已知限制

1. **Logger 依赖**: 由于 Logger 中间件依赖全局 logger 实例，无法进行完整的集成测试
2. **HTTP 请求测试**: 为避免 logger 问题，减少了实际 HTTP 请求的测试
3. **CORS 测试**: CORS 中间件的测试受到 Logger 中间件的影响

## 未来改进

1. **Mock Logger**: 创建 Mock Logger 以支持完整的集成测试
2. **更多边界测试**: 增加更多的边界条件和错误情况测试
3. **集成测试**: 添加完整的中间件链集成测试
4. **压力测试**: 添加高并发场景下的压力测试