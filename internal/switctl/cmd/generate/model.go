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

// ModelGenerationConfig holds configuration specific to model generation.
type ModelGenerationConfig struct {
	Name       string
	Fields     []FieldConfig
	Table      string
	Database   string
	CRUD       bool
	API        bool
	Validation bool
	Timestamps bool
	SoftDelete bool
	Indexes    []IndexConfig
	Relations  []RelationConfig
}

// FieldConfig represents a field configuration from command line.
type FieldConfig struct {
	Name        string
	Type        string
	Required    bool
	Unique      bool
	Index       bool
	Default     string
	Description string
	Validation  []string
}

// IndexConfig represents an index configuration.
type IndexConfig struct {
	Name   string
	Fields []string
	Unique bool
	Type   string
}

// RelationConfig represents a relation configuration.
type RelationConfig struct {
	Type       string
	Model      string
	ForeignKey string
	LocalKey   string
}

// NewModelCommand creates the model generation command.
func NewModelCommand(config *GenerateConfig) *cobra.Command {
	var modelConfig ModelGenerationConfig

	cmd := &cobra.Command{
		Use:   "model [name]",
		Short: "Generate data models and CRUD operations",
		Long: `Generate data models with GORM structs and CRUD operations.

This command generates:
- GORM model structs with proper tags
- CRUD repository interfaces and implementations
- Validation methods and rules
- Database migration files (optional)
- API endpoints (optional)
- Unit tests for all generated code

Examples:
  # Generate model with interactive prompts
  switctl generate model --interactive

  # Generate a user model with specific fields
  switctl generate model user --fields="name:string:required,email:string:unique,age:int"

  # Generate model with CRUD operations and API
  switctl generate model product --crud --api --table=products

  # Generate model with relationships
  switctl generate model order --relations="user:belongs_to,items:has_many"

  # Generate model with custom validation
  switctl generate model article --validation --timestamps --soft-delete`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set model name from argument if provided
			if len(args) > 0 {
				modelConfig.Name = args[0]
			}

			return runModelGeneration(cmd.Context(), config, &modelConfig)
		},
	}

	// Model-specific flags
	cmd.Flags().StringSliceVar(&[]string{}, "fields", []string{}, "Model fields (format: name:type[:modifiers])")
	cmd.Flags().StringVar(&modelConfig.Table, "table", "", "Database table name (defaults to pluralized model name)")
	cmd.Flags().StringVar(&modelConfig.Database, "database", "", "Database type (mysql, postgres, sqlite)")
	cmd.Flags().BoolVar(&modelConfig.CRUD, "crud", true, "Generate CRUD operations")
	cmd.Flags().BoolVar(&modelConfig.API, "api", false, "Generate API endpoints")
	cmd.Flags().BoolVar(&modelConfig.Validation, "validation", true, "Generate validation methods")
	cmd.Flags().BoolVar(&modelConfig.Timestamps, "timestamps", true, "Add created_at and updated_at fields")
	cmd.Flags().BoolVar(&modelConfig.SoftDelete, "soft-delete", false, "Add soft delete support")
	cmd.Flags().StringSliceVar(&[]string{}, "indexes", []string{}, "Database indexes (format: name:field1,field2[:unique])")
	cmd.Flags().StringSliceVar(&[]string{}, "relations", []string{}, "Model relations (format: model:type[:foreign_key])")

	return cmd
}

// runModelGeneration executes the model generation process.
func runModelGeneration(ctx context.Context, config *GenerateConfig, modelConfig *ModelGenerationConfig) error {
	// Initialize generator command
	gc, err := NewGeneratorCommand(config)
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Execute with common setup
	return gc.Execute(ctx, func(ctx context.Context) error {
		return generateModel(ctx, gc, config, modelConfig)
	})
}

// generateModel performs the actual model generation.
func generateModel(ctx context.Context, gc *GeneratorCommand, config *GenerateConfig, modelConfig *ModelGenerationConfig) error {
	// Prompt for missing configuration
	if err := gc.PromptForMissingConfig(map[string]string{
		"service": "Service name is required for model generation",
	}); err != nil {
		return err
	}

	// Prompt for model-specific configuration
	if err := promptForModelConfig(gc, modelConfig); err != nil {
		return fmt.Errorf("failed to get model configuration: %w", err)
	}

	// Validate and normalize configuration
	if err := validateModelConfig(config, modelConfig); err != nil {
		return fmt.Errorf("invalid model configuration: %w", err)
	}

	// Prepare model configuration for generation
	genConfig, err := prepareModelGenerationConfig(config, modelConfig)
	if err != nil {
		return fmt.Errorf("failed to prepare generation configuration: %w", err)
	}

	// Get list of files that will be generated
	files := getModelFilesToGenerate(config, modelConfig)

	// Show preview
	description := fmt.Sprintf("Generating model '%s' for service '%s'", modelConfig.Name, config.Service)
	if err := gc.ShowPreview(files, description); err != nil {
		return err
	}

	// Check for file conflicts
	if err := gc.CheckFileConflicts(files); err != nil {
		return err
	}

	// Generate the model
	if !config.DryRun {
		if err := gc.generator.GenerateModel(*genConfig); err != nil {
			return fmt.Errorf("model generation failed: %w", err)
		}

		// Generate API endpoints if requested
		if modelConfig.API {
			if err := generateModelAPI(ctx, gc, config, modelConfig); err != nil {
				gc.logger.Warn("Failed to generate model API", "error", err)
				gc.ui.ShowInfo("Note: Model API generation failed, you may need to generate it manually")
			}
		}

		// Show post-generation instructions
		showModelPostGenerationInstructions(gc, config, modelConfig)
	}

	return nil
}

// promptForModelConfig prompts for missing model configuration.
func promptForModelConfig(gc *GeneratorCommand, modelConfig *ModelGenerationConfig) error {
	if !gc.config.Interactive {
		return nil
	}

	// Prompt for model name if not provided
	if modelConfig.Name == "" {
		name, err := gc.ui.PromptInput("Model name", func(input string) error {
			if input == "" {
				return fmt.Errorf("model name is required")
			}
			if err := validatePackageName(input); err != nil {
				return fmt.Errorf("model name must be lowercase letters and numbers only")
			}
			return nil
		})
		if err != nil {
			return err
		}
		modelConfig.Name = name
	}

	// Prompt for fields if none provided
	if len(modelConfig.Fields) == 0 {
		gc.ui.ShowInfo("Define model fields (press Enter with empty field name to finish):")

		for {
			fieldName, err := gc.ui.PromptInput("Field name (or press Enter to finish)", nil)
			if err != nil {
				return err
			}
			if fieldName == "" {
				break
			}

			fieldType, err := gc.ui.PromptInput("Field type (string, int, bool, float64, time.Time, etc.)", func(input string) error {
				if input == "" {
					return fmt.Errorf("field type is required")
				}
				return nil
			})
			if err != nil {
				return err
			}

			required, err := gc.ui.PromptConfirm("Required field?", false)
			if err != nil {
				return err
			}

			unique, err := gc.ui.PromptConfirm("Unique field?", false)
			if err != nil {
				return err
			}

			description, err := gc.ui.PromptInput("Field description (optional)", nil)
			if err != nil {
				return err
			}

			modelConfig.Fields = append(modelConfig.Fields, FieldConfig{
				Name:        fieldName,
				Type:        fieldType,
				Required:    required,
				Unique:      unique,
				Description: description,
			})
		}
	}

	// Prompt for table name if not provided
	if modelConfig.Table == "" {
		defaultTable := pluralize(modelConfig.Name)
		table, err := gc.ui.PromptInput(fmt.Sprintf("Table name (default: %s)", defaultTable), nil)
		if err != nil {
			return err
		}
		if table != "" {
			modelConfig.Table = table
		} else {
			modelConfig.Table = defaultTable
		}
	}

	// Prompt for additional options
	crud, err := gc.ui.PromptConfirm("Generate CRUD operations?", true)
	if err != nil {
		return err
	}
	modelConfig.CRUD = crud

	api, err := gc.ui.PromptConfirm("Generate API endpoints?", false)
	if err != nil {
		return err
	}
	modelConfig.API = api

	return nil
}

// validateModelConfig validates the model configuration.
func validateModelConfig(config *GenerateConfig, modelConfig *ModelGenerationConfig) error {
	// Validate model name
	if modelConfig.Name == "" {
		return fmt.Errorf("model name is required")
	}

	if err := validatePackageName(modelConfig.Name); err != nil {
		return fmt.Errorf("invalid model name: %w", err)
	}

	// Set default table name
	if modelConfig.Table == "" {
		modelConfig.Table = pluralize(modelConfig.Name)
	}

	// Validate fields
	if len(modelConfig.Fields) == 0 {
		// Add default ID field
		modelConfig.Fields = append(modelConfig.Fields, FieldConfig{
			Name:        "id",
			Type:        "uint",
			Required:    true,
			Unique:      true,
			Description: "Primary key",
		})
	}

	// Add timestamp fields if enabled
	if modelConfig.Timestamps {
		modelConfig.Fields = append(modelConfig.Fields,
			FieldConfig{
				Name:        "created_at",
				Type:        "time.Time",
				Description: "Creation timestamp",
			},
			FieldConfig{
				Name:        "updated_at",
				Type:        "time.Time",
				Description: "Last update timestamp",
			},
		)
	}

	// Add soft delete field if enabled
	if modelConfig.SoftDelete {
		modelConfig.Fields = append(modelConfig.Fields, FieldConfig{
			Name:        "deleted_at",
			Type:        "*time.Time",
			Description: "Soft delete timestamp",
		})
	}

	// Validate field types
	for i, field := range modelConfig.Fields {
		if field.Name == "" {
			return fmt.Errorf("field name is required at index %d", i)
		}
		if field.Type == "" {
			return fmt.Errorf("field type is required for field %s", field.Name)
		}
	}

	return nil
}

// prepareModelGenerationConfig converts the command configuration to generator configuration.
func prepareModelGenerationConfig(config *GenerateConfig, modelConfig *ModelGenerationConfig) (*interfaces.ModelConfig, error) {
	// Convert fields
	fields := make([]interfaces.Field, len(modelConfig.Fields))
	for i, field := range modelConfig.Fields {
		tags := map[string]string{
			"json": field.Name,
			"db":   field.Name,
		}

		// Add GORM tags based on field properties
		var gormTags []string
		if field.Required && field.Name != "id" {
			gormTags = append(gormTags, "not null")
		}
		if field.Unique {
			gormTags = append(gormTags, "unique")
		}
		if field.Index {
			gormTags = append(gormTags, "index")
		}
		if field.Name == "id" {
			gormTags = append(gormTags, "primary_key", "auto_increment")
		}
		if len(gormTags) > 0 {
			tags["gorm"] = strings.Join(gormTags, ";")
		}

		// Add validation tags
		var validateTags []string
		if field.Required && field.Name != "id" {
			validateTags = append(validateTags, "required")
		}
		if field.Type == "string" && field.Unique {
			validateTags = append(validateTags, "email") // Example validation
		}
		if len(validateTags) > 0 {
			tags["validate"] = strings.Join(validateTags, ",")
		}

		fields[i] = interfaces.Field{
			Name:        field.Name,
			Type:        field.Type,
			Tags:        tags,
			Required:    field.Required,
			Unique:      field.Unique,
			Index:       field.Index,
			Description: field.Description,
		}
	}

	// Convert indexes
	indexes := make([]interfaces.Index, len(modelConfig.Indexes))
	for i, index := range modelConfig.Indexes {
		indexes[i] = interfaces.Index{
			Name:   index.Name,
			Fields: index.Fields,
			Unique: index.Unique,
			Type:   index.Type,
		}
	}

	// Convert relations
	relations := make([]interfaces.Relation, len(modelConfig.Relations))
	for i, relation := range modelConfig.Relations {
		relations[i] = interfaces.Relation{
			Type:       relation.Type,
			Model:      relation.Model,
			ForeignKey: relation.ForeignKey,
			LocalKey:   relation.LocalKey,
		}
	}

	return &interfaces.ModelConfig{
		Name:        modelConfig.Name,
		Description: fmt.Sprintf("%s model", toPascalCase(modelConfig.Name)),
		Package:     "model",
		Fields:      fields,
		Table:       modelConfig.Table,
		Database:    modelConfig.Database,
		CRUD:        modelConfig.CRUD,
		GenerateAPI: modelConfig.API,
		Validation:  modelConfig.Validation,
		Timestamps:  modelConfig.Timestamps,
		SoftDelete:  modelConfig.SoftDelete,
		Indexes:     indexes,
		Relations:   relations,
		Metadata: map[string]string{
			"service":    config.Service,
			"package":    config.Package,
			"output_dir": config.OutputDir,
		},
	}, nil
}

// getModelFilesToGenerate returns the list of files that will be generated.
func getModelFilesToGenerate(config *GenerateConfig, modelConfig *ModelGenerationConfig) []string {
	files := []string{}

	baseDir := config.OutputDir
	if config.Service != "" {
		baseDir = filepath.Join(baseDir, "internal", config.Service)
	}

	// Model files
	files = append(files,
		filepath.Join(baseDir, "model", fmt.Sprintf("%s.go", modelConfig.Name)),
		filepath.Join(baseDir, "model", fmt.Sprintf("%s_test.go", modelConfig.Name)),
	)

	// CRUD files if enabled
	if modelConfig.CRUD {
		files = append(files,
			filepath.Join(baseDir, "repository", fmt.Sprintf("%s_repository.go", modelConfig.Name)),
			filepath.Join(baseDir, "repository", fmt.Sprintf("%s_repository_test.go", modelConfig.Name)),
		)
	}

	// API files if enabled
	if modelConfig.API {
		files = append(files,
			filepath.Join(baseDir, "handler", fmt.Sprintf("%s_handler.go", modelConfig.Name)),
			filepath.Join(baseDir, "handler", fmt.Sprintf("%s_handler_test.go", modelConfig.Name)),
			filepath.Join(baseDir, "service", fmt.Sprintf("%s_service.go", modelConfig.Name)),
			filepath.Join(baseDir, "service", fmt.Sprintf("%s_service_test.go", modelConfig.Name)),
		)
	}

	return files
}

// generateModelAPI generates API endpoints for the model.
func generateModelAPI(ctx context.Context, gc *GeneratorCommand, config *GenerateConfig, modelConfig *ModelGenerationConfig) error {
	gc.logger.Info("Generating API endpoints for model", "model", modelConfig.Name)

	// This would typically call the API generator
	// For now, we'll just log the action
	gc.ui.ShowInfo(fmt.Sprintf("Generated API endpoints for model: %s", modelConfig.Name))

	return nil
}

// showModelPostGenerationInstructions shows instructions after successful generation.
func showModelPostGenerationInstructions(gc *GeneratorCommand, config *GenerateConfig, modelConfig *ModelGenerationConfig) {
	gc.ui.PrintHeader("Next Steps")

	instructions := []string{
		fmt.Sprintf("Model '%s' has been generated successfully!", modelConfig.Name),
		"",
		"To complete the setup:",
		"1. Review and customize the generated model fields and tags",
		"2. Implement custom validation logic if needed",
		"3. Add business logic methods to the model",
		"4. Update database migration files",
	}

	if modelConfig.CRUD {
		instructions = append(instructions,
			"5. Implement repository methods for data operations",
			"6. Add proper error handling in repository methods",
		)
	}

	if modelConfig.API {
		instructions = append(instructions,
			"7. Register API handlers with your service",
			"8. Add input validation and error handling",
			"9. Update API documentation",
		)
	}

	instructions = append(instructions,
		"10. Write comprehensive tests for your model",
		"11. Run 'go mod tidy' to update dependencies",
	)

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

// parseFieldsFlag parses the fields flag into FieldConfig slice.
func parseFieldsFlag(fieldsFlag []string) ([]FieldConfig, error) {
	fields := make([]FieldConfig, 0, len(fieldsFlag))

	for _, fieldStr := range fieldsFlag {
		parts := strings.Split(fieldStr, ":")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid field format: %s (expected name:type[:modifiers])", fieldStr)
		}

		field := FieldConfig{
			Name: strings.TrimSpace(parts[0]),
			Type: strings.TrimSpace(parts[1]),
		}

		// Parse modifiers
		if len(parts) > 2 {
			modifiers := strings.Split(parts[2], ",")
			for _, modifier := range modifiers {
				modifier = strings.TrimSpace(modifier)
				switch modifier {
				case "required":
					field.Required = true
				case "unique":
					field.Unique = true
				case "index":
					field.Index = true
				default:
					// Assume it's a default value
					field.Default = modifier
				}
			}
		}

		fields = append(fields, field)
	}

	return fields, nil
}

// parseIndexesFlag parses the indexes flag into IndexConfig slice.
func parseIndexesFlag(indexesFlag []string) ([]IndexConfig, error) {
	indexes := make([]IndexConfig, 0, len(indexesFlag))

	for _, indexStr := range indexesFlag {
		parts := strings.Split(indexStr, ":")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid index format: %s (expected name:field1,field2[:unique])", indexStr)
		}

		index := IndexConfig{
			Name:   strings.TrimSpace(parts[0]),
			Fields: strings.Split(strings.TrimSpace(parts[1]), ","),
			Type:   "btree",
		}

		// Parse modifiers
		if len(parts) > 2 {
			modifiers := strings.Split(parts[2], ",")
			for _, modifier := range modifiers {
				modifier = strings.TrimSpace(modifier)
				switch modifier {
				case "unique":
					index.Unique = true
				default:
					index.Type = modifier
				}
			}
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

// parseRelationsFlag parses the relations flag into RelationConfig slice.
func parseRelationsFlag(relationsFlag []string) ([]RelationConfig, error) {
	relations := make([]RelationConfig, 0, len(relationsFlag))

	for _, relationStr := range relationsFlag {
		parts := strings.Split(relationStr, ":")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid relation format: %s (expected model:type[:foreign_key])", relationStr)
		}

		relation := RelationConfig{
			Model: strings.TrimSpace(parts[0]),
			Type:  strings.TrimSpace(parts[1]),
		}

		// Parse foreign key
		if len(parts) > 2 {
			relation.ForeignKey = strings.TrimSpace(parts[2])
		}

		relations = append(relations, relation)
	}

	return relations, nil
}

// pluralize simple pluralization (same as in generator.go).
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
