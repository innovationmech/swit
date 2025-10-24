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
	"bytes"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Generator generates Go code from Saga DSL definitions.
type Generator struct {
	// TemplateDir is the directory containing custom templates
	TemplateDir string

	// OutputDir is the directory where generated files will be written
	OutputDir string

	// logger for debugging
	logger Logger

	// templates cache
	templates *template.Template
}

// GenerateOptions configures the code generation process.
type GenerateOptions struct {
	// PackageName is the Go package name for generated code
	PackageName string

	// GenerateTests indicates whether to generate test skeleton
	GenerateTests bool

	// FormatCode indicates whether to run gofmt on generated code
	FormatCode bool

	// IncludeComments indicates whether to include detailed comments
	IncludeComments bool

	// CustomTemplates allows overriding default templates
	CustomTemplates map[string]string
}

// NewGenerator creates a new code generator.
func NewGenerator(opts ...GeneratorOption) *Generator {
	g := &Generator{
		TemplateDir: "",
		OutputDir:   ".",
		logger:      &defaultLogger{logger: log.New(os.Stdout, "", log.LstdFlags)},
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// GeneratorOption configures the Generator.
type GeneratorOption func(*Generator)

// WithTemplateDir sets the custom template directory.
func WithTemplateDir(dir string) GeneratorOption {
	return func(g *Generator) {
		g.TemplateDir = dir
	}
}

// WithOutputDir sets the output directory for generated files.
func WithOutputDir(dir string) GeneratorOption {
	return func(g *Generator) {
		g.OutputDir = dir
	}
}

// WithGeneratorLogger sets the logger for the generator.
func WithGeneratorLogger(log Logger) GeneratorOption {
	return func(g *Generator) {
		g.logger = log
	}
}

// Generate generates Go code from a Saga definition and writes it to files.
func (g *Generator) Generate(def *SagaDefinition, opts GenerateOptions) error {
	g.logger.Info("generating code for Saga", "saga_id", def.Saga.ID, "package", opts.PackageName)

	// Apply defaults
	if opts.PackageName == "" {
		opts.PackageName = "saga"
	}

	// Initialize templates
	if err := g.initTemplates(opts); err != nil {
		return fmt.Errorf("failed to initialize templates: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(g.OutputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate main Saga file
	mainFile := filepath.Join(g.OutputDir, fmt.Sprintf("%s.go", toSnakeCase(def.Saga.ID)))
	if err := g.generateMainFile(def, opts, mainFile); err != nil {
		return fmt.Errorf("failed to generate main file: %w", err)
	}
	g.logger.Info("generated main file", "path", mainFile)

	// Generate test file if requested
	if opts.GenerateTests {
		testFile := filepath.Join(g.OutputDir, fmt.Sprintf("%s_test.go", toSnakeCase(def.Saga.ID)))
		if err := g.generateTestFile(def, opts, testFile); err != nil {
			return fmt.Errorf("failed to generate test file: %w", err)
		}
		g.logger.Info("generated test file", "path", testFile)
	}

	g.logger.Info("code generation completed successfully")
	return nil
}

// GenerateToWriter generates Go code from a Saga definition and writes it to a writer.
func (g *Generator) GenerateToWriter(def *SagaDefinition, w io.Writer, opts GenerateOptions) error {
	g.logger.Info("generating code to writer", "saga_id", def.Saga.ID)

	// Apply defaults
	if opts.PackageName == "" {
		opts.PackageName = "saga"
	}

	// Initialize templates
	if err := g.initTemplates(opts); err != nil {
		return fmt.Errorf("failed to initialize templates: %w", err)
	}

	// Generate code to buffer first
	var buf bytes.Buffer
	if err := g.generateToBuffer(def, opts, &buf); err != nil {
		return err
	}

	// Format code if requested
	var output []byte
	var err error
	if opts.FormatCode {
		output, err = format.Source(buf.Bytes())
		if err != nil {
			g.logger.Warn("failed to format generated code, using unformatted", "error", err)
			output = buf.Bytes()
		}
	} else {
		output = buf.Bytes()
	}

	// Write to output
	if _, err := w.Write(output); err != nil {
		return fmt.Errorf("failed to write generated code: %w", err)
	}

	return nil
}

// initTemplates initializes the template engine with default and custom templates.
func (g *Generator) initTemplates(opts GenerateOptions) error {
	g.templates = template.New("saga").Funcs(g.templateFuncs())

	// Load default templates
	defaultTemplates := g.getDefaultTemplates()
	for name, tmpl := range defaultTemplates {
		if _, err := g.templates.New(name).Parse(tmpl); err != nil {
			return fmt.Errorf("failed to parse default template %s: %w", name, err)
		}
	}

	// Override with custom templates if provided
	for name, tmpl := range opts.CustomTemplates {
		if _, err := g.templates.New(name).Parse(tmpl); err != nil {
			return fmt.Errorf("failed to parse custom template %s: %w", name, err)
		}
	}

	// Load templates from directory if specified
	if g.TemplateDir != "" {
		pattern := filepath.Join(g.TemplateDir, "*.tmpl")
		if _, err := g.templates.ParseGlob(pattern); err != nil {
			g.logger.Warn("failed to load templates from directory", "path", g.TemplateDir, "error", err)
		}
	}

	return nil
}

// generateMainFile generates the main Saga implementation file.
func (g *Generator) generateMainFile(def *SagaDefinition, opts GenerateOptions, filename string) error {
	var buf bytes.Buffer
	if err := g.generateToBuffer(def, opts, &buf); err != nil {
		return err
	}

	// Format code if requested
	var output []byte
	var err error
	if opts.FormatCode {
		output, err = format.Source(buf.Bytes())
		if err != nil {
			g.logger.Warn("failed to format generated code, using unformatted", "error", err)
			output = buf.Bytes()
		}
	} else {
		output = buf.Bytes()
	}

	// Write to file
	if err := os.WriteFile(filename, output, 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// generateTestFile generates the test skeleton file.
func (g *Generator) generateTestFile(def *SagaDefinition, opts GenerateOptions, filename string) error {
	var buf bytes.Buffer

	data := g.prepareTemplateData(def, opts)
	if err := g.templates.ExecuteTemplate(&buf, "test", data); err != nil {
		return fmt.Errorf("failed to execute test template: %w", err)
	}

	// Format code if requested
	var output []byte
	var err error
	if opts.FormatCode {
		output, err = format.Source(buf.Bytes())
		if err != nil {
			g.logger.Warn("failed to format test code, using unformatted", "error", err)
			output = buf.Bytes()
		}
	} else {
		output = buf.Bytes()
	}

	// Write to file
	if err := os.WriteFile(filename, output, 0o644); err != nil {
		return fmt.Errorf("failed to write test file: %w", err)
	}

	return nil
}

// generateToBuffer generates code to a buffer.
func (g *Generator) generateToBuffer(def *SagaDefinition, opts GenerateOptions, buf *bytes.Buffer) error {
	data := g.prepareTemplateData(def, opts)

	if err := g.templates.ExecuteTemplate(buf, "main", data); err != nil {
		return fmt.Errorf("failed to execute main template: %w", err)
	}

	return nil
}

// prepareTemplateData prepares the data for template execution.
func (g *Generator) prepareTemplateData(def *SagaDefinition, opts GenerateOptions) map[string]interface{} {
	return map[string]interface{}{
		"PackageName":        opts.PackageName,
		"Saga":               def.Saga,
		"Steps":              def.Steps,
		"GlobalRetry":        def.GlobalRetryPolicy,
		"GlobalCompensation": def.GlobalCompensation,
		"IncludeComments":    opts.IncludeComments,
		"SagaStructName":     toPascalCase(def.Saga.ID),
		"SagaVarName":        toCamelCase(def.Saga.ID),
	}
}

// templateFuncs returns custom template functions.
func (g *Generator) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"toPascalCase":     toPascalCase,
		"toCamelCase":      toCamelCase,
		"toSnakeCase":      toSnakeCase,
		"toUpper":          strings.ToUpper,
		"toLower":          strings.ToLower,
		"quote":            quote,
		"hasCompensation":  hasCompensation,
		"needsRetry":       needsRetry,
		"formatDuration":   formatDuration,
		"stepHandlerName":  stepHandlerName,
		"compensationName": compensationName,
		"actionTypeName":   actionTypeName,
		"indent":           indent,
	}
}

// getDefaultTemplates returns the default code generation templates.
func (g *Generator) getDefaultTemplates() map[string]string {
	return map[string]string{
		"main": mainTemplate,
		"test": testTemplate,
	}
}

// Template helper functions

func toPascalCase(s string) string {
	words := splitWords(s)
	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]))
			result.WriteString(strings.ToLower(word[1:]))
		}
	}
	return result.String()
}

func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) > 0 {
		return strings.ToLower(pascal[:1]) + pascal[1:]
	}
	return pascal
}

func toSnakeCase(s string) string {
	words := splitWords(s)
	for i := range words {
		words[i] = strings.ToLower(words[i])
	}
	return strings.Join(words, "_")
}

func splitWords(s string) []string {
	// Split by common delimiters
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")

	var words []string
	var current strings.Builder

	for i, r := range s {
		if r == ' ' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		} else if i > 0 && isUpperCase(r) && !isUpperCase(rune(s[i-1])) {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			current.WriteRune(r)
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

func isUpperCase(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func quote(s string) string {
	return fmt.Sprintf(`"%s"`, s)
}

func hasCompensation(step StepConfig) bool {
	return step.Compensation != nil && step.Compensation.Type != CompensationTypeSkip
}

func needsRetry(step StepConfig) bool {
	return step.RetryPolicy != nil && step.RetryPolicy.MaxAttempts > 0
}

func formatDuration(d Duration) string {
	// Convert duration to time.Duration constant expression
	dur := time.Duration(d)
	switch {
	case dur == 0:
		return "0"
	case dur%time.Hour == 0:
		return fmt.Sprintf("%d * time.Hour", dur/time.Hour)
	case dur%time.Minute == 0:
		return fmt.Sprintf("%d * time.Minute", dur/time.Minute)
	case dur%time.Second == 0:
		return fmt.Sprintf("%d * time.Second", dur/time.Second)
	case dur%time.Millisecond == 0:
		return fmt.Sprintf("%d * time.Millisecond", dur/time.Millisecond)
	default:
		return fmt.Sprintf("time.Duration(%d)", dur)
	}
}

func stepHandlerName(stepID string) string {
	return "Handle" + toPascalCase(stepID)
}

func compensationName(stepID string) string {
	return "Compensate" + toPascalCase(stepID)
}

func actionTypeName(stepType StepType) string {
	switch stepType {
	case StepTypeService:
		return "ServiceAction"
	case StepTypeHTTP:
		return "HTTPAction"
	case StepTypeGRPC:
		return "GRPCAction"
	case StepTypeFunction:
		return "FunctionAction"
	case StepTypeMessage:
		return "MessageAction"
	case StepTypeCustom:
		return "CustomAction"
	default:
		return "Action"
	}
}

func indent(spaces int, s string) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i := range lines {
		if lines[i] != "" {
			lines[i] = prefix + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

// Default templates

const mainTemplate = `// Code generated by Saga DSL generator. DO NOT EDIT.

package {{.PackageName}}

import (
	"context"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

{{if .IncludeComments}}
// {{.SagaStructName}} implements the {{.Saga.Name}} saga.
// {{.Saga.Description}}
{{end}}
type {{.SagaStructName}} struct {
	orchestrator *saga.Orchestrator
	{{range .Steps}}
	{{toCamelCase .ID}}Handler {{actionTypeName .Type}}
	{{end}}
}

{{if .IncludeComments}}
// New{{.SagaStructName}} creates a new instance of {{.SagaStructName}}.
{{end}}
func New{{.SagaStructName}}() *{{.SagaStructName}} {
	return &{{.SagaStructName}}{}
}

{{if .IncludeComments}}
// Execute executes the {{.Saga.Name}} saga.
{{end}}
func (s *{{.SagaStructName}}) Execute(ctx context.Context, input map[string]interface{}) error {
	{{if .IncludeComments}}
	// Create saga instance
	{{end}}
	sagaInstance := saga.NewSaga("{{.Saga.ID}}", "{{.Saga.Name}}")
	
	{{if .IncludeComments}}
	// Set timeout if specified
	{{end}}
	{{if .Saga.Timeout}}
	ctx, cancel := context.WithTimeout(ctx, {{formatDuration .Saga.Timeout}})
	defer cancel()
	{{end}}

	{{if .IncludeComments}}
	// Register steps
	{{end}}
	{{range .Steps}}
	sagaInstance.AddStep(saga.Step{
		ID: "{{.ID}}",
		Name: "{{.Name}}",
		Action: s.{{stepHandlerName .ID}},
		{{if hasCompensation .}}
		Compensation: s.{{compensationName .ID}},
		{{end}}
		{{if .Timeout}}
		Timeout: {{formatDuration .Timeout}},
		{{end}}
	})
	{{end}}

	{{if .IncludeComments}}
	// Execute saga
	{{end}}
	return sagaInstance.Execute(ctx, input)
}

{{if .IncludeComments}}
// Step handler functions
{{end}}
{{range .Steps}}
{{if $.IncludeComments}}
// {{stepHandlerName .ID}} handles the {{.Name}} step.
// {{.Description}}
{{end}}
func (s *{{$.SagaStructName}}) {{stepHandlerName .ID}}(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	{{if $.IncludeComments}}
	// TODO: Implement {{.Name}} logic
	{{end}}
	{{if eq .Type "service"}}
	// Service call: {{.Action.Service.Name}}
	// Method: {{.Action.Service.Method}}
	// Endpoint: {{.Action.Service.Endpoint}}
	{{else if eq .Type "grpc"}}
	// gRPC call: {{.Action.Service.Name}}
	// Method: {{.Action.Service.Method}}
	{{else if eq .Type "http"}}
	// HTTP call: {{.Action.Service.Method}} {{.Action.Service.Path}}
	{{else if eq .Type "message"}}
	// Publish message to: {{.Action.Message.Topic}}
	{{else if eq .Type "function"}}
	// Execute function: {{.Action.Function.Name}}
	{{else}}
	// Custom action
	{{end}}
	
	return nil, fmt.Errorf("not implemented: {{.Name}}")
}

{{if hasCompensation .}}
{{if $.IncludeComments}}
// {{compensationName .ID}} compensates the {{.Name}} step.
{{end}}
func (s *{{$.SagaStructName}}) {{compensationName .ID}}(ctx context.Context, input map[string]interface{}) error {
	{{if $.IncludeComments}}
	// TODO: Implement {{.Name}} compensation logic
	{{end}}
	{{if eq .Compensation.Type "custom"}}
	// Custom compensation
	{{else if eq .Compensation.Type "automatic"}}
	// Automatic compensation
	{{end}}
	
	return fmt.Errorf("not implemented: compensate {{.Name}}")
}
{{end}}

{{end}}
`

const testTemplate = `// Code generated by Saga DSL generator. DO NOT EDIT.

package {{.PackageName}}

import (
	"context"
	"testing"
	"time"
)

func Test{{.SagaStructName}}_Execute(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		wantErr bool
	}{
		{
			name: "successful execution",
			input: map[string]interface{}{
				// TODO: Add test input
			},
			wantErr: false,
		},
		{
			name: "execution with error",
			input: map[string]interface{}{
				// TODO: Add test input that triggers error
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New{{.SagaStructName}}()
			ctx := context.Background()
			
			err := s.Execute(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

{{range .Steps}}
func Test{{$.SagaStructName}}_{{stepHandlerName .ID}}(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "successful {{.Name}}",
			input: map[string]interface{}{
				// TODO: Add test input
			},
			want: map[string]interface{}{
				// TODO: Add expected output
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New{{$.SagaStructName}}()
			ctx := context.Background()
			
			got, err := s.{{stepHandlerName .ID}}(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("{{stepHandlerName .ID}}() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			// TODO: Add proper comparison of got and tt.want
			_ = got
		})
	}
}

{{if hasCompensation .}}
func Test{{$.SagaStructName}}_{{compensationName .ID}}(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		wantErr bool
	}{
		{
			name: "successful compensation",
			input: map[string]interface{}{
				// TODO: Add test input
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New{{$.SagaStructName}}()
			ctx := context.Background()
			
			err := s.{{compensationName .ID}}(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("{{compensationName .ID}}() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
{{end}}

{{end}}
`
