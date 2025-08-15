# 社区和贡献

欢迎来到 Swit 框架社区！我们是一个开放、包容的开源社区，欢迎所有开发者参与项目的发展和改进。

## 🤝 如何参与

### 贡献方式

1. **框架核心开发** - 改进 `pkg/server/` 和 `pkg/transport/` 组件
2. **示例服务** - 在 `examples/` 目录中添加新的示例
3. **文档改进** - 改进框架文档和指南
4. **测试覆盖** - 为框架组件和示例添加测试
5. **问题反馈** - 报告框架功能问题
6. **功能建议** - 建议新的框架功能

### 参与步骤

1. **Fork 仓库** 并克隆您的 fork
2. **设置开发环境**: `make setup-dev`
3. **运行测试**确保环境正常: `make test`
4. **进行更改** 遵循现有模式
5. **添加测试** 为新功能编写测试
6. **提交 PR** 并提供清晰的描述

## 📋 贡献指南

### 代码贡献

#### 框架核心开发
```bash
# 克隆项目
git clone https://github.com/innovationmech/swit.git
cd swit

# 设置开发环境
make setup-dev

# 创建功能分支
git checkout -b feature/my-new-feature

# 进行开发
vim pkg/server/your-changes.go

# 运行测试
make test

# 提交更改
git commit -m "feat: add new framework feature"
git push origin feature/my-new-feature
```

#### 示例服务开发
```bash
# 创建新示例
mkdir examples/my-example
cd examples/my-example

# 实现示例
vim main.go
vim README.md

# 测试示例
go run main.go

# 添加到构建系统
vim scripts/mk/build.mk  # 如需要
```

### 文档贡献

#### 改进现有文档
```bash
# 编辑文档
vim docs/guide/your-topic.md

# 本地预览（如果设置了文档系统）
make docs-serve

# 提交更改
git add docs/
git commit -m "docs: improve topic documentation"
```

#### 添加新文档
```bash
# 创建新文档
mkdir docs/advanced/
vim docs/advanced/new-topic.md

# 更新导航
vim docs/.vitepress/config.ts
```

### 代码规范

#### Go 代码规范
- 使用 `gofmt` 格式化代码
- 遵循 Go 官方代码风格
- 为公共函数添加文档注释
- 使用有意义的变量和函数名

```go
// UserService 提供用户管理功能
type UserService struct {
    db     *gorm.DB
    logger *zap.Logger
}

// CreateUser 创建新用户
// 参数:
//   - ctx: 请求上下文
//   - user: 用户信息
// 返回:
//   - *User: 创建的用户
//   - error: 错误信息
func (s *UserService) CreateUser(ctx context.Context, user *User) (*User, error) {
    if err := s.validateUser(user); err != nil {
        return nil, fmt.Errorf("用户验证失败: %w", err)
    }
    
    // 实现逻辑...
    return user, nil
}
```

#### 提交信息规范
我们使用 [Conventional Commits](https://www.conventionalcommits.org/zh-hans/) 规范：

```bash
# 功能添加
git commit -m "feat: 添加用户管理 API"

# 问题修复
git commit -m "fix: 修复数据库连接泄漏问题"

# 文档更新
git commit -m "docs: 更新快速开始指南"

# 性能优化
git commit -m "perf: 优化数据库查询性能"

# 重构代码
git commit -m "refactor: 重构传输层架构"

# 测试添加
git commit -m "test: 添加用户服务单元测试"
```

### 测试要求

#### 单元测试
```go
func TestUserService_CreateUser(t *testing.T) {
    tests := []struct {
        name        string
        user        *User
        wantErr     bool
        expectedErr string
    }{
        {
            name: "有效用户",
            user: &User{
                Name:  "张三",
                Email: "zhangsan@example.com",
            },
            wantErr: false,
        },
        {
            name: "无效邮箱",
            user: &User{
                Name:  "李四",
                Email: "invalid-email",
            },
            wantErr:     true,
            expectedErr: "邮箱格式无效",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            service := setupTestService()
            
            result, err := service.CreateUser(context.Background(), tt.user)
            
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedErr)
                assert.Nil(t, result)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
                assert.NotZero(t, result.ID)
            }
        })
    }
}
```

#### 集成测试
```go
func TestHTTPIntegration(t *testing.T) {
    // 设置测试服务器
    config := &server.ServerConfig{
        HTTP: server.HTTPConfig{
            Port:     "0", // 随机端口
            Enabled:  true,
            TestMode: true,
        },
    }
    
    service := &TestService{}
    srv, err := server.NewBusinessServerCore(config, service, nil)
    require.NoError(t, err)
    
    go srv.Start(context.Background())
    defer srv.Shutdown()
    
    // 等待服务器启动
    time.Sleep(100 * time.Millisecond)
    
    // 测试端点
    baseURL := fmt.Sprintf("http://%s", srv.GetHTTPAddress())
    
    t.Run("健康检查", func(t *testing.T) {
        resp, err := http.Get(baseURL + "/health")
        require.NoError(t, err)
        assert.Equal(t, http.StatusOK, resp.StatusCode)
    })
    
    t.Run("用户创建", func(t *testing.T) {
        user := map[string]interface{}{
            "name":  "测试用户",
            "email": "test@example.com",
        }
        
        body, _ := json.Marshal(user)
        resp, err := http.Post(baseURL+"/api/v1/users", "application/json", bytes.NewReader(body))
        
        require.NoError(t, err)
        assert.Equal(t, http.StatusCreated, resp.StatusCode)
    })
}
```

## 🐛 问题报告

### 如何报告 Bug

1. **搜索现有问题** - 先检查是否已有相似问题
2. **使用问题模板** - 填写完整的问题报告模板
3. **提供详细信息** - 包括复现步骤、环境信息、错误日志
4. **添加标签** - 选择适当的标签（bug, enhancement, question 等）

### Bug 报告模板

```markdown
## 问题描述
简要描述遇到的问题

## 复现步骤
1. 执行 `make build`
2. 运行 `./bin/swit-serve`
3. 访问 `http://localhost:9000/api/v1/users`
4. 看到错误...

## 期望行为
描述您期望发生的情况

## 实际行为
描述实际发生的情况

## 环境信息
- 操作系统: macOS 13.0
- Go 版本: 1.24.0
- Swit 版本: v1.0.0
- 数据库: MySQL 8.0

## 错误日志
```
错误日志内容...
```

## 额外信息
任何其他有助于理解问题的信息
```

### 功能请求模板

```markdown
## 功能描述
清晰简洁地描述您想要的功能

## 使用场景
描述这个功能的使用场景和必要性

## 解决方案建议
描述您希望如何实现这个功能

## 替代方案
描述您考虑过的其他解决方案

## 附加信息
任何其他相关信息或截图
```

## 🎯 开发路线图

### 当前版本 (v1.0)
- ✅ 核心服务器框架
- ✅ HTTP 和 gRPC 传输层
- ✅ 依赖注入系统
- ✅ 基本中间件支持
- ✅ 服务发现集成

### 下一版本 (v1.1)
- 🔄 改进的监控和指标
- 🔄 更多中间件选项
- 🔄 配置热重载
- 🔄 分布式追踪支持

### 未来规划 (v2.0)
- 📋 微服务网格集成
- 📋 自动化测试工具
- 📋 性能分析工具
- 📋 云原生部署支持

## 📢 社区资源

### 官方资源
- **GitHub 仓库**: [innovationmech/swit](https://github.com/innovationmech/swit)
- **问题跟踪**: [GitHub Issues](https://github.com/innovationmech/swit/issues)
- **发布说明**: [GitHub Releases](https://github.com/innovationmech/swit/releases)
- **贡献指南**: [CONTRIBUTING.md](https://github.com/innovationmech/swit/blob/master/CONTRIBUTING.md)

### 社区交流
- **讨论区**: [GitHub Discussions](https://github.com/innovationmech/swit/discussions)
- **技术支持**: 通过 GitHub Issues 获取帮助
- **功能建议**: 通过 GitHub Issues 或 Discussions 提交

### 社区规范
我们致力于创建一个开放、包容的社区环境。请阅读并遵守我们的 [行为准则](https://github.com/innovationmech/swit/blob/master/CODE_OF_CONDUCT.md)。

#### 核心价值观
- **包容性** - 欢迎所有背景的开发者
- **尊重** - 尊重不同的观点和经验
- **协作** - 共同努力改进项目
- **学习** - 互相学习和成长
- **专业** - 保持专业和建设性的讨论

### 认可贡献者

我们感谢所有为 Swit 框架做出贡献的开发者。贡献者将在以下地方被认可：

- **README.md** 贡献者部分
- **CONTRIBUTORS.md** 详细贡献记录
- **发布说明** 中的特别感谢
- **GitHub 贡献图** 展示贡献活动

## 🎁 奖励和激励

### 贡献奖励
- **首次贡献者** - 特别徽章和欢迎礼品
- **核心贡献者** - 项目维护者权限
- **文档专家** - 文档团队认证
- **测试冠军** - 质量保证团队认证

### 成长机会
- **技能提升** - 在真实项目中实践 Go 编程
- **开源经验** - 建立开源贡献记录
- **网络建设** - 与其他开发者建立联系
- **职业发展** - 增强简历和专业技能

## 📚 学习资源

### 框架学习
- **官方文档** - 完整的框架文档
- **示例代码** - 实用的代码示例
- **视频教程** - 框架使用教程（规划中）
- **博客文章** - 深度技术文章

### Go 语言学习
- [Go 官方文档](https://golang.org/doc/)
- [Go by Example](https://gobyexample.com/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go 语言圣经](https://books.studygolang.com/gopl-zh/)

### 微服务学习
- [微服务架构](https://microservices.io/)
- [12-Factor App](https://12factor.net/zh_cn/)
- [云原生应用开发](https://www.cncf.io/)
- [gRPC 官方文档](https://grpc.io/docs/)

加入我们，一起构建更好的微服务框架！