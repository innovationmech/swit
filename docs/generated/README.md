# SWIT 项目 API 文档汇总

本目录提供 SWIT 项目中所有服务的 API 文档统一访问入口。

## 📊 文档概览

| 服务 | 文档位置 | Swagger UI | API Base | 状态 |
|------|----------|------------|----------|------|
| **SwitServe** | [内部文档](../../internal/switserve/docs/) | [UI界面](http://localhost:9000/swagger/index.html) | http://localhost:9000 | ✅ 已完成 |
| **SwitAuth** | [内部文档](../../internal/switauth/docs/) | [UI界面](http://localhost:8090/swagger/index.html) | http://localhost:8090 | ✅ 已完成 |

## 🚀 快速访问

### SwitServe - 用户管理服务
- **源码文档**: [internal/switserve/docs/](../../internal/switserve/docs/)
- **使用指南**: [docs/services/switserve/](../services/switserve/)
- **在线API**: http://localhost:9000/swagger/index.html

### SwitAuth - 认证授权服务  
- **源码文档**: [internal/switauth/docs/](../../internal/switauth/docs/)
- **使用指南**: [docs/services/switauth/](../services/switauth/)
- **在线API**: http://localhost:8090/swagger/index.html

## 🛠 文档管理

### 生成所有服务文档
```bash
make swagger
```

### 生成特定服务文档
```bash
make swagger-switserve    # 仅生成 SwitServe 文档
make swagger-switauth     # 仅生成 SwitAuth 文档
```

### 创建统一访问链接
```bash
make swagger-copy        # 创建本目录下的文档链接
```

### 格式化注释
```bash
make swagger-fmt         # 格式化所有服务的注释
```

## 📁 文档架构说明

```
docs/
├── generated/              # 本目录：统一文档入口
│   ├── README.md          # 本文件：文档汇总导航
│   ├── switserve/         # SwitServe 文档快速访问
│   └── switauth/          # SwitAuth 文档快速访问
├── services/              # 手写服务指南
└── ...                    # 其他项目文档

internal/switserve/docs/   # SwitServe 实际生成文档
├── docs.go               # Go 代码格式文档
├── swagger.json          # JSON 格式 OpenAPI 规范
└── swagger.yaml          # YAML 格式 OpenAPI 规范

internal/switauth/docs/    # SwitAuth 实际生成文档（将来）
├── docs.go
├── swagger.json
└── swagger.yaml
```

## 🔄 文档同步

- **自动生成的文档**在各服务的 `internal/*/docs/` 目录
- **手写的指南文档**在 `docs/services/` 目录  
- **统一访问入口**在 `docs/generated/` 目录

这种架构确保：
1. ✅ 生成的文档与代码版本同步
2. ✅ 手写文档便于维护和更新
3. ✅ 统一入口便于开发者查找
4. ✅ 支持多服务独立文档管理

## 📖 相关文档

- [项目文档首页](../README.md)
- [服务API指南](../services/README.md)
- [OpenAPI集成说明](../openapi-integration.md) 