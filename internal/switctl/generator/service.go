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

// Package generator provides code generation functionality for switctl.
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// ServiceGenerator generates service code and directory structure.
type ServiceGenerator struct {
	templateEngine interfaces.TemplateEngine
	fileSystem     interfaces.FileSystem
	logger         interfaces.Logger
}

// ServiceFile represents a file to be generated for the service.
type ServiceFile struct {
	Path     string
	Template string
	Data     interface{}
	Mode     os.FileMode
}

// NewServiceGenerator creates a new service generator.
func NewServiceGenerator(templateEngine interfaces.TemplateEngine, fileSystem interfaces.FileSystem, logger interfaces.Logger) *ServiceGenerator {
	return &ServiceGenerator{
		templateEngine: templateEngine,
		fileSystem:     fileSystem,
		logger:         logger,
	}
}

// GenerateService generates a complete service based on the configuration.
func (sg *ServiceGenerator) GenerateService(config interfaces.ServiceConfig) error {
	sg.logger.Info("Starting service generation", "service", config.Name)

	// Validate configuration
	if err := sg.validateConfig(config); err != nil {
		return fmt.Errorf("invalid service configuration: %w", err)
	}

	// Prepare template data
	templateData, err := sg.prepareTemplateData(config)
	if err != nil {
		return fmt.Errorf("failed to prepare template data: %w", err)
	}

	// Get files to generate
	files, err := sg.getFilesToGenerate(config, templateData)
	if err != nil {
		return fmt.Errorf("failed to get files to generate: %w", err)
	}

	// Create output directory
	outputDir := config.OutputDir
	if outputDir == "" {
		outputDir = config.Name
	}

	if err := sg.fileSystem.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Generate all files
	for _, file := range files {
		if err := sg.generateFile(outputDir, file); err != nil {
			return fmt.Errorf("failed to generate file %s: %w", file.Path, err)
		}
		sg.logger.Debug("Generated file", "path", file.Path)
	}

	// Generate additional files based on features
	if err := sg.generateFeatureFiles(config, templateData, outputDir); err != nil {
		return fmt.Errorf("failed to generate feature files: %w", err)
	}

	sg.logger.Info("Service generation completed successfully", "service", config.Name, "output", outputDir)
	return nil
}

// validateConfig validates the service configuration.
func (sg *ServiceGenerator) validateConfig(config interfaces.ServiceConfig) error {
	if config.Name == "" {
		return fmt.Errorf("service name is required")
	}

	if config.ModulePath == "" {
		return fmt.Errorf("module path is required")
	}

	// Validate service name format
	if !isValidServiceName(config.Name) {
		return fmt.Errorf("invalid service name: %s (must be lowercase with hyphens)", config.Name)
	}

	// Validate database configuration if enabled
	if config.Features.Database {
		if config.Database.Type == "" {
			config.Database.Type = "mysql" // Default
		}
		if config.Database.Database == "" {
			config.Database.Database = strings.ReplaceAll(config.Name, "-", "_")
		}
	}

	// Validate auth configuration if enabled
	if config.Features.Authentication {
		if config.Auth.Type == "" {
			config.Auth.Type = "jwt" // Default
		}
		if config.Auth.SecretKey == "" {
			return fmt.Errorf("auth secret key is required when authentication is enabled")
		}
	}

	// Validate ports
	if config.Ports.HTTP == 0 {
		config.Ports.HTTP = 9000 // Default
	}
	if config.Ports.GRPC == 0 {
		config.Ports.GRPC = 10000 // Default
	}

	return nil
}

// prepareTemplateData prepares the data to be passed to templates.
func (sg *ServiceGenerator) prepareTemplateData(config interfaces.ServiceConfig) (*interfaces.TemplateData, error) {
	// Calculate package information
	packageName := toPackageName(config.Name)
	serviceName := toPascalCase(config.Name)

	return &interfaces.TemplateData{
		Service: config,
		Package: interfaces.PackageInfo{
			Name:       packageName,
			Path:       config.ModulePath + "/internal/" + packageName,
			ModulePath: config.ModulePath,
			Version:    config.Version,
		},
		Imports: sg.getRequiredImports(config),
		Functions: []interfaces.FunctionInfo{
			{
				Name:    "New" + serviceName + "Server",
				Comment: "New" + serviceName + "Server creates a new " + serviceName + " server instance.",
			},
		},
		Structs:    sg.getServiceStructs(config),
		Interfaces: sg.getServiceInterfaces(config),
		Metadata:   config.Metadata,
		Timestamp:  time.Now(),
		Author:     config.Author,
		Version:    config.Version,
	}, nil
}

// getFilesToGenerate returns the list of files to generate for the service.
func (sg *ServiceGenerator) getFilesToGenerate(config interfaces.ServiceConfig, data *interfaces.TemplateData) ([]ServiceFile, error) {
	files := []ServiceFile{
		// Core service files
		{Path: "go.mod", Template: "service/go.mod", Data: data, Mode: 0644},
		{Path: "go.sum", Template: "service/go.sum", Data: data, Mode: 0644},
		{Path: "main.go", Template: "service/main.go", Data: data, Mode: 0644},
		{Path: "Makefile", Template: "service/Makefile", Data: data, Mode: 0644},
		{Path: "README.md", Template: "service/README.md", Data: data, Mode: 0644},
		{Path: ".gitignore", Template: "service/.gitignore", Data: data, Mode: 0644},

		// Internal package structure
		{Path: "internal/" + data.Package.Name + "/server.go", Template: "service/internal/server.go", Data: data, Mode: 0644},
		{Path: "internal/" + data.Package.Name + "/adapter.go", Template: "service/internal/adapter.go", Data: data, Mode: 0644},
		{Path: "internal/" + data.Package.Name + "/interfaces/interfaces.go", Template: "service/internal/interfaces/interfaces.go", Data: data, Mode: 0644},
		{Path: "internal/" + data.Package.Name + "/config/config.go", Template: "service/internal/config/config.go", Data: data, Mode: 0644},
		{Path: "internal/" + data.Package.Name + "/types/types.go", Template: "service/internal/types/types.go", Data: data, Mode: 0644},
		{Path: "internal/" + data.Package.Name + "/types/errors.go", Template: "service/internal/types/errors.go", Data: data, Mode: 0644},

		// Command structure
		{Path: "internal/" + data.Package.Name + "/cmd/cmd.go", Template: "service/internal/cmd/cmd.go", Data: data, Mode: 0644},
		{Path: "internal/" + data.Package.Name + "/cmd/serve/serve.go", Template: "service/internal/cmd/serve/serve.go", Data: data, Mode: 0644},
		{Path: "internal/" + data.Package.Name + "/cmd/version/version.go", Template: "service/internal/cmd/version/version.go", Data: data, Mode: 0644},

		// Deps directory (dependency injection)
		{Path: "internal/" + data.Package.Name + "/deps/.gitkeep", Template: "service/internal/deps/.gitkeep", Data: data, Mode: 0644},

		// Configuration files
		{Path: config.Name + ".yaml", Template: "service/config.yaml", Data: data, Mode: 0644},

		// Docker files
		{Path: "Dockerfile", Template: "service/Dockerfile", Data: data, Mode: 0644},
		{Path: ".dockerignore", Template: "service/.dockerignore", Data: data, Mode: 0644},
	}

	// Add HTTP handlers if needed
	if shouldGenerateHTTPHandlers(config) {
		files = append(files, []ServiceFile{
			{Path: "internal/" + data.Package.Name + "/handler/http/health/health.go", Template: "service/internal/handler/http/health/health.go", Data: data, Mode: 0644},
		}...)
	}

	// Add gRPC handlers if needed
	if shouldGenerateGRPCHandlers(config) {
		files = append(files, []ServiceFile{
			{Path: "internal/" + data.Package.Name + "/handler/grpc/.gitkeep", Template: "service/internal/handler/grpc/.gitkeep", Data: data, Mode: 0644},
		}...)
	}

	// Add service layer files
	files = append(files, []ServiceFile{
		{Path: "internal/" + data.Package.Name + "/service/health/health.go", Template: "service/internal/service/health/health.go", Data: data, Mode: 0644},
	}...)

	return files, nil
}

// generateFile generates a single file using the template engine.
func (sg *ServiceGenerator) generateFile(outputDir string, file ServiceFile) error {
	// Load template
	tmpl, err := sg.templateEngine.LoadTemplate(file.Template)
	if err != nil {
		return fmt.Errorf("failed to load template %s: %w", file.Template, err)
	}

	// Render template
	content, err := sg.templateEngine.RenderTemplate(tmpl, file.Data)
	if err != nil {
		return fmt.Errorf("failed to render template %s: %w", file.Template, err)
	}

	// Ensure directory exists
	fullPath := filepath.Join(outputDir, file.Path)
	dir := filepath.Dir(fullPath)
	if err := sg.fileSystem.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := sg.fileSystem.WriteFile(fullPath, content, int(file.Mode)); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

// generateFeatureFiles generates additional files based on enabled features.
func (sg *ServiceGenerator) generateFeatureFiles(config interfaces.ServiceConfig, data *interfaces.TemplateData, outputDir string) error {
	// Database files
	if config.Features.Database {
		if err := sg.generateDatabaseFiles(config, data, outputDir); err != nil {
			return fmt.Errorf("failed to generate database files: %w", err)
		}
	}

	// Authentication files
	if config.Features.Authentication {
		if err := sg.generateAuthFiles(config, data, outputDir); err != nil {
			return fmt.Errorf("failed to generate auth files: %w", err)
		}
	}

	// Cache files
	if config.Features.Cache {
		if err := sg.generateCacheFiles(config, data, outputDir); err != nil {
			return fmt.Errorf("failed to generate cache files: %w", err)
		}
	}

	// Message queue files
	if config.Features.MessageQueue {
		if err := sg.generateMessageQueueFiles(config, data, outputDir); err != nil {
			return fmt.Errorf("failed to generate message queue files: %w", err)
		}
	}

	// Kubernetes files
	if config.Features.Kubernetes {
		if err := sg.generateKubernetesFiles(config, data, outputDir); err != nil {
			return fmt.Errorf("failed to generate Kubernetes files: %w", err)
		}
	}

	return nil
}

// generateDatabaseFiles generates database-related files.
func (sg *ServiceGenerator) generateDatabaseFiles(config interfaces.ServiceConfig, data *interfaces.TemplateData, outputDir string) error {
	files := []ServiceFile{
		{Path: "internal/" + data.Package.Name + "/db/db.go", Template: "service/internal/db/db.go", Data: data, Mode: 0644},
		{Path: "internal/" + data.Package.Name + "/model/.gitkeep", Template: "service/internal/model/.gitkeep", Data: data, Mode: 0644},
		{Path: "internal/" + data.Package.Name + "/repository/.gitkeep", Template: "service/internal/repository/.gitkeep", Data: data, Mode: 0644},
		{Path: "migrations/.gitkeep", Template: "service/migrations/.gitkeep", Data: data, Mode: 0644},
	}

	for _, file := range files {
		if err := sg.generateFile(outputDir, file); err != nil {
			return err
		}
	}

	return nil
}

// generateAuthFiles generates authentication-related files.
func (sg *ServiceGenerator) generateAuthFiles(config interfaces.ServiceConfig, data *interfaces.TemplateData, outputDir string) error {
	files := []ServiceFile{
		{Path: "internal/" + data.Package.Name + "/auth/auth.go", Template: "service/internal/auth/auth.go", Data: data, Mode: 0644},
		{Path: "internal/" + data.Package.Name + "/middleware/auth.go", Template: "service/internal/middleware/auth.go", Data: data, Mode: 0644},
	}

	for _, file := range files {
		if err := sg.generateFile(outputDir, file); err != nil {
			return err
		}
	}

	return nil
}

// generateCacheFiles generates cache-related files.
func (sg *ServiceGenerator) generateCacheFiles(config interfaces.ServiceConfig, data *interfaces.TemplateData, outputDir string) error {
	files := []ServiceFile{
		{Path: "internal/" + data.Package.Name + "/cache/cache.go", Template: "service/internal/cache/cache.go", Data: data, Mode: 0644},
	}

	for _, file := range files {
		if err := sg.generateFile(outputDir, file); err != nil {
			return err
		}
	}

	return nil
}

// generateMessageQueueFiles generates message queue-related files.
func (sg *ServiceGenerator) generateMessageQueueFiles(config interfaces.ServiceConfig, data *interfaces.TemplateData, outputDir string) error {
	files := []ServiceFile{
		{Path: "internal/" + data.Package.Name + "/queue/queue.go", Template: "service/internal/queue/queue.go", Data: data, Mode: 0644},
		{Path: "internal/" + data.Package.Name + "/consumer/.gitkeep", Template: "service/internal/consumer/.gitkeep", Data: data, Mode: 0644},
		{Path: "internal/" + data.Package.Name + "/producer/.gitkeep", Template: "service/internal/producer/.gitkeep", Data: data, Mode: 0644},
	}

	for _, file := range files {
		if err := sg.generateFile(outputDir, file); err != nil {
			return err
		}
	}

	return nil
}

// generateKubernetesFiles generates Kubernetes deployment files.
func (sg *ServiceGenerator) generateKubernetesFiles(config interfaces.ServiceConfig, data *interfaces.TemplateData, outputDir string) error {
	files := []ServiceFile{
		{Path: "deployments/k8s/deployment.yaml", Template: "service/deployments/k8s/deployment.yaml", Data: data, Mode: 0644},
		{Path: "deployments/k8s/service.yaml", Template: "service/deployments/k8s/service.yaml", Data: data, Mode: 0644},
		{Path: "deployments/k8s/configmap.yaml", Template: "service/deployments/k8s/configmap.yaml", Data: data, Mode: 0644},
		{Path: "deployments/k8s/secret.yaml", Template: "service/deployments/k8s/secret.yaml", Data: data, Mode: 0644},
		{Path: "deployments/k8s/ingress.yaml", Template: "service/deployments/k8s/ingress.yaml", Data: data, Mode: 0644},
	}

	for _, file := range files {
		if err := sg.generateFile(outputDir, file); err != nil {
			return err
		}
	}

	return nil
}

// getRequiredImports returns the required imports for the service.
func (sg *ServiceGenerator) getRequiredImports(config interfaces.ServiceConfig) []interfaces.ImportInfo {
	imports := []interfaces.ImportInfo{
		{Path: "context"},
		{Path: "fmt"},
		{Path: "log"},
		{Path: "os"},
		{Path: "github.com/spf13/cobra"},
		{Path: "github.com/innovationmech/swit/pkg/server"},
		{Path: "github.com/innovationmech/swit/pkg/transport"},
	}

	// Add feature-specific imports
	if config.Features.Database {
		imports = append(imports, interfaces.ImportInfo{Path: "gorm.io/gorm"})
		switch config.Database.Type {
		case "mysql":
			imports = append(imports, interfaces.ImportInfo{Path: "gorm.io/driver/mysql"})
		case "postgres":
			imports = append(imports, interfaces.ImportInfo{Path: "gorm.io/driver/postgres"})
		case "sqlite":
			imports = append(imports, interfaces.ImportInfo{Path: "gorm.io/driver/sqlite"})
		}
	}

	if config.Features.Cache {
		imports = append(imports, interfaces.ImportInfo{Path: "github.com/go-redis/redis/v8"})
	}

	if config.Features.Authentication {
		imports = append(imports, interfaces.ImportInfo{Path: "github.com/golang-jwt/jwt/v4"})
	}

	if config.Features.MessageQueue {
		imports = append(imports, interfaces.ImportInfo{Path: "github.com/streadway/amqp"})
	}

	return imports
}

// getServiceStructs returns the structs for the service.
func (sg *ServiceGenerator) getServiceStructs(config interfaces.ServiceConfig) []interfaces.StructInfo {
	serviceName := toPascalCase(config.Name)

	structs := []interfaces.StructInfo{
		{
			Name:    serviceName + "Server",
			Comment: serviceName + "Server represents the main server instance.",
			Fields: []interfaces.FieldInfo{
				{Name: "config", Type: "*Config", Comment: "Server configuration"},
				{Name: "server", Type: "server.BusinessServerCore", Comment: "Core server instance"},
			},
		},
		{
			Name:    "Config",
			Comment: "Config represents the service configuration.",
			Fields: []interfaces.FieldInfo{
				{Name: "Server", Type: "server.ServerConfig", Tags: map[string]string{"yaml": "server", "json": "server"}},
				{Name: "Service", Type: "ServiceConfig", Tags: map[string]string{"yaml": "service", "json": "service"}},
			},
		},
		{
			Name:    "ServiceConfig",
			Comment: "ServiceConfig represents service-specific configuration.",
			Fields: []interfaces.FieldInfo{
				{Name: "Name", Type: "string", Tags: map[string]string{"yaml": "name", "json": "name"}},
				{Name: "Version", Type: "string", Tags: map[string]string{"yaml": "version", "json": "version"}},
			},
		},
	}

	// Add database-specific structs
	if config.Features.Database {
		structs = append(structs, interfaces.StructInfo{
			Name:    "DatabaseConfig",
			Comment: "DatabaseConfig represents database configuration.",
			Fields: []interfaces.FieldInfo{
				{Name: "Type", Type: "string", Tags: map[string]string{"yaml": "type", "json": "type"}},
				{Name: "Host", Type: "string", Tags: map[string]string{"yaml": "host", "json": "host"}},
				{Name: "Port", Type: "int", Tags: map[string]string{"yaml": "port", "json": "port"}},
				{Name: "Database", Type: "string", Tags: map[string]string{"yaml": "database", "json": "database"}},
				{Name: "Username", Type: "string", Tags: map[string]string{"yaml": "username", "json": "username"}},
				{Name: "Password", Type: "string", Tags: map[string]string{"yaml": "password", "json": "password"}},
			},
		})
	}

	return structs
}

// getServiceInterfaces returns the interfaces for the service.
func (sg *ServiceGenerator) getServiceInterfaces(config interfaces.ServiceConfig) []interfaces.InterfaceInfo {
	interfaces := []interfaces.InterfaceInfo{
		{
			Name:    "HealthService",
			Comment: "HealthService provides health check functionality.",
			Methods: []interfaces.MethodInfo{
				{
					Name:    "Check",
					Comment: "Check performs a health check.",
					Parameters: []interfaces.ParameterInfo{
						{Name: "ctx", Type: "context.Context"},
					},
					Returns: []interfaces.ReturnInfo{
						{Type: "error"},
					},
				},
			},
		},
	}

	return interfaces
}

// Helper functions

func isValidServiceName(name string) bool {
	if name == "" {
		return false
	}

	// Check if name contains only lowercase letters, numbers, and hyphens
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}

	// Must start and end with alphanumeric
	return (name[0] >= 'a' && name[0] <= 'z') || (name[0] >= '0' && name[0] <= '9')
}

func shouldGenerateHTTPHandlers(config interfaces.ServiceConfig) bool {
	return true // Always generate HTTP handlers for now
}

func shouldGenerateGRPCHandlers(config interfaces.ServiceConfig) bool {
	return true // Always generate gRPC handlers for now
}

func toPackageName(s string) string {
	// Convert service name to valid Go package name
	name := strings.ReplaceAll(s, "-", "")
	return strings.ToLower(name)
}

func toPascalCase(s string) string {
	if s == "" {
		return s
	}

	// Split by hyphens and capitalize each part
	parts := strings.Split(s, "-")
	var result strings.Builder

	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				result.WriteString(strings.ToLower(part[1:]))
			}
		}
	}

	return result.String()
}
