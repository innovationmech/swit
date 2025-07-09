BOILERPLATE_FILE := scripts/boilerplate.txt

# 查找所有 Go 文件，但排除生成的代码目录
GO_FILES := $(shell find . -name '*.go' -not -path './api/gen/*' -not -path './_output/*' -not -path './vendor/*' -not -path './internal/*/docs/docs.go')

MISSING_COPYRIGHT := $(shell for file in $(GO_FILES); do if ! grep -q "Copyright" $$file; then echo $$file; fi; done)

# 生成标准版权声明的哈希值
STANDARD_COPYRIGHT_HASH := $(shell sed 's/^/\/\/ /' $(BOILERPLATE_FILE) | shasum -a 256 | cut -d' ' -f1)

# 检查过期版权声明：比较文件中的版权声明哈希值与标准哈希值
OUTDATED_COPYRIGHT := $(shell for file in $(GO_FILES); do \
	if grep -q "Copyright" $$file; then \
		file_hash=$$(awk '/^\/\/ Copyright/{found=1} found && !/^\/\//{found=0; exit} found{print}' $$file | shasum -a 256 | cut -d' ' -f1); \
		if [ "$$file_hash" != "$(STANDARD_COPYRIGHT_HASH)" ]; then \
			echo $$file; \
		fi; \
	fi; \
done)

# 调试目标：显示哪些文件被包含或排除
.PHONY: copyright-files
copyright-files:
	@echo "📋 版权管理包含的文件："
	@for file in $(GO_FILES); do echo "$$file"; done
	@echo "📊 统计信息："
	@echo "  总文件数:       $$(echo '$(GO_FILES)' | wc -w)"
	@echo "  缺少版权:       $$(echo '$(MISSING_COPYRIGHT)' | wc -w)"
	@echo "  过期版权:       $$(echo '$(OUTDATED_COPYRIGHT)' | wc -w)"
	@echo ""
	@echo "🚫 排除的目录："
	@echo "  - api/gen/*               (生成的 gRPC 代码)"
	@echo "  - _output/*               (构建输出)"
	@echo "  - vendor/*                (第三方依赖)"
	@echo "  - internal/*/docs/docs.go (生成的 Swagger 文档)"
	@echo ""
	@echo "📂 示例排除的文件："
	@find . -name '*.go' \( -path './api/gen/*' -o -path './_output/*' -o -path './vendor/*' -o -path './internal/*/docs/docs.go' \) 2>/dev/null | head -5 || echo "  (暂无排除的文件)"

# 调试目标：显示哈希值比较信息（用于故障排除）
.PHONY: copyright-debug
copyright-debug:
	@echo "Standard copyright hash: $(STANDARD_COPYRIGHT_HASH)"
	@echo "Standard copyright content:"
	@sed 's/^/\/\/ /' $(BOILERPLATE_FILE)
	@echo "--- Checking first 3 files ---"
	@for file in $(shell echo $(GO_FILES) | tr ' ' '\n' | head -3); do \
		if grep -q "Copyright" $$file; then \
			echo "File: $$file"; \
			file_hash=$$(awk '/^\/\/ Copyright/{found=1} found && !/^\/\//{found=0; exit} found{print}' $$file | shasum -a 256 | cut -d' ' -f1); \
			echo "File hash: $$file_hash"; \
			echo "File copyright content:"; \
			awk '/^\/\/ Copyright/{found=1} found && !/^\/\//{found=0; exit} found{print}' $$file; \
			echo "Match: $$(if [ "$$file_hash" = "$(STANDARD_COPYRIGHT_HASH)" ]; then echo "YES"; else echo "NO"; fi)"; \
			echo "---"; \
		fi; \
	done

.PHONY: copyright-check
copyright-check:
	@echo "🔍 检查 Go 文件版权声明（排除生成代码）"
	@if [ -n "$(MISSING_COPYRIGHT)" ]; then \
		echo "❌ 以下文件缺少版权声明:"; \
		echo "$(MISSING_COPYRIGHT)" | tr ' ' '\n'; \
		echo ""; \
	fi
	@if [ -n "$(OUTDATED_COPYRIGHT)" ]; then \
		echo "⚠️  以下文件版权声明过期:"; \
		echo "$(OUTDATED_COPYRIGHT)" | tr ' ' '\n'; \
		echo ""; \
	fi
	@if [ -z "$(MISSING_COPYRIGHT)" ] && [ -z "$(OUTDATED_COPYRIGHT)" ]; then \
		echo "✅ 所有 Go 文件都有最新的版权声明"; \
	fi
	@echo "📊 文件统计: $(shell echo $(GO_FILES) | wc -w) 个文件已检查"

.PHONY: copyright-add
copyright-add:
	@echo "📝 为 Go 文件添加版权声明"
	@for file in $(MISSING_COPYRIGHT); do \
		echo "Adding copyright statement to $$file"; \
		sed 's/^/\/\/ /' $(BOILERPLATE_FILE) > temp_boilerplate.txt; \
		echo "" >> temp_boilerplate.txt; \
		cat temp_boilerplate.txt $$file > $$file.tmp && mv $$file.tmp $$file; \
		rm temp_boilerplate.txt; \
	done

.PHONY: copyright-update
copyright-update:
	@echo "🔄 更新过期的版权声明"
	@for file in $(OUTDATED_COPYRIGHT); do \
		echo "Updating copyright statement in $$file"; \
		sed 's/^/\/\/ /' $(BOILERPLATE_FILE) > temp_boilerplate.txt; \
		echo "" >> temp_boilerplate.txt; \
		awk '/^\/\/ Copyright/{found=1} found && /^$$/{found=0; next} !found' $$file > temp_file.txt; \
		cat temp_boilerplate.txt temp_file.txt > $$file.tmp && mv $$file.tmp $$file; \
		rm temp_boilerplate.txt temp_file.txt; \
	done

.PHONY: copyright-force
copyright-force:
	@echo "⚡ 强制更新所有 Go 文件的版权声明"
	@for file in $(GO_FILES); do \
		echo "Force updating copyright statement in $$file"; \
		sed 's/^/\/\/ /' $(BOILERPLATE_FILE) > temp_boilerplate.txt; \
		echo "" >> temp_boilerplate.txt; \
		awk '/^\/\/ Copyright/{found=1} found && /^$$/{found=0; next} !found' $$file > temp_file.txt; \
		cat temp_boilerplate.txt temp_file.txt > $$file.tmp && mv $$file.tmp $$file; \
		rm temp_boilerplate.txt temp_file.txt; \
	done

.PHONY: copyright
copyright:
	@$(MAKE) copyright-check
	@if [ -n "$(MISSING_COPYRIGHT)" ]; then \
		echo "发现缺少版权声明的文件，正在添加..."; \
		$(MAKE) copyright-add; \
		echo "✅ 版权声明已添加"; \
	fi
	@if [ -n "$(OUTDATED_COPYRIGHT)" ]; then \
		echo "发现过期版权声明的文件，正在更新..."; \
		$(MAKE) copyright-update; \
		echo "✅ 版权声明已更新"; \
	fi
