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
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switauth/service/health"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

func TestNewServiceRegistrar(t *testing.T) {
	registrar := health.NewServiceRegistrar()
	assert.NotNil(t, registrar)
	assert.Equal(t, "health", registrar.GetName())
}

func TestServiceRegistrar_RegisterHTTP(t *testing.T) {
	// Initialize logger to prevent nil pointer panic
	logger.InitLogger()

	registrar := health.NewServiceRegistrar()
	router := gin.New()

	err := registrar.RegisterHTTP(router)
	require.NoError(t, err)

	// Test health check endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "switauth", response["service"])
	assert.NotZero(t, response["timestamp"])
}

func TestHealthServiceRegistrar_RegisterHTTP_Detailed(t *testing.T) {
	// Initialize logger to prevent nil pointer panic
	logger.InitLogger()

	registrar := health.NewServiceRegistrar()
	router := gin.New()

	err := registrar.RegisterHTTP(router)
	require.NoError(t, err)

	// Test detailed health check endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health/detailed", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "switauth", response["service"])
	assert.NotZero(t, response["timestamp"])
	assert.Contains(t, response, "checks")
}

func TestServiceRegistrar_RegisterGRPC(t *testing.T) {
	// Initialize logger to prevent nil pointer panic
	logger.InitLogger()

	registrar := health.NewServiceRegistrar()

	// Create test gRPC server
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	defer server.Stop()

	// Register health service
	err := registrar.RegisterGRPC(server)
	require.NoError(t, err)

	// Start server in background
	go func() {
		_ = server.Serve(lis)
	}()

	// Create client
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithInsecure())
	require.NoError(t, err)
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)

	// Test health check
	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
}

func TestHealthService_CheckHealth(t *testing.T) {
	service := health.NewHealthService()
	ctx := context.Background()

	status := service.CheckHealth(ctx)
	assert.NotNil(t, status)
	assert.Equal(t, "healthy", status.Status)
	assert.Equal(t, "switauth", status.Service)
	assert.Equal(t, "1.0.0", status.Version)
	assert.NotZero(t, status.Timestamp)
	assert.Contains(t, status.Details, "uptime")
}

func TestHealthService_GetHealthDetails(t *testing.T) {
	service := health.NewHealthService()
	ctx := context.Background()

	details := service.GetHealthDetails(ctx)
	assert.NotNil(t, details)
	assert.Equal(t, "switauth", details["service"])
	assert.Equal(t, "1.0.0", details["version"])
	assert.Equal(t, "healthy", details["status"])
	assert.NotZero(t, details["timestamp"])
	assert.Contains(t, details, "checks")

	checks := details["checks"].(map[string]interface{})
	assert.Equal(t, "ok", checks["database"])
	assert.Equal(t, "ok", checks["cache"])
	assert.Equal(t, "ok", checks["external"])
}

func TestHealthServiceIntegration(t *testing.T) {
	// Test full integration with HTTP server
	registrar := health.NewServiceRegistrar()
	router := gin.New()

	err := registrar.RegisterHTTP(router)
	require.NoError(t, err)

	// Create test server
	srv := httptest.NewServer(router)
	defer srv.Close()

	// Test health endpoint
	resp, err := http.Get(srv.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var healthResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&healthResp)
	require.NoError(t, err)

	assert.Equal(t, "healthy", healthResp["status"])
	assert.Equal(t, "switauth", healthResp["service"])
}

func TestHealthServiceRegistrar_RegisterHTTP_Routes(t *testing.T) {
	registrar := health.NewServiceRegistrar()
	router := gin.New()

	err := registrar.RegisterHTTP(router)
	require.NoError(t, err)

	// Verify routes are registered
	routes := router.Routes()
	assert.GreaterOrEqual(t, len(routes), 2)

	var foundHealth, foundDetailed bool
	for _, route := range routes {
		if route.Path == "/health" && route.Method == "GET" {
			foundHealth = true
		}
		if route.Path == "/health/detailed" && route.Method == "GET" {
			foundDetailed = true
		}
	}

	assert.True(t, foundHealth)
	assert.True(t, foundDetailed)
}

func TestStatus_JSON(t *testing.T) {
	status := &health.Status{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		Service:   "test-service",
		Version:   "1.0.0",
		Details: map[string]string{
			"uptime": "running",
		},
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)

	var decoded health.Status
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, status.Status, decoded.Status)
	assert.Equal(t, status.Service, decoded.Service)
	assert.Equal(t, status.Version, decoded.Version)
	assert.Equal(t, status.Details, decoded.Details)
}
