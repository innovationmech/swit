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

// ModelGeneratorTestSuite tests the ModelGenerator implementation
type ModelGeneratorTestSuite struct {
	suite.Suite
	generator      *ModelGenerator
	templateEngine *testutil.MockTemplateEngine
	fileSystem     *testutil.MockFileSystem
	logger         interfaces.Logger
}

func TestModelGeneratorTestSuite(t *testing.T) {
	suite.Run(t, new(ModelGeneratorTestSuite))
}

func (s *ModelGeneratorTestSuite) SetupTest() {
	s.templateEngine = testutil.NewMockTemplateEngine()
	s.fileSystem = testutil.NewMockFileSystem()
	s.logger = testutil.NewNoOpLogger()

	s.generator = NewModelGenerator(s.templateEngine, s.fileSystem, s.logger)
}

func (s *ModelGeneratorTestSuite) TearDownTest() {
	// Reset all mocks
	s.templateEngine.AssertExpectations(s.T())
	s.fileSystem.AssertExpectations(s.T())
	// No need to assert expectations on NoOpLogger
}

func (s *ModelGeneratorTestSuite) TestNewModelGenerator() {
	// Test basic constructor
	generator := NewModelGenerator(s.templateEngine, s.fileSystem, s.logger)

	s.NotNil(generator)
	s.Equal(s.templateEngine, generator.templateEngine)
	s.Equal(s.fileSystem, generator.fileSystem)
	s.Equal(s.logger, generator.logger)
}

func (s *ModelGeneratorTestSuite) TestNewModelGenerator_NilDependencies() {
	// Test with nil dependencies - should not panic
	s.NotPanics(func() {
		NewModelGenerator(nil, nil, nil)
	})
}

// TestGenerateModel tests basic model generation
func (s *ModelGeneratorTestSuite) TestGenerateModel() {
	config := testutil.TestModelConfig()

	s.setupModelGenerationExpectations(config)

	err := s.generator.GenerateModel(config)
	s.NoError(err)
}

func (s *ModelGeneratorTestSuite) TestGenerateModel_WithCRUD() {
	config := testutil.TestModelConfig()
	config.CRUD = true

	s.setupModelGenerationExpectations(config)

	err := s.generator.GenerateModel(config)
	s.NoError(err)
}

func (s *ModelGeneratorTestSuite) TestGenerateModel_WithoutCRUD() {
	config := testutil.TestModelConfig()
	config.CRUD = false

	s.setupModelGenerationExpectations(config)

	err := s.generator.GenerateModel(config)
	s.NoError(err)
}

func (s *ModelGeneratorTestSuite) TestGenerateModel_WithDatabase() {
	config := testutil.TestModelConfig()
	config.Database = "mysql"

	s.setupModelGenerationExpectations(config)

	err := s.generator.GenerateModel(config)
	s.NoError(err)
}

func (s *ModelGeneratorTestSuite) TestGenerateModel_WithoutDatabase() {
	config := testutil.TestModelConfig()
	config.Database = ""

	s.setupModelGenerationExpectations(config)

	err := s.generator.GenerateModel(config)
	s.NoError(err)
}

// TestValidateModelConfig tests model configuration validation
func (s *ModelGeneratorTestSuite) TestValidateModelConfig_ValidConfig() {
	config := testutil.TestModelConfig()

	err := s.generator.validateModelConfig(&config)
	s.NoError(err)
}

func (s *ModelGeneratorTestSuite) TestValidateModelConfig_EmptyName() {
	config := testutil.TestModelConfig()
	config.Name = ""

	err := s.generator.validateModelConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "model name is required")
}

func (s *ModelGeneratorTestSuite) TestValidateModelConfig_NoFields() {
	config := testutil.TestModelConfig()
	config.Fields = []interfaces.Field{}

	err := s.generator.validateModelConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "at least one field is required")
}

func (s *ModelGeneratorTestSuite) TestValidateModelConfig_EmptyFieldName() {
	config := testutil.TestModelConfig()
	config.Fields = []interfaces.Field{
		{Name: "", Type: "string"},
	}

	err := s.generator.validateModelConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "field name is required")
}

func (s *ModelGeneratorTestSuite) TestValidateModelConfig_EmptyFieldType() {
	config := testutil.TestModelConfig()
	config.Fields = []interfaces.Field{
		{Name: "test", Type: ""},
	}

	err := s.generator.validateModelConfig(&config)
	s.Error(err)
	s.Contains(err.Error(), "field type is required")
}

func (s *ModelGeneratorTestSuite) TestValidateModelConfig_DefaultValues() {
	config := interfaces.ModelConfig{
		Name: "TestModel",
		Fields: []interfaces.Field{
			{Name: "id", Type: "uint"},
		},
	}

	err := s.generator.validateModelConfig(&config)
	s.NoError(err)

	// Check defaults are set
	s.Equal("model", config.Package)    // Default package
	s.Equal("testmodels", config.Table) // Default table name (pluralized and lowercase)
}

// TestPrepareModelTemplateData tests template data preparation
func (s *ModelGeneratorTestSuite) TestPrepareModelTemplateData() {
	config := testutil.TestModelConfig()

	data, err := s.generator.prepareModelTemplateData(config)
	s.NoError(err)
	s.NotNil(data)

	// Check basic template data structure
	s.Equal(config.Package, data.Package.Name)
	s.Equal(config.Package, data.Package.Path)
	s.Equal(1, len(data.Structs))
	s.Equal(toPascalCase(config.Name), data.Structs[0].Name)

	// Check metadata
	s.NotNil(data.Metadata)
	s.Equal(toPascalCase(config.Name), data.Metadata["model_name"])
	s.Equal(config.Table, data.Metadata["table_name"])
	s.Equal(config.Package, data.Metadata["package_name"])

	// Check timestamp
	s.WithinDuration(time.Now(), data.Timestamp, time.Second)
}

func (s *ModelGeneratorTestSuite) TestPrepareModelTemplateData_ComplexFields() {
	config := testutil.TestModelConfig()
	config.Fields = []interfaces.Field{
		{
			Name:        "id",
			Type:        "uint",
			Required:    true,
			Unique:      true,
			Tags:        map[string]string{"json": "id", "gorm": "primaryKey"},
			Description: "Primary key",
		},
		{
			Name:        "email",
			Type:        "string",
			Required:    true,
			Unique:      true,
			Tags:        map[string]string{"json": "email", "validate": "email"},
			Description: "User email address",
		},
		{
			Name: "age",
			Type: "int",
			Tags: map[string]string{"json": "age", "validate": "min=0,max=150"},
		},
	}

	data, err := s.generator.prepareModelTemplateData(config)
	s.NoError(err)
	s.NotNil(data)

	// Check fields conversion
	s.Equal(len(config.Fields), len(data.Structs[0].Fields))

	// Check first field (ID)
	idField := data.Structs[0].Fields[0]
	s.Equal("Id", idField.Name) // Pascal case
	s.Equal("uint", idField.Type)
	s.Equal("Primary key", idField.Comment)
	s.Equal("id", idField.Tags["json"])
	s.Equal("primaryKey", idField.Tags["gorm"])

	// Check email field
	emailField := data.Structs[0].Fields[1]
	s.Equal("Email", emailField.Name)
	s.Equal("string", emailField.Type)
	s.Equal("email", emailField.Tags["json"])
}

// TestGenerateModelStruct tests model struct generation
func (s *ModelGeneratorTestSuite) TestGenerateModelStruct() {
	config := testutil.TestModelConfig()
	data, _ := s.generator.prepareModelTemplateData(config)

	// Setup template expectations
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("model/model.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated model struct"), nil)

	s.templateEngine.On("LoadTemplate", "model/model.go").Return(mockTemplate, nil)
	s.templateEngine.On("RenderTemplate", mockTemplate, data).Return([]byte("generated model struct"), nil)
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil)
	// No logger expectations needed with NoOpLogger

	err := s.generator.generateModelStruct(config, data)
	s.NoError(err)
}

func (s *ModelGeneratorTestSuite) TestGenerateModelStruct_TemplateLoadError() {
	config := testutil.TestModelConfig()
	data, _ := s.generator.prepareModelTemplateData(config)

	s.templateEngine.On("LoadTemplate", "model/model.go").Return(nil, assert.AnError)

	err := s.generator.generateModelStruct(config, data)
	s.Error(err)
	s.Contains(err.Error(), "failed to load template")
}

func (s *ModelGeneratorTestSuite) TestGenerateModelStruct_TemplateRenderError() {
	config := testutil.TestModelConfig()
	data, _ := s.generator.prepareModelTemplateData(config)

	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("model/model.go")

	s.templateEngine.On("LoadTemplate", "model/model.go").Return(mockTemplate, nil)
	s.templateEngine.On("RenderTemplate", mockTemplate, data).Return(nil, assert.AnError)

	err := s.generator.generateModelStruct(config, data)
	s.Error(err)
	s.Contains(err.Error(), "failed to render template")
}

func (s *ModelGeneratorTestSuite) TestGenerateModelStruct_FileWriteError() {
	config := testutil.TestModelConfig()
	data, _ := s.generator.prepareModelTemplateData(config)

	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("model/model.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated content"), nil)

	s.templateEngine.On("LoadTemplate", "model/model.go").Return(mockTemplate, nil)
	s.templateEngine.On("RenderTemplate", mockTemplate, data).Return([]byte("generated content"), nil)
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(assert.AnError)

	err := s.generator.generateModelStruct(config, data)
	s.Error(err)
	s.Contains(err.Error(), "failed to write file")
}

// TestGenerateCRUDOperations tests CRUD operations generation
func (s *ModelGeneratorTestSuite) TestGenerateCRUDOperations() {
	config := testutil.TestModelConfig()
	config.CRUD = true
	data, _ := s.generator.prepareModelTemplateData(config)

	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("model/crud.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated CRUD operations"), nil)

	s.templateEngine.On("LoadTemplate", "model/crud.go").Return(mockTemplate, nil)
	s.templateEngine.On("RenderTemplate", mockTemplate, data).Return([]byte("generated CRUD operations"), nil)
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil)
	// No logger expectations needed with NoOpLogger

	err := s.generator.generateCRUDOperations(config, data)
	s.NoError(err)
}

// TestGenerateRepository tests repository generation
func (s *ModelGeneratorTestSuite) TestGenerateRepository() {
	config := testutil.TestModelConfig()
	config.Database = "mysql"
	data, _ := s.generator.prepareModelTemplateData(config)

	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("model/repository.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated repository"), nil)

	s.templateEngine.On("LoadTemplate", "model/repository.go").Return(mockTemplate, nil)
	s.templateEngine.On("RenderTemplate", mockTemplate, data).Return([]byte("generated repository"), nil)
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil)
	// No logger expectations needed with NoOpLogger

	err := s.generator.generateRepository(config, data)
	s.NoError(err)
}

// TestConvertConfigFieldsToFieldInfo tests field conversion
func (s *ModelGeneratorTestSuite) TestConvertConfigFieldsToFieldInfo() {
	fields := []interfaces.Field{
		{
			Name:        "user_id",
			Type:        "uint",
			Description: "User identifier",
			Tags:        map[string]string{"custom": "value"},
		},
		{
			Name:        "full_name",
			Type:        "string",
			Description: "User full name",
		},
	}

	result := s.generator.convertConfigFieldsToFieldInfo(fields)
	s.Equal(2, len(result))

	// Check first field
	s.Equal("UserId", result[0].Name) // Pascal case conversion
	s.Equal("uint", result[0].Type)
	s.Equal("User identifier", result[0].Comment)
	s.Equal("value", result[0].Tags["custom"])
	s.Equal("user_id", result[0].Tags["json"]) // Default tag added
	s.Equal("user_id", result[0].Tags["db"])   // Default tag added

	// Check second field
	s.Equal("FullName", result[1].Name)
	s.Equal("string", result[1].Type)
	s.Equal("User full name", result[1].Comment)
	s.Equal("full_name", result[1].Tags["json"])
	s.Equal("full_name", result[1].Tags["db"])
}

func (s *ModelGeneratorTestSuite) TestConvertConfigFieldsToFieldInfo_WithExistingTags() {
	fields := []interfaces.Field{
		{
			Name: "email",
			Type: "string",
			Tags: map[string]string{
				"json":     "email_address",
				"db":       "email_col",
				"validate": "email",
			},
		},
	}

	result := s.generator.convertConfigFieldsToFieldInfo(fields)
	s.Equal(1, len(result))

	// Should preserve existing tags
	s.Equal("email_address", result[0].Tags["json"])
	s.Equal("email_col", result[0].Tags["db"])
	s.Equal("email", result[0].Tags["validate"])
}

// TestGetModelInterfaces tests interface generation
func (s *ModelGeneratorTestSuite) TestGetModelInterfaces_WithCRUD() {
	config := testutil.TestModelConfig()
	config.CRUD = true

	interfaces := s.generator.getModelInterfaces(config)
	s.Equal(1, len(interfaces))

	repoInterface := interfaces[0]
	s.Equal(toPascalCase(config.Name)+"Repository", repoInterface.Name)
	s.Contains(repoInterface.Comment, config.Name)

	// Check CRUD methods are generated
	s.Equal(5, len(repoInterface.Methods)) // Create, GetByID, Update, Delete, List

	// Check Create method
	createMethod := repoInterface.Methods[0]
	s.Equal("Create", createMethod.Name)
	s.Equal(2, len(createMethod.Parameters)) // ctx, model
	s.Equal(1, len(createMethod.Returns))    // error

	// Check GetByID method
	getMethod := repoInterface.Methods[1]
	s.Equal("GetByID", getMethod.Name)
	s.Equal(2, len(getMethod.Parameters)) // ctx, id
	s.Equal(2, len(getMethod.Returns))    // model, error

	// Check List method
	listMethod := repoInterface.Methods[4]
	s.Equal("List", listMethod.Name)
	s.Equal(1, len(listMethod.Parameters)) // ctx
	s.Equal(2, len(listMethod.Returns))    // []Model, error
}

func (s *ModelGeneratorTestSuite) TestGetModelInterfaces_WithoutCRUD() {
	config := testutil.TestModelConfig()
	config.CRUD = false

	interfaces := s.generator.getModelInterfaces(config)
	s.Nil(interfaces)
}

// TestGetModelFunctions tests function generation
func (s *ModelGeneratorTestSuite) TestGetModelFunctions() {
	config := testutil.TestModelConfig()

	functions := s.generator.getModelFunctions(config)
	s.Equal(2, len(functions)) // Constructor and Validate

	// Check constructor function
	constructor := functions[0]
	s.Equal("New"+toPascalCase(config.Name), constructor.Name)
	s.Contains(constructor.Comment, "creates a new")
	s.Equal(1, len(constructor.Returns))

	// Check validate function
	validator := functions[1]
	s.Equal("Validate", validator.Name)
	s.Equal("*"+toPascalCase(config.Name), validator.Receiver)
	s.Contains(validator.Comment, "validates")
	s.Equal(1, len(validator.Returns))
	s.Equal("error", validator.Returns[0].Type)
}

// Test edge cases and error scenarios
func (s *ModelGeneratorTestSuite) TestGenerateModel_CompleteFlow() {
	config := testutil.TestModelConfig()
	config.CRUD = true
	config.Database = "postgres"

	// Setup expectations for complete flow
	s.setupCompleteModelGenerationExpectations(config)

	err := s.generator.GenerateModel(config)
	s.NoError(err)
}

func (s *ModelGeneratorTestSuite) TestGenerateModel_MinimalConfig() {
	config := interfaces.ModelConfig{
		Name: "Simple",
		Fields: []interfaces.Field{
			{Name: "id", Type: "uint"},
		},
	}

	s.setupModelGenerationExpectations(config)

	err := s.generator.GenerateModel(config)
	s.NoError(err)
}

func (s *ModelGeneratorTestSuite) TestGenerateModel_ComplexFields() {
	config := testutil.TestModelConfig()
	config.Fields = []interfaces.Field{
		{
			Name:        "id",
			Type:        "uuid.UUID",
			Required:    true,
			Unique:      true,
			Tags:        map[string]string{"json": "id", "gorm": "type:uuid;primaryKey"},
			Description: "Unique identifier",
		},
		{
			Name:    "metadata",
			Type:    "json.RawMessage",
			Tags:    map[string]string{"json": "metadata", "gorm": "type:jsonb"},
			Default: "{}",
		},
		{
			Name: "created_at",
			Type: "time.Time",
			Tags: map[string]string{"json": "created_at", "gorm": "autoCreateTime"},
		},
		{
			Name: "updated_at",
			Type: "time.Time",
			Tags: map[string]string{"json": "updated_at", "gorm": "autoUpdateTime"},
		},
	}

	s.setupModelGenerationExpectations(config)

	err := s.generator.GenerateModel(config)
	s.NoError(err)
}

// Test concurrent model generation
func (s *ModelGeneratorTestSuite) TestConcurrentModelGeneration() {
	configs := []interfaces.ModelConfig{
		testutil.TestModelConfig(),
		testutil.TestModelConfig(),
		testutil.TestModelConfig(),
	}

	// Make each config unique with valid lowercase names
	for i := range configs {
		configs[i].Name = configs[i].Name + "-" + string(rune('a'+i))
	}

	// Set up expectations for all configurations with flexible mock matching
	s.setupModelGenerationExpectations(configs[0], 0)

	// Run concurrent generation
	done := make(chan error, len(configs))
	for _, config := range configs {
		go func(cfg interfaces.ModelConfig) {
			done <- s.generator.GenerateModel(cfg)
		}(config)
	}

	// Wait for all to complete
	for i := 0; i < len(configs); i++ {
		err := <-done
		s.NoError(err)
	}
}

// Test helper function coverage
func (s *ModelGeneratorTestSuite) TestPluralizeFunction() {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "users"},
		{"category", "categories"},
		{"box", "boxes"},
		{"brush", "brushes"},
		{"quiz", "quizzes"},
		{"child", "childs"}, // Simple pluralization
		{"", ""},
	}

	for _, test := range tests {
		result := pluralize(test.input)
		s.Equal(test.expected, result, "Failed to pluralize %s", test.input)
	}
}

// Helper methods for setting up expectations

func (s *ModelGeneratorTestSuite) setupModelGenerationExpectations(_ interfaces.ModelConfig, times ...int) {
	// Setup for generateModelStruct
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("model.go").Maybe()
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated model"), nil).Maybe()

	if len(times) > 0 && times[0] > 0 {
		callCount := times[0]
		s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Times(callCount)
		s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated model"), nil).Times(callCount)
		s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Times(callCount)
	} else {
		s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Maybe()
		s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated model"), nil).Maybe()
		s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Maybe()
	}
	// No logger expectations needed with NoOpLogger
}

func (s *ModelGeneratorTestSuite) setupCompleteModelGenerationExpectations(_ interfaces.ModelConfig) {
	// Setup for all generation steps
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("model.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("generated content"), nil)

	// Multiple template loads for different files
	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil).Times(3) // model, crud, repository
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("generated content"), nil).Times(3)
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil).Times(3)
	// No logger expectations needed with NoOpLogger
	// No logger expectations needed with NoOpLogger
}

// Benchmark tests
func BenchmarkModelGenerator_GenerateModel(b *testing.B) {
	templateEngine := testutil.NewMockTemplateEngine()
	fileSystem := testutil.NewMockFileSystem()
	logger := testutil.NewNoOpLogger()

	generator := NewModelGenerator(templateEngine, fileSystem, logger)
	config := testutil.TestModelConfig()

	// Setup mock expectations
	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("model.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("test content"), nil)

	templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil)
	templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("test content"), nil)
	fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(nil)
	// No logger expectations needed with NoOpLogger

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.GenerateModel(config)
	}
}

func BenchmarkModelGenerator_PrepareTemplateData(b *testing.B) {
	templateEngine := testutil.NewMockTemplateEngine()
	fileSystem := testutil.NewMockFileSystem()
	logger := testutil.NewNoOpLogger()

	generator := NewModelGenerator(templateEngine, fileSystem, logger)
	config := testutil.TestModelConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = generator.prepareModelTemplateData(config)
	}
}

// Test error scenarios with realistic file system errors
func (s *ModelGeneratorTestSuite) TestGenerateModel_FileSystemErrors() {
	config := testutil.TestModelConfig()

	mockTemplate := &testutil.MockTemplate{}
	mockTemplate.On("Name").Return("model.go")
	mockTemplate.On("Render", mock.Anything).Return([]byte("content"), nil)

	s.templateEngine.On("LoadTemplate", mock.AnythingOfType("string")).Return(mockTemplate, nil)
	s.templateEngine.On("RenderTemplate", mock.Anything, mock.Anything).Return([]byte("content"), nil)
	s.fileSystem.On("WriteFile", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int")).Return(assert.AnError)
	// No logger expectations needed with NoOpLogger

	err := s.generator.GenerateModel(config)
	s.Error(err)
	s.Contains(err.Error(), "failed to generate model struct")
}

// Test memory usage with large models
func (s *ModelGeneratorTestSuite) TestGenerateModel_LargeModel() {
	config := testutil.TestModelConfig()

	// Add many fields to simulate a large model
	for i := 0; i < 100; i++ {
		config.Fields = append(config.Fields, interfaces.Field{
			Name:        "field" + string(rune('A'+i%26)) + string(rune('0'+i%10)),
			Type:        "string",
			Description: "Auto-generated field",
			Tags: map[string]string{
				"json": "field" + string(rune('a'+i%26)) + string(rune('0'+i%10)),
				"db":   "field" + string(rune('a'+i%26)) + string(rune('0'+i%10)),
			},
		})
	}

	s.setupModelGenerationExpectations(config)

	start := time.Now()
	err := s.generator.GenerateModel(config)
	duration := time.Since(start)

	s.NoError(err)
	s.True(duration < 5*time.Second, "Large model generation should complete within reasonable time")
}
