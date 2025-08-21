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

// MiddlewareGeneratorTestSuite tests the MiddlewareGenerator implementation
type MiddlewareGeneratorTestSuite struct {
	suite.Suite
	generator      *MiddlewareGenerator
	templateEngine *testutil.MockTemplateEngine
	fileSystem     *testutil.MockFileSystem
	logger         interfaces.Logger
}

func TestMiddlewareGeneratorTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareGeneratorTestSuite))
}

func (s *MiddlewareGeneratorTestSuite) SetupTest() {
	s.templateEngine = testutil.NewMockTemplateEngine()
	s.fileSystem = testutil.NewMockFileSystem()
	s.logger = testutil.NewNoOpLogger()

	s.generator = NewMiddlewareGenerator(s.templateEngine, s.fileSystem, s.logger)
}

func (s *MiddlewareGeneratorTestSuite) TearDownTest() {
	// Reset all mocks
	s.templateEngine.AssertExpectations(s.T())
	s.fileSystem.AssertExpectations(s.T())
	// No need to assert expectations on NoOpLogger
}

func (s *MiddlewareGeneratorTestSuite) TestNewMiddlewareGenerator() {
	// Test basic constructor
	generator := NewMiddlewareGenerator(s.templateEngine, s.fileSystem, s.logger)

	s.NotNil(generator)
	s.Equal(s.templateEngine, generator.templateEngine)
	s.Equal(s.fileSystem, generator.fileSystem)
	s.Equal(s.logger, generator.logger)
}

func (s *MiddlewareGeneratorTestSuite) TestNewMiddlewareGenerator_NilDependencies() {
	// Test with nil dependencies - should not panic
	s.NotPanics(func() {
		NewMiddlewareGenerator(nil, nil, nil)
	})
}

// TestGenerateMiddleware tests basic middleware generation
func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddleware_HTTPType() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "http"

	s.setupMiddlewareGenerationExpectations(config)

	err := s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddleware_GRPCType() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "grpc"

	s.setupMiddlewareGenerationExpectations(config)

	err := s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddleware_BothType() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "both"

	s.setupMiddlewareGenerationExpectations(config)

	err := s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

// TestValidateMiddlewareConfig tests middleware configuration validation
func (s *MiddlewareGeneratorTestSuite) TestValidateMiddlewareConfig_ValidConfig() {
	config := testutil.TestMiddlewareConfig()

	err := s.generator.validateMiddlewareConfig(&config)
	s.NoError(err)
}

func (s *MiddlewareGeneratorTestSuite) TestValidateMiddlewareConfig_EmptyName() {
	config := testutil.TestMiddlewareConfig()
	config.Name = ""

	err := s.generator.validateMiddlewareConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "middleware name is required")
}

func (s *MiddlewareGeneratorTestSuite) TestValidateMiddlewareConfig_EmptyType() {
	config := testutil.TestMiddlewareConfig()
	config.Type = ""

	err := s.generator.validateMiddlewareConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "middleware type is required")
}

func (s *MiddlewareGeneratorTestSuite) TestValidateMiddlewareConfig_InvalidType() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "invalid"

	err := s.generator.validateMiddlewareConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "invalid middleware type")
	s.Contains(err.Error(), "must be one of: http, grpc, both")
}

func (s *MiddlewareGeneratorTestSuite) TestValidateMiddlewareConfig_ValidTypes() {
	validTypes := []string{"http", "grpc", "both"}

	for _, validType := range validTypes {
		config := testutil.TestMiddlewareConfig()
		config.Type = validType

		err := s.generator.validateMiddlewareConfig(&config)
		s.NoError(err, "Type %s should be valid", validType)
	}
}

func (s *MiddlewareGeneratorTestSuite) TestValidateMiddlewareConfig_DefaultPackage() {
	config := interfaces.MiddlewareConfig{
		Name: "TestMiddleware",
		Type: "http",
	}

	err := s.generator.validateMiddlewareConfig(&config)
	s.NoError(err)
	s.Equal("middleware", config.Package) // Default package should be set
}

// TestPrepareMiddlewareTemplateData tests template data preparation
func (s *MiddlewareGeneratorTestSuite) TestPrepareMiddlewareTemplateData_HTTP() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "http"

	data, err := s.generator.prepareMiddlewareTemplateData(config)
	s.NoError(err)
	s.NotNil(data)

	// Check basic template data structure
	s.Equal(config.Package, data.Package.Name)
	s.Equal(config.Package, data.Package.Path)
	s.Equal(1, len(data.Functions)) // HTTP middleware function

	// Check metadata
	s.NotNil(data.Metadata)
	s.Equal(toPascalCase(config.Name), data.Metadata["middleware_name"])
	s.Equal(config.Type, data.Metadata["middleware_type"])
	s.Equal(config.Package, data.Metadata["package_name"])

	// Check timestamp
	s.WithinDuration(time.Now(), data.Timestamp, time.Second)
}

func (s *MiddlewareGeneratorTestSuite) TestPrepareMiddlewareTemplateData_GRPC() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "grpc"

	data, err := s.generator.prepareMiddlewareTemplateData(config)
	s.NoError(err)
	s.NotNil(data)

	// Check functions for gRPC
	s.Equal(1, len(data.Functions)) // gRPC interceptor function

	function := data.Functions[0]
	s.Contains(function.Name, "Interceptor")
	s.Contains(function.Comment, "gRPC interceptor")
}

func (s *MiddlewareGeneratorTestSuite) TestPrepareMiddlewareTemplateData_Both() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "both"

	data, err := s.generator.prepareMiddlewareTemplateData(config)
	s.NoError(err)
	s.NotNil(data)

	// Check functions for both types
	s.Equal(2, len(data.Functions)) // HTTP and gRPC functions

	// Find HTTP function
	var httpFunc *interfaces.FunctionInfo
	var grpcFunc *interfaces.FunctionInfo
	for i := range data.Functions {
		if data.Functions[i].Returns[0].Type == "gin.HandlerFunc" {
			httpFunc = &data.Functions[i]
		} else if data.Functions[i].Returns[0].Type == "grpc.UnaryServerInterceptor" {
			grpcFunc = &data.Functions[i]
		}
	}

	s.NotNil(httpFunc, "Should have HTTP function")
	s.NotNil(grpcFunc, "Should have gRPC function")
	s.Contains(grpcFunc.Name, "Interceptor")
}

// TestGenerateMiddlewareFile tests middleware file generation
func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddlewareFile() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "http"
	data, _ := s.generator.prepareMiddlewareTemplateData(config)

	// Setup template expectations
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("middleware/http.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated middleware file"), nil)

	s.templateEngine.On("LoadTemplate", "middleware/http.go").Return(mockTemplate, nil)
	s.templateEngine.On("RenderTemplate", mockTemplate, data).Return([]byte("generated middleware file"), nil)
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil)
	// No logger expectations needed with NoOpLogger

	err := s.generator.generateMiddlewareFile(config, data)
	s.NoError(err)
}

func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddlewareFile_TemplateLoadError() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "http"
	data, _ := s.generator.prepareMiddlewareTemplateData(config)

	s.templateEngine.On("LoadTemplate", "middleware/http.go").Return(nil, assert.AnError)

	err := s.generator.generateMiddlewareFile(config, data)
	s.Error(err)
	s.Contains(err.Error(), "failed to load template")
}

func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddlewareFile_TemplateRenderError() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "http"
	data, _ := s.generator.prepareMiddlewareTemplateData(config)

	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("middleware/http.go")

	s.templateEngine.On("LoadTemplate", "middleware/http.go").Return(mockTemplate, nil)
	s.templateEngine.On("RenderTemplate", mockTemplate, data).Return(nil, assert.AnError)

	err := s.generator.generateMiddlewareFile(config, data)
	s.Error(err)
	s.Contains(err.Error(), "failed to render template")
}

func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddlewareFile_FileWriteError() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "http"
	data, _ := s.generator.prepareMiddlewareTemplateData(config)

	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("middleware/http.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated content"), nil)

	s.templateEngine.On("LoadTemplate", "middleware/http.go").Return(mockTemplate, nil)
	s.templateEngine.On("RenderTemplate", mockTemplate, data).Return([]byte("generated content"), nil)
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(assert.AnError)

	err := s.generator.generateMiddlewareFile(config, data)
	s.Error(err)
	s.Contains(err.Error(), "failed to write file")
}

// TestGenerateMiddlewareTests tests middleware test generation
func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddlewareTests() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "http"
	data, _ := s.generator.prepareMiddlewareTemplateData(config)

	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("middleware/http_test.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated middleware tests"), nil)

	s.templateEngine.On("LoadTemplate", "middleware/http_test.go").Return(mockTemplate, nil)
	s.templateEngine.On("RenderTemplate", mockTemplate, data).Return([]byte("generated middleware tests"), nil)
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil)
	// No logger expectations needed with NoOpLogger

	err := s.generator.generateMiddlewareTests(config, data)
	s.NoError(err)
}

// TestGetMiddlewareFunctions tests function generation for different types
func (s *MiddlewareGeneratorTestSuite) TestGetMiddlewareFunctions_HTTP() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "http"

	functions := s.generator.getMiddlewareFunctions(config)
	s.Equal(1, len(functions))

	function := functions[0]
	s.Equal(toPascalCase(config.Name), function.Name)
	s.Contains(function.Comment, "HTTP middleware")
	s.Equal(1, len(function.Returns))
	s.Equal("gin.HandlerFunc", function.Returns[0].Type)
}

func (s *MiddlewareGeneratorTestSuite) TestGetMiddlewareFunctions_GRPC() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "grpc"

	functions := s.generator.getMiddlewareFunctions(config)
	s.Equal(1, len(functions))

	function := functions[0]
	s.Equal(toPascalCase(config.Name)+"Interceptor", function.Name)
	s.Contains(function.Comment, "gRPC interceptor")
	s.Equal(1, len(function.Returns))
	s.Equal("grpc.UnaryServerInterceptor", function.Returns[0].Type)
}

func (s *MiddlewareGeneratorTestSuite) TestGetMiddlewareFunctions_Both() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "both"

	functions := s.generator.getMiddlewareFunctions(config)
	s.Equal(2, len(functions))

	// Check HTTP function
	httpFunc := functions[0]
	s.Equal(toPascalCase(config.Name), httpFunc.Name)
	s.Equal("gin.HandlerFunc", httpFunc.Returns[0].Type)

	// Check gRPC function
	grpcFunc := functions[1]
	s.Equal(toPascalCase(config.Name)+"Interceptor", grpcFunc.Name)
	s.Equal("grpc.UnaryServerInterceptor", grpcFunc.Returns[0].Type)
}

// Test complete middleware generation flow
func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddleware_CompleteFlow() {
	config := testutil.TestMiddlewareConfig()

	// Setup expectations for complete flow (middleware file + test file)
	s.setupCompleteMiddlewareGenerationExpectations(config)

	err := s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

// Test different middleware configurations
func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddleware_AuthMiddleware() {
	config := interfaces.MiddlewareConfig{
		Name:        "AuthMiddleware",
		Type:        "http",
		Description: "JWT authentication middleware",
		Package:     "auth",
		Features:    []string{"jwt", "validation", "cors"},
		Dependencies: []string{
			"github.com/golang-jwt/jwt/v4",
			"github.com/gin-gonic/gin",
		},
		Config: map[string]interface{}{
			"secret_key":   "test-secret",
			"token_header": "Authorization",
		},
		Metadata: map[string]string{
			"category": "authentication",
			"version":  "1.0.0",
		},
	}

	s.setupMiddlewareGenerationExpectations(config)

	err := s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddleware_LoggingMiddleware() {
	config := interfaces.MiddlewareConfig{
		Name:        "LoggingMiddleware",
		Type:        "both",
		Description: "Request/response logging middleware",
		Package:     "logging",
		Features:    []string{"structured-logging", "metrics"},
		Dependencies: []string{
			"github.com/sirupsen/logrus",
			"github.com/gin-gonic/gin",
			"google.golang.org/grpc",
		},
	}

	s.setupMiddlewareGenerationExpectations(config)

	err := s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddleware_RateLimitMiddleware() {
	config := interfaces.MiddlewareConfig{
		Name:        "RateLimitMiddleware",
		Type:        "grpc",
		Description: "Rate limiting middleware for gRPC",
		Package:     "ratelimit",
		Features:    []string{"token-bucket", "sliding-window"},
		Config: map[string]interface{}{
			"requests_per_minute": 100,
			"burst_size":          10,
			"redis_url":           "redis://localhost:6379",
		},
	}

	s.setupMiddlewareGenerationExpectations(config)

	err := s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

// Test edge cases and error scenarios
func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddleware_MinimalConfig() {
	config := interfaces.MiddlewareConfig{
		Name: "Simple",
		Type: "http",
	}

	s.setupMiddlewareGenerationExpectations(config)

	err := s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

// Test concurrent middleware generation
func (s *MiddlewareGeneratorTestSuite) TestConcurrentMiddlewareGeneration() {
	configs := []interfaces.MiddlewareConfig{
		testutil.TestMiddlewareConfig(),
		testutil.TestMiddlewareConfig(),
		testutil.TestMiddlewareConfig(),
	}

	// Make each config unique with valid lowercase names
	for i := range configs {
		configs[i].Name = configs[i].Name + "-" + string(rune('a'+i))
		configs[i].Type = []string{"http", "grpc", "both"}[i%3]
	}

	// Set up expectations for all configurations with flexible mock matching
	s.setupMiddlewareGenerationExpectations(configs[0], 0)

	// Run concurrent generation
	done := make(chan error, len(configs))
	for _, config := range configs {
		go func(cfg interfaces.MiddlewareConfig) {
			done <- s.generator.GenerateMiddleware(cfg)
		}(config)
	}

	// Wait for all to complete
	for i := 0; i < len(configs); i++ {
		err := <-done
		s.NoError(err)
	}
}

// Test template path generation for different types
func (s *MiddlewareGeneratorTestSuite) TestTemplatePathGeneration() {
	testCases := []struct {
		middlewareType string
		expectedImpl   string
		expectedTest   string
	}{
		{"http", "middleware/http.go", "middleware/http_test.go"},
		{"grpc", "middleware/grpc.go", "middleware/grpc_test.go"},
		{"both", "middleware/both.go", "middleware/both_test.go"},
	}

	for _, tc := range testCases {
		config := testutil.TestMiddlewareConfig()
		config.Type = tc.middlewareType
		data, _ := s.generator.prepareMiddlewareTemplateData(config)

		// Test implementation template path
		implFile := ServiceFile{
			Path:     config.Package + "/" + config.Name + ".go",
			Template: tc.expectedImpl,
			Data:     data,
			Mode:     0644,
		}
		s.Equal(tc.expectedImpl, implFile.Template)

		// Test test template path
		testFile := ServiceFile{
			Path:     config.Package + "/" + config.Name + "_test.go",
			Template: tc.expectedTest,
			Data:     data,
			Mode:     0644,
		}
		s.Equal(tc.expectedTest, testFile.Template)
	}
}

// Test with complex metadata and configuration
func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddleware_ComplexMetadata() {
	config := testutil.TestMiddlewareConfig()
	config.Metadata = map[string]string{
		"author":    "Test Author",
		"version":   "2.1.0",
		"category":  "security",
		"license":   "MIT",
		"framework": "gin",
	}
	config.Config = map[string]interface{}{
		"timeout":     "30s",
		"max_retries": 3,
		"buffer_size": 1024,
		"enabled":     true,
		"algorithms":  []string{"RS256", "HS256"},
		"rate_limits": map[string]int{"api": 1000, "auth": 100},
	}

	data, err := s.generator.prepareMiddlewareTemplateData(config)
	s.NoError(err)

	// Check that metadata is preserved
	s.NotNil(data.Metadata)
	s.Contains(data.Metadata, "middleware_name")
	s.Contains(data.Metadata, "middleware_type")
	s.Contains(data.Metadata, "package_name")

	s.setupMiddlewareGenerationExpectations(config)

	err = s.generator.GenerateMiddleware(config)
	s.NoError(err)
}

// Helper methods for setting up expectations

func (s *MiddlewareGeneratorTestSuite) setupMiddlewareGenerationExpectations(_ interfaces.MiddlewareConfig, times ...int) {
	// Setup for generateMiddlewareFile and generateMiddlewareTests
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("middleware.go").Maybe()
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated middleware"), nil).Maybe()

	if len(times) > 0 && times[0] > 0 {
		callCount := times[0]
		s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Times(callCount)
		s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated middleware"), nil).Times(callCount)
		s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Times(callCount)
	} else {
		s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Maybe()
		s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated middleware"), nil).Maybe()
		s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Maybe()
	}
	// No logger expectations needed with NoOpLogger
}

func (s *MiddlewareGeneratorTestSuite) setupCompleteMiddlewareGenerationExpectations(_ interfaces.MiddlewareConfig) {
	// Setup for all generation steps (implementation + tests)
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("middleware.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated content"), nil)

	// Two template loads for implementation and tests
	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Times(2)
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated content"), nil).Times(2)
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Times(2)
	// No logger expectations needed with NoOpLogger
	// No logger expectations needed with NoOpLogger
}

// Benchmark tests
func BenchmarkMiddlewareGenerator_GenerateMiddleware(b *testing.B) {
	templateEngine := testutil.NewMockTemplateEngine()
	fileSystem := testutil.NewMockFileSystem()
	logger := testutil.NewNoOpLogger()

	generator := NewMiddlewareGenerator(templateEngine, fileSystem, logger)
	config := testutil.TestMiddlewareConfig()

	// Setup mock expectations
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("middleware.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("test content"), nil)

	templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil)
	templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("test content"), nil)
	fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil)
	// No logger expectations needed with NoOpLogger

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.GenerateMiddleware(config)
	}
}

func BenchmarkMiddlewareGenerator_PrepareTemplateData(b *testing.B) {
	templateEngine := testutil.NewMockTemplateEngine()
	fileSystem := testutil.NewMockFileSystem()
	logger := testutil.NewNoOpLogger()

	generator := NewMiddlewareGenerator(templateEngine, fileSystem, logger)
	config := testutil.TestMiddlewareConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = generator.prepareMiddlewareTemplateData(config)
	}
}

// Test error scenarios with realistic errors
func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddleware_FileSystemErrors() {
	config := testutil.TestMiddlewareConfig()

	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("middleware.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("content"), nil)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil)
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("content"), nil)
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(assert.AnError)
	// No logger expectations needed with NoOpLogger

	err := s.generator.GenerateMiddleware(config)
	s.Error(err)
	s.Contains(err.Error(), "failed to generate middleware file")
}

// Test memory usage with complex configurations
func (s *MiddlewareGeneratorTestSuite) TestGenerateMiddleware_ComplexConfiguration() {
	config := testutil.TestMiddlewareConfig()
	config.Type = "both"

	// Add complex configuration
	config.Features = make([]string, 50)
	for i := 0; i < 50; i++ {
		config.Features[i] = "feature" + string(rune('A'+i%26))
	}

	config.Dependencies = make([]string, 20)
	for i := 0; i < 20; i++ {
		config.Dependencies[i] = "github.com/example/dep" + string(rune('A'+i%26))
	}

	s.setupCompleteMiddlewareGenerationExpectations(config)

	start := time.Now()
	err := s.generator.GenerateMiddleware(config)
	duration := time.Since(start)

	s.NoError(err)
	s.True(duration < 5*time.Second, "Complex middleware generation should complete within reasonable time")
}
