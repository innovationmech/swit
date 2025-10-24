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

package dsl

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

var (
	// envVarPattern matches environment variable references like ${VAR_NAME} or $VAR_NAME
	envVarPattern = regexp.MustCompile(`\$\{([A-Z0-9_]+)\}|\$([A-Z0-9_]+)`)

	// includePattern matches include directives like !include path/to/file.yaml
	includePattern = regexp.MustCompile(`^!\s*include\s+(.+)$`)
)

// Logger defines the interface for logging operations.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// defaultLogger implements Logger using standard log package.
type defaultLogger struct {
	logger *log.Logger
}

func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	l.logger.Printf("[DEBUG] "+msg, args...)
}

func (l *defaultLogger) Info(msg string, args ...interface{}) {
	l.logger.Printf("[INFO] "+msg, args...)
}

func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	l.logger.Printf("[WARN] "+msg, args...)
}

func (l *defaultLogger) Error(msg string, args ...interface{}) {
	l.logger.Printf("[ERROR] "+msg, args...)
}

// Parser handles parsing of Saga DSL YAML files.
type Parser struct {
	validator *validator.Validate
	logger    Logger

	// options
	enableEnvVars    bool
	enableIncludes   bool
	maxIncludeDepth  int
	basePath         string
	visitedFiles     map[string]bool // track visited files for circular dependency detection
	currentDepth     int
}

// ParserOption configures the Parser.
type ParserOption func(*Parser)

// WithEnvVars enables environment variable substitution.
func WithEnvVars(enable bool) ParserOption {
	return func(p *Parser) {
		p.enableEnvVars = enable
	}
}

// WithIncludes enables file include functionality.
func WithIncludes(enable bool) ParserOption {
	return func(p *Parser) {
		p.enableIncludes = enable
	}
}

// WithMaxIncludeDepth sets the maximum include depth to prevent infinite recursion.
func WithMaxIncludeDepth(depth int) ParserOption {
	return func(p *Parser) {
		p.maxIncludeDepth = depth
	}
}

// WithBasePath sets the base path for resolving relative file paths.
func WithBasePath(path string) ParserOption {
	return func(p *Parser) {
		p.basePath = path
	}
}

// WithLogger sets the logger for the parser.
func WithLogger(log Logger) ParserOption {
	return func(p *Parser) {
		p.logger = log
	}
}

// NewParser creates a new DSL parser with the given options.
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{
		validator:       validator.New(validator.WithRequiredStructEnabled()),
		logger:          &defaultLogger{logger: log.New(os.Stdout, "", log.LstdFlags)},
		enableEnvVars:   true,
		enableIncludes:  true,
		maxIncludeDepth: 10,
		basePath:        ".",
		visitedFiles:    make(map[string]bool),
		currentDepth:    0,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// ParseFile parses a Saga definition from a YAML file.
func ParseFile(path string, opts ...ParserOption) (*SagaDefinition, error) {
	parser := NewParser(opts...)
	return parser.ParseFile(path)
}

// ParseBytes parses a Saga definition from YAML bytes.
func ParseBytes(data []byte, opts ...ParserOption) (*SagaDefinition, error) {
	parser := NewParser(opts...)
	return parser.ParseBytes(data)
}

// ParseReader parses a Saga definition from an io.Reader.
func ParseReader(r io.Reader, opts ...ParserOption) (*SagaDefinition, error) {
	parser := NewParser(opts...)
	return parser.ParseReader(r)
}

// ParseFile parses a Saga definition from a YAML file.
func (p *Parser) ParseFile(path string) (*SagaDefinition, error) {
	p.logger.Debug("parsing Saga DSL file", "path", path)

	// Resolve absolute path
	absPath, err := p.resolveFilePath(path)
	if err != nil {
		return nil, &ParseError{
			Message: fmt.Sprintf("failed to resolve file path: %v", err),
			File:    path,
		}
	}

	// Check for circular dependencies
	if p.visitedFiles[absPath] {
		return nil, &ParseError{
			Message: "circular dependency detected",
			File:    absPath,
		}
	}
	p.visitedFiles[absPath] = true

	// Check include depth
	if p.currentDepth > p.maxIncludeDepth {
		return nil, &ParseError{
			Message: fmt.Sprintf("maximum include depth (%d) exceeded", p.maxIncludeDepth),
			File:    absPath,
		}
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, &ParseError{
			Message: "file not found",
			File:    absPath,
		}
	}

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, &ParseError{
			Message: fmt.Sprintf("failed to read file: %v", err),
			File:    absPath,
		}
	}

	p.logger.Debug("successfully read file", "path", absPath, "size", len(data))

	// Update base path for relative includes
	oldBasePath := p.basePath
	p.basePath = filepath.Dir(absPath)
	defer func() {
		p.basePath = oldBasePath
	}()

	return p.parseBytes(data, absPath)
}

// ParseBytes parses a Saga definition from YAML bytes.
func (p *Parser) ParseBytes(data []byte) (*SagaDefinition, error) {
	return p.parseBytes(data, "<bytes>")
}

// ParseReader parses a Saga definition from an io.Reader.
func (p *Parser) ParseReader(r io.Reader) (*SagaDefinition, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, &ParseError{
			Message: fmt.Sprintf("failed to read from reader: %v", err),
			File:    "<reader>",
		}
	}

	return p.parseBytes(data, "<reader>")
}

// parseBytes is the internal implementation for parsing YAML bytes.
func (p *Parser) parseBytes(data []byte, source string) (*SagaDefinition, error) {
	if len(data) == 0 {
		return nil, &ParseError{
			Message: "empty YAML content",
			File:    source,
		}
	}

	// Process includes if enabled
	if p.enableIncludes {
		var err error
		data, err = p.processIncludes(data, source)
		if err != nil {
			return nil, err
		}
	}

	// Expand environment variables if enabled
	if p.enableEnvVars {
		data = p.expandEnvVars(data)
	}

	// Unmarshal YAML
	var def SagaDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, &ParseError{
			Message: fmt.Sprintf("YAML syntax error: %v", err),
			File:    source,
			Cause:   err,
		}
	}

	p.logger.Debug("successfully parsed YAML", "source", source, "saga_id", def.Saga.ID)

	// Apply default values
	p.applyDefaults(&def)

	// Validate the definition
	if err := p.validate(&def); err != nil {
		return nil, &ParseError{
			Message: fmt.Sprintf("validation failed: %v", err),
			File:    source,
			Cause:   err,
		}
	}

	p.logger.Info("successfully parsed and validated Saga definition",
		"source", source,
		"saga_id", def.Saga.ID,
		"saga_name", def.Saga.Name,
		"steps", len(def.Steps))

	return &def, nil
}

// resolveFilePath resolves a file path relative to the base path.
func (p *Parser) resolveFilePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	absPath := filepath.Join(p.basePath, path)
	return filepath.Clean(absPath), nil
}

// expandEnvVars expands environment variables in the YAML content.
func (p *Parser) expandEnvVars(data []byte) []byte {
	content := string(data)

	// Find and replace all environment variable references
	result := envVarPattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract variable name from ${VAR} or $VAR
		matches := envVarPattern.FindStringSubmatch(match)
		var varName string
		if matches[1] != "" {
			varName = matches[1] // ${VAR} format
		} else {
			varName = matches[2] // $VAR format
		}

		// Get environment variable value
		if value, exists := os.LookupEnv(varName); exists {
			p.logger.Debug("expanded environment variable", "var", varName, "value", value)
			return value
		}

		// If variable doesn't exist, keep the original placeholder
		p.logger.Warn("environment variable not found, keeping placeholder", "var", varName)
		return match
	})

	return []byte(result)
}

// processIncludes processes include directives in the YAML content.
func (p *Parser) processIncludes(data []byte, source string) ([]byte, error) {
	content := string(data)
	lines := strings.Split(content, "\n")
	var result []string

	p.currentDepth++
	defer func() {
		p.currentDepth--
	}()

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for include directive
		if matches := includePattern.FindStringSubmatch(trimmed); len(matches) > 1 {
			includePath := strings.TrimSpace(matches[1])

			p.logger.Debug("processing include directive", "path", includePath, "line", i+1)

			// Parse included file
			includedDef, err := p.ParseFile(includePath)
			if err != nil {
				return nil, &ParseError{
					Message: fmt.Sprintf("failed to process include at line %d: %v", i+1, err),
					File:    source,
					Line:    i + 1,
					Cause:   err,
				}
			}

			// Convert included definition back to YAML
			includedYAML, err := yaml.Marshal(includedDef)
			if err != nil {
				return nil, &ParseError{
					Message: fmt.Sprintf("failed to marshal included content: %v", err),
					File:    source,
					Line:    i + 1,
					Cause:   err,
				}
			}

			// Add included content
			result = append(result, string(includedYAML))
		} else {
			result = append(result, line)
		}
	}

	return []byte(strings.Join(result, "\n")), nil
}

// applyDefaults applies default values to the Saga definition.
func (p *Parser) applyDefaults(def *SagaDefinition) {
	// Apply default execution mode
	if def.Saga.Mode == "" {
		def.Saga.Mode = ExecutionModeOrchestration
		p.logger.Debug("applied default execution mode", "mode", ExecutionModeOrchestration)
	}

	// Apply global retry policy to steps that don't have one
	if def.GlobalRetryPolicy != nil {
		for i := range def.Steps {
			if def.Steps[i].RetryPolicy == nil {
				def.Steps[i].RetryPolicy = def.GlobalRetryPolicy
				p.logger.Debug("applied global retry policy to step", "step_id", def.Steps[i].ID)
			}
		}
	}

	// Apply default compensation strategy
	if def.GlobalCompensation != nil {
		for i := range def.Steps {
			if def.Steps[i].Compensation != nil && def.Steps[i].Compensation.Strategy == "" {
				def.Steps[i].Compensation.Strategy = def.GlobalCompensation.Strategy
			}
		}
	}
}

// validate validates the Saga definition using struct tags and custom rules.
func (p *Parser) validate(def *SagaDefinition) error {
	// Validate using struct tags
	if err := p.validator.Struct(def); err != nil {
		return p.formatValidationError(err)
	}

	// Custom validation rules
	if err := p.validateCustomRules(def); err != nil {
		return err
	}

	return nil
}

// formatValidationError formats a validator error into a more readable form.
func (p *Parser) formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, e := range validationErrors {
			msg := fmt.Sprintf("field '%s' validation failed on '%s' tag", e.Field(), e.Tag())
			if e.Param() != "" {
				msg += fmt.Sprintf(" (param: %s)", e.Param())
			}
			messages = append(messages, msg)
		}
		return fmt.Errorf("validation errors: %s", strings.Join(messages, "; "))
	}
	return err
}

// validateCustomRules validates custom business rules.
func (p *Parser) validateCustomRules(def *SagaDefinition) error {
	// Validate step IDs are unique
	stepIDs := make(map[string]bool)
	for _, step := range def.Steps {
		if stepIDs[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		stepIDs[step.ID] = true
	}

	// Validate dependencies reference existing steps
	for _, step := range def.Steps {
		for _, depID := range step.Dependencies {
			if !stepIDs[depID] {
				return fmt.Errorf("step '%s' depends on non-existent step '%s'", step.ID, depID)
			}
		}
	}

	// Validate no circular dependencies
	if err := p.validateNoCyclicDependencies(def); err != nil {
		return err
	}

	// Validate action configuration based on step type
	for _, step := range def.Steps {
		if err := p.validateStepAction(&step); err != nil {
			return fmt.Errorf("step '%s': %w", step.ID, err)
		}
	}

	// Validate compensation configuration
	for _, step := range def.Steps {
		if err := p.validateCompensation(&step); err != nil {
			return fmt.Errorf("step '%s': %w", step.ID, err)
		}
	}

	return nil
}

// validateNoCyclicDependencies checks for circular dependencies between steps.
func (p *Parser) validateNoCyclicDependencies(def *SagaDefinition) error {
	// Build dependency graph
	graph := make(map[string][]string)
	for _, step := range def.Steps {
		graph[step.ID] = step.Dependencies
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(stepID string) bool {
		visited[stepID] = true
		recStack[stepID] = true

		for _, dep := range graph[stepID] {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[stepID] = false
		return false
	}

	for stepID := range graph {
		if !visited[stepID] {
			if hasCycle(stepID) {
				return fmt.Errorf("circular dependency detected in step dependencies involving step '%s'", stepID)
			}
		}
	}

	return nil
}

// validateStepAction validates that the action configuration matches the step type.
func (p *Parser) validateStepAction(step *StepConfig) error {
	switch step.Type {
	case StepTypeService, StepTypeHTTP, StepTypeGRPC:
		if step.Action.Service == nil {
			return fmt.Errorf("service action is required for step type '%s'", step.Type)
		}
	case StepTypeFunction:
		if step.Action.Function == nil {
			return fmt.Errorf("function action is required for step type '%s'", step.Type)
		}
	case StepTypeMessage:
		if step.Action.Message == nil {
			return fmt.Errorf("message action is required for step type '%s'", step.Type)
		}
	case StepTypeCustom:
		if len(step.Action.Custom) == 0 {
			return fmt.Errorf("custom action is required for step type '%s'", step.Type)
		}
	default:
		return fmt.Errorf("invalid step type: %s", step.Type)
	}
	return nil
}

// validateCompensation validates compensation configuration.
func (p *Parser) validateCompensation(step *StepConfig) error {
	if step.Compensation == nil {
		return nil
	}

	// Validate that custom compensation has an action
	if step.Compensation.Type == CompensationTypeCustom && step.Compensation.Action == nil {
		return fmt.Errorf("custom compensation requires an action")
	}

	// Validate that skip compensation doesn't have an action
	if step.Compensation.Type == CompensationTypeSkip && step.Compensation.Action != nil {
		p.logger.Warn("skip compensation has an action defined, it will be ignored", "step_id", step.ID)
	}

	return nil
}

// ParseError represents an error that occurred during parsing.
type ParseError struct {
	Message string
	File    string
	Line    int
	Column  int
	Cause   error
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	msg := fmt.Sprintf("parse error in %s", e.File)
	if e.Line > 0 {
		msg += fmt.Sprintf(" at line %d", e.Line)
		if e.Column > 0 {
			msg += fmt.Sprintf(", column %d", e.Column)
		}
	}
	msg += ": " + e.Message
	return msg
}

// Unwrap returns the underlying cause of the error.
func (e *ParseError) Unwrap() error {
	return e.Cause
}

