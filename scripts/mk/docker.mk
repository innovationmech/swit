# Docker 相关规则
# 统一使用 scripts/tools/docker-manage.sh 脚本进行管理

# 标准Docker构建（推荐用于生产发布）
.PHONY: docker
docker:
	@echo "执行标准Docker构建..."
	@$(PROJECT_ROOT)/scripts/tools/docker-manage.sh standard

# 快速Docker构建（开发时使用缓存）
.PHONY: docker-dev
docker-dev:
	@echo "执行快速Docker构建..."
	@$(PROJECT_ROOT)/scripts/tools/docker-manage.sh quick

# Docker开发环境设置（启动完整的开发环境）
.PHONY: docker-setup
docker-setup:
	@echo "设置Docker开发环境..."
	@$(PROJECT_ROOT)/scripts/tools/docker-manage.sh setup

# 高级Docker管理（精确控制特定操作）
.PHONY: docker-advanced
docker-advanced:
	@echo "执行高级Docker管理..."
	@if [ -z "$(OPERATION)" ]; then \
		echo "使用方法: make docker-advanced OPERATION=<操作> [COMPONENT=<组件>] [SERVICE=<服务>]"; \
		echo "示例: make docker-advanced OPERATION=build COMPONENT=images SERVICE=auth"; \
		$(PROJECT_ROOT)/scripts/tools/docker-manage.sh advanced help; \
	else \
		$(PROJECT_ROOT)/scripts/tools/docker-manage.sh advanced $(OPERATION) $(COMPONENT) $(SERVICE); \
	fi 