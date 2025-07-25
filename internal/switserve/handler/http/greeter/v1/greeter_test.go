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

package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switserve/interfaces"
	"github.com/innovationmech/swit/internal/switserve/types"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// MockGreeterService is a mock implementation of GreeterService for testing
type MockGreeterService struct {
	mock.Mock
}

func (m *MockGreeterService) GenerateGreeting(ctx context.Context, name, language string) (string, error) {
	args := m.Called(ctx, name, language)
	return args.String(0), args.Error(1)
}

// Test helper functions
func init() {
	// Initialize logger for tests
	logger.InitLogger()
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func createTestHandler(service interfaces.GreeterService) *GreeterHandler {
	return NewGreeterHandler(service)
}

// TestNewGreeterHandler tests the constructor
func TestNewGreeterHandler(t *testing.T) {
	tests := []struct {
		name        string
		service     interfaces.GreeterService
		description string
	}{
		{
			name:        "success_create_handler",
			service:     &MockGreeterService{},
			description: "Should create handler successfully with valid service",
		},
		{
			name:        "with_nil_service",
			service:     nil,
			description: "Should create handler even with nil service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewGreeterHandler(tt.service)

			assert.NotNil(t, handler)
			assert.Equal(t, tt.service, handler.service)
			assert.WithinDuration(t, time.Now(), handler.startTime, time.Second)
		})
	}
}

// TestGreeterHandler_SayHello tests the SayHello HTTP handler
func TestGreeterHandler_SayHello(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockGreeterService)
		expectedStatus int
		expectedError  string
		description    string
	}{
		{
			name: "success_with_name_only",
			requestBody: map[string]interface{}{
				"name": "John",
			},
			mockSetup: func(m *MockGreeterService) {
				m.On("GenerateGreeting", mock.Anything, "John", "").Return("Hello, John!", nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Should generate greeting successfully with name only",
		},
		{
			name: "success_with_name_and_language",
			requestBody: map[string]interface{}{
				"name":     "John",
				"language": "es",
			},
			mockSetup: func(m *MockGreeterService) {
				m.On("GenerateGreeting", mock.Anything, "John", "es").Return("¡Hola, John!", nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Should generate greeting successfully with name and language",
		},
		{
			name:           "invalid_json",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockGreeterService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
			description:    "Should return error for invalid JSON",
		},
		{
			name: "missing_name",
			requestBody: map[string]interface{}{
				"language": "en",
			},
			mockSetup:      func(m *MockGreeterService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
			description:    "Should return error when name is missing",
		},
		{
			name: "empty_name",
			requestBody: map[string]interface{}{
				"name": "",
			},
			mockSetup:      func(m *MockGreeterService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
			description:    "Should return error when name is empty",
		},
		{
			name: "service_error",
			requestBody: map[string]interface{}{
				"name": "John",
			},
			mockSetup: func(m *MockGreeterService) {
				m.On("GenerateGreeting", mock.Anything, "John", "").Return("", errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal server error",
			description:    "Should return error when service fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockGreeterService)
			tt.mockSetup(mockService)
			handler := createTestHandler(mockService)
			router := setupTestRouter()
			router.POST("/greet", handler.SayHello)

			// Create request
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/greet", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Request-ID", "test-request-123")
			w := httptest.NewRecorder()

			// Execute
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			} else {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "message")
				assert.Contains(t, response, "metadata")

				// Check metadata
				metadata := response["metadata"].(map[string]interface{})
				assert.Equal(t, "test-request-123", metadata["request_id"])
				assert.Equal(t, "swit-serve-1", metadata["server_id"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestGreeterHandler_RegisterHTTP tests HTTP route registration
func TestGreeterHandler_RegisterHTTP(t *testing.T) {
	tests := []struct {
		name        string
		service     interfaces.GreeterService
		description string
	}{
		{
			name:        "success_register_routes",
			service:     &MockGreeterService{},
			description: "Should register HTTP routes successfully",
		},
		{
			name:        "with_nil_service",
			service:     nil,
			description: "Should register routes even with nil service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createTestHandler(tt.service)
			router := setupTestRouter()

			err := handler.RegisterHTTP(router)

			assert.NoError(t, err)
			// Verify route is registered by making a test request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/greet", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			// Should not return 404 (route exists)
			assert.NotEqual(t, http.StatusNotFound, w.Code)
		})
	}
}

// TestGreeterHandler_RegisterGRPC tests gRPC service registration
func TestGreeterHandler_RegisterGRPC(t *testing.T) {
	tests := []struct {
		name        string
		service     interfaces.GreeterService
		description string
	}{
		{
			name:        "success_register_grpc",
			service:     &MockGreeterService{},
			description: "Should register gRPC service successfully",
		},
		{
			name:        "with_nil_service",
			service:     nil,
			description: "Should register gRPC service even with nil service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createTestHandler(tt.service)
			server := grpc.NewServer()

			err := handler.RegisterGRPC(server)

			assert.NoError(t, err)
		})
	}
}

// TestGreeterHandler_GetMetadata tests service metadata retrieval
func TestGreeterHandler_GetMetadata(t *testing.T) {
	tests := []struct {
		name        string
		service     interfaces.GreeterService
		description string
	}{
		{
			name:        "success_get_metadata",
			service:     &MockGreeterService{},
			description: "Should return correct metadata",
		},
		{
			name:        "with_nil_service",
			service:     nil,
			description: "Should return metadata even with nil service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createTestHandler(tt.service)

			metadata := handler.GetMetadata()

			assert.NotNil(t, metadata)
			assert.Equal(t, "greeter", metadata.Name)
			assert.Equal(t, "v1", metadata.Version)
			assert.Equal(t, "Greeter service providing greeting functionality", metadata.Description)
			assert.Equal(t, "/api/v1/health", metadata.HealthEndpoint)
			assert.Contains(t, metadata.Tags, "greeter")
			assert.Contains(t, metadata.Tags, "messaging")
			assert.Contains(t, metadata.Tags, "api")
			assert.Empty(t, metadata.Dependencies)
		})
	}
}

// TestGreeterHandler_GetHealthEndpoint tests health endpoint retrieval
func TestGreeterHandler_GetHealthEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		service     interfaces.GreeterService
		expected    string
		description string
	}{
		{
			name:        "success_get_health_endpoint",
			service:     &MockGreeterService{},
			expected:    "/api/v1/health",
			description: "Should return correct health endpoint",
		},
		{
			name:        "with_nil_service",
			service:     nil,
			expected:    "/api/v1/health",
			description: "Should return health endpoint even with nil service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createTestHandler(tt.service)

			endpoint := handler.GetHealthEndpoint()

			assert.Equal(t, tt.expected, endpoint)
		})
	}
}

// TestGreeterHandler_IsHealthy tests health status check
func TestGreeterHandler_IsHealthy(t *testing.T) {
	tests := []struct {
		name           string
		service        interfaces.GreeterService
		expectedStatus string
		description    string
	}{
		{
			name:           "healthy_with_service",
			service:        &MockGreeterService{},
			expectedStatus: types.HealthStatusHealthy,
			description:    "Should return healthy status when service is available",
		},
		{
			name:           "unhealthy_with_nil_service",
			service:        nil,
			expectedStatus: types.HealthStatusUnhealthy,
			description:    "Should return unhealthy status when service is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createTestHandler(tt.service)
			ctx := context.Background()

			status, err := handler.IsHealthy(ctx)

			assert.NoError(t, err)
			assert.NotNil(t, status)
			assert.Equal(t, tt.expectedStatus, status.Status)
			assert.Equal(t, "v1", status.Version)
			assert.GreaterOrEqual(t, status.Uptime, time.Duration(0))
			assert.WithinDuration(t, time.Now(), status.Timestamp, time.Second)
		})
	}
}

// TestGreeterHandler_Initialize tests service initialization
func TestGreeterHandler_Initialize(t *testing.T) {
	tests := []struct {
		name        string
		service     interfaces.GreeterService
		description string
	}{
		{
			name:        "success_initialize",
			service:     &MockGreeterService{},
			description: "Should initialize successfully",
		},
		{
			name:        "with_nil_service",
			service:     nil,
			description: "Should initialize even with nil service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createTestHandler(tt.service)
			ctx := context.Background()

			err := handler.Initialize(ctx)

			assert.NoError(t, err)
		})
	}
}

// TestGreeterHandler_Shutdown tests service shutdown
func TestGreeterHandler_Shutdown(t *testing.T) {
	tests := []struct {
		name        string
		service     interfaces.GreeterService
		description string
	}{
		{
			name:        "success_shutdown",
			service:     &MockGreeterService{},
			description: "Should shutdown successfully",
		},
		{
			name:        "with_nil_service",
			service:     nil,
			description: "Should shutdown even with nil service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createTestHandler(tt.service)
			ctx := context.Background()

			err := handler.Shutdown(ctx)

			assert.NoError(t, err)
		})
	}
}

// TestGreeterHandler_Integration tests complete workflow
func TestGreeterHandler_Integration(t *testing.T) {
	mockService := new(MockGreeterService)
	mockService.On("GenerateGreeting", mock.Anything, "Alice", "fr").Return("Bonjour, Alice!", nil)

	handler := createTestHandler(mockService)
	ctx := context.Background()

	// Test initialization
	err := handler.Initialize(ctx)
	assert.NoError(t, err)

	// Test metadata
	metadata := handler.GetMetadata()
	assert.Equal(t, "greeter", metadata.Name)

	// Test health check
	status, err := handler.IsHealthy(ctx)
	assert.NoError(t, err)
	assert.Equal(t, types.HealthStatusHealthy, status.Status)

	// Test HTTP registration and request
	router := setupTestRouter()
	err = handler.RegisterHTTP(router)
	assert.NoError(t, err)

	requestBody := map[string]interface{}{
		"name":     "Alice",
		"language": "fr",
	}
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/greet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Bonjour, Alice!", response["message"])

	// Test shutdown
	err = handler.Shutdown(ctx)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

// Benchmark tests
func BenchmarkGreeterHandler_SayHello(b *testing.B) {
	mockService := new(MockGreeterService)
	mockService.On("GenerateGreeting", mock.Anything, "John", "").Return("Hello, John!", nil)

	handler := createTestHandler(mockService)
	router := setupTestRouter()
	router.POST("/greet", handler.SayHello)

	requestBody := map[string]interface{}{"name": "John"}
	body, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/greet", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkGreeterHandler_GetMetadata(b *testing.B) {
	handler := createTestHandler(&MockGreeterService{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.GetMetadata()
	}
}

func BenchmarkGreeterHandler_IsHealthy(b *testing.B) {
	handler := createTestHandler(&MockGreeterService{})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.IsHealthy(ctx)
	}
}
