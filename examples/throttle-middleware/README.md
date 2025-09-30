# Throttle Middleware Example

这个示例演示了如何使用 SWIT 框架的速率限制和节流中间件。

## 功能特性

- **Token Bucket 算法**: 允许突发流量，适合 API 速率限制
- **Leaky Bucket 算法**: 平滑流量，适合恒定速率处理
- **灵活配置**: 支持自定义容量、速率、键提取函数等
- **多维度限流**: 支持基于 IP、用户 ID 或其他维度的限流
- **自动清理**: 自动清理长期未使用的限流器以防止内存泄漏

## 运行示例

```bash
# 从项目根目录运行
go run examples/throttle-middleware/main.go
```

服务器将在 http://localhost:8080 启动。

## 测试端点

### 1. 默认 Token Bucket 配置

```bash
# 快速发送多个请求测试突发流量
for i in {1..250}; do curl http://localhost:8080/api/v1/default; echo; done
```

默认配置允许 200 个突发请求，然后以 100 req/s 的速率补充。

### 2. 自定义 Token Bucket (10 容量, 5 req/s)

```bash
# 发送 15 个请求，前 10 个会成功，之后被限流
for i in {1..15}; do 
  curl http://localhost:8080/api/v1/token-bucket
  echo
  sleep 0.1
done
```

### 3. Leaky Bucket (20 容量, 10 req/s)

```bash
# 发送请求测试平滑流量处理
for i in {1..25}; do 
  curl http://localhost:8080/api/v1/leaky-bucket
  echo
done
```

### 4. 基于用户 ID 的限流

```bash
# 不同用户独立计数
curl -H 'X-User-ID: user123' http://localhost:8080/api/v1/user-based
curl -H 'X-User-ID: user456' http://localhost:8080/api/v1/user-based

# 或使用 query 参数
curl 'http://localhost:8080/api/v1/user-based?user_id=user789'
```

### 5. 严格限流 (1 req/s)

```bash
# 快速发送 5 个请求，大部分会被限流
for i in {1..5}; do 
  curl -X POST http://localhost:8080/api/v1/sensitive
  echo
done

# 带延迟的请求会成功
for i in {1..5}; do 
  curl -X POST http://localhost:8080/api/v1/sensitive
  echo
  sleep 1
done
```

### 6. 路由组限流

```bash
# 同一路由组共享限流配置
curl http://localhost:8080/api/v2/users
curl http://localhost:8080/api/v2/posts
```

## 配置说明

### Token Bucket vs Leaky Bucket

**Token Bucket（令牌桶）**:
- 允许突发流量（burst traffic）
- 适合需要偶尔超过平均速率的场景
- 示例: API 速率限制，允许短时间内多次请求

**Leaky Bucket（漏桶）**:
- 平滑流量，恒定速率处理
- 适合需要严格控制流量的场景
- 示例: 消息队列处理，避免下游服务过载

### 配置参数

```go
type ThrottleConfig struct {
    Strategy        ThrottleStrategy  // token_bucket 或 leaky_bucket
    Capacity        int64            // 桶容量
    Rate            float64          // 速率（每秒请求数）
    KeyFunc         func(*gin.Context) string  // 提取客户端标识
    CleanupInterval time.Duration    // 清理间隔
    MaxIdleTime     time.Duration    // 最大空闲时间
    ErrorHandler    func(*gin.Context)  // 自定义错误处理
}
```

## 使用场景

1. **公共 API 保护**: 防止滥用和 DDoS 攻击
2. **用户级限流**: 不同用户不同配额
3. **敏感操作保护**: 登录、支付等操作的严格限流
4. **资源保护**: 防止下游服务过载
5. **公平性保证**: 确保所有用户公平访问资源

## 性能考虑

- Token Bucket 和 Leaky Bucket 都使用高效的时间计算，无需定时器
- 使用 RWMutex 优化并发访问性能
- 自动清理机制防止内存泄漏
- 基准测试显示单次检查延迟 < 1μs

## 最佳实践

1. **选择合适的策略**: 根据业务需求选择 Token Bucket 或 Leaky Bucket
2. **合理设置容量**: 容量应考虑正常突发流量需求
3. **监控限流指标**: 记录被限流的请求以优化配置
4. **提供清晰的错误信息**: 告诉用户限流原因和重试时间
5. **分层限流**: 全局、用户、IP 等多层次限流策略

## 扩展

要添加到现有 SWIT 服务中：

```go
import "github.com/innovationmech/swit/pkg/middleware"

// 创建中间件
throttle, err := middleware.NewTokenBucketMiddleware(100, 50.0)
if err != nil {
    log.Fatal(err)
}
defer throttle.Stop()

// 应用到路由
router.Use(throttle.Handler())
```

## 相关文档

- [pkg/middleware/throttle.go](../../pkg/middleware/throttle.go) - 中间件实现
- [pkg/middleware/token_bucket.go](../../pkg/middleware/token_bucket.go) - Token Bucket 算法
- [pkg/middleware/leaky_bucket.go](../../pkg/middleware/leaky_bucket.go) - Leaky Bucket 算法
- [pkg/middleware/*_test.go](../../pkg/middleware/) - 测试用例
