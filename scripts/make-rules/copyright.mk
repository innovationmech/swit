BOILERPLATE_FILE := scripts/boilerplate.txt
GO_FILES := $(shell find . -name '*.go')
MISSING_COPYRIGHT := $(shell for file in $(GO_FILES); do if ! grep -q "Copyright" $$file; then echo $$file; fi; done)

.PHONY: copyright-check
copyright-check:
	@echo "Checking Go files for copyright statements"
	@if [ -n "$(MISSING_COPYRIGHT)" ]; then \
		echo "The following files are missing copyright statements:"; \
		echo "$(MISSING_COPYRIGHT)" | tr ' ' '\n'; \
	else \
		echo "All Go files contain copyright statements"; \
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

.PHONY: copyright
copyright:
	@$(MAKE) copyright-check
	@if [ -n "$(MISSING_COPYRIGHT)" ]; then \
		echo "Found files missing copyright statements, adding..."; \
		$(MAKE) copyright-add; \
		echo "Copyright statements added"; \
	fi
