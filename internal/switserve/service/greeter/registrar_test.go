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

package greeter

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	greeterv1 "github.com/innovationmech/swit/internal/switserve/service/greeter/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func setupTest() {
	logger.InitLogger()
	gin.SetMode(gin.TestMode)
}

func TestNewServiceRegistrar(t *testing.T) {
	setupTest()

	// Create a mock service for testing
	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)

	assert.NotNil(t, registrar)
	assert.NotNil(t, registrar.httpHandler)
	assert.NotNil(t, registrar.grpcHandler)
}

func TestServiceRegistrar_GetName(t *testing.T) {
	setupTest()

	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)

	assert.Equal(t, "greeter", registrar.GetName())
}

func TestServiceRegistrar_RegisterGRPC(t *testing.T) {
	setupTest()

	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	server := grpc.NewServer()

	err := registrar.RegisterGRPC(server)

	assert.NoError(t, err)
	// Note: We can't easily test if the service is actually registered without
	// starting the server, but we can verify no error occurred
}

func TestServiceRegistrar_RegisterHTTP(t *testing.T) {
	setupTest()

	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()

	err := registrar.RegisterHTTP(router)

	assert.NoError(t, err)

	// Verify routes are registered by checking the router's routes
	routes := router.Routes()
	found := false
	for _, route := range routes {
		if route.Path == "/api/v1/greeter/hello" && route.Method == "POST" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected route /api/v1/greeter/hello POST should be registered")
}

func TestServiceRegistrar_sayHelloHTTP_Success(t *testing.T) {
	setupTest()

	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()
	registrar.RegisterHTTP(router)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "valid_request_english",
			requestBody: map[string]interface{}{
				"name": "World",
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Hello, World!",
		},
		{
			name: "valid_request_with_language",
			requestBody: map[string]interface{}{
				"name":     "世界",
				"language": "chinese",
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "你好, 世界！",
		},
		{
			name: "valid_request_spanish",
			requestBody: map[string]interface{}{
				"name":     "Mundo",
				"language": "spanish",
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "¡Hola, Mundo!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/greeter/hello", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Request-ID", "test-request-123")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedMsg, response["message"])

			// Check metadata
			metadata, ok := response["metadata"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "test-request-123", metadata["request_id"])
			assert.Equal(t, "swit-serve-1", metadata["server_id"])
		})
	}
}

func TestServiceRegistrar_sayHelloHTTP_BadRequest(t *testing.T) {
	setupTest()

	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()
	registrar.RegisterHTTP(router)

	tests := []struct {
		name        string
		requestBody interface{}
		contentType string
	}{
		{
			name:        "missing_name",
			requestBody: map[string]interface{}{"language": "english"},
			contentType: "application/json",
		},
		{
			name:        "empty_name",
			requestBody: map[string]interface{}{"name": ""},
			contentType: "application/json",
		},
		{
			name:        "invalid_json",
			requestBody: "invalid json",
			contentType: "application/json",
		},
		{
			name:        "nil_body",
			requestBody: nil,
			contentType: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.requestBody != nil {
				if str, ok := tt.requestBody.(string); ok {
					body = []byte(str)
				} else {
					body, _ = json.Marshal(tt.requestBody)
				}
			}

			req := httptest.NewRequest("POST", "/api/v1/greeter/hello", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response["error"], "invalid request body")
		})
	}
}

func TestServiceRegistrar_sayHelloHTTP_WithoutRequestID(t *testing.T) {
	setupTest()

	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()
	registrar.RegisterHTTP(router)

	requestBody := map[string]interface{}{
		"name": "TestUser",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/greeter/hello", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// No X-Request-ID header

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Hello, TestUser!", response["message"])

	// Check metadata - request_id should be empty when not provided
	metadata, ok := response["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "", metadata["request_id"])
	assert.Equal(t, "swit-serve-1", metadata["server_id"])
}

func TestServiceRegistrar_sayHelloHTTP_ContentTypeVariations(t *testing.T) {
	setupTest()

	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()
	registrar.RegisterHTTP(router)

	requestBody := map[string]interface{}{
		"name": "TestUser",
	}

	tests := []struct {
		name        string
		contentType string
		expectError bool
	}{
		{
			name:        "application_json",
			contentType: "application/json",
			expectError: false,
		},
		{
			name:        "application_json_charset",
			contentType: "application/json; charset=utf-8",
			expectError: false,
		},
		{
			name:        "no_content_type",
			contentType: "",
			expectError: false, // Gin accepts JSON without explicit content-type
		},
		{
			name:        "wrong_content_type",
			contentType: "text/plain",
			expectError: false, // Gin still parses JSON body regardless of content-type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(requestBody)
			req := httptest.NewRequest("POST", "/api/v1/greeter/hello", bytes.NewBuffer(body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}

func TestServiceRegistrar_sayHelloHTTP_SpecialCharacters(t *testing.T) {
	setupTest()

	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()
	registrar.RegisterHTTP(router)

	tests := []struct {
		name        string
		inputName   string
		expectedMsg string
	}{
		{
			name:        "unicode_characters",
			inputName:   "世界",
			expectedMsg: "Hello, 世界!",
		},
		{
			name:        "emoji",
			inputName:   "World 🌍",
			expectedMsg: "Hello, World 🌍!",
		},
		{
			name:        "special_symbols",
			inputName:   "Test@#$%",
			expectedMsg: "Hello, Test@#$%!",
		},
		{
			name:        "spaces_and_tabs",
			inputName:   "  John Doe  ",
			expectedMsg: "Hello, John Doe!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody := map[string]interface{}{
				"name": tt.inputName,
			}

			body, _ := json.Marshal(requestBody)
			req := httptest.NewRequest("POST", "/api/v1/greeter/hello", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedMsg, response["message"])
		})
	}
}

func TestServiceRegistrar_sayHelloHTTP_ConcurrentRequests(t *testing.T) {
	setupTest()

	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()
	registrar.RegisterHTTP(router)

	const numRequests = 50
	results := make([]int, numRequests)
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(index int) {
			defer func() { done <- true }()

			requestBody := map[string]interface{}{
				"name": "ConcurrentUser" + string(rune('A'+index%26)),
			}

			body, _ := json.Marshal(requestBody)
			req := httptest.NewRequest("POST", "/api/v1/greeter/hello", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			results[index] = w.Code
		}(i)
	}

	for i := 0; i < numRequests; i++ {
		<-done
	}

	for i, statusCode := range results {
		assert.Equal(t, http.StatusOK, statusCode, "Request %d should return 200", i)
	}
}

func TestServiceRegistrar_Integration(t *testing.T) {
	setupTest()

	// Test that both gRPC and HTTP registration work together
	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)

	// Test gRPC registration
	grpcServer := grpc.NewServer()
	err := registrar.RegisterGRPC(grpcServer)
	assert.NoError(t, err)

	// Test HTTP registration
	router := gin.New()
	err = registrar.RegisterHTTP(router)
	assert.NoError(t, err)

	// Test HTTP endpoint works after both registrations
	requestBody := map[string]interface{}{
		"name": "IntegrationTest",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/greeter/hello", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Hello, IntegrationTest!", response["message"])
}

// Benchmark tests
func BenchmarkServiceRegistrar_sayHelloHTTP(b *testing.B) {
	logger.InitLogger()
	gin.SetMode(gin.TestMode)

	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()
	registrar.RegisterHTTP(router)

	requestBody := map[string]interface{}{
		"name": "BenchmarkUser",
	}
	body, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/greeter/hello", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkServiceRegistrar_sayHelloHTTP_WithLanguage(b *testing.B) {
	logger.InitLogger()
	gin.SetMode(gin.TestMode)

	mockService := greeterv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()
	registrar.RegisterHTTP(router)

	requestBody := map[string]interface{}{
		"name":     "测试用户",
		"language": "chinese",
	}
	body, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/greeter/hello", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
