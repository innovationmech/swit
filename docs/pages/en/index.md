---
layout: home

title: Swit
titleTemplate: Modern Go Microservice Framework

hero:
  name: Swit
  text: Modern Go Microservice Framework
  tagline: A comprehensive, production-ready foundation for building scalable microservices
  actions:
    - theme: brand
      text: Get Started
      link: /en/guide/getting-started
    - theme: alt
      text: View Examples
      link: /en/examples/
    - theme: alt
      text: API Documentation
      link: /en/api/
  image:
    src: /images/hero-logo.svg
    alt: Swit Framework Logo

features:
  - icon: 🚀
    title: Rapid Development
    details: Complete server framework with unified transport layer supporting HTTP and gRPC protocols, letting you focus on business logic
    link: /en/guide/getting-started

  - icon: ⚡
    title: High Performance
    details: Built-in performance monitoring, optimized concurrency handling, and resource management for production-grade performance
    link: /en/guide/performance

  - icon: 🔧
    title: Dependency Injection
    details: Factory-based dependency container with singleton and transient support, automatic lifecycle management
    link: /en/guide/dependency-injection

  - icon: 🌐
    title: Service Discovery
    details: Consul-based service registration and discovery with health checks and automatic deregistration
    link: /en/guide/service-discovery

  - icon: 🛡️
    title: Middleware Framework
    details: Configurable middleware stack with CORS, rate limiting, timeout, and authentication support
    link: /en/guide/middleware

  - icon: 📊
    title: Monitoring & Health Checks
    details: Comprehensive health monitoring with service aggregation, timeout handling, and built-in performance metrics collection
    link: /en/guide/monitoring

---

## Quick Experience

Create your first microservice with Swit framework in just minutes:

```go
// main.go
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
    ginRouter.GET("/hello", h.handleHello)
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return "my-service"
}

func (h *MyHTTPHandler) handleHello(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "Hello from Swit framework!"})
}

func main() {
    config := &server.ServerConfig{
        ServiceName: "my-service",
        HTTP: server.HTTPConfig{Port: "8080", Enabled: true},
    }
    
    service := &MyService{}
    baseServer, _ := server.NewBusinessServerCore(config, service, nil)
    
    ctx := context.Background()
    baseServer.Start(ctx)
    defer baseServer.Shutdown()
    
    select {} // Keep running
}
```

```bash
go run main.go
curl http://localhost:8080/hello
```

## Framework Features

### Core Server Framework

- **BusinessServerCore**: Complete server lifecycle management
- **BusinessServiceRegistrar**: Unified service registration patterns
- **BusinessDependencyContainer**: Dependency injection and lifecycle management

### Unified Transport Layer

- **TransportCoordinator**: HTTP and gRPC transport coordinator
- **NetworkTransport**: Pluggable transport architecture
- **MultiTransportRegistry**: Cross-transport service registration management

### Complete Example Services

- **switserve**: User management service showcasing complete framework integration
- **switauth**: Authentication service with JWT integration and security patterns
- **switctl**: Command-line tool demonstrating framework integration patterns

## Development Tools and Commands

### Framework Development Commands
```bash
make build          # Build framework components and example services
make test           # Run complete test suite
make proto          # Generate Protocol Buffer code
make swagger        # Generate Swagger documentation
```

### Quick Start Commands
```bash
make setup-dev      # Setup complete development environment
make build-dev      # Quick build (skip quality checks)
make test-dev       # Quick testing
```

## Project Statistics

<div class="stats">
  <div class="stat">
    <div class="stat-number">⭐</div>
    <div class="stat-label">GitHub Stars</div>
  </div>
  <div class="stat">
    <div class="stat-number">v1.0.0</div>
    <div class="stat-label">Latest Version</div>
  </div>
  <div class="stat">
    <div class="stat-number">MIT</div>
    <div class="stat-label">Open Source License</div>
  </div>
  <div class="stat">
    <div class="stat-number">Go 1.24+</div>
    <div class="stat-label">Language Version</div>
  </div>
</div>

## Why Choose Swit?

### 🎯 Production-Ready Design
Swit framework is built from real project experience, providing all the features needed for production environments including monitoring, health checks, graceful shutdown, and error handling.

### 🔌 Pluggable Architecture
Flexible interface design allows you to easily extend functionality, replace transport layers, or add custom middleware without modifying the core framework.

### 📈 Performance First
Built-in performance monitoring and optimization, supporting high concurrency scenarios with detailed performance metrics and bottleneck analysis.

### 👥 Developer Friendly
Complete example code, detailed documentation, and active community support help you get started quickly and continue improving.

## Community and Support

- **GitHub Repository**: [innovationmech/swit](https://github.com/innovationmech/swit)
- **Issue Tracking**: [GitHub Issues](https://github.com/innovationmech/swit/issues)
- **Contributing Guide**: [How to Contribute](https://github.com/innovationmech/swit/blob/master/CONTRIBUTING.md)
- **Code of Conduct**: [Community Guidelines](https://github.com/innovationmech/swit/blob/master/CODE_OF_CONDUCT.md)

Start your microservice development journey today!

<style>
.stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 1rem;
  margin: 2rem 0;
}

.stat {
  text-align: center;
  padding: 1rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  background: var(--vp-c-bg-soft);
}

.stat-number {
  font-size: 2rem;
  font-weight: bold;
  color: var(--vp-c-brand-1);
}

.stat-label {
  font-size: 0.875rem;
  color: var(--vp-c-text-2);
  margin-top: 0.5rem;
}

@media (max-width: 768px) {
  .stats {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>