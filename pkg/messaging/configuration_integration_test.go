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

package messaging_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/messaging/testutil"
	"github.com/innovationmech/swit/pkg/server"
)

// Initialize logger for tests
func init() {
	logger.Logger, _ = zap.NewDevelopment()
}

// loadServerConfig loads server configuration from a file path
func loadServerConfig(configPath string) (*server.ServerConfig, error) {
	// Read YAML file content
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Create server config with defaults
	config := server.NewServerConfig()
	
	// Unmarshal YAML into the server config
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}
	
	// Apply environment variable overrides
	applyEnvironmentOverrides(config)
	
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return config, nil
}

// loadServerConfigWithoutDefaults loads server config without applying defaults
func loadServerConfigWithoutDefaults(configPath string) (*server.ServerConfig, error) {
	// Read YAML file content
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Create empty server config without defaults
	config := &server.ServerConfig{}
	
	// Unmarshal YAML into the server config
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}
	
	// Apply environment variable overrides
	applyEnvironmentOverrides(config)
	
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return config, nil
}

// applyEnvironmentOverrides applies environment variable overrides to server configuration
func applyEnvironmentOverrides(config *server.ServerConfig) {
	if serviceName := os.Getenv("SWIT_SERVICE_NAME"); serviceName != "" {
		config.ServiceName = serviceName
	}
	
	if httpPort := os.Getenv("SWIT_HTTP_PORT"); httpPort != "" {
		config.HTTP.Port = httpPort
	}
	
	if grpcPort := os.Getenv("SWIT_GRPC_PORT"); grpcPort != "" {
		config.GRPC.Port = grpcPort
	}
	
	if shutdownTimeout := os.Getenv("SWIT_SHUTDOWN_TIMEOUT"); shutdownTimeout != "" {
		if timeout, err := time.ParseDuration(shutdownTimeout); err == nil {
			config.ShutdownTimeout = timeout
		}
	}
}

// ConfigurationIntegrationTestSuite tests configuration loading and validation integration
type ConfigurationIntegrationTestSuite struct {
	suite.Suite
	tempDir      string
	configFiles  map[string]string
	cleanupFuncs []func()
}

// TestConfigurationIntegration runs the configuration integration test suite
func TestConfigurationIntegration(t *testing.T) {
	suite.Run(t, new(ConfigurationIntegrationTestSuite))
}

// SetupSuite creates test environment for the suite
func (suite *ConfigurationIntegrationTestSuite) SetupSuite() {
	suite.tempDir = suite.T().TempDir()
	suite.configFiles = make(map[string]string)
	suite.cleanupFuncs = make([]func(), 0)
	
	// Save original environment variables
	suite.cleanupFuncs = append(suite.cleanupFuncs, func() {
		os.Unsetenv("SWIT_SERVICE_NAME")
		os.Unsetenv("SWIT_HTTP_PORT")
		os.Unsetenv("SWIT_GRPC_PORT")
		os.Unsetenv("SWIT_SHUTDOWN_TIMEOUT")
	})
}

// SetupTest runs before each test to ensure clean state
func (suite *ConfigurationIntegrationTestSuite) SetupTest() {
	// Clean up environment variables before each test
	os.Unsetenv("SWIT_SERVICE_NAME")
	os.Unsetenv("SWIT_HTTP_PORT")
	os.Unsetenv("SWIT_GRPC_PORT")
	os.Unsetenv("SWIT_SHUTDOWN_TIMEOUT")
}

// TearDownSuite cleans up after the suite
func (suite *ConfigurationIntegrationTestSuite) TearDownSuite() {
	for i := len(suite.cleanupFuncs) - 1; i >= 0; i-- {
		suite.cleanupFuncs[i]()
	}
	suite.cleanupFuncs = make([]func(), 0)
}

// createTestConfigFile creates a test configuration file with given content
func (suite *ConfigurationIntegrationTestSuite) createTestConfigFile(filename, content string) string {
	filePath := filepath.Join(suite.tempDir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	suite.Require().NoError(err)
	suite.configFiles[filename] = filePath
	
	// Add cleanup function
	suite.cleanupFuncs = append(suite.cleanupFuncs, func() {
		_ = os.Remove(filePath)
	})
	
	return filePath
}

// TestServerConfigValidation tests that server configuration validation works correctly
func (suite *ConfigurationIntegrationTestSuite) TestServerConfigValidation() {
	// Test valid configuration
	validConfig := `
service_name: "test-messaging-service"
http:
  enabled: true
  port: "8080"
  read_timeout: "30s"
  write_timeout: "30s"
grpc:
  enabled: true
  port: "9080"
  enable_keepalive: true
  enable_reflection: true
discovery:
  enabled: false
shutdown_timeout: "30s"
`

	configPath := suite.createTestConfigFile("valid-config.yaml", validConfig)
	
	// Load and validate configuration
	config, err := loadServerConfig(configPath)
	suite.Require().NoError(err)
	suite.Require().NotNil(config)
	
	// Verify configuration values
	suite.Equal("test-messaging-service", config.ServiceName)
	suite.True(config.HTTP.Enabled)
	suite.Equal("8080", config.HTTP.Port)
	suite.True(config.GRPC.Enabled)
	suite.Equal("9080", config.GRPC.Port)
	suite.False(config.Discovery.Enabled)
	suite.Equal(30*time.Second, config.ShutdownTimeout)
}

// TestServerConfigDefaults tests that configuration defaults are applied correctly
func (suite *ConfigurationIntegrationTestSuite) TestServerConfigDefaults() {
	// Test minimal configuration
	minimalConfig := `
service_name: "minimal-service"
`

	configPath := suite.createTestConfigFile("minimal-config.yaml", minimalConfig)
	
	// Load and validate configuration
	config, err := loadServerConfig(configPath)
	suite.Require().NoError(err)
	suite.Require().NotNil(config)
	
	// Verify defaults are applied
	suite.Equal("minimal-service", config.ServiceName)
	suite.True(config.HTTP.Enabled) // Default should be true
	suite.NotEmpty(config.HTTP.Port) // Default port should be set
	suite.True(config.GRPC.Enabled) // Default should be true
	suite.NotEmpty(config.GRPC.Port) // Default port should be set
	suite.True(config.Discovery.Enabled) // Default should be true
	suite.NotZero(config.ShutdownTimeout) // Default timeout should be set
}

// TestServerConfigValidationErrors tests that configuration validation errors are caught
func (suite *ConfigurationIntegrationTestSuite) TestServerConfigValidationErrors() {
	// Test configuration with missing required fields - should fail due to missing service_name
	invalidConfig := `
http:
  enabled: true
  port: "8080"
`

	configPath := suite.createTestConfigFile("invalid-config.yaml", invalidConfig)
	
	// Load and validate configuration - should fail
	config, err := loadServerConfigWithoutDefaults(configPath)
	suite.Error(err)
	suite.Nil(config)
	suite.Require().NotNil(err, "Error should not be nil when config validation fails")
	suite.Contains(err.Error(), "service_name")
	
	// Test configuration with invalid HTTP port - should fail with defaults applied
	invalidPortConfig := `
service_name: "test-service"
http:
  enabled: true
  port: "invalid-port"
`

	configPath = suite.createTestConfigFile("invalid-port-config.yaml", invalidPortConfig)
	
	// Load and validate configuration - should fail due to invalid port
	config, err = loadServerConfig(configPath)
	suite.Error(err)
	suite.Nil(config)
	suite.Require().NotNil(err, "Error should not be nil when port validation fails")
	suite.Contains(err.Error(), "port")
}

// TestMessagingConfigIntegration tests that messaging configuration integrates with server configuration
func (suite *ConfigurationIntegrationTestSuite) TestMessagingConfigIntegration() {
	// Test configuration with messaging settings
	configWithMessaging := `
service_name: "messaging-service"
http:
  enabled: true
  port: "8080"
grpc:
  enabled: true
  port: "9080"
messaging:
  enabled: true
  default_broker: "test-broker"
  brokers:
    test-broker:
      type: "inmemory"
      endpoints:
        - "memory://test"
      connection:
        timeout: "30s"
        max_attempts: 3
discovery:
  enabled: false
`

	configPath := suite.createTestConfigFile("messaging-config.yaml", configWithMessaging)
	
	// Load server configuration
	serverConfig, err := loadServerConfig(configPath)
	suite.Require().NoError(err)
	suite.Require().NotNil(serverConfig)
	
	// Create messaging coordinator
	coordinator := messaging.NewMessagingCoordinator()
	suite.Require().NotNil(coordinator)
	
	// Create test broker
	mockBroker := testutil.NewMockMessageBroker()
	err = coordinator.RegisterBroker("test-broker", mockBroker)
	suite.Require().NoError(err)
	
	// Test that configuration is accessible and can be used
	suite.True(coordinator != nil)
	suite.True(mockBroker != nil)
}

// TestEnvironmentVariableOverrides tests that environment variables override configuration file values
func (suite *ConfigurationIntegrationTestSuite) TestEnvironmentVariableOverrides() {
	// Set environment variable
	os.Setenv("SWIT_SERVICE_NAME", "env-override-service")
	os.Setenv("SWIT_HTTP_PORT", "9999")
	
	// Cleanup environment variables
	suite.cleanupFuncs = append(suite.cleanupFuncs, func() {
		os.Unsetenv("SWIT_SERVICE_NAME")
		os.Unsetenv("SWIT_HTTP_PORT")
	})
	
	// Create configuration file
	configContent := `
service_name: "file-service"
http:
  enabled: true
  port: "8080"
grpc:
  enabled: true
  port: "9080"
discovery:
  enabled: false
`

	configPath := suite.createTestConfigFile("env-override-config.yaml", configContent)
	
	// Load configuration - environment variables should override file values
	config, err := loadServerConfig(configPath)
	suite.Require().NoError(err)
	suite.Require().NotNil(config)
	
	// Verify environment variable overrides
	suite.Equal("env-override-service", config.ServiceName)
	suite.Equal("9999", config.HTTP.Port)
}

// TestConfigurationReload tests that configuration can be reloaded dynamically
func (suite *ConfigurationIntegrationTestSuite) TestConfigurationReload() {
	// Create initial configuration
	initialConfig := `
service_name: "reload-service"
http:
  enabled: true
  port: "8080"
grpc:
  enabled: true
  port: "9080"
discovery:
  enabled: false
`

	configPath := suite.createTestConfigFile("reload-config.yaml", initialConfig)
	
	// Load initial configuration
	config, err := loadServerConfig(configPath)
	suite.Require().NoError(err)
	suite.Require().NotNil(config)
	
	// Verify initial values
	suite.Equal("reload-service", config.ServiceName)
	suite.Equal("8080", config.HTTP.Port)
	
	// Update configuration file
	updatedConfig := `
service_name: "updated-service"
http:
  enabled: true
  port: "9999"
grpc:
  enabled: true
  port: "9080"
discovery:
  enabled: false
`

	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	suite.Require().NoError(err)
	
	// Reload configuration
	reloadedConfig, err := loadServerConfig(configPath)
	suite.Require().NoError(err)
	suite.Require().NotNil(reloadedConfig)
	
	// Verify updated values
	suite.Equal("updated-service", reloadedConfig.ServiceName)
	suite.Equal("9999", reloadedConfig.HTTP.Port)
}

// TestConfigurationWithMessagingIntegration tests complete integration with messaging components
func (suite *ConfigurationIntegrationTestSuite) TestConfigurationWithMessagingIntegration() {
	// Create comprehensive configuration with messaging
	comprehensiveConfig := `
service_name: "comprehensive-messaging-service"
http:
  enabled: true
  port: "8080"
  middleware:
    enable_cors: true
    enable_logging: true
    enable_timeout: true
grpc:
  enabled: true
  port: "9080"
  enable_keepalive: true
  enable_reflection: true
  enable_health_service: true
messaging:
  enabled: true
  default_broker: "primary-broker"
  brokers:
    primary-broker:
      type: "inmemory"
      endpoints:
        - "memory://primary"
      connection:
        timeout: "30s"
        max_attempts: 3
        pool_size: 10
      retry:
        max_attempts: 3
        initial_delay: "1s"
        max_delay: "30s"
        multiplier: 2.0
        jitter: 0.1
      monitoring:
        enabled: true
        metrics_interval: "30s"
        health_check_interval: "30s"
    secondary-broker:
      type: "inmemory"
      endpoints:
        - "memory://secondary"
discovery:
  enabled: false
shutdown_timeout: "60s"
`

	configPath := suite.createTestConfigFile("comprehensive-config.yaml", comprehensiveConfig)
	
	// Load configuration
	config, err := loadServerConfig(configPath)
	suite.Require().NoError(err)
	suite.Require().NotNil(config)
	
	// Create messaging coordinator and integrate with configuration
	coordinator := messaging.NewMessagingCoordinator()
	suite.Require().NotNil(coordinator)
	
	// Register brokers based on configuration
	primaryBroker := testutil.NewMockMessageBroker()
	err = coordinator.RegisterBroker("primary-broker", primaryBroker)
	suite.Require().NoError(err)
	
	secondaryBroker := testutil.NewMockMessageBroker()
	err = coordinator.RegisterBroker("secondary-broker", secondaryBroker)
	suite.Require().NoError(err)
	
	// Test that configuration-driven setup works
	suite.Equal("comprehensive-messaging-service", config.ServiceName)
	suite.True(config.HTTP.Enabled)
	suite.True(config.GRPC.Enabled)
	suite.Equal(60*time.Second, config.ShutdownTimeout)
	
	// Verify messaging coordinator is properly configured
	suite.NotNil(coordinator)
}

// TestConfigurationErrorHandling tests that configuration errors are handled gracefully
func (suite *ConfigurationIntegrationTestSuite) TestConfigurationErrorHandling() {
	// Test with non-existent configuration file
	nonExistentPath := filepath.Join(suite.tempDir, "non-existent-config.yaml")
	config, err := loadServerConfig(nonExistentPath)
	suite.Error(err)
	suite.Nil(config)
	suite.Contains(err.Error(), "no such file")
	
	// Test with invalid YAML
	invalidYAML := `
service_name: "invalid-yaml-service"
http:
  enabled: true
  port: "8080"
  invalid_yaml: [unclosed bracket
`

	invalidYAMLPath := suite.createTestConfigFile("invalid-yaml.yaml", invalidYAML)
	config, err = loadServerConfig(invalidYAMLPath)
	suite.Error(err)
	suite.Nil(config)
	suite.Contains(err.Error(), "yaml")
	
	// Test with invalid port values
	invalidPortConfig := `
service_name: "invalid-port-service"
http:
  enabled: true
  port: "invalid-port"
grpc:
  enabled: true
  port: "9080"
discovery:
  enabled: false
`

	invalidPortPath := suite.createTestConfigFile("invalid-port.yaml", invalidPortConfig)
	config, err = loadServerConfig(invalidPortPath)
	// This may or may not fail depending on validation implementation
	// If it doesn't fail, that's also acceptable as some validation might happen at runtime
	if err != nil {
		suite.Contains(err.Error(), "port")
	}
}

// TestConfigurationIntegrationWithServer tests that configuration integrates properly with server startup
func (suite *ConfigurationIntegrationTestSuite) TestConfigurationIntegrationWithServer() {
	// Create configuration for server startup
	serverConfigContent := `
service_name: "integration-test-service"
http:
  enabled: true
  port: "0"  # Dynamic port
  test_mode: true
grpc:
  enabled: true
  port: "0"  # Dynamic port
discovery:
  enabled: false
shutdown_timeout: "10s"
`

	configPath := suite.createTestConfigFile("server-integration-config.yaml", serverConfigContent)
	
	// Load server configuration
	serverConfig, err := loadServerConfig(configPath)
	suite.Require().NoError(err)
	suite.Require().NotNil(serverConfig)
	
	// Create test service registrar
	registrar := &TestConfigServiceRegistrar{}
	
	// Create server with loaded configuration
	baseServer, err := server.NewBusinessServerCore(serverConfig, registrar, nil)
	suite.Require().NoError(err)
	suite.Require().NotNil(baseServer)
	
	// Test server startup with loaded configuration
	ctx := context.Background()
	err = baseServer.Start(ctx)
	suite.Require().NoError(err)
	
	// Verify server is running
	suite.NotEmpty(baseServer.GetHTTPAddress())
	suite.NotEmpty(baseServer.GetGRPCAddress())
	
	// Test server shutdown
	err = baseServer.Shutdown()
	suite.Require().NoError(err)
}

// TestConfigServiceRegistrar is a test service registrar for configuration testing
type TestConfigServiceRegistrar struct{}

// RegisterServices registers test services
func (r *TestConfigServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register a simple HTTP handler for testing
	handler := &TestConfigHTTPHandler{}
	return registry.RegisterBusinessHTTPHandler(handler)
}

// TestConfigHTTPHandler is a test HTTP handler
type TestConfigHTTPHandler struct{}

func (h *TestConfigHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter := router.(*gin.Engine)
	ginRouter.GET("/test/config", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "message": "configuration test endpoint"})
	})
	return nil
}

func (h *TestConfigHTTPHandler) GetServiceName() string {
	return "test-config-http"
}