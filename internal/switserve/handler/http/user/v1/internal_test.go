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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/innovationmech/swit/internal/switserve/model"
	"github.com/innovationmech/swit/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestValidateUserCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testPassword := "testpassword123"
	hashedPassword, _ := utils.HashPassword(testPassword)

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockUserService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "success_valid_credentials",
			requestBody: map[string]string{
				"username": "testuser",
				"password": testPassword,
			},
			setupMocks: func(mockSrv *MockUserService) {
				testID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				user := &model.User{
					ID:           testID,
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: hashedPassword,
					IsActive:     true,
				}
				mockSrv.On("GetUserByUsername", mock.Anything, "testuser").Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"valid": true,
				"user": map[string]interface{}{
					"id":       "550e8400-e29b-41d4-a716-446655440000",
					"username": "testuser",
					"email":    "test@example.com",
					"role":     "",
				},
			},
		},
		{
			name: "error_invalid_json",
			requestBody: `{
				"username": "testuser",
				"password": 
			}`,
			setupMocks:     func(mockSrv *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "invalid character '}' looking for beginning of value",
			},
		},
		{
			name: "error_missing_username",
			requestBody: map[string]string{
				"password": testPassword,
			},
			setupMocks: func(mockSrv *MockUserService) {
				// 验证失败不会调用服务
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Key: 'Username' Error:Field validation for 'Username' failed on the 'required' tag",
			},
		},
		{
			name: "error_missing_password",
			requestBody: map[string]string{
				"username": "testuser",
			},
			setupMocks: func(mockSrv *MockUserService) {
				// 验证失败不会调用服务
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Key: 'Password' Error:Field validation for 'Password' failed on the 'required' tag",
			},
		},
		{
			name: "error_user_not_found",
			requestBody: map[string]string{
				"username": "nonexistentuser",
				"password": testPassword,
			},
			setupMocks: func(mockSrv *MockUserService) {
				mockSrv.On("GetUserByUsername", mock.Anything, "nonexistentuser").Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid credentials",
			},
		},
		{
			name: "error_wrong_password",
			requestBody: map[string]string{
				"username": "testuser",
				"password": "wrongpassword",
			},
			setupMocks: func(mockSrv *MockUserService) {
				testID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				user := &model.User{
					ID:           testID,
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: hashedPassword,
					IsActive:     true,
				}
				mockSrv.On("GetUserByUsername", mock.Anything, "testuser").Return(user, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid credentials",
			},
		},
		{
			name: "error_service_failure",
			requestBody: map[string]string{
				"username": "testuser",
				"password": testPassword,
			},
			setupMocks: func(mockSrv *MockUserService) {
				mockSrv.On("GetUserByUsername", mock.Anything, "testuser").Return(nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid credentials",
			},
		},
		{
			name: "success_inactive_user",
			requestBody: map[string]string{
				"username": "inactiveuser",
				"password": testPassword,
			},
			setupMocks: func(mockSrv *MockUserService) {
				testID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
				user := &model.User{
					ID:           testID,
					Username:     "inactiveuser",
					Email:        "inactive@example.com",
					PasswordHash: hashedPassword,
					IsActive:     false,
				}
				mockSrv.On("GetUserByUsername", mock.Anything, "inactiveuser").Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"valid": true,
				"user": map[string]interface{}{
					"id":       "550e8400-e29b-41d4-a716-446655440001",
					"username": "inactiveuser",
					"email":    "inactive@example.com",
					"role":     "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &MockUserService{}
			tt.setupMocks(mockSrv)

			uc := &Controller{
				userSrv: mockSrv,
			}

			router := gin.New()
			router.POST("/internal/validate", uc.ValidateUserCredentials)

			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req, err := http.NewRequest("POST", "/internal/validate", bytes.NewBuffer(body))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			for key, expectedValue := range tt.expectedBody {
				if key == "user" {
					// 处理嵌套的 user 对象
					actualUser, ok := response[key].(map[string]interface{})
					assert.True(t, ok, "user field should be a map")
					expectedUser, ok := expectedValue.(map[string]interface{})
					assert.True(t, ok, "expected user should be a map")

					for userKey, userExpectedValue := range expectedUser {
						assert.Equal(t, userExpectedValue, actualUser[userKey])
					}
				} else {
					assert.Equal(t, expectedValue, response[key])
				}
			}

			mockSrv.AssertExpectations(t)
		})
	}
}
