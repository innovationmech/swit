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

package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter(adapter *HealthCheckAdapter, serviceName, version string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewEndpointHandler(adapter, serviceName, version)
	handler.RegisterRoutes(router)
	return router
}

func TestEndpointHandler_HealthCheck(t *testing.T) {
	tests := []struct {
		name               string
		checkers           []HealthChecker
		expectedStatusCode int
		expectHealthy      bool
	}{
		{
			name:               "healthy service",
			checkers:           []HealthChecker{&mockHealthChecker{name: "check1"}},
			expectedStatusCode: http.StatusOK,
			expectHealthy:      true,
		},
		{
			name: "unhealthy service",
			checkers: []HealthChecker{
				&mockHealthChecker{
					name: "check1",
					checkFunc: func(ctx context.Context) error {
						return errors.New("failed")
					},
				},
			},
			expectedStatusCode: http.StatusServiceUnavailable,
			expectHealthy:      false,
		},
		{
			name:               "no checkers",
			checkers:           []HealthChecker{},
			expectedStatusCode: http.StatusOK,
			expectHealthy:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewHealthCheckAdapter(nil)
			for _, checker := range tt.checkers {
				adapter.RegisterHealthChecker(checker)
			}

			router := setupTestRouter(adapter, "test-service", "v1.0.0")

			req, _ := http.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", tt.expectedStatusCode, w.Code)
			}

			var response HealthResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if response.Service != "test-service" {
				t.Errorf("expected service 'test-service', got %s", response.Service)
			}

			if response.Version != "v1.0.0" {
				t.Errorf("expected version 'v1.0.0', got %s", response.Version)
			}
		})
	}
}

func TestEndpointHandler_ReadinessCheck(t *testing.T) {
	tests := []struct {
		name               string
		checkers           []ReadinessChecker
		expectedStatusCode int
		expectReady        bool
	}{
		{
			name:               "ready service",
			checkers:           []ReadinessChecker{&mockReadinessChecker{name: "ready1"}},
			expectedStatusCode: http.StatusOK,
			expectReady:        true,
		},
		{
			name: "not ready service",
			checkers: []ReadinessChecker{
				&mockReadinessChecker{
					name: "ready1",
					checkFunc: func(ctx context.Context) error {
						return errors.New("not ready")
					},
				},
			},
			expectedStatusCode: http.StatusServiceUnavailable,
			expectReady:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewHealthCheckAdapter(nil)
			for _, checker := range tt.checkers {
				adapter.RegisterReadinessChecker(checker)
			}

			router := setupTestRouter(adapter, "test-service", "v1.0.0")

			// Test /health/ready endpoint
			req, _ := http.NewRequest("GET", "/health/ready", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", tt.expectedStatusCode, w.Code)
			}

			var response ReadinessResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if response.Ready != tt.expectReady {
				t.Errorf("expected ready %v, got %v", tt.expectReady, response.Ready)
			}
		})
	}
}

func TestEndpointHandler_LivenessCheck(t *testing.T) {
	tests := []struct {
		name               string
		checkers           []LivenessChecker
		expectedStatusCode int
		expectAlive        bool
	}{
		{
			name:               "alive service",
			checkers:           []LivenessChecker{&mockLivenessChecker{name: "live1"}},
			expectedStatusCode: http.StatusOK,
			expectAlive:        true,
		},
		{
			name: "not alive service",
			checkers: []LivenessChecker{
				&mockLivenessChecker{
					name: "live1",
					checkFunc: func(ctx context.Context) error {
						return errors.New("not alive")
					},
				},
			},
			expectedStatusCode: http.StatusServiceUnavailable,
			expectAlive:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewHealthCheckAdapter(nil)
			for _, checker := range tt.checkers {
				adapter.RegisterLivenessChecker(checker)
			}

			router := setupTestRouter(adapter, "test-service", "v1.0.0")

			// Test /health/live endpoint
			req, _ := http.NewRequest("GET", "/health/live", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", tt.expectedStatusCode, w.Code)
			}

			var response LivenessResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if response.Alive != tt.expectAlive {
				t.Errorf("expected alive %v, got %v", tt.expectAlive, response.Alive)
			}
		})
	}
}

func TestEndpointHandler_AlternativePaths(t *testing.T) {
	adapter := NewHealthCheckAdapter(nil)
	router := setupTestRouter(adapter, "test-service", "v1.0.0")

	tests := []struct {
		path               string
		expectedStatusCode int
	}{
		{path: "/health", expectedStatusCode: http.StatusOK},
		{path: "/health/ready", expectedStatusCode: http.StatusOK},
		{path: "/ready", expectedStatusCode: http.StatusOK},
		{path: "/health/live", expectedStatusCode: http.StatusOK},
		{path: "/live", expectedStatusCode: http.StatusOK},
		{path: "/health/detailed", expectedStatusCode: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatusCode {
				t.Errorf("path %s: expected status code %d, got %d", tt.path, tt.expectedStatusCode, w.Code)
			}
		})
	}
}

func TestEndpointHandler_DetailedHealthCheck(t *testing.T) {
	adapter := NewHealthCheckAdapter(nil)

	// Register various checkers
	adapter.RegisterHealthChecker(&mockHealthChecker{name: "health1"})
	adapter.RegisterReadinessChecker(&mockReadinessChecker{name: "ready1"})
	adapter.RegisterLivenessChecker(&mockLivenessChecker{name: "live1"})

	router := setupTestRouter(adapter, "test-service", "v1.0.0")

	req, _ := http.NewRequest("GET", "/health/detailed", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Check that response contains expected fields
	expectedFields := []string{"service", "version", "timestamp", "uptime", "health", "readiness", "liveness"}
	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("expected field '%s' in response", field)
		}
	}
}
