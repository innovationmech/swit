# 变量定义
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