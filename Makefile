.DEFAULT_GOAL := all

GO:=go
BIN_NAME=myapp
MAIN_FILE=main.go
OUTPUTDIR=_output

.PHONY: all
all: tidy build

define USAGE_OPTIONS

Options:
  DEBUG            Whether to generate debug symbols. Default is 0.
  BINS             The binaries to build. Default is all of cmd.
                   This option is available when using: make build/build.multiarch
                   Example: make build BINS="iam-apiserver iam-authz-server"
  IMAGES           Backend images to make. Default is all of cmd starting with iam-.
                   This option is available when using: make image/image.multiarch/push/push.multiarch
                   Example: make image.multiarch IMAGES="iam-apiserver iam-authz-server"
  REGISTRY_PREFIX  Docker registry prefix. Default is marmotedu.
                   Example: make push REGISTRY_PREFIX=ccr.ccs.tencentyun.com/marmotedu VERSION=v1.6.2
  PLATFORMS        The multiple platforms to build. Default is linux_amd64 and linux_arm64.
                   This option is available when using: make build.multiarch/image.multiarch/push.multiarch
                   Example: make image.multiarch IMAGES="iam-apiserver iam-pump" PLATFORMS="linux_amd64 linux_arm64"
  VERSION          The version information compiled into binaries.
                   The default is obtained from gsemver or git.
  V                Set to 1 enable verbose build. Default is 0.
 endef
 export USAGE_OPTIONS

.PHONY: tidy
tidy:
	@echo "go mod tidy"
	@$(GO) mod tidy

.PHONY: build
build:
	@echo "build go program"
	@$(GO) build -o $(OUTPUTDIR)/$(BIN_NAME) $(MAIN_FILE)

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