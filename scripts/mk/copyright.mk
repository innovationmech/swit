# ==================================================================================
# SWIT Copyright管理
# ==================================================================================
# 基于统一的copyright管理脚本，提供4个核心命令
# 参考: scripts/tools/copyright-manage.sh

COPYRIGHT_SCRIPT := scripts/tools/copyright-manage.sh

# ==================================================================================
# 核心Commands (推荐使用)
# ==================================================================================

# 主要版权管理命令（检查+自动修复）
.PHONY: copyright
copyright:
	@echo "🔧 版权声明管理 (检查+自动修复)"
	@$(COPYRIGHT_SCRIPT)

# 只检查版权声明，不修改文件
.PHONY: copyright-check
copyright-check:
	@echo "🔍 检查版权声明"
	@$(COPYRIGHT_SCRIPT) --check

# 初始版权设置（为新项目添加版权声明）
.PHONY: copyright-setup
copyright-setup:
	@echo "🔧 初始版权声明设置"
	@$(COPYRIGHT_SCRIPT) --setup

# 高级版权操作
.PHONY: copyright-advanced
copyright-advanced:
	@echo "⚙️  高级版权操作: $(OPERATION)"
	@if [ -z "$(OPERATION)" ]; then \
		echo "❌ 错误: 需要指定 OPERATION 参数"; \
		echo "💡 用法: make copyright-advanced OPERATION=<操作类型>"; \
		echo "📝 支持的操作: force, debug, files, validate"; \
		echo "📖 示例: make copyright-advanced OPERATION=debug"; \
		exit 1; \
	fi
	@$(COPYRIGHT_SCRIPT) --advanced $(OPERATION)

# ==================================================================================
# 帮助信息
# ==================================================================================

.PHONY: copyright-help
copyright-help:
	@echo "📋 SWIT Copyright管理命令 (精简版)"
	@echo ""
	@echo "🎯 核心命令 (推荐使用):"
	@echo "  copyright               版权声明管理 (检查+自动修复)"
	@echo "  copyright-check         只检查版权声明"
	@echo "  copyright-setup         初始版权设置"
	@echo "  copyright-advanced      高级操作 (需要 OPERATION 参数)"
	@echo ""
	@echo "⚙️  高级操作示例:"
	@echo "  make copyright-advanced OPERATION=force     强制更新所有文件"
	@echo "  make copyright-advanced OPERATION=debug     调试版权检测"
	@echo "  make copyright-advanced OPERATION=files     显示文件信息"
	@echo "  make copyright-advanced OPERATION=validate  验证配置"
	@echo ""
	@echo "📖 直接使用脚本:"
	@echo "  $(COPYRIGHT_SCRIPT) --help"
