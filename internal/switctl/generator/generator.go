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
	"fmt"
	"strings"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// Generator implements the main Generator interface that coordinates all generation tasks.
type Generator struct {
	serviceGenerator    *ServiceGenerator
	apiGenerator        *APIGenerator
	modelGenerator      *ModelGenerator
	middlewareGenerator *MiddlewareGenerator
	templateEngine      interfaces.TemplateEngine
	fileSystem          interfaces.FileSystem
	logger              interfaces.Logger
}

// NewGenerator creates a new main generator that coordinates all generation tasks.
func NewGenerator(templateEngine interfaces.TemplateEngine, fileSystem interfaces.FileSystem, logger interfaces.Logger) *Generator {
	return &Generator{
		serviceGenerator:    NewServiceGenerator(templateEngine, fileSystem, logger),
		apiGenerator:        NewAPIGenerator(templateEngine, fileSystem, logger),
		modelGenerator:      NewModelGenerator(templateEngine, fileSystem, logger),
		middlewareGenerator: NewMiddlewareGenerator(templateEngine, fileSystem, logger),
		templateEngine:      templateEngine,
		fileSystem:          fileSystem,
		logger:              logger,
	}
}

// GenerateService generates a new service based on the configuration.
func (g *Generator) GenerateService(config interfaces.ServiceConfig) error {
	return g.serviceGenerator.GenerateService(config)
}

// GenerateAPI generates API endpoints and related code.
func (g *Generator) GenerateAPI(config interfaces.APIConfig) error {
	return g.apiGenerator.GenerateAPI(config)
}

// GenerateModel generates data models and CRUD operations.
func (g *Generator) GenerateModel(config interfaces.ModelConfig) error {
	return g.modelGenerator.GenerateModel(config)
}

// GenerateMiddleware generates middleware templates.
func (g *Generator) GenerateMiddleware(config interfaces.MiddlewareConfig) error {
	return g.middlewareGenerator.GenerateMiddleware(config)
}

// ModelGenerator generates data models and CRUD operations.
type ModelGenerator struct {
	templateEngine interfaces.TemplateEngine
	fileSystem     interfaces.FileSystem
	logger         interfaces.Logger
}

// NewModelGenerator creates a new model generator.
func NewModelGenerator(templateEngine interfaces.TemplateEngine, fileSystem interfaces.FileSystem, logger interfaces.Logger) *ModelGenerator {
	return &ModelGenerator{
		templateEngine: templateEngine,
		fileSystem:     fileSystem,
		logger:         logger,
	}
}

// GenerateModel generates data models and CRUD operations.
func (mg *ModelGenerator) GenerateModel(config interfaces.ModelConfig) error {
	mg.logger.Info("Starting model generation", "model", config.Name)

	// Validate configuration
	if err := mg.validateModelConfig(config); err != nil {
		return fmt.Errorf("invalid model configuration: %w", err)
	}

	// Prepare template data
	templateData, err := mg.prepareModelTemplateData(config)
	if err != nil {
		return fmt.Errorf("failed to prepare model template data: %w", err)
	}

	// Generate model struct
	if err := mg.generateModelStruct(config, templateData); err != nil {
		return fmt.Errorf("failed to generate model struct: %w", err)
	}

	// Generate CRUD operations if enabled
	if config.CRUD {
		if err := mg.generateCRUDOperations(config, templateData); err != nil {
			return fmt.Errorf("failed to generate CRUD operations: %w", err)
		}
	}

	// Generate repository if database is configured
	if config.Database != "" {
		if err := mg.generateRepository(config, templateData); err != nil {
			return fmt.Errorf("failed to generate repository: %w", err)
		}
	}

	mg.logger.Info("Model generation completed successfully", "model", config.Name)
	return nil
}

// validateModelConfig validates the model configuration.
func (mg *ModelGenerator) validateModelConfig(config interfaces.ModelConfig) error {
	if config.Name == "" {
		return fmt.Errorf("model name is required")
	}

	if len(config.Fields) == 0 {
		return fmt.Errorf("at least one field is required")
	}

	// Validate fields
	for i, field := range config.Fields {
		if field.Name == "" {
			return fmt.Errorf("field name is required at index %d", i)
		}
		if field.Type == "" {
			return fmt.Errorf("field type is required for field %s", field.Name)
		}
	}

	// Set defaults
	if config.Package == "" {
		config.Package = "model"
	}
	if config.Table == "" {
		config.Table = strings.ToLower(pluralize(config.Name))
	}

	return nil
}

// prepareModelTemplateData prepares template data for model generation.
func (mg *ModelGenerator) prepareModelTemplateData(config interfaces.ModelConfig) (*interfaces.TemplateData, error) {
	modelName := toPascalCase(config.Name)

	templateData := &interfaces.TemplateData{
		Package: interfaces.PackageInfo{
			Name: config.Package,
			Path: config.Package,
		},
		Structs: []interfaces.StructInfo{
			{
				Name:    modelName,
				Comment: modelName + " represents a " + config.Name + " entity.",
				Fields:  mg.convertConfigFieldsToFieldInfo(config.Fields),
			},
		},
		Interfaces: mg.getModelInterfaces(config),
		Functions:  mg.getModelFunctions(config),
		Metadata:   config.Metadata,
		Timestamp:  time.Now(),
	}

	// Add model-specific metadata
	if templateData.Metadata == nil {
		templateData.Metadata = make(map[string]string)
	}
	templateData.Metadata["model_name"] = modelName
	templateData.Metadata["table_name"] = config.Table
	templateData.Metadata["package_name"] = config.Package

	return templateData, nil
}

// generateModelStruct generates the model struct file.
func (mg *ModelGenerator) generateModelStruct(config interfaces.ModelConfig, data *interfaces.TemplateData) error {
	modelFile := ServiceFile{
		Path:     fmt.Sprintf("%s/%s.go", config.Package, strings.ToLower(config.Name)),
		Template: "model/model.go",
		Data:     data,
		Mode:     0644,
	}

	return mg.generateModelFile(modelFile)
}

// generateCRUDOperations generates CRUD operation files.
func (mg *ModelGenerator) generateCRUDOperations(config interfaces.ModelConfig, data *interfaces.TemplateData) error {
	crudFile := ServiceFile{
		Path:     fmt.Sprintf("%s/%s_crud.go", config.Package, strings.ToLower(config.Name)),
		Template: "model/crud.go",
		Data:     data,
		Mode:     0644,
	}

	return mg.generateModelFile(crudFile)
}

// generateRepository generates repository files.
func (mg *ModelGenerator) generateRepository(config interfaces.ModelConfig, data *interfaces.TemplateData) error {
	repoFile := ServiceFile{
		Path:     fmt.Sprintf("repository/%s_repository.go", strings.ToLower(config.Name)),
		Template: "model/repository.go",
		Data:     data,
		Mode:     0644,
	}

	return mg.generateModelFile(repoFile)
}

// generateModelFile generates a single model file.
func (mg *ModelGenerator) generateModelFile(file ServiceFile) error {
	// Load template
	tmpl, err := mg.templateEngine.LoadTemplate(file.Template)
	if err != nil {
		return fmt.Errorf("failed to load template %s: %w", file.Template, err)
	}

	// Render template
	content, err := mg.templateEngine.RenderTemplate(tmpl, file.Data)
	if err != nil {
		return fmt.Errorf("failed to render template %s: %w", file.Template, err)
	}

	// Write file
	if err := mg.fileSystem.WriteFile(file.Path, content, int(file.Mode)); err != nil {
		return fmt.Errorf("failed to write file %s: %w", file.Path, err)
	}

	mg.logger.Debug("Generated model file", "path", file.Path)
	return nil
}

// convertConfigFieldsToFieldInfo converts model config fields to FieldInfo.
func (mg *ModelGenerator) convertConfigFieldsToFieldInfo(fields []interfaces.Field) []interfaces.FieldInfo {
	var fieldInfos []interfaces.FieldInfo

	for _, field := range fields {
		fieldInfo := interfaces.FieldInfo{
			Name:    toPascalCase(field.Name),
			Type:    field.Type,
			Tags:    field.Tags,
			Comment: field.Description,
		}

		// Add default tags if not specified
		if fieldInfo.Tags == nil {
			fieldInfo.Tags = make(map[string]string)
		}
		if _, exists := fieldInfo.Tags["json"]; !exists {
			fieldInfo.Tags["json"] = strings.ToLower(field.Name)
		}
		if _, exists := fieldInfo.Tags["db"]; !exists {
			fieldInfo.Tags["db"] = strings.ToLower(field.Name)
		}

		fieldInfos = append(fieldInfos, fieldInfo)
	}

	return fieldInfos
}

// getModelInterfaces returns interfaces for model generation.
func (mg *ModelGenerator) getModelInterfaces(config interfaces.ModelConfig) []interfaces.InterfaceInfo {
	if !config.CRUD {
		return nil
	}

	modelName := toPascalCase(config.Name)

	return []interfaces.InterfaceInfo{
		{
			Name:    modelName + "Repository",
			Comment: modelName + "Repository defines the interface for " + config.Name + " data operations.",
			Methods: []interfaces.MethodInfo{
				{
					Name:    "Create",
					Comment: "Create creates a new " + config.Name + ".",
					Parameters: []interfaces.ParameterInfo{
						{Name: "ctx", Type: "context.Context"},
						{Name: "model", Type: "*" + modelName},
					},
					Returns: []interfaces.ReturnInfo{
						{Type: "error"},
					},
				},
				{
					Name:    "GetByID",
					Comment: "GetByID retrieves a " + config.Name + " by ID.",
					Parameters: []interfaces.ParameterInfo{
						{Name: "ctx", Type: "context.Context"},
						{Name: "id", Type: "uint"},
					},
					Returns: []interfaces.ReturnInfo{
						{Type: "*" + modelName},
						{Type: "error"},
					},
				},
				{
					Name:    "Update",
					Comment: "Update updates an existing " + config.Name + ".",
					Parameters: []interfaces.ParameterInfo{
						{Name: "ctx", Type: "context.Context"},
						{Name: "model", Type: "*" + modelName},
					},
					Returns: []interfaces.ReturnInfo{
						{Type: "error"},
					},
				},
				{
					Name:    "Delete",
					Comment: "Delete deletes a " + config.Name + " by ID.",
					Parameters: []interfaces.ParameterInfo{
						{Name: "ctx", Type: "context.Context"},
						{Name: "id", Type: "uint"},
					},
					Returns: []interfaces.ReturnInfo{
						{Type: "error"},
					},
				},
				{
					Name:    "List",
					Comment: "List retrieves a list of " + pluralize(config.Name) + ".",
					Parameters: []interfaces.ParameterInfo{
						{Name: "ctx", Type: "context.Context"},
					},
					Returns: []interfaces.ReturnInfo{
						{Type: "[]" + modelName},
						{Type: "error"},
					},
				},
			},
		},
	}
}

// getModelFunctions returns functions for model generation.
func (mg *ModelGenerator) getModelFunctions(config interfaces.ModelConfig) []interfaces.FunctionInfo {
	var functions []interfaces.FunctionInfo

	modelName := toPascalCase(config.Name)

	// Constructor function
	functions = append(functions, interfaces.FunctionInfo{
		Name:    "New" + modelName,
		Comment: "New" + modelName + " creates a new " + modelName + " instance.",
		Returns: []interfaces.ReturnInfo{
			{Type: "*" + modelName},
		},
	})

	// Validation function
	functions = append(functions, interfaces.FunctionInfo{
		Name:     "Validate",
		Comment:  "Validate validates the " + config.Name + " data.",
		Receiver: "*" + modelName,
		Returns: []interfaces.ReturnInfo{
			{Type: "error"},
		},
	})

	return functions
}

// MiddlewareGenerator generates middleware templates.
type MiddlewareGenerator struct {
	templateEngine interfaces.TemplateEngine
	fileSystem     interfaces.FileSystem
	logger         interfaces.Logger
}

// NewMiddlewareGenerator creates a new middleware generator.
func NewMiddlewareGenerator(templateEngine interfaces.TemplateEngine, fileSystem interfaces.FileSystem, logger interfaces.Logger) *MiddlewareGenerator {
	return &MiddlewareGenerator{
		templateEngine: templateEngine,
		fileSystem:     fileSystem,
		logger:         logger,
	}
}

// GenerateMiddleware generates middleware templates.
func (mw *MiddlewareGenerator) GenerateMiddleware(config interfaces.MiddlewareConfig) error {
	mw.logger.Info("Starting middleware generation", "middleware", config.Name, "type", config.Type)

	// Validate configuration
	if err := mw.validateMiddlewareConfig(config); err != nil {
		return fmt.Errorf("invalid middleware configuration: %w", err)
	}

	// Prepare template data
	templateData, err := mw.prepareMiddlewareTemplateData(config)
	if err != nil {
		return fmt.Errorf("failed to prepare middleware template data: %w", err)
	}

	// Generate middleware file
	if err := mw.generateMiddlewareFile(config, templateData); err != nil {
		return fmt.Errorf("failed to generate middleware file: %w", err)
	}

	// Generate middleware tests
	if err := mw.generateMiddlewareTests(config, templateData); err != nil {
		return fmt.Errorf("failed to generate middleware tests: %w", err)
	}

	mw.logger.Info("Middleware generation completed successfully", "middleware", config.Name)
	return nil
}

// validateMiddlewareConfig validates the middleware configuration.
func (mw *MiddlewareGenerator) validateMiddlewareConfig(config interfaces.MiddlewareConfig) error {
	if config.Name == "" {
		return fmt.Errorf("middleware name is required")
	}

	if config.Type == "" {
		return fmt.Errorf("middleware type is required")
	}

	validTypes := []string{"http", "grpc", "both"}
	isValidType := false
	for _, validType := range validTypes {
		if config.Type == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return fmt.Errorf("invalid middleware type %s, must be one of: %s", config.Type, strings.Join(validTypes, ", "))
	}

	// Set defaults
	if config.Package == "" {
		config.Package = "middleware"
	}

	return nil
}

// prepareMiddlewareTemplateData prepares template data for middleware generation.
func (mw *MiddlewareGenerator) prepareMiddlewareTemplateData(config interfaces.MiddlewareConfig) (*interfaces.TemplateData, error) {
	middlewareName := toPascalCase(config.Name)

	templateData := &interfaces.TemplateData{
		Package: interfaces.PackageInfo{
			Name: config.Package,
			Path: config.Package,
		},
		Functions: mw.getMiddlewareFunctions(config),
		Metadata:  config.Metadata,
		Timestamp: time.Now(),
	}

	// Add middleware-specific metadata
	if templateData.Metadata == nil {
		templateData.Metadata = make(map[string]string)
	}
	templateData.Metadata["middleware_name"] = middlewareName
	templateData.Metadata["middleware_type"] = config.Type
	templateData.Metadata["package_name"] = config.Package

	return templateData, nil
}

// generateMiddlewareFile generates the middleware implementation file.
func (mw *MiddlewareGenerator) generateMiddlewareFile(config interfaces.MiddlewareConfig, data *interfaces.TemplateData) error {
	template := fmt.Sprintf("middleware/%s.go", config.Type)

	middlewareFile := ServiceFile{
		Path:     fmt.Sprintf("%s/%s.go", config.Package, strings.ToLower(config.Name)),
		Template: template,
		Data:     data,
		Mode:     0644,
	}

	return mw.generateMiddlewareFileInternal(middlewareFile)
}

// generateMiddlewareTests generates middleware test files.
func (mw *MiddlewareGenerator) generateMiddlewareTests(config interfaces.MiddlewareConfig, data *interfaces.TemplateData) error {
	template := fmt.Sprintf("middleware/%s_test.go", config.Type)

	testFile := ServiceFile{
		Path:     fmt.Sprintf("%s/%s_test.go", config.Package, strings.ToLower(config.Name)),
		Template: template,
		Data:     data,
		Mode:     0644,
	}

	return mw.generateMiddlewareFileInternal(testFile)
}

// generateMiddlewareFileInternal generates a single middleware file.
func (mw *MiddlewareGenerator) generateMiddlewareFileInternal(file ServiceFile) error {
	// Load template
	tmpl, err := mw.templateEngine.LoadTemplate(file.Template)
	if err != nil {
		return fmt.Errorf("failed to load template %s: %w", file.Template, err)
	}

	// Render template
	content, err := mw.templateEngine.RenderTemplate(tmpl, file.Data)
	if err != nil {
		return fmt.Errorf("failed to render template %s: %w", file.Template, err)
	}

	// Write file
	if err := mw.fileSystem.WriteFile(file.Path, content, int(file.Mode)); err != nil {
		return fmt.Errorf("failed to write file %s: %w", file.Path, err)
	}

	mw.logger.Debug("Generated middleware file", "path", file.Path)
	return nil
}

// getMiddlewareFunctions returns functions for middleware generation.
func (mw *MiddlewareGenerator) getMiddlewareFunctions(config interfaces.MiddlewareConfig) []interfaces.FunctionInfo {
	var functions []interfaces.FunctionInfo

	middlewareName := toPascalCase(config.Name)

	switch config.Type {
	case "http":
		functions = append(functions, interfaces.FunctionInfo{
			Name:    middlewareName,
			Comment: middlewareName + " creates a new HTTP middleware for " + config.Name + ".",
			Returns: []interfaces.ReturnInfo{
				{Type: "gin.HandlerFunc"},
			},
		})
	case "grpc":
		functions = append(functions, interfaces.FunctionInfo{
			Name:    middlewareName + "Interceptor",
			Comment: middlewareName + "Interceptor creates a new gRPC interceptor for " + config.Name + ".",
			Returns: []interfaces.ReturnInfo{
				{Type: "grpc.UnaryServerInterceptor"},
			},
		})
	case "both":
		functions = append(functions, []interfaces.FunctionInfo{
			{
				Name:    middlewareName,
				Comment: middlewareName + " creates a new HTTP middleware for " + config.Name + ".",
				Returns: []interfaces.ReturnInfo{
					{Type: "gin.HandlerFunc"},
				},
			},
			{
				Name:    middlewareName + "Interceptor",
				Comment: middlewareName + "Interceptor creates a new gRPC interceptor for " + config.Name + ".",
				Returns: []interfaces.ReturnInfo{
					{Type: "grpc.UnaryServerInterceptor"},
				},
			},
		}...)
	}

	return functions
}

// Helper function for pluralization (simple implementation)
func pluralize(s string) string {
	if s == "" {
		return s
	}

	// Simple pluralization rules
	if strings.HasSuffix(s, "y") && len(s) > 1 {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "z") || strings.HasSuffix(s, "ch") ||
		strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	return s + "s"
}
