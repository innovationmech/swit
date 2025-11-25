# mTLS（双向 TLS）认证示例

本示例演示如何在 Swit 框架服务中实现双向 TLS（mTLS）认证。mTLS 通过要求服务器和客户端都出示并验证证书来提供强身份认证。

## 功能特性

### 安全特性

- ✅ **双向 TLS 认证** - 服务器和客户端相互验证对方的证书
- ✅ **基于证书的身份识别** - 从证书属性中提取客户端身份
- ✅ **基于角色的访问控制** - 根据证书属性限制端点访问
- ✅ **TLS 1.2/1.3 支持** - 现代 TLS 协议版本
- ✅ **安全密码套件** - 仅使用强加密算法

### 实现特性

- ✅ **HTTP mTLS 服务器** - 基于 Gin 的 HTTPS 服务器，支持客户端证书验证
- ✅ **gRPC mTLS 服务器** - 支持双向 TLS 的 gRPC 服务器
- ✅ **证书信息提取** - 解析 CN、OU、SAN 等证书字段
- ✅ **管理员授权** - 基于证书的授权示例
- ✅ **证书生成脚本** - 自动化证书生成，轻松设置

## 快速开始

### 前置条件

- Go 1.23+
- OpenSSL（用于证书生成）
- curl（用于测试）

### 1. 生成证书

```bash
cd examples/mtls-authentication

# 生成 CA、服务器和客户端证书
./scripts/generate-certs.sh
```

这将创建：
- `certs/ca.crt` / `certs/ca.key` - 证书颁发机构
- `certs/server.crt` / `certs/server.key` - 服务器证书
- `certs/client.crt` / `certs/client.key` - 普通客户端证书
- `certs/admin-client.crt` / `certs/admin-client.key` - 管理员客户端证书

### 2. 启动服务器

```bash
cd examples/mtls-authentication
go run main.go
```

服务器将启动在：
- **HTTPS**: https://localhost:8443
- **gRPC**: localhost:50443

### 3. 测试端点

#### 使用普通客户端证书测试：

```bash
# 公开端点
curl --cacert certs/ca.crt \
     --cert certs/client.crt \
     --key certs/client.key \
     https://localhost:8443/api/v1/public/info

# 受保护端点
curl --cacert certs/ca.crt \
     --cert certs/client.crt \
     --key certs/client.key \
     https://localhost:8443/api/v1/protected/profile

# 管理员端点（使用普通客户端证书将被拒绝）
curl --cacert certs/ca.crt \
     --cert certs/client.crt \
     --key certs/client.key \
     https://localhost:8443/api/v1/admin/dashboard
```

#### 使用管理员客户端证书测试：

```bash
# 管理员端点（使用管理员证书将成功）
curl --cacert certs/ca.crt \
     --cert certs/admin-client.crt \
     --key certs/admin-client.key \
     https://localhost:8443/api/v1/admin/dashboard
```

#### 不使用客户端证书测试（将失败）：

```bash
# 这将失败，因为 mTLS 需要客户端证书
curl --cacert certs/ca.crt \
     https://localhost:8443/api/v1/public/info
```

## API 端点

### 公开端点（需要 mTLS，无身份检查）

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/public/info` | 服务信息和客户端证书详情 |
| GET | `/api/v1/public/health` | 健康检查端点 |

### 受保护端点（需要 mTLS）

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/protected/profile` | 客户端证书配置文件详情 |
| GET | `/api/v1/protected/data` | 受保护数据访问 |

### 管理员端点（需要管理员证书）

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/admin/dashboard` | 管理员仪表板 |
| GET | `/api/v1/admin/certificates` | 列出所有证书 |

## 配置

### 配置文件 (swit.yaml)

```yaml
http:
  enabled: true
  port: "8443"
  tls:
    enabled: true
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"
    ca_files:
      - "certs/ca.crt"
    client_auth: "require_and_verify"  # 完整 mTLS
    min_version: "TLS1.2"
    max_version: "TLS1.3"
```

### 客户端认证模式

| 模式 | 描述 |
|------|------|
| `none` | 不需要客户端证书 |
| `request` | 请求客户端证书但不强制要求 |
| `require` | 要求任意客户端证书 |
| `verify_if_given` | 如果提供则验证客户端证书 |
| `require_and_verify` | 要求并验证客户端证书（完整 mTLS） |

### 环境变量

```bash
# 服务器配置
export HTTP_PORT="8443"
export GRPC_PORT="50443"
export GRPC_ENABLED="true"

# TLS 配置
export TLS_CERT_FILE="certs/server.crt"
export TLS_KEY_FILE="certs/server.key"
export TLS_CA_FILE="certs/ca.crt"
export TLS_CLIENT_AUTH="require_and_verify"
```

## 证书管理

### 重新生成证书

```bash
# 清理并重新生成所有证书
./scripts/generate-certs.sh clean
./scripts/generate-certs.sh
```

### 验证证书

```bash
# 验证所有证书
./scripts/generate-certs.sh verify
```

### 自定义证书配置

编辑 `scripts/generate-certs.sh` 中的脚本变量：

```bash
# CA 配置
CA_SUBJECT="/C=CN/ST=Beijing/L=Beijing/O=YourOrg/OU=Security/CN=YourOrg-CA"

# 服务器配置
SERVER_SUBJECT="/C=CN/ST=Beijing/L=Beijing/O=YourOrg/OU=Server/CN=your-domain.com"
SERVER_SAN="DNS:your-domain.com,DNS:*.your-domain.com,IP:10.0.0.1"

# 客户端配置
CLIENT_SUBJECT="/C=CN/ST=Beijing/L=Beijing/O=YourOrg/OU=Client/CN=client-name"
```

## 架构

```
┌─────────────┐         ┌──────────────────┐
│   客户端    │◄───────►│  mTLS 服务器     │
│ (带证书)    │  mTLS   │  (Swit 框架)     │
└─────────────┘         └──────────────────┘
       │                        │
       │                        │
       ▼                        ▼
┌─────────────┐         ┌──────────────────┐
│ client.crt  │         │ server.crt       │
│ client.key  │         │ server.key       │
└─────────────┘         │ ca.crt (用于     │
       │                │ 验证客户端)      │
       │                └──────────────────┘
       ▼
┌─────────────┐
│   ca.crt    │
│ (用于验证   │
│   服务器)   │
└─────────────┘
```

### mTLS 握手流程

```
1. 客户端 → 服务器: ClientHello
2. 服务器 → 客户端: ServerHello + 服务器证书
3. 服务器 → 客户端: 证书请求
4. 客户端 → 服务器: 客户端证书
5. 客户端 → 服务器: 证书验证
6. 双方: 根据 CA 验证证书
7. 建立双向认证连接
```

## 安全注意事项

### 生产部署

1. **证书存储**: 安全存储私钥（如 HSM、Vault）
2. **证书轮换**: 实现自动证书轮换
3. **CRL/OCSP**: 实现证书吊销检查
4. **审计日志**: 记录所有基于证书的认证事件
5. **网络安全**: 对 mTLS 服务使用网络分段

### 证书最佳实践

1. **密钥大小**: 至少使用 2048 位 RSA 或 256 位 ECDSA 密钥
2. **有效期**: 使用短期证书（30-90 天）
3. **SAN 使用**: 始终包含主题备用名称
4. **密钥用途**: 设置适当的密钥用途扩展
5. **CA 安全**: 使用硬件安全保护 CA 私钥

### 常见问题

#### 连接被拒绝
- 确保服务器在正确的端口上运行
- 检查防火墙规则

#### 证书验证失败
- 验证证书链是否完整
- 检查证书过期日期
- 确保 CA 证书正确

#### 握手失败
- 检查 TLS 版本兼容性
- 验证密码套件支持
- 确保证书密钥匹配

## 使用 gRPC 测试

### 使用 grpcurl

```bash
# 安装 grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# 列出服务（带 mTLS）
grpcurl -cacert certs/ca.crt \
        -cert certs/client.crt \
        -key certs/client.key \
        localhost:50443 list
```

### 使用 Go 客户端

```go
import (
    "crypto/tls"
    "crypto/x509"
    "os"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

func createMTLSClient() (*grpc.ClientConn, error) {
    // 加载客户端证书
    cert, err := tls.LoadX509KeyPair("certs/client.crt", "certs/client.key")
    if err != nil {
        return nil, err
    }
    
    // 加载 CA 证书
    caCert, err := os.ReadFile("certs/ca.crt")
    if err != nil {
        return nil, err
    }
    caPool := x509.NewCertPool()
    caPool.AppendCertsFromPEM(caCert)
    
    // 创建 TLS 配置
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caPool,
    }
    
    // 创建 gRPC 连接
    return grpc.Dial("localhost:50443",
        grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
}
```

## 延伸阅读

- [TLS 1.3 规范 (RFC 8446)](https://datatracker.ietf.org/doc/html/rfc8446)
- [X.509 证书标准](https://datatracker.ietf.org/doc/html/rfc5280)
- [mTLS 最佳实践](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls/)
- [Go crypto/tls 包](https://pkg.go.dev/crypto/tls)

## 许可证

Copyright 2025 Swit. 保留所有权利。

