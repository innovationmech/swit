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

package switserve

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/innovationmech/swit/internal/switserve/config"
	"github.com/innovationmech/swit/internal/switserve/deps"
	greeterv1 "github.com/innovationmech/swit/internal/switserve/handler/http/greeter/v1"
	"github.com/innovationmech/swit/internal/switserve/interfaces"
	"github.com/innovationmech/swit/internal/switserve/model"
	notificationv1 "github.com/innovationmech/swit/internal/switserve/service/notification/v1"
	"github.com/innovationmech/swit/internal/switserve/types"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockServiceRegistry implements server.ServiceRegistry for testing
type MockServiceRegistry struct {
	mock.Mock
}

func (m *MockServiceRegistry) RegisterHTTPHandler(handler server.BusinessHTTPHandler) error {
	args := m.Called(handler)
	return args.Error(0)
}

func (m *MockServiceRegistry) RegisterGRPCService(service server.BusinessGRPCService) error {
	args := m.Called(service)
	return args.Error(0)
}

func (m *MockServiceRegistry) RegisterBusinessGRPCService(service server.BusinessGRPCService) error {
	args := m.Called(service)
	return args.Error(0)
}

func (m *MockServiceRegistry) RegisterBusinessHTTPHandler(handler server.BusinessHTTPHandler) error {
	args := m.Called(handler)
	return args.Error(0)
}

func (m *MockServiceRegistry) RegisterHealthCheck(check server.BusinessHealthCheck) error {
	args := m.Called(check)
	return args.Error(0)
}

func (m *MockServiceRegistry) RegisterBusinessHealthCheck(check server.BusinessHealthCheck) error {
	args := m.Called(check)
	return args.Error(0)
}

// MockDependencies creates mock dependencies for testing
func createMockDependencies() *deps.Dependencies {
	return &deps.Dependencies{
		// Mock services - in real tests these would be proper mocks
		GreeterSrv:      &mockGreeterService{},
		NotificationSrv: &mockNotificationService{},
		HealthSrv:       &mockHealthService{},
		StopSrv:         &mockStopService{},
		UserSrv:         &mockUserService{},
	}
}

// Mock service implementations
type mockGreeterService struct{}

func (m *mockGreeterService) GenerateGreeting(ctx context.Context, name, language string) (string, error) {
	return "Hello " + name, nil
}

type mockNotificationService struct{}

func (m *mockNotificationService) CreateNotification(ctx context.Context, userID, title, content string) (*notificationv1.Notification, error) {
	return &notificationv1.Notification{
		ID:      "test-id",
		UserID:  userID,
		Title:   title,
		Content: content,
	}, nil
}

func (m *mockNotificationService) GetNotifications(ctx context.Context, userID string, limit, offset int) ([]*notificationv1.Notification, error) {
	return []*notificationv1.Notification{}, nil
}

func (m *mockNotificationService) MarkAsRead(ctx context.Context, notificationID string) error {
	return nil
}

func (m *mockNotificationService) DeleteNotification(ctx context.Context, notificationID string) error {
	return nil
}

type mockHealthService struct{}

func (m *mockHealthService) CheckHealth(ctx context.Context) (*interfaces.HealthStatus, error) {
	return &interfaces.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
	}, nil
}

type mockStopService struct{}

func (m *mockStopService) InitiateShutdown(ctx context.Context) (*interfaces.ShutdownStatus, error) {
	return &interfaces.ShutdownStatus{
		Status:      types.StatusShutdownInitiated,
		Message:     "Shutdown initiated",
		InitiatedAt: time.Now().Unix(),
	}, nil
}

type mockUserService struct{}

func (m *mockUserService) CreateUser(ctx context.Context, user *model.User) error {
	return nil
}

func (m *mockUserService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	return &model.User{
		ID:       uuid.New(),
		Username: username,
		Email:    "test@example.com",
	}, nil
}

func (m *mockUserService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return &model.User{
		ID:       uuid.New(),
		Username: "testuser",
		Email:    email,
	}, nil
}

func (m *mockUserService) DeleteUser(ctx context.Context, id string) error {
	return nil
}

func TestNewServiceRegistrar(t *testing.T) {
	deps := createMockDependencies()
	registrar := NewServiceRegistrar(deps)

	assert.NotNil(t, registrar)
	assert.Equal(t, deps, registrar.deps)
}

func TestServiceRegistrar_RegisterServices(t *testing.T) {
	// Initialize logger for tests
	logger.InitLogger()

	deps := createMockDependencies()
	registrar := NewServiceRegistrar(deps)

	mockRegistry := &MockServiceRegistry{}

	// Expect all HTTP handlers to be registered
	mockRegistry.On("RegisterBusinessHTTPHandler", mock.AnythingOfType("*switserve.GreeterBusinessHTTPHandler")).Return(nil)
	mockRegistry.On("RegisterBusinessHTTPHandler", mock.AnythingOfType("*switserve.NotificationBusinessHTTPHandler")).Return(nil)
	mockRegistry.On("RegisterBusinessHTTPHandler", mock.AnythingOfType("*switserve.HealthBusinessHTTPHandler")).Return(nil)
	mockRegistry.On("RegisterBusinessHTTPHandler", mock.AnythingOfType("*switserve.StopBusinessHTTPHandler")).Return(nil)
	mockRegistry.On("RegisterBusinessHTTPHandler", mock.AnythingOfType("*switserve.UserBusinessHTTPHandler")).Return(nil)

	// Expect all health checks to be registered
	mockRegistry.On("RegisterBusinessHealthCheck", mock.AnythingOfType("*switserve.GreeterBusinessHealthCheck")).Return(nil)
	mockRegistry.On("RegisterBusinessHealthCheck", mock.AnythingOfType("*switserve.UserBusinessHealthCheck")).Return(nil)
	mockRegistry.On("RegisterBusinessHealthCheck", mock.AnythingOfType("*switserve.NotificationBusinessHealthCheck")).Return(nil)
	mockRegistry.On("RegisterBusinessHealthCheck", mock.AnythingOfType("*switserve.HealthServiceCheck")).Return(nil)
	mockRegistry.On("RegisterBusinessHealthCheck", mock.AnythingOfType("*switserve.StopServiceCheck")).Return(nil)

	err := registrar.RegisterServices(mockRegistry)

	assert.NoError(t, err)
	mockRegistry.AssertExpectations(t)
}

func TestGreeterHTTPHandler_RegisterRoutes(t *testing.T) {
	// Initialize logger for tests
	logger.InitLogger()

	deps := createMockDependencies()
	handler := &GreeterBusinessHTTPHandler{
		handler: greeterv1.NewGreeterHandler(deps.GreeterSrv),
	}

	// Test with gin.Engine
	router := gin.New()
	err := handler.RegisterRoutes(router)
	assert.NoError(t, err)

	// Test with wrong router type
	err = handler.RegisterRoutes("not a router")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected *gin.Engine")
}

func TestGreeterHTTPHandler_GetServiceName(t *testing.T) {
	handler := &GreeterBusinessHTTPHandler{}
	assert.Equal(t, "greeter-service", handler.GetServiceName())
}

func TestHealthChecks(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		healthCheck server.BusinessHealthCheck
		serviceName string
		expectError bool
	}{
		{
			name:        "GreeterHealthCheck with service",
			healthCheck: &GreeterBusinessHealthCheck{greeterService: &mockGreeterService{}, startTime: time.Now()},
			serviceName: "greeter-service",
			expectError: false,
		},
		{
			name:        "GreeterHealthCheck without service",
			healthCheck: &GreeterBusinessHealthCheck{greeterService: nil, startTime: time.Now()},
			serviceName: "greeter-service",
			expectError: true,
		},
		{
			name:        "UserHealthCheck with service",
			healthCheck: &UserBusinessHealthCheck{userService: &mockUserService{}, startTime: time.Now()},
			serviceName: "user-service",
			expectError: false,
		},
		{
			name:        "NotificationHealthCheck with service",
			healthCheck: &NotificationBusinessHealthCheck{notificationService: &mockNotificationService{}, startTime: time.Now()},
			serviceName: "notification-service",
			expectError: false,
		},
		{
			name:        "HealthServiceCheck with service",
			healthCheck: &HealthServiceCheck{healthService: &mockHealthService{}, startTime: time.Now()},
			serviceName: "health-service",
			expectError: false,
		},
		{
			name:        "StopServiceCheck with service",
			healthCheck: &StopServiceCheck{stopService: &mockStopService{}, startTime: time.Now()},
			serviceName: "stop-service",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.serviceName, tt.healthCheck.GetServiceName())

			err := tt.healthCheck.Check(ctx)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDependencyContainer(t *testing.T) {
	deps := createMockDependencies()
	container := NewBusinessDependencyContainer(deps)

	assert.NotNil(t, container)
	assert.Equal(t, deps, container.deps)
	assert.False(t, container.closed)

	// Test GetService
	service, err := container.GetService("greeter-service")
	assert.NoError(t, err)
	assert.Equal(t, deps.GreeterSrv, service)

	service, err = container.GetService("user-service")
	assert.NoError(t, err)
	assert.Equal(t, deps.UserSrv, service)

	// Test unknown service
	_, err = container.GetService("unknown-service")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service unknown-service not found")

	// Test Close
	err = container.Close()
	assert.NoError(t, err)
	assert.True(t, container.closed)

	// Test GetService after close
	_, err = container.GetService("greeter-service")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency container is closed")

	// Test Close again (should be idempotent)
	err = container.Close()
	assert.NoError(t, err)
}

func TestConfigMapper(t *testing.T) {
	serveConfig := &config.ServeConfig{
		Server: struct {
			Port     string `json:"port" mapstructure:"port"`
			GRPCPort string `json:"grpc_port" mapstructure:"grpc_port"`
		}{
			Port:     "9000",
			GRPCPort: "10000",
		},
		ServiceDiscovery: struct {
			Address string `json:"address" mapstructure:"address"`
		}{
			Address: "localhost:8500",
		},
	}

	mapper := NewConfigMapper(serveConfig)
	serverConfig := mapper.ToServerConfig()

	assert.NotNil(t, serverConfig)
	assert.Equal(t, "swit-serve", serverConfig.ServiceName)
	assert.Equal(t, "9000", serverConfig.HTTP.Port)
	assert.Equal(t, ":9000", serverConfig.HTTP.Address)
	assert.True(t, serverConfig.HTTP.Enabled)
	assert.True(t, serverConfig.HTTP.EnableReady)
	assert.Equal(t, "10000", serverConfig.GRPC.Port)
	assert.Equal(t, ":10000", serverConfig.GRPC.Address)
	assert.True(t, serverConfig.GRPC.Enabled)
	assert.True(t, serverConfig.GRPC.EnableKeepalive)
	assert.True(t, serverConfig.GRPC.EnableReflection)
	assert.True(t, serverConfig.GRPC.EnableHealthService)
	assert.Equal(t, "localhost:8500", serverConfig.Discovery.Address)
	assert.Equal(t, "swit-serve", serverConfig.Discovery.ServiceName)
	assert.Equal(t, []string{"api", "v1"}, serverConfig.Discovery.Tags)
	assert.True(t, serverConfig.Discovery.Enabled)
	assert.Equal(t, 5*time.Second, serverConfig.ShutdownTimeout)
}

func TestConfigMapper_WithDefaults(t *testing.T) {
	// Test with empty configuration to verify defaults
	serveConfig := &config.ServeConfig{}

	mapper := NewConfigMapper(serveConfig)
	serverConfig := mapper.ToServerConfig()

	assert.Equal(t, "9000", serverConfig.HTTP.Port)
	assert.Equal(t, ":9000", serverConfig.HTTP.Address)
	assert.Equal(t, "10000", serverConfig.GRPC.Port)
	assert.Equal(t, ":10000", serverConfig.GRPC.Address)
}

func TestServeGRPCService(t *testing.T) {
	service := NewServeBusinessGRPCService()

	assert.NotNil(t, service)
	assert.Equal(t, "serve-grpc-service", service.GetServiceName())

	// Test RegisterGRPC with grpc.Server
	grpcServer := grpc.NewServer()
	err := service.RegisterGRPC(grpcServer)
	assert.NoError(t, err)

	// Test RegisterGRPC with wrong type
	err = service.RegisterGRPC("not a grpc server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected *grpc.Server")
}
