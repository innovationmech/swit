# ==================================================================================
# SWIT 清理管理
# ==================================================================================
# 基于统一的清理管理脚本，提供4个核心命令
# 参考: scripts/tools/clean-manage.sh

CLEAN_SCRIPT := scripts/tools/clean-manage.sh

# ==================================================================================
# 核心Commands (推荐使用)
# ==================================================================================

# 主要清理命令（标准清理 - 所有生成的代码和构建产物）
.PHONY: clean
clean:
	@echo "🧹 标准清理 - 删除所有生成的代码和构建产物"
	@$(CLEAN_SCRIPT)
	@echo ""
	@echo "💡 快速提示："
	@echo "  make clean-dev     # 快速清理（仅构建输出）"
	@echo "  make clean-setup   # 深度清理（重置环境）"

# 快速清理 - 仅删除构建输出（开发时常用）
.PHONY: clean-dev
clean-dev:
	@echo "🔥 快速清理 - 仅删除构建输出（开发模式）"
	@$(CLEAN_SCRIPT) --dev

# 深度清理 - 重置环境（包括缓存和依赖）
.PHONY: clean-setup
clean-setup:
	@echo "🔄 深度清理 - 重置开发环境"
	@$(CLEAN_SCRIPT) --setup

# 高级清理 - 精确控制特定类型
.PHONY: clean-advanced
clean-advanced:
	@echo "⚙️  高级清理操作"
	@if [ -z "$(TYPE)" ]; then \
		echo "❌ 错误: 需要指定 TYPE 参数"; \
		echo "💡 用法: make clean-advanced TYPE=<清理类型>"; \
		echo "📝 支持的类型: build, proto, swagger, test, temp, cache, all"; \
		echo "📖 示例: make clean-advanced TYPE=build"; \
		exit 1; \
	fi
	@$(CLEAN_SCRIPT) --advanced $(TYPE)

# ==================================================================================
# 内部清理目标 (供其他mk文件调用)
# ==================================================================================

# 内部构建清理目标（供build.mk等调用，不显示提示信息）
.PHONY: clean-build-internal
clean-build-internal:
	@$(CLEAN_SCRIPT) --advanced build

# 内部proto清理目标（供proto.mk等调用）
.PHONY: clean-proto-internal
clean-proto-internal:
	@$(CLEAN_SCRIPT) --advanced proto

# 内部swagger清理目标（供swagger.mk等调用）
.PHONY: clean-swagger-internal
clean-swagger-internal:
	@$(CLEAN_SCRIPT) --advanced swagger

# ==================================================================================
# 帮助信息
# ==================================================================================

.PHONY: clean-help
clean-help:
	@echo "📋 SWIT 清理管理命令 (精简版)"
	@echo ""
	@echo "🎯 核心命令 (推荐使用):"
	@echo "  clean                 标准清理 - 删除所有生成的代码和构建产物"
	@echo "  clean-dev             快速清理 - 仅删除构建输出（开发常用）"
	@echo "  clean-setup           深度清理 - 重置环境（包括缓存和依赖）"
	@echo "  clean-advanced        高级清理 - 精确控制特定类型 (需要 TYPE 参数)"
	@echo ""
	@echo "⚙️  高级清理示例:"
	@echo "  make clean-advanced TYPE=build      仅清理构建输出"
	@echo "  make clean-advanced TYPE=proto      仅清理proto生成代码"
	@echo "  make clean-advanced TYPE=swagger    仅清理swagger文档"
	@echo "  make clean-advanced TYPE=test       仅清理测试文件"
	@echo "  make clean-advanced TYPE=temp       仅清理临时文件"
	@echo "  make clean-advanced TYPE=cache      仅清理缓存文件"
	@echo "  make clean-advanced TYPE=all        清理所有类型"
	@echo ""
	@echo "📖 直接使用脚本:"
	@echo "  $(CLEAN_SCRIPT) --help"

# ==================================================================================
# 实用功能
# ==================================================================================

# 列出所有可清理的目标
.PHONY: clean-list
clean-list:
	@echo "📋 列出所有可清理的目标"
	@$(CLEAN_SCRIPT) --list

# 试运行模式 - 显示将要执行的命令但不实际执行
.PHONY: clean-dry-run
clean-dry-run:
	@echo "🔍 试运行模式 - 显示清理命令但不执行"
	@$(CLEAN_SCRIPT) --dry-run 