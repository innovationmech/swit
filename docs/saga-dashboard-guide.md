# Saga Dashboard 用户指南

本指南介绍如何使用 Saga 监控面板来监控和管理分布式事务。

## 目录

- [概述](#概述)
- [快速开始](#快速开始)
- [功能介绍](#功能介绍)
- [配置说明](#配置说明)
- [使用场景](#使用场景)
- [故障排查](#故障排查)
- [最佳实践](#最佳实践)

## 概述

Saga Dashboard 是一个 Web 监控面板，提供以下核心功能：

- **实时监控**: 查看 Saga 执行状态和系统指标
- **流程可视化**: 直观展示 Saga 执行流程
- **操作干预**: 手动取消或重试 Saga
- **告警管理**: 接收和处理告警通知
- **历史查询**: 查询历史 Saga 执行记录

### 系统架构

```
┌─────────────┐
│   浏览器    │
└──────┬──────┘
       │ HTTP/WebSocket
       ▼
┌─────────────────────┐
│  Dashboard Server   │
│  (监控面板服务)     │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Saga Coordinator    │
│ (Saga 协调器)       │
└─────────────────────┘
```

## 快速开始

### 启动监控面板

#### 方式 1: 使用示例程序

```bash
# 创建配置文件
cp examples/saga-dashboard-config.yaml config.yaml

# 启动监控面板
go run examples/saga-monitoring/main.go
```

#### 方式 2: 编程方式启动

```go
package main

import (
    "context"
    "log"

    "github.com/innovationmech/swit/pkg/saga"
    "github.com/innovationmech/swit/pkg/saga/monitoring"
)

func main() {
    // 创建 Saga 协调器
    coordinator, err := saga.NewCoordinator(/* config */)
    if err != nil {
        log.Fatal(err)
    }

    // 创建监控面板
    dashboard, err := monitoring.NewSagaDashboard(&monitoring.DashboardConfig{
        Coordinator: coordinator,
        ServerConfig: &monitoring.ServerConfig{
            Address: ":8080",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // 启动面板
    if err := dashboard.Start(context.Background()); err != nil {
        log.Fatal(err)
    }

    log.Println("Dashboard started at http://localhost:8080")
    
    // 等待程序退出
    select {}
}
```

### 访问面板

在浏览器中打开：`http://localhost:8080`

## 功能介绍

### 1. 实时指标面板

面板顶部显示关键指标：

#### 指标卡片

- **总计 Saga**: 系统中所有 Saga 的总数
- **成功完成**: 成功完成的 Saga 数量
- **运行中**: 当前正在执行的 Saga 数量
- **失败**: 执行失败的 Saga 数量

#### 详细指标

- **平均执行时间**: Saga 平均执行耗时
- **成功率**: Saga 成功完成的百分比

#### 自动刷新

指标数据每 5 秒自动刷新一次，确保实时性。

### 2. 告警列表

当系统检测到异常情况时，会显示告警区域：

#### 告警类型

- **执行超时**: Saga 执行时间超过阈值
- **失败率高**: 系统失败率超过设定值
- **资源不足**: 系统资源使用率过高
- **依赖服务异常**: 依赖的外部服务不可用

#### 告警操作

- **确认**: 标记告警为已知，不再重复提醒
- **查看详情**: 跳转到相关 Saga 详情页面

### 3. Saga 列表

展示系统中的 Saga 实例，支持以下功能：

#### 筛选功能

使用状态下拉菜单过滤 Saga：
- 全部
- 运行中
- 已完成
- 失败
- 补偿中
- 已补偿

#### 列表字段

| 字段 | 说明 |
|------|------|
| Saga ID | Saga 唯一标识符，点击可查看详情 |
| 类型 | Saga 业务类型 |
| 状态 | 当前执行状态 |
| 进度 | 执行进度百分比 |
| 开始时间 | Saga 启动时间 |
| 持续时间 | 已执行时长 |
| 操作 | 可执行的操作按钮 |

#### 操作按钮

- **详情**: 查看 Saga 完整信息
- **流程图**: 查看 Saga 执行流程可视化
- **取消**: 取消正在运行的 Saga（仅运行中状态可用）
- **重试**: 重新执行失败的 Saga（仅失败/已补偿状态可用）

#### 分页导航

- 使用底部分页控件浏览多页数据
- 显示当前页码和总页数

### 4. Saga 详情弹窗

点击 "详情" 按钮或 Saga ID 打开详情弹窗，包含：

#### 基本信息

- Saga ID (完整)
- 类型
- 状态
- 开始时间
- 完成时间 (如果已完成)
- 执行时间

#### 错误信息

如果 Saga 失败，显示详细错误信息。

#### 执行步骤

列出所有执行步骤，每个步骤包含：
- 步骤名称
- 状态
- 开始时间
- 完成时间
- 执行时长
- 错误信息（如果有）

### 5. 流程可视化

点击 "流程图" 按钮查看 Saga 执行流程的可视化展示：

#### 可视化元素

- **步骤节点**: 每个步骤显示为一个矩形框
- **状态颜色**: 
  - 绿色: 成功完成
  - 黄色: 正在运行
  - 红色: 失败
- **流程箭头**: 显示步骤执行顺序
- **时间信息**: 每个步骤显示执行耗时

#### 布局方式

- 顺序执行: 从上到下垂直排列
- 并行执行: 横向并列显示

## 配置说明

### 服务器配置

```yaml
# 监控面板服务器配置
monitoring:
  # 服务地址
  address: ":8080"
  
  # 服务器超时设置
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s
  
  # Gin 框架模式
  gin_mode: release  # debug, release, test
  
  # 优雅关闭
  enable_graceful_shutdown: true
  graceful_shutdown_timeout: 30s
  
  # TLS 配置 (可选)
  enable_tls: false
  tls:
    cert_file: /path/to/cert.pem
    key_file: /path/to/key.pem
    min_version: "1.2"
```

### 健康检查配置

```yaml
# 健康检查配置
health:
  # 健康检查路径
  path: /api/health
  
  # 检查超时
  timeout: 5s
  
  # 组件检查
  components:
    - name: coordinator
      enabled: true
    - name: database
      enabled: true
    - name: message_broker
      enabled: true
```

### 告警配置

```yaml
# 告警规则配置
alerts:
  # 启用告警
  enabled: true
  
  # 告警规则
  rules:
    # Saga 执行超时
    - name: saga_duration_threshold
      condition: duration_ms > 30000
      severity: high
      enabled: true
    
    # 失败率过高
    - name: saga_failure_rate
      condition: failure_rate > 0.1
      severity: critical
      enabled: true
    
    # 运行中 Saga 过多
    - name: running_sagas_threshold
      condition: running_sagas > 1000
      severity: medium
      enabled: true
```

### 指标采集配置

```yaml
# 指标采集配置
metrics:
  # 启用 Prometheus
  enable_prometheus: true
  prometheus_path: /metrics
  
  # 采集间隔
  collection_interval: 5s
  
  # 保留时长
  retention_period: 24h
```

## 使用场景

### 场景 1: 监控 Saga 执行状态

**需求**: 实时了解系统中 Saga 的执行情况

**操作步骤**:
1. 打开监控面板首页
2. 查看顶部指标卡片，了解整体运行状况
3. 观察 "运行中" 和 "失败" 指标是否正常
4. 如有异常，查看告警区域获取详细信息

### 场景 2: 排查失败的 Saga

**需求**: 找出失败原因并采取措施

**操作步骤**:
1. 在状态过滤器中选择 "失败"
2. 浏览失败的 Saga 列表
3. 点击某个 Saga 的 "详情" 按钮
4. 查看错误信息和执行步骤，定位失败原因
5. 如需查看流程，点击 "流程图" 按钮
6. 修复问题后，点击 "重试" 按钮

### 场景 3: 手动干预 Saga

**需求**: 取消异常的 Saga 执行

**操作步骤**:
1. 在状态过滤器中选择 "运行中"
2. 找到需要取消的 Saga
3. 点击 "取消" 按钮
4. 在确认对话框中确认操作
5. 系统将触发 Saga 补偿流程

### 场景 4: 分析性能问题

**需求**: 找出执行缓慢的 Saga

**操作步骤**:
1. 查看 "平均执行时间" 指标
2. 在 Saga 列表中按持续时间排序（如果支持）
3. 查看执行时间较长的 Saga 详情
4. 分析各步骤耗时，找出性能瓶颈
5. 优化慢步骤的实现

### 场景 5: 响应告警

**需求**: 处理系统告警

**操作步骤**:
1. 注意到告警区域出现新告警
2. 阅读告警标题和描述
3. 点击告警中的 Saga ID 查看详情
4. 根据告警类型采取相应措施
5. 完成处理后，点击 "确认" 按钮

## 故障排查

### 问题 1: 无法访问监控面板

**症状**: 浏览器无法打开面板页面

**排查步骤**:
1. 检查服务是否正常启动
   ```bash
   curl http://localhost:8080/api/health
   ```
2. 检查端口是否被占用
   ```bash
   lsof -i :8080
   ```
3. 查看服务日志
4. 检查防火墙设置

**解决方案**:
- 确保配置的端口未被占用
- 检查监听地址配置是否正确
- 确认防火墙允许访问该端口

### 问题 2: 数据不更新

**症状**: 面板显示的数据长时间不变化

**排查步骤**:
1. 检查浏览器控制台是否有错误
2. 检查网络连接是否正常
3. 验证 API 端点是否响应
   ```bash
   curl http://localhost:8080/api/metrics/realtime
   ```
4. 检查协调器是否正常运行

**解决方案**:
- 刷新浏览器页面
- 检查协调器与面板的连接
- 确认指标采集功能正常

### 问题 3: 操作失败

**症状**: 取消或重试 Saga 时报错

**排查步骤**:
1. 查看浏览器控制台错误信息
2. 检查 Saga 当前状态是否支持该操作
3. 验证 API 响应
   ```bash
   curl -X POST http://localhost:8080/api/sagas/{saga_id}/cancel
   ```
4. 查看服务端日志

**解决方案**:
- 确认 Saga 状态正确
- 检查权限配置
- 查看详细错误信息并修复

### 问题 4: 性能问题

**症状**: 面板加载缓慢或卡顿

**排查步骤**:
1. 检查 Saga 数量是否过多
2. 验证数据库查询性能
3. 检查网络延迟
4. 查看服务器资源使用情况

**解决方案**:
- 使用分页和过滤减少数据量
- 优化数据库查询
- 增加服务器资源
- 启用缓存机制

## 最佳实践

### 1. 监控策略

**定期检查**
- 每日查看关键指标趋势
- 关注失败率和平均执行时间
- 及时处理告警

**建立基线**
- 记录正常情况下的指标范围
- 设置合理的告警阈值
- 定期审查和调整

### 2. 操作规范

**取消 Saga**
- 谨慎使用取消功能
- 取消前确认业务影响
- 记录取消原因

**重试 Saga**
- 确认失败原因已修复
- 避免重复重试相同问题
- 监控重试后的执行情况

### 3. 性能优化

**数据查询**
- 使用状态过滤减少数据量
- 合理设置分页大小
- 避免频繁刷新

**历史数据**
- 定期归档旧数据
- 设置合理的保留策略
- 使用时间范围过滤

### 4. 安全建议

**访问控制**
- 限制监控面板的访问权限
- 使用身份认证机制
- 记录操作审计日志

**网络安全**
- 生产环境启用 HTTPS
- 配置防火墙规则
- 限制 API 访问速率

### 5. 集成建议

**告警通知**
- 集成 Slack、钉钉等通知渠道
- 配置告警分级策略
- 设置值班轮换机制

**监控系统**
- 集成 Prometheus 采集指标
- 配置 Grafana 可视化面板
- 设置 SLA 监控

**日志系统**
- 集成 ELK 或 Loki 日志系统
- 关联 Saga ID 追踪日志
- 配置日志告警规则

## 键盘快捷键

| 快捷键 | 功能 |
|--------|------|
| `Esc` | 关闭当前弹窗 |
| `F5` | 刷新数据 |
| `Ctrl/Cmd + F` | 搜索 (浏览器默认) |

## 浏览器兼容性

支持的浏览器版本：
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

**注意**: 不支持 IE 浏览器

## 移动端支持

监控面板采用响应式设计，支持移动设备访问：
- 自适应屏幕尺寸
- 触摸操作优化
- 简化的移动端布局

## 常见问题 (FAQ)

**Q: 如何修改默认端口？**
A: 在配置文件中修改 `monitoring.address` 或 `monitoring.port` 配置项。

**Q: 能否同时监控多个 Saga 协调器？**
A: 当前版本每个面板实例只能监控一个协调器。可以启动多个面板实例监控不同的协调器。

**Q: 历史数据保留多久？**
A: 默认保留 24 小时，可通过 `metrics.retention_period` 配置修改。

**Q: 如何导出 Saga 数据？**
A: 可以通过 API 导出数据，或集成 Prometheus 等监控系统长期存储。

**Q: 面板支持多语言吗？**
A: 当前版本仅支持中文，未来版本将支持多语言。

## 相关资源

- [API 文档](saga-dashboard-api.md)
- [配置示例](../examples/saga-dashboard-config.yaml)
- [Saga 用户指南](saga-user-guide.md)
- [Saga 监控指南](saga-monitoring-guide.md)
- [GitHub Issues](https://github.com/innovationmech/swit/issues)

## 获取帮助

如果遇到问题或需要帮助：

1. 查看本指南和 [API 文档](saga-dashboard-api.md)
2. 搜索 [GitHub Issues](https://github.com/innovationmech/swit/issues)
3. 提交新 Issue 描述问题
4. 加入社区讨论群

## 更新日志

### v1.0.0 (2025-10-23)
- 初始版本发布
- 实现基础监控功能
- 支持 Saga 列表和详情查询
- 实现流程可视化
- 添加操作干预功能
- 集成告警系统

