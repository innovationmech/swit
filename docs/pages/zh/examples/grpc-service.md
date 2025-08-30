# gRPC 服务示例

此示例演示了如何使用 Swit 框架和 Protocol Buffers 构建纯 gRPC 服务。它展示了 gRPC 服务注册、Protocol Buffer 消息处理、服务器反射和健康服务集成。

## 概述

gRPC 服务示例（`examples/grpc-service/`）提供：
- **纯 gRPC 服务** - 专注于基于 Protocol Buffer 的 RPC 功能
- **Protocol Buffer 集成** - 使用从 .proto 定义生成的 Go 代码
- **服务器反射** - 用于开发和调试的 gRPC 反射
- **健康服务** - 标准 gRPC 健康检查协议
- **Keepalive 配置** - 连接管理和优化

## 主要特性

### 服务架构
- 实现 `BusinessServiceRegistrar` 以实现框架集成
- 使用适当的服务器配置注册 gRPC 服务
- 使用 Protocol Buffer 定义实现类型安全通信
- 演示适当的 gRPC 服务生命周期管理

### gRPC 服务
- **GreeterService** - 实现 `swit.interaction.v1.GreeterService`
- **SayHello** RPC 方法 - 接受姓名参数并返回问候消息
- **健康服务** - 标准 gRPC 健康检查集成
- **反射服务** - 用于开发工具和调试

### 高级 gRPC 功能
- 消息大小限制（默认 4MB）
- 用于连接管理的 Keepalive 参数
- 服务器端验证和错误处理
- 适当的 gRPC 状态码和错误响应

## 代码结构

### 主服务实现

```go
// GreeterService 实现 ServiceRegistrar 接口
type GreeterService struct {
    name string
}

func (s *GreeterService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // 注册 gRPC 服务
    grpcService := &GreeterGRPCService{serviceName: s.name}
    if err := registry.RegisterBusinessGRPCService(grpcService); err != nil {
        return fmt.Errorf("注册 gRPC 服务失败: %w", err)
    }

    // 注册健康检查
    healthCheck := &GreeterHealthCheck{serviceName: s.name}
    if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
        return fmt.Errorf("注册健康检查失败: %w", err)
    }

    return nil
}
```

### gRPC 服务实现

```go
type GreeterGRPCService struct {
    serviceName string
    interaction.UnimplementedGreeterServiceServer
}

func (s *GreeterGRPCService) RegisterGRPC(server interface{}) error {
    grpcServer := server.(*grpc.Server)
    interaction.RegisterGreeterServiceServer(grpcServer, s)
    return nil
}

func (s *GreeterGRPCService) SayHello(ctx context.Context, req *interaction.SayHelloRequest) (*interaction.SayHelloResponse, error) {
    // 验证请求
    if req.GetName() == "" {
        return nil, status.Error(codes.InvalidArgument, "姓名不能为空")
    }

    // 创建响应
    response := &interaction.SayHelloResponse{
        Message: fmt.Sprintf("Hello, %s!", req.GetName()),
    }

    return response, nil
}
```

### gRPC 配置

```go
config := &server.ServerConfig{
    ServiceName: "grpc-greeter-service",
    HTTP: server.HTTPConfig{
        Enabled: false, // 仅 gRPC 服务
    },
    GRPC: server.GRPCConfig{
        Port:                getEnv("GRPC_PORT", "9090"),
        EnableReflection:    true,
        EnableHealthService: true,
        Enabled:             true,
        MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
        MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
        KeepaliveParams: server.GRPCKeepaliveParams{
            MaxConnectionIdle:     15 * time.Minute,
            MaxConnectionAge:      30 * time.Minute,
            MaxConnectionAgeGrace: 5 * time.Minute,
            Time:                  5 * time.Minute,
            Timeout:               1 * time.Minute,
        },
        KeepalivePolicy: server.GRPCKeepalivePolicy{
            MinTime:             5 * time.Minute,
            PermitWithoutStream: false,
        },
    },
}
```

## Protocol Buffer 定义

服务使用来自 `api/proto/swit/interaction/v1/` 的 Protocol Buffer 定义：

```protobuf
syntax = "proto3";

package swit.interaction.v1;

service GreeterService {
  rpc SayHello(SayHelloRequest) returns (SayHelloResponse);
}

message SayHelloRequest {
  string name = 1;
}

message SayHelloResponse {
  string message = 1;
}
```

## 运行示例

### 前置条件
- 已安装 Go 1.23.12+
- 已安装 gRPC 工具（用于测试）
- 框架依赖可用

### 快速开始

1. **导航到示例目录：**
   ```bash
   cd examples/grpc-service
   ```

2. **运行服务：**
   ```bash
   go run main.go
   ```

3. **使用 grpcurl 测试：**
   ```bash
   # 如果不可用，请安装 grpcurl
   go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
   
   # 测试 SayHello 方法
   grpcurl -plaintext -d '{"name": "Alice"}' \
           localhost:9090 \
           swit.interaction.v1.GreeterService/SayHello
   
   # 检查健康服务
   grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
   
   # 列出可用服务（反射）
   grpcurl -plaintext localhost:9090 list
   ```

### 环境配置

```bash
# 设置自定义 gRPC 端口
export GRPC_PORT=9999

# 启用服务发现
export DISCOVERY_ENABLED=true
export CONSUL_ADDRESS=localhost:8500

# 使用自定义配置运行
go run main.go
```

### 预期响应

**SayHello RPC：**
```json
{
  "message": "Hello, Alice!"
}
```

**健康检查：**
```json
{
  "status": "SERVING"
}
```

**服务列表：**
```
grpc.health.v1.Health
grpc.reflection.v1alpha.ServerReflection
swit.interaction.v1.GreeterService
```

## 开发模式

### 添加新的 RPC 方法

1. **更新 Protocol Buffer 定义：**
   ```protobuf
   service GreeterService {
     rpc SayHello(SayHelloRequest) returns (SayHelloResponse);
     rpc SayGoodbye(SayGoodbyeRequest) returns (SayGoodbyeResponse);
   }
   
   message SayGoodbyeRequest {
     string name = 1;
   }
   
   message SayGoodbyeResponse {
     string message = 1;
   }
   ```

2. **重新生成 Go 代码：**
   ```bash
   make proto
   ```

3. **实现方法：**
   ```go
   func (s *GreeterGRPCService) SayGoodbye(ctx context.Context, req *interaction.SayGoodbyeRequest) (*interaction.SayGoodbyeResponse, error) {
       if req.GetName() == "" {
           return nil, status.Error(codes.InvalidArgument, "姓名不能为空")
       }
   
       response := &interaction.SayGoodbyeResponse{
           Message: fmt.Sprintf("Goodbye, %s!", req.GetName()),
       }
   
       return response, nil
   }
   ```

### 错误处理模式

```go
func (s *GreeterGRPCService) ValidatedMethod(ctx context.Context, req *SomeRequest) (*SomeResponse, error) {
    // 输入验证
    if req.GetField() == "" {
        return nil, status.Error(codes.InvalidArgument, "field 是必需的")
    }
    
    // 带错误处理的业务逻辑
    result, err := s.processRequest(req)
    if err != nil {
        switch {
        case errors.Is(err, ErrNotFound):
            return nil, status.Error(codes.NotFound, "未找到资源")
        case errors.Is(err, ErrPermissionDenied):
            return nil, status.Error(codes.PermissionDenied, "访问被拒绝")
        default:
            return nil, status.Error(codes.Internal, "内部错误")
        }
    }
    
    return result, nil
}
```

### 上下文和超时处理

```go
func (s *GreeterGRPCService) LongRunningMethod(ctx context.Context, req *SomeRequest) (*SomeResponse, error) {
    // 检查取消
    select {
    case <-ctx.Done():
        return nil, status.Error(codes.Canceled, "请求被取消")
    default:
    }
    
    // 为外部调用设置超时
    timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    result, err := s.externalService.Call(timeoutCtx, req)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            return nil, status.Error(codes.DeadlineExceeded, "操作超时")
        }
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    return result, nil
}
```

## 测试 gRPC 服务

### 单元测试

```go
func TestGreeterService_SayHello(t *testing.T) {
    service := &GreeterGRPCService{serviceName: "test-service"}
    
    tests := []struct {
        name    string
        request *interaction.SayHelloRequest
        want    string
        wantErr codes.Code
    }{
        {
            name:    "有效请求",
            request: &interaction.SayHelloRequest{Name: "Alice"},
            want:    "Hello, Alice!",
            wantErr: codes.OK,
        },
        {
            name:    "空姓名",
            request: &interaction.SayHelloRequest{Name: ""},
            want:    "",
            wantErr: codes.InvalidArgument,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resp, err := service.SayHello(context.Background(), tt.request)
            
            if tt.wantErr != codes.OK {
                assert.Error(t, err)
                st, ok := status.FromError(err)
                assert.True(t, ok)
                assert.Equal(t, tt.wantErr, st.Code())
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.want, resp.GetMessage())
            }
        })
    }
}
```

### 集成测试

```go
func TestGRPCServer(t *testing.T) {
    // 启动测试服务器
    config := &server.ServerConfig{
        GRPC: server.GRPCConfig{
            Port:    "0", // 动态端口
            Enabled: true,
            EnableReflection: true,
        },
    }
    
    service := NewGreeterService("test")
    srv, err := server.NewBusinessServerCore(config, service, nil)
    require.NoError(t, err)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    err = srv.Start(ctx)
    require.NoError(t, err)
    defer srv.Shutdown()
    
    // 创建客户端连接
    conn, err := grpc.DialContext(ctx, srv.GetGRPCAddress(),
        grpc.WithInsecure(),
        grpc.WithBlock(),
    )
    require.NoError(t, err)
    defer conn.Close()
    
    // 测试服务
    client := interaction.NewGreeterServiceClient(conn)
    resp, err := client.SayHello(ctx, &interaction.SayHelloRequest{
        Name: "TestUser",
    })
    
    assert.NoError(t, err)
    assert.Equal(t, "Hello, TestUser!", resp.GetMessage())
}
```

## 演示的最佳实践

1. **Protocol Buffer 设计** - 具有适当字段命名的清洁消息定义
2. **错误处理** - 针对不同错误条件的适当 gRPC 状态码
3. **输入验证** - 带有有意义错误消息的服务器端验证
4. **上下文使用** - 适当的上下文传播和超时处理
5. **服务注册** - 与框架服务注册表的清洁集成

## 下一步

在理解这个 gRPC 示例后：

1. **与 HTTP 结合** - 查看 `full-featured-service` 了解双 HTTP/gRPC 传输
2. **高级功能** - 探索流式 RPC、拦截器和元数据
3. **生产设置** - 添加身份验证、速率限制和监控
4. **客户端库** - 为其他语言生成客户端代码

此示例使用现代 Protocol Buffer 模式为使用 Swit 框架构建基于 gRPC 的微服务提供了坚实的基础。