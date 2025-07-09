.DEFAULT_GOAL := all

# 导入所有子规则文件
include scripts/mk/variables.mk
include scripts/mk/build.mk
include scripts/mk/test.mk
include scripts/mk/docker.mk
include scripts/mk/swagger.mk
include scripts/mk/proto.mk
include scripts/mk/dev.mk
include scripts/mk/copyright.mk

# 主要组合目标（优化版，避免重复执行）
.PHONY: all
all: proto swagger tidy copyright build-fast

define USAGE_OPTIONS

Options:
  TIDY             Run go mod tidy.
  FORMAT           Format the code using gofmt.
  QUALITY          Run all quality checks (format, lint, vet).
  BUILD            Build the binaries, output binaries are in _output/{application_name}/ directory.
  CLEAN            Delete all generated code and build artifacts (binaries, protobuf, swagger, coverage, logs).
  CLEAN-BUILD      Delete build outputs only.
  CLEAN-PROTO      Delete protobuf generated code only.
  CLEAN-SWAGGER    Delete Swagger generated documentation only.
  CLEAN-TEST       Delete test artifacts (coverage files, logs) only.
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
  COPYRIGHT        Check and add/update copyright statements to Go files (excludes generated code).
  COPYRIGHT-CHECK  Check Go files for copyright statements.
  COPYRIGHT-ADD    Add copyright statements to Go files missing them.
  COPYRIGHT-UPDATE Update outdated copyright statements in Go files.
  COPYRIGHT-FORCE  Force update all Go files with current copyright statement.
  COPYRIGHT-DEBUG  Debug copyright hash comparison for troubleshooting.
  COPYRIGHT-FILES  Show which files are included/excluded in copyright management.
  PROTO            Generate protobuf code (format, lint, generate).
  PROTO-SETUP      Setup protobuf development environment.
  PROTO-GENERATE   Generate protobuf code only.
  PROTO-LINT       Lint protobuf files.
  PROTO-FORMAT     Format protobuf files.
  PROTO-BREAKING   Check for breaking changes in protobuf files.
  PROTO-CLEAN      Clean generated protobuf code.
  PROTO-DOCS       Generate OpenAPI documentation from protobuf.
  PROTO-VALIDATE   Validate protobuf configuration.
  BUF-INSTALL      Install Buf CLI tool.
endef
export USAGE_OPTIONS

.PHONY: help
help:
	@echo "$$USAGE_OPTIONS"

