.DEFAULT_GOAL := all

GO := go
GOFMT := gofmt
GOVET := $(GO) vet
SERVE_BIN_NAME := swit-serve
SERVE_MAIN_DIR := cmd/swit-serve
SERVE_MAIN_FILE := swit-serve.go
CTL_BIN_NAME := switctl
CTL_MAIN_DIR := cmd/switctl
CTL_MAIN_FILE := switctl.go
AUTH_BIN_NAME := swit-auth
AUTH_MAIN_DIR := cmd/swit-auth
AUTH_MAIN_FILE := swit-auth.go
OUTPUTDIR := _output
DOCKER := docker
DOCKER_IMAGE_NAME := swit-serve
DOCKER_AUTH_IMAGE_NAME := swit-auth
DOCKER_IMAGE_TAG := $(shell git rev-parse --abbrev-ref HEAD)
SWAG := swag

include scripts/make-rules/copyright.mk

.PHONY: all
all: tidy copyright build swagger

define USAGE_OPTIONS

Options:
  TIDY             Run go mod tidy.
  FORMAT           Format the code using gofmt.
  QUALITY          Run all quality checks (format, lint, vet).
  BUILD            Build the binaries, output binaries are in _output/{application_name}/ directory.
  CLEAN            Delete the output binaries.
  TEST             Run all tests (internal + pkg).
  TEST-PKG         Run tests for pkg packages only.
  TEST-INTERNAL    Run tests for internal packages only.
  TEST-COVERAGE    Run tests with coverage report.
  TEST-RACE        Run tests with race detection.
  CI               Run full CI pipeline (tidy, copyright, quality, test).
  IMAGE-SERVE      Build Docker image for swit-serve.
  IMAGE-AUTH       Build Docker image for swit-auth.
  IMAGE-ALL        Build Docker images for all services.
  INSTALL-HOOKS    Install Git pre-commit hooks.
  SETUP-DEV        Setup development environment (tools + hooks).
  SWAGGER          Generate/update Swagger documentation for all services.
  SWAGGER-SWITSERVE Generate Swagger documentation for switserve only.
  SWAGGER-SWITAUTH Generate Swagger documentation for switauth only.
  SWAGGER-COPY     Create unified documentation links in docs/generated/.
  SWAGGER-INSTALL  Install swag tool for generating Swagger docs.
endef
export USAGE_OPTIONS

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

.PHONY: help
help:
	@echo "$$USAGE_OPTIONS"

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

.PHONY: image-serve
image-serve:
	@echo "Building Docker image for swit-serve"
	@$(DOCKER) build -f build/docker/swit-serve/Dockerfile -t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .

.PHONY: image-auth
image-auth:
	@echo "Building Docker image for swit-auth"
	@$(DOCKER) build -f build/docker/switauth/Dockerfile -t $(DOCKER_AUTH_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .

.PHONY: image-all
image-all: image-serve image-auth
	@echo "All Docker images built successfully"

.PHONY: ci
ci: tidy copyright quality test
	@echo "CI pipeline completed successfully"

.PHONY: install-hooks
install-hooks:
	@echo "Installing Git hooks"
	@cp scripts/pre-commit.sh .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed"

.PHONY: setup-dev
setup-dev: install-hooks swagger-install
	@echo "Development environment setup completed"

.PHONY: swagger-install
swagger-install:
	@echo "Installing swag tool"
	@$(GO) install github.com/swaggo/swag/cmd/swag@v1.8.12
	@echo "swag tool installed"

# Generate Swagger documentation for all services
.PHONY: swagger
swagger: swagger-fmt swagger-switserve swagger-switauth swagger-copy
	@echo "All Swagger documentation generated and organized"

# Generate Swagger documentation for switserve
.PHONY: swagger-switserve
swagger-switserve:
	@echo "Generating Swagger documentation for switserve"
	@$(SWAG) init -g cmd/swit-serve/swit-serve.go -o internal/switserve/docs --parseDependency --parseInternal --exclude internal/switauth
	@echo "SwitServe Swagger documentation generated at: internal/switserve/docs/"

# Generate Swagger documentation for switauth
.PHONY: swagger-switauth
swagger-switauth:
	@echo "Generating Swagger documentation for switauth"
	@$(SWAG) init -g cmd/swit-auth/swit-auth.go -o internal/switauth/docs --parseDependency --parseInternal --exclude internal/switserve
	@echo "SwitAuth Swagger documentation generated at: internal/switauth/docs/"

# Format Swagger annotations for all services
.PHONY: swagger-fmt
swagger-fmt: swagger-fmt-switserve swagger-fmt-switauth
	@echo "All Swagger annotations formatted"

.PHONY: swagger-fmt-switserve
swagger-fmt-switserve:
	@echo "Formatting Swagger annotations for switserve"
	@$(SWAG) fmt -g cmd/swit-serve
	@echo "SwitServe Swagger annotations formatted"

.PHONY: swagger-fmt-switauth
swagger-fmt-switauth:
	@echo "Formatting Swagger annotations for switauth"
	@$(SWAG) fmt -g cmd/swit-auth
	@echo "SwitAuth Swagger annotations formatted"

# Copy generated docs to unified location for easy access
.PHONY: swagger-copy
swagger-copy: swagger-switserve swagger-switauth
	@echo "Creating unified documentation links"
	@mkdir -p docs/generated/switserve
	@mkdir -p docs/generated/switauth
	@echo "# SwitServe API Documentation" > docs/generated/switserve/README.md
	@echo "" >> docs/generated/switserve/README.md
	@echo "**Generated API Documentation**: [internal/switserve/docs/](../../../internal/switserve/docs/)" >> docs/generated/switserve/README.md
	@echo "**Swagger UI**: http://localhost:9000/swagger/index.html" >> docs/generated/switserve/README.md
	@echo "**API Base URL**: http://localhost:9000" >> docs/generated/switserve/README.md
	@echo "" >> docs/generated/switserve/README.md
	@echo "文档文件位置：" >> docs/generated/switserve/README.md
	@echo "- JSON: [swagger.json](../../../internal/switserve/docs/swagger.json)" >> docs/generated/switserve/README.md
	@echo "- YAML: [swagger.yaml](../../../internal/switserve/docs/swagger.yaml)" >> docs/generated/switserve/README.md
	@echo "- Go Code: [docs.go](../../../internal/switserve/docs/docs.go)" >> docs/generated/switserve/README.md
	@echo "# SwitAuth API Documentation" > docs/generated/switauth/README.md
	@echo "" >> docs/generated/switauth/README.md
	@echo "**Generated API Documentation**: [internal/switauth/docs/](../../../internal/switauth/docs/)" >> docs/generated/switauth/README.md
	@echo "**Swagger UI**: http://localhost:8090/swagger/index.html" >> docs/generated/switauth/README.md
	@echo "**API Base URL**: http://localhost:8090" >> docs/generated/switauth/README.md
	@echo "" >> docs/generated/switauth/README.md
	@echo "文档文件位置：" >> docs/generated/switauth/README.md
	@echo "- JSON: [swagger.json](../../../internal/switauth/docs/swagger.json)" >> docs/generated/switauth/README.md
	@echo "- YAML: [swagger.yaml](../../../internal/switauth/docs/swagger.yaml)" >> docs/generated/switauth/README.md
	@echo "- Go Code: [docs.go](../../../internal/switauth/docs/docs.go)" >> docs/generated/switauth/README.md
	@echo "Unified documentation access created"

