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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		userID         string
		setupMocks     func(*MockUserService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:   "success_delete_user",
			userID: "123",
			setupMocks: func(mockSrv *MockUserService) {
				mockSrv.On("DeleteUser", mock.Anything, "123").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   nil,
		},
		{
			name:   "error_service_failure",
			userID: "123",
			setupMocks: func(mockSrv *MockUserService) {
				mockSrv.On("DeleteUser", mock.Anything, "123").Return(errors.New("user not found"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "user not found",
			},
		},
		{
			name:   "empty_user_id",
			userID: "",
			setupMocks: func(mockSrv *MockUserService) {
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   nil,
		},
		{
			name:   "special_characters_in_id",
			userID: "abc-123_456",
			setupMocks: func(mockSrv *MockUserService) {
				mockSrv.On("DeleteUser", mock.Anything, "abc-123_456").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   nil,
		},
		{
			name:   "database_connection_error",
			userID: "123",
			setupMocks: func(mockSrv *MockUserService) {
				mockSrv.On("DeleteUser", mock.Anything, "123").Return(errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "database connection failed",
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
			router.DELETE("/users/:id", uc.DeleteUser)

			req, err := http.NewRequest("DELETE", "/users/"+tt.userID, nil)
			assert.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			}

			mockSrv.AssertExpectations(t)
		})
	}
}
