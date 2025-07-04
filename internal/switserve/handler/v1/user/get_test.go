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

package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/innovationmech/swit/internal/switserve/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetUserByUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		username       string
		setupMocks     func(*MockUserService)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:     "success_get_user_by_username",
			username: "testuser",
			setupMocks: func(mockSrv *MockUserService) {
				testID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				user := &model.User{
					ID:           testID,
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: "hashedpassword",
					IsActive:     true,
				}
				mockSrv.On("GetUserByUsername", mock.Anything, "testuser").Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &model.User{
				ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashedpassword",
				IsActive:     true,
			},
		},
		{
			name:     "error_user_not_found",
			username: "nonexistent",
			setupMocks: func(mockSrv *MockUserService) {
				mockSrv.On("GetUserByUsername", mock.Anything, "nonexistent").Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error": "User not found",
			},
		},
		{
			name:     "error_service_failure",
			username: "testuser",
			setupMocks: func(mockSrv *MockUserService) {
				mockSrv.On("GetUserByUsername", mock.Anything, "testuser").Return(nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error": "User not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &MockUserService{}
			tt.setupMocks(mockSrv)

			uc := &UserController{
				userSrv: mockSrv,
			}

			router := gin.New()
			router.GET("/users/:username", uc.GetUserByUsername)

			req, err := http.NewRequest("GET", "/users/"+tt.username, nil)
			assert.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedStatus == http.StatusOK {
				responseUser := &model.User{}
				err = json.Unmarshal(w.Body.Bytes(), responseUser)
				assert.NoError(t, err)
				expectedUser := tt.expectedBody.(*model.User)
				assert.Equal(t, expectedUser.ID, responseUser.ID)
				assert.Equal(t, expectedUser.Username, responseUser.Username)
				assert.Equal(t, expectedUser.Email, responseUser.Email)
				// PasswordHash字段由于json:"-"标签在序列化时被忽略，这是正确的安全行为
				// 因此不检查PasswordHash字段
				assert.Equal(t, expectedUser.IsActive, responseUser.IsActive)
			} else {
				responseMap := response.(map[string]interface{})
				expectedMap := tt.expectedBody.(map[string]interface{})
				assert.Equal(t, expectedMap["error"], responseMap["error"])
			}

			mockSrv.AssertExpectations(t)
		})
	}
}

func TestGetUserByEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		email          string
		setupMocks     func(*MockUserService)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:  "success_get_user_by_email",
			email: "test@example.com",
			setupMocks: func(mockSrv *MockUserService) {
				testID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
				user := &model.User{
					ID:           testID,
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: "hashedpassword",
					IsActive:     true,
				}
				mockSrv.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &model.User{
				ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashedpassword",
				IsActive:     true,
			},
		},
		{
			name:  "error_user_not_found",
			email: "nonexistent@example.com",
			setupMocks: func(mockSrv *MockUserService) {
				mockSrv.On("GetUserByEmail", mock.Anything, "nonexistent@example.com").Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error": "User not found",
			},
		},
		{
			name:  "error_service_failure",
			email: "test@example.com",
			setupMocks: func(mockSrv *MockUserService) {
				mockSrv.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error": "User not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &MockUserService{}
			tt.setupMocks(mockSrv)

			uc := &UserController{
				userSrv: mockSrv,
			}

			router := gin.New()
			router.GET("/users/email/:email", uc.GetUserByEmail)

			req, err := http.NewRequest("GET", "/users/email/"+tt.email, nil)
			assert.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedStatus == http.StatusOK {
				responseUser := &model.User{}
				err = json.Unmarshal(w.Body.Bytes(), responseUser)
				assert.NoError(t, err)
				expectedUser := tt.expectedBody.(*model.User)
				assert.Equal(t, expectedUser.ID, responseUser.ID)
				assert.Equal(t, expectedUser.Username, responseUser.Username)
				assert.Equal(t, expectedUser.Email, responseUser.Email)
				// PasswordHash字段由于json:"-"标签在序列化时被忽略，这是正确的安全行为
				// 因此不检查PasswordHash字段
				assert.Equal(t, expectedUser.IsActive, responseUser.IsActive)
			} else {
				responseMap := response.(map[string]interface{})
				expectedMap := tt.expectedBody.(map[string]interface{})
				assert.Equal(t, expectedMap["error"], responseMap["error"])
			}

			mockSrv.AssertExpectations(t)
		})
	}
}
