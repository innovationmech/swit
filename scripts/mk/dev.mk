# 开发环境相关规则

.PHONY: install-hooks
install-hooks:
	@echo "Installing Git hooks"
	@cp scripts/tools/pre-commit.sh .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed"

.PHONY: setup-dev
setup-dev: install-hooks swagger-install proto-setup
	@echo "Development environment setup completed"

.PHONY: ci
ci: tidy copyright quality proto test
	@echo "CI pipeline completed successfully" 