# Saga Dashboard API 文档

本文档描述了 Saga 监控面板提供的 REST API 接口。

## 目录

- [基础信息](#基础信息)
- [健康检查](#健康检查)
- [Saga 管理](#saga-管理)
- [指标数据](#指标数据)
- [流程可视化](#流程可视化)
- [告警管理](#告警管理)
- [错误处理](#错误处理)

## 基础信息

### 服务器配置

- **基础 URL**: `http://localhost:8080` (默认)
- **API 前缀**: `/api`
- **内容类型**: `application/json`
- **字符编码**: `UTF-8`

### 通用响应格式

成功响应示例：
```json
{
  "data": { ... },
  "message": "success"
}
```

错误响应示例：
```json
{
  "error": "错误描述",
  "code": "ERROR_CODE",
  "message": "详细错误信息"
}
```

## 健康检查

### 1. 基础健康检查

获取服务健康状态。

**请求**
```http
GET /api/health
```

**响应**
```json
{
  "status": "healthy",
  "service": "saga-monitoring",
  "timestamp": "2025-10-23T12:00:00Z"
}
```

### 2. 存活检查 (Liveness Probe)

Kubernetes 存活探针，检查应用是否存活。

**请求**
```http
GET /api/health/live
```

**响应**
```json
{
  "status": "alive",
  "timestamp": "2025-10-23T12:00:00Z"
}
```

### 3. 就绪检查 (Readiness Probe)

Kubernetes 就绪探针，检查应用是否准备好接收流量。

**请求**
```http
GET /api/health/ready
```

**响应**
```json
{
  "status": "ready",
  "timestamp": "2025-10-23T12:00:00Z"
}
```

## Saga 管理

### 1. 获取 Saga 列表

查询 Saga 实例列表，支持分页和过滤。

**请求**
```http
GET /api/sagas?page=1&page_size=20&status=running
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | integer | 否 | 页码，从 1 开始，默认 1 |
| page_size | integer | 否 | 每页数量，默认 20，最大 100 |
| status | string | 否 | 状态过滤：running, completed, failed, compensating, compensated |
| type | string | 否 | Saga 类型过滤 |
| from | string | 否 | 开始时间过滤 (ISO 8601 格式) |
| to | string | 否 | 结束时间过滤 (ISO 8601 格式) |

**响应**
```json
{
  "sagas": [
    {
      "id": "saga-123e4567-e89b-12d3-a456-426614174000",
      "type": "OrderProcessingSaga",
      "status": "running",
      "started_at": "2025-10-23T10:30:00Z",
      "completed_at": null,
      "duration_ms": 5000,
      "steps": [
        {
          "name": "ReserveInventory",
          "status": "completed"
        },
        {
          "name": "ProcessPayment",
          "status": "running"
        }
      ],
      "error": null
    }
  ],
  "pagination": {
    "current_page": 1,
    "page_size": 20,
    "total_pages": 5,
    "total_count": 98
  }
}
```

**状态说明**

- `running`: Saga 正在执行中
- `completed`: Saga 成功完成
- `failed`: Saga 执行失败
- `compensating`: 正在执行补偿操作
- `compensated`: 补偿操作完成
- `cancelled`: Saga 已被取消

### 2. 获取 Saga 详情

获取指定 Saga 的详细信息。

**请求**
```http
GET /api/sagas/{saga_id}
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| saga_id | string | Saga 唯一标识符 |

**响应**
```json
{
  "id": "saga-123e4567-e89b-12d3-a456-426614174000",
  "type": "OrderProcessingSaga",
  "status": "completed",
  "started_at": "2025-10-23T10:30:00Z",
  "completed_at": "2025-10-23T10:30:15Z",
  "duration_ms": 15000,
  "steps": [
    {
      "name": "ReserveInventory",
      "status": "completed",
      "started_at": "2025-10-23T10:30:00Z",
      "completed_at": "2025-10-23T10:30:05Z",
      "duration_ms": 5000,
      "error": null
    },
    {
      "name": "ProcessPayment",
      "status": "completed",
      "started_at": "2025-10-23T10:30:05Z",
      "completed_at": "2025-10-23T10:30:15Z",
      "duration_ms": 10000,
      "error": null
    }
  ],
  "context": {
    "order_id": "ORD-12345",
    "user_id": "USR-67890"
  },
  "error": null
}
```

### 3. 取消 Saga

取消正在运行的 Saga 实例。

**请求**
```http
POST /api/sagas/{saga_id}/cancel
Content-Type: application/json

{
  "reason": "用户取消订单"
}
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| saga_id | string | Saga 唯一标识符 |

**请求体**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| reason | string | 否 | 取消原因 |

**响应**
```json
{
  "success": true,
  "message": "Saga cancelled successfully",
  "saga_id": "saga-123e4567-e89b-12d3-a456-426614174000"
}
```

### 4. 重试 Saga

重试失败或已补偿的 Saga。

**请求**
```http
POST /api/sagas/{saga_id}/retry
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| saga_id | string | Saga 唯一标识符 |

**响应**
```json
{
  "success": true,
  "message": "Saga retry initiated",
  "new_saga_id": "saga-234e5678-e89b-12d3-a456-426614174001"
}
```

## 指标数据

### 1. 获取指标数据

获取 Saga 执行的统计指标。

**请求**
```http
GET /api/metrics
```

**响应**
```json
{
  "total_sagas": 1000,
  "completed_sagas": 850,
  "running_sagas": 100,
  "failed_sagas": 50,
  "avg_duration_ms": 8500,
  "success_rate": 0.85,
  "throughput": {
    "last_minute": 10,
    "last_hour": 450,
    "last_day": 8000
  },
  "timestamp": "2025-10-23T12:00:00Z"
}
```

### 2. 获取实时指标

获取实时指标快照，用于监控面板实时更新。

**请求**
```http
GET /api/metrics/realtime
```

**响应**
```json
{
  "total_sagas": 1000,
  "completed_sagas": 850,
  "running_sagas": 100,
  "failed_sagas": 50,
  "avg_duration_ms": 8500,
  "success_rate": 0.85,
  "active_connections": 25,
  "timestamp": "2025-10-23T12:00:00Z"
}
```

### 3. 指标流 (SSE)

通过 Server-Sent Events 获取实时指标推送。

**请求**
```http
GET /api/metrics/stream
Accept: text/event-stream
```

**响应** (事件流)
```
event: metrics
data: {"total_sagas":1000,"completed_sagas":850,"running_sagas":100}

event: metrics
data: {"total_sagas":1001,"completed_sagas":851,"running_sagas":99}
```

## 流程可视化

### 获取 Saga 流程可视化数据

获取 Saga 执行流程的可视化数据。

**请求**
```http
GET /api/sagas/{saga_id}/visualization
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| saga_id | string | Saga 唯一标识符 |

**响应**
```json
{
  "saga_id": "saga-123e4567-e89b-12d3-a456-426614174000",
  "type": "OrderProcessingSaga",
  "steps": [
    {
      "name": "ReserveInventory",
      "status": "completed",
      "duration_ms": 5000,
      "dependencies": []
    },
    {
      "name": "ProcessPayment",
      "status": "completed",
      "duration_ms": 10000,
      "dependencies": ["ReserveInventory"]
    },
    {
      "name": "SendConfirmation",
      "status": "completed",
      "duration_ms": 2000,
      "dependencies": ["ProcessPayment"]
    }
  ],
  "layout": "sequential",
  "total_duration_ms": 17000
}
```

## 告警管理

### 1. 获取告警列表

获取活跃的告警信息。

**请求**
```http
GET /api/alerts?severity=high&status=active
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| severity | string | 否 | 严重级别：low, medium, high, critical |
| status | string | 否 | 状态：active, acknowledged, resolved |
| page | integer | 否 | 页码，默认 1 |
| page_size | integer | 否 | 每页数量，默认 20 |

**响应**
```json
{
  "alerts": [
    {
      "id": "alert-001",
      "title": "Saga 执行时间过长",
      "description": "Saga saga-123 执行时间超过阈值",
      "severity": "high",
      "status": "active",
      "saga_id": "saga-123e4567-e89b-12d3-a456-426614174000",
      "timestamp": "2025-10-23T11:30:00Z",
      "rule": "saga_duration_threshold",
      "threshold": 30000,
      "actual_value": 45000
    }
  ],
  "pagination": {
    "current_page": 1,
    "page_size": 20,
    "total_count": 5
  }
}
```

### 2. 获取告警统计

获取告警统计信息。

**请求**
```http
GET /api/alerts/stats
```

**响应**
```json
{
  "total_alerts": 50,
  "active_alerts": 10,
  "acknowledged_alerts": 25,
  "resolved_alerts": 15,
  "by_severity": {
    "critical": 2,
    "high": 5,
    "medium": 18,
    "low": 25
  },
  "last_24h": 30
}
```

### 3. 确认告警

确认一个告警。

**请求**
```http
POST /api/alerts/{alert_id}/acknowledge
Content-Type: application/json

{
  "comment": "已知问题，正在处理"
}
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| alert_id | string | 告警唯一标识符 |

**请求体**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| comment | string | 否 | 确认备注 |

**响应**
```json
{
  "success": true,
  "message": "Alert acknowledged",
  "alert_id": "alert-001"
}
```

### 4. 获取告警规则

获取配置的告警规则。

**请求**
```http
GET /api/alerts/rules
```

**响应**
```json
{
  "rules": [
    {
      "name": "saga_duration_threshold",
      "description": "Saga 执行时间超过阈值",
      "condition": "duration_ms > 30000",
      "severity": "high",
      "enabled": true
    },
    {
      "name": "saga_failure_rate",
      "description": "Saga 失败率过高",
      "condition": "failure_rate > 0.1",
      "severity": "critical",
      "enabled": true
    }
  ]
}
```

## 错误处理

### HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 201 | 资源创建成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |
| 503 | 服务不可用 |

### 错误响应格式

```json
{
  "error": "invalid_parameter",
  "message": "Invalid saga_id format",
  "details": {
    "field": "saga_id",
    "value": "invalid-id",
    "expected": "UUID format"
  }
}
```

### 常见错误代码

| 错误代码 | 说明 |
|----------|------|
| `invalid_parameter` | 请求参数无效 |
| `saga_not_found` | Saga 不存在 |
| `saga_not_running` | Saga 未在运行状态 |
| `operation_failed` | 操作执行失败 |
| `internal_error` | 内部错误 |

## API 使用示例

### cURL 示例

**获取 Saga 列表**
```bash
curl -X GET "http://localhost:8080/api/sagas?page=1&page_size=10&status=running"
```

**获取 Saga 详情**
```bash
curl -X GET "http://localhost:8080/api/sagas/saga-123e4567-e89b-12d3-a456-426614174000"
```

**取消 Saga**
```bash
curl -X POST "http://localhost:8080/api/sagas/saga-123e4567-e89b-12d3-a456-426614174000/cancel" \
  -H "Content-Type: application/json" \
  -d '{"reason": "用户取消"}'
```

**获取实时指标**
```bash
curl -X GET "http://localhost:8080/api/metrics/realtime"
```

### JavaScript/Fetch 示例

```javascript
// 获取 Saga 列表
async function getSagas(page = 1, status = '') {
  const query = new URLSearchParams({ page, page_size: 20 });
  if (status) query.append('status', status);
  
  const response = await fetch(`/api/sagas?${query}`);
  return await response.json();
}

// 取消 Saga
async function cancelSaga(sagaId, reason) {
  const response = await fetch(`/api/sagas/${sagaId}/cancel`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ reason })
  });
  return await response.json();
}

// 监听实时指标流
function subscribeToMetrics() {
  const eventSource = new EventSource('/api/metrics/stream');
  
  eventSource.addEventListener('metrics', (event) => {
    const metrics = JSON.parse(event.data);
    console.log('Received metrics:', metrics);
  });
  
  return eventSource;
}
```

### Go 客户端示例

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

// 获取 Saga 列表
func getSagas(baseURL string, page int, status string) (map[string]interface{}, error) {
    url := fmt.Sprintf("%s/api/sagas?page=%d&page_size=20&status=%s", baseURL, page, status)
    
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result, nil
}

// 取消 Saga
func cancelSaga(baseURL, sagaID, reason string) error {
    url := fmt.Sprintf("%s/api/sagas/%s/cancel", baseURL, sagaID)
    
    body := map[string]string{"reason": reason}
    jsonBody, _ := json.Marshal(body)
    
    resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("cancel failed: status %d", resp.StatusCode)
    }
    
    return nil
}
```

## 性能优化建议

1. **分页查询**: 大量数据时使用分页参数，避免一次性加载所有数据
2. **状态过滤**: 使用状态过滤减少返回数据量
3. **SSE 连接**: 使用 `/api/metrics/stream` 替代轮询获取实时数据
4. **缓存策略**: 合理使用 HTTP 缓存头
5. **批量操作**: 对于需要操作多个 Saga 的场景，考虑使用批量 API (如果实现)

## 安全说明

1. **认证**: 生产环境应启用 API 认证机制
2. **授权**: 根据用户角色限制 API 访问权限
3. **HTTPS**: 生产环境应使用 HTTPS 保护数据传输
4. **速率限制**: 建议配置 API 速率限制防止滥用
5. **CORS**: 根据需要配置 CORS 策略

## 相关文档

- [Saga Dashboard 用户指南](saga-dashboard-guide.md)
- [Saga 用户指南](saga-user-guide.md)
- [Saga 监控指南](saga-monitoring-guide.md)
- [配置参考](configuration-reference.md)

