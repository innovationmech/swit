# ==================================================================================
# SWIT 测试管理
# ==================================================================================
# 基于统一的测试管理脚本，提供4个核心命令
# 参考: scripts/tools/test-manage.sh

TEST_SCRIPT := scripts/tools/test-manage.sh

# ==================================================================================
# 核心Commands (推荐使用)
# ==================================================================================

# 主要测试命令（标准测试 - 生成依赖+运行所有测试）
.PHONY: test
test:
	@echo "🧪 标准测试 - 运行所有测试（包含依赖生成）"
	@$(TEST_SCRIPT)
	@echo ""
	@echo "💡 快速提示："
	@echo "  make test-dev      # 快速开发测试（跳过依赖生成）"
	@echo "  make test-coverage # 覆盖率测试（生成报告）"

# 快速开发测试 - 跳过依赖生成，直接测试
.PHONY: test-dev
test-dev:
	@echo "🔥 快速开发测试 - 跳过依赖生成（开发模式）"
	@if [ -f "$(TEST_SCRIPT)" ]; then \
		$(TEST_SCRIPT) --dev; \
	else \
		echo "❌ 测试脚本未找到: $(TEST_SCRIPT)"; \
		exit 1; \
	fi

# 覆盖率测试 - 生成详细的覆盖率报告
.PHONY: test-coverage
test-coverage:
	@echo "📊 覆盖率测试 - 生成详细的覆盖率报告"
	@$(TEST_SCRIPT) --coverage

# 高级测试 - 精确控制测试类型和包范围
.PHONY: test-advanced
test-advanced:
	@echo "⚙️  高级测试操作"
	@if [ -z "$(TYPE)" ]; then \
		echo "❌ 错误: 需要指定 TYPE 参数"; \
		echo "💡 用法: make test-advanced TYPE=<测试类型> [PACKAGE=<包类型>]"; \
		echo "📝 测试类型: unit, race, bench, short"; \
		echo "📝 包类型: all, internal, pkg (默认: all)"; \
		echo "📖 示例: make test-advanced TYPE=race PACKAGE=internal"; \
		exit 1; \
	fi
	@if [ -n "$(PACKAGE)" ]; then \
		$(TEST_SCRIPT) --advanced --type $(TYPE) --package $(PACKAGE); \
	else \
		$(TEST_SCRIPT) --advanced --type $(TYPE) --package all; \
	fi

# ==================================================================================
# 向后兼容Commands (显示迁移建议)
# ==================================================================================

# 传统命令的兼容性支持，引导用户使用新的核心命令
.PHONY: test-pkg
test-pkg:
	@echo "ℹ️  注意: test-pkg 已整合到核心命令中"
	@echo "💡 推荐使用: make test（运行所有测试）或 make test-advanced TYPE=unit PACKAGE=pkg"
	@$(TEST_SCRIPT) --advanced --type unit --package pkg

.PHONY: test-internal
test-internal:
	@echo "ℹ️  注意: test-internal 已整合到核心命令中"
	@echo "💡 推荐使用: make test（运行所有测试）或 make test-advanced TYPE=unit PACKAGE=internal"
	@$(TEST_SCRIPT) --advanced --type unit --package internal

.PHONY: test-race
test-race:
	@echo "ℹ️  注意: test-race 已整合到高级命令中"
	@echo "💡 推荐使用: make test-advanced TYPE=race 或 make test-advanced TYPE=race PACKAGE=all"
	@$(TEST_SCRIPT) --advanced --type race --package all

# ==================================================================================
# 内部测试目标 (供其他mk文件调用)
# ==================================================================================

# 内部测试目标（供CI等调用，不显示提示信息）
.PHONY: test-internal-only
test-internal-only:
	@$(TEST_SCRIPT) --dev --skip-deps

# 内部快速测试目标（供构建流程调用）
.PHONY: test-quick-internal
test-quick-internal:
	@$(TEST_SCRIPT) --advanced --type short --package all --skip-deps

# ==================================================================================
# 帮助信息
# ==================================================================================

.PHONY: test-help
test-help:
	@echo "📋 SWIT 测试管理命令 (精简版)"
	@echo ""
	@echo "🎯 核心命令 (推荐使用):"
	@echo "  test                  标准测试 - 运行所有测试（包含依赖生成）"
	@echo "  test-dev              快速开发测试 - 跳过依赖生成（开发常用）"
	@echo "  test-coverage         覆盖率测试 - 生成详细的覆盖率报告"
	@echo "  test-advanced         高级测试 - 精确控制测试类型和包范围"
	@echo ""
	@echo "⚙️  高级测试示例:"
	@echo "  make test-advanced TYPE=unit                     运行单元测试（所有包）"
	@echo "  make test-advanced TYPE=race                     运行竞态检测（所有包）"
	@echo "  make test-advanced TYPE=unit PACKAGE=internal    运行内部包单元测试"
	@echo "  make test-advanced TYPE=race PACKAGE=pkg         运行公共包竞态检测"
	@echo "  make test-advanced TYPE=bench PACKAGE=all        运行性能基准测试"
	@echo "  make test-advanced TYPE=short                    运行快速测试"
	@echo ""
	@echo "📝 测试类型说明:"
	@echo "  unit      标准单元测试"
	@echo "  race      竞态检测测试"
	@echo "  bench     性能基准测试"
	@echo "  short     快速测试（跳过耗时测试）"
	@echo ""
	@echo "📦 包类型说明:"
	@echo "  all       所有包 (internal + pkg)"
	@echo "  internal  仅内部包"
	@echo "  pkg       仅公共包"
	@echo ""
	@echo "🔄 传统命令 (向后兼容):"
	@echo "  test-pkg              → 推荐使用 test 或 test-advanced TYPE=unit PACKAGE=pkg"
	@echo "  test-internal         → 推荐使用 test 或 test-advanced TYPE=unit PACKAGE=internal"
	@echo "  test-race             → 推荐使用 test-advanced TYPE=race"
	@echo ""
	@echo "📖 直接使用脚本:"
	@echo "  $(TEST_SCRIPT) --help"

# ==================================================================================
# 实用功能
# ==================================================================================

# 试运行模式 - 显示将要执行的测试命令但不实际执行
.PHONY: test-dry-run
test-dry-run:
	@echo "🔍 试运行模式 - 显示测试命令但不执行"
	@$(TEST_SCRIPT) --dry-run 