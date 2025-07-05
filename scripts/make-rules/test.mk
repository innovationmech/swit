# 测试相关规则

.PHONY: test
test:
	@echo "Running tests"
	@$(GO) test -v ./internal/... ./pkg/...

.PHONY: test-pkg
test-pkg:
	@echo "Running tests for pkg"
	@$(GO) test -v ./pkg/...

.PHONY: test-internal
test-internal:
	@echo "Running tests for internal"
	@$(GO) test -v ./internal/...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage"
	@$(GO) test -v -coverprofile=coverage.out ./internal/... ./pkg/...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-race
test-race:
	@echo "Running tests with race detection"
	@$(GO) test -v -race ./internal/... ./pkg/... 