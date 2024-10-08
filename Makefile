.DEFAULT_GOAL := all

GO := go
SERVE_BIN_NAME := swit-serve
SERVE_MAIN_DIR := cmd/swit-serve
SERVE_MAIN_FILE := swit-serve.go
CTL_BIN_NAME := switctl
CTL_MAIN_DIR := cmd/switctl
CTL_MAIN_FILE := switctl.go
OUTPUTDIR := _output
DOCKER := docker
DOCKER_IMAGE_NAME := swit-serve
DOCKER_IMAGE_TAG := $(shell git rev-parse --abbrev-ref HEAD)

.PHONY: all
all: tidy build

define USAGE_OPTIONS

Options:
  TIDY             Format the code.
  BUILD            Build the binaries, output binaries are in _output/{application_name}/ directory.
  CLEAN            Delete the output binaries.
  TEST             Run the tests.
  IMAGE-SERVE      Build Docker image for swit-serve.
endef
export USAGE_OPTIONS

.PHONY: tidy
tidy:
	@echo "Running go mod tidy"
	@$(GO) mod tidy

.PHONY: build
build: build-serve build-ctl

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

.PHONY: clean
clean:
	@echo "Cleaning go programs"
	@$(RM) -rf $(OUTPUTDIR)/

.PHONY: help
help:
	@echo "$$USAGE_OPTIONS"

.PHONY: test
test:
	@$(GO) test -v ./internal/...

.PHONY: image-serve
image-serve:
	@echo "Building Docker image for swit-serve"
	@$(DOCKER) build -f build/docker/swit-serve/Dockerfile -t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .
