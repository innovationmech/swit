# Bulk Operations with Partial Failure Handling

## 概述

批量操作模块提供了一个通用的批量操作执行器，支持并发执行、部分失败处理和详细的结果报告。

## 特性

- **并发执行**：支持配置并发度，可以串行或并发执行批量操作
- **部分失败处理**：支持两种失败处理策略
  - **尽力而为（Best Effort）**：即使部分操作失败也继续执行
  - **快速失败（Fail Fast）**：遇到第一个失败立即停止
- **重试支持**：可选的重试机制，支持指数退避等策略
- **进度追踪**：实时进度回调，便于监控执行进度
- **详细报告**：提供成功/失败统计、执行时间、错误详情等

## 核心类型

### BulkOperation[T]

批量操作的单个项：

```go
type BulkOperation[T any] struct {
    Index int  // 操作索引
    Data  T    // 操作数据
}
```

### BulkResult[T]

单个操作的结果：

```go
type BulkResult[T any] struct {
    Index      int           // 操作索引
    Data       T             // 结果数据
    Error      error         // 错误（如果有）
    Duration   time.Duration // 执行耗时
    RetryCount int           // 重试次数
}
```

### BulkReport[T]

批量操作的完整报告：

```go
type BulkReport[T any] struct {
    TotalCount    int           // 总操作数
    SuccessCount  int           // 成功数
    FailureCount  int           // 失败数
    TotalDuration time.Duration // 总耗时
    Results       []BulkResult[T] // 所有结果
}
```

### BulkConfig

批量操作配置：

```go
type BulkConfig struct {
    Concurrency     int        // 并发数（0=串行，<0=无限并发）
    ContinueOnError bool       // 遇到错误是否继续
    RetryEnabled    bool       // 是否启用重试
    RetryConfig     *Config    // 重试配置
    OnProgress      func(completed, total int) // 进度回调
}
```

## 使用示例

### 基本用法

```go
// 创建批量执行器
config := resilience.BulkConfig{
    Concurrency:     5,    // 并发度为 5
    ContinueOnError: true, // 尽力而为
}

executor, err := resilience.NewBulkExecutor[int, int](config)
if err != nil {
    log.Fatal(err)
}

// 准备操作
operations := []resilience.BulkOperation[int]{
    {Index: 0, Data: 1},
    {Index: 1, Data: 2},
    {Index: 2, Data: 3},
}

// 定义操作函数
operationFn := func(ctx context.Context, data int) (int, error) {
    return data * 2, nil
}

// 执行批量操作
report := executor.Execute(context.Background(), operations, operationFn)

// 检查结果
fmt.Printf("Success: %d, Failed: %d\n", 
    report.SuccessCount, report.FailureCount)
```

### 部分失败处理

```go
config := resilience.BulkConfig{
    Concurrency:     3,
    ContinueOnError: true, // 即使部分失败也继续
}

executor, _ := resilience.NewBulkExecutor[string, string](config)

operations := []resilience.BulkOperation[string]{
    {Index: 0, Data: "user1"},
    {Index: 1, Data: "user2"},
    {Index: 2, Data: ""}, // 会失败
    {Index: 3, Data: "user3"},
}

operationFn := func(ctx context.Context, data string) (string, error) {
    if data == "" {
        return "", errors.New("empty user ID")
    }
    return fmt.Sprintf("processed_%s", data), nil
}

report := executor.Execute(context.Background(), operations, operationFn)

// 处理部分成功的情况
if report.HasPartialSuccess() {
    // 获取成功的结果
    successResults := report.GetSuccessfulResults()
    for _, r := range successResults {
        fmt.Printf("Success: %s\n", r.Data)
    }
    
    // 获取失败的结果
    failedResults := report.GetFailedResults()
    for _, r := range failedResults {
        fmt.Printf("Failed at index %d: %v\n", r.Index, r.Error)
    }
}
```

### 带重试的批量操作

```go
// 配置重试策略
retryConfig := resilience.Config{
    Strategy:     resilience.StrategyExponential,
    MaxRetries:   2,
    InitialDelay: 100 * time.Millisecond,
    Multiplier:   2.0,
}

config := resilience.BulkConfig{
    Concurrency:     2,
    ContinueOnError: true,
    RetryEnabled:    true,
    RetryConfig:     &retryConfig,
}

executor, _ := resilience.NewBulkExecutor[int, string](config)

// 执行会自动重试失败的操作
report := executor.Execute(context.Background(), operations, operationFn)
```

### 进度追踪

```go
config := resilience.BulkConfig{
    Concurrency: 5,
    OnProgress: func(completed, total int) {
        percentage := (completed * 100) / total
        fmt.Printf("Progress: %d%% (%d/%d)\n", 
            percentage, completed, total)
    },
}

executor, _ := resilience.NewBulkExecutor[int, int](config)
report := executor.Execute(context.Background(), operations, operationFn)
```

### 快速失败模式

```go
config := resilience.BulkConfig{
    Concurrency:     0,     // 串行执行
    ContinueOnError: false, // 遇到错误立即停止
}

executor, _ := resilience.NewBulkExecutor[int, int](config)
report := executor.Execute(context.Background(), operations, operationFn)

// 失败后的操作会被跳过
fmt.Printf("Processed: %d, Skipped: %d\n",
    report.SuccessCount,
    report.TotalCount - report.SuccessCount - 1)
```

## 报告分析

`BulkReport` 提供了多个便捷方法来分析执行结果：

```go
report := executor.Execute(ctx, operations, operationFn)

// 检查执行状态
if report.IsFullSuccess() {
    fmt.Println("All operations succeeded")
} else if report.IsFullFailure() {
    fmt.Println("All operations failed")
} else if report.HasPartialSuccess() {
    fmt.Println("Partial success")
}

// 获取特定结果
successResults := report.GetSuccessfulResults()
failedResults := report.GetFailedResults()

// 计算统计信息
successRate := (float64(report.SuccessCount) / float64(report.TotalCount)) * 100
avgDuration := report.TotalDuration / time.Duration(report.TotalCount)

fmt.Printf("Success rate: %.1f%%\n", successRate)
fmt.Printf("Average duration: %s\n", avgDuration)
```

## 使用场景

1. **批量数据处理**：批量插入、更新或删除数据库记录
2. **批量 API 调用**：批量调用外部 API 或微服务
3. **批量文件处理**：批量上传、下载或转换文件
4. **批量消息发送**：批量发送邮件、短信或推送通知
5. **批量数据验证**：批量验证和清洗数据

## 性能考虑

1. **并发度设置**：根据 I/O 密集型还是 CPU 密集型任务调整并发度
   - I/O 密集型（网络调用、文件操作）：可以设置较高的并发度
   - CPU 密集型（数据处理、计算）：建议设置为 CPU 核心数的 1-2 倍

2. **内存使用**：批量操作会在内存中保存所有结果，大批量操作时注意内存使用

3. **错误处理**：
   - 使用 `ContinueOnError=true` 可以避免少数失败影响整体
   - 使用 `ContinueOnError=false` 可以在发现问题时快速停止，节省资源

4. **重试策略**：合理配置重试次数和延迟，避免过度重试造成资源浪费

## 最佳实践

1. **合理设置批量大小**：避免单次批量操作过大，建议分批处理
2. **使用上下文控制**：传入带超时或取消的 context，避免操作无限期阻塞
3. **记录失败详情**：通过 `GetFailedResults()` 获取失败的操作，便于排查和重试
4. **监控执行进度**：使用 `OnProgress` 回调实时监控执行进度
5. **错误分类处理**：根据错误类型决定是否重试或发送告警

## 与其他模块的集成

批量操作可以与其他 resilience 模块配合使用：

- **Circuit Breaker**：在操作函数中使用熔断器，避免级联失败
- **DLQ Manager**：将失败的操作发送到死信队列，后续处理
- **Metrics**：收集批量操作的性能指标，用于监控和告警

