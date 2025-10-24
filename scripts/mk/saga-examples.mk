# ==================================================================================
# SWIT Saga 示例管理
# ==================================================================================
# 提供便捷的 Saga 示例运行、监控和管理命令
# 参考: pkg/saga/examples/scripts/

SAGA_EXAMPLES_DIR := pkg/saga/examples
SAGA_SCRIPTS_DIR := $(SAGA_EXAMPLES_DIR)/scripts

# ==================================================================================
# 核心 Commands (推荐使用)
# ==================================================================================

# 运行 Saga 示例
.PHONY: saga-examples-run
saga-examples-run:
	@echo "🚀 运行 Saga 示例"
	@if [ -z "$(EXAMPLE)" ]; then \
		echo "❌ 错误: 需要指定 EXAMPLE 参数"; \
		echo "💡 用法: make saga-examples-run EXAMPLE=<示例名称>"; \
		echo "📝 可用示例: order, payment, inventory, user, e2e, all"; \
		echo "📖 示例: make saga-examples-run EXAMPLE=order"; \
		exit 1; \
	fi
	@if [ -f "$(SAGA_SCRIPTS_DIR)/run.sh" ]; then \
		cd $(SAGA_SCRIPTS_DIR) && ./run.sh $(EXAMPLE); \
	else \
		echo "❌ 运行脚本未找到: $(SAGA_SCRIPTS_DIR)/run.sh"; \
		exit 1; \
	fi

# 查看 Saga 示例状态
.PHONY: saga-examples-status
saga-examples-status:
	@echo "📊 查看 Saga 示例状态"
	@if [ -f "$(SAGA_SCRIPTS_DIR)/status.sh" ]; then \
		cd $(SAGA_SCRIPTS_DIR) && ./status.sh; \
	else \
		echo "❌ 状态脚本未找到: $(SAGA_SCRIPTS_DIR)/status.sh"; \
		exit 1; \
	fi

# 查看 Saga 示例日志
.PHONY: saga-examples-logs
saga-examples-logs:
	@echo "📋 查看 Saga 示例日志"
	@if [ -f "$(SAGA_SCRIPTS_DIR)/logs.sh" ]; then \
		cd $(SAGA_SCRIPTS_DIR) && ./logs.sh $(if $(TYPE),$(TYPE),all); \
	else \
		echo "❌ 日志脚本未找到: $(SAGA_SCRIPTS_DIR)/logs.sh"; \
		exit 1; \
	fi

# 停止和清理 Saga 示例
.PHONY: saga-examples-stop
saga-examples-stop:
	@echo "🛑 停止和清理 Saga 示例"
	@if [ -f "$(SAGA_SCRIPTS_DIR)/stop.sh" ]; then \
		cd $(SAGA_SCRIPTS_DIR) && ./stop.sh --all --force; \
	else \
		echo "❌ 停止脚本未找到: $(SAGA_SCRIPTS_DIR)/stop.sh"; \
		exit 1; \
	fi

# ==================================================================================
# 快捷 Commands - 常用示例
# ==================================================================================

# 运行订单处理示例
.PHONY: saga-examples-order
saga-examples-order:
	@echo "🛒 运行订单处理 Saga 示例"
	@$(SAGA_SCRIPTS_DIR)/run.sh order

# 运行支付处理示例
.PHONY: saga-examples-payment
saga-examples-payment:
	@echo "💳 运行支付处理 Saga 示例"
	@$(SAGA_SCRIPTS_DIR)/run.sh payment

# 运行库存管理示例
.PHONY: saga-examples-inventory
saga-examples-inventory:
	@echo "📦 运行库存管理 Saga 示例"
	@$(SAGA_SCRIPTS_DIR)/run.sh inventory

# 运行用户注册示例
.PHONY: saga-examples-user
saga-examples-user:
	@echo "👤 运行用户注册 Saga 示例"
	@$(SAGA_SCRIPTS_DIR)/run.sh user

# 运行端到端测试
.PHONY: saga-examples-e2e
saga-examples-e2e:
	@echo "🔄 运行端到端测试"
	@$(SAGA_SCRIPTS_DIR)/run.sh e2e

# 运行所有示例
.PHONY: saga-examples-all
saga-examples-all:
	@echo "🎯 运行所有 Saga 示例"
	@$(SAGA_SCRIPTS_DIR)/run.sh all

# ==================================================================================
# 高级 Commands
# ==================================================================================

# 高级运行模式
.PHONY: saga-examples-run-advanced
saga-examples-run-advanced:
	@echo "⚙️  高级运行模式"
	@if [ -z "$(EXAMPLE)" ]; then \
		echo "❌ 错误: 需要指定 EXAMPLE 参数"; \
		exit 1; \
	fi
	@OPTS=""; \
	[ -n "$(VERBOSE)" ] && OPTS="$$OPTS --verbose"; \
	[ -n "$(COVERAGE)" ] && OPTS="$$OPTS --coverage"; \
	[ -n "$(RACE)" ] && OPTS="$$OPTS --race"; \
	[ -n "$(TIMEOUT)" ] && OPTS="$$OPTS --timeout $(TIMEOUT)"; \
	$(SAGA_SCRIPTS_DIR)/run.sh $$OPTS $(EXAMPLE)

# 监控模式 - 持续查看状态
.PHONY: saga-examples-watch
saga-examples-watch:
	@echo "👀 监控 Saga 示例（持续刷新）"
	@$(SAGA_SCRIPTS_DIR)/status.sh --watch

# 覆盖率报告
.PHONY: saga-examples-coverage
saga-examples-coverage:
	@echo "📊 生成覆盖率报告"
	@if [ -z "$(EXAMPLE)" ]; then \
		EXAMPLE=all; \
	fi; \
	$(SAGA_SCRIPTS_DIR)/run.sh --coverage $(EXAMPLE)

# 查看详细日志
.PHONY: saga-examples-logs-detailed
saga-examples-logs-detailed:
	@echo "📋 查看详细日志"
	@$(SAGA_SCRIPTS_DIR)/logs.sh --verbose $(if $(TYPE),$(TYPE),all)

# 跟踪日志
.PHONY: saga-examples-logs-follow
saga-examples-logs-follow:
	@echo "🔄 跟踪日志（持续输出）"
	@$(SAGA_SCRIPTS_DIR)/logs.sh --follow $(if $(TYPE),$(TYPE),test)

# 清理特定类型
.PHONY: saga-examples-clean
saga-examples-clean:
	@echo "🧹 清理 Saga 示例"
	@if [ -n "$(TYPE)" ]; then \
		case "$(TYPE)" in \
			coverage) \
				$(SAGA_SCRIPTS_DIR)/stop.sh --coverage --force;; \
			logs) \
				$(SAGA_SCRIPTS_DIR)/stop.sh --logs --force;; \
			processes) \
				$(SAGA_SCRIPTS_DIR)/stop.sh --processes --force;; \
			*) \
				echo "❌ 未知的清理类型: $(TYPE)"; \
				echo "💡 可用类型: coverage, logs, processes"; \
				exit 1;; \
		esac \
	else \
		$(SAGA_SCRIPTS_DIR)/stop.sh --all --force; \
	fi

# ==================================================================================
# 实用功能
# ==================================================================================

# 列出所有可用示例
.PHONY: saga-examples-list
saga-examples-list:
	@echo "📋 可用的 Saga 示例："
	@echo ""
	@echo "  ${GREEN}order${NC}       订单处理 Saga - 电商订单处理完整流程"
	@echo "  ${GREEN}payment${NC}     支付处理 Saga - 跨账户资金转账流程"
	@echo "  ${GREEN}inventory${NC}   库存管理 Saga - 多仓库库存协调流程"
	@echo "  ${GREEN}user${NC}        用户注册 Saga - 用户注册和初始化流程"
	@echo "  ${GREEN}e2e${NC}         端到端测试 - 完整的集成测试套件"
	@echo "  ${GREEN}all${NC}         所有示例 - 运行所有测试"
	@echo ""
	@echo "💡 使用方法："
	@echo "  make saga-examples-run EXAMPLE=order      # 运行订单示例"
	@echo "  make saga-examples-order                  # 快捷方式"
	@echo "  make saga-examples-all                    # 运行所有示例"

# 健康检查
.PHONY: saga-examples-health
saga-examples-health:
	@echo "🏥 健康检查"
	@$(SAGA_SCRIPTS_DIR)/status.sh --health

# 显示示例文档
.PHONY: saga-examples-docs
saga-examples-docs:
	@echo "📚 Saga 示例文档"
	@echo ""
	@echo "📖 主文档: $(SAGA_EXAMPLES_DIR)/README.md"
	@echo ""
	@echo "📝 详细文档:"
	@echo "  订单处理: $(SAGA_EXAMPLES_DIR)/docs/order_saga.md"
	@echo "  支付处理: $(SAGA_EXAMPLES_DIR)/docs/payment_saga.md"
	@echo "  库存管理: $(SAGA_EXAMPLES_DIR)/docs/inventory_saga.md"
	@echo "  用户注册: $(SAGA_EXAMPLES_DIR)/docs/user_registration_saga.md"
	@echo "  架构设计: $(SAGA_EXAMPLES_DIR)/docs/architecture.md"
	@echo "  端到端测试: $(SAGA_EXAMPLES_DIR)/E2E_TESTING.md"
	@echo ""
	@echo "💡 在线查看:"
	@if command -v open >/dev/null 2>&1; then \
		echo "  open $(SAGA_EXAMPLES_DIR)/README.md"; \
	elif command -v xdg-open >/dev/null 2>&1; then \
		echo "  xdg-open $(SAGA_EXAMPLES_DIR)/README.md"; \
	else \
		echo "  cat $(SAGA_EXAMPLES_DIR)/README.md"; \
	fi

# ==================================================================================
# 帮助信息
# ==================================================================================

.PHONY: saga-examples-help
saga-examples-help:
	@echo "📋 SWIT Saga 示例管理命令"
	@echo ""
	@echo "🎯 核心命令:"
	@echo "  saga-examples-run              运行指定示例 (需要 EXAMPLE 参数)"
	@echo "  saga-examples-status           查看示例状态"
	@echo "  saga-examples-logs             查看示例日志"
	@echo "  saga-examples-stop             停止并清理所有内容"
	@echo ""
	@echo "🚀 快捷命令 (常用示例):"
	@echo "  saga-examples-order            运行订单处理示例"
	@echo "  saga-examples-payment          运行支付处理示例"
	@echo "  saga-examples-inventory        运行库存管理示例"
	@echo "  saga-examples-user             运行用户注册示例"
	@echo "  saga-examples-e2e              运行端到端测试"
	@echo "  saga-examples-all              运行所有示例"
	@echo ""
	@echo "⚙️  高级命令:"
	@echo "  saga-examples-run-advanced     高级运行模式 (支持多个选项)"
	@echo "  saga-examples-watch            监控模式（持续刷新状态）"
	@echo "  saga-examples-coverage         生成覆盖率报告"
	@echo "  saga-examples-logs-detailed    查看详细日志"
	@echo "  saga-examples-logs-follow      跟踪日志（持续输出）"
	@echo "  saga-examples-clean            清理示例文件"
	@echo ""
	@echo "🛠️  实用功能:"
	@echo "  saga-examples-list             列出所有可用示例"
	@echo "  saga-examples-health           健康检查"
	@echo "  saga-examples-docs             显示文档位置"
	@echo ""
	@echo "📖 使用示例:"
	@echo "  make saga-examples-run EXAMPLE=order              # 运行订单示例"
	@echo "  make saga-examples-order                          # 快捷方式"
	@echo "  make saga-examples-run-advanced EXAMPLE=payment COVERAGE=1  # 带覆盖率"
	@echo "  make saga-examples-watch                          # 监控模式"
	@echo "  make saga-examples-logs TYPE=coverage             # 查看覆盖率日志"
	@echo "  make saga-examples-clean TYPE=coverage            # 清理覆盖率文件"
	@echo ""
	@echo "⚙️  高级选项 (用于 saga-examples-run-advanced):"
	@echo "  EXAMPLE=<name>     示例名称 (order, payment, inventory, user, e2e, all)"
	@echo "  VERBOSE=1          详细输出"
	@echo "  COVERAGE=1         生成覆盖率"
	@echo "  RACE=1             启用竞态检测"
	@echo "  TIMEOUT=<duration> 超时时间 (例如: 30s, 5m)"
	@echo ""
	@echo "🔧 日志类型 (用于 saga-examples-logs):"
	@echo "  TYPE=test          测试输出日志"
	@echo "  TYPE=coverage      覆盖率报告"
	@echo "  TYPE=error         错误日志"
	@echo "  TYPE=debug         调试日志"
	@echo "  TYPE=all           所有日志 (默认)"
	@echo ""
	@echo "🧹 清理类型 (用于 saga-examples-clean):"
	@echo "  TYPE=coverage      清理覆盖率文件"
	@echo "  TYPE=logs          清理日志文件"
	@echo "  TYPE=processes     停止运行中的进程"
	@echo "  (不指定TYPE则清理所有)"
	@echo ""
	@echo "📚 更多信息:"
	@echo "  make saga-examples-docs                           # 查看文档位置"
	@echo "  cat $(SAGA_EXAMPLES_DIR)/README.md               # 查看主文档"
	@echo ""
	@echo "💡 直接使用脚本:"
	@echo "  $(SAGA_SCRIPTS_DIR)/run.sh --help"
	@echo "  $(SAGA_SCRIPTS_DIR)/status.sh --help"
	@echo "  $(SAGA_SCRIPTS_DIR)/logs.sh --help"
	@echo "  $(SAGA_SCRIPTS_DIR)/stop.sh --help"

# ==================================================================================
# 别名 (向后兼容)
# ==================================================================================

.PHONY: saga-run
saga-run: saga-examples-run

.PHONY: saga-status
saga-status: saga-examples-status

.PHONY: saga-logs
saga-logs: saga-examples-logs

.PHONY: saga-stop
saga-stop: saga-examples-stop

