.DEFAULT_GOAL := all

GO:=go
BIN_NAME=swit-serve
MAIN_DIR=cmd/swit-serve
MAIN_FILE=swit-serve.go
OUTPUTDIR=_output

.PHONY: all
all: tidy build

define USAGE_OPTIONS

Options:
  TIDY             Format the code.
  BUILD            Build the binary, output binary is in _output directory.
  CLEAN            Delete the output binary.
 endef
 export USAGE_OPTIONS

.PHONY: tidy
tidy:
	@echo "go mod tidy"
	@$(GO) mod tidy

.PHONY: build
build:
	@echo "build go program: swit-serve"
	@$(GO) build -o $(OUTPUTDIR)/$(BIN_NAME) $(MAIN_DIR)/$(MAIN_FILE)

.PHONY: clean
clean:
	@echo "clean go program"
	@$(RM) -rf $(OUTPUTDIR)/

.PHONY: help
help:
	@echo "$$USAGE_OPTIONS"

.PHONY: test
test:
	@echo $(shell pwd)