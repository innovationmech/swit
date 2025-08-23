# Sentry Integration Testing Workflow

这个 GitHub Action 专门用于测试项目中的 Sentry 错误监控集成。

## 功能特性

### 🔄 自动触发条件
- **Push** 到 `master/main` 分支时
- **Pull Request** 到 `master/main` 分支时  
- 当修改以下文件时：
  - `examples/sentry-example-service/**`
  - `pkg/server/sentry*`
  - `pkg/middleware/sentry*`
  - `.github/workflows/sentry-integration-test.yml`
- **手动触发** (workflow_dispatch)

### 🧪 测试模式

#### 1. 禁用模式 (disabled)
- 不设置 `SENTRY_DSN` 环境变量
- 验证应用在没有 Sentry 的情况下正常工作
- 测试健康检查和基本功能

#### 2. 模拟 DSN 模式 (mock_dsn)  
- 使用虚假的 Sentry DSN
- 测试 Sentry 中间件和错误处理逻辑
- 验证所有端点功能
- 不会发送真实数据到 Sentry

#### 3. 真实 DSN 模式 (real_dsn)
- 使用存储在 GitHub Secrets 中的真实 `SENTRY_DSN`
- 实际发送错误和事件到 Sentry 仪表板
- 只在以下情况下运行：
  - 手动触发且选择了 `test_real_sentry: true`
  - 推送到 `master` 分支

## 设置指南

### 1. 配置 GitHub Secrets

在你的 GitHub 仓库中添加以下 Secret：

1. 进入仓库 Settings → Secrets and variables → Actions
2. 点击 "New repository secret"
3. 添加：
   - **Name**: `SENTRY_DSN`
   - **Value**: `https://your-key@sentry.io/your-project-id`

### 2. 手动运行测试

要测试真实的 Sentry 集成：

1. 进入 Actions 标签页
2. 选择 "Sentry Integration Test" workflow
3. 点击 "Run workflow"
4. 勾选 "Test with real Sentry DSN"
5. 点击 "Run workflow"

### 3. 查看结果

运行后你可以：

- 在 GitHub Actions 中查看详细日志
- 在 Sentry 仪表板中看到实际的错误和事件
- 查看生成的测试摘要报告

## 测试的端点

### 自动测试的 Sentry 功能

#### 错误捕获
- `GET /api/v1/error` - 触发 HTTP 500 错误
- 自动捕获和上报到 Sentry

#### 自定义事件
- `GET /api/v1/custom-sentry` - 发送自定义事件
- 包含自定义标签和上下文

#### 性能监控
- `POST /api/v1/users` - 用户创建流程
- 包含事务追踪和性能指标
- 设置用户上下文信息

#### 忽略的端点
- `GET /health` - 健康检查 (被 Sentry 忽略)
- `GET /api/v1/success` - 成功响应 (不会被捕获)

## 预期的 Sentry 数据

运行真实 DSN 测试后，你应该在 Sentry 仪表板中看到：

### 错误事件
1. **Internal Server Error** 来自 `/api/v1/error`
   - 错误类型：HTTP 500
   - 包含请求上下文 (IP, User-Agent, URL)
   - 标签：`error_type=simulated`

### 自定义事件  
2. **Custom Event** 来自 `/api/v1/custom-sentry`
   - 级别：Warning
   - 标签：`custom_event=true`
   - 自定义上下文数据

### 性能事务
3. **create_user Transaction** 来自 `POST /api/v1/users`
   - 事务名称：`create_user`
   - 包含子 spans：`validate_input`, `save_to_database`
   - 用户上下文：演示用户信息

### 用户上下文
- 用户 ID：`demo-user-123`
- 用户名：`demo_user`
- 邮箱：`demo@example.com`

## 故障排除

### 常见问题

1. **Secret 未配置**
   ```text
   ⚠️ SENTRY_DSN secret not configured, skipping real DSN test
   ```

   解决：按照上述步骤配置 GitHub Secret

2. **服务启动失败**
   - 检查依赖是否正确安装
   - 验证 Go 版本兼容性

3. **端点测试失败**
   - 确保服务已完全启动
   - 检查端口是否冲突

### 查看详细日志

在 Actions 运行页面中：
1. 点击具体的 job
2. 展开相关步骤查看详细输出
3. 查看 "Generate Test Summary" 获取测试摘要

## 自定义配置

你可以修改 workflow 文件来：

1. **调整测试超时时间**：修改 `timeout` 值
2. **添加更多测试端点**：在相应步骤中添加 curl 命令
3. **更改触发条件**：修改 `on` 部分的路径和分支
4. **添加通知**：扩展 `notify-completion` job

## 安全注意事项

- 真实的 SENTRY_DSN 只存储在 GitHub Secrets 中
- 不会在日志中暴露敏感信息
- 模拟模式不会发送任何数据到外部服务
- 测试数据都是演示性质的，不包含真实用户信息

## 贡献指南

如果需要修改这个 workflow：

1. 测试你的更改
2. 更新这个 README
3. 确保所有安全措施仍然有效
4. 在 PR 中说明你的更改
