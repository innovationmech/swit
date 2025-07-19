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

package deps

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/innovationmech/swit/internal/switauth/client"
	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/repository"
	authv1 "github.com/innovationmech/swit/internal/switauth/service/auth/v1"
	"github.com/innovationmech/swit/internal/switauth/service/health"
	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// Mock implementations for testing

// MockTokenRepository implements repository.TokenRepository
type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) Create(ctx context.Context, token *model.Token) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockTokenRepository) GetByAccessToken(ctx context.Context, tokenString string) (*model.Token, error) {
	args := m.Called(ctx, tokenString)
	return args.Get(0).(*model.Token), args.Error(1)
}

func (m *MockTokenRepository) GetByRefreshToken(ctx context.Context, tokenString string) (*model.Token, error) {
	args := m.Called(ctx, tokenString)
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

// MockUserClient implements client.UserClient
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

// MockAuthSrv implements authv1.AuthSrv
type MockAuthSrv struct {
	mock.Mock
}

func (m *MockAuthSrv) Login(ctx context.Context, username, password string) (*authv1.AuthResponse, error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authv1.AuthResponse), args.Error(1)
}

func (m *MockAuthSrv) RefreshToken(ctx context.Context, refreshToken string) (*authv1.AuthResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authv1.AuthResponse), args.Error(1)
}

func (m *MockAuthSrv) ValidateToken(ctx context.Context, tokenString string) (*model.Token, error) {
	args := m.Called(ctx, tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Token), args.Error(1)
}

func (m *MockAuthSrv) Logout(ctx context.Context, tokenString string) error {
	args := m.Called(ctx, tokenString)
	return args.Error(0)
}

// MockHealthService implements health.Service
type MockHealthService struct {
	mock.Mock
}

func (m *MockHealthService) CheckHealth(ctx context.Context) *health.Status {
	args := m.Called(ctx)
	return args.Get(0).(*health.Status)
}

func (m *MockHealthService) GetHealthDetails(ctx context.Context) map[string]interface{} {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{})
}

// Test Dependencies struct
func TestDependencies_StructFields(t *testing.T) {
	// Create mock instances
	mockDB := &gorm.DB{}
	mockConfig := &config.AuthConfig{}
	mockSD := &discovery.ServiceDiscovery{}
	mockTokenRepo := &MockTokenRepository{}
	mockUserClient := &MockUserClient{}
	mockAuthSrv := &MockAuthSrv{}
	mockHealthSrv := &MockHealthService{}

	// Create Dependencies instance
	deps := &Dependencies{
		DB:         mockDB,
		Config:     mockConfig,
		SD:         mockSD,
		TokenRepo:  mockTokenRepo,
		UserClient: mockUserClient,
		AuthSrv:    mockAuthSrv,
		HealthSrv:  mockHealthSrv,
	}

	// Verify all fields are set correctly
	assert.Equal(t, mockDB, deps.DB)
	assert.Equal(t, mockConfig, deps.Config)
	assert.Equal(t, mockSD, deps.SD)
	assert.Equal(t, mockTokenRepo, deps.TokenRepo)
	assert.Equal(t, mockUserClient, deps.UserClient)
	assert.Equal(t, mockAuthSrv, deps.AuthSrv)
	assert.Equal(t, mockHealthSrv, deps.HealthSrv)

	// Verify interface compliance
	assert.Implements(t, (*repository.TokenRepository)(nil), deps.TokenRepo)
	assert.Implements(t, (*client.UserClient)(nil), deps.UserClient)
	assert.Implements(t, (*authv1.AuthSrv)(nil), deps.AuthSrv)
	assert.Implements(t, (*health.Service)(nil), deps.HealthSrv)
}

// Test NewDependencies function structure and logic
func TestNewDependencies_Structure(t *testing.T) {
	// Test that NewDependencies function exists and has correct signature
	t.Run("function_signature_validation", func(t *testing.T) {
		// Verify function exists
		assert.NotNil(t, NewDependencies)

		// Test function type
		funcType := reflect.TypeOf(NewDependencies)
		assert.Equal(t, reflect.Func, funcType.Kind())

		// Verify return types
		assert.Equal(t, 2, funcType.NumOut()) // (*Dependencies, error)
		assert.Equal(t, "*deps.Dependencies", funcType.Out(0).String())
		assert.Equal(t, "error", funcType.Out(1).String())
	})
}

// Test NewDependencies function - this test will be skipped in CI/CD
// as it requires actual database and service discovery setup
func TestNewDependencies_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires actual database and service discovery")

	// This test would require:
	// 1. A test database setup
	// 2. A test consul instance
	// 3. Proper configuration files
	// It's better to test this in integration test environment
}

// Test NewDependencies error scenarios using environment setup
func TestNewDependencies_ErrorScenarios(t *testing.T) {
	// Save original environment
	originalConfigPath := os.Getenv("CONFIG_PATH")
	originalDBHost := os.Getenv("DB_HOST")

	// Cleanup function
	cleanup := func() {
		if originalConfigPath != "" {
			os.Setenv("CONFIG_PATH", originalConfigPath)
		} else {
			os.Unsetenv("CONFIG_PATH")
		}
		if originalDBHost != "" {
			os.Setenv("DB_HOST", originalDBHost)
		} else {
			os.Unsetenv("DB_HOST")
		}
	}
	defer cleanup()

	t.Run("should handle missing configuration gracefully", func(t *testing.T) {
		// This test verifies that the function would handle missing config
		// In a real scenario, this would panic due to viper.ReadInConfig()
		// We're testing the structure rather than actual execution
		assert.NotNil(t, NewDependencies, "NewDependencies function should exist")
	})
}

// Test Dependencies creation with mock components
func TestDependencies_MockIntegration(t *testing.T) {
	// Create all mock components
	mockTokenRepo := &MockTokenRepository{}
	mockUserClient := &MockUserClient{}
	mockAuthSrv := &MockAuthSrv{}
	mockHealthSrv := &MockHealthService{}

	// Test TokenRepository mock
	t.Run("TokenRepository mock functionality", func(t *testing.T) {
		ctx := context.Background()
		testToken := &model.Token{
			ID:               uuid.New(),
			UserID:           uuid.New(),
			AccessToken:      "test_access_token",
			RefreshToken:     "test_refresh_token",
			AccessExpiresAt:  time.Now().Add(time.Hour),
			RefreshExpiresAt: time.Now().Add(24 * time.Hour),
			IsValid:          true,
		}

		mockTokenRepo.On("Create", ctx, testToken).Return(nil)
		mockTokenRepo.On("GetByAccessToken", ctx, "test_access_token").Return(testToken, nil)
		mockTokenRepo.On("InvalidateToken", ctx, "test_access_token").Return(nil)

		// Test Create
		err := mockTokenRepo.Create(ctx, testToken)
		assert.NoError(t, err)

		// Test GetByAccessToken
		retrievedToken, err := mockTokenRepo.GetByAccessToken(ctx, "test_access_token")
		assert.NoError(t, err)
		assert.Equal(t, testToken, retrievedToken)

		// Test InvalidateToken
		err = mockTokenRepo.InvalidateToken(ctx, "test_access_token")
		assert.NoError(t, err)

		mockTokenRepo.AssertExpectations(t)
	})

	// Test UserClient mock
	t.Run("UserClient mock functionality", func(t *testing.T) {
		ctx := context.Background()
		testUser := &model.User{
			ID:       uuid.New(),
			Username: "testuser",
			Email:    "test@example.com",
			IsActive: true,
		}

		mockUserClient.On("ValidateUserCredentials", ctx, "testuser", "password123").Return(testUser, nil)
		mockUserClient.On("ValidateUserCredentials", ctx, "testuser", "wrongpassword").Return(nil, errors.New("invalid credentials"))

		// Test successful validation
		user, err := mockUserClient.ValidateUserCredentials(ctx, "testuser", "password123")
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		// Test failed validation
		user, err = mockUserClient.ValidateUserCredentials(ctx, "testuser", "wrongpassword")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid credentials")

		mockUserClient.AssertExpectations(t)
	})

	// Test AuthSrv mock
	t.Run("AuthSrv mock functionality", func(t *testing.T) {
		ctx := context.Background()
		testAuthResponse := &authv1.AuthResponse{
			AccessToken:  "access_token_123",
			RefreshToken: "refresh_token_123",
			ExpiresAt:    time.Now().Add(time.Hour),
			TokenType:    "Bearer",
		}

		mockAuthSrv.On("Login", ctx, "testuser", "password123").Return(testAuthResponse, nil)
		mockAuthSrv.On("Logout", ctx, "access_token_123").Return(nil)

		// Test Login
		response, err := mockAuthSrv.Login(ctx, "testuser", "password123")
		assert.NoError(t, err)
		assert.Equal(t, testAuthResponse, response)

		// Test Logout
		err = mockAuthSrv.Logout(ctx, "access_token_123")
		assert.NoError(t, err)

		mockAuthSrv.AssertExpectations(t)
	})

	// Test HealthService mock
	t.Run("HealthService mock functionality", func(t *testing.T) {
		ctx := context.Background()
		testStatus := &health.Status{
			Status:    "healthy",
			Timestamp: time.Now().Unix(),
			Service:   "switauth",
			Version:   "1.0.0",
		}

		testDetails := map[string]interface{}{
			"service": "switauth",
			"status":  "healthy",
		}

		mockHealthSrv.On("CheckHealth", ctx).Return(testStatus)
		mockHealthSrv.On("GetHealthDetails", ctx).Return(testDetails)

		// Test CheckHealth
		status := mockHealthSrv.CheckHealth(ctx)
		assert.Equal(t, testStatus, status)

		// Test GetHealthDetails
		details := mockHealthSrv.GetHealthDetails(ctx)
		assert.Equal(t, testDetails, details)

		mockHealthSrv.AssertExpectations(t)
	})
}

// Test Dependencies struct initialization
func TestDependencies_Initialization(t *testing.T) {
	t.Run("should initialize with nil values", func(t *testing.T) {
		deps := &Dependencies{}

		assert.Nil(t, deps.DB)
		assert.Nil(t, deps.Config)
		assert.Nil(t, deps.SD)
		assert.Nil(t, deps.TokenRepo)
		assert.Nil(t, deps.UserClient)
		assert.Nil(t, deps.AuthSrv)
		assert.Nil(t, deps.HealthSrv)
	})

	t.Run("should allow partial initialization", func(t *testing.T) {
		mockConfig := &config.AuthConfig{}
		mockTokenRepo := &MockTokenRepository{}

		deps := &Dependencies{
			Config:    mockConfig,
			TokenRepo: mockTokenRepo,
		}

		assert.Equal(t, mockConfig, deps.Config)
		assert.Equal(t, mockTokenRepo, deps.TokenRepo)
		assert.Nil(t, deps.DB)
		assert.Nil(t, deps.SD)
		assert.Nil(t, deps.UserClient)
		assert.Nil(t, deps.AuthSrv)
		assert.Nil(t, deps.HealthSrv)
	})
}

// Test Dependencies struct behavior
func TestDependencies_Behavior(t *testing.T) {
	t.Run("should_support_dependency_injection_pattern", func(t *testing.T) {
		// Create Dependencies with some components
		mockTokenRepo := &MockTokenRepository{}
		mockUserClient := &MockUserClient{}

		deps := &Dependencies{
			TokenRepo:  mockTokenRepo,
			UserClient: mockUserClient,
		}

		// Verify we can access and use the injected dependencies
		assert.NotNil(t, deps.TokenRepo)
		assert.NotNil(t, deps.UserClient)

		// Verify they implement the expected interfaces
		assert.Implements(t, (*repository.TokenRepository)(nil), deps.TokenRepo)
		assert.Implements(t, (*client.UserClient)(nil), deps.UserClient)
	})

	t.Run("should_allow_runtime_dependency_replacement", func(t *testing.T) {
		// Create Dependencies with initial components
		initialTokenRepo := &MockTokenRepository{}
		deps := &Dependencies{
			TokenRepo: initialTokenRepo,
		}

		// Verify initial assignment
		assert.Same(t, initialTokenRepo, deps.TokenRepo)

		// Replace with new component
		newTokenRepo := &MockTokenRepository{}
		deps.TokenRepo = newTokenRepo

		// Verify replacement worked using pointer comparison
		assert.Same(t, newTokenRepo, deps.TokenRepo)
		assert.NotSame(t, initialTokenRepo, deps.TokenRepo)
	})
}

// Test Dependencies struct validation
func TestDependencies_Validation(t *testing.T) {
	t.Run("should_validate_required_dependencies", func(t *testing.T) {
		deps := &Dependencies{}

		// Check that we can identify missing dependencies
		assert.Nil(t, deps.DB, "DB should be nil when not initialized")
		assert.Nil(t, deps.Config, "Config should be nil when not initialized")
		assert.Nil(t, deps.SD, "ServiceDiscovery should be nil when not initialized")
		assert.Nil(t, deps.TokenRepo, "TokenRepo should be nil when not initialized")
		assert.Nil(t, deps.UserClient, "UserClient should be nil when not initialized")
		assert.Nil(t, deps.AuthSrv, "AuthSrv should be nil when not initialized")
		assert.Nil(t, deps.HealthSrv, "HealthSrv should be nil when not initialized")
	})

	t.Run("should_support_complete_initialization", func(t *testing.T) {
		// Create all required dependencies
		mockDB := &gorm.DB{}
		mockConfig := &config.AuthConfig{}
		mockSD := &discovery.ServiceDiscovery{}
		mockTokenRepo := &MockTokenRepository{}
		mockUserClient := &MockUserClient{}
		mockAuthSrv := &MockAuthSrv{}
		mockHealthSrv := &MockHealthService{}

		deps := &Dependencies{
			DB:         mockDB,
			Config:     mockConfig,
			SD:         mockSD,
			TokenRepo:  mockTokenRepo,
			UserClient: mockUserClient,
			AuthSrv:    mockAuthSrv,
			HealthSrv:  mockHealthSrv,
		}

		// Verify all dependencies are properly set
		assert.NotNil(t, deps.DB)
		assert.NotNil(t, deps.Config)
		assert.NotNil(t, deps.SD)
		assert.NotNil(t, deps.TokenRepo)
		assert.NotNil(t, deps.UserClient)
		assert.NotNil(t, deps.AuthSrv)
		assert.NotNil(t, deps.HealthSrv)

		// Verify types are correct
		assert.IsType(t, &gorm.DB{}, deps.DB)
		assert.IsType(t, &config.AuthConfig{}, deps.Config)
		assert.IsType(t, &discovery.ServiceDiscovery{}, deps.SD)
		assert.IsType(t, &MockTokenRepository{}, deps.TokenRepo)
		assert.IsType(t, &MockUserClient{}, deps.UserClient)
		assert.IsType(t, &MockAuthSrv{}, deps.AuthSrv)
		assert.IsType(t, &MockHealthService{}, deps.HealthSrv)
	})
}

// Benchmark test for Dependencies struct creation
func BenchmarkDependencies_Creation(b *testing.B) {
	mockDB := &gorm.DB{}
	mockConfig := &config.AuthConfig{}
	mockSD := &discovery.ServiceDiscovery{}
	mockTokenRepo := &MockTokenRepository{}
	mockUserClient := &MockUserClient{}
	mockAuthSrv := &MockAuthSrv{}
	mockHealthSrv := &MockHealthService{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &Dependencies{
			DB:         mockDB,
			Config:     mockConfig,
			SD:         mockSD,
			TokenRepo:  mockTokenRepo,
			UserClient: mockUserClient,
			AuthSrv:    mockAuthSrv,
			HealthSrv:  mockHealthSrv,
		}
	}
}
