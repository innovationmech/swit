# ==================================================================================
# SWIT 开发环境管理
# ==================================================================================
# 基于统一的开发环境管理脚本，提供4个核心命令
# 参考: scripts/tools/dev-manage.sh

DEV_SCRIPT := scripts/tools/dev-manage.sh

# ==================================================================================
# 核心Commands (推荐使用)
# ==================================================================================

# 主要开发环境设置（标准设置 - 完整的开发环境）
.PHONY: setup-dev
setup-dev:
	@echo "🚀 标准开发环境设置 - 完整的开发环境（推荐）"
	@$(DEV_SCRIPT)
	@echo ""
	@echo "💡 快速提示："
	@echo "  make setup-quick   # 快速设置（最小必要组件）"
	@echo "  make ci           # CI流水线（测试和检查）"

# 快速开发设置 - 最小必要组件，快速开始开发
.PHONY: setup-quick
setup-quick:
	@echo "🔥 快速开发设置 - 最小必要组件（开发模式）"
	@$(DEV_SCRIPT) --quick

# CI流水线 - 自动化测试和质量检查
.PHONY: ci
ci:
	@echo "🔄 CI流水线 - 自动化测试和质量检查"
	@$(DEV_SCRIPT) --ci

# 高级开发环境管理 - 精确控制特定组件
.PHONY: dev-advanced
dev-advanced:
	@echo "⚙️  高级开发环境管理"
	@if [ -z "$(COMPONENT)" ]; then \
		echo "❌ 错误: 需要指定 COMPONENT 参数"; \
		echo "💡 用法: make dev-advanced COMPONENT=<组件类型>"; \
		echo "📝 支持的组件: hooks, swagger, proto, copyright, build, test, clean, validate"; \
		echo "📖 示例: make dev-advanced COMPONENT=hooks"; \
		exit 1; \
	fi
	@$(DEV_SCRIPT) --advanced $(COMPONENT)

# ==================================================================================
# 内部开发目标 (供其他mk文件调用)
# ==================================================================================

# 内部CI目标（供构建流程调用，不显示提示信息）
.PHONY: ci-internal
ci-internal:
	@$(DEV_SCRIPT) --ci

# 内部验证目标（供其他脚本调用）
.PHONY: dev-validate-internal
dev-validate-internal:
	@$(DEV_SCRIPT) --advanced validate

# ==================================================================================
# 帮助信息
# ==================================================================================

.PHONY: dev-help
dev-help:
	@echo "📋 SWIT 开发环境管理命令 (精简版)"
	@echo ""
	@echo "🎯 核心命令 (推荐使用):"
	@echo "  setup-dev             标准开发环境设置 - 完整的开发环境（推荐）"
	@echo "  setup-quick           快速开发设置 - 最小必要组件（快速开始）"
	@echo "  ci                    CI流水线 - 自动化测试和质量检查"
	@echo "  dev-advanced          高级开发环境管理 - 精确控制特定组件"
	@echo ""
	@echo "⚙️  高级管理示例:"
	@echo "  make dev-advanced COMPONENT=hooks       安装Git钩子"
	@echo "  make dev-advanced COMPONENT=swagger     设置Swagger工具"
	@echo "  make dev-advanced COMPONENT=proto       设置Protobuf工具"
	@echo "  make dev-advanced COMPONENT=copyright   设置版权管理工具"
	@echo "  make dev-advanced COMPONENT=build       设置构建工具"
	@echo "  make dev-advanced COMPONENT=test        设置测试环境"
	@echo "  make dev-advanced COMPONENT=clean       清理开发环境"
	@echo "  make dev-advanced COMPONENT=validate    验证开发环境"
	@echo ""
	@echo "📝 组件说明:"
	@echo "  hooks         Git钩子管理（预提交检查）"
	@echo "  swagger       Swagger工具设置"
	@echo "  proto         Protobuf工具设置"
	@echo "  copyright     版权管理工具"
	@echo "  build         构建工具设置"
	@echo "  test          测试环境设置"
	@echo "  clean         清理开发环境"
	@echo "  validate      验证开发环境"
	@echo ""
	@echo "📖 直接使用脚本:"
	@echo "  $(DEV_SCRIPT) --help"

# ==================================================================================
# 实用功能
# ==================================================================================

# 试运行模式 - 显示将要执行的开发环境命令但不实际执行
.PHONY: dev-dry-run
dev-dry-run:
	@echo "🔍 试运行模式 - 显示开发环境命令但不执行"
	@$(DEV_SCRIPT) --dry-run

# 列出可用的开发环境组件
.PHONY: dev-list
dev-list:
	@echo "📋 列出可用的开发环境组件"
	@$(DEV_SCRIPT) --list 