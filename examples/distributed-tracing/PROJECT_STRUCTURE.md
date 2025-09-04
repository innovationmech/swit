# 分布式追踪示例项目结构文档

## 项目概览

本文档详细说明了分布式追踪示例项目的完整目录结构、各组件职责和相互关系。该项目展示了如何在 SWIT 微服务框架中集成 OpenTelemetry 分布式追踪。

## 完整目录结构

```bash
examples/distributed-tracing/
├── README.md                          # 项目概览和快速开始指南
├── PROJECT_STRUCTURE.md               # 项目结构文档（本文件）
├── docker-compose.yml                 # Docker 编排配置
├── .env.example                       # 环境变量示例
├── .gitignore                         # Git 忽略文件配置
│
├── jaeger/                            # Jaeger 追踪后端配置
│   ├── README.md                      # Jaeger 配置说明
│   ├── config/                        # Jaeger 配置文件
│   │   ├── jaeger.yml                # Jaeger All-in-One 配置
│   │   ├── sampling.json             # 采样策略配置
│   │   └── storage.yml               # 存储配置（生产环境）
│   └── docker-compose.jaeger.yml     # 独立 Jaeger 部署配置
│
├── services/                          # 微服务实现
│   ├── README.md                      # 服务架构总览
│   │
│   ├── order-service/                 # 订单服务
│   │   ├── README.md                 # 服务说明文档
│   │   ├── main.go                   # 服务入口点
│   │   ├── Dockerfile                # Docker 镜像构建
│   │   ├── go.mod                    # Go 模块依赖
│   │   ├── go.sum                    # 依赖校验文件
│   │   ├── config/                   # 配置文件
│   │   │   └── config.yaml          # 服务配置
│   │   ├── internal/                 # 内部实现
│   │   │   ├── handler/             # HTTP 处理器
│   │   │   ├── service/             # gRPC 服务实现
│   │   │   ├── model/               # 数据模型
│   │   │   ├── repository/          # 数据访问层
│   │   │   └── config/              # 配置结构
│   │   ├── proto/                    # Protocol Buffer 定义
│   │   │   └── order.proto          # 订单服务接口定义
│   │   ├── migrations/               # 数据库迁移脚本
│   │   │   └── 001_initial.sql      # 初始化脚本
│   │   └── tests/                    # 测试文件
│   │       ├── integration/         # 集成测试
│   │       └── unit/                # 单元测试
│   │
│   ├── payment-service/               # 支付服务
│   │   ├── README.md                 # 服务说明文档
│   │   ├── main.go                   # 服务入口点
│   │   ├── Dockerfile                # Docker 镜像构建
│   │   ├── go.mod                    # Go 模块依赖
│   │   ├── go.sum                    # 依赖校验文件
│   │   ├── config/                   # 配置文件
│   │   │   └── config.yaml          # 服务配置
│   │   ├── internal/                 # 内部实现
│   │   │   ├── service/             # gRPC 服务实现
│   │   │   ├── model/               # 数据模型
│   │   │   ├── client/              # 第三方 API 客户端
│   │   │   └── config/              # 配置结构
│   │   ├── proto/                    # Protocol Buffer 定义
│   │   │   └── payment.proto        # 支付服务接口定义
│   │   └── tests/                    # 测试文件
│   │       ├── integration/         # 集成测试
│   │       └── unit/                # 单元测试
│   │
│   └── inventory-service/             # 库存服务
│       ├── README.md                 # 服务说明文档
│       ├── main.go                   # 服务入口点
│       ├── Dockerfile                # Docker 镜像构建
│       ├── go.mod                    # Go 模块依赖
│       ├── go.sum                    # 依赖校验文件
│       ├── config/                   # 配置文件
│       │   └── config.yaml          # 服务配置
│       ├── internal/                 # 内部实现
│       │   ├── handler/             # HTTP 处理器
│       │   ├── service/             # gRPC 服务实现
│       │   ├── model/               # 数据模型
│       │   ├── repository/          # 数据访问层
│       │   └── config/              # 配置结构
│       ├── proto/                    # Protocol Buffer 定义
│       │   └── inventory.proto      # 库存服务接口定义
│       ├── migrations/               # 数据库迁移脚本
│       │   └── 001_initial.sql      # 初始化脚本
│       └── tests/                    # 测试文件
│           ├── integration/         # 集成测试
│           └── unit/                # 单元测试
│
├── scripts/                           # 自动化脚本
│   ├── README.md                      # 脚本使用说明
│   ├── setup.sh                      # 环境搭建脚本
│   ├── health-check.sh               # 健康检查脚本
│   ├── cleanup.sh                    # 环境清理脚本
│   ├── load-test.sh                  # 压力测试脚本
│   ├── demo-scenarios.sh             # 演示场景脚本
│   ├── trace-analysis.sh             # 追踪分析脚本
│   ├── dev-start.sh                  # 开发环境启动
│   ├── logs.sh                       # 日志查看脚本
│   └── config/                       # 脚本配置
│       ├── demo-config.env          # 演示环境配置
│       └── test-data.json           # 测试数据模板
│
└── docs/                              # 详细文档
    ├── README.md                      # 文档目录说明
    ├── user-guide.md                  # 用户使用指南
    ├── developer-guide.md             # 开发者指南
    ├── operations-guide.md            # 运维指南
    ├── architecture.md                # 架构设计文档
    ├── troubleshooting.md             # 故障排查手册
    ├── api-reference.md               # API 接口文档
    ├── performance-guide.md           # 性能优化指南
    └── best-practices.md              # 最佳实践指南
```

## 目录职责说明

### 根目录文件

| 文件 | 职责 | 状态 |
|------|------|------|
| `README.md` | 项目概览、快速开始指南 | ✅ 已完成 |
| `PROJECT_STRUCTURE.md` | 项目结构详细说明 | ✅ 已完成 |
| `docker-compose.yml` | Docker 服务编排配置 | ✅ 已完成 |
| `.env.example` | 环境变量配置示例 | 📋 待创建 |
| `.gitignore` | Git 忽略文件配置 | 📋 待创建 |

### Jaeger 配置 (`jaeger/`)

| 组件 | 职责 | 状态 |
|------|------|------|
| `README.md` | Jaeger 配置和使用说明 | ✅ 已完成 |
| `config/jaeger.yml` | Jaeger All-in-One 配置 | ✅ 已完成 |
| `config/sampling.json` | 采样策略配置 | ✅ 已完成 |
| `config/storage.yml` | 存储后端配置（生产环境） | 📋 待创建 |

### 微服务实现 (`services/`)

#### 订单服务 (`order-service/`)
- **职责**: 订单管理、服务协调
- **端口**: HTTP 8081, gRPC 9081
- **特点**: HTTP + gRPC 双协议支持
- **状态**: 📋 框架已搭建，实现待完成

#### 支付服务 (`payment-service/`)
- **职责**: 支付处理、第三方 API 模拟
- **端口**: gRPC 9082
- **特点**: 纯 gRPC 服务
- **状态**: 📋 框架已搭建，实现待完成

#### 库存服务 (`inventory-service/`)
- **职责**: 库存管理、数据库操作
- **端口**: HTTP 8083, gRPC 9083
- **特点**: HTTP + gRPC 双协议，包含数据库操作
- **状态**: 📋 框架已搭建，实现待完成

### 自动化脚本 (`scripts/`)

| 脚本 | 职责 | 状态 |
|------|------|------|
| `setup.sh` | 一键环境搭建 | 📋 待实现 |
| `health-check.sh` | 服务健康状态检查 | 📋 待实现 |
| `cleanup.sh` | 环境清理和资源释放 | 📋 待实现 |
| `load-test.sh` | 负载测试和性能验证 | 📋 待实现 |
| `demo-scenarios.sh` | 演示场景自动化 | 📋 待实现 |
| `trace-analysis.sh` | 追踪数据分析 | 📋 待实现 |

### 文档系统 (`docs/`)

| 文档 | 目标读者 | 状态 |
|------|----------|------|
| `user-guide.md` | 系统用户、产品经理 | 📋 待编写 |
| `developer-guide.md` | 开发者、架构师 | 📋 待编写 |
| `operations-guide.md` | 运维工程师、SRE | 📋 待编写 |
| `architecture.md` | 架构师、技术负责人 | 📋 待编写 |
| `troubleshooting.md` | 开发、运维、技术支持 | 📋 待编写 |
| `api-reference.md` | 前端开发者、接口调用方 | 📋 待编写 |

## 技术架构层次

### 第一层：基础设施
- **Docker 容器化**: 统一的部署和运行环境
- **Docker Compose**: 服务编排和网络配置
- **Jaeger**: 分布式追踪后端存储和查询

### 第二层：微服务层
- **订单服务**: 业务协调中心
- **支付服务**: 外部服务集成示例
- **库存服务**: 数据库操作示例

### 第三层：追踪层
- **OpenTelemetry SDK**: 追踪数据采集
- **自动instrumentation**: HTTP/gRPC/数据库自动追踪
- **手动instrumentation**: 业务逻辑追踪

### 第四层：工具层
- **自动化脚本**: 环境管理和测试
- **监控工具**: 健康检查和性能监控
- **分析工具**: 追踪数据分析和报告

## 数据流向

### 正常业务流程
```text
客户端请求 → 订单服务 → 库存服务（检查库存）
                ↓
        支付服务（处理支付） ← 订单服务
                ↓
        库存服务（预留库存） ← 订单服务
                ↓
            返回订单结果
```

### 追踪数据流向
```text
各服务 Span → OpenTelemetry Collector → Jaeger Collector → 
存储后端（内存/Elasticsearch） → Jaeger Query → Jaeger UI
```

## 开发阶段规划

### Phase 1: 基础框架（当前阶段）✅
- [x] 目录结构搭建
- [x] README 文档编写
- [x] Docker Compose 配置
- [x] Jaeger 基础配置

### Phase 2: 服务实现 📋
- [ ] 订单服务完整实现
- [ ] 支付服务完整实现
- [ ] 库存服务完整实现
- [ ] Protocol Buffer 定义

### Phase 3: 追踪集成 📋
- [ ] OpenTelemetry SDK 集成
- [ ] 自动追踪配置
- [ ] 手动追踪埋点
- [ ] 追踪属性定义

### Phase 4: 自动化工具 📋
- [ ] 环境管理脚本实现
- [ ] 测试脚本实现
- [ ] 监控脚本实现
- [ ] 分析工具实现

### Phase 5: 文档完善 📋
- [ ] 用户指南编写
- [ ] 开发者指南编写
- [ ] 运维指南编写
- [ ] API 文档编写

### Phase 6: 优化和扩展 📋
- [ ] 性能优化
- [ ] 更多演示场景
- [ ] 高级功能展示
- [ ] 最佳实践总结

## 配置管理

### 环境变量
- 开发环境: `.env.development`
- 测试环境: `.env.testing`
- 生产环境: `.env.production`

### 配置文件
- 服务配置: `services/*/config/config.yaml`
- Jaeger 配置: `jaeger/config/*.yml`
- 脚本配置: `scripts/config/*.env`

## 数据持久化

### 开发环境
- SQLite 数据库文件
- Docker volumes 挂载
- 内存存储（Jaeger）

### 生产环境建议
- PostgreSQL/MySQL 数据库
- Elasticsearch 追踪存储
- 持久化存储卷

## 网络架构

### Docker 网络
- 网络名称: `tracing-network`
- 子网: `172.20.0.0/16`
- 驱动: `bridge`

### 端口分配
- Jaeger UI: 16686
- 订单服务: 8081 (HTTP), 9081 (gRPC)
- 支付服务: 9082 (gRPC)
- 库存服务: 8083 (HTTP), 9083 (gRPC)

## 监控和观测

### 健康检查
- HTTP 端点: `/health`
- gRPC 健康检查协议
- Docker 健康检查配置

### 指标收集
- 业务指标: 订单数、支付成功率、库存周转
- 技术指标: 响应时间、错误率、吞吐量
- 系统指标: CPU、内存、网络

### 日志管理
- 结构化日志: JSON 格式
- 日志级别: DEBUG, INFO, WARN, ERROR
- 日志聚合: 容器日志收集

## 安全考虑

### 网络安全
- 内部网络隔离
- 端口最小化暴露
- 容器间通信加密（生产环境）

### 数据安全
- 敏感信息脱敏
- 追踪数据过滤
- 配置文件加密

### 访问控制
- Jaeger UI 访问控制（生产环境）
- 服务间认证（生产环境）
- API 访问限制

## 扩展性设计

### 水平扩展
- 服务实例负载均衡
- 数据库读写分离
- 追踪数据分片存储

### 功能扩展
- 新服务集成模板
- 追踪插件机制
- 监控指标扩展

### 部署扩展
- Kubernetes 部署支持
- 云原生架构适配
- CI/CD 流水线集成

---

*本文档将随项目开发进展持续更新。*
