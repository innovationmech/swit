# ==================================================================================
# SWIT Swagger文档管理
# ==================================================================================
# 基于统一的swagger管理脚本，提供4个核心命令
# 参考: scripts/tools/swagger-manage.sh

SWAGGER_SCRIPT := scripts/tools/swagger-manage.sh

# ==================================================================================
# 核心Commands (推荐使用)
# ==================================================================================

# 主要swagger文档生成（格式化+生成+整理）
.PHONY: swagger
swagger:
	@echo "📚 标准swagger文档生成（推荐用于开发和发布）"
	@$(SWAGGER_SCRIPT)
	@echo ""
	@echo "💡 快速提示："
	@echo "  make swagger-dev     # 快速开发模式（跳过格式化）"
	@echo "  make swagger-setup   # 首次环境设置"

# 快速开发模式 - 跳过格式化，加速开发迭代
.PHONY: swagger-dev
swagger-dev:
	@echo "🚀 快速swagger文档生成（开发模式）"
	@$(SWAGGER_SCRIPT) --dev

# 环境设置 - 安装swag工具（首次使用）
.PHONY: swagger-setup
swagger-setup:
	@echo "⚙️  设置swagger开发环境"
	@$(SWAGGER_SCRIPT) --setup

# 高级swagger操作 - 支持所有参数的灵活命令
.PHONY: swagger-advanced
swagger-advanced:
	@echo "⚙️  高级swagger操作"
	@if [ -z "$(OPERATION)" ]; then \
		echo "❌ 错误: 需要指定 OPERATION 参数"; \
		echo "💡 用法: make swagger-advanced OPERATION=<操作类型>"; \
		echo "📝 支持的操作: format, switserve, switauth, copy, clean, validate"; \
		echo "📖 示例: make swagger-advanced OPERATION=format"; \
		exit 1; \
	fi
	@$(SWAGGER_SCRIPT) --advanced $(OPERATION)

# ==================================================================================
# 内部清理目标 (已迁移到clean.mk)
# ==================================================================================
# 注意: swagger清理功能已迁移到 scripts/mk/clean.mk

# ==================================================================================
# 帮助信息
# ==================================================================================

.PHONY: swagger-help
swagger-help:
	@echo "📋 SWIT Swagger文档管理命令 (精简版)"
	@echo ""
	@echo "🎯 核心命令 (推荐使用):"
	@echo "  swagger               标准swagger文档生成 (格式化+生成+整理)"
	@echo "  swagger-dev           快速开发模式 (跳过格式化，加速迭代)"
	@echo "  swagger-setup         环境设置 (安装swag工具)"
	@echo "  swagger-advanced      高级操作 (需要 OPERATION 参数)"
	@echo ""
	@echo "⚙️  高级操作示例:"
	@echo "  make swagger-advanced OPERATION=format      格式化swagger注释"
	@echo "  make swagger-advanced OPERATION=switserve   只生成switserve文档"
	@echo "  make swagger-advanced OPERATION=switauth    只生成switauth文档"
	@echo "  make swagger-advanced OPERATION=copy        整理文档位置"
	@echo "  make swagger-advanced OPERATION=clean       清理生成文档"
	@echo "  make swagger-advanced OPERATION=validate    验证swagger配置"
	@echo ""
	@echo "📖 直接使用脚本:"
	@echo "  $(SWAGGER_SCRIPT) --help" 