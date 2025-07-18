# 变量定义
# 优先使用git获取项目根目录，如果失败则使用Makefile位置
PROJECT_ROOT := $(shell git rev-parse --show-toplevel 2>/dev/null || echo "$(MAKEFILE_DIR)" | sed 's/\/$$//g')
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

# Protobuf 相关变量
BUF := buf
BUF_VERSION := v1.28.1
API_DIR := api
PROTO_DIR := $(API_DIR)/proto
GEN_DIR := $(API_DIR)/gen 