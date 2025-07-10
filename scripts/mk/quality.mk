# 代码质量管理规则
# 统一管理依赖、格式化、代码检查等质量相关功能

# =============================================================================
# 基础质量操作
# =============================================================================

# 依赖管理 - 整理Go模块依赖
.PHONY: tidy
tidy: proto swagger
	@echo "🔧 整理Go模块依赖..."
	@$(GO) mod tidy
	@echo "✅ Go模块依赖整理完成"

# 代码格式化 - 使用gofmt格式化代码
.PHONY: format
format:
	@echo "🎨 格式化Go代码..."
	@$(GOFMT) -w .
	@echo "✅ 代码格式化完成"

# 代码检查（完整版）- 包含依赖生成
.PHONY: vet
vet: proto swagger
	@echo "🔍 运行代码检查（包含依赖生成）..."
	@$(GOVET) ./...
	@echo "✅ 代码检查完成"

# 代码检查（快速版）- 跳过依赖生成
.PHONY: vet-fast
vet-fast:
	@echo "🔍 运行快速代码检查..."
	@$(GOVET) ./...
	@echo "✅ 快速代码检查完成"

# 代码静态分析 - 使用golint进行代码规范检查
.PHONY: lint
lint:
	@echo "📝 运行代码规范检查..."
	@if command -v golint >/dev/null 2>&1; then \
		golint ./...; \
	else \
		echo "⚠️  golint未安装，跳过代码规范检查"; \
		echo "💡 安装方法: go install golang.org/x/lint/golint@latest"; \
	fi
	@echo "✅ 代码规范检查完成"

# 代码安全检查 - 使用gosec进行安全扫描
.PHONY: security
security:
	@echo "🔒 运行安全扫描..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "⚠️  gosec未安装，跳过安全扫描"; \
		echo "💡 安装方法: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi
	@echo "✅ 安全扫描完成"

# =============================================================================
# 核心质量目标 (用户主要使用)
# =============================================================================

# 标准质量检查（推荐用于CI/CD和发布前）
.PHONY: quality
quality: tidy format vet lint
	@echo "🎯 标准质量检查完成"
	@echo "✅ 包含: 依赖整理 + 代码格式化 + 完整检查 + 规范检查"

# 快速质量检查（开发时使用）
.PHONY: quality-dev
quality-dev: format vet-fast
	@echo "🚀 快速质量检查完成"
	@echo "✅ 包含: 代码格式化 + 快速检查"

# 质量环境设置（安装必要的质量检查工具）
.PHONY: quality-setup
quality-setup:
	@echo "🛠️  设置代码质量检查环境..."
	@echo "📦 检查并安装质量检查工具..."
	
	@echo "检查golint..."
	@if ! command -v golint >/dev/null 2>&1; then \
		echo "📥 安装golint..."; \
		go install golang.org/x/lint/golint@latest; \
	else \
		echo "✅ golint已安装"; \
	fi
	
	@echo "检查gosec..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "📥 安装gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	else \
		echo "✅ gosec已安装"; \
	fi
	
	@echo "检查goimports..."
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "📥 安装goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	else \
		echo "✅ goimports已安装"; \
	fi
	
	@echo "检查staticcheck..."
	@if ! command -v staticcheck >/dev/null 2>&1; then \
		echo "📥 安装staticcheck..."; \
		go install honnef.co/go/tools/cmd/staticcheck@latest; \
	else \
		echo "✅ staticcheck已安装"; \
	fi
	
	@echo "🎉 质量检查环境设置完成"

# 高级质量管理（精确控制特定操作）
.PHONY: quality-advanced
quality-advanced:
	@if [ -z "$(OPERATION)" ]; then \
		echo "🔧 高级质量管理"; \
		echo ""; \
		echo "用法: make quality-advanced OPERATION=<操作> [TARGET=<目标>]"; \
		echo ""; \
		echo "📝 支持的操作:"; \
		echo "  tidy        - 整理Go模块依赖"; \
		echo "  format      - 格式化代码"; \
		echo "  vet         - 代码检查"; \
		echo "  lint        - 代码规范检查"; \
		echo "  security    - 安全扫描"; \
		echo "  imports     - 整理导入语句"; \
		echo "  static      - 静态代码分析"; \
		echo "  all         - 运行所有检查"; \
		echo ""; \
		echo "📖 示例:"; \
		echo "  make quality-advanced OPERATION=tidy"; \
		echo "  make quality-advanced OPERATION=lint TARGET=./internal/..."; \
		echo "  make quality-advanced OPERATION=all"; \
		exit 1; \
	fi
	@case "$(OPERATION)" in \
		tidy) \
			$(MAKE) tidy ;; \
		format) \
			$(MAKE) format ;; \
		vet) \
			$(MAKE) vet ;; \
		lint) \
			$(MAKE) quality-advanced-lint ;; \
		security) \
			$(MAKE) security ;; \
		imports) \
			$(MAKE) quality-advanced-imports ;; \
		static) \
			$(MAKE) quality-advanced-static ;; \
		all) \
			$(MAKE) quality && $(MAKE) security && $(MAKE) quality-advanced-imports && $(MAKE) quality-advanced-static ;; \
		*) \
			echo "❌ 不支持的操作: $(OPERATION)"; \
			$(MAKE) quality-advanced; \
			exit 1 ;; \
	esac

# =============================================================================
# 高级质量操作的具体实现
# =============================================================================

# 高级代码规范检查 - 支持指定目标
.PHONY: quality-advanced-lint
quality-advanced-lint:
	@echo "📝 运行高级代码规范检查..."
	@TARGET=$${TARGET:-./...}; \
	if command -v golint >/dev/null 2>&1; then \
		echo "🔍 检查目标: $$TARGET"; \
		golint $$TARGET; \
	else \
		echo "❌ golint未安装"; \
		echo "💡 请先运行: make quality-setup"; \
		exit 1; \
	fi

# 导入语句整理 - 使用goimports
.PHONY: quality-advanced-imports
quality-advanced-imports:
	@echo "📦 整理导入语句..."
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
		echo "✅ 导入语句整理完成"; \
	else \
		echo "❌ goimports未安装"; \
		echo "💡 请先运行: make quality-setup"; \
		exit 1; \
	fi

# 静态代码分析 - 使用staticcheck
.PHONY: quality-advanced-static
quality-advanced-static:
	@echo "🔬 运行静态代码分析..."
	@TARGET=$${TARGET:-./...}; \
	if command -v staticcheck >/dev/null 2>&1; then \
		echo "🔍 分析目标: $$TARGET"; \
		staticcheck $$TARGET; \
		echo "✅ 静态代码分析完成"; \
	else \
		echo "❌ staticcheck未安装"; \
		echo "💡 请先运行: make quality-setup"; \
		exit 1; \
	fi 