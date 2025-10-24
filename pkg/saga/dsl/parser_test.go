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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBytes_ValidDefinition(t *testing.T) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga
  description: A test saga
  version: "1.0.0"
  timeout: 5m
  mode: orchestration

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
        path: /api/test
    timeout: 30s
`

	def, err := ParseBytes([]byte(yamlContent))
	require.NoError(t, err)
	require.NotNil(t, def)

	assert.Equal(t, "test-saga", def.Saga.ID)
	assert.Equal(t, "Test Saga", def.Saga.Name)
	assert.Equal(t, "A test saga", def.Saga.Description)
	assert.Equal(t, "1.0.0", def.Saga.Version)
	assert.Equal(t, Duration(5*time.Minute), def.Saga.Timeout)
	assert.Equal(t, ExecutionModeOrchestration, def.Saga.Mode)

	require.Len(t, def.Steps, 1)
	assert.Equal(t, "step1", def.Steps[0].ID)
	assert.Equal(t, "Step 1", def.Steps[0].Name)
	assert.Equal(t, StepTypeService, def.Steps[0].Type)
	assert.NotNil(t, def.Steps[0].Action.Service)
	assert.Equal(t, "test-service", def.Steps[0].Action.Service.Name)
}

func TestParseBytes_EmptyContent(t *testing.T) {
	_, err := ParseBytes([]byte(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty YAML content")
}

func TestParseBytes_InvalidYAML(t *testing.T) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      - invalid yaml structure
`

	_, err := ParseBytes([]byte(yamlContent))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "YAML syntax error")
}

func TestParseBytes_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectedErr string
	}{
		{
			name: "missing saga ID",
			yaml: `
saga:
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: "validation",
		},
		{
			name: "missing saga name",
			yaml: `
saga:
  id: test-saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: "validation",
		},
		{
			name: "missing steps",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps: []
`,
			expectedErr: "validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestParseBytes_GlobalRetryPolicy(t *testing.T) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

global_retry_policy:
  type: exponential_backoff
  max_attempts: 3
  initial_delay: 1s
  max_delay: 30s
  multiplier: 2.0
  jitter: true

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST

  - id: step2
    name: Step 2
    type: service
    action:
      service:
        name: test-service
        method: POST
    retry_policy:
      type: fixed_delay
      max_attempts: 5
      initial_delay: 5s
`

	def, err := ParseBytes([]byte(yamlContent))
	require.NoError(t, err)

	// Step1 should inherit global retry policy
	assert.NotNil(t, def.Steps[0].RetryPolicy)
	assert.Equal(t, "exponential_backoff", def.Steps[0].RetryPolicy.Type)
	assert.Equal(t, 3, def.Steps[0].RetryPolicy.MaxAttempts)

	// Step2 should use its own retry policy
	assert.NotNil(t, def.Steps[1].RetryPolicy)
	assert.Equal(t, "fixed_delay", def.Steps[1].RetryPolicy.Type)
	assert.Equal(t, 5, def.Steps[1].RetryPolicy.MaxAttempts)
}

func TestParseBytes_StepDependencies(t *testing.T) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST

  - id: step2
    name: Step 2
    type: service
    dependencies:
      - step1
    action:
      service:
        name: test-service
        method: POST

  - id: step3
    name: Step 3
    type: service
    dependencies:
      - step1
      - step2
    action:
      service:
        name: test-service
        method: POST
`

	def, err := ParseBytes([]byte(yamlContent))
	require.NoError(t, err)

	assert.Len(t, def.Steps[0].Dependencies, 0)
	assert.Equal(t, []string{"step1"}, def.Steps[1].Dependencies)
	assert.Equal(t, []string{"step1", "step2"}, def.Steps[2].Dependencies)
}

func TestParseBytes_InvalidDependency(t *testing.T) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    dependencies:
      - nonexistent-step
    action:
      service:
        name: test-service
        method: POST
`

	_, err := ParseBytes([]byte(yamlContent))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-existent step")
}

func TestParseBytes_CircularDependency(t *testing.T) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    dependencies:
      - step2
    action:
      service:
        name: test-service
        method: POST

  - id: step2
    name: Step 2
    type: service
    dependencies:
      - step1
    action:
      service:
        name: test-service
        method: POST
`

	_, err := ParseBytes([]byte(yamlContent))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestParseBytes_DuplicateStepIDs(t *testing.T) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST

  - id: step1
    name: Step 1 Duplicate
    type: service
    action:
      service:
        name: test-service
        method: POST
`

	_, err := ParseBytes([]byte(yamlContent))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate step ID")
}

func TestParseBytes_StepTypes(t *testing.T) {
	tests := []struct {
		name     string
		stepType StepType
		action   string
	}{
		{
			name:     "service type",
			stepType: StepTypeService,
			action: `
      service:
        name: test-service
        method: POST
`,
		},
		{
			name:     "function type",
			stepType: StepTypeFunction,
			action: `
      function:
        name: test-func
        handler: pkg.Handler
`,
		},
		{
			name:     "message type",
			stepType: StepTypeMessage,
			action: `
      message:
        topic: test-topic
        broker: default
`,
		},
		{
			name:     "http type",
			stepType: StepTypeHTTP,
			action: `
      service:
        name: test-service
        endpoint: http://example.com
        method: POST
        path: /api/test
`,
		},
		{
			name:     "grpc type",
			stepType: StepTypeGRPC,
			action: `
      service:
        name: test-service
        endpoint: example.com:9090
        method: TestMethod
`,
		},
		{
			name:     "custom type",
			stepType: StepTypeCustom,
			action: `
      custom:
        type: my-custom-type
        config:
          key: value
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: ` + string(tt.stepType) + `
    action:
` + tt.action

			def, err := ParseBytes([]byte(yamlContent))
			require.NoError(t, err)
			assert.Equal(t, tt.stepType, def.Steps[0].Type)
		})
	}
}

func TestParseBytes_InvalidStepAction(t *testing.T) {
	tests := []struct {
		name        string
		stepType    StepType
		action      string
		expectedErr string
	}{
		{
			name:     "service without service action",
			stepType: StepTypeService,
			action: `
      function:
        name: test-func
        handler: pkg.Handler
`,
			expectedErr: "service action is required",
		},
		{
			name:     "function without function action",
			stepType: StepTypeFunction,
			action: `
      service:
        name: test-service
        method: POST
`,
			expectedErr: "function action is required",
		},
		{
			name:     "message without message action",
			stepType: StepTypeMessage,
			action: `
      service:
        name: test-service
        method: POST
`,
			expectedErr: "message action is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: ` + string(tt.stepType) + `
    action:
` + tt.action

			_, err := ParseBytes([]byte(yamlContent))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestParseBytes_Compensation(t *testing.T) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
    compensation:
      type: custom
      action:
        service:
          name: test-service
          method: DELETE
      strategy: sequential
      timeout: 1m
      max_attempts: 3

  - id: step2
    name: Step 2
    type: service
    action:
      service:
        name: test-service
        method: POST
    compensation:
      type: skip
`

	def, err := ParseBytes([]byte(yamlContent))
	require.NoError(t, err)

	// Step1 has custom compensation
	assert.Equal(t, CompensationTypeCustom, def.Steps[0].Compensation.Type)
	assert.NotNil(t, def.Steps[0].Compensation.Action)
	assert.Equal(t, CompensationStrategySequential, def.Steps[0].Compensation.Strategy)

	// Step2 skips compensation
	assert.Equal(t, CompensationTypeSkip, def.Steps[1].Compensation.Type)
}

func TestParseBytes_InvalidCompensation(t *testing.T) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
    compensation:
      type: custom
      # Missing action for custom compensation
`

	_, err := ParseBytes([]byte(yamlContent))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "custom compensation requires an action")
}

func TestParseFile_Success(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-saga.yaml")

	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`

	err := os.WriteFile(testFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Parse the file
	def, err := ParseFile(testFile, WithBasePath(tmpDir))
	require.NoError(t, err)
	require.NotNil(t, def)

	assert.Equal(t, "test-saga", def.Saga.ID)
	assert.Equal(t, "Test Saga", def.Saga.Name)
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile("/nonexistent/file.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestExpandEnvVars(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_SERVICE_NAME", "my-service")
	os.Setenv("TEST_ENDPOINT", "http://example.com")
	defer func() {
		os.Unsetenv("TEST_SERVICE_NAME")
		os.Unsetenv("TEST_ENDPOINT")
	}()

	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: ${TEST_SERVICE_NAME}
        endpoint: $TEST_ENDPOINT
        method: POST
`

	def, err := ParseBytes([]byte(yamlContent), WithEnvVars(true))
	require.NoError(t, err)

	assert.Equal(t, "my-service", def.Steps[0].Action.Service.Name)
	assert.Equal(t, "http://example.com", def.Steps[0].Action.Service.Endpoint)
}

func TestExpandEnvVars_Disabled(t *testing.T) {
	os.Setenv("TEST_SERVICE_NAME", "my-service")
	defer os.Unsetenv("TEST_SERVICE_NAME")

	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: ${TEST_SERVICE_NAME}
        method: POST
`

	def, err := ParseBytes([]byte(yamlContent), WithEnvVars(false))
	require.NoError(t, err)

	// Environment variable should not be expanded
	assert.Equal(t, "${TEST_SERVICE_NAME}", def.Steps[0].Action.Service.Name)
}

func TestExpandEnvVars_NotFound(t *testing.T) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: ${NONEXISTENT_VAR}
        method: POST
`

	def, err := ParseBytes([]byte(yamlContent), WithEnvVars(true))
	require.NoError(t, err)

	// Non-existent variable should remain as placeholder
	assert.Equal(t, "${NONEXISTENT_VAR}", def.Steps[0].Action.Service.Name)
}

func TestProcessIncludes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an included file
	includedFile := filepath.Join(tmpDir, "included.yaml")
	includedContent := `
saga:
  id: included-saga
  name: Included Saga

steps:
  - id: step1
    name: Included Step
    type: service
    action:
      service:
        name: included-service
        method: POST
`
	err := os.WriteFile(includedFile, []byte(includedContent), 0644)
	require.NoError(t, err)

	// Create main file with include directive
	mainFile := filepath.Join(tmpDir, "main.yaml")
	mainContent := `!include included.yaml`
	err = os.WriteFile(mainFile, []byte(mainContent), 0644)
	require.NoError(t, err)

	// Parse with includes enabled
	def, err := ParseFile(mainFile, WithBasePath(tmpDir), WithIncludes(true))
	require.NoError(t, err)

	assert.Equal(t, "included-saga", def.Saga.ID)
	assert.Equal(t, "Included Saga", def.Saga.Name)
}

func TestProcessIncludes_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	mainFile := filepath.Join(tmpDir, "main.yaml")
	mainContent := `!include nonexistent.yaml`
	err := os.WriteFile(mainFile, []byte(mainContent), 0644)
	require.NoError(t, err)

	// Parse with includes disabled - should fail on invalid YAML
	_, err = ParseFile(mainFile, WithBasePath(tmpDir), WithIncludes(false))
	require.Error(t, err)
}

func TestProcessIncludes_CircularDependency(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file1 that includes file2
	file1 := filepath.Join(tmpDir, "file1.yaml")
	file1Content := `!include file2.yaml`
	err := os.WriteFile(file1, []byte(file1Content), 0644)
	require.NoError(t, err)

	// Create file2 that includes file1
	file2 := filepath.Join(tmpDir, "file2.yaml")
	file2Content := `!include file1.yaml`
	err = os.WriteFile(file2, []byte(file2Content), 0644)
	require.NoError(t, err)

	// Parse should detect circular dependency
	_, err = ParseFile(file1, WithBasePath(tmpDir), WithIncludes(true))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestProcessIncludes_MaxDepth(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a chain of includes
	for i := 1; i <= 15; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("file%d.yaml", i))
		var content string
		if i < 15 {
			content = fmt.Sprintf("!include file%d.yaml", i+1)
		} else {
			content = `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`
		}
		err := os.WriteFile(filename, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Parse with default max depth (10) should fail
	_, err := ParseFile(filepath.Join(tmpDir, "file1.yaml"), WithBasePath(tmpDir))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum include depth")
}

func TestParseReader(t *testing.T) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`

	reader := strings.NewReader(yamlContent)
	def, err := ParseReader(reader)
	require.NoError(t, err)
	require.NotNil(t, def)

	assert.Equal(t, "test-saga", def.Saga.ID)
	assert.Equal(t, "Test Saga", def.Saga.Name)
}

func TestApplyDefaults(t *testing.T) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

global_retry_policy:
  type: exponential_backoff
  max_attempts: 3
  initial_delay: 1s

global_compensation:
  strategy: sequential
  timeout: 2m

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
    compensation:
      type: custom
      action:
        service:
          name: test-service
          method: DELETE
`

	def, err := ParseBytes([]byte(yamlContent))
	require.NoError(t, err)

	// Default mode should be orchestration
	assert.Equal(t, ExecutionModeOrchestration, def.Saga.Mode)

	// Step should inherit global retry policy
	assert.NotNil(t, def.Steps[0].RetryPolicy)
	assert.Equal(t, "exponential_backoff", def.Steps[0].RetryPolicy.Type)

	// Step compensation should inherit global strategy
	assert.Equal(t, CompensationStrategySequential, def.Steps[0].Compensation.Strategy)
}

func TestParseError_Error(t *testing.T) {
	err := &ParseError{
		Message: "test error",
		File:    "test.yaml",
		Line:    10,
		Column:  5,
	}

	errMsg := err.Error()
	assert.Contains(t, errMsg, "test.yaml")
	assert.Contains(t, errMsg, "line 10")
	assert.Contains(t, errMsg, "column 5")
	assert.Contains(t, errMsg, "test error")
}

func TestComplexSagaDefinition(t *testing.T) {
	// Use the example.yaml file
	def, err := ParseFile("example.yaml")
	require.NoError(t, err)
	require.NotNil(t, def)

	// Validate basic structure
	assert.Equal(t, "order-processing-saga", def.Saga.ID)
	assert.Equal(t, "Order Processing Saga", def.Saga.Name)
	assert.NotEmpty(t, def.Saga.Description)
	assert.Equal(t, "1.0.0", def.Saga.Version)

	// Validate global policies
	assert.NotNil(t, def.GlobalRetryPolicy)
	assert.Equal(t, "exponential_backoff", def.GlobalRetryPolicy.Type)

	assert.NotNil(t, def.GlobalCompensation)
	assert.Equal(t, CompensationStrategySequential, def.GlobalCompensation.Strategy)

	// Validate steps
	assert.Len(t, def.Steps, 5)

	// Validate first step
	step1 := def.Steps[0]
	assert.Equal(t, "validate-order", step1.ID)
	assert.Equal(t, StepTypeService, step1.Type)
	assert.NotNil(t, step1.Action.Service)
	assert.Equal(t, CompensationTypeSkip, step1.Compensation.Type)

	// Validate step with dependencies
	step3 := def.Steps[2]
	assert.Equal(t, "process-payment", step3.ID)
	assert.Contains(t, step3.Dependencies, "validate-order")
	assert.NotNil(t, step3.Compensation)
	assert.Equal(t, CompensationTypeCustom, step3.Compensation.Type)

	// Validate async step
	step5 := def.Steps[4]
	assert.Equal(t, "notify-shipping", step5.ID)
	assert.True(t, step5.Async)
	assert.Equal(t, StepTypeMessage, step5.Type)
}

// Benchmark tests
func BenchmarkParseBytes_Small(b *testing.B) {
	yamlContent := `
saga:
  id: test-saga
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`

	data := []byte(yamlContent)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseBytes(data)
	}
}

func BenchmarkParseBytes_Large(b *testing.B) {
	// Read the example.yaml file
	data, err := os.ReadFile("example.yaml")
	if err != nil {
		b.Fatalf("failed to read example.yaml: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseBytes(data)
	}
}

func TestParsePerformance_LargeConfig(t *testing.T) {
	// Read the example.yaml file
	data, err := os.ReadFile("example.yaml")
	require.NoError(t, err)

	start := time.Now()
	_, err = ParseBytes(data)
	duration := time.Since(start)

	require.NoError(t, err)

	// Verify performance requirement: < 100ms
	assert.Less(t, duration, 100*time.Millisecond,
		"parsing large config took %v, expected < 100ms", duration)

	t.Logf("Parse time for example.yaml: %v", duration)
}

func TestValidateStepAction_AllTypes(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name      string
		step      StepConfig
		expectErr bool
	}{
		{
			name: "valid service step",
			step: StepConfig{
				ID:   "step1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "valid function step",
			step: StepConfig{
				ID:   "step1",
				Type: StepTypeFunction,
				Action: ActionConfig{
					Function: &FunctionConfig{
						Name:    "test-func",
						Handler: "pkg.Handler",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "valid message step",
			step: StepConfig{
				ID:   "step1",
				Type: StepTypeMessage,
				Action: ActionConfig{
					Message: &MessageConfig{
						Topic: "test-topic",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "invalid service step - missing service config",
			step: StepConfig{
				ID:     "step1",
				Type:   StepTypeService,
				Action: ActionConfig{},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.validateStepAction(&tt.step)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCompensation_Rules(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name      string
		step      StepConfig
		expectErr bool
	}{
		{
			name: "valid custom compensation",
			step: StepConfig{
				ID: "step1",
				Compensation: &CompensationAction{
					Type: CompensationTypeCustom,
					Action: &ActionConfig{
						Service: &ServiceConfig{
							Name:   "test-service",
							Method: "DELETE",
						},
					},
				},
			},
			expectErr: false,
		},
		{
			name: "valid skip compensation",
			step: StepConfig{
				ID: "step1",
				Compensation: &CompensationAction{
					Type: CompensationTypeSkip,
				},
			},
			expectErr: false,
		},
		{
			name: "invalid custom compensation - missing action",
			step: StepConfig{
				ID: "step1",
				Compensation: &CompensationAction{
					Type: CompensationTypeCustom,
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.validateCompensation(&tt.step)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

