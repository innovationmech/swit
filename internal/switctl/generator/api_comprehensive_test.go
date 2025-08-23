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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/innovationmech/swit/internal/switctl/testutil"
)

// APIGeneratorComprehensiveTestSuite tests the APIGenerator implementation comprehensively
type APIGeneratorComprehensiveTestSuite struct {
	suite.Suite
	generator      *APIGenerator
	templateEngine *testutil.MockTemplateEngine
	fileSystem     *testutil.MockFileSystem
	logger         interfaces.Logger
}

func TestAPIGeneratorComprehensiveTestSuite(t *testing.T) {
	suite.Run(t, new(APIGeneratorComprehensiveTestSuite))
}

func (s *APIGeneratorComprehensiveTestSuite) SetupTest() {
	s.templateEngine = testutil.NewMockTemplateEngine()
	s.fileSystem = testutil.NewMockFileSystem()
	s.logger = testutil.NewNoOpLogger()

	s.generator = NewAPIGenerator(s.templateEngine, s.fileSystem, s.logger)
}

func (s *APIGeneratorComprehensiveTestSuite) TearDownTest() {
	// Reset all mocks
	s.templateEngine.AssertExpectations(s.T())
	s.fileSystem.AssertExpectations(s.T())
	// No need to assert expectations on NoOpLogger
}

func (s *APIGeneratorComprehensiveTestSuite) TestNewAPIGenerator() {
	// Test basic constructor
	generator := NewAPIGenerator(s.templateEngine, s.fileSystem, s.logger)

	s.NotNil(generator)
	s.Equal(s.templateEngine, generator.templateEngine)
	s.Equal(s.fileSystem, generator.fileSystem)
	s.Equal(s.logger, generator.logger)
}

func (s *APIGeneratorComprehensiveTestSuite) TestNewAPIGenerator_NilDependencies() {
	// Test with nil dependencies - should not panic
	s.NotPanics(func() {
		NewAPIGenerator(nil, nil, nil)
	})
}

// TestGenerateAPI tests basic API generation
func (s *APIGeneratorComprehensiveTestSuite) TestGenerateAPI_BasicHTTPEndpoints() {
	config := testutil.TestAPIConfig()
	config.GRPCMethods = nil // Only HTTP methods

	s.setupAPIGenerationExpectations(config, 5) // Handler, service, interface files

	err := s.generator.GenerateAPI(config)
	s.NoError(err)
}

func (s *APIGeneratorComprehensiveTestSuite) TestGenerateAPI_BasicGRPCEndpoints() {
	config := testutil.TestAPIConfig()
	config.Methods = nil // Only gRPC methods

	s.setupAPIGenerationExpectations(config, 7) // gRPC handler, proto, service, interface files

	err := s.generator.GenerateAPI(config)
	s.NoError(err)
}

func (s *APIGeneratorComprehensiveTestSuite) TestGenerateAPI_BothHTTPAndGRPC() {
	config := testutil.TestAPIConfig()

	s.setupAPIGenerationExpectations(config, 8) // All files for both protocols

	err := s.generator.GenerateAPI(config)
	s.NoError(err)
}

func (s *APIGeneratorComprehensiveTestSuite) TestGenerateAPI_WithModels() {
	config := testutil.TestAPIConfig()
	config.Models = []interfaces.Model{
		{
			Name:        "Product",
			Description: "Product model",
			Fields: []interfaces.Field{
				{Name: "id", Type: "string"},
				{Name: "name", Type: "string"},
				{Name: "price", Type: "float64"},
			},
		},
	}

	s.setupAPIGenerationExpectations(config, 9) // Additional model file

	err := s.generator.GenerateAPI(config)
	s.NoError(err)
}

// TestValidateAPIConfig tests API configuration validation
func (s *APIGeneratorComprehensiveTestSuite) TestValidateAPIConfig_ValidConfig() {
	config := testutil.TestAPIConfig()

	err := s.generator.validateAPIConfig(&config)
	s.NoError(err)
}

func (s *APIGeneratorComprehensiveTestSuite) TestValidateAPIConfig_EmptyName() {
	config := testutil.TestAPIConfig()
	config.Name = ""

	err := s.generator.validateAPIConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "API name is required")
}

func (s *APIGeneratorComprehensiveTestSuite) TestValidateAPIConfig_EmptyService() {
	config := testutil.TestAPIConfig()
	config.Service = ""

	err := s.generator.validateAPIConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "service name is required")
}

func (s *APIGeneratorComprehensiveTestSuite) TestValidateAPIConfig_DefaultVersion() {
	config := testutil.TestAPIConfig()
	config.Version = ""

	err := s.generator.validateAPIConfig(&config)
	s.NoError(err)
	s.Equal("v1", config.Version) // Should set default
}

func (s *APIGeneratorComprehensiveTestSuite) TestValidateAPIConfig_InvalidHTTPMethod() {
	config := testutil.TestAPIConfig()
	config.Methods = []interfaces.HTTPMethod{
		{
			Name:   "TestMethod",
			Method: "INVALID",
			Path:   "/test",
		},
	}

	err := s.generator.validateAPIConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "invalid HTTP method")
}

func (s *APIGeneratorComprehensiveTestSuite) TestValidateAPIConfig_EmptyHTTPMethodName() {
	config := testutil.TestAPIConfig()
	config.Methods = []interfaces.HTTPMethod{
		{
			Name:   "",
			Method: "GET",
			Path:   "/test",
		},
	}

	err := s.generator.validateAPIConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "HTTP method name is required")
}

func (s *APIGeneratorComprehensiveTestSuite) TestValidateAPIConfig_EmptyHTTPMethodVerb() {
	config := testutil.TestAPIConfig()
	config.Methods = []interfaces.HTTPMethod{
		{
			Name:   "TestMethod",
			Method: "",
			Path:   "/test",
		},
	}

	err := s.generator.validateAPIConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "HTTP method verb is required")
}

func (s *APIGeneratorComprehensiveTestSuite) TestValidateAPIConfig_EmptyHTTPMethodPath() {
	config := testutil.TestAPIConfig()
	config.Methods = []interfaces.HTTPMethod{
		{
			Name:   "TestMethod",
			Method: "GET",
			Path:   "",
		},
	}

	err := s.generator.validateAPIConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "HTTP method path is required")
}

func (s *APIGeneratorComprehensiveTestSuite) TestValidateAPIConfig_EmptyGRPCMethodName() {
	config := testutil.TestAPIConfig()
	config.GRPCMethods = []interfaces.GRPCMethod{
		{
			Name: "",
		},
	}

	err := s.generator.validateAPIConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "gRPC method name is required")
}

func (s *APIGeneratorComprehensiveTestSuite) TestValidateAPIConfig_ValidHTTPMethods() {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range validMethods {
		config := testutil.TestAPIConfig()
		config.Methods = []interfaces.HTTPMethod{
			{
				Name:   "TestMethod",
				Method: method,
				Path:   "/test",
			},
		}

		err := s.generator.validateAPIConfig(&config)
		s.NoError(err, "Method %s should be valid", method)
	}
}

// TestPrepareAPITemplateData tests template data preparation
func (s *APIGeneratorComprehensiveTestSuite) TestPrepareAPITemplateData() {
	config := testutil.TestAPIConfig()

	data, err := s.generator.prepareAPITemplateData(config)
	s.NoError(err)
	s.NotNil(data)

	// Check basic structure
	s.Equal(toPackageName(config.Service), data.Package.Name)
	s.Equal("internal/"+toPackageName(config.Service), data.Package.Path)
	s.Equal(config.Version, data.Package.Version)

	// Check imports
	s.Greater(len(data.Imports), 0)
	s.Contains(getImportPaths(data.Imports), "context")
	s.Contains(getImportPaths(data.Imports), "fmt")
	s.Contains(getImportPaths(data.Imports), "net/http")

	// Check functions
	s.Greater(len(data.Functions), 0)

	// Check structs
	s.Greater(len(data.Structs), 0)

	// Check interfaces
	s.Greater(len(data.Interfaces), 0)

	// Check metadata
	s.NotNil(data.Metadata)
	s.Equal(toPascalCase(config.Name), data.Metadata["api_name"])
	s.Equal(config.Version, data.Metadata["api_version"])
	s.Equal(config.Service, data.Metadata["service_name"])

	// Check timestamp
	s.WithinDuration(time.Now(), data.Timestamp, time.Second)
}

func (s *APIGeneratorComprehensiveTestSuite) TestPrepareAPITemplateData_WithAuth() {
	config := testutil.TestAPIConfig()
	config.Auth = true

	data, err := s.generator.prepareAPITemplateData(config)
	s.NoError(err)

	// Should include auth middleware import
	imports := getImportPaths(data.Imports)
	s.Contains(imports, "github.com/innovationmech/swit/pkg/middleware")
}

func (s *APIGeneratorComprehensiveTestSuite) TestPrepareAPITemplateData_WithGRPC() {
	config := testutil.TestAPIConfig()
	config.Methods = nil // Only gRPC

	data, err := s.generator.prepareAPITemplateData(config)
	s.NoError(err)

	// Should include gRPC imports
	imports := getImportPaths(data.Imports)
	s.Contains(imports, "google.golang.org/grpc")
	s.Contains(imports, "google.golang.org/grpc/codes")
	s.Contains(imports, "google.golang.org/grpc/status")
}

// TestGenerateHTTPHandlers tests HTTP handler generation
func (s *APIGeneratorComprehensiveTestSuite) TestGenerateHTTPHandlers() {
	config := testutil.TestAPIConfig()
	data, _ := s.generator.prepareAPITemplateData(config)

	s.setupHTTPHandlerExpectations(len(config.Methods) + 2) // Main handler + methods + test

	err := s.generator.generateHTTPHandlers(config, data)
	s.NoError(err)
}

func (s *APIGeneratorComprehensiveTestSuite) TestGenerateHTTPHandlers_TemplateError() {
	config := testutil.TestAPIConfig()
	data, _ := s.generator.prepareAPITemplateData(config)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(nil, assert.AnError)

	err := s.generator.generateHTTPHandlers(config, data)
	s.Error(err)
	s.Contains(err.Error(), "failed to generate HTTP handler")
}

// TestGenerateGRPCHandlers tests gRPC handler generation
func (s *APIGeneratorComprehensiveTestSuite) TestGenerateGRPCHandlers() {
	config := testutil.TestAPIConfig()
	data, _ := s.generator.prepareAPITemplateData(config)

	s.setupGRPCHandlerExpectations(2) // Handler + test

	err := s.generator.generateGRPCHandlers(config, data)
	s.NoError(err)
}

// TestGenerateProtoDefinitions tests Protocol Buffer generation
func (s *APIGeneratorComprehensiveTestSuite) TestGenerateProtoDefinitions() {
	config := testutil.TestAPIConfig()
	data, _ := s.generator.prepareAPITemplateData(config)

	expectedFiles := 1 // Service proto
	if len(config.Models) > 0 {
		expectedFiles++ // Messages proto
	}

	s.setupProtoExpectations(expectedFiles)

	err := s.generator.generateProtoDefinitions(config, data)
	s.NoError(err)
}

func (s *APIGeneratorComprehensiveTestSuite) TestGenerateProtoDefinitions_WithModels() {
	config := testutil.TestAPIConfig()
	config.Models = []interfaces.Model{
		{Name: "User", Fields: []interfaces.Field{{Name: "id", Type: "string"}}},
	}
	data, _ := s.generator.prepareAPITemplateData(config)

	s.setupProtoExpectations(2) // Service + messages

	err := s.generator.generateProtoDefinitions(config, data)
	s.NoError(err)
}

// TestGenerateServiceLayer tests service layer generation
func (s *APIGeneratorComprehensiveTestSuite) TestGenerateServiceLayer() {
	config := testutil.TestAPIConfig()
	data, _ := s.generator.prepareAPITemplateData(config)

	expectedFiles := 2 // Service + test
	if len(config.GRPCMethods) > 0 {
		expectedFiles += 2 // gRPC handler + test
	}

	s.setupServiceLayerExpectations(expectedFiles)

	err := s.generator.generateServiceLayer(config, data)
	s.NoError(err)
}

func (s *APIGeneratorComprehensiveTestSuite) TestGenerateServiceLayer_GRPCOnly() {
	config := testutil.TestAPIConfig()
	config.Methods = nil // Only gRPC
	data, _ := s.generator.prepareAPITemplateData(config)

	s.setupServiceLayerExpectations(4) // Service + test + gRPC handler + test

	err := s.generator.generateServiceLayer(config, data)
	s.NoError(err)
}

// TestGenerateInterfaces tests interface generation
func (s *APIGeneratorComprehensiveTestSuite) TestGenerateInterfaces() {
	config := testutil.TestAPIConfig()
	data, _ := s.generator.prepareAPITemplateData(config)

	s.setupInterfaceExpectations(1)

	err := s.generator.generateInterfaces(config, data)
	s.NoError(err)
}

// TestGenerateModels tests model generation
func (s *APIGeneratorComprehensiveTestSuite) TestGenerateModels() {
	config := testutil.TestAPIConfig()
	config.Models = []interfaces.Model{
		{Name: "User"},
		{Name: "Product"},
	}
	data, _ := s.generator.prepareAPITemplateData(config)

	s.setupModelExpectations(len(config.Models))

	err := s.generator.generateModels(config, data)
	s.NoError(err)
}

// TestGetAPIImports tests import generation
func (s *APIGeneratorComprehensiveTestSuite) TestGetAPIImports_HTTP() {
	config := testutil.TestAPIConfig()
	config.GRPCMethods = nil

	imports := s.generator.getAPIImports(config)
	paths := getImportPaths(imports)

	s.Contains(paths, "context")
	s.Contains(paths, "fmt")
	s.Contains(paths, "net/http")
	s.Contains(paths, "github.com/gin-gonic/gin")
	s.Contains(paths, "github.com/innovationmech/swit/pkg/transport")
}

func (s *APIGeneratorComprehensiveTestSuite) TestGetAPIImports_GRPC() {
	config := testutil.TestAPIConfig()
	config.Methods = nil

	imports := s.generator.getAPIImports(config)
	paths := getImportPaths(imports)

	s.Contains(paths, "google.golang.org/grpc")
	s.Contains(paths, "google.golang.org/grpc/codes")
	s.Contains(paths, "google.golang.org/grpc/status")
}

func (s *APIGeneratorComprehensiveTestSuite) TestGetAPIImports_WithAuth() {
	config := testutil.TestAPIConfig()
	config.Auth = true

	imports := s.generator.getAPIImports(config)
	paths := getImportPaths(imports)

	s.Contains(paths, "github.com/innovationmech/swit/pkg/middleware")
}

// TestGetAPIFunctions tests function generation
func (s *APIGeneratorComprehensiveTestSuite) TestGetAPIFunctions() {
	config := testutil.TestAPIConfig()

	functions := s.generator.getAPIFunctions(config)

	// Should have HTTP handlers + gRPC methods + constructor
	expectedCount := len(config.Methods) + len(config.GRPCMethods) + 1
	s.Equal(expectedCount, len(functions))

	// Check constructor function
	var constructorFound bool
	for _, fn := range functions {
		if fn.Name == "New"+toPascalCase(config.Name)+"Service" {
			constructorFound = true
			s.Contains(fn.Comment, "creates a new")
			break
		}
	}
	s.True(constructorFound, "Constructor function should be generated")
}

// TestGetAPIStructs tests struct generation
func (s *APIGeneratorComprehensiveTestSuite) TestGetAPIStructs() {
	config := testutil.TestAPIConfig()

	structs := s.generator.getAPIStructs(config)

	// Should have service struct, HTTP handler struct, gRPC handler struct, request/response structs
	s.GreaterOrEqual(len(structs), 3)

	// Check service struct
	var serviceStructFound bool
	for _, st := range structs {
		if st.Name == toPascalCase(config.Name)+"Service" {
			serviceStructFound = true
			s.Contains(st.Comment, "provides")
			break
		}
	}
	s.True(serviceStructFound, "Service struct should be generated")
}

// TestGetAPIInterfaces tests interface generation
func (s *APIGeneratorComprehensiveTestSuite) TestGetAPIInterfaces() {
	config := testutil.TestAPIConfig()

	interfaces := s.generator.getAPIInterfaces(config)

	s.Equal(1, len(interfaces))

	serviceInterface := interfaces[0]
	s.Equal(toPascalCase(config.Name)+"Service", serviceInterface.Name)
	s.Contains(serviceInterface.Comment, "defines the interface")

	// Should have methods from both HTTP and gRPC
	expectedMethods := len(config.Methods) + len(config.GRPCMethods)
	s.Equal(expectedMethods, len(serviceInterface.Methods))
}

// TestConvertFieldsToFieldInfo tests field conversion
func (s *APIGeneratorComprehensiveTestSuite) TestConvertFieldsToFieldInfo() {
	fields := []interfaces.Field{
		{
			Name:        "user_id",
			Type:        "string",
			Description: "User identifier",
			Tags:        map[string]string{"json": "user_id"},
		},
		{
			Name: "email",
			Type: "string",
		},
	}

	result := s.generator.convertFieldsToFieldInfo(fields)
	s.Equal(2, len(result))

	s.Equal("UserId", result[0].Name)
	s.Equal("string", result[0].Type)
	s.Equal("User identifier", result[0].Comment)
	s.Equal("user_id", result[0].Tags["json"])

	s.Equal("Email", result[1].Name)
	s.Equal("string", result[1].Type)
}

// TestIsValidHTTPMethod tests HTTP method validation
func (s *APIGeneratorComprehensiveTestSuite) TestIsValidHTTPMethod() {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	invalidMethods := []string{"INVALID", "CUSTOM", ""}

	for _, method := range validMethods {
		s.True(isValidHTTPMethod(method), "Method %s should be valid", method)
		s.True(isValidHTTPMethod(strings.ToLower(method)), "Method %s (lowercase) should be valid", method)
	}

	for _, method := range invalidMethods {
		s.False(isValidHTTPMethod(method), "Method %s should be invalid", method)
	}
}

// Test error scenarios and edge cases
func (s *APIGeneratorComprehensiveTestSuite) TestGenerateAPI_CompleteFlow() {
	config := testutil.TestAPIConfig()

	s.setupAPIGenerationExpectations(config, 8) // All files

	err := s.generator.GenerateAPI(config)
	s.NoError(err)
}

// Test concurrent API generation
func (s *APIGeneratorComprehensiveTestSuite) TestConcurrentAPIGeneration() {
	configs := []interfaces.APIConfig{
		testutil.TestAPIConfig(),
		testutil.TestAPIConfig(),
		testutil.TestAPIConfig(),
	}

	// Make each config unique with valid lowercase names
	for i := range configs {
		configs[i].Name = configs[i].Name + "-" + string(rune('a'+i))
	}

	// Set up expectations for all configurations with flexible mock matching
	s.setupAPIGenerationExpectations(configs[0], 0)

	// Run concurrent generation
	done := make(chan error, len(configs))
	for _, config := range configs {
		go func(cfg interfaces.APIConfig) {
			done <- s.generator.GenerateAPI(cfg)
		}(config)
	}

	// Wait for all to complete
	for i := 0; i < len(configs); i++ {
		err := <-done
		s.NoError(err)
	}
}

// Helper methods for setting up expectations

func (s *APIGeneratorComprehensiveTestSuite) setupAPIGenerationExpectations(_ interfaces.APIConfig, fileCount int) {
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("api.go").Maybe()
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated API"), nil).Maybe()

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Maybe()
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated API"), nil).Maybe()
	s.fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil).Maybe()
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Maybe()
	// No logger expectations needed with NoOpLogger
}

func (s *APIGeneratorComprehensiveTestSuite) setupHTTPHandlerExpectations(fileCount int) {
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("handler.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated handler"), nil)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Times(fileCount)
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated handler"), nil).Times(fileCount)
	s.fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil).Maybe()
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Times(fileCount)
}

func (s *APIGeneratorComprehensiveTestSuite) setupGRPCHandlerExpectations(fileCount int) {
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("grpc_handler.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated gRPC handler"), nil)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Times(fileCount)
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated gRPC handler"), nil).Times(fileCount)
	s.fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil).Maybe()
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Times(fileCount)
}

func (s *APIGeneratorComprehensiveTestSuite) setupProtoExpectations(fileCount int) {
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("service.proto")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated proto"), nil)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Times(fileCount)
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated proto"), nil).Times(fileCount)
	s.fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil).Maybe()
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Times(fileCount)
}

func (s *APIGeneratorComprehensiveTestSuite) setupServiceLayerExpectations(fileCount int) {
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("service.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated service"), nil)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Times(fileCount)
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated service"), nil).Times(fileCount)
	s.fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil).Maybe()
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Times(fileCount)
}

func (s *APIGeneratorComprehensiveTestSuite) setupInterfaceExpectations(fileCount int) {
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("interface.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated interface"), nil)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Times(fileCount)
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated interface"), nil).Times(fileCount)
	s.fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil).Maybe()
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Times(fileCount)
}

func (s *APIGeneratorComprehensiveTestSuite) setupModelExpectations(fileCount int) {
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("model.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated model"), nil)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Times(fileCount)
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated model"), nil).Times(fileCount)
	s.fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil).Maybe()
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Times(fileCount)
}

// Helper function to extract import paths
func getImportPaths(imports []interfaces.ImportInfo) []string {
	paths := make([]string, len(imports))
	for i, imp := range imports {
		paths[i] = imp.Path
	}
	return paths
}

// Benchmark tests
func BenchmarkAPIGeneratorComprehensive_GenerateAPI(b *testing.B) {
	templateEngine := testutil.NewMockTemplateEngine()
	fileSystem := testutil.NewMockFileSystem()
	logger := testutil.NewNoOpLogger()

	generator := NewAPIGenerator(templateEngine, fileSystem, logger)
	config := testutil.TestAPIConfig()

	// Setup mock expectations
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("api.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("test content"), nil)

	templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil)
	templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("test content"), nil)
	fileSystem.On("MkdirAll", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(nil).Maybe()
	fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil)
	// No logger expectations needed with NoOpLogger

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.GenerateAPI(config)
	}
}

func BenchmarkAPIGeneratorComprehensive_PrepareTemplateData(b *testing.B) {
	templateEngine := testutil.NewMockTemplateEngine()
	fileSystem := testutil.NewMockFileSystem()
	logger := testutil.NewNoOpLogger()

	generator := NewAPIGenerator(templateEngine, fileSystem, logger)
	config := testutil.TestAPIConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = generator.prepareAPITemplateData(config)
	}
}

// Test large API configurations
func (s *APIGeneratorComprehensiveTestSuite) TestGenerateAPI_LargeConfiguration() {
	config := testutil.TestAPIConfig()

	// Add many HTTP methods
	for i := 0; i < 50; i++ {
		config.Methods = append(config.Methods, interfaces.HTTPMethod{
			Name:   "Method" + string(rune('A'+i%26)) + string(rune('0'+i%10)),
			Method: []string{"GET", "POST", "PUT", "DELETE"}[i%4],
			Path:   "/api/method" + string(rune('0'+i%10)),
		})
	}

	// Add many gRPC methods
	for i := 0; i < 30; i++ {
		config.GRPCMethods = append(config.GRPCMethods, interfaces.GRPCMethod{
			Name: "GRPCMethod" + string(rune('A'+i%26)),
		})
	}

	// Add many models
	for i := 0; i < 20; i++ {
		config.Models = append(config.Models, interfaces.Model{
			Name: "Model" + string(rune('A'+i%26)),
			Fields: []interfaces.Field{
				{Name: "id", Type: "string"},
				{Name: "name", Type: "string"},
			},
		})
	}

	s.setupAPIGenerationExpectations(config, 150) // Many files for large config

	start := time.Now()
	err := s.generator.GenerateAPI(config)
	duration := time.Since(start)

	s.NoError(err)
	s.True(duration < 10*time.Second, "Large API generation should complete within reasonable time")
}
