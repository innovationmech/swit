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
	"path/filepath"
	"strings"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// APIGenerator generates API endpoints and related code.
type APIGenerator struct {
	templateEngine interfaces.TemplateEngine
	fileSystem     interfaces.FileSystem
	logger         interfaces.Logger
}

// NewAPIGenerator creates a new API generator.
func NewAPIGenerator(templateEngine interfaces.TemplateEngine, fileSystem interfaces.FileSystem, logger interfaces.Logger) *APIGenerator {
	return &APIGenerator{
		templateEngine: templateEngine,
		fileSystem:     fileSystem,
		logger:         logger,
	}
}

// GenerateAPI generates API endpoints and related code.
func (ag *APIGenerator) GenerateAPI(config interfaces.APIConfig) error {
	ag.logger.Info("Starting API generation", "api", config.Name, "service", config.Service)

	// Validate configuration
	if err := ag.validateAPIConfig(config); err != nil {
		return fmt.Errorf("invalid API configuration: %w", err)
	}

	// Prepare template data
	templateData, err := ag.prepareAPITemplateData(config)
	if err != nil {
		return fmt.Errorf("failed to prepare API template data: %w", err)
	}

	// Generate HTTP handlers
	if len(config.Methods) > 0 {
		if err := ag.generateHTTPHandlers(config, templateData); err != nil {
			return fmt.Errorf("failed to generate HTTP handlers: %w", err)
		}
	}

	// Generate gRPC handlers
	if len(config.GRPCMethods) > 0 {
		if err := ag.generateGRPCHandlers(config, templateData); err != nil {
			return fmt.Errorf("failed to generate gRPC handlers: %w", err)
		}

		// Generate Protocol Buffer definitions
		if err := ag.generateProtoDefinitions(config, templateData); err != nil {
			return fmt.Errorf("failed to generate proto definitions: %w", err)
		}
	}

	// Generate service layer
	if err := ag.generateServiceLayer(config, templateData); err != nil {
		return fmt.Errorf("failed to generate service layer: %w", err)
	}

	// Generate interfaces
	if err := ag.generateInterfaces(config, templateData); err != nil {
		return fmt.Errorf("failed to generate interfaces: %w", err)
	}

	// Generate models if specified
	if len(config.Models) > 0 {
		if err := ag.generateModels(config, templateData); err != nil {
			return fmt.Errorf("failed to generate models: %w", err)
		}
	}

	ag.logger.Info("API generation completed successfully", "api", config.Name)
	return nil
}

// validateAPIConfig validates the API configuration.
func (ag *APIGenerator) validateAPIConfig(config interfaces.APIConfig) error {
	if config.Name == "" {
		return fmt.Errorf("API name is required")
	}

	if config.Service == "" {
		return fmt.Errorf("service name is required")
	}

	if config.Version == "" {
		config.Version = "v1" // Default version
	}

	// Validate HTTP methods
	for i, method := range config.Methods {
		if method.Name == "" {
			return fmt.Errorf("HTTP method name is required at index %d", i)
		}
		if method.Method == "" {
			return fmt.Errorf("HTTP method verb is required for %s", method.Name)
		}
		if method.Path == "" {
			return fmt.Errorf("HTTP method path is required for %s", method.Name)
		}
		if !isValidHTTPMethod(method.Method) {
			return fmt.Errorf("invalid HTTP method %s for %s", method.Method, method.Name)
		}
	}

	// Validate gRPC methods
	for i, method := range config.GRPCMethods {
		if method.Name == "" {
			return fmt.Errorf("gRPC method name is required at index %d", i)
		}
	}

	return nil
}

// prepareAPITemplateData prepares the data to be passed to API templates.
func (ag *APIGenerator) prepareAPITemplateData(config interfaces.APIConfig) (*interfaces.TemplateData, error) {
	packageName := toPackageName(config.Service)
	apiName := toPascalCase(config.Name)

	templateData := &interfaces.TemplateData{
		Package: interfaces.PackageInfo{
			Name:       packageName,
			Path:       "internal/" + packageName,
			ModulePath: "", // Will be determined from context
			Version:    config.Version,
		},
		Imports:    ag.getAPIImports(config),
		Functions:  ag.getAPIFunctions(config),
		Structs:    ag.getAPIStructs(config),
		Interfaces: ag.getAPIInterfaces(config),
		Metadata:   config.Metadata,
		Timestamp:  time.Now(),
		Version:    config.Version,
	}

	// Add API-specific metadata
	if templateData.Metadata == nil {
		templateData.Metadata = make(map[string]string)
	}
	templateData.Metadata["api_name"] = apiName
	templateData.Metadata["api_version"] = config.Version
	templateData.Metadata["service_name"] = config.Service
	templateData.Metadata["package_name"] = packageName

	return templateData, nil
}

// generateHTTPHandlers generates HTTP handler files.
func (ag *APIGenerator) generateHTTPHandlers(config interfaces.APIConfig, data *interfaces.TemplateData) error {
	ag.logger.Debug("Generating HTTP handlers", "count", len(config.Methods))

	// Group methods by version and service
	serviceName := config.Service
	version := config.Version
	packageName := toPackageName(serviceName)

	// Create handler directory structure
	handlerDir := filepath.Join("internal", packageName, "handler", "http", strings.ToLower(config.Name), version)

	// Generate handler file
	handlerFile := ServiceFile{
		Path:     filepath.Join(handlerDir, strings.ToLower(config.Name)+".go"),
		Template: "api/http/handler.go",
		Data:     data,
		Mode:     0644,
	}

	if err := ag.generateAPIFile(handlerFile); err != nil {
		return fmt.Errorf("failed to generate HTTP handler: %w", err)
	}

	// Generate individual method handlers if needed
	for _, method := range config.Methods {
		methodData := ag.prepareMethodData(method, data)
		methodFile := ServiceFile{
			Path:     filepath.Join(handlerDir, strings.ToLower(method.Name)+".go"),
			Template: "api/http/method.go",
			Data:     methodData,
			Mode:     0644,
		}

		if err := ag.generateAPIFile(methodFile); err != nil {
			return fmt.Errorf("failed to generate HTTP method %s: %w", method.Name, err)
		}
	}

	// Generate handler tests
	testFile := ServiceFile{
		Path:     filepath.Join(handlerDir, strings.ToLower(config.Name)+"_test.go"),
		Template: "api/http/handler_test.go",
		Data:     data,
		Mode:     0644,
	}

	if err := ag.generateAPIFile(testFile); err != nil {
		return fmt.Errorf("failed to generate HTTP handler tests: %w", err)
	}

	return nil
}

// generateGRPCHandlers generates gRPC handler files.
func (ag *APIGenerator) generateGRPCHandlers(config interfaces.APIConfig, data *interfaces.TemplateData) error {
	ag.logger.Debug("Generating gRPC handlers", "count", len(config.GRPCMethods))

	serviceName := config.Service
	version := config.Version
	packageName := toPackageName(serviceName)

	// Create handler directory structure
	handlerDir := filepath.Join("internal", packageName, "handler", "grpc", strings.ToLower(config.Name), version)

	// Generate gRPC handler file
	handlerFile := ServiceFile{
		Path:     filepath.Join(handlerDir, strings.ToLower(config.Name)+".go"),
		Template: "api/grpc/handler.go",
		Data:     data,
		Mode:     0644,
	}

	if err := ag.generateAPIFile(handlerFile); err != nil {
		return fmt.Errorf("failed to generate gRPC handler: %w", err)
	}

	// Generate gRPC handler tests
	testFile := ServiceFile{
		Path:     filepath.Join(handlerDir, strings.ToLower(config.Name)+"_test.go"),
		Template: "api/grpc/handler_test.go",
		Data:     data,
		Mode:     0644,
	}

	if err := ag.generateAPIFile(testFile); err != nil {
		return fmt.Errorf("failed to generate gRPC handler tests: %w", err)
	}

	return nil
}

// generateProtoDefinitions generates Protocol Buffer definition files.
func (ag *APIGenerator) generateProtoDefinitions(config interfaces.APIConfig, data *interfaces.TemplateData) error {
	ag.logger.Debug("Generating proto definitions")

	serviceName := config.Service
	version := config.Version

	// Create proto directory structure
	protoDir := filepath.Join("api", "proto", strings.ToLower(serviceName), version)

	// Generate main service proto file
	protoFile := ServiceFile{
		Path:     filepath.Join(protoDir, strings.ToLower(config.Name)+".proto"),
		Template: "api/proto/service.proto",
		Data:     data,
		Mode:     0644,
	}

	if err := ag.generateAPIFile(protoFile); err != nil {
		return fmt.Errorf("failed to generate proto definition: %w", err)
	}

	// Generate message proto files if there are models
	if len(config.Models) > 0 {
		messagesFile := ServiceFile{
			Path:     filepath.Join(protoDir, "messages.proto"),
			Template: "api/proto/messages.proto",
			Data:     data,
			Mode:     0644,
		}

		if err := ag.generateAPIFile(messagesFile); err != nil {
			return fmt.Errorf("failed to generate proto messages: %w", err)
		}
	}

	return nil
}

// generateServiceLayer generates service layer implementation.
func (ag *APIGenerator) generateServiceLayer(config interfaces.APIConfig, data *interfaces.TemplateData) error {
	ag.logger.Debug("Generating service layer")

	serviceName := config.Service
	version := config.Version
	packageName := toPackageName(serviceName)

	// Create service directory structure
	serviceDir := filepath.Join("internal", packageName, "service", strings.ToLower(config.Name), version)

	// Generate service implementation
	serviceFile := ServiceFile{
		Path:     filepath.Join(serviceDir, strings.ToLower(config.Name)+".go"),
		Template: "api/service/service.go",
		Data:     data,
		Mode:     0644,
	}

	if err := ag.generateAPIFile(serviceFile); err != nil {
		return fmt.Errorf("failed to generate service implementation: %w", err)
	}

	// Generate service tests
	testFile := ServiceFile{
		Path:     filepath.Join(serviceDir, strings.ToLower(config.Name)+"_test.go"),
		Template: "api/service/service_test.go",
		Data:     data,
		Mode:     0644,
	}

	if err := ag.generateAPIFile(testFile); err != nil {
		return fmt.Errorf("failed to generate service tests: %w", err)
	}

	// Generate gRPC service handler if gRPC methods exist
	if len(config.GRPCMethods) > 0 {
		grpcFile := ServiceFile{
			Path:     filepath.Join(serviceDir, "grpc_handler.go"),
			Template: "api/service/grpc_handler.go",
			Data:     data,
			Mode:     0644,
		}

		if err := ag.generateAPIFile(grpcFile); err != nil {
			return fmt.Errorf("failed to generate gRPC service handler: %w", err)
		}

		grpcTestFile := ServiceFile{
			Path:     filepath.Join(serviceDir, "grpc_handler_test.go"),
			Template: "api/service/grpc_handler_test.go",
			Data:     data,
			Mode:     0644,
		}

		if err := ag.generateAPIFile(grpcTestFile); err != nil {
			return fmt.Errorf("failed to generate gRPC service handler tests: %w", err)
		}
	}

	return nil
}

// generateInterfaces generates interface definitions.
func (ag *APIGenerator) generateInterfaces(config interfaces.APIConfig, data *interfaces.TemplateData) error {
	ag.logger.Debug("Generating interfaces")

	serviceName := config.Service
	packageName := toPackageName(serviceName)

	// Update existing interfaces file or create new one
	interfacesDir := filepath.Join("internal", packageName, "interfaces")
	interfacesFile := ServiceFile{
		Path:     filepath.Join(interfacesDir, strings.ToLower(config.Name)+"_service.go"),
		Template: "api/interfaces/service.go",
		Data:     data,
		Mode:     0644,
	}

	if err := ag.generateAPIFile(interfacesFile); err != nil {
		return fmt.Errorf("failed to generate service interface: %w", err)
	}

	return nil
}

// generateModels generates model files.
func (ag *APIGenerator) generateModels(config interfaces.APIConfig, data *interfaces.TemplateData) error {
	ag.logger.Debug("Generating models", "count", len(config.Models))

	serviceName := config.Service
	packageName := toPackageName(serviceName)

	// Create model directory
	modelDir := filepath.Join("internal", packageName, "model")

	// Generate models based on configuration
	for _, model := range config.Models {
		modelData := ag.prepareModelDataFromModel(model, data)
		modelFile := ServiceFile{
			Path:     filepath.Join(modelDir, strings.ToLower(model.Name)+".go"),
			Template: "api/model/model.go",
			Data:     modelData,
			Mode:     0644,
		}

		if err := ag.generateAPIFile(modelFile); err != nil {
			return fmt.Errorf("failed to generate model %s: %w", model.Name, err)
		}
	}

	return nil
}

// generateAPIFile generates a single API file.
func (ag *APIGenerator) generateAPIFile(file ServiceFile) error {
	// Load template
	tmpl, err := ag.templateEngine.LoadTemplate(file.Template)
	if err != nil {
		return fmt.Errorf("failed to load template %s: %w", file.Template, err)
	}

	// Render template
	content, err := ag.templateEngine.RenderTemplate(tmpl, file.Data)
	if err != nil {
		return fmt.Errorf("failed to render template %s: %w", file.Template, err)
	}

	// Ensure directory exists
	dir := filepath.Dir(file.Path)
	if err := ag.fileSystem.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := ag.fileSystem.WriteFile(file.Path, content, int(file.Mode)); err != nil {
		return fmt.Errorf("failed to write file %s: %w", file.Path, err)
	}

	ag.logger.Debug("Generated API file", "path", file.Path)
	return nil
}

// getAPIImports returns required imports for API generation.
func (ag *APIGenerator) getAPIImports(config interfaces.APIConfig) []interfaces.ImportInfo {
	imports := []interfaces.ImportInfo{
		{Path: "context"},
		{Path: "fmt"},
		{Path: "net/http"},
	}

	// Add HTTP-specific imports
	if len(config.Methods) > 0 {
		imports = append(imports, []interfaces.ImportInfo{
			{Path: "github.com/gin-gonic/gin"},
			{Path: "github.com/innovationmech/swit/pkg/transport"},
		}...)
	}

	// Add gRPC-specific imports
	if len(config.GRPCMethods) > 0 {
		imports = append(imports, []interfaces.ImportInfo{
			{Path: "google.golang.org/grpc"},
			{Path: "google.golang.org/grpc/codes"},
			{Path: "google.golang.org/grpc/status"},
		}...)
	}

	// Add authentication imports if needed
	if config.Auth {
		imports = append(imports, interfaces.ImportInfo{
			Path: "github.com/innovationmech/swit/pkg/middleware",
		})
	}

	return imports
}

// getAPIFunctions returns functions for API generation.
func (ag *APIGenerator) getAPIFunctions(config interfaces.APIConfig) []interfaces.FunctionInfo {
	var functions []interfaces.FunctionInfo

	apiName := toPascalCase(config.Name)

	// HTTP handler functions
	for _, method := range config.Methods {
		methodName := toPascalCase(method.Name)
		functions = append(functions, interfaces.FunctionInfo{
			Name:    "Handle" + methodName,
			Comment: "Handle" + methodName + " handles the " + method.Name + " HTTP endpoint.",
			Parameters: []interfaces.ParameterInfo{
				{Name: "c", Type: "*gin.Context"},
			},
		})
	}

	// gRPC method functions
	for _, method := range config.GRPCMethods {
		methodName := toPascalCase(method.Name)
		functions = append(functions, interfaces.FunctionInfo{
			Name:    methodName,
			Comment: methodName + " implements the " + method.Name + " gRPC method.",
			Parameters: []interfaces.ParameterInfo{
				{Name: "ctx", Type: "context.Context"},
				{Name: "req", Type: "*" + toPascalCase(method.Request.Type)},
			},
			Returns: []interfaces.ReturnInfo{
				{Type: "*" + toPascalCase(method.Response.Type)},
				{Type: "error"},
			},
		})
	}

	// Constructor function
	functions = append(functions, interfaces.FunctionInfo{
		Name:    "New" + apiName + "Service",
		Comment: "New" + apiName + "Service creates a new " + apiName + " service instance.",
		Returns: []interfaces.ReturnInfo{
			{Type: apiName + "Service"},
		},
	})

	return functions
}

// getAPIStructs returns structs for API generation.
func (ag *APIGenerator) getAPIStructs(config interfaces.APIConfig) []interfaces.StructInfo {
	var structs []interfaces.StructInfo

	apiName := toPascalCase(config.Name)

	// Service struct
	structs = append(structs, interfaces.StructInfo{
		Name:    apiName + "Service",
		Comment: apiName + "Service provides " + config.Name + " functionality.",
		Fields: []interfaces.FieldInfo{
			{Name: "logger", Type: "interfaces.Logger", Comment: "Logger instance"},
		},
	})

	// HTTP handler struct
	if len(config.Methods) > 0 {
		structs = append(structs, interfaces.StructInfo{
			Name:    apiName + "HTTPHandler",
			Comment: apiName + "HTTPHandler handles HTTP requests for " + config.Name + ".",
			Fields: []interfaces.FieldInfo{
				{Name: "service", Type: apiName + "Service", Comment: "Service instance"},
				{Name: "logger", Type: "interfaces.Logger", Comment: "Logger instance"},
			},
		})
	}

	// gRPC handler struct
	if len(config.GRPCMethods) > 0 {
		structs = append(structs, interfaces.StructInfo{
			Name:    apiName + "GRPCHandler",
			Comment: apiName + "GRPCHandler handles gRPC requests for " + config.Name + ".",
			Fields: []interfaces.FieldInfo{
				{Name: "service", Type: apiName + "Service", Comment: "Service instance"},
				{Name: "logger", Type: "interfaces.Logger", Comment: "Logger instance"},
			},
		})
	}

	// Request/Response structs for HTTP methods
	for _, method := range config.Methods {
		if method.Request.Type != "" {
			structs = append(structs, interfaces.StructInfo{
				Name:    toPascalCase(method.Request.Type),
				Comment: toPascalCase(method.Request.Type) + " represents the request for " + method.Name + ".",
				Fields:  ag.convertFieldsToFieldInfo(method.Request.Fields),
			})
		}

		if method.Response.Type != "" {
			structs = append(structs, interfaces.StructInfo{
				Name:    toPascalCase(method.Response.Type),
				Comment: toPascalCase(method.Response.Type) + " represents the response for " + method.Name + ".",
				Fields:  ag.convertFieldsToFieldInfo(method.Response.Fields),
			})
		}
	}

	return structs
}

// getAPIInterfaces returns interfaces for API generation.
func (ag *APIGenerator) getAPIInterfaces(config interfaces.APIConfig) []interfaces.InterfaceInfo {
	var interfaceList []interfaces.InterfaceInfo

	apiName := toPascalCase(config.Name)

	// Service interface
	var methods []interfaces.MethodInfo

	// Add methods from HTTP endpoints
	for _, method := range config.Methods {
		methodName := toPascalCase(method.Name)
		methods = append(methods, interfaces.MethodInfo{
			Name:    methodName,
			Comment: methodName + " " + method.Description,
			Parameters: []interfaces.ParameterInfo{
				{Name: "ctx", Type: "context.Context"},
			},
			Returns: []interfaces.ReturnInfo{
				{Type: "error"},
			},
		})
	}

	// Add methods from gRPC endpoints
	for _, method := range config.GRPCMethods {
		methodName := toPascalCase(method.Name)
		methods = append(methods, interfaces.MethodInfo{
			Name:    methodName,
			Comment: methodName + " " + method.Description,
			Parameters: []interfaces.ParameterInfo{
				{Name: "ctx", Type: "context.Context"},
				{Name: "req", Type: "*" + toPascalCase(method.Request.Type)},
			},
			Returns: []interfaces.ReturnInfo{
				{Type: "*" + toPascalCase(method.Response.Type)},
				{Type: "error"},
			},
		})
	}

	interfaceList = append(interfaceList, interfaces.InterfaceInfo{
		Name:    apiName + "Service",
		Comment: apiName + "Service defines the interface for " + config.Name + " operations.",
		Methods: methods,
	})

	return interfaceList
}

// convertFieldsToFieldInfo converts Field slice to FieldInfo slice.
func (ag *APIGenerator) convertFieldsToFieldInfo(fields []interfaces.Field) []interfaces.FieldInfo {
	var fieldInfos []interfaces.FieldInfo

	for _, field := range fields {
		fieldInfo := interfaces.FieldInfo{
			Name:    toPascalCase(field.Name),
			Type:    field.Type,
			Tags:    field.Tags,
			Comment: field.Description,
		}
		fieldInfos = append(fieldInfos, fieldInfo)
	}

	return fieldInfos
}

// prepareMethodData prepares template data for a specific HTTP method.
func (ag *APIGenerator) prepareMethodData(method interfaces.HTTPMethod, baseData *interfaces.TemplateData) *interfaces.TemplateData {
	// Clone base data and add method-specific information
	methodData := *baseData
	methodData.Metadata = make(map[string]string)

	// Copy base metadata
	for k, v := range baseData.Metadata {
		methodData.Metadata[k] = v
	}

	// Add method-specific metadata
	methodData.Metadata["method_name"] = method.Name
	methodData.Metadata["method_verb"] = method.Method
	methodData.Metadata["method_path"] = method.Path
	methodData.Metadata["method_description"] = method.Description

	return &methodData
}

// prepareModelData prepares template data for a specific model.
func (ag *APIGenerator) prepareModelData(modelName string, baseData *interfaces.TemplateData) *interfaces.TemplateData {
	// Clone base data and add model-specific information
	modelData := *baseData
	modelData.Metadata = make(map[string]string)

	// Copy base metadata
	for k, v := range baseData.Metadata {
		modelData.Metadata[k] = v
	}

	// Add model-specific metadata
	modelData.Metadata["model_name"] = modelName
	modelData.Metadata["model_struct"] = toPascalCase(modelName)
	modelData.Metadata["model_package"] = toPackageName(modelName)

	return &modelData
}

// prepareModelDataFromModel prepares template data for a specific model object.
func (ag *APIGenerator) prepareModelDataFromModel(model interfaces.Model, baseData *interfaces.TemplateData) *interfaces.TemplateData {
	// Clone base data and add model-specific information
	modelData := *baseData
	modelData.Metadata = make(map[string]string)

	// Copy base metadata
	for k, v := range baseData.Metadata {
		modelData.Metadata[k] = v
	}

	// Add model-specific metadata
	modelData.Metadata["model_name"] = model.Name
	modelData.Metadata["model_struct"] = toPascalCase(model.Name)
	modelData.Metadata["model_package"] = toPackageName(model.Name)
	modelData.Metadata["model_description"] = model.Description

	return &modelData
}

// isValidHTTPMethod checks if the HTTP method is valid.
func isValidHTTPMethod(method string) bool {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	method = strings.ToUpper(method)

	for _, valid := range validMethods {
		if method == valid {
			return true
		}
	}

	return false
}
