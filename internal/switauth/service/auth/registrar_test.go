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

package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/service/auth"
	"github.com/innovationmech/swit/internal/switauth/service/auth/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// MockUserClient is a mock implementation of client.UserClient
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

// MockTokenRepository is a mock implementation of repository.TokenRepository
type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) Create(ctx context.Context, token *model.Token) error {
	return m.Called(ctx, token).Error(0)
}

func (m *MockTokenRepository) GetByAccessToken(ctx context.Context, token string) (*model.Token, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Token), args.Error(1)
}

func (m *MockTokenRepository) GetByRefreshToken(ctx context.Context, token string) (*model.Token, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Token), args.Error(1)
}

func (m *MockTokenRepository) Update(ctx context.Context, token *model.Token) error {
	return m.Called(ctx, token).Error(0)
}

func (m *MockTokenRepository) InvalidateToken(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

func TestNewServiceRegistrar(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)

	registrar, err := auth.NewServiceRegistrar(mockUserClient, mockTokenRepo)
	assert.NoError(t, err)
	assert.NotNil(t, registrar)
	assert.Equal(t, "auth", registrar.GetName())
}

func TestServiceRegistrar_RegisterHTTP(t *testing.T) {
	// Initialize logger to prevent nil pointer panic
	logger.InitLogger()

	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)
	registrar, err := auth.NewServiceRegistrar(mockUserClient, mockTokenRepo)
	assert.NoError(t, err)

	router := gin.New()
	httpErr := registrar.RegisterHTTP(router)
	require.NoError(t, httpErr)

	// Verify routes are registered
	routes := router.Routes()
	assert.GreaterOrEqual(t, len(routes), 4)

	// Check for specific routes
	routeMap := make(map[string]string)
	for _, route := range routes {
		routeMap[route.Path] = route.Method
	}

	assert.Contains(t, routeMap, "/api/v1/auth/login")
	assert.Contains(t, routeMap, "/api/v1/auth/logout")
	assert.Contains(t, routeMap, "/api/v1/auth/refresh")
	assert.Contains(t, routeMap, "/api/v1/auth/validate")
}

func TestServiceRegistrar_RegisterGRPC(t *testing.T) {
	// Initialize logger to prevent nil pointer panic
	logger.InitLogger()

	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)
	registrar, err := auth.NewServiceRegistrar(mockUserClient, mockTokenRepo)
	assert.NoError(t, err)

	server := grpc.NewServer()
	grpcErr := registrar.RegisterGRPC(server)
	require.NoError(t, grpcErr)

	// Currently gRPC registration is a placeholder
	// This test ensures it doesn't fail
}

func TestServiceRegistrar_GetService(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)
	registrar, err := auth.NewServiceRegistrar(mockUserClient, mockTokenRepo)
	assert.NoError(t, err)

	service := registrar.GetService()
	assert.NotNil(t, service)
}

func TestAuthService_Login_Success(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)

	userID := uuid.New()
	user := &model.User{
		ID:       userID,
		Username: "testuser",
		IsActive: true,
	}

	// Setup expectations
	mockUserClient.On("ValidateUserCredentials", mock.Anything, "testuser", "testpass").Return(user, nil)
	mockTokenRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	v1Service, authErr := v1.NewAuthSrv(
		v1.WithUserClient(mockUserClient),
		v1.WithTokenRepository(mockTokenRepo),
	)
	assert.NoError(t, authErr)
	service := auth.NewAuthServiceAdapter(v1Service)
	accessToken, refreshToken, err := service.Login(context.Background(), "testuser", "testpass")

	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	mockUserClient.AssertExpectations(t)
	mockTokenRepo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)

	mockUserClient.On("ValidateUserCredentials", mock.Anything, "testuser", "wrongpass").Return(
		(*model.User)(nil), errors.New("invalid credentials"))

	v1Service, authErr := v1.NewAuthSrv(
		v1.WithUserClient(mockUserClient),
		v1.WithTokenRepository(mockTokenRepo),
	)
	assert.NoError(t, authErr)
	service := auth.NewAuthServiceAdapter(v1Service)
	accessToken, refreshToken, err := service.Login(context.Background(), "testuser", "wrongpass")

	assert.Error(t, err)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)
	assert.Contains(t, err.Error(), "invalid credentials")

	mockUserClient.AssertExpectations(t)
	mockTokenRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)

	userID := uuid.New()
	user := &model.User{
		ID:       userID,
		Username: "testuser",
		IsActive: false,
	}

	mockUserClient.On("ValidateUserCredentials", mock.Anything, "testuser", "testpass").Return(user, nil)

	v1Service, authErr := v1.NewAuthSrv(
		v1.WithUserClient(mockUserClient),
		v1.WithTokenRepository(mockTokenRepo),
	)
	assert.NoError(t, authErr)
	service := auth.NewAuthServiceAdapter(v1Service)
	accessToken, refreshToken, err := service.Login(context.Background(), "testuser", "testpass")

	assert.Error(t, err)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)
	assert.Contains(t, err.Error(), "not active")

	mockUserClient.AssertExpectations(t)
	mockTokenRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestAuthService_RefreshToken_Success(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)

	userID := uuid.New()
	// Create a valid refresh token using the actual token generation
	refreshToken, _, err := utils.GenerateRefreshToken(userID.String())
	require.NoError(t, err)

	oldToken := &model.Token{
		UserID:           userID,
		AccessToken:      "old-access",
		RefreshToken:     refreshToken,
		AccessExpiresAt:  time.Now().Add(1 * time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		IsValid:          true,
	}

	mockTokenRepo.On("GetByRefreshToken", mock.Anything, refreshToken).Return(oldToken, nil)
	mockTokenRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	v1Service, authErr := v1.NewAuthSrv(
		v1.WithUserClient(mockUserClient),
		v1.WithTokenRepository(mockTokenRepo),
	)
	assert.NoError(t, authErr)
	service := auth.NewAuthServiceAdapter(v1Service)
	newAccessToken, newRefreshToken, err := service.RefreshToken(context.Background(), refreshToken)

	assert.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)
	assert.NotEmpty(t, newRefreshToken)
	assert.NotEqual(t, "old-access", newAccessToken)

	mockTokenRepo.AssertExpectations(t)
}

func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)

	// Create a valid format but non-existent token
	userID := uuid.New()
	invalidToken, _, err := utils.GenerateRefreshToken(userID.String())
	require.NoError(t, err)

	mockTokenRepo.On("GetByRefreshToken", mock.Anything, invalidToken).Return(
		(*model.Token)(nil), errors.New("token not found"))

	v1Service, authErr := v1.NewAuthSrv(
		v1.WithUserClient(mockUserClient),
		v1.WithTokenRepository(mockTokenRepo),
	)
	assert.NoError(t, authErr)
	service := auth.NewAuthServiceAdapter(v1Service)
	newAccessToken, newRefreshToken, err := service.RefreshToken(context.Background(), invalidToken)

	assert.Error(t, err)
	assert.Empty(t, newAccessToken)
	assert.Empty(t, newRefreshToken)

	mockTokenRepo.AssertExpectations(t)
}

func TestAuthService_Logout_Success(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)

	mockTokenRepo.On("InvalidateToken", mock.Anything, "valid-token").Return(nil)

	v1Service, authErr := v1.NewAuthSrv(
		v1.WithUserClient(mockUserClient),
		v1.WithTokenRepository(mockTokenRepo),
	)
	assert.NoError(t, authErr)
	service := auth.NewAuthServiceAdapter(v1Service)
	err := service.Logout(context.Background(), "valid-token")

	assert.NoError(t, err)
	mockTokenRepo.AssertExpectations(t)
}

func TestAuthService_Logout_InvalidToken(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)

	mockTokenRepo.On("InvalidateToken", mock.Anything, "invalid-token").Return(
		errors.New("token not found"))

	v1Service, authErr := v1.NewAuthSrv(
		v1.WithUserClient(mockUserClient),
		v1.WithTokenRepository(mockTokenRepo),
	)
	assert.NoError(t, authErr)
	service := auth.NewAuthServiceAdapter(v1Service)
	err := service.Logout(context.Background(), "invalid-token")

	assert.Error(t, err)
	mockTokenRepo.AssertExpectations(t)
}

func TestAuthService_ValidateToken_Success(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)

	userID := uuid.New()
	// Create a valid access token
	accessToken, _, err := utils.GenerateAccessToken(userID.String())
	require.NoError(t, err)

	token := &model.Token{
		UserID:          userID,
		AccessToken:     accessToken,
		AccessExpiresAt: time.Now().Add(1 * time.Hour),
		IsValid:         true,
	}

	mockTokenRepo.On("GetByAccessToken", mock.Anything, accessToken).Return(token, nil)

	v1Service, authErr := v1.NewAuthSrv(
		v1.WithUserClient(mockUserClient),
		v1.WithTokenRepository(mockTokenRepo),
	)
	assert.NoError(t, authErr)
	service := auth.NewAuthServiceAdapter(v1Service)
	result, err := service.ValidateToken(context.Background(), accessToken)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, accessToken, result.AccessToken)
	assert.Equal(t, userID, result.UserID)

	mockTokenRepo.AssertExpectations(t)
}

func TestAuthService_ValidateToken_Expired(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)

	userID := uuid.New()
	// Create a valid but expired token
	accessToken, _, err := utils.GenerateAccessToken(userID.String())
	require.NoError(t, err)

	token := &model.Token{
		UserID:          userID,
		AccessToken:     accessToken,
		AccessExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		IsValid:         true,
	}

	mockTokenRepo.On("GetByAccessToken", mock.Anything, accessToken).Return(token, nil)

	v1Service, authErr := v1.NewAuthSrv(
		v1.WithUserClient(mockUserClient),
		v1.WithTokenRepository(mockTokenRepo),
	)
	assert.NoError(t, authErr)
	service := auth.NewAuthServiceAdapter(v1Service)
	result, err := service.ValidateToken(context.Background(), accessToken)

	assert.Error(t, err)
	assert.Nil(t, result)

	mockTokenRepo.AssertExpectations(t)
}

func TestAuthService_ValidateToken_Invalid(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)

	// Create a valid format but non-existent token
	userID := uuid.New()
	invalidToken, _, err := utils.GenerateAccessToken(userID.String())
	require.NoError(t, err)

	mockTokenRepo.On("GetByAccessToken", mock.Anything, invalidToken).Return(
		(*model.Token)(nil), errors.New("token not found"))

	v1Service, authErr := v1.NewAuthSrv(
		v1.WithUserClient(mockUserClient),
		v1.WithTokenRepository(mockTokenRepo),
	)
	assert.NoError(t, authErr)
	service := auth.NewAuthServiceAdapter(v1Service)
	result, err := service.ValidateToken(context.Background(), invalidToken)

	assert.Error(t, err)
	assert.Nil(t, result)

	mockTokenRepo.AssertExpectations(t)
}

func TestHTTPHandler_Login_Route(t *testing.T) {
	// Skip this test as it requires proper mock setup for HTTP handlers
	t.Skip("Skipping HTTP handler integration test - requires mock service setup")
}

func TestHTTPHandler_RegisterRoutes(t *testing.T) {
	mockUserClient := new(MockUserClient)
	mockTokenRepo := new(MockTokenRepository)
	registrar, err := auth.NewServiceRegistrar(mockUserClient, mockTokenRepo)
	assert.NoError(t, err)

	router := gin.New()
	httpErr := registrar.RegisterHTTP(router)
	require.NoError(t, httpErr)

	// Verify all routes are registered
	routes := router.Routes()
	assert.GreaterOrEqual(t, len(routes), 4)

	// Check specific routes exist
	paths := make(map[string]bool)
	for _, route := range routes {
		paths[route.Path] = true
	}

	assert.True(t, paths["/api/v1/auth/login"])
	assert.True(t, paths["/api/v1/auth/logout"])
	assert.True(t, paths["/api/v1/auth/refresh"])
	assert.True(t, paths["/api/v1/auth/validate"])
}
