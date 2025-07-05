BOILERPLATE_FILE := scripts/boilerplate.txt
GO_FILES := $(shell find . -name '*.go')
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
	@echo "Checking Go files for copyright statements"
	@if [ -n "$(MISSING_COPYRIGHT)" ]; then \
		echo "The following files are missing copyright statements:"; \
		echo "$(MISSING_COPYRIGHT)" | tr ' ' '\n'; \
	fi
	@if [ -n "$(OUTDATED_COPYRIGHT)" ]; then \
		echo "The following files have outdated copyright statements:"; \
		echo "$(OUTDATED_COPYRIGHT)" | tr ' ' '\n'; \
	fi
	@if [ -z "$(MISSING_COPYRIGHT)" ] && [ -z "$(OUTDATED_COPYRIGHT)" ]; then \
		echo "All Go files have up-to-date copyright statements"; \
	fi

.PHONY: copyright-add
copyright-add:
	@echo "Adding copyright statements to Go files"
	@for file in $(MISSING_COPYRIGHT); do \
		echo "Adding copyright statement to $$file"; \
		sed 's/^/\/\/ /' $(BOILERPLATE_FILE) > temp_boilerplate.txt; \
		echo "" >> temp_boilerplate.txt; \
		cat temp_boilerplate.txt $$file > $$file.tmp && mv $$file.tmp $$file; \
		rm temp_boilerplate.txt; \
	done

.PHONY: copyright-update
copyright-update:
	@echo "Updating outdated copyright statements in Go files"
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
	@echo "Force updating all Go files with current copyright statement"
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
		echo "Found files missing copyright statements, adding..."; \
		$(MAKE) copyright-add; \
		echo "Copyright statements added"; \
	fi
	@if [ -n "$(OUTDATED_COPYRIGHT)" ]; then \
		echo "Found files with outdated copyright statements, updating..."; \
		$(MAKE) copyright-update; \
		echo "Copyright statements updated"; \
	fi
