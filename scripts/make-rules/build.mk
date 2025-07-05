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

.PHONY: vet
vet:
	@echo "Running go vet"
	@$(GOVET) ./...
	@echo "Vet completed"

.PHONY: quality
quality: format vet
	@echo "All quality checks completed"

.PHONY: build
build: quality build-serve build-ctl build-auth
	@echo "All binaries built successfully"

.PHONY: build-serve
build-serve:
	@echo "Building go program: swit-serve"
	@mkdir -p $(OUTPUTDIR)/$(SERVE_BIN_NAME)
	@$(GO) build -o $(OUTPUTDIR)/$(SERVE_BIN_NAME)/$(SERVE_BIN_NAME) $(SERVE_MAIN_DIR)/$(SERVE_MAIN_FILE)

.PHONY: build-ctl
build-ctl:
	@echo "Building go program: switctl"
	@mkdir -p $(OUTPUTDIR)/$(CTL_BIN_NAME)
	@$(GO) build -o $(OUTPUTDIR)/$(CTL_BIN_NAME)/$(CTL_BIN_NAME) $(CTL_MAIN_DIR)/$(CTL_MAIN_FILE)

.PHONY: build-auth
build-auth:
	@echo "Building go program: swit-auth"
	@mkdir -p $(OUTPUTDIR)/$(AUTH_BIN_NAME)
	@$(GO) build -o $(OUTPUTDIR)/$(AUTH_BIN_NAME)/$(AUTH_BIN_NAME) $(AUTH_MAIN_DIR)/$(AUTH_MAIN_FILE)

.PHONY: clean
clean:
	@echo "Cleaning go programs"
	@$(RM) -rf $(OUTPUTDIR)/ 