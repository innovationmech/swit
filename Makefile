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
DOCKER_IMAGE_TAG := $(shell git rev-parse --abbrev-ref HEAD)

include scripts/make-rules/copyright.mk

.PHONY: all
all: tidy copyright quality build

define USAGE_OPTIONS

Options:
  TIDY             Run go mod tidy.
  FORMAT           Format the code using gofmt.
  QUALITY          Run all quality checks (format, lint, vet).
  BUILD            Build the binaries, output binaries are in _output/{application_name}/ directory.
  CLEAN            Delete the output binaries.
  TEST             Run the tests.
  TEST-COVERAGE    Run tests with coverage report.
  TEST-RACE        Run tests with race detection.
  CI               Run full CI pipeline (tidy, copyright, quality, test).
  IMAGE-SERVE      Build Docker image for swit-serve.
  INSTALL-HOOKS    Install Git pre-commit hooks.
  SETUP-DEV        Setup development environment (tools + hooks).
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
	@$(GO) test -v ./internal/...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage"
	@$(GO) test -v -coverprofile=coverage.out ./internal/...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-race
test-race:
	@echo "Running tests with race detection"
	@$(GO) test -v -race ./internal/...

.PHONY: image-serve
image-serve:
	@echo "Building Docker image for swit-serve"
	@$(DOCKER) build -f build/docker/swit-serve/Dockerfile -t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .

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
setup-dev: install-hooks
	@echo "Development environment setup completed"
