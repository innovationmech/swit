# 对称传输架构示例

本示例演示了如何使用 SWIT 项目中新的对称传输架构，解决了之前 gRPC 服务依赖 HTTP 传输注册表的耦合问题。

## 架构特点

- **独立注册表**: 每个传输类型（HTTP、gRPC）都有独立的服务注册表
- **解耦设计**: gRPC 服务不再依赖 HTTP 传输的注册表
- **灵活注册**: 服务可以选择注册到特定传输或多个传输
- **向后兼容**: 现有代码无需修改即可继续工作

## 使用方式

```go
// 创建对称服务器
server := NewSymmetricServer()

// 注册只支持 HTTP 的服务
httpService := &MyHTTPService{}
server.RegisterHTTPOnlyService(httpService)

// 注册只支持 gRPC 的服务  
grpcService := &MyGRPCService{}
server.RegisterGRPCOnlyService(grpcService)

// 注册同时支持两种协议的服务
dualService := &MyDualService{}
server.RegisterDualProtocolService(dualService)

// 启动服务器
ctx := context.Background()
if err := server.Start(ctx); err != nil {
    log.Fatal(err)
}
```

## 架构优势

1. **解耦**: 传输层之间无依赖关系
2. **可扩展**: 易于添加新的传输类型（如 WebSocket）
3. **灵活**: 服务可按需选择支持的传输协议
4. **维护性**: 清晰的关注点分离