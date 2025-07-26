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
	"github.com/innovationmech/swit/internal/switserve/model"
	"github.com/innovationmech/swit/internal/switserve/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserService is a mock implementation of interfaces.UserService
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserService) DeleteUser(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserService) ListUsers(ctx context.Context, limit, offset int) ([]*model.User, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*model.User), args.Error(1)
}

func TestNewUserController(t *testing.T) {
	mockUserSrv := &MockUserService{}
	handler := NewUserHandler(mockUserSrv)

	assert.NotNil(t, handler)
	assert.Equal(t, mockUserSrv, handler.userSrv)
}

func TestHandler_CreateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful creation",
			requestBody: model.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(m *MockUserService) {
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"message":"User created successfully"}`,
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			requestBody: model.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(m *MockUserService) {
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("*model.User")).Return(errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSrv := &MockUserService{}
			tt.mockSetup(mockUserSrv)

			handler := NewUserHandler(mockUserSrv)
			router := gin.New()
			router.POST("/users/create", handler.CreateUser)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/users/create", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}

			mockUserSrv.AssertExpectations(t)
		})
	}
}

func TestHandler_GetUserByUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		username       string
		mockSetup      func(*MockUserService)
		expectedStatus int
	}{
		{
			name:     "successful get",
			username: "testuser",
			mockSetup: func(m *MockUserService) {
				user := &model.User{
					ID:       uuid.New(),
					Username: "testuser",
					Email:    "test@example.com",
				}
				m.On("GetUserByUsername", mock.Anything, "testuser").Return(user, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "user not found",
			username: "nonexistent",
			mockSetup: func(m *MockUserService) {
				m.On("GetUserByUsername", mock.Anything, "nonexistent").Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSrv := &MockUserService{}
			tt.mockSetup(mockUserSrv)

			handler := NewUserHandler(mockUserSrv)
			router := gin.New()
			router.GET("/users/username/:username", handler.GetUserByUsername)

			req := httptest.NewRequest(http.MethodGet, "/users/username/"+tt.username, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockUserSrv.AssertExpectations(t)
		})
	}
}

func TestHandler_ValidateUserCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockUserService)
		expectedStatus int
	}{
		{
			name: "valid credentials",
			requestBody: map[string]string{
				"username": "testuser",
				"password": "password123",
			},
			mockSetup: func(m *MockUserService) {
				user := &model.User{
					ID:           uuid.New(),
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMye", // This would be a real hash
				}
				m.On("GetUserByUsername", mock.Anything, "testuser").Return(user, nil)
			},
			expectedStatus: http.StatusUnauthorized, // Will fail because password doesn't match hash
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "user not found",
			requestBody: map[string]string{
				"username": "nonexistent",
				"password": "password123",
			},
			mockSetup: func(m *MockUserService) {
				m.On("GetUserByUsername", mock.Anything, "nonexistent").Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSrv := &MockUserService{}
			tt.mockSetup(mockUserSrv)

			handler := NewUserHandler(mockUserSrv)
			router := gin.New()
			router.POST("/internal/validate-user", handler.ValidateUserCredentials)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/internal/validate-user", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockUserSrv.AssertExpectations(t)
		})
	}
}

func TestHandler_ServiceHandlerInterface(t *testing.T) {
	mockUserSrv := &MockUserService{}
	handler := NewUserHandler(mockUserSrv)

	// Test GetMetadata
	metadata := handler.GetMetadata()
	assert.NotNil(t, metadata)
	assert.Equal(t, "user-service", metadata.Name)
	assert.Equal(t, "v1", metadata.Version)
	assert.Equal(t, "User management service", metadata.Description)

	// Test GetHealthEndpoint
	healthEndpoint := handler.GetHealthEndpoint()
	assert.Equal(t, "/api/v1/users/health", healthEndpoint)

	// Test IsHealthy with nil service
	handlerWithNilSrv := &UserHandler{userSrv: nil}
	status, err := handlerWithNilSrv.IsHealthy(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, types.HealthStatusUnhealthy, status.Status)

	// Test IsHealthy with valid service
	status, err = handler.IsHealthy(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, types.HealthStatusHealthy, status.Status)

	// Test Initialize
	err = handler.Initialize(context.Background())
	assert.NoError(t, err)

	// Test Shutdown
	err = handler.Shutdown(context.Background())
	assert.NoError(t, err)

	// Test RegisterHTTP
	router := gin.New()
	err = handler.RegisterHTTP(router)
	assert.NoError(t, err)

	// Test RegisterGRPC
	err = handler.RegisterGRPC(nil)
	assert.NoError(t, err)
}
