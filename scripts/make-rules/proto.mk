# Protobuf 相关规则

# Buf 工具安装
.PHONY: buf-install
buf-install:
	@echo "Installing Buf CLI"
	@if ! command -v $(BUF) &> /dev/null; then \
		echo "Installing buf..."; \
		$(GO) install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION); \
	else \
		echo "buf is already installed"; \
	fi

# 下载 protobuf 依赖
.PHONY: proto-deps
proto-deps: buf-install
	@echo "Downloading protobuf dependencies"
	@cd $(API_DIR) && $(BUF) mod update

# 跳过依赖下载的快速生成（用于开发）
.PHONY: proto-generate-dev
proto-generate-dev: buf-install
	@echo "Generating protobuf code (dev mode - skipping deps)"
	@cd $(API_DIR) && \
	if $(BUF) generate --exclude-imports; then \
		echo "✅ Protobuf code generation completed (dev mode)"; \
	else \
		echo "❌ Protobuf code generation failed"; \
		echo "💡 Try using 'make proto-generate' for full generation with dependencies"; \
		exit 1; \
	fi

# 生成 protobuf 代码（带重试逻辑）
.PHONY: proto-generate
proto-generate: proto-deps
	@echo "Generating protobuf code"
	@cd $(API_DIR) && \
	for i in 1 2 3; do \
		echo "🔄 Attempt $$i/3..."; \
		if $(BUF) generate 2>&1; then \
			echo "✅ Protobuf code generation completed"; \
			exit 0; \
		else \
			if [ $$i -lt 3 ]; then \
				echo "⚠️  Generation failed, waiting 30 seconds before retry..."; \
				sleep 30; \
			else \
				echo "❌ Protobuf code generation failed after 3 attempts"; \
				echo "💡 This might be due to BSR rate limits or network issues"; \
				echo "📖 See: https://buf.build/docs/bsr/rate-limits/"; \
				echo "🔧 Try running 'make proto-generate' again in a few minutes"; \
				exit 1; \
			fi; \
		fi; \
	done



# 检查 protobuf 文件
.PHONY: proto-lint
proto-lint: buf-install
	@echo "Linting protobuf files"
	@cd $(API_DIR) && $(BUF) lint

# 检查破坏性变更
.PHONY: proto-breaking
proto-breaking: buf-install
	@echo "Checking for breaking changes"
	@cd $(API_DIR) && $(BUF) breaking --against '.git#branch=main'

# 格式化 protobuf 文件
.PHONY: proto-format
proto-format: buf-install
	@echo "Formatting protobuf files"
	@cd $(API_DIR) && $(BUF) format -w

# 清理生成的 protobuf 代码
.PHONY: proto-clean
proto-clean: clean-proto

# 完整的 protobuf 工作流
.PHONY: proto
proto: proto-format proto-lint proto-generate
	@echo "Protobuf generation completed"

# 设置 protobuf 开发环境
.PHONY: proto-setup
proto-setup: buf-install proto-deps
	@echo "Protobuf development environment setup completed"

# 验证 protobuf 配置
.PHONY: proto-validate
proto-validate: buf-install
	@echo "Validating protobuf configuration"
	@cd $(API_DIR) && $(BUF) mod ls-lint-rules
	@cd $(API_DIR) && $(BUF) mod ls-breaking-rules
	@echo "Protobuf configuration validation completed"

# 生成 OpenAPI 文档
.PHONY: proto-docs
proto-docs: proto-generate
	@echo "Generating OpenAPI documentation"
	@if [ -d "$(API_DIR)/gen/openapiv2" ]; then \
		echo "OpenAPI documentation generated at: $(API_DIR)/gen/openapiv2/"; \
		find $(API_DIR)/gen/openapiv2 -name "*.json" -o -name "*.yaml" | head -5; \
	else \
		echo "No OpenAPI documentation found"; \
	fi
