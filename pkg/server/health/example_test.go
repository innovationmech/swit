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

package health_test

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/server/health"
)

// Example_basicUsage demonstrates basic usage of the health check system
func Example_basicUsage() {
	// Create health check adapter with default configuration
	adapter := health.NewHealthCheckAdapter(nil)

	// Register a basic health checker
	dbChecker := health.NewBasicHealthChecker("database", func(ctx context.Context) error {
		// Simulate database ping
		return nil
	})
	adapter.RegisterHealthChecker(dbChecker)

	// Perform health check
	ctx := context.Background()
	status, err := adapter.CheckHealth(ctx)
	if err != nil {
		fmt.Printf("Health check error: %v\n", err)
		return
	}

	fmt.Printf("Health status: %s\n", status.Status)
	// Output: Health status: healthy
}

// Example_readinessProbe demonstrates readiness probe usage
func Example_readinessProbe() {
	// Create health check adapter
	adapter := health.NewHealthCheckAdapter(nil)

	// Create readiness probe with 100ms minimum start time
	readinessProbe := health.NewServiceReadinessProbe("my-service", 100*time.Millisecond)

	// Register the probe
	adapter.RegisterReadinessChecker(readinessProbe)

	// Initially not ready (before min start time and not marked ready)
	ctx := context.Background()
	err := readinessProbe.CheckReadiness(ctx)
	if err != nil {
		fmt.Println("Service not ready initially")
	}

	// Wait for minimum start time and mark as ready
	time.Sleep(150 * time.Millisecond)
	readinessProbe.SetReady(true)

	// Check readiness again
	err = readinessProbe.CheckReadiness(ctx)
	if err == nil {
		fmt.Println("Service is ready")
	}

	// Output:
	// Service not ready initially
	// Service is ready
}

// Example_livenessProbe demonstrates liveness probe usage with heartbeat
func Example_livenessProbe() {
	// Create liveness probe with 200ms timeout
	livenessProbe := health.NewServiceLivenessProbe("my-service", 200*time.Millisecond)

	// Start heartbeat with 50ms interval
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	livenessProbe.StartHeartbeat(ctx, 50*time.Millisecond)

	// Check liveness multiple times
	checkCtx := context.Background()
	for i := 0; i < 3; i++ {
		time.Sleep(100 * time.Millisecond)
		if err := livenessProbe.CheckLiveness(checkCtx); err == nil {
			fmt.Printf("Check %d: Service is alive\n", i+1)
		}
	}

	// Output:
	// Check 1: Service is alive
	// Check 2: Service is alive
	// Check 3: Service is alive
}

// Example_httpEndpoints demonstrates HTTP endpoint registration
func Example_httpEndpoints() {
	// Create health check adapter
	adapter := health.NewHealthCheckAdapter(nil)

	// Register health checkers
	checker := health.NewBasicHealthChecker("example", func(ctx context.Context) error {
		return nil
	})
	adapter.RegisterHealthChecker(checker)

	// Setup HTTP router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Register health check endpoints
	handler := health.NewEndpointHandler(adapter, "example-service", "v1.0.0")
	handler.RegisterRoutes(router)

	fmt.Println("Registered health check endpoints:")
	fmt.Println("- GET /health")
	fmt.Println("- GET /health/ready")
	fmt.Println("- GET /health/live")
	fmt.Println("- GET /health/detailed")

	// Output:
	// Registered health check endpoints:
	// - GET /health
	// - GET /health/ready
	// - GET /health/live
	// - GET /health/detailed
}

// Example_compositeChecker demonstrates composite health checker usage
func Example_compositeChecker() {
	// Create individual checkers
	dbChecker := health.NewBasicHealthChecker("database", func(ctx context.Context) error {
		return nil
	})

	redisChecker := health.NewBasicHealthChecker("redis", func(ctx context.Context) error {
		return nil
	})

	// Create composite checker
	compositeChecker := health.NewCompositeHealthChecker(
		"dependencies",
		dbChecker,
		redisChecker,
	)

	// Check all dependencies
	ctx := context.Background()
	if err := compositeChecker.Check(ctx); err == nil {
		fmt.Println("All dependencies are healthy")
	}

	// Output: All dependencies are healthy
}

// Example_customConfiguration demonstrates custom health check configuration
func Example_customConfiguration() {
	// Create custom configuration
	config := &health.HealthCheckConfig{
		Timeout:       5 * time.Second,
		Interval:      30 * time.Second,
		MaxConcurrent: 10,
	}

	// Create adapter with custom config
	adapter := health.NewHealthCheckAdapter(config)

	// Register checkers
	checker := health.NewBasicHealthChecker("service", func(ctx context.Context) error {
		// Simulate work
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	adapter.RegisterHealthChecker(checker)

	ctx := context.Background()
	status, _ := adapter.CheckHealth(ctx)

	fmt.Printf("Health check completed with status: %s\n", status.Status)
	// Output: Health check completed with status: healthy
}
