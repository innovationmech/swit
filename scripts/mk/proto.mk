# Protobuf 相关规则
# 统一使用 scripts/tools/proto-generate.sh 作为后端

# =============================================================================
# 核心Proto目标 (用户主要使用)
# =============================================================================

# 主要proto命令 - 标准代码生成（推荐使用）
.PHONY: proto
proto:
	@echo "🔧 标准proto代码生成（推荐用于开发和发布）"
	@$(PROJECT_ROOT)/scripts/tools/proto-generate.sh
	@echo ""
	@echo "💡 快速提示："
	@echo "  make proto-dev     # 快速开发模式（跳过依赖下载）"
	@echo "  make proto-setup   # 首次环境设置"

# 快速开发模式 - 跳过依赖下载，最快生成速度
.PHONY: proto-dev
proto-dev:
	@echo "🚀 快速proto代码生成（开发模式）"
	@$(PROJECT_ROOT)/scripts/tools/proto-generate.sh --dev

# 环境设置 - 安装工具和下载依赖（首次使用）
.PHONY: proto-setup
proto-setup:
	@echo "⚙️  设置protobuf开发环境"
	@$(PROJECT_ROOT)/scripts/tools/proto-generate.sh --setup

# 高级proto操作 - 支持所有参数的灵活命令
.PHONY: proto-advanced
proto-advanced:
	@echo "⚙️  高级proto操作"
	@if [ -z "$(OPERATION)" ]; then \
		echo "用法: make proto-advanced OPERATION=操作类型"; \
		echo ""; \
		echo "支持的操作:"; \
		echo "  format     - 格式化proto文件"; \
		echo "  lint       - 检查proto语法"; \
		echo "  breaking   - 检查破坏性变更"; \
		echo "  clean      - 清理生成的代码"; \
		echo "  docs       - 生成OpenAPI文档"; \
		echo "  validate   - 验证proto配置"; \
		echo "  dry-run    - 查看命令（试运行）"; \
		echo ""; \
		echo "示例:"; \
		echo "  make proto-advanced OPERATION=format"; \
		echo "  make proto-advanced OPERATION=lint"; \
		echo "  make proto-advanced OPERATION=clean"; \
		echo "  make proto-advanced OPERATION=dry-run"; \
	else \
		case "$(OPERATION)" in \
			format) $(PROJECT_ROOT)/scripts/tools/proto-generate.sh --format ;; \
			lint) $(PROJECT_ROOT)/scripts/tools/proto-generate.sh --lint ;; \
			breaking) $(PROJECT_ROOT)/scripts/tools/proto-generate.sh --breaking ;; \
			clean) $(PROJECT_ROOT)/scripts/tools/proto-generate.sh --clean ;; \
			docs) $(PROJECT_ROOT)/scripts/tools/proto-generate.sh --docs ;; \
			validate) $(PROJECT_ROOT)/scripts/tools/proto-generate.sh --validate ;; \
			dry-run) $(PROJECT_ROOT)/scripts/tools/proto-generate.sh --dry-run ;; \
			*) echo "❌ 未知操作: $(OPERATION)"; echo "运行 'make proto-advanced' 查看支持的操作" ;; \
		esac \
	fi

# =============================================================================
# 内部清理目标 (已迁移到clean.mk)
# =============================================================================
# 注意: proto清理功能已迁移到 scripts/mk/clean.mk
# 这里保留内部目标供构建流程使用
