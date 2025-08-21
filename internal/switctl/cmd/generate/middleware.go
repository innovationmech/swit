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

package generate

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/spf13/cobra"
)

// MiddlewareGenerationConfig holds configuration specific to middleware generation.
type MiddlewareGenerationConfig struct {
	Name         string
	Type         string
	Description  string
	Features     []string
	Dependencies []string
	Config       map[string]interface{}
	Templates    []string
}

// MiddlewareType represents supported middleware types.
type MiddlewareType string

const (
	MiddlewareTypeHTTP MiddlewareType = "http"
	MiddlewareTypeGRPC MiddlewareType = "grpc"
	MiddlewareTypeBoth MiddlewareType = "both"
)

// MiddlewareTemplate represents predefined middleware templates.
type MiddlewareTemplate struct {
	Name         string
	Description  string
	Type         MiddlewareType
	Features     []string
	Dependencies []string
}

// Predefined middleware templates
var middlewareTemplates = map[string]MiddlewareTemplate{
	"cors": {
		Name:         "cors",
		Description:  "Cross-Origin Resource Sharing middleware",
		Type:         MiddlewareTypeHTTP,
		Features:     []string{"cors", "security"},
		Dependencies: []string{"github.com/gin-contrib/cors"},
	},
	"auth": {
		Name:         "auth",
		Description:  "Authentication and authorization middleware",
		Type:         MiddlewareTypeBoth,
		Features:     []string{"auth", "jwt", "security"},
		Dependencies: []string{"github.com/golang-jwt/jwt/v4"},
	},
	"logging": {
		Name:         "logging",
		Description:  "Request/response logging middleware",
		Type:         MiddlewareTypeBoth,
		Features:     []string{"logging", "monitoring"},
		Dependencies: []string{"go.uber.org/zap"},
	},
	"ratelimit": {
		Name:         "ratelimit",
		Description:  "Rate limiting middleware",
		Type:         MiddlewareTypeBoth,
		Features:     []string{"ratelimit", "security", "performance"},
		Dependencies: []string{"golang.org/x/time/rate"},
	},
	"recovery": {
		Name:         "recovery",
		Description:  "Panic recovery middleware",
		Type:         MiddlewareTypeBoth,
		Features:     []string{"recovery", "stability"},
		Dependencies: []string{},
	},
	"metrics": {
		Name:         "metrics",
		Description:  "Prometheus metrics collection middleware",
		Type:         MiddlewareTypeBoth,
		Features:     []string{"metrics", "monitoring", "prometheus"},
		Dependencies: []string{"github.com/prometheus/client_golang"},
	},
	"timeout": {
		Name:         "timeout",
		Description:  "Request timeout middleware",
		Type:         MiddlewareTypeBoth,
		Features:     []string{"timeout", "performance"},
		Dependencies: []string{},
	},
	"compress": {
		Name:         "compress",
		Description:  "Response compression middleware",
		Type:         MiddlewareTypeHTTP,
		Features:     []string{"compression", "performance"},
		Dependencies: []string{"github.com/gin-contrib/gzip"},
	},
}

// NewMiddlewareCommand creates the middleware generation command.
func NewMiddlewareCommand(config *GenerateConfig) *cobra.Command {
	var middlewareConfig MiddlewareGenerationConfig

	cmd := &cobra.Command{
		Use:   "middleware [name]",
		Short: "Generate HTTP and gRPC middleware templates",
		Long: `Generate middleware components for HTTP and gRPC services.

This command generates:
- HTTP middleware for Gin framework
- gRPC interceptors for unary and streaming calls
- Configuration structures and validation
- Unit tests with mock dependencies
- Integration examples and documentation

Predefined templates available:
- cors: Cross-Origin Resource Sharing
- auth: Authentication and authorization
- logging: Request/response logging
- ratelimit: Rate limiting
- recovery: Panic recovery
- metrics: Prometheus metrics
- timeout: Request timeout
- compress: Response compression

Examples:
  # Generate middleware with interactive prompts
  switctl generate middleware --interactive

  # Generate from predefined template
  switctl generate middleware auth --template=auth

  # Generate custom HTTP-only middleware
  switctl generate middleware custom --type=http --features=security,validation

  # Generate middleware for both HTTP and gRPC
  switctl generate middleware monitor --type=both --features=logging,metrics

  # List available templates
  switctl generate middleware --list-templates`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle --list-templates flag
			if listTemplates, _ := cmd.Flags().GetBool("list-templates"); listTemplates {
				return showMiddlewareTemplates()
			}

			// Set middleware name from argument if provided
			if len(args) > 0 {
				middlewareConfig.Name = args[0]
			}

			return runMiddlewareGeneration(cmd.Context(), config, &middlewareConfig)
		},
	}

	// Middleware-specific flags
	cmd.Flags().StringVar(&middlewareConfig.Type, "type", "both", "Middleware type (http, grpc, both)")
	cmd.Flags().StringVar(&middlewareConfig.Description, "description", "", "Middleware description")
	cmd.Flags().StringSliceVar(&middlewareConfig.Features, "features", []string{}, "Middleware features (auth, logging, metrics, etc.)")
	cmd.Flags().StringSliceVar(&middlewareConfig.Dependencies, "dependencies", []string{}, "Additional Go module dependencies")
	cmd.Flags().StringSliceVar(&middlewareConfig.Templates, "template", []string{}, "Use predefined template(s)")
	cmd.Flags().Bool("list-templates", false, "List available middleware templates")

	return cmd
}

// runMiddlewareGeneration executes the middleware generation process.
func runMiddlewareGeneration(ctx context.Context, config *GenerateConfig, middlewareConfig *MiddlewareGenerationConfig) error {
	// Initialize generator command
	gc, err := NewGeneratorCommand(config)
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Execute with common setup
	return gc.Execute(ctx, func(ctx context.Context) error {
		return generateMiddleware(ctx, gc, config, middlewareConfig)
	})
}

// generateMiddleware performs the actual middleware generation.
func generateMiddleware(ctx context.Context, gc *GeneratorCommand, config *GenerateConfig, middlewareConfig *MiddlewareGenerationConfig) error {
	// Prompt for missing configuration
	if err := gc.PromptForMissingConfig(map[string]string{
		"service": "Service name is required for middleware generation",
	}); err != nil {
		return err
	}

	// Prompt for middleware-specific configuration
	if err := promptForMiddlewareConfig(gc, middlewareConfig); err != nil {
		return fmt.Errorf("failed to get middleware configuration: %w", err)
	}

	// Apply template configuration if specified
	if err := applyMiddlewareTemplates(middlewareConfig); err != nil {
		return fmt.Errorf("failed to apply middleware templates: %w", err)
	}

	// Validate and normalize configuration
	if err := validateMiddlewareConfig(config, middlewareConfig); err != nil {
		return fmt.Errorf("invalid middleware configuration: %w", err)
	}

	// Prepare middleware configuration for generation
	genConfig, err := prepareMiddlewareGenerationConfig(config, middlewareConfig)
	if err != nil {
		return fmt.Errorf("failed to prepare generation configuration: %w", err)
	}

	// Get list of files that will be generated
	files := getMiddlewareFilesToGenerate(config, middlewareConfig)

	// Show preview
	description := fmt.Sprintf("Generating middleware '%s' (%s) for service '%s'", middlewareConfig.Name, middlewareConfig.Type, config.Service)
	if err := gc.ShowPreview(files, description); err != nil {
		return err
	}

	// Check for file conflicts
	if err := gc.CheckFileConflicts(files); err != nil {
		return err
	}

	// Generate the middleware
	if !config.DryRun {
		if err := gc.generator.GenerateMiddleware(*genConfig); err != nil {
			return fmt.Errorf("middleware generation failed: %w", err)
		}

		// Show post-generation instructions
		showMiddlewarePostGenerationInstructions(gc, config, middlewareConfig)
	}

	return nil
}

// promptForMiddlewareConfig prompts for missing middleware configuration.
func promptForMiddlewareConfig(gc *GeneratorCommand, middlewareConfig *MiddlewareGenerationConfig) error {
	if !gc.config.Interactive {
		return nil
	}

	// Prompt for middleware name if not provided
	if middlewareConfig.Name == "" {
		name, err := gc.ui.PromptInput("Middleware name", func(input string) error {
			if input == "" {
				return fmt.Errorf("middleware name is required")
			}
			if err := validatePackageName(input); err != nil {
				return fmt.Errorf("middleware name must be lowercase letters and numbers only")
			}
			return nil
		})
		if err != nil {
			return err
		}
		middlewareConfig.Name = name
	}

	// Prompt for middleware type
	if middlewareConfig.Type == "" || middlewareConfig.Type == "both" {
		options := []interfaces.MenuOption{
			{Label: "Both HTTP and gRPC", Value: "both", Icon: "🌐", Description: "Generate middleware for both HTTP and gRPC services"},
			{Label: "HTTP only", Value: "http", Icon: "📡", Description: "Generate only HTTP middleware (Gin HandlerFunc)"},
			{Label: "gRPC only", Value: "grpc", Icon: "⚡", Description: "Generate only gRPC interceptor"},
		}

		choice, err := gc.ui.ShowMenu("Select middleware type:", options)
		if err != nil {
			return err
		}
		if stringValue, ok := options[choice].Value.(string); ok {
			middlewareConfig.Type = stringValue
		} else {
			return fmt.Errorf("invalid middleware type selection")
		}
	}

	// Prompt for description
	if middlewareConfig.Description == "" {
		description, err := gc.ui.PromptInput("Middleware description (optional)", nil)
		if err != nil {
			return err
		}
		middlewareConfig.Description = description
	}

	// Prompt for predefined templates
	if len(middlewareConfig.Templates) == 0 {
		useTemplate, err := gc.ui.PromptConfirm("Use predefined template?", false)
		if err != nil {
			return err
		}

		if useTemplate {
			// Show available templates
			templateOptions := []interfaces.MenuOption{}
			for name, template := range middlewareTemplates {
				if middlewareConfig.Type == "both" || string(template.Type) == middlewareConfig.Type || template.Type == MiddlewareTypeBoth {
					templateOptions = append(templateOptions, interfaces.MenuOption{
						Label:       name,
						Value:       name,
						Description: template.Description,
						Icon:        "📦",
					})
				}
			}

			if len(templateOptions) > 0 {
				choice, err := gc.ui.ShowMenu("Select template:", templateOptions)
				if err != nil {
					return err
				}
				if stringValue, ok := templateOptions[choice].Value.(string); ok {
					middlewareConfig.Templates = []string{stringValue}
				} else {
					return fmt.Errorf("invalid template selection")
				}
			}
		}
	}

	// Prompt for features if no template selected
	if len(middlewareConfig.Templates) == 0 && len(middlewareConfig.Features) == 0 {
		gc.ui.ShowInfo("Define middleware features (press Enter with empty feature to finish):")

		for {
			feature, err := gc.ui.PromptInput("Feature name (auth, logging, metrics, etc.)", nil)
			if err != nil {
				return err
			}
			if feature == "" {
				break
			}
			middlewareConfig.Features = append(middlewareConfig.Features, feature)
		}
	}

	return nil
}

// applyMiddlewareTemplates applies predefined template configurations.
func applyMiddlewareTemplates(middlewareConfig *MiddlewareGenerationConfig) error {
	for _, templateName := range middlewareConfig.Templates {
		template, exists := middlewareTemplates[templateName]
		if !exists {
			return fmt.Errorf("unknown middleware template: %s", templateName)
		}

		// Apply template configuration
		if middlewareConfig.Description == "" {
			middlewareConfig.Description = template.Description
		}

		// Merge features
		for _, feature := range template.Features {
			found := false
			for _, existing := range middlewareConfig.Features {
				if existing == feature {
					found = true
					break
				}
			}
			if !found {
				middlewareConfig.Features = append(middlewareConfig.Features, feature)
			}
		}

		// Merge dependencies
		for _, dep := range template.Dependencies {
			found := false
			for _, existing := range middlewareConfig.Dependencies {
				if existing == dep {
					found = true
					break
				}
			}
			if !found {
				middlewareConfig.Dependencies = append(middlewareConfig.Dependencies, dep)
			}
		}

		// Set type if compatible
		if middlewareConfig.Type == "both" || string(template.Type) == middlewareConfig.Type || template.Type == MiddlewareTypeBoth {
			if template.Type != MiddlewareTypeBoth {
				middlewareConfig.Type = string(template.Type)
			}
		}
	}

	return nil
}

// validateMiddlewareConfig validates the middleware configuration.
func validateMiddlewareConfig(config *GenerateConfig, middlewareConfig *MiddlewareGenerationConfig) error {
	// Validate middleware name
	if middlewareConfig.Name == "" {
		return fmt.Errorf("middleware name is required")
	}

	if err := validatePackageName(middlewareConfig.Name); err != nil {
		return fmt.Errorf("invalid middleware name: %w", err)
	}

	// Validate middleware type
	validTypes := []string{"http", "grpc", "both"}
	isValidType := false
	for _, validType := range validTypes {
		if middlewareConfig.Type == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return fmt.Errorf("invalid middleware type: %s (must be one of: %s)", middlewareConfig.Type, strings.Join(validTypes, ", "))
	}

	// Set default description
	if middlewareConfig.Description == "" {
		middlewareConfig.Description = fmt.Sprintf("%s middleware for %s", toPascalCase(middlewareConfig.Name), config.Service)
	}

	// Initialize config map if nil
	if middlewareConfig.Config == nil {
		middlewareConfig.Config = make(map[string]interface{})
	}

	return nil
}

// prepareMiddlewareGenerationConfig converts the command configuration to generator configuration.
func prepareMiddlewareGenerationConfig(config *GenerateConfig, middlewareConfig *MiddlewareGenerationConfig) (*interfaces.MiddlewareConfig, error) {
	return &interfaces.MiddlewareConfig{
		Name:         middlewareConfig.Name,
		Type:         middlewareConfig.Type,
		Description:  middlewareConfig.Description,
		Package:      "middleware",
		Features:     middlewareConfig.Features,
		Dependencies: middlewareConfig.Dependencies,
		Config:       middlewareConfig.Config,
		Metadata: map[string]string{
			"service":    config.Service,
			"package":    config.Package,
			"output_dir": config.OutputDir,
			"templates":  strings.Join(middlewareConfig.Templates, ","),
		},
	}, nil
}

// getMiddlewareFilesToGenerate returns the list of files that will be generated.
func getMiddlewareFilesToGenerate(config *GenerateConfig, middlewareConfig *MiddlewareGenerationConfig) []string {
	files := []string{}

	baseDir := config.OutputDir
	if config.Service != "" {
		baseDir = filepath.Join(baseDir, "internal", config.Service)
	}

	middlewareDir := filepath.Join(baseDir, "middleware")

	// Main middleware file
	files = append(files,
		filepath.Join(middlewareDir, fmt.Sprintf("%s.go", middlewareConfig.Name)),
		filepath.Join(middlewareDir, fmt.Sprintf("%s_test.go", middlewareConfig.Name)),
	)

	// Configuration file if middleware has config
	if len(middlewareConfig.Config) > 0 || len(middlewareConfig.Features) > 0 {
		files = append(files,
			filepath.Join(middlewareDir, fmt.Sprintf("%s_config.go", middlewareConfig.Name)),
		)
	}

	// Example usage file
	files = append(files,
		filepath.Join(middlewareDir, "examples", fmt.Sprintf("%s_example.go", middlewareConfig.Name)),
	)

	return files
}

// showMiddlewarePostGenerationInstructions shows instructions after successful generation.
func showMiddlewarePostGenerationInstructions(gc *GeneratorCommand, config *GenerateConfig, middlewareConfig *MiddlewareGenerationConfig) {
	gc.ui.PrintHeader("Next Steps")

	instructions := []string{
		fmt.Sprintf("Middleware '%s' has been generated successfully!", middlewareConfig.Name),
		"",
		"To complete the setup:",
		"1. Review and customize the generated middleware logic",
		"2. Configure middleware options in the config file",
		"3. Add the middleware to your service registration",
	}

	// Type-specific instructions
	switch middlewareConfig.Type {
	case "http":
		instructions = append(instructions,
			"4. Register HTTP middleware in your Gin router setup",
			"5. Test HTTP endpoints with the middleware enabled",
		)
	case "grpc":
		instructions = append(instructions,
			"4. Register gRPC interceptor in your server options",
			"5. Test gRPC methods with the interceptor enabled",
		)
	case "both":
		instructions = append(instructions,
			"4. Register HTTP middleware in your Gin router setup",
			"5. Register gRPC interceptor in your server options",
			"6. Test both HTTP and gRPC endpoints",
		)
	}

	instructions = append(instructions,
		"7. Write comprehensive tests for all middleware scenarios",
		"8. Update service documentation to describe middleware behavior",
	)

	// Dependencies instructions
	if len(middlewareConfig.Dependencies) > 0 {
		instructions = append(instructions,
			"9. Run 'go mod tidy' to download new dependencies:",
		)
		for _, dep := range middlewareConfig.Dependencies {
			instructions = append(instructions, fmt.Sprintf("   - %s", dep))
		}
	}

	for _, instruction := range instructions {
		if instruction == "" {
			fmt.Println()
		} else {
			fmt.Printf("  %s\n", instruction)
		}
	}

	gc.ui.PrintSeparator()

	// Show usage example
	gc.ui.PrintSubHeader("Usage Example")
	showMiddlewareUsageExample(gc, config, middlewareConfig)

	gc.ui.PrintSeparator()
	gc.ui.ShowInfo(fmt.Sprintf("Generated files are located in: %s", config.OutputDir))
}

// showMiddlewareUsageExample shows a usage example for the generated middleware.
func showMiddlewareUsageExample(gc *GeneratorCommand, config *GenerateConfig, middlewareConfig *MiddlewareGenerationConfig) {
	middlewareName := toPascalCase(middlewareConfig.Name)

	switch middlewareConfig.Type {
	case "http":
		fmt.Printf(`// HTTP usage in your service registration:
router.Use(middleware.%s())
`, middlewareName)

	case "grpc":
		fmt.Printf(`// gRPC usage in your server options:
grpc.UnaryInterceptor(middleware.%sInterceptor())
`, middlewareName)

	case "both":
		fmt.Printf(`// HTTP usage in your service registration:
router.Use(middleware.%s())

// gRPC usage in your server options:
grpc.UnaryInterceptor(middleware.%sInterceptor())
`, middlewareName, middlewareName)
	}
}

// showMiddlewareTemplates shows available middleware templates.
func showMiddlewareTemplates() error {
	fmt.Println("Available middleware templates:")

	for name, template := range middlewareTemplates {
		fmt.Printf("  %s (%s)\n", name, template.Type)
		fmt.Printf("    %s\n", template.Description)
		if len(template.Features) > 0 {
			fmt.Printf("    Features: %s\n", strings.Join(template.Features, ", "))
		}
		if len(template.Dependencies) > 0 {
			fmt.Printf("    Dependencies: %s\n", strings.Join(template.Dependencies, ", "))
		}
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  switctl generate middleware <name> --template=<template_name>")
	fmt.Println("  switctl generate middleware auth --template=auth")

	return nil
}
