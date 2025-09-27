---
layout: home
title: Swit Go Microservice Framework
titleTemplate: Go Microservice Development Framework

hero:
  name: "Swit"
  text: "Go Microservice Framework"
  tagline: Production-ready microservice development foundation
  actions:
    - theme: brand
      text: Get Started
      link: /en/guide/getting-started
    - theme: alt
      text: View API
      link: /en/api/

features:
  - title: Unified Server Framework
    details: Complete server lifecycle management with transport coordination and health monitoring
    icon: 🚀
  - title: Multi-Transport Support
    details: Seamless HTTP and gRPC transport coordination with pluggable architecture
    icon: 🔄
  - title: Dependency Injection
    details: Factory-based dependency container with automatic lifecycle management
    icon: 📦
  - title: Performance Monitoring
    details: Built-in metrics collection and performance profiling with threshold monitoring
    icon: 📊
  - title: Service Discovery
    details: Consul-based service registration with health check integration
    icon: 🔍
  - title: Rich Examples
    details: Complete reference implementations and best practice examples
    icon: 📚
---

# Swit

## Project Status

<div class="project-badges">

[![CI](https://github.com/innovationmech/swit/workflows/CI/badge.svg)](https://github.com/innovationmech/swit/actions/workflows/ci.yml)
[![Security Checks](https://github.com/innovationmech/swit/workflows/Security%20Checks/badge.svg)](https://github.com/innovationmech/swit/actions/workflows/security-checks.yml)
[![codecov](https://codecov.io/gh/innovationmech/swit/branch/master/graph/badge.svg)](https://codecov.io/gh/innovationmech/swit)
[![Go Report Card](https://goreportcard.com/badge/github.com/innovationmech/swit)](https://goreportcard.com/report/github.com/innovationmech/swit)
[![Go Reference](https://pkg.go.dev/badge/github.com/innovationmech/swit.svg)](https://pkg.go.dev/github.com/innovationmech/swit)
[![GitHub release](https://img.shields.io/github/release/innovationmech/swit.svg)](https://github.com/innovationmech/swit/releases)
[![License](https://img.shields.io/github/license/innovationmech/swit.svg)](LICENSE)

</div>

<style>
.project-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 1rem 0;
}

.project-badges img {
  height: 20px;
}
</style>

![Go Version](https://img.shields.io/badge/go-%3E%3D1.23.12-blue.svg)



A comprehensive microservice framework for Go that provides a unified, production-ready foundation for building scalable microservices.

📖 **[Complete Documentation](https://innovationmech.github.io/swit/)** | [中文文档](https://innovationmech.github.io/swit/zh/)

## Key Features

- 🚀 **Complete Server Framework**: Unified HTTP and gRPC transport coordination
- 💉 **Dependency Injection**: Factory-based container with automatic lifecycle management
- 📊 **Performance Monitoring**: Built-in metrics collection and monitoring hooks
- 🔍 **Service Discovery**: Consul-based registration with health check integration
- 🛡️ **Middleware Stack**: Configurable CORS, rate limiting, authentication, and timeouts
- ⚡ **Protocol Buffers**: Full Buf toolchain support for API development
- 📱 **Example Services**: Complete reference implementations and usage patterns

## Architecture Overview

### Core Components
- **`pkg/server/`** - Base server framework and lifecycle management
- **`pkg/transport/`** - HTTP/gRPC transport coordination layer
- **`pkg/middleware/`** - Configurable middleware stack
- **`pkg/discovery/`** - Service discovery integration

### Example Services
- **`examples/`** - Simple examples for getting started
- **`internal/switserve/`** - User management reference service
- **`internal/switauth/`** - Authentication service with JWT
- **`internal/switctl/`** - CLI tool implementation

## Requirements

- **Go 1.23.12+** with generics support
- **Optional**: MySQL 8.0+, Redis 6.0+, Consul 1.12+ (for examples)
- **Development**: Buf CLI, Docker, Make

## Quick Start

### 1. Get the Framework
```bash {1-10}
git clone https://github.com/innovationmech/swit.git
cd swit
go mod download
```

### 2. Create a Simple Service
```go {1-10}
package main

import (
    "context"
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/server"
)

type MyService struct{}

func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
    return registry.RegisterBusinessHTTPHandler(&MyHTTPHandler{})
}

type MyHTTPHandler struct{}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    ginRouter.GET("/hello", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "Hello from Swit!"})
    })
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return "my-service"
}

func main() {
    config := &server.ServerConfig{
        ServiceName: "my-service",
        HTTP: server.HTTPConfig{Port: "8080", Enabled: true},
        GRPC: server.GRPCConfig{Enabled: false},
    }
    
    service := &MyService{}
    baseServer, _ := server.NewBusinessServerCore(config, service, nil)
    
    ctx := context.Background()
    baseServer.Start(ctx)
    defer baseServer.Shutdown()
    
    select {} // Keep running
}
```

### 3. Run Your Service
```bash {1-10}
go run main.go
curl http://localhost:8080/hello
```

## Examples

### Simple Examples
```bash {1-10}
# HTTP service
cd examples/simple-http-service && go run main.go

# gRPC service  
cd examples/grpc-service && go run main.go

# Full-featured service
cd examples/full-featured-service && go run main.go
```

### Reference Services
```bash {1-10}
# Build all services
make build

# Run services
./bin/swit-serve    # User management (HTTP: 9000, gRPC: 10000)
./bin/swit-auth     # Authentication (HTTP: 9001, gRPC: 50051)
./bin/switctl --help # CLI tool
```

## Development

### Setup Development Environment
```bash {1-10}
# Complete setup
make setup-dev

# Quick setup
make setup-quick
```

### Common Commands
```bash {1-10}
# Build
make build          # Full build
make build-dev      # Quick build

# Test
make test           # All tests
make test-dev       # Quick tests
make test-coverage  # Coverage report

# API Development
make proto          # Generate protobuf code
make swagger        # Generate API docs

# Code Quality
make tidy           # Tidy modules
make format         # Format code
make quality        # Quality checks
```

## Docker Deployment

```bash {1-10}
# Build images
make docker

# Run with Docker Compose
docker-compose up -d

# Or run individually
docker run -p 9000:9000 -p 10000:10000 swit-serve:latest
docker run -p 9001:9001 swit-auth:latest
```

## License

MIT License - See [LICENSE](https://github.com/innovationmech/swit/blob/master/LICENSE) file for details

---

**For complete documentation, examples, and advanced usage, visit [innovationmech.github.io/swit](https://innovationmech.github.io/swit/)**