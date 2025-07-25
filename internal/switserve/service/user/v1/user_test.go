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
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/innovationmech/swit/internal/switserve/interfaces"
	"github.com/innovationmech/swit/internal/switserve/model"
	"github.com/innovationmech/swit/internal/switserve/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of UserRepository interface
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteUser(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestNewUserSrv(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func() *MockUserRepository
		expectError bool
		errorMsg    string
	}{
		{
			name: "success_with_options",
			setupMocks: func() *MockUserRepository {
				mockRepo := &MockUserRepository{}
				return mockRepo
			},
			expectError: false,
		},
		{
			name: "error_missing_repository",
			setupMocks: func() *MockUserRepository {
				return nil
			},
			expectError: true,
			errorMsg:    "user repository is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := tt.setupMocks()

			var srv interfaces.UserService
			var err error

			if mockRepo != nil {
				srv, err = NewUserSrv(WithUserRepository(mockRepo))
			} else {
				srv, err = NewUserSrv()
			}

			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, types.IsServiceError(err))
				serviceErr := types.GetServiceError(err)
				assert.Equal(t, types.ErrCodeValidation, serviceErr.Code)
				assert.Nil(t, srv)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, srv)
			}
		})
	}
}

func TestNewUserSrvWithConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *UserServiceConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "success_with_config",
			config: &UserServiceConfig{
				UserRepo: &MockUserRepository{},
			},
			expectError: false,
		},
		{
			name: "error_nil_repository",
			config: &UserServiceConfig{
				UserRepo: nil,
			},
			expectError: true,
			errorMsg:    "user repository is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, err := NewUserSrvWithConfig(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, types.IsServiceError(err))
				serviceErr := types.GetServiceError(err)
				assert.Equal(t, types.ErrCodeValidation, serviceErr.Code)
				assert.Nil(t, srv)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, srv)
			}
		})
	}
}

func TestUserService_CreateUser(t *testing.T) {
	tests := []struct {
		name        string
		user        *model.User
		setupMocks  func(*MockUserRepository)
		expectError bool
		errorMsg    string
	}{
		{
			name: "success_create_user",
			user: &model.User{
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "password123",
			},
			setupMocks: func(mockRepo *MockUserRepository) {
				mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)
			},
			expectError: false,
		},
		{
			name: "error_empty_username",
			user: &model.User{
				Username:     "",
				Email:        "test@example.com",
				PasswordHash: "password123",
			},
			setupMocks:  func(mockRepo *MockUserRepository) {},
			expectError: true,
			errorMsg:    "VALIDATION_ERROR",
		},
		{
			name: "error_empty_email",
			user: &model.User{
				Username:     "testuser",
				Email:        "",
				PasswordHash: "password123",
			},
			setupMocks:  func(mockRepo *MockUserRepository) {},
			expectError: true,
			errorMsg:    "VALIDATION_ERROR",
		},
		{
			name: "error_duplicate_email",
			user: &model.User{
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "password123",
			},
			setupMocks: func(mockRepo *MockUserRepository) {
				mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*model.User")).Return(errors.New("Duplicate entry"))
			},
			expectError: true,
			errorMsg:    "CONFLICT",
		},
		{
			name: "error_repository_failure",
			user: &model.User{
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "password123",
			},
			setupMocks: func(mockRepo *MockUserRepository) {
				mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*model.User")).Return(errors.New("database error"))
			},
			expectError: true,
			errorMsg:    "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockUserRepository{}
			tt.setupMocks(mockRepo)

			srv, err := NewUserSrv(WithUserRepository(mockRepo))
			assert.NoError(t, err)

			ctx := context.Background()
			err = srv.CreateUser(ctx, tt.user)

			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, types.IsServiceError(err))
				serviceErr := types.GetServiceError(err)
				assert.Equal(t, tt.errorMsg, serviceErr.Code)
			} else {
				assert.NoError(t, err)
				// Verify that password was hashed
				assert.NotEqual(t, "password123", tt.user.PasswordHash)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetUserByUsername(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		setupMocks  func(*MockUserRepository) *model.User
		expectError bool
		errorMsg    string
	}{
		{
			name:     "success_get_user",
			username: "testuser",
			setupMocks: func(mockRepo *MockUserRepository) *model.User {
				user := &model.User{
					ID:           uuid.New(),
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: "hashed_password",
				}
				mockRepo.On("GetUserByUsername", mock.Anything, "testuser").Return(user, nil)
				return user
			},
			expectError: false,
		},
		{
			name:     "error_user_not_found",
			username: "nonexistent",
			setupMocks: func(mockRepo *MockUserRepository) *model.User {
				mockRepo.On("GetUserByUsername", mock.Anything, "nonexistent").Return(nil, errors.New("user not found"))
				return nil
			},
			expectError: true,
			errorMsg:    "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockUserRepository{}
			expectedUser := tt.setupMocks(mockRepo)

			srv, err := NewUserSrv(WithUserRepository(mockRepo))
			assert.NoError(t, err)

			ctx := context.Background()
			user, err := srv.GetUserByUsername(ctx, tt.username)

			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, types.IsServiceError(err))
				serviceErr := types.GetServiceError(err)
				assert.Equal(t, tt.errorMsg, serviceErr.Code)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, expectedUser.Username, user.Username)
				assert.Equal(t, expectedUser.Email, user.Email)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetUserByEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		setupMocks  func(*MockUserRepository) *model.User
		expectError bool
		errorMsg    string
	}{
		{
			name:  "success_get_user",
			email: "test@example.com",
			setupMocks: func(mockRepo *MockUserRepository) *model.User {
				user := &model.User{
					ID:           uuid.New(),
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: "hashed_password",
				}
				mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)
				return user
			},
			expectError: false,
		},
		{
			name:  "error_user_not_found",
			email: "nonexistent@example.com",
			setupMocks: func(mockRepo *MockUserRepository) *model.User {
				mockRepo.On("GetUserByEmail", mock.Anything, "nonexistent@example.com").Return(nil, errors.New("user not found"))
				return nil
			},
			expectError: true,
			errorMsg:    "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockUserRepository{}
			expectedUser := tt.setupMocks(mockRepo)

			srv, err := NewUserSrv(WithUserRepository(mockRepo))
			assert.NoError(t, err)

			ctx := context.Background()
			user, err := srv.GetUserByEmail(ctx, tt.email)

			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, types.IsServiceError(err))
				serviceErr := types.GetServiceError(err)
				assert.Equal(t, tt.errorMsg, serviceErr.Code)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, expectedUser.Username, user.Username)
				assert.Equal(t, expectedUser.Email, user.Email)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		setupMocks  func(*MockUserRepository)
		expectError bool
		errorMsg    string
	}{
		{
			name:   "success_delete_user",
			userID: "test-user-id",
			setupMocks: func(mockRepo *MockUserRepository) {
				mockRepo.On("DeleteUser", mock.Anything, "test-user-id").Return(nil)
			},
			expectError: false,
		},
		{
			name:   "error_user_not_found",
			userID: "nonexistent-id",
			setupMocks: func(mockRepo *MockUserRepository) {
				mockRepo.On("DeleteUser", mock.Anything, "nonexistent-id").Return(errors.New("user not found"))
			},
			expectError: true,
			errorMsg:    "NOT_FOUND",
		},
		{
			name:   "error_database_failure",
			userID: "test-user-id",
			setupMocks: func(mockRepo *MockUserRepository) {
				mockRepo.On("DeleteUser", mock.Anything, "test-user-id").Return(errors.New("database error"))
			},
			expectError: true,
			errorMsg:    "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockUserRepository{}
			tt.setupMocks(mockRepo)

			srv, err := NewUserSrv(WithUserRepository(mockRepo))
			assert.NoError(t, err)

			ctx := context.Background()
			err = srv.DeleteUser(ctx, tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, types.IsServiceError(err))
				serviceErr := types.GetServiceError(err)
				assert.Equal(t, tt.errorMsg, serviceErr.Code)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestWithUserRepository(t *testing.T) {
	mockRepo := &MockUserRepository{}
	config := &UserServiceConfig{}

	option := WithUserRepository(mockRepo)
	option(config)

	assert.Equal(t, mockRepo, config.UserRepo)
}
