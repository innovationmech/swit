# Getting Started

Welcome to the Swit Go Microservice Framework! This guide will help you get up and running quickly with your first microservice.

## Requirements

- **Go 1.24+** - Modern Go version with generics support
- **Git** - For framework and example code management

## Installation

First, make sure you have the required Go version installed.

```bash
go get github.com/innovationmech/swit
```

## Create Your First Service

Create a new Go project and initialize the module:

```bash
mkdir my-service
cd my-service
go mod init my-service
```

## Basic Example

Create a simple HTTP service:

```go
package main

import (
    "context"
    "log"
    
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/server"
)

type MyService struct {
    // Service fields
}

func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Register HTTP handlers
    registry.RegisterBusinessHTTPHandler(&MyHTTPHandler{})
    return nil
}

type MyHTTPHandler struct{}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    r := router.(*gin.Engine)
    r.GET("/health", h.Health)
    r.GET("/api/v1/hello", h.Hello)
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return "my-service"
}

func (h *MyHTTPHandler) Health(c *gin.Context) {
    c.JSON(200, gin.H{"status": "healthy"})
}

func (h *MyHTTPHandler) Hello(c *gin.Context) {
    c.JSON(200, gin.H{"message": "Hello from Swit!"})
}

func main() {
    config := &server.ServerConfig{
        Name:    "my-service",
        Version: "1.0.0",
        HTTP: server.HTTPConfig{
            Port:     8080,
            Enabled:  true,
        },
    }
    
    service := &MyService{}
    
    srv, err := server.NewBusinessServerCore(config, service, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    if err := srv.Start(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## Run the Service

```bash
go run main.go
```

Your service is now running at `http://localhost:8080`. You can test it with:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/hello
```

## Next Steps

- Learn how to [configure the server](./server-config)
- Learn to [create gRPC services](./grpc-service)
- Explore [middleware](./middleware) capabilities
- Check out [complete examples](/en/examples/full-featured)