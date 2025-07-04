// Copyright © 2023 jackelyj <dreamerlyj@gmail.com>
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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestValidateToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUserID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	mockToken := &model.Token{
		ID:               uuid.New(),
		UserID:           testUserID,
		AccessToken:      "valid_access_token",
		RefreshToken:     "valid_refresh_token",
		AccessExpiresAt:  time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		IsValid:          true,
	}

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
				mockSrv.On("ValidateToken", mock.Anything, "valid_access_token").Return(mockToken, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "Token is valid",
				"user_id": testUserID.String(),
			},
		},
		{
			name:           "error_missing_authorization_header",
			authHeader:     "",
			setupMocks:     func(mockSrv *MockAuthService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Missing Authorization header",
			},
		},
		{
			name:           "error_invalid_authorization_header_format",
			authHeader:     "InvalidFormat",
			setupMocks:     func(mockSrv *MockAuthService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid Authorization header format",
			},
		},
		{
			name:           "error_invalid_bearer_format",
			authHeader:     "NotBearer token_here",
			setupMocks:     func(mockSrv *MockAuthService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid Authorization header format",
			},
		},
		{
			name:       "error_invalid_token",
			authHeader: "Bearer invalid_token",
			setupMocks: func(mockSrv *MockAuthService) {
				mockSrv.On("ValidateToken", mock.Anything, "invalid_token").Return(
					nil, errors.New("invalid token"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "invalid token",
			},
		},
		{
			name:       "error_expired_token",
			authHeader: "Bearer expired_token",
			setupMocks: func(mockSrv *MockAuthService) {
				mockSrv.On("ValidateToken", mock.Anything, "expired_token").Return(
					nil, errors.New("token has expired"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "token has expired",
			},
		},
		{
			name:       "error_token_not_found",
			authHeader: "Bearer nonexistent_token",
			setupMocks: func(mockSrv *MockAuthService) {
				mockSrv.On("ValidateToken", mock.Anything, "nonexistent_token").Return(
					nil, errors.New("token not found"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "token not found",
			},
		},
		{
			name:       "error_malformed_jwt",
			authHeader: "Bearer malformed.jwt.token",
			setupMocks: func(mockSrv *MockAuthService) {
				mockSrv.On("ValidateToken", mock.Anything, "malformed.jwt.token").Return(
					nil, errors.New("token is malformed"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "token is malformed",
			},
		},
		{
			name:       "error_service_failure",
			authHeader: "Bearer valid_token",
			setupMocks: func(mockSrv *MockAuthService) {
				mockSrv.On("ValidateToken", mock.Anything, "valid_token").Return(
					nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "database connection failed",
			},
		},
		{
			name:       "success_validate_with_different_user_id",
			authHeader: "Bearer another_valid_token",
			setupMocks: func(mockSrv *MockAuthService) {
				anotherUserID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
				anotherToken := &model.Token{
					ID:               uuid.New(),
					UserID:           anotherUserID,
					AccessToken:      "another_valid_token",
					RefreshToken:     "another_refresh_token",
					AccessExpiresAt:  time.Now().Add(time.Hour),
					RefreshExpiresAt: time.Now().Add(24 * time.Hour),
					IsValid:          true,
				}
				mockSrv.On("ValidateToken", mock.Anything, "another_valid_token").Return(anotherToken, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "Token is valid",
				"user_id": "550e8400-e29b-41d4-a716-446655440001",
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
