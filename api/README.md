# API 目录说明

## 📁 目录结构

```
api/
├── buf.yaml              # Buf 配置
├── buf.gen.yaml          # 代码生成配置
├── buf.lock              # 依赖锁文件
├── proto/swit/v1/        # Proto 文件
│   ├── greeter/         # 问候服务
└── gen/                 # 生成的代码
    ├── go/              # Go 代码
    └── openapiv2/       # OpenAPI 文档
```

## 🛠️ 常用命令

```bash
# 生成代码
make proto-generate

# 检查语法
make proto-lint

# 格式化
make proto-format

# 完整流程
make proto
```

## 🔧 IDE 配置

### VS Code

如果看到 `Cannot resolve import 'google/api/annotations.proto'` 错误，这是正常的。这些依赖通过 Buf 管理，IDE 无法直接解析，但代码生成和构建是正常的。

推荐安装插件：
- **vscode-proto3** - Protocol Buffers 语法支持

### 验证配置

运行以下命令确认一切正常：

```bash
# 检查生成的文件
ls -la api/gen/go/proto/swit/v1/

# 检查 gRPC Gateway 代码
find api/gen/go -name "*.pb.gw.go"
```

如果看到生成的文件，说明配置正确。

## 📝 开发指南

1. **修改 proto 文件**：在 `api/proto/swit/v1/` 目录下修改
2. **生成代码**：运行 `make proto-generate`
3. **检查**：运行 `make proto-lint`
4. **提交**：提交时会自动检查和格式化 