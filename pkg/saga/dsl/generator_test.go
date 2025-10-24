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
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name string
		opts []GeneratorOption
		want *Generator
	}{
		{
			name: "default generator",
			opts: nil,
			want: &Generator{
				OutputDir: ".",
			},
		},
		{
			name: "with output dir",
			opts: []GeneratorOption{WithOutputDir("/tmp/output")},
			want: &Generator{
				OutputDir: "/tmp/output",
			},
		},
		{
			name: "with template dir",
			opts: []GeneratorOption{WithTemplateDir("/tmp/templates")},
			want: &Generator{
				TemplateDir: "/tmp/templates",
				OutputDir:   ".",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewGenerator(tt.opts...)
			if got.OutputDir != tt.want.OutputDir {
				t.Errorf("NewGenerator() OutputDir = %v, want %v", got.OutputDir, tt.want.OutputDir)
			}
			if got.TemplateDir != tt.want.TemplateDir {
				t.Errorf("NewGenerator() TemplateDir = %v, want %v", got.TemplateDir, tt.want.TemplateDir)
			}
		})
	}
}

func TestGenerator_GenerateToWriter(t *testing.T) {
	def := createTestSagaDefinition()

	tests := []struct {
		name      string
		def       *SagaDefinition
		opts      GenerateOptions
		wantErr   bool
		checkFunc func(t *testing.T, output string)
	}{
		{
			name: "basic generation",
			def:  def,
			opts: GenerateOptions{
				PackageName: "testpkg",
				FormatCode:  true,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "package testpkg") {
					t.Error("generated code should contain package declaration")
				}
				if !strings.Contains(output, "type OrderProcessingSaga struct") {
					t.Error("generated code should contain saga struct")
				}
			},
		},
		{
			name: "with comments",
			def:  def,
			opts: GenerateOptions{
				PackageName:     "testpkg",
				IncludeComments: true,
				FormatCode:      true,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "// OrderProcessingSaga") {
					t.Error("generated code should contain struct comment")
				}
			},
		},
		{
			name: "without formatting",
			def:  def,
			opts: GenerateOptions{
				PackageName: "testpkg",
				FormatCode:  false,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, output string) {
				// Should still contain basic code structure
				if !strings.Contains(output, "package testpkg") {
					t.Error("generated code should contain package declaration")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator()
			var buf bytes.Buffer

			err := g.GenerateToWriter(tt.def, &buf, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, buf.String())
			}

			// Validate that generated code is syntactically correct
			if !tt.wantErr && tt.opts.FormatCode {
				fset := token.NewFileSet()
				_, err := parser.ParseFile(fset, "test.go", buf.Bytes(), parser.AllErrors)
				if err != nil {
					t.Errorf("Generated code has syntax errors: %v\nCode:\n%s", err, buf.String())
				}
			}
		})
	}
}

func TestGenerator_Generate(t *testing.T) {
	def := createTestSagaDefinition()
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		def       *SagaDefinition
		opts      GenerateOptions
		outputDir string
		wantErr   bool
		checkFunc func(t *testing.T, outputDir string)
	}{
		{
			name: "generate main file",
			def:  def,
			opts: GenerateOptions{
				PackageName:   "saga",
				FormatCode:    true,
				GenerateTests: false,
			},
			outputDir: tempDir,
			wantErr:   false,
			checkFunc: func(t *testing.T, outputDir string) {
				mainFile := filepath.Join(outputDir, "order_processing_saga.go")
				if _, err := os.Stat(mainFile); os.IsNotExist(err) {
					t.Errorf("main file should be generated at %s", mainFile)
				}
			},
		},
		{
			name: "generate with tests",
			def:  def,
			opts: GenerateOptions{
				PackageName:   "saga",
				FormatCode:    true,
				GenerateTests: true,
			},
			outputDir: tempDir,
			wantErr:   false,
			checkFunc: func(t *testing.T, outputDir string) {
				mainFile := filepath.Join(outputDir, "order_processing_saga.go")
				testFile := filepath.Join(outputDir, "order_processing_saga_test.go")

				if _, err := os.Stat(mainFile); os.IsNotExist(err) {
					t.Errorf("main file should be generated at %s", mainFile)
				}
				if _, err := os.Stat(testFile); os.IsNotExist(err) {
					t.Errorf("test file should be generated at %s", testFile)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator(WithOutputDir(tt.outputDir))

			err := g.Generate(tt.def, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, tt.outputDir)
			}
		})
	}
}

func TestGenerator_CustomTemplates(t *testing.T) {
	def := createTestSagaDefinition()

	customMainTemplate := `package {{.PackageName}}

// Custom template for {{.Saga.Name}}
type Custom{{.SagaStructName}} struct {}
`

	tests := []struct {
		name    string
		def     *SagaDefinition
		opts    GenerateOptions
		wantErr bool
		want    string
	}{
		{
			name: "custom template",
			def:  def,
			opts: GenerateOptions{
				PackageName: "custom",
				CustomTemplates: map[string]string{
					"main": customMainTemplate,
				},
			},
			wantErr: false,
			want:    "CustomOrderProcessingSaga",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator()
			var buf bytes.Buffer

			err := g.GenerateToWriter(tt.def, &buf, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()
			if !strings.Contains(output, tt.want) {
				t.Errorf("Generated code should contain %q, got:\n%s", tt.want, output)
			}
		})
	}
}

func TestTemplateFunctions(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		input    interface{}
		want     string
	}{
		{
			name:     "toPascalCase with dashes",
			funcName: "toPascalCase",
			input:    "order-processing-saga",
			want:     "OrderProcessingSaga",
		},
		{
			name:     "toPascalCase with underscores",
			funcName: "toPascalCase",
			input:    "order_processing_saga",
			want:     "OrderProcessingSaga",
		},
		{
			name:     "toCamelCase",
			funcName: "toCamelCase",
			input:    "order-processing-saga",
			want:     "orderProcessingSaga",
		},
		{
			name:     "toSnakeCase",
			funcName: "toSnakeCase",
			input:    "OrderProcessingSaga",
			want:     "order_processing_saga",
		},
		{
			name:     "stepHandlerName",
			funcName: "stepHandlerName",
			input:    "validate-order",
			want:     "HandleValidateOrder",
		},
		{
			name:     "compensationName",
			funcName: "compensationName",
			input:    "validate-order",
			want:     "CompensateValidateOrder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			switch tt.funcName {
			case "toPascalCase":
				got = toPascalCase(tt.input.(string))
			case "toCamelCase":
				got = toCamelCase(tt.input.(string))
			case "toSnakeCase":
				got = toSnakeCase(tt.input.(string))
			case "stepHandlerName":
				got = stepHandlerName(tt.input.(string))
			case "compensationName":
				got = compensationName(tt.input.(string))
			}

			if got != tt.want {
				t.Errorf("%s(%v) = %v, want %v", tt.funcName, tt.input, got, tt.want)
			}
		})
	}
}

func TestHasCompensation(t *testing.T) {
	tests := []struct {
		name string
		step StepConfig
		want bool
	}{
		{
			name: "has custom compensation",
			step: StepConfig{
				Compensation: &CompensationAction{
					Type: CompensationTypeCustom,
				},
			},
			want: true,
		},
		{
			name: "has skip compensation",
			step: StepConfig{
				Compensation: &CompensationAction{
					Type: CompensationTypeSkip,
				},
			},
			want: false,
		},
		{
			name: "no compensation",
			step: StepConfig{
				Compensation: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasCompensation(tt.step); got != tt.want {
				t.Errorf("hasCompensation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedsRetry(t *testing.T) {
	tests := []struct {
		name string
		step StepConfig
		want bool
	}{
		{
			name: "has retry policy",
			step: StepConfig{
				RetryPolicy: &RetryPolicy{
					MaxAttempts: 3,
				},
			},
			want: true,
		},
		{
			name: "no retry policy",
			step: StepConfig{
				RetryPolicy: nil,
			},
			want: false,
		},
		{
			name: "zero max attempts",
			step: StepConfig{
				RetryPolicy: &RetryPolicy{
					MaxAttempts: 0,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsRetry(tt.step); got != tt.want {
				t.Errorf("needsRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActionTypeName(t *testing.T) {
	tests := []struct {
		name     string
		stepType StepType
		want     string
	}{
		{"service", StepTypeService, "ServiceAction"},
		{"http", StepTypeHTTP, "HTTPAction"},
		{"grpc", StepTypeGRPC, "GRPCAction"},
		{"function", StepTypeFunction, "FunctionAction"},
		{"message", StepTypeMessage, "MessageAction"},
		{"custom", StepTypeCustom, "CustomAction"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := actionTypeName(tt.stepType); got != tt.want {
				t.Errorf("actionTypeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndent(t *testing.T) {
	tests := []struct {
		name   string
		spaces int
		input  string
		want   string
	}{
		{
			name:   "single line",
			spaces: 4,
			input:  "hello",
			want:   "    hello",
		},
		{
			name:   "multiple lines",
			spaces: 2,
			input:  "line1\nline2\nline3",
			want:   "  line1\n  line2\n  line3",
		},
		{
			name:   "empty lines",
			spaces: 4,
			input:  "line1\n\nline3",
			want:   "    line1\n\n    line3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indent(tt.spaces, tt.input); got != tt.want {
				t.Errorf("indent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    Duration
		want string
	}{
		{
			name: "seconds",
			d:    Duration(5 * time.Second),
			want: "5 * time.Second",
		},
		{
			name: "minutes",
			d:    Duration(2 * time.Minute),
			want: "2 * time.Minute",
		},
		{
			name: "hours",
			d:    Duration(3 * time.Hour),
			want: "3 * time.Hour",
		},
		{
			name: "milliseconds",
			d:    Duration(500 * time.Millisecond),
			want: "500 * time.Millisecond",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.d); got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to create a test saga definition
func createTestSagaDefinition() *SagaDefinition {
	return &SagaDefinition{
		Saga: SagaConfig{
			ID:          "order-processing-saga",
			Name:        "Order Processing Saga",
			Description: "Processes customer orders",
			Version:     "1.0.0",
			Timeout:     Duration(5 * time.Minute),
			Mode:        ExecutionModeOrchestration,
		},
		Steps: []StepConfig{
			{
				ID:          "validate-order",
				Name:        "Validate Order",
				Description: "Validates order data",
				Type:        StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:     "order-service",
						Endpoint: "http://order-service:8080",
						Method:   "POST",
						Path:     "/api/orders/validate",
					},
				},
				Compensation: &CompensationAction{
					Type: CompensationTypeSkip,
				},
				Timeout: Duration(30 * time.Second),
			},
			{
				ID:          "reserve-inventory",
				Name:        "Reserve Inventory",
				Description: "Reserves inventory",
				Type:        StepTypeGRPC,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:     "inventory-service",
						Endpoint: "inventory-service:9090",
						Method:   "ReserveInventory",
					},
				},
				Compensation: &CompensationAction{
					Type: CompensationTypeCustom,
					Action: &ActionConfig{
						Service: &ServiceConfig{
							Name:   "inventory-service",
							Method: "ReleaseInventory",
						},
					},
				},
				RetryPolicy: &RetryPolicy{
					Type:        "exponential_backoff",
					MaxAttempts: 3,
				},
				Dependencies: []string{"validate-order"},
			},
			{
				ID:          "process-payment",
				Name:        "Process Payment",
				Description: "Processes payment",
				Type:        StepTypeHTTP,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:     "payment-service",
						Endpoint: "http://payment-service:8080",
						Method:   "POST",
						Path:     "/api/payments/process",
					},
				},
				Compensation: &CompensationAction{
					Type: CompensationTypeCustom,
					Action: &ActionConfig{
						Service: &ServiceConfig{
							Name:   "payment-service",
							Method: "POST",
							Path:   "/api/payments/refund",
						},
					},
				},
				Dependencies: []string{"validate-order"},
			},
		},
		GlobalRetryPolicy: &RetryPolicy{
			Type:        "exponential_backoff",
			MaxAttempts: 3,
		},
		GlobalCompensation: &CompensationConfig{
			Strategy:    CompensationStrategySequential,
			MaxAttempts: 3,
		},
	}
}
