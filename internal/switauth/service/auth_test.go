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

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserClient is a mock implementation of UserClient interface
type MockUserClient struct {
	mock.Mock
}

func (m *MockUserClient) ValidateUserCredentials(ctx context.Context, username, password string) (*model.User, error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

// MockTokenRepository is a mock implementation of TokenRepository interface
type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) Create(ctx context.Context, token *model.Token) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockTokenRepository) GetByAccessToken(ctx context.Context, tokenString string) (*model.Token, error) {
	args := m.Called(ctx, tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Token), args.Error(1)
}

func (m *MockTokenRepository) GetByRefreshToken(ctx context.Context, refreshToken string) (*model.Token, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Token), args.Error(1)
}

func (m *MockTokenRepository) Update(ctx context.Context, token *model.Token) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockTokenRepository) InvalidateToken(ctx context.Context, tokenString string) error {
	args := m.Called(ctx, tokenString)
	return args.Error(0)
}

func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name            string
		username        string
		password        string
		setupMocks      func(*MockUserClient, *MockTokenRepository)
		expectedAccess  string
		expectedRefresh string
		expectedError   string
	}{
		{
			name:     "successful_login",
			username: "testuser",
			password: "password123",
			setupMocks: func(mockUserClient *MockUserClient, mockTokenRepo *MockTokenRepository) {
				user := &model.User{
					ID:       uuid.New(),
					Username: "testuser",
					Email:    "test@example.com",
					IsActive: true,
				}
				mockUserClient.On("ValidateUserCredentials", mock.Anything, "testuser", "password123").Return(user, nil)
				mockTokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Token")).Return(nil)
			},
			expectedAccess:  "", // We'll check that tokens are not empty
			expectedRefresh: "",
			expectedError:   "",
		},
		{
			name:     "user_validation_failed",
			username: "invaliduser",
			password: "wrongpassword",
			setupMocks: func(mockUserClient *MockUserClient, mockTokenRepo *MockTokenRepository) {
				mockUserClient.On("ValidateUserCredentials", mock.Anything, "invaliduser", "wrongpassword").Return(nil, errors.New("invalid credentials"))
			},
			expectedError: "invalid credentials",
		},
		{
			name:     "user_not_active",
			username: "inactiveuser",
			password: "password123",
			setupMocks: func(mockUserClient *MockUserClient, mockTokenRepo *MockTokenRepository) {
				user := &model.User{
					ID:       uuid.New(),
					Username: "inactiveuser",
					Email:    "inactive@example.com",
					IsActive: false,
				}
				mockUserClient.On("ValidateUserCredentials", mock.Anything, "inactiveuser", "password123").Return(user, nil)
			},
			expectedError: "user is not active",
		},
		{
			name:     "token_creation_failed",
			username: "testuser",
			password: "password123",
			setupMocks: func(mockUserClient *MockUserClient, mockTokenRepo *MockTokenRepository) {
				user := &model.User{
					ID:       uuid.New(),
					Username: "testuser",
					Email:    "test@example.com",
					IsActive: true,
				}
				mockUserClient.On("ValidateUserCredentials", mock.Anything, "testuser", "password123").Return(user, nil)
				mockTokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Token")).Return(errors.New("database error"))
			},
			expectedError: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserClient := &MockUserClient{}
			mockTokenRepo := &MockTokenRepository{}

			tt.setupMocks(mockUserClient, mockTokenRepo)

			authSvc := NewAuthService(mockUserClient, mockTokenRepo)

			accessToken, refreshToken, err := authSvc.Login(context.Background(), tt.username, tt.password)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, accessToken)
				assert.Empty(t, refreshToken)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, accessToken)
				assert.NotEmpty(t, refreshToken)
			}

			mockUserClient.AssertExpectations(t)
			mockTokenRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	userID := uuid.New()

	// Generate a valid refresh token for testing
	validRefreshToken, _, err := utils.GenerateRefreshToken(userID.String())
	if err != nil {
		t.Fatalf("Failed to generate valid refresh token: %v", err)
	}

	tests := []struct {
		name          string
		refreshToken  string
		setupMocks    func(*MockTokenRepository)
		expectedError string
	}{
		{
			name:         "successful_refresh",
			refreshToken: validRefreshToken,
			setupMocks: func(mockTokenRepo *MockTokenRepository) {
				token := &model.Token{
					ID:               uuid.New(),
					UserID:           userID,
					AccessToken:      "old_access_token",
					RefreshToken:     validRefreshToken,
					AccessExpiresAt:  time.Now().Add(time.Hour),
					RefreshExpiresAt: time.Now().Add(24 * time.Hour),
					IsValid:          true,
				}
				mockTokenRepo.On("GetByRefreshToken", mock.Anything, validRefreshToken).Return(token, nil)
				mockTokenRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.Token")).Return(nil)
			},
		},
		{
			name:         "invalid_refresh_token",
			refreshToken: "invalid_token",
			setupMocks: func(mockTokenRepo *MockTokenRepository) {
				// Mock will not be called due to JWT validation failure
			},
			expectedError: "invalid refresh token",
		},
		{
			name:         "token_not_found_in_database",
			refreshToken: validRefreshToken,
			setupMocks: func(mockTokenRepo *MockTokenRepository) {
				mockTokenRepo.On("GetByRefreshToken", mock.Anything, validRefreshToken).Return(nil, errors.New("token not found"))
			},
			expectedError: "token not found",
		},
		{
			name:         "expired_refresh_token",
			refreshToken: validRefreshToken,
			setupMocks: func(mockTokenRepo *MockTokenRepository) {
				token := &model.Token{
					ID:               uuid.New(),
					UserID:           userID,
					AccessToken:      "old_access_token",
					RefreshToken:     validRefreshToken,
					AccessExpiresAt:  time.Now().Add(-time.Hour),
					RefreshExpiresAt: time.Now().Add(-time.Hour), // Expired
					IsValid:          true,
				}
				mockTokenRepo.On("GetByRefreshToken", mock.Anything, validRefreshToken).Return(token, nil)
			},
			expectedError: "refresh token has expired",
		},
		{
			name:         "update_token_failed",
			refreshToken: validRefreshToken,
			setupMocks: func(mockTokenRepo *MockTokenRepository) {
				token := &model.Token{
					ID:               uuid.New(),
					UserID:           userID,
					AccessToken:      "old_access_token",
					RefreshToken:     validRefreshToken,
					AccessExpiresAt:  time.Now().Add(time.Hour),
					RefreshExpiresAt: time.Now().Add(24 * time.Hour),
					IsValid:          true,
				}
				mockTokenRepo.On("GetByRefreshToken", mock.Anything, validRefreshToken).Return(token, nil)
				mockTokenRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.Token")).Return(errors.New("database update error"))
			},
			expectedError: "database update error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserClient := &MockUserClient{}
			mockTokenRepo := &MockTokenRepository{}

			tt.setupMocks(mockTokenRepo)

			authSvc := NewAuthService(mockUserClient, mockTokenRepo)

			newAccessToken, newRefreshToken, err := authSvc.RefreshToken(context.Background(), tt.refreshToken)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, newAccessToken)
				assert.Empty(t, newRefreshToken)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, newAccessToken)
				assert.NotEmpty(t, newRefreshToken)
			}

			mockTokenRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	userID := uuid.New()

	// Generate a valid access token for testing
	validAccessToken, _, err := utils.GenerateAccessToken(userID.String())
	if err != nil {
		t.Fatalf("Failed to generate valid access token: %v", err)
	}

	tests := []struct {
		name          string
		accessToken   string
		setupMocks    func(*MockTokenRepository)
		expectedError string
	}{
		{
			name:        "successful_validation",
			accessToken: validAccessToken,
			setupMocks: func(mockTokenRepo *MockTokenRepository) {
				token := &model.Token{
					ID:              uuid.New(),
					UserID:          userID,
					AccessToken:     validAccessToken,
					AccessExpiresAt: time.Now().Add(time.Hour),
					IsValid:         true,
				}
				mockTokenRepo.On("GetByAccessToken", mock.Anything, validAccessToken).Return(token, nil)
			},
		},
		{
			name:        "invalid_jwt_token",
			accessToken: "invalid_access_token",
			setupMocks: func(mockTokenRepo *MockTokenRepository) {
				// Mock will not be called due to JWT validation failure
			},
			expectedError: "token is malformed",
		},
		{
			name:        "token_not_found_in_database",
			accessToken: validAccessToken,
			setupMocks: func(mockTokenRepo *MockTokenRepository) {
				mockTokenRepo.On("GetByAccessToken", mock.Anything, validAccessToken).Return(nil, errors.New("token not found"))
			},
			expectedError: "token not found",
		},
		{
			name:        "expired_access_token",
			accessToken: validAccessToken,
			setupMocks: func(mockTokenRepo *MockTokenRepository) {
				token := &model.Token{
					ID:              uuid.New(),
					UserID:          userID,
					AccessToken:     validAccessToken,
					AccessExpiresAt: time.Now().Add(-time.Hour), // Expired
					IsValid:         true,
				}
				mockTokenRepo.On("GetByAccessToken", mock.Anything, validAccessToken).Return(token, nil)
			},
			expectedError: "token has expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserClient := &MockUserClient{}
			mockTokenRepo := &MockTokenRepository{}

			tt.setupMocks(mockTokenRepo)

			authSvc := NewAuthService(mockUserClient, mockTokenRepo)

			token, err := authSvc.ValidateToken(context.Background(), tt.accessToken)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, tt.accessToken, token.AccessToken)
			}

			mockTokenRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	validAccessToken := "valid_access_token"

	tests := []struct {
		name          string
		accessToken   string
		setupMocks    func(*MockTokenRepository)
		expectedError string
	}{
		{
			name:        "successful_logout",
			accessToken: validAccessToken,
			setupMocks: func(mockTokenRepo *MockTokenRepository) {
				mockTokenRepo.On("InvalidateToken", mock.Anything, validAccessToken).Return(nil)
			},
		},
		{
			name:        "invalidate_token_failed",
			accessToken: validAccessToken,
			setupMocks: func(mockTokenRepo *MockTokenRepository) {
				mockTokenRepo.On("InvalidateToken", mock.Anything, validAccessToken).Return(errors.New("database error"))
			},
			expectedError: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserClient := &MockUserClient{}
			mockTokenRepo := &MockTokenRepository{}

			tt.setupMocks(mockTokenRepo)

			authSvc := NewAuthService(mockUserClient, mockTokenRepo)

			err := authSvc.Logout(context.Background(), tt.accessToken)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockTokenRepo.AssertExpectations(t)
		})
	}
}

func TestNewAuthService(t *testing.T) {
	mockUserClient := &MockUserClient{}
	mockTokenRepo := &MockTokenRepository{}

	authSvc := NewAuthService(mockUserClient, mockTokenRepo)

	assert.NotNil(t, authSvc)
	assert.Implements(t, (*AuthService)(nil), authSvc)
}

// Integration-style test that tests the service with more realistic scenarios
func TestAuthService_Integration(t *testing.T) {
	t.Run("full_authentication_flow", func(t *testing.T) {
		mockUserClient := &MockUserClient{}
		mockTokenRepo := &MockTokenRepository{}

		// Setup user for login
		user := &model.User{
			ID:       uuid.New(),
			Username: "testuser",
			Email:    "test@example.com",
			IsActive: true,
		}

		// Mock login
		mockUserClient.On("ValidateUserCredentials", mock.Anything, "testuser", "password123").Return(user, nil)
		mockTokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Token")).Return(nil)

		authSvc := NewAuthService(mockUserClient, mockTokenRepo)

		// Test login
		accessToken, refreshToken, err := authSvc.Login(context.Background(), "testuser", "password123")
		assert.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)

		// Mock logout
		mockTokenRepo.On("InvalidateToken", mock.Anything, accessToken).Return(nil)

		// Test logout
		err = authSvc.Logout(context.Background(), accessToken)
		assert.NoError(t, err)

		mockUserClient.AssertExpectations(t)
		mockTokenRepo.AssertExpectations(t)
	})
}

// Benchmark tests
func BenchmarkAuthService_Login(b *testing.B) {
	mockUserClient := &MockUserClient{}
	mockTokenRepo := &MockTokenRepository{}

	user := &model.User{
		ID:       uuid.New(),
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
	}

	// Setup mocks for benchmark
	mockUserClient.On("ValidateUserCredentials", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)
	mockTokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Token")).Return(nil)

	authSvc := NewAuthService(mockUserClient, mockTokenRepo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := authSvc.Login(context.Background(), "testuser", "password123")
		if err != nil {
			b.Fatalf("Login failed: %v", err)
		}
	}
}

func BenchmarkAuthService_ValidateToken(b *testing.B) {
	mockUserClient := &MockUserClient{}
	mockTokenRepo := &MockTokenRepository{}

	userID := uuid.New()
	accessToken, _, err := utils.GenerateAccessToken(userID.String())
	if err != nil {
		b.Fatalf("Failed to generate access token: %v", err)
	}

	token := &model.Token{
		ID:              uuid.New(),
		UserID:          userID,
		AccessToken:     accessToken,
		AccessExpiresAt: time.Now().Add(time.Hour),
		IsValid:         true,
	}

	mockTokenRepo.On("GetByAccessToken", mock.Anything, mock.Anything).Return(token, nil)

	authSvc := NewAuthService(mockUserClient, mockTokenRepo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := authSvc.ValidateToken(context.Background(), accessToken)
		if err != nil {
			b.Fatalf("ValidateToken failed: %v", err)
		}
	}
}

// Edge case tests
func TestAuthService_EdgeCases(t *testing.T) {
	t.Run("login_with_empty_credentials", func(t *testing.T) {
		mockUserClient := &MockUserClient{}
		mockTokenRepo := &MockTokenRepository{}

		mockUserClient.On("ValidateUserCredentials", mock.Anything, "", "").Return(nil, errors.New("empty credentials"))

		authSvc := NewAuthService(mockUserClient, mockTokenRepo)

		accessToken, refreshToken, err := authSvc.Login(context.Background(), "", "")
		assert.Error(t, err)
		assert.Empty(t, accessToken)
		assert.Empty(t, refreshToken)
		assert.Contains(t, err.Error(), "empty credentials")

		mockUserClient.AssertExpectations(t)
	})

	t.Run("validate_empty_token", func(t *testing.T) {
		mockUserClient := &MockUserClient{}
		mockTokenRepo := &MockTokenRepository{}

		authSvc := NewAuthService(mockUserClient, mockTokenRepo)

		token, err := authSvc.ValidateToken(context.Background(), "")
		assert.Error(t, err)
		assert.Nil(t, token)
	})

	t.Run("logout_empty_token", func(t *testing.T) {
		mockUserClient := &MockUserClient{}
		mockTokenRepo := &MockTokenRepository{}

		mockTokenRepo.On("InvalidateToken", mock.Anything, "").Return(nil)

		authSvc := NewAuthService(mockUserClient, mockTokenRepo)

		err := authSvc.Logout(context.Background(), "")
		assert.NoError(t, err) // 系统允许空 token 的 logout

		mockTokenRepo.AssertExpectations(t)
	})
}
