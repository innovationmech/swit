// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/types"
)

// DemoServiceHandler implements a simple service for the performance demo
type DemoServiceHandler struct {
	name string
}

func NewDemoServiceHandler() *DemoServiceHandler {
	return &DemoServiceHandler{
		name: "demo-service",
	}
}

func (d *DemoServiceHandler) RegisterHTTP(router gin.IRouter) error {
	// Register a simple route with simulated delay
	router.GET("/hello", func(c *gin.Context) {
		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello from performance demo!",
			"time":    time.Now().Format(time.RFC3339),
		})
	})
	return nil
}

func (d *DemoServiceHandler) RegisterGRPC(server interface{}) error {
	// No gRPC services for this demo
	return nil
}

func (d *DemoServiceHandler) GetMetadata() interface{} {
	return map[string]string{
		"name":    d.name,
		"version": "1.0.0",
	}
}

func (d *DemoServiceHandler) GetHealthEndpoint() string {
	return "/health/demo"
}

func (d *DemoServiceHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	return &types.HealthStatus{
		Status:  "healthy",
		Version: "1.0.0",
		Uptime:  time.Duration(0),
	}, nil
}

func (d *DemoServiceHandler) Initialize(ctx context.Context) error {
	return nil
}

func (d *DemoServiceHandler) Shutdown(ctx context.Context) error {
	return nil
}

// DemoServiceRegistrar implements ServiceRegistrar
type DemoServiceRegistrar struct {
	services []interface{}
}

func NewDemoServiceRegistrar() *DemoServiceRegistrar {
	return &DemoServiceRegistrar{
		services: make([]interface{}, 0),
	}
}

func (r *DemoServiceRegistrar) AddService(service interface{}) {
	r.services = append(r.services, service)
}

func (r *DemoServiceRegistrar) RegisterServices(registry server.ServiceRegistry) error {
	for _, service := range r.services {
		if httpHandler, ok := service.(server.HTTPHandler); ok {
			if err := registry.RegisterHTTPHandler(httpHandler); err != nil {
				return err
			}
		}
	}
	return nil
}

// HTTPHandlerAdapter adapts our service to the HTTPHandler interface
type HTTPHandlerAdapter struct {
	service *DemoServiceHandler
}

func (h *HTTPHandlerAdapter) RegisterRoutes(router gin.IRouter) error {
	return h.service.RegisterHTTP(router)
}

func (h *HTTPHandlerAdapter) GetServiceName() string {
	return h.service.name
}

func (h *HTTPHandlerAdapter) GetVersion() string {
	return "1.0.0"
}

func main() {
	// Create server configuration
	config := server.NewServerConfig()
	config.ServiceName = "performance-demo"
	config.HTTPPort = "8080"
	config.EnableHTTP = true
	config.SetDefaults()

	// Create service registrar
	registrar := NewDemoServiceRegistrar()

	// Create and register demo service
	demoService := NewDemoServiceHandler()
	httpAdapter := &HTTPHandlerAdapter{service: demoService}
	registrar.AddService(httpAdapter)

	// Create server
	baseServer, err := server.NewServer(config, registrar, nil)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start the server
	ctx := context.Background()
	if err := baseServer.Start(ctx); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	fmt.Println("Performance demo server started on :8080")
	fmt.Println("Try:")
	fmt.Println("  curl http://localhost:8080/hello")
	fmt.Println("  curl http://localhost:8080/metrics")
	fmt.Println("  curl http://localhost:8080/health")

	// Keep the server running
	select {}
}
