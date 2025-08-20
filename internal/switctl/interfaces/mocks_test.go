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

package interfaces

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockCommand is a mock implementation of the Command interface
type MockCommand struct {
	mock.Mock
}

func (m *MockCommand) Execute(ctx context.Context, args []string) error {
	ret := m.Called(ctx, args)
	return ret.Error(0)
}

func (m *MockCommand) GetHelp() string {
	ret := m.Called()
	return ret.String(0)
}

func (m *MockCommand) GetUsage() string {
	ret := m.Called()
	return ret.String(0)
}

// MockInteractiveUI is a mock implementation of the InteractiveUI interface
type MockInteractiveUI struct {
	mock.Mock
}

func (m *MockInteractiveUI) ShowWelcome() error {
	ret := m.Called()
	return ret.Error(0)
}

func (m *MockInteractiveUI) PromptInput(prompt string, validator InputValidator) (string, error) {
	ret := m.Called(prompt, validator)
	return ret.String(0), ret.Error(1)
}

func (m *MockInteractiveUI) ShowMenu(title string, options []MenuOption) (int, error) {
	ret := m.Called(title, options)
	return ret.Int(0), ret.Error(1)
}

func (m *MockInteractiveUI) ShowProgress(title string, total int) ProgressBar {
	ret := m.Called(title, total)
	return ret.Get(0).(ProgressBar)
}

func (m *MockInteractiveUI) ShowSuccess(message string) error {
	ret := m.Called(message)
	return ret.Error(0)
}

func (m *MockInteractiveUI) ShowError(err error) error {
	ret := m.Called(err)
	return ret.Error(0)
}

func (m *MockInteractiveUI) ShowTable(headers []string, rows [][]string) error {
	ret := m.Called(headers, rows)
	return ret.Error(0)
}

// MockGenerator is a mock implementation of the Generator interface
type MockGenerator struct {
	mock.Mock
}

func (m *MockGenerator) GenerateService(config ServiceConfig) error {
	ret := m.Called(config)
	return ret.Error(0)
}

func (m *MockGenerator) GenerateAPI(config APIConfig) error {
	ret := m.Called(config)
	return ret.Error(0)
}

func (m *MockGenerator) GenerateModel(config ModelConfig) error {
	ret := m.Called(config)
	return ret.Error(0)
}

func (m *MockGenerator) GenerateMiddleware(config MiddlewareConfig) error {
	ret := m.Called(config)
	return ret.Error(0)
}

// MockTemplateEngine is a mock implementation of the TemplateEngine interface
type MockTemplateEngine struct {
	mock.Mock
}

func (m *MockTemplateEngine) LoadTemplate(name string) (Template, error) {
	ret := m.Called(name)
	return ret.Get(0).(Template), ret.Error(1)
}

func (m *MockTemplateEngine) RenderTemplate(template Template, data interface{}) ([]byte, error) {
	ret := m.Called(template, data)
	return ret.Get(0).([]byte), ret.Error(1)
}

func (m *MockTemplateEngine) RegisterFunction(name string, fn interface{}) error {
	ret := m.Called(name, fn)
	return ret.Error(0)
}

func (m *MockTemplateEngine) SetTemplateDir(dir string) error {
	ret := m.Called(dir)
	return ret.Error(0)
}

// MockTemplate is a mock implementation of the Template interface
type MockTemplate struct {
	mock.Mock
}

func (m *MockTemplate) Name() string {
	ret := m.Called()
	return ret.String(0)
}

func (m *MockTemplate) Render(data interface{}) ([]byte, error) {
	ret := m.Called(data)
	return ret.Get(0).([]byte), ret.Error(1)
}

// MockQualityChecker is a mock implementation of the QualityChecker interface
type MockQualityChecker struct {
	mock.Mock
}

func (m *MockQualityChecker) CheckCodeStyle() CheckResult {
	ret := m.Called()
	return ret.Get(0).(CheckResult)
}

func (m *MockQualityChecker) RunTests() TestResult {
	ret := m.Called()
	return ret.Get(0).(TestResult)
}

func (m *MockQualityChecker) CheckSecurity() SecurityResult {
	ret := m.Called()
	return ret.Get(0).(SecurityResult)
}

func (m *MockQualityChecker) CheckCoverage() CoverageResult {
	ret := m.Called()
	return ret.Get(0).(CoverageResult)
}

func (m *MockQualityChecker) ValidateConfig() ValidationResult {
	ret := m.Called()
	return ret.Get(0).(ValidationResult)
}

func (m *MockQualityChecker) CheckInterfaces() CheckResult {
	ret := m.Called()
	return ret.Get(0).(CheckResult)
}

func (m *MockQualityChecker) CheckImports() CheckResult {
	ret := m.Called()
	return ret.Get(0).(CheckResult)
}

func (m *MockQualityChecker) CheckCopyright() CheckResult {
	ret := m.Called()
	return ret.Get(0).(CheckResult)
}

// MockPlugin is a mock implementation of the Plugin interface
type MockPlugin struct {
	mock.Mock
}

func (m *MockPlugin) Name() string {
	ret := m.Called()
	return ret.String(0)
}

func (m *MockPlugin) Version() string {
	ret := m.Called()
	return ret.String(0)
}

func (m *MockPlugin) Initialize(config PluginConfig) error {
	ret := m.Called(config)
	return ret.Error(0)
}

func (m *MockPlugin) Execute(ctx context.Context, args []string) error {
	ret := m.Called(ctx, args)
	return ret.Error(0)
}

func (m *MockPlugin) Cleanup() error {
	ret := m.Called()
	return ret.Error(0)
}

// MockPluginManager is a mock implementation of the PluginManager interface
type MockPluginManager struct {
	mock.Mock
}

func (m *MockPluginManager) LoadPlugins() error {
	ret := m.Called()
	return ret.Error(0)
}

func (m *MockPluginManager) GetPlugin(name string) (Plugin, bool) {
	ret := m.Called(name)
	return ret.Get(0).(Plugin), ret.Bool(1)
}

func (m *MockPluginManager) ListPlugins() []string {
	ret := m.Called()
	return ret.Get(0).([]string)
}

func (m *MockPluginManager) RegisterPlugin(plugin Plugin) error {
	ret := m.Called(plugin)
	return ret.Error(0)
}

func (m *MockPluginManager) UnloadPlugin(name string) error {
	ret := m.Called(name)
	return ret.Error(0)
}

// MockDependencyContainer is a mock implementation of the DependencyContainer interface
type MockDependencyContainer struct {
	mock.Mock
}

func (m *MockDependencyContainer) Register(name string, factory ServiceFactory) error {
	ret := m.Called(name, factory)
	return ret.Error(0)
}

func (m *MockDependencyContainer) RegisterSingleton(name string, factory ServiceFactory) error {
	ret := m.Called(name, factory)
	return ret.Error(0)
}

func (m *MockDependencyContainer) GetService(name string) (interface{}, error) {
	ret := m.Called(name)
	return ret.Get(0), ret.Error(1)
}

func (m *MockDependencyContainer) HasService(name string) bool {
	ret := m.Called(name)
	return ret.Bool(0)
}

func (m *MockDependencyContainer) Close() error {
	ret := m.Called()
	return ret.Error(0)
}

// MockConfigManager is a mock implementation of the ConfigManager interface
type MockConfigManager struct {
	mock.Mock
}

func (m *MockConfigManager) Load() error {
	ret := m.Called()
	return ret.Error(0)
}

func (m *MockConfigManager) Get(key string) interface{} {
	ret := m.Called(key)
	return ret.Get(0)
}

func (m *MockConfigManager) GetString(key string) string {
	ret := m.Called(key)
	return ret.String(0)
}

func (m *MockConfigManager) GetInt(key string) int {
	ret := m.Called(key)
	return ret.Int(0)
}

func (m *MockConfigManager) GetBool(key string) bool {
	ret := m.Called(key)
	return ret.Bool(0)
}

func (m *MockConfigManager) Set(key string, value interface{}) {
	m.Called(key, value)
}

func (m *MockConfigManager) Validate() error {
	ret := m.Called()
	return ret.Error(0)
}

func (m *MockConfigManager) Save(path string) error {
	ret := m.Called(path)
	return ret.Error(0)
}

// MockFileSystem is a mock implementation of the FileSystem interface
type MockFileSystem struct {
	mock.Mock
}

func (m *MockFileSystem) WriteFile(path string, content []byte, perm int) error {
	ret := m.Called(path, content, perm)
	return ret.Error(0)
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	ret := m.Called(path)
	return ret.Get(0).([]byte), ret.Error(1)
}

func (m *MockFileSystem) MkdirAll(path string, perm int) error {
	ret := m.Called(path, perm)
	return ret.Error(0)
}

func (m *MockFileSystem) Exists(path string) bool {
	ret := m.Called(path)
	return ret.Bool(0)
}

func (m *MockFileSystem) Remove(path string) error {
	ret := m.Called(path)
	return ret.Error(0)
}

func (m *MockFileSystem) Copy(src, dst string) error {
	ret := m.Called(src, dst)
	return ret.Error(0)
}

// MockProgressBar is a mock implementation of the ProgressBar interface
type MockProgressBar struct {
	mock.Mock
}

func (m *MockProgressBar) Update(current int) error {
	ret := m.Called(current)
	return ret.Error(0)
}

func (m *MockProgressBar) SetMessage(message string) error {
	ret := m.Called(message)
	return ret.Error(0)
}

func (m *MockProgressBar) Finish() error {
	ret := m.Called()
	return ret.Error(0)
}

func (m *MockProgressBar) SetTotal(total int) error {
	ret := m.Called(total)
	return ret.Error(0)
}

// MockColor is a mock implementation of the Color interface
type MockColor struct {
	mock.Mock
}

func (m *MockColor) Sprint(a ...interface{}) string {
	ret := m.Called(a)
	return ret.String(0)
}

func (m *MockColor) Sprintf(format string, a ...interface{}) string {
	ret := m.Called(format, a)
	return ret.String(0)
}

// MockProjectManager is a mock implementation of the ProjectManager interface
type MockProjectManager struct {
	mock.Mock
}

func (m *MockProjectManager) InitProject(config ProjectConfig) error {
	ret := m.Called(config)
	return ret.Error(0)
}

func (m *MockProjectManager) ValidateProject() error {
	ret := m.Called()
	return ret.Error(0)
}

func (m *MockProjectManager) GetProjectInfo() (ProjectInfo, error) {
	ret := m.Called()
	return ret.Get(0).(ProjectInfo), ret.Error(1)
}

func (m *MockProjectManager) UpdateProject(updates map[string]interface{}) error {
	ret := m.Called(updates)
	return ret.Error(0)
}

// MockDependencyManager is a mock implementation of the DependencyManager interface
type MockDependencyManager struct {
	mock.Mock
}

func (m *MockDependencyManager) UpdateDependencies() error {
	ret := m.Called()
	return ret.Error(0)
}

func (m *MockDependencyManager) AuditDependencies() AuditResult {
	ret := m.Called()
	return ret.Get(0).(AuditResult)
}

func (m *MockDependencyManager) ListDependencies() ([]Dependency, error) {
	ret := m.Called()
	return ret.Get(0).([]Dependency), ret.Error(1)
}

func (m *MockDependencyManager) AddDependency(dep Dependency) error {
	ret := m.Called(dep)
	return ret.Error(0)
}

func (m *MockDependencyManager) RemoveDependency(name string) error {
	ret := m.Called(name)
	return ret.Error(0)
}

// MockVersionManager is a mock implementation of the VersionManager interface
type MockVersionManager struct {
	mock.Mock
}

func (m *MockVersionManager) CheckCompatibility() CompatibilityResult {
	ret := m.Called()
	return ret.Get(0).(CompatibilityResult)
}

func (m *MockVersionManager) GetCurrentVersion() (string, error) {
	ret := m.Called()
	return ret.String(0), ret.Error(1)
}

func (m *MockVersionManager) GetLatestVersion() (string, error) {
	ret := m.Called()
	return ret.String(0), ret.Error(1)
}

func (m *MockVersionManager) MigrateProject(targetVersion string) error {
	ret := m.Called(targetVersion)
	return ret.Error(0)
}

// MockLogger is a mock implementation of the Logger interface
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Fatal(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

// Test helper functions

// CreateMockServiceConfig creates a test service configuration
func CreateMockServiceConfig() ServiceConfig {
	return ServiceConfig{
		Name:        "test-service",
		Description: "Test service for unit tests",
		Author:      "Test Author",
		Version:     "1.0.0",
		Features: ServiceFeatures{
			Database:       true,
			Authentication: false,
			Cache:          true,
			MessageQueue:   false,
			Monitoring:     true,
			Tracing:        true,
			Logging:        true,
			HealthCheck:    true,
			Metrics:        true,
			Docker:         true,
			Kubernetes:     false,
		},
		Database: DatabaseConfig{
			Type:     "mysql",
			Host:     "localhost",
			Port:     3306,
			Database: "testdb",
			Username: "testuser",
			Password: "testpass",
			Schema:   "public",
		},
		Auth: AuthConfig{
			Type:       "jwt",
			SecretKey:  "test-secret-key",
			Expiration: 15 * time.Minute,
			Issuer:     "test-issuer",
			Audience:   "test-audience",
			Algorithm:  "HS256",
		},
		Ports: PortConfig{
			HTTP:    8080,
			GRPC:    9090,
			Metrics: 9091,
			Health:  8081,
		},
		Metadata: map[string]string{
			"framework": "swit",
			"type":      "microservice",
			"tier":      "backend",
		},
		ModulePath: "github.com/example/test-service",
		OutputDir:  "./output",
	}
}

// CreateMockAPIConfig creates a test API configuration
func CreateMockAPIConfig() APIConfig {
	return APIConfig{
		Name:    "test-api",
		Version: "v1",
		Service: "test-service",
		Methods: []HTTPMethod{
			{
				Name:        "GetUser",
				Method:      "GET",
				Path:        "/api/v1/users/{id}",
				Description: "Get user by ID",
				Request: RequestModel{
					Type: "GetUserRequest",
					Fields: []Field{
						{
							Name:     "id",
							Type:     "string",
							Required: true,
						},
					},
				},
				Response: ResponseModel{
					Type: "GetUserResponse",
					Fields: []Field{
						{
							Name: "user",
							Type: "User",
						},
					},
				},
				Auth:       true,
				Middleware: []string{"auth", "rate-limit"},
				Tags:       []string{"users"},
			},
		},
		GRPCMethods: []GRPCMethod{
			{
				Name:        "GetUser",
				Description: "Get user by ID via gRPC",
				Request: RequestModel{
					Type: "GetUserRequest",
					Fields: []Field{
						{
							Name:     "id",
							Type:     "string",
							Required: true,
						},
					},
				},
				Response: ResponseModel{
					Type: "GetUserResponse",
					Fields: []Field{
						{
							Name: "user",
							Type: "User",
						},
					},
				},
				Streaming: StreamingNone,
			},
		},
		Models: []Model{
			{
				Name:        "User",
				Description: "User model",
				Fields: []Field{
					{
						Name:     "id",
						Type:     "string",
						Required: true,
					},
					{
						Name:     "name",
						Type:     "string",
						Required: true,
					},
				},
			},
			{
				Name:        "UserList",
				Description: "User list model",
				Fields: []Field{
					{
						Name: "users",
						Type: "[]User",
					},
					{
						Name: "total",
						Type: "int",
					},
				},
			},
		},
		Auth:     true,
		Metadata: map[string]string{"version": "v1"},
	}
}

// CreateMockModelConfig creates a test model configuration
func CreateMockModelConfig() ModelConfig {
	return ModelConfig{
		Name:    "User",
		Package: "models",
		Fields: []Field{
			{
				Name:        "id",
				Type:        "string",
				Tags:        map[string]string{"json": "id", "db": "id"},
				Required:    true,
				Unique:      true,
				Index:       true,
				Description: "User unique identifier",
			},
			{
				Name:        "name",
				Type:        "string",
				Tags:        map[string]string{"json": "name", "db": "name"},
				Required:    true,
				Description: "User full name",
				Validation: []ValidationRule{
					{
						Type:    "min_length",
						Value:   2,
						Message: "Name must be at least 2 characters",
					},
				},
			},
			{
				Name:        "email",
				Type:        "string",
				Tags:        map[string]string{"json": "email", "db": "email"},
				Required:    true,
				Unique:      true,
				Description: "User email address",
			},
		},
		Table:      "users",
		Database:   "main",
		CRUD:       true,
		Validation: true,
		Timestamps: true,
		SoftDelete: false,
		Indexes: []Index{
			{
				Name:   "idx_user_email",
				Fields: []string{"email"},
				Unique: true,
				Type:   "btree",
			},
		},
		Relations: []Relation{
			{
				Type:       "has_many",
				Model:      "Post",
				ForeignKey: "user_id",
				LocalKey:   "id",
			},
		},
		Metadata: map[string]string{"table": "users"},
	}
}

// CreateMockProjectConfig creates a test project configuration
func CreateMockProjectConfig() ProjectConfig {
	return ProjectConfig{
		Name:        "test-project",
		Type:        ProjectTypeMicroservice,
		Description: "Test microservice project",
		Author:      "Test Author",
		License:     "MIT",
		ModulePath:  "github.com/example/test-project",
		GoVersion:   "1.19",
		Services:    []ServiceConfig{CreateMockServiceConfig()},
		Infrastructure: Infrastructure{
			Database: DatabaseConfig{
				Type: "mysql",
				Host: "localhost",
				Port: 3306,
			},
			Cache: CacheConfig{
				Type: "redis",
				Host: "localhost",
				Port: 6379,
			},
		},
		Metadata:  map[string]string{"framework": "swit"},
		OutputDir: "./output",
	}
}

// CreateMockExecutionContext creates a test execution context
func CreateMockExecutionContext() ExecutionContext {
	return ExecutionContext{
		WorkDir:    "/test/workdir",
		ConfigFile: "/test/config.yaml",
		Verbose:    true,
		NoColor:    false,
		Options: map[string]interface{}{
			"output": "/test/output",
			"format": "yaml",
		},
		Logger:    &MockLogger{},
		StartTime: time.Now(),
	}
}

// Helper validation functions for testing

// ValidateInputAlphanumeric validates that input contains only alphanumeric characters
func ValidateInputAlphanumeric(input string) error {
	if input == "" {
		return NewInvalidInputError("Input cannot be empty")
	}
	// Simple alphanumeric validation for testing
	for _, r := range input {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return NewInvalidInputError("Input must be alphanumeric")
		}
	}
	return nil
}

// ValidateInputEmail validates that input is a valid email format
func ValidateInputEmail(input string) error {
	if input == "" {
		return NewInvalidInputError("Email cannot be empty")
	}
	// Simple email validation for testing
	if !contains(input, "@") || !contains(input, ".") {
		return NewInvalidInputError("Invalid email format")
	}
	return nil
}

// ValidateInputMinLength validates that input has minimum length
func ValidateInputMinLength(minLength int) InputValidator {
	return func(input string) error {
		if len(input) < minLength {
			return NewInvalidInputError("Input too short", fmt.Sprintf("minimum length: %d", minLength))
		}
		return nil
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)*2 && containsInner(s[len(substr):len(s)-len(substr)], substr))))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
