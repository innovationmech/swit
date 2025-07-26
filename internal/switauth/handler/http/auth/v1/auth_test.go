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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/innovationmech/swit/internal/switauth/interfaces"
	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Ensure MockAuthService implements interfaces.AuthService
var _ interfaces.AuthService = (*MockAuthService)(nil)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Login(ctx context.Context, username, password string) (*types.AuthResponse, error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.AuthResponse), args.Error(1)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*types.AuthResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.AuthResponse), args.Error(1)
}

func (m *MockAuthService) ValidateToken(ctx context.Context, tokenString string) (*model.Token, error) {
	args := m.Called(ctx, tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Token), args.Error(1)
}

func (m *MockAuthService) Logout(ctx context.Context, tokenString string) error {
	args := m.Called(ctx, tokenString)
	return args.Error(0)
}

func TestNewAuthController(t *testing.T) {
	tests := []struct {
		name        string
		authService interfaces.AuthService
		expected    *AuthHandler
	}{
		{
			name:        "Create controller with valid auth service",
			authService: &MockAuthService{},
			expected: &AuthHandler{
				authService: &MockAuthService{},
			},
		},
		{
			name:        "Create controller with nil auth service",
			authService: nil,
			expected: &AuthHandler{
				authService: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := NewAuthHandler(tt.authService)

			assert.NotNil(t, controller)
			assert.IsType(t, &AuthHandler{}, controller)

			if tt.authService != nil {
				assert.NotNil(t, controller.authService)
				assert.IsType(t, tt.authService, controller.authService)
			} else {
				assert.Nil(t, controller.authService)
			}
		})
	}
}

func TestControllerStruct(t *testing.T) {
	mockAuthService := &MockAuthService{}
	controller := &AuthHandler{authService: mockAuthService}

	assert.NotNil(t, controller)
	assert.Equal(t, mockAuthService, controller.authService)
}

func TestControllerWithMockService(t *testing.T) {
	mockAuthService := &MockAuthService{}
	controller := NewAuthHandler(mockAuthService)

	assert.NotNil(t, controller)
	assert.Equal(t, mockAuthService, controller.authService)

	mockAuthService.AssertExpectations(t)
}

func TestControllerFieldAccess(t *testing.T) {
	mockAuthService := &MockAuthService{}
	controller := NewAuthHandler(mockAuthService)

	assert.NotNil(t, controller.authService)

	_, ok := controller.authService.(interfaces.AuthService)
	assert.True(t, ok, "authService should implement AuthService interface")
}

func BenchmarkNewAuthController(b *testing.B) {
	mockAuthService := &MockAuthService{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewAuthHandler(mockAuthService)
	}
}

func TestControllerInterfaceCompliance(t *testing.T) {
	mockAuthService := &MockAuthService{}
	controller := NewAuthHandler(mockAuthService)

	assert.Implements(t, (*interfaces.AuthService)(nil), controller.authService)
}

// Login Tests

func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "success_login",
			requestBody: map[string]string{
				"username": "testuser",
				"password": "password123",
			},
			setupMocks: func(mockSrv *MockAuthService) {
				response := &types.AuthResponse{
					AccessToken:  "access_token_123",
					RefreshToken: "refresh_token_456",
				}
				mockSrv.On("Login", mock.Anything, "testuser", "password123").Return(response, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"access_token":  "access_token_123",
				"refresh_token": "refresh_token_456",
			},
		},
		{
			name: "error_invalid_json",
			requestBody: `{
				"username": "testuser",
				"password": 
			}`,
			setupMocks:     func(mockSrv *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"message": "Invalid request format",
			},
		},
		{
			name: "error_invalid_credentials",
			requestBody: map[string]string{
				"username": "wronguser",
				"password": "wrongpassword",
			},
			setupMocks: func(mockSrv *MockAuthService) {
				mockSrv.On("Login", mock.Anything, "wronguser", "wrongpassword").Return(
					(*types.AuthResponse)(nil), errors.New("invalid credentials"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"message": "Authentication failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &MockAuthService{}
			tt.setupMocks(mockSrv)

			controller := &AuthHandler{
				authService: mockSrv,
			}

			router := gin.New()
			router.POST("/auth/login", controller.Login)

			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req, err := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			for key, expectedValue := range tt.expectedBody {
				if expectedValue == mock.Anything {
					assert.Contains(t, response, key)
				} else {
					assert.Equal(t, expectedValue, response[key])
				}
			}

			mockSrv.AssertExpectations(t)
		})
	}
}

// Logout Tests

func TestLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		authHeader     string
		setupMocks     func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:       "success_logout",
			authHeader: "Bearer access_token_123",
			setupMocks: func(mockSrv *MockAuthService) {
				mockSrv.On("Logout", mock.Anything, "Bearer access_token_123").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "Logged out successfully",
			},
		},
		{
			name:           "error_missing_authorization_header",
			authHeader:     "",
			setupMocks:     func(mockSrv *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"message": "Authorization header is missing",
			},
		},
		{
			name:       "error_service_failure",
			authHeader: "Bearer invalid_token",
			setupMocks: func(mockSrv *MockAuthService) {
				mockSrv.On("Logout", mock.Anything, "Bearer invalid_token").Return(
					errors.New("token not found"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"message": "Logout failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &MockAuthService{}
			tt.setupMocks(mockSrv)

			controller := &AuthHandler{
				authService: mockSrv,
			}

			router := gin.New()
			router.POST("/auth/logout", controller.Logout)

			req, err := http.NewRequest("POST", "/auth/logout", nil)
			assert.NoError(t, err)

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			for key, expectedValue := range tt.expectedBody {
				assert.Equal(t, expectedValue, response[key])
			}

			mockSrv.AssertExpectations(t)
		})
	}
}

// RefreshToken Tests

func TestRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "success_refresh_token",
			requestBody: map[string]string{
				"refresh_token": "valid_refresh_token",
			},
			setupMocks: func(mockSrv *MockAuthService) {
				response := &types.AuthResponse{
					AccessToken:  "new_access_token",
					RefreshToken: "new_refresh_token",
				}
				mockSrv.On("RefreshToken", mock.Anything, "valid_refresh_token").Return(response, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"access_token":  "new_access_token",
				"refresh_token": "new_refresh_token",
			},
		},
		{
			name: "error_invalid_json",
			requestBody: `{
				"refresh_token": 
			}`,
			setupMocks:     func(mockSrv *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"message": "Invalid request format",
			},
		},
		{
			name: "error_invalid_refresh_token",
			requestBody: map[string]string{
				"refresh_token": "invalid_refresh_token",
			},
			setupMocks: func(mockSrv *MockAuthService) {
				mockSrv.On("RefreshToken", mock.Anything, "invalid_refresh_token").Return(
					(*types.AuthResponse)(nil), errors.New("invalid refresh token"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"message": "Token refresh failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &MockAuthService{}
			tt.setupMocks(mockSrv)

			controller := &AuthHandler{
				authService: mockSrv,
			}

			router := gin.New()
			router.POST("/auth/refresh", controller.RefreshToken)

			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req, err := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(body))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			for key, expectedValue := range tt.expectedBody {
				if expectedValue == mock.Anything {
					assert.Contains(t, response, key)
				} else {
					assert.Equal(t, expectedValue, response[key])
				}
			}

			mockSrv.AssertExpectations(t)
		})
	}
}

// ValidateToken Tests

func TestValidateToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		authHeader     string
		setupMocks     func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:       "success_validate_token",
			authHeader: "Bearer valid_access_token",
			setupMocks: func(mockSrv *MockAuthService) {
				userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				token := &model.Token{
					UserID: userID,
				}
				mockSrv.On("ValidateToken", mock.Anything, "valid_access_token").Return(token, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "Token is valid",
				"user_id": "550e8400-e29b-41d4-a716-446655440000",
			},
		},
		{
			name:           "error_missing_authorization_header",
			authHeader:     "",
			setupMocks:     func(mockSrv *MockAuthService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"message": "Missing Authorization header",
			},
		},
		{
			name:           "error_invalid_authorization_header_format",
			authHeader:     "InvalidFormat",
			setupMocks:     func(mockSrv *MockAuthService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"message": "Invalid Authorization header format",
			},
		},
		{
			name:       "error_invalid_token",
			authHeader: "Bearer invalid_token",
			setupMocks: func(mockSrv *MockAuthService) {
				mockSrv.On("ValidateToken", mock.Anything, "invalid_token").Return(nil, errors.New("invalid token"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"message": "Token validation failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &MockAuthService{}
			tt.setupMocks(mockSrv)

			controller := &AuthHandler{
				authService: mockSrv,
			}

			router := gin.New()
			router.GET("/auth/validate", controller.ValidateToken)

			req, err := http.NewRequest("GET", "/auth/validate", nil)
			assert.NoError(t, err)

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			for key, expectedValue := range tt.expectedBody {
				assert.Equal(t, expectedValue, response[key])
			}

			mockSrv.AssertExpectations(t)
		})
	}
}
