# 构建相关规则

.PHONY: tidy
tidy:
	@echo "Running go mod tidy"
	@$(GO) mod tidy

.PHONY: format
format:
	@echo "Formatting Go code"
	@$(GOFMT) -w .
	@echo "Code formatting completed"

# 独立vet目标（包含依赖）
.PHONY: vet
vet: proto-generate swagger
	@echo "Running go vet"
	@$(GOVET) ./...
	@echo "Vet completed"

# 快速vet目标（用于组合目标）
.PHONY: vet-fast
vet-fast:
	@echo "Running go vet"
	@$(GOVET) ./...
	@echo "Vet completed"

# 独立质量检查目标
.PHONY: quality
quality: format vet
	@echo "All quality checks completed"

# 快速质量检查目标（用于组合目标）
.PHONY: quality-fast
quality-fast: format vet-fast
	@echo "All quality checks completed"

# 独立构建目标（包含所有依赖，优化版）
.PHONY: build  
build: proto-generate swagger quality-fast build-serve-fast build-ctl-fast build-auth-fast
	@echo "All binaries built successfully"

# 组合构建目标（用于all，假设依赖已满足）
.PHONY: build-fast
build-fast: quality-fast build-serve-fast build-ctl-fast build-auth-fast
	@echo "All binaries built successfully"

# 独立构建目标（带依赖）
.PHONY: build-serve-standalone
build-serve-standalone: proto-generate swagger
	@echo "Building go program: swit-serve"
	@mkdir -p $(OUTPUTDIR)/$(SERVE_BIN_NAME)
	@$(GO) build -o $(OUTPUTDIR)/$(SERVE_BIN_NAME)/$(SERVE_BIN_NAME) $(SERVE_MAIN_DIR)/$(SERVE_MAIN_FILE)

.PHONY: build-ctl-standalone  
build-ctl-standalone: proto-generate
	@echo "Building go program: switctl"
	@mkdir -p $(OUTPUTDIR)/$(CTL_BIN_NAME)
	@$(GO) build -o $(OUTPUTDIR)/$(CTL_BIN_NAME)/$(CTL_BIN_NAME) $(CTL_MAIN_DIR)/$(CTL_MAIN_FILE)

.PHONY: build-auth-standalone
build-auth-standalone: proto-generate swagger
	@echo "Building go program: swit-auth"
	@mkdir -p $(OUTPUTDIR)/$(AUTH_BIN_NAME)
	@$(GO) build -o $(OUTPUTDIR)/$(AUTH_BIN_NAME)/$(AUTH_BIN_NAME) $(AUTH_MAIN_DIR)/$(AUTH_MAIN_FILE)

# 快速构建目标（无重复依赖）
.PHONY: build-serve-fast
build-serve-fast:
	@echo "Building go program: swit-serve"
	@mkdir -p $(OUTPUTDIR)/$(SERVE_BIN_NAME)
	@$(GO) build -o $(OUTPUTDIR)/$(SERVE_BIN_NAME)/$(SERVE_BIN_NAME) $(SERVE_MAIN_DIR)/$(SERVE_MAIN_FILE)

.PHONY: build-ctl-fast
build-ctl-fast:
	@echo "Building go program: switctl"
	@mkdir -p $(OUTPUTDIR)/$(CTL_BIN_NAME)
	@$(GO) build -o $(OUTPUTDIR)/$(CTL_BIN_NAME)/$(CTL_BIN_NAME) $(CTL_MAIN_DIR)/$(CTL_MAIN_FILE)

.PHONY: build-auth-fast
build-auth-fast:
	@echo "Building go program: swit-auth"
	@mkdir -p $(OUTPUTDIR)/$(AUTH_BIN_NAME)
	@$(GO) build -o $(OUTPUTDIR)/$(AUTH_BIN_NAME)/$(AUTH_BIN_NAME) $(AUTH_MAIN_DIR)/$(AUTH_MAIN_FILE)

# 兼容性别名
.PHONY: build-serve build-ctl build-auth
build-serve: build-serve-standalone
build-ctl: build-ctl-standalone  
build-auth: build-auth-standalone

# 清理所有自动生成的代码和文件
.PHONY: clean
clean:
	@echo "🧹 Cleaning all generated code and build artifacts..."
	@echo "  Removing build outputs..."
	@$(RM) -rf $(OUTPUTDIR)/
	@echo "  Removing protobuf generated code..."
	@$(RM) -rf api/gen/
	@echo "  Removing Swagger generated documentation..."
	@$(RM) -rf docs/generated/
	@echo "  Removing individual Swagger docs.go files..."
	@find internal -path "*/docs/docs.go" -type f -delete 2>/dev/null || true
	@echo "  Removing test coverage files..."
	@$(RM) -f coverage.out coverage.html
	@echo "  Removing log files..."
	@$(RM) -f *.log test.log
	@echo "  Removing temporary files..."
	@$(RM) -f .DS_Store
	@find . -name "*.tmp" -type f -delete 2>/dev/null || true
	@echo "✅ All generated code and build artifacts cleaned"

# 快速清理（仅删除构建输出）
.PHONY: clean-build
clean-build:
	@echo "Cleaning build outputs only"
	@$(RM) -rf $(OUTPUTDIR)/
	@echo "Build outputs cleaned"

# 清理protobuf生成代码
.PHONY: clean-proto
clean-proto:
	@echo "Cleaning protobuf generated code"
	@$(RM) -rf api/gen/
	@echo "Protobuf generated code cleaned"

# 清理Swagger生成文档
.PHONY: clean-swagger
clean-swagger:
	@echo "Cleaning Swagger generated documentation"
	@$(RM) -rf docs/generated/
	@find internal -path "*/docs/docs.go" -type f -delete 2>/dev/null || true
	@echo "Swagger generated documentation cleaned"

# 清理测试文件
.PHONY: clean-test
clean-test:
	@echo "Cleaning test artifacts"
	@$(RM) -f coverage.out coverage.html
	@$(RM) -f *.log test.log
	@echo "Test artifacts cleaned" 