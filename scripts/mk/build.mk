# 构建相关规则
# 注意: 代码质量相关功能已迁移到 scripts/mk/quality.mk

# =============================================================================
# 核心构建目标 (用户主要使用)
# =============================================================================

# 主要构建命令 - 开发和测试使用
.PHONY: build
build: quality-dev proto swagger
	@echo "🔨 构建项目 (开发模式)"
	@echo "📦 构建当前平台的所有服务..."
	@$(PROJECT_ROOT)/scripts/tools/build-multiplatform.sh -p $$(go env GOOS)/$$(go env GOARCH)

# 生成配置文档
.PHONY: docgen
docgen:
	@echo "📚 生成配置参考文档..."
	@mkdir -p $(OUTPUTDIR)/dist
	@mkdir -p docs/generated
	@$(GO) build -o $(OUTPUTDIR)/dist/swit-docgen ./cmd/swit-docgen
	@$(OUTPUTDIR)/dist/swit-docgen -out docs/generated/configuration-reference.md

# 快速开发构建 - 跳过质量检查，加速迭代
.PHONY: build-dev
build-dev:
	@echo "🚀 快速开发构建（跳过质量检查）"
	@$(PROJECT_ROOT)/scripts/tools/build-multiplatform.sh -p $$(go env GOOS)/$$(go env GOARCH)

# 发布构建 - 构建所有平台的发布版本
.PHONY: build-release
build-release: proto swagger
	@echo "🎯 构建发布版本"
	@echo "📦 构建所有平台并生成发布包..."
	@$(PROJECT_ROOT)/scripts/tools/build-multiplatform.sh --clean --archive --checksum

# 高级构建 - 精确控制服务和平台
.PHONY: build-advanced
build-advanced: proto swagger
	@if [ -z "$(SERVICE)" ] || [ -z "$(PLATFORM)" ]; then \
		echo "❌ 用法: make build-advanced SERVICE=服务名 PLATFORM=平台"; \
		echo ""; \
		echo "📋 示例:"; \
		echo "  make build-advanced SERVICE=swit-serve PLATFORM=linux/amd64"; \
		echo "  make build-advanced SERVICE=switctl PLATFORM=windows/amd64"; \
		echo ""; \
		echo "📋 支持的服务: swit-serve, swit-auth, switctl"; \
		echo "📋 支持的平台: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64"; \
		exit 1; \
	fi
	@echo "📦 构建 $(SERVICE) for $(PLATFORM)..."
	@$(PROJECT_ROOT)/scripts/tools/build-multiplatform.sh -p $(PLATFORM) -s $(SERVICE)

# =============================================================================
# 清理目标 (使用统一的清理系统)
# =============================================================================
# 注意: 清理功能已迁移到 scripts/mk/clean.mk
# 这里保留的目标主要用于内部构建流程

# 内部构建清理目标 (供其他目标使用，不显示用户提示)
.PHONY: clean-build-for-build
clean-build-for-build:
	@echo "🧹 清理构建输出..."
	@$(RM) -rf $(OUTPUTDIR)/
	@echo "构建输出已清理" 