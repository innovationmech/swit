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

package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
				"error": "Authorization header is missing",
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
				"error": "token not found",
			},
		},
		{
			name:       "error_database_failure",
			authHeader: "Bearer valid_token",
			setupMocks: func(mockSrv *MockAuthService) {
				mockSrv.On("Logout", mock.Anything, "Bearer valid_token").Return(
					errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "database connection failed",
			},
		},
		{
			name:       "logout_with_different_token_format",
			authHeader: "access_token_without_bearer",
			setupMocks: func(mockSrv *MockAuthService) {
				mockSrv.On("Logout", mock.Anything, "access_token_without_bearer").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "Logged out successfully",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &MockAuthService{}
			tt.setupMocks(mockSrv)

			controller := &AuthController{
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
