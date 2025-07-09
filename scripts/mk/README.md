# Makefile 拆分说明

## 文件结构

原来的 Makefile 已经按照功能拆分为以下几个文件：

### 1. 主 Makefile
- **位置**: `Makefile`
- **功能**: 导入所有子规则文件，定义主要的组合目标和帮助信息
- **主要内容**:
  - 导入所有子规则
  - 定义 `all` 目标（默认目标）
  - 定义帮助信息
  - 提供 `help` 目标

### 2. 变量定义文件
- **位置**: `scripts/mk/variables.mk`
- **功能**: 定义所有的变量
- **主要内容**:
  - Go 工具链变量
  - 项目构建变量
  - Docker 变量
  - 工具变量

### 3. 构建相关文件
- **位置**: `scripts/mk/build.mk`
- **功能**: 包含所有构建相关的规则
- **主要内容**:
  - `tidy`: 运行 go mod tidy
  - `format`: 代码格式化
  - `vet`: 代码检查
  - `quality`: 综合质量检查
  - `build`: 构建所有二进制文件
  - `build-serve`, `build-ctl`, `build-auth`: 构建单独的二进制文件
  - `clean`: 清理构建产物

### 4. 测试相关文件
- **位置**: `scripts/mk/test.mk`
- **功能**: 包含所有测试相关的规则
- **主要内容**:
  - `test`: 运行所有测试
  - `test-pkg`: 运行 pkg 包测试
  - `test-internal`: 运行 internal 包测试
  - `test-coverage`: 运行覆盖率测试
  - `test-race`: 运行竞态检测测试

### 5. Docker 相关文件
- **位置**: `scripts/mk/docker.mk`
- **功能**: 包含所有 Docker 相关的规则
- **主要内容**:
  - `image-serve`: 构建 swit-serve Docker 镜像
  - `image-auth`: 构建 swit-auth Docker 镜像
  - `image-all`: 构建所有 Docker 镜像

### 6. Swagger 相关文件
- **位置**: `scripts/mk/swagger.mk`
- **功能**: 包含所有 Swagger 文档相关的规则
- **主要内容**:
  - `swagger-install`: 安装 swag 工具
  - `swagger`: 生成所有 Swagger 文档
  - `swagger-switserve`: 生成 switserve 文档
  - `swagger-switauth`: 生成 switauth 文档
  - `swagger-fmt`: 格式化 Swagger 注释
  - `swagger-copy`: 创建统一的文档访问链接

### 7. 开发环境相关文件
- **位置**: `scripts/mk/dev.mk`
- **功能**: 包含开发环境设置和 CI 相关的规则
- **主要内容**:
  - `install-hooks`: 安装 Git 钩子
  - `setup-dev`: 设置开发环境
  - `ci`: 运行 CI 流水线

### 8. 版权相关文件
- **位置**: `scripts/mk/copyright.mk`
- **功能**: 包含版权检查和添加相关的规则
- **主要内容**:
  - `copyright-check`: 检查版权声明（基于哈希比较）
  - `copyright-add`: 为缺少版权声明的文件添加版权声明
  - `copyright-update`: 更新过期的版权声明
  - `copyright-force`: 强制更新所有文件的版权声明
  - `copyright`: 综合版权处理（检查、添加、更新）
  - `copyright-debug`: 调试哈希比较过程（故障排除）

#### 智能版权检测机制
- **哈希比较**: 使用 SHA-256 哈希值比较版权声明的完整内容
- **通用检测**: 能检测版权声明的任何变化（年份、作者、许可证文本等）
- **精确匹配**: 确保版权声明与 `boilerplate.txt` 完全一致

## 工具脚本

### 9. 工具脚本目录
- **位置**: `scripts/tools/`
- **功能**: 包含项目所需的工具脚本
- **脚本列表**:
  - `pre-commit.sh`: Git pre-commit 钩子脚本，运行代码质量检查
  - `update-docs.sh`: 文档更新脚本，用于生成和更新API文档

#### pre-commit.sh
自动化的代码质量检查脚本，在提交代码前运行：
- 检查 Go 环境
- 运行 go mod tidy
- 格式化代码
- 运行 go vet
- 运行相关包的测试

#### update-docs.sh
文档更新自动化脚本：
- 生成 Swagger API 文档
- 创建统一文档访问链接
- 提供文档访问指引

## 使用方法

拆分后的使用方法与原来完全相同，所有的命令都保持不变：

```bash
# 查看帮助
make help

# 构建项目
make build

# 运行测试
make test

# 生成 Swagger 文档
make swagger

# 构建 Docker 镜像
make image-all

# 运行 CI 流水线
make ci

# 设置开发环境
make setup-dev

# 版权管理
make copyright              # 检查并更新版权声明
make copyright-check        # 仅检查版权声明
make copyright-update       # 更新过期的版权声明
make copyright-force        # 强制更新所有文件的版权声明
make copyright-debug        # 调试版权检测过程（故障排除）

# 手动更新文档
bash scripts/tools/update-docs.sh
```

## 优势

1. **模块化**: 每个文件专注于特定的功能领域
2. **可维护性**: 易于理解和修改特定功能的规则
3. **可扩展性**: 可以轻松添加新的功能文件
4. **重用性**: 可以在其他项目中重用特定的规则文件
5. **可读性**: 文件结构清晰，便于理解和维护
6. **工具化**: 独立的工具脚本便于单独调用和维护

## 注意事项

1. 所有的变量都在 `variables.mk` 中定义，如果需要修改变量，请在该文件中进行
2. 如果需要添加新的规则，请将其添加到相应的功能文件中
3. 主 Makefile 主要用于导入子规则和定义组合目标，不要在其中添加具体的规则
4. 确保所有子规则文件都在主 Makefile 中正确导入
5. 工具脚本位于 `scripts/tools/` 目录，修改脚本位置时需要同步更新相关引用
6. Git hooks 会自动从 `scripts/tools/pre-commit.sh` 安装，确保该脚本有执行权限 