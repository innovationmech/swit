# Health Check System

健康检查系统提供了一个可扩展的、基于适配器模式的健康检查框架，支持健康（health）、就绪（readiness）和存活（liveness）探针。

## 功能特性

- **适配器模式**：统一的健康检查适配器，支持注册多种类型的检查器
- **就绪和存活探针**：完整的 Kubernetes 兼容的探针接口
- **并发执行**：健康检查可并发执行，提高检查效率
- **超时控制**：可配置的超时机制，防止检查阻塞
- **结果缓存**：缓存最近的检查结果，便于查询和调试
- **HTTP 端点**：提供标准的 HTTP 端点，与 Kubernetes 和负载均衡器集成

## 使用示例

### 基本使用

```go
package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/server/health"
)

func main() {
	// 创建健康检查适配器
	adapter := health.NewHealthCheckAdapter(nil) // 使用默认配置

	// 注册健康检查器
	checker := health.NewBasicHealthChecker("database", func(ctx context.Context) error {
		// 检查数据库连接
		// return db.Ping(ctx)
		return nil
	})
	adapter.RegisterHealthChecker(checker)

	// 注册就绪检查器
	readinessProbe := health.NewServiceReadinessProbe("service", 5*time.Second)
	adapter.RegisterReadinessChecker(readinessProbe)

	// 标记服务为就绪
	readinessProbe.SetReady(true)

	// 注册存活检查器
	livenessProbe := health.NewServiceLivenessProbe("service", 30*time.Second)
	adapter.RegisterLivenessChecker(livenessProbe)

	// 启动心跳
	ctx := context.Background()
	livenessProbe.StartHeartbeat(ctx, 5*time.Second)

	// 注册 HTTP 端点
	router := gin.Default()
	handler := health.NewEndpointHandler(adapter, "my-service", "v1.0.0")
	handler.RegisterRoutes(router)

	router.Run(":8080")
}
```

### 自定义配置

```go
config := &health.HealthCheckConfig{
	Timeout:       10 * time.Second,  // 单个检查的超时时间
	Interval:      60 * time.Second,  // 周期性检查的间隔
	MaxConcurrent: 5,                  // 最大并发检查数
}

adapter := health.NewHealthCheckAdapter(config)
```

### 复合健康检查器

```go
// 创建复合检查器，聚合多个依赖检查
compositeChecker := health.NewCompositeHealthChecker(
	"dependencies",
	health.NewBasicHealthChecker("redis", checkRedis),
	health.NewBasicHealthChecker("kafka", checkKafka),
	health.NewBasicHealthChecker("mysql", checkMySQL),
)

adapter.RegisterHealthChecker(compositeChecker)
```

### 依赖健康检查

```go
// 包装已有的健康检查器
dbChecker := health.NewBasicHealthChecker("db", checkDB)
depChecker := health.NewDependencyHealthChecker("database-dependency", dbChecker)

adapter.RegisterHealthChecker(depChecker)
```

## HTTP 端点

健康检查系统提供以下 HTTP 端点：

### `/health`
主健康检查端点，返回服务的整体健康状态。

**响应示例**：
```json
{
  "status": "healthy",
  "service": "my-service",
  "version": "v1.0.0",
  "timestamp": "2025-09-30T10:00:00Z",
  "uptime": "1h30m15s",
  "dependencies": {
    "database": {
      "name": "database",
      "status": "healthy",
      "duration": 5000000,
      "timestamp": "2025-09-30T10:00:00Z"
    }
  }
}
```

### `/health/ready` 或 `/ready`
就绪探针端点，检查服务是否准备好接收流量。

**响应示例**：
```json
{
  "ready": true,
  "service": "my-service",
  "timestamp": "2025-09-30T10:00:00Z",
  "checks": {
    "service": {
      "name": "service",
      "status": "healthy",
      "message": "service ready",
      "duration": 100000,
      "timestamp": "2025-09-30T10:00:00Z"
    }
  }
}
```

### `/health/live` 或 `/live`
存活探针端点，检查服务是否存活（不应重启）。

**响应示例**：
```json
{
  "alive": true,
  "service": "my-service",
  "timestamp": "2025-09-30T10:00:00Z",
  "checks": {
    "service": {
      "name": "service",
      "status": "healthy",
      "message": "service alive",
      "duration": 50000,
      "timestamp": "2025-09-30T10:00:00Z"
    }
  }
}
```

### `/health/detailed`
详细健康检查端点，返回所有检查的完整状态信息。

**响应示例**：
```json
{
  "service": "my-service",
  "version": "v1.0.0",
  "timestamp": "2025-09-30T10:00:00Z",
  "uptime": "1h30m15s",
  "health": {
    "status": "healthy",
    "dependencies": { ... }
  },
  "readiness": {
    "status": "healthy",
    "checks": { ... }
  },
  "liveness": {
    "status": "healthy",
    "checks": { ... }
  },
  "cached_results": { ... }
}
```

## Kubernetes 集成

### Deployment 配置示例

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-service
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: my-service
        image: my-service:latest
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
```

## 最佳实践

1. **区分健康、就绪和存活检查**：
   - 健康检查：检查服务及其依赖的整体状态
   - 就绪检查：检查服务是否准备好接收流量（如初始化完成）
   - 存活检查：检查服务是否仍在运行（如心跳）

2. **设置合理的超时**：
   - 避免检查时间过长，影响系统响应
   - 根据依赖服务的实际响应时间调整超时设置

3. **使用心跳机制**：
   - 对于长时间运行的服务，使用心跳机制维护存活状态
   - 定期更新心跳时间戳

4. **缓存结果**：
   - 利用结果缓存机制，避免频繁执行昂贵的检查操作
   - 在详细检查端点中查看缓存结果

5. **监控和告警**：
   - 集成 Prometheus 等监控系统
   - 设置告警规则，及时发现健康问题

## 架构设计

健康检查系统采用适配器模式，提供统一的接口来管理不同类型的健康检查：

```
┌─────────────────────────────────────────────────┐
│          HealthCheckAdapter                      │
│  ┌───────────────────────────────────────────┐ │
│  │  HealthCheckers                            │ │
│  │  - Database Checker                        │ │
│  │  - Redis Checker                           │ │
│  │  - ...                                     │ │
│  └───────────────────────────────────────────┘ │
│  ┌───────────────────────────────────────────┐ │
│  │  ReadinessCheckers                         │ │
│  │  - Service Readiness Probe                 │ │
│  └───────────────────────────────────────────┘ │
│  ┌───────────────────────────────────────────┐ │
│  │  LivenessCheckers                          │ │
│  │  - Service Liveness Probe                  │ │
│  └───────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
                    │
                    ▼
      ┌────────────────────────┐
      │   EndpointHandler       │
      │  - /health              │
      │  - /health/ready        │
      │  - /health/live         │
      │  - /health/detailed     │
      └────────────────────────┘
```

## 相关文档

- [SWIT Server Framework](../README.md)
- [Observability Integration](../observability.go)
- [Types Package](../../types/health.go)
