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

package generator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/innovationmech/swit/internal/switctl/testutil"
)

// GeneratorTestSuite tests the main Generator implementation
type GeneratorTestSuite struct {
	suite.Suite
	generator      *Generator
	templateEngine *testutil.MockTemplateEngine
	fileSystem     *testutil.MockFileSystem
	logger         interfaces.Logger
}

func TestGeneratorTestSuite(t *testing.T) {
	suite.Run(t, new(GeneratorTestSuite))
}

func (s *GeneratorTestSuite) SetupTest() {
	s.templateEngine = testutil.NewMockTemplateEngine()
	s.fileSystem = testutil.NewMockFileSystem()
	s.logger = testutil.NewNoOpLogger()

	s.generator = NewGenerator(s.templateEngine, s.fileSystem, s.logger)
}

func (s *GeneratorTestSuite) TearDownTest() {
	// Reset all mocks
	s.templateEngine.AssertExpectations(s.T())
	s.fileSystem.AssertExpectations(s.T())
	// No need to assert expectations on NoOpLogger
}

func (s *GeneratorTestSuite) TestNewGenerator() {
	// Test basic constructor
	generator := NewGenerator(s.templateEngine, s.fileSystem, s.logger)

	s.NotNil(generator)
	s.NotNil(generator.serviceGenerator)
	s.NotNil(generator.apiGenerator)
	s.NotNil(generator.modelGenerator)
	s.NotNil(generator.middlewareGenerator)
	s.Equal(s.templateEngine, generator.templateEngine)
	s.Equal(s.fileSystem, generator.fileSystem)
	s.Equal(s.logger, generator.logger)
}

func (s *GeneratorTestSuite) TestNewGenerator_NilDependencies() {
	// Test with nil dependencies - should not panic but may not work properly
	s.NotPanics(func() {
		NewGenerator(nil, nil, nil)
	})
}

// TestGenerateService tests service generation through main generator
func (s *GeneratorTestSuite) TestGenerateService() {
	config := testutil.TestServiceConfig()

	// Set up expectations for service generation
	s.setupServiceGenerationExpectations(config)

	err := s.generator.GenerateService(config)
	s.NoError(err)
}

func (s *GeneratorTestSuite) TestGenerateService_InvalidConfig() {
	config := testutil.TestServiceConfig()
	config.Name = "" // Invalid - empty name

	err := s.generator.GenerateService(config)
	s.Error(err)
	s.Contains(err.Error(), "invalid service configuration")
}

// TestGenerateAPI tests API generation through main generator
func (s *GeneratorTestSuite) TestGenerateAPI() {
	config := testutil.TestAPIConfig()

	// Set up expectations for API generation
	s.setupAPIGenerationExpectations(config)

	err := s.generator.GenerateAPI(config)
	s.NoError(err)
}

func (s *GeneratorTestSuite) TestGenerateAPI_InvalidConfig() {
	config := testutil.TestAPIConfig()
	config.Name = "" // Invalid - empty name

	err := s.generator.GenerateAPI(config)
	s.Error(err)
	s.Contains(err.Error(), "invalid API configuration")
}

func (s *GeneratorTestSuite) TestGenerateAPI_WithHTTPMethods() {
	config := testutil.TestAPIConfig()
	config.Methods = []interfaces.HTTPMethod{
		{
			Name:        "GetUser",
			Method:      "GET",
			Path:        "/users/{id}",
			Description: "Get user by ID",
			Response: interfaces.ResponseModel{
				Type: "UserResponse",
				Fields: []interfaces.Field{
					{Name: "id", Type: "uint"},
					{Name: "name", Type: "string"},
				},
			},
		},
		{
			Name:   "CreateUser",
			Method: "POST",
			Path:   "/users",
			Request: interfaces.RequestModel{
				Type: "CreateUserRequest",
				Fields: []interfaces.Field{
					{Name: "name", Type: "string"},
					{Name: "email", Type: "string"},
				},
			},
			Response: interfaces.ResponseModel{
				Type: "UserResponse",
				Fields: []interfaces.Field{
					{Name: "id", Type: "uint"},
					{Name: "name", Type: "string"},
				},
			},
		},
	}

	s.setupAPIGenerationExpectations(config)

	err := s.generator.GenerateAPI(config)
	s.NoError(err)
}

func (s *GeneratorTestSuite) TestGenerateAPI_WithGRPCMethods() {
	config := testutil.TestAPIConfig()
	config.GRPCMethods = []interfaces.GRPCMethod{
		{
			Name:        "GetUser",
			Description: "Get user by ID",
			Request: interfaces.RequestModel{
				Type: "GetUserRequest",
				Fields: []interfaces.Field{
					{Name: "id", Type: "uint"},
				},
			},
			Response: interfaces.ResponseModel{
				Type: "GetUserResponse",
				Fields: []interfaces.Field{
					{Name: "user", Type: "User"},
				},
			},
		},
	}

	s.setupAPIGenerationExpectations(config)

	err := s.generator.GenerateAPI(config)
	s.NoError(err)
}

// TestGenerateModel tests model generation through main generator
func (s *GeneratorTestSuite) TestGenerateModel() {
	config := testutil.TestModelConfig()

	// Set up expectations for model generation
	s.setupModelGenerationExpectations(config)

	err := s.generator.GenerateModel(config)
	s.NoError(err)
}

func (s *GeneratorTestSuite) TestGenerateModel_InvalidConfig() {
	config := testutil.TestModelConfig()
	config.Name = "" // Invalid - empty name

	err := s.generator.GenerateModel(config)
	s.Error(err)
	s.Contains(err.Error(), "invalid model configuration")
}

func (s *GeneratorTestSuite) TestGenerateModel_WithCRUD() {
	config := testutil.TestModelConfig()
	config.CRUD = true
	config.Database = "mysql"

	s.setupModelGenerationExpectations(config)

	err := s.generator.GenerateModel(config)
	s.NoError(err)
}

func (s *GeneratorTestSuite) TestGenerateModel_WithoutCRUD() {
	config := testutil.TestModelConfig()
	config.CRUD = false

	s.setupModelGenerationExpectations(config)

	err := s.generator.GenerateModel(config)
	s.NoError(err)
}

// TestGenerateMiddleware tests middleware generation through main generator
func (s *GeneratorTestSuite) TestGenerateMiddleware() {
	config := testutil.TestMiddlewareConfig()

	// Set up expectations for middleware generation
	s.setupMiddlewareGenerationExpectations(config)

	err := s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

func (s *GeneratorTestSuite) TestGenerateMiddleware_InvalidConfig() {
	config := testutil.TestMiddlewareConfig()
	config.Name = "" // Invalid - empty name

	err := s.generator.GenerateMiddleware(config)
	s.Error(err)
	s.Contains(err.Error(), "invalid middleware configuration")
}

func (s *GeneratorTestSuite) TestGenerateMiddleware_HTTPType() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "http"

	s.setupMiddlewareGenerationExpectations(config)

	err := s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

func (s *GeneratorTestSuite) TestGenerateMiddleware_GRPCType() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "grpc"

	s.setupMiddlewareGenerationExpectations(config)

	err := s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

func (s *GeneratorTestSuite) TestGenerateMiddleware_BothType() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "both"

	s.setupMiddlewareGenerationExpectations(config)

	err := s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

// Test concurrent generation operations
func (s *GeneratorTestSuite) TestConcurrentGeneration() {
	configs := []interfaces.ServiceConfig{
		testutil.TestServiceConfig(),
		testutil.TestServiceConfig(),
		testutil.TestServiceConfig(),
	}

	// Make each config unique with valid lowercase names
	for i := range configs {
		configs[i].Name = configs[i].Name + "-" + string(rune('a'+i))
	}

	// Set up expectations for all configurations with flexible mock matching
	s.setupServiceGenerationExpectations(configs[0], 0)

	// Run concurrent generation
	done := make(chan error, len(configs))
	for _, config := range configs {
		go func(cfg interfaces.ServiceConfig) {
			done <- s.generator.GenerateService(cfg)
		}(config)
	}

	// Wait for all to complete
	for i := 0; i < len(configs); i++ {
		err := <-done
		s.NoError(err)
	}
}

// Test generation with template engine failures
func (s *GeneratorTestSuite) TestGenerateService_TemplateEngineFailure() {
	config := testutil.TestServiceConfig()

	// Mock file system operations that happen before template error
	s.fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil).Maybe()

	// Mock template engine to return error
	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(nil, assert.AnError)

	// No logger expectations needed with NoOpLogger

	err := s.generator.GenerateService(config)
	s.Error(err)
}

// Test generation with file system failures
func (s *GeneratorTestSuite) TestGenerateService_FileSystemFailure() {
	config := testutil.TestServiceConfig()

	// Setup successful template operations
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("test.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("test content"), nil)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Maybe()
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("test content"), nil).Maybe()

	// Mock file system to return error - this is what we're testing
	s.fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(assert.AnError).Maybe()
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(assert.AnError).Maybe()

	// No logger expectations needed with NoOpLogger

	err := s.generator.GenerateService(config)
	s.Error(err)
}

// Test memory usage and performance
func (s *GeneratorTestSuite) TestGenerateService_MemoryUsage() {
	config := testutil.TestServiceConfig()

	// Create large template data to test memory usage
	config.Metadata = make(map[string]string)
	for i := 0; i < 1000; i++ {
		config.Metadata[string(rune('A'+i%26))] = "test value " + string(rune('0'+i%10))
	}

	s.setupServiceGenerationExpectations(config)

	start := time.Now()
	err := s.generator.GenerateService(config)
	duration := time.Since(start)

	s.NoError(err)
	s.True(duration < 5*time.Second, "Generation should complete within reasonable time")
}

// Helper methods for setting up expectations

func (s *GeneratorTestSuite) setupServiceGenerationExpectations(_ interfaces.ServiceConfig, times ...int) {
	// Mock template operations
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("service.go").Maybe()
	mockTemplate.On("Render", mock.Anything).Return([]byte("package main\n// Generated service"), nil).Maybe()

	if len(times) > 0 && times[0] > 0 {
		callCount := times[0]
		s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Times(callCount)
		s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("package main\n// Generated service"), nil).Times(callCount)
		s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Times(callCount)
	} else {
		s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Maybe()
		s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("package main\n// Generated service"), nil).Maybe()
		s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Maybe()
	}

	// Mock file system operations
	s.fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil).Maybe()

	// Mock logger operations
	// No logger expectations needed with NoOpLogger
}

func (s *GeneratorTestSuite) setupAPIGenerationExpectations(_ interfaces.APIConfig) {
	// Mock template operations for API generation
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("api.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("package main\n// Generated API"), nil)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil)
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("package main\n// Generated API"), nil)

	// Mock file system operations
	s.fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil)
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil)

	// Mock logger operations
	// No logger expectations needed with NoOpLogger
}

func (s *GeneratorTestSuite) setupModelGenerationExpectations(_ interfaces.ModelConfig) {
	// Mock template operations for model generation
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("model.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("package main\n// Generated model"), nil)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Maybe()
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("package main\n// Generated model"), nil).Maybe()

	// Mock file system operations - model generation can call multiple files
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Maybe()

	// Mock logger operations
	// No logger expectations needed with NoOpLogger
}

func (s *GeneratorTestSuite) setupMiddlewareGenerationExpectations(_ interfaces.MiddlewareConfig) {
	// Mock template operations for middleware generation
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("middleware.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("package main\n// Generated middleware"), nil)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Maybe()
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("package main\n// Generated middleware"), nil).Maybe()

	// Mock file system operations
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Maybe()

	// Mock logger operations
	// No logger expectations needed with NoOpLogger
}

// Benchmark tests
func BenchmarkGenerator_GenerateService(b *testing.B) {
	templateEngine := testutil.NewMockTemplateEngine()
	fileSystem := testutil.NewMockFileSystem()
	logger := testutil.NewNoOpLogger()

	generator := NewGenerator(templateEngine, fileSystem, logger)
	config := testutil.TestServiceConfig()

	// Setup mock expectations
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("service.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("test content"), nil)

	templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil)
	templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("test content"), nil)
	fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil)
	fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil)
	// No logger expectations needed with NoOpLogger

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.GenerateService(config)
	}
}

func BenchmarkGenerator_GenerateAPI(b *testing.B) {
	templateEngine := testutil.NewMockTemplateEngine()
	fileSystem := testutil.NewMockFileSystem()
	logger := testutil.NewNoOpLogger()

	generator := NewGenerator(templateEngine, fileSystem, logger)
	config := testutil.TestAPIConfig()

	// Setup mock expectations
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("api.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("test content"), nil)

	templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil)
	templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("test content"), nil)
	fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil)
	fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil)
	// No logger expectations needed with NoOpLogger

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.GenerateAPI(config)
	}
}

// Edge case tests
func (s *GeneratorTestSuite) TestGenerateWithEmptyMetadata() {
	config := testutil.TestServiceConfig()
	config.Metadata = nil

	s.setupServiceGenerationExpectations(config)

	err := s.generator.GenerateService(config)
	s.NoError(err)
}

func (s *GeneratorTestSuite) TestGenerateWithLargeFields() {
	config := testutil.TestModelConfig()

	// Add many fields to test large model generation
	for i := 0; i < 100; i++ {
		config.Fields = append(config.Fields, interfaces.Field{
			Name: "field" + string(rune('A'+i%26)) + string(rune('0'+i%10)),
			Type: "string",
			Tags: map[string]string{
				"json": "field" + string(rune('a'+i%26)) + string(rune('0'+i%10)),
				"db":   "field" + string(rune('a'+i%26)) + string(rune('0'+i%10)),
			},
		})
	}

	s.setupModelGenerationExpectations(config)

	err := s.generator.GenerateModel(config)
	s.NoError(err)
}

// Error recovery tests
func (s *GeneratorTestSuite) TestGenerateService_ErrorRecovery() {
	config := testutil.TestServiceConfig()

	// Mock file system operations that happen before template error
	s.fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil).Maybe()

	// First call fails
	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(nil, assert.AnError).Once()
	// No logger expectations needed with NoOpLogger

	err1 := s.generator.GenerateService(config)
	s.Error(err1)

	// Reset mocks for second call
	s.templateEngine.ExpectedCalls = nil
	s.fileSystem.ExpectedCalls = nil

	// Second call succeeds
	s.setupServiceGenerationExpectations(config)

	err2 := s.generator.GenerateService(config)
	s.NoError(err2)
}

// Integration-style tests with realistic scenarios
func (s *GeneratorTestSuite) TestGenerateCompleteAPIWithModels() {
	// Test generating a complete API with multiple models
	apiConfig := testutil.TestAPIConfig()
	apiConfig.Models = []interfaces.Model{
		{
			Name:        "User",
			Description: "User entity",
			Fields: []interfaces.Field{
				{Name: "id", Type: "uint", Tags: map[string]string{"json": "id"}},
				{Name: "name", Type: "string", Tags: map[string]string{"json": "name"}},
				{Name: "email", Type: "string", Tags: map[string]string{"json": "email"}},
			},
		},
		{
			Name:        "Order",
			Description: "Order entity",
			Fields: []interfaces.Field{
				{Name: "id", Type: "uint", Tags: map[string]string{"json": "id"}},
				{Name: "user_id", Type: "uint", Tags: map[string]string{"json": "user_id"}},
				{Name: "total", Type: "float64", Tags: map[string]string{"json": "total"}},
			},
		},
	}

	s.setupAPIGenerationExpectations(apiConfig)

	err := s.generator.GenerateAPI(apiConfig)
	s.NoError(err)
}

func (s *GeneratorTestSuite) TestGenerateServiceWithAllFeatures() {
	// Test generating a service with all features enabled
	config := testutil.TestServiceConfig()
	config.Features = interfaces.ServiceFeatures{
		Database:       true,
		Authentication: true,
		Cache:          true,
		MessageQueue:   true,
		Monitoring:     true,
		Tracing:        true,
		Logging:        true,
		HealthCheck:    true,
		Metrics:        true,
		Docker:         true,
		Kubernetes:     true,
	}

	s.setupServiceGenerationExpectations(config)

	err := s.generator.GenerateService(config)
	s.NoError(err)
}
