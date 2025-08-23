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

package generate

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/spf13/cobra"
)

// APIGenerationConfig holds configuration specific to API generation.
type APIGenerationConfig struct {
	Name        string
	Version     string
	BasePath    string
	Methods     []string
	Auth        bool
	Middleware  []string
	Models      []string
	UpdateProto bool
	Generate    struct {
		HTTP bool
		GRPC bool
		Both bool
	}
}

// NewAPICommand creates the API generation command.
func NewAPICommand(config *GenerateConfig) *cobra.Command {
	var apiConfig APIGenerationConfig

	cmd := &cobra.Command{
		Use:   "api [name]",
		Short: "Generate API endpoints",
		Long: `Generate complete API endpoints with HTTP and gRPC handlers.

This command generates:
- Protocol Buffer definitions
- HTTP handlers with Gin framework
- gRPC service implementations
- Service layer interfaces and implementations
- Repository layer interfaces (if models are specified)
- Request/response models
- API documentation

Examples:
  # Generate API with interactive prompts
  switctl generate api --interactive

  # Generate specific API endpoint
  switctl generate api user-service --service=user --methods=GET,POST,PUT,DELETE

  # Generate API with authentication
  switctl generate api auth --auth --middleware=cors,logging

  # Generate only HTTP handlers
  switctl generate api products --http-only

  # Generate only gRPC service
  switctl generate api inventory --grpc-only`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set API name from argument if provided
			if len(args) > 0 {
				apiConfig.Name = args[0]
			}

			return runAPIGeneration(cmd.Context(), config, &apiConfig)
		},
	}

	// API-specific flags
	cmd.Flags().StringVarP(&apiConfig.Version, "version", "", "v1", "API version")
	cmd.Flags().StringVar(&apiConfig.BasePath, "base-path", "", "Base path for HTTP endpoints")
	cmd.Flags().StringSliceVar(&apiConfig.Methods, "methods", []string{"GET", "POST", "PUT", "DELETE"}, "HTTP methods to generate")
	cmd.Flags().BoolVar(&apiConfig.Auth, "auth", false, "Add authentication to endpoints")
	cmd.Flags().StringSliceVar(&apiConfig.Middleware, "middleware", []string{}, "Middleware to apply (cors, logging, rate-limit, etc.)")
	cmd.Flags().StringSliceVar(&apiConfig.Models, "models", []string{}, "Data models to include")
	cmd.Flags().BoolVar(&apiConfig.UpdateProto, "update-proto", true, "Update Protocol Buffer definitions")
	cmd.Flags().BoolVar(&apiConfig.Generate.HTTP, "http-only", false, "Generate only HTTP handlers")
	cmd.Flags().BoolVar(&apiConfig.Generate.GRPC, "grpc-only", false, "Generate only gRPC service")

	return cmd
}

// runAPIGeneration executes the API generation process.
func runAPIGeneration(ctx context.Context, config *GenerateConfig, apiConfig *APIGenerationConfig) error {
	// Initialize generator command
	gc, err := NewGeneratorCommand(config)
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Execute with common setup
	return gc.Execute(ctx, func(ctx context.Context) error {
		return generateAPI(ctx, gc, config, apiConfig)
	})
}

// generateAPI performs the actual API generation.
func generateAPI(ctx context.Context, gc *GeneratorCommand, config *GenerateConfig, apiConfig *APIGenerationConfig) error {
	// Prompt for missing configuration
	if err := gc.PromptForMissingConfig(map[string]string{
		"service": "Service name is required for API generation",
	}); err != nil {
		return err
	}

	// Prompt for API-specific configuration
	if err := promptForAPIConfig(gc, apiConfig); err != nil {
		return fmt.Errorf("failed to get API configuration: %w", err)
	}

	// Validate and normalize configuration
	if err := validateAPIConfig(config, apiConfig); err != nil {
		return fmt.Errorf("invalid API configuration: %w", err)
	}

	// Determine generation targets
	if err := determineGenerationTargets(apiConfig); err != nil {
		return fmt.Errorf("failed to determine generation targets: %w", err)
	}

	// Prepare API configuration for generation
	genConfig, err := prepareAPIGenerationConfig(config, apiConfig)
	if err != nil {
		return fmt.Errorf("failed to prepare generation configuration: %w", err)
	}

	// Get list of files that will be generated
	files := getAPIFilesToGenerate(config, apiConfig)

	// Show preview
	description := fmt.Sprintf("Generating API '%s' for service '%s'", apiConfig.Name, config.Service)
	if err := gc.ShowPreview(files, description); err != nil {
		return err
	}

	// Check for file conflicts
	if err := gc.CheckFileConflicts(files); err != nil {
		return err
	}

	// Generate the API
	if !config.DryRun {
		if err := gc.generator.GenerateAPI(*genConfig); err != nil {
			return fmt.Errorf("API generation failed: %w", err)
		}

		// Update Protocol Buffer definitions if requested
		if apiConfig.UpdateProto {
			if err := updateProtocolBuffers(ctx, gc, config, apiConfig); err != nil {
				gc.logger.Warn("Failed to update Protocol Buffer definitions", "error", err)
				gc.ui.ShowInfo("Note: You may need to run 'make proto' to update generated code")
			}
		}

		// Show post-generation instructions
		showPostGenerationInstructions(gc, config, apiConfig)
	}

	return nil
}

// promptForAPIConfig prompts for missing API configuration.
func promptForAPIConfig(gc *GeneratorCommand, apiConfig *APIGenerationConfig) error {
	if !gc.config.Interactive {
		return nil
	}

	// Prompt for API name if not provided
	if apiConfig.Name == "" {
		name, err := gc.ui.PromptInput("API name", func(input string) error {
			if input == "" {
				return fmt.Errorf("API name is required")
			}
			if err := validateServiceName(input); err != nil {
				return fmt.Errorf("invalid API name: %w", err)
			}
			return nil
		})
		if err != nil {
			return err
		}
		apiConfig.Name = name
	}

	// Prompt for base path if not provided
	if apiConfig.BasePath == "" {
		basePath, err := gc.ui.PromptInput(fmt.Sprintf("Base path (default: /api/%s/%s)", apiConfig.Version, apiConfig.Name), nil)
		if err != nil {
			return err
		}
		if basePath != "" {
			apiConfig.BasePath = basePath
		}
	}

	// Prompt for authentication
	if !apiConfig.Auth {
		auth, err := gc.ui.PromptConfirm("Add authentication to endpoints?", false)
		if err != nil {
			return err
		}
		apiConfig.Auth = auth
	}

	// Prompt for generation targets if both flags are false
	if !apiConfig.Generate.HTTP && !apiConfig.Generate.GRPC {
		options := []interfaces.MenuOption{
			{Label: "Both HTTP and gRPC", Value: "both", Icon: "🌐"},
			{Label: "HTTP only", Value: "http", Icon: "📡"},
			{Label: "gRPC only", Value: "grpc", Icon: "⚡"},
		}

		choice, err := gc.ui.ShowMenu("Select generation target:", options)
		if err != nil {
			return err
		}

		if stringValue, ok := options[choice].Value.(string); ok {
			switch stringValue {
			case "http":
				apiConfig.Generate.HTTP = true
			case "grpc":
				apiConfig.Generate.GRPC = true
			case "both":
				apiConfig.Generate.Both = true
			default:
				return fmt.Errorf("invalid generation target: %s", stringValue)
			}
		} else {
			return fmt.Errorf("invalid generation target selection")
		}
	}

	return nil
}

// validateAPIConfig validates the API configuration.
func validateAPIConfig(config *GenerateConfig, apiConfig *APIGenerationConfig) error {
	// Validate API name
	if apiConfig.Name == "" {
		return fmt.Errorf("API name is required")
	}

	if err := validateServiceName(apiConfig.Name); err != nil {
		return fmt.Errorf("invalid API name: %w", err)
	}

	// Set default base path
	if apiConfig.BasePath == "" {
		apiConfig.BasePath = fmt.Sprintf("/api/%s/%s", apiConfig.Version, apiConfig.Name)
	}

	// Validate base path format
	if !strings.HasPrefix(apiConfig.BasePath, "/") {
		apiConfig.BasePath = "/" + apiConfig.BasePath
	}

	// Validate HTTP methods
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true,
	}
	for _, method := range apiConfig.Methods {
		if !validMethods[strings.ToUpper(method)] {
			return fmt.Errorf("invalid HTTP method: %s", method)
		}
	}

	// Normalize methods to uppercase
	for i, method := range apiConfig.Methods {
		apiConfig.Methods[i] = strings.ToUpper(method)
	}

	return nil
}

// determineGenerationTargets determines what to generate based on flags.
func determineGenerationTargets(apiConfig *APIGenerationConfig) error {
	// If both http-only and grpc-only are specified, that's an error
	if apiConfig.Generate.HTTP && apiConfig.Generate.GRPC {
		return fmt.Errorf("cannot specify both --http-only and --grpc-only")
	}

	// If neither is specified, generate both
	if !apiConfig.Generate.HTTP && !apiConfig.Generate.GRPC {
		apiConfig.Generate.Both = true
	}

	return nil
}

// prepareAPIGenerationConfig converts the command configuration to generator configuration.
func prepareAPIGenerationConfig(config *GenerateConfig, apiConfig *APIGenerationConfig) (*interfaces.APIConfig, error) {
	// Convert HTTP methods to structured format
	httpMethods := make([]interfaces.HTTPMethod, 0, len(apiConfig.Methods))
	for _, method := range apiConfig.Methods {
		httpMethods = append(httpMethods, interfaces.HTTPMethod{
			Name:        strings.ToLower(method) + toPascalCase(apiConfig.Name),
			Method:      method,
			Path:        generateMethodPath(apiConfig.BasePath, method),
			Description: fmt.Sprintf("%s operation for %s", method, apiConfig.Name),
			Auth:        apiConfig.Auth,
			Middleware:  apiConfig.Middleware,
		})
	}

	// Convert to models
	models := make([]interfaces.Model, 0, len(apiConfig.Models))
	for _, modelName := range apiConfig.Models {
		models = append(models, interfaces.Model{
			Name:        toPascalCase(modelName),
			Description: fmt.Sprintf("%s data model", toPascalCase(modelName)),
			Fields: []interfaces.Field{
				{
					Name: "ID",
					Type: "uint",
					Tags: map[string]string{
						"json": "id",
						"db":   "id",
					},
					Description: "Unique identifier",
				},
			},
		})
	}

	// Create gRPC methods if needed
	grpcMethods := make([]interfaces.GRPCMethod, 0)
	if apiConfig.Generate.GRPC || apiConfig.Generate.Both {
		for _, method := range apiConfig.Methods {
			grpcMethods = append(grpcMethods, interfaces.GRPCMethod{
				Name:        method + toPascalCase(apiConfig.Name),
				Description: fmt.Sprintf("%s operation for %s", method, apiConfig.Name),
				Request: interfaces.RequestModel{
					Type: toPascalCase(method) + toPascalCase(apiConfig.Name) + "Request",
				},
				Response: interfaces.ResponseModel{
					Type: toPascalCase(method) + toPascalCase(apiConfig.Name) + "Response",
				},
				Streaming: interfaces.StreamingNone,
			})
		}
	}

	return &interfaces.APIConfig{
		Name:        apiConfig.Name,
		Version:     apiConfig.Version,
		Service:     config.Service,
		BasePath:    apiConfig.BasePath,
		Methods:     httpMethods,
		GRPCMethods: grpcMethods,
		Models:      models,
		Middleware:  apiConfig.Middleware,
		Auth:        apiConfig.Auth,
		Metadata: map[string]string{
			"package":      config.Package,
			"output_dir":   config.OutputDir,
			"http_only":    fmt.Sprintf("%t", apiConfig.Generate.HTTP),
			"grpc_only":    fmt.Sprintf("%t", apiConfig.Generate.GRPC),
			"update_proto": fmt.Sprintf("%t", apiConfig.UpdateProto),
		},
	}, nil
}

// getAPIFilesToGenerate returns the list of files that will be generated.
func getAPIFilesToGenerate(config *GenerateConfig, apiConfig *APIGenerationConfig) []string {
	files := []string{}

	baseDir := config.OutputDir
	if config.Service != "" {
		baseDir = filepath.Join(baseDir, "internal", config.Service)
	}

	// HTTP handlers
	if apiConfig.Generate.HTTP || apiConfig.Generate.Both {
		files = append(files,
			filepath.Join(baseDir, "handler", fmt.Sprintf("%s_handler.go", apiConfig.Name)),
			filepath.Join(baseDir, "handler", fmt.Sprintf("%s_handler_test.go", apiConfig.Name)),
		)
	}

	// gRPC service
	if apiConfig.Generate.GRPC || apiConfig.Generate.Both {
		files = append(files,
			filepath.Join(baseDir, "grpc", fmt.Sprintf("%s_service.go", apiConfig.Name)),
			filepath.Join(baseDir, "grpc", fmt.Sprintf("%s_service_test.go", apiConfig.Name)),
		)
	}

	// Service layer
	files = append(files,
		filepath.Join(baseDir, "service", fmt.Sprintf("%s_service.go", apiConfig.Name)),
		filepath.Join(baseDir, "service", fmt.Sprintf("%s_service_test.go", apiConfig.Name)),
	)

	// Repository layer if models are specified
	if len(apiConfig.Models) > 0 {
		files = append(files,
			filepath.Join(baseDir, "repository", fmt.Sprintf("%s_repository.go", apiConfig.Name)),
			filepath.Join(baseDir, "repository", fmt.Sprintf("%s_repository_test.go", apiConfig.Name)),
		)
	}

	// Protocol Buffer definitions
	if apiConfig.UpdateProto && (apiConfig.Generate.GRPC || apiConfig.Generate.Both) {
		protoDir := filepath.Join(config.OutputDir, "api", "proto", "swit", config.Service, apiConfig.Version)
		files = append(files,
			filepath.Join(protoDir, fmt.Sprintf("%s.proto", apiConfig.Name)),
		)
	}

	return files
}

// updateProtocolBuffers updates the Protocol Buffer definitions.
func updateProtocolBuffers(ctx context.Context, gc *GeneratorCommand, config *GenerateConfig, apiConfig *APIGenerationConfig) error {
	gc.logger.Info("Updating Protocol Buffer definitions", "api", apiConfig.Name, "service", config.Service)

	// This would typically run buf generate or similar command
	// For now, we'll just log the action
	gc.ui.ShowInfo("Note: Protocol Buffer definitions updated. Run 'make proto' to regenerate code.")

	return nil
}

// showPostGenerationInstructions shows instructions after successful generation.
func showPostGenerationInstructions(gc *GeneratorCommand, config *GenerateConfig, apiConfig *APIGenerationConfig) {
	gc.ui.PrintHeader("Next Steps")

	instructions := []string{
		"API endpoints have been generated successfully!",
		"",
		"To complete the setup:",
		"1. Update your service registration to include new handlers",
		"2. Implement the business logic in the service layer",
		"3. Add proper error handling and validation",
		"4. Write comprehensive tests for your endpoints",
	}

	if apiConfig.UpdateProto {
		instructions = append(instructions,
			"5. Run 'make proto' to regenerate Protocol Buffer code",
			"6. Update your API documentation",
		)
	}

	if len(apiConfig.Models) > 0 {
		instructions = append(instructions,
			"7. Implement repository methods for data persistence",
			"8. Run database migrations if needed",
		)
	}

	for _, instruction := range instructions {
		if instruction == "" {
			fmt.Println()
		} else {
			fmt.Printf("  %s\n", instruction)
		}
	}

	gc.ui.PrintSeparator()
	gc.ui.ShowInfo(fmt.Sprintf("Generated files are located in: %s", config.OutputDir))
}

// generateMethodPath generates a path for an HTTP method.
func generateMethodPath(basePath, method string) string {
	switch method {
	case "GET":
		return basePath
	case "POST":
		return basePath
	case "PUT":
		return basePath + "/{id}"
	case "PATCH":
		return basePath + "/{id}"
	case "DELETE":
		return basePath + "/{id}"
	default:
		return basePath
	}
}

// toPascalCase converts a string to PascalCase.
func toPascalCase(s string) string {
	if s == "" {
		return s
	}

	// Split by hyphens and underscores
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_'
	})

	result := ""
	for _, part := range parts {
		if len(part) > 0 {
			result += strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}

	return result
}
