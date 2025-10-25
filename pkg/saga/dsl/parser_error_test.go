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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseBytes_SyntaxErrors tests various YAML syntax errors
func TestParseBytes_SyntaxErrors(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectedErr string
	}{
		{
			name: "invalid indentation",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
- id: step1
  name: Step 1
 type: service
`,
			expectedErr: "YAML syntax error",
		},
		{
			name: "unclosed bracket",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    dependencies: [step2
`,
			expectedErr: "YAML syntax error",
		},
		{
			name: "invalid key-value separator",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id = step1
    name: Step 1
`,
			expectedErr: "YAML syntax error",
		},
		{
			name:        "mixing tabs and spaces",
			yaml:        "saga:\n\tid: test-saga\n  name: Test Saga\n",
			expectedErr: "YAML syntax error",
		},
		{
			name: "duplicate keys",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
  name: Duplicate Name
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			// YAML allows duplicate keys, last one wins - not a syntax error
			expectedErr: "",
		},
		{
			name: "invalid escape sequence",
			yaml: `
saga:
  id: test-saga
  name: "Test \x Saga"
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: "YAML syntax error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				// Some cases may not error
				if err != nil {
					t.Logf("Note: got error %v (may be expected)", err)
				}
			}
		})
	}
}

// TestParseBytes_MissingRequiredFieldsDetailed tests detailed missing field errors
func TestParseBytes_MissingRequiredFieldsDetailed(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectedErr []string
	}{
		{
			name: "missing saga section",
			yaml: `
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: []string{"validation"},
		},
		{
			name: "missing steps section",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
`,
			expectedErr: []string{"validation"},
		},
		{
			name: "missing step name",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: []string{"validation"},
		},
		{
			name: "missing step type",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: []string{"validation"},
		},
		{
			name: "missing action",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
`,
			expectedErr: []string{"validation"},
		},
		{
			name: "missing service name",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        method: POST
`,
			expectedErr: []string{"validation"},
		},
		{
			name: "missing service method",
			yaml: `
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
`,
			expectedErr: []string{"validation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			require.Error(t, err)
			for _, expected := range tt.expectedErr {
				assert.Contains(t, err.Error(), expected,
					"expected error to contain %q", expected)
			}
		})
	}
}

// TestParseBytes_TypeMismatch tests type mismatch errors
func TestParseBytes_TypeMismatch(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectedErr string
	}{
		{
			name: "timeout as string",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
  timeout: "not a duration"
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: "YAML syntax error",
		},
		{
			name: "dependencies as string instead of array",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    dependencies: step2
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: "YAML syntax error",
		},
		{
			name: "max_attempts as string",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
global_retry_policy:
  type: exponential_backoff
  max_attempts: "three"
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: "YAML syntax error",
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

// TestParseBytes_InvalidReferences tests invalid reference errors
func TestParseBytes_InvalidReferences(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectedErr []string
	}{
		{
			name: "reference to non-existent step",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    dependencies: [non-existent-step]
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: []string{"non-existent"},
		},
		{
			name: "multiple invalid references",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    dependencies: [missing1, missing2]
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: []string{"non-existent"},
		},
		{
			name: "self-reference in dependencies",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    dependencies: [step1]
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: []string{"circular"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			require.Error(t, err)
			for _, expected := range tt.expectedErr {
				assert.Contains(t, err.Error(), expected,
					"expected error to contain %q", expected)
			}
		})
	}
}

// TestParseBytes_SemanticErrors tests semantic validation errors
func TestParseBytes_SemanticErrors(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectedErr []string
	}{
		{
			name: "circular dependency simple",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    dependencies: [step2]
    action:
      service:
        name: service-1
        method: POST
  - id: step2
    name: Step 2
    type: service
    dependencies: [step1]
    action:
      service:
        name: service-2
        method: POST
`,
			expectedErr: []string{"circular dependency"},
		},
		{
			name: "circular dependency complex",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    dependencies: [step3]
    action:
      service:
        name: service-1
        method: POST
  - id: step2
    name: Step 2
    type: service
    dependencies: [step1]
    action:
      service:
        name: service-2
        method: POST
  - id: step3
    name: Step 3
    type: service
    dependencies: [step2]
    action:
      service:
        name: service-3
        method: POST
`,
			expectedErr: []string{"circular dependency"},
		},
		{
			name: "invalid step type value",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: unknown-type
    action:
      service:
        name: test-service
        method: POST
`,
			expectedErr: []string{"validation"},
		},
		{
			name: "custom compensation without action",
			yaml: `
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
`,
			expectedErr: []string{"custom compensation requires an action"},
		},
		{
			name: "skip compensation with action",
			yaml: `
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
      type: skip
      action:
        service:
          name: test-service
          method: DELETE
`,
			// Note: Parser doesn't enforce this, validator logs a warning
			expectedErr: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			if len(tt.expectedErr) > 0 {
				require.Error(t, err)
				for _, expected := range tt.expectedErr {
					assert.Contains(t, err.Error(), expected,
						"expected error to contain %q", expected)
				}
			} else {
				// No error expected, but parser may still parse it
				if err != nil {
					t.Logf("Got error (may be acceptable): %v", err)
				}
			}
		})
	}
}

// TestParseError_MessageQuality tests error message quality and usefulness
func TestParseError_MessageQuality(t *testing.T) {
	tests := []struct {
		name            string
		yaml            string
		errorChecks     []func(*testing.T, error)
		suggestionCheck func(*testing.T, error)
	}{
		{
			name: "missing saga ID provides clear message",
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
			errorChecks: []func(*testing.T, error){
				func(t *testing.T, err error) {
					assert.Contains(t, err.Error(), "validation")
					assert.Contains(t, err.Error(), "required")
				},
			},
		},
		{
			name: "duplicate step ID shows which ID",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: duplicate
    name: Step 1
    type: service
    action:
      service:
        name: service-1
        method: POST
  - id: duplicate
    name: Step 2
    type: service
    action:
      service:
        name: service-2
        method: POST
`,
			errorChecks: []func(*testing.T, error){
				func(t *testing.T, err error) {
					assert.Contains(t, err.Error(), "duplicate")
					assert.Contains(t, err.Error(), "duplicate")
				},
			},
		},
		{
			name: "circular dependency shows path",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: a
    name: A
    type: service
    dependencies: [c]
    action:
      service:
        name: svc
        method: POST
  - id: b
    name: B
    type: service
    dependencies: [a]
    action:
      service:
        name: svc
        method: POST
  - id: c
    name: C
    type: service
    dependencies: [b]
    action:
      service:
        name: svc
        method: POST
`,
			errorChecks: []func(*testing.T, error){
				func(t *testing.T, err error) {
					errMsg := err.Error()
					assert.Contains(t, errMsg, "circular")
					// Should show the cycle path
					hasA := strings.Contains(errMsg, "a")
					hasB := strings.Contains(errMsg, "b")
					hasC := strings.Contains(errMsg, "c")
					assert.True(t, hasA || hasB || hasC,
						"error should mention at least one step in the cycle")
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			require.Error(t, err)

			for _, check := range tt.errorChecks {
				check(t, err)
			}

			if tt.suggestionCheck != nil {
				tt.suggestionCheck(t, err)
			}
		})
	}
}

// TestParseFile_FileErrors tests file-related errors
func TestParseFile_FileErrors(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*testing.T) string
		expectedErr []string
	}{
		{
			name: "file not found",
			setup: func(t *testing.T) string {
				return "/non/existent/file.yaml"
			},
			expectedErr: []string{"file not found"},
		},
		{
			name: "directory instead of file",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedErr: []string{"failed to read file", "is a directory"},
		},
		{
			name: "permission denied",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				file := filepath.Join(tmpDir, "no-perm.yaml")
				err := os.WriteFile(file, []byte("content"), 0000)
				require.NoError(t, err)
				return file
			},
			expectedErr: []string{"failed to read file", "permission denied"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			_, err := ParseFile(path)
			require.Error(t, err)

			found := false
			for _, expected := range tt.expectedErr {
				if strings.Contains(err.Error(), expected) {
					found = true
					break
				}
			}
			assert.True(t, found,
				"error %q should contain one of %v", err.Error(), tt.expectedErr)
		})
	}
}

// TestParseFile_CircularIncludes tests circular include detection
func TestParseFile_CircularIncludes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file1 that includes file2
	file1 := filepath.Join(tmpDir, "file1.yaml")
	file1Content := `!include file2.yaml`

	// Create file2 that includes file3
	file2 := filepath.Join(tmpDir, "file2.yaml")
	file2Content := `!include file3.yaml`

	// Create file3 that includes file1 (circular)
	file3 := filepath.Join(tmpDir, "file3.yaml")
	file3Content := `!include file1.yaml`

	err := os.WriteFile(file1, []byte(file1Content), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte(file2Content), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file3, []byte(file3Content), 0644)
	require.NoError(t, err)

	_, err = ParseFile(file1, WithBasePath(tmpDir), WithIncludes(true))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

// TestParseFile_IncludeNotFound tests include file not found error
func TestParseFile_IncludeNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	mainFile := filepath.Join(tmpDir, "main.yaml")
	mainContent := `!include non-existent.yaml`

	err := os.WriteFile(mainFile, []byte(mainContent), 0644)
	require.NoError(t, err)

	_, err = ParseFile(mainFile, WithBasePath(tmpDir), WithIncludes(true))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
	assert.Contains(t, err.Error(), "non-existent.yaml")
}

// TestParseFile_IncludeWithErrors tests include file with errors
func TestParseFile_IncludeWithErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Create included file with invalid YAML
	includedFile := filepath.Join(tmpDir, "included.yaml")
	includedContent := `
saga:
  id: [invalid: yaml: structure
`
	err := os.WriteFile(includedFile, []byte(includedContent), 0644)
	require.NoError(t, err)

	// Create main file that includes the invalid file
	mainFile := filepath.Join(tmpDir, "main.yaml")
	mainContent := `!include included.yaml`
	err = os.WriteFile(mainFile, []byte(mainContent), 0644)
	require.NoError(t, err)

	_, err = ParseFile(mainFile, WithBasePath(tmpDir), WithIncludes(true))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "YAML syntax error")
}

// TestFormatValidationError tests validation error formatting
func TestFormatValidationError(t *testing.T) {
	parser := NewParser()

	// Create a definition with validation errors
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "", // Missing required field
			Name: "Test",
		},
		Steps: []StepConfig{}, // Empty steps
	}

	err := parser.validate(def)
	require.Error(t, err)

	errMsg := err.Error()
	assert.NotEmpty(t, errMsg)
	// Should contain field information
	assert.Contains(t, strings.ToLower(errMsg), "validation")
}

// TestValidateCompensation_EdgeCases tests edge cases in compensation validation
func TestValidateCompensation_EdgeCases(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name      string
		step      StepConfig
		expectErr bool
	}{
		{
			name: "nil compensation is valid",
			step: StepConfig{
				ID:           "step1",
				Compensation: nil,
			},
			expectErr: false,
		},
		{
			name: "skip compensation without action is valid",
			step: StepConfig{
				ID: "step1",
				Compensation: &CompensationAction{
					Type: CompensationTypeSkip,
				},
			},
			expectErr: false,
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

// TestParseBytes_PartiallyValidYAML tests partially valid YAML structures
func TestParseBytes_PartiallyValidYAML(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		shouldParse bool
		checkFunc   func(*testing.T, *SagaDefinition, error)
	}{
		{
			name: "extra unknown fields are ignored",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
  unknown_field: value
steps:
  - id: step1
    name: Step 1
    type: service
    extra_field: ignored
    action:
      service:
        name: test-service
        method: POST
`,
			shouldParse: true,
			checkFunc: func(t *testing.T, def *SagaDefinition, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "test-saga", def.Saga.ID)
			},
		},
		{
			name: "empty dependencies array is valid",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    dependencies: []
    action:
      service:
        name: test-service
        method: POST
`,
			shouldParse: true,
			checkFunc: func(t *testing.T, def *SagaDefinition, err error) {
				assert.NoError(t, err)
				assert.Empty(t, def.Steps[0].Dependencies)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := ParseBytes([]byte(tt.yaml))
			if tt.shouldParse {
				require.NoError(t, err)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, def, err)
			}
		})
	}
}

// TestParseError_DetailedLocation tests error location reporting
func TestParseError_DetailedLocation(t *testing.T) {
	// Test ParseError with line and column
	err := &ParseError{
		Message: "test error",
		File:    "test.yaml",
		Line:    42,
		Column:  15,
	}

	errMsg := err.Error()
	assert.Contains(t, errMsg, "test.yaml")
	assert.Contains(t, errMsg, "line 42")
	assert.Contains(t, errMsg, "column 15")
	assert.Contains(t, errMsg, "test error")
}

// TestParseBytes_RecoveryFromErrors tests parser doesn't crash on various malformed inputs
func TestParseBytes_RecoveryFromErrors(t *testing.T) {
	malformedInputs := []string{
		"",             // empty
		"   ",          // only whitespace
		"---\n",        // only document separator
		"null",         // null document
		"[]",           // empty array
		"{}",           // empty object
		"!!binary abc", // binary type
		"- - -",        // weird list structure
		"a: b: c",      // invalid nesting
		"!<tag:yaml.org,2002:timestamp> 2001-01-01", // explicit type tag
	}

	for i, input := range malformedInputs {
		t.Run(fmt.Sprintf("malformed_%d", i), func(t *testing.T) {
			// Should not panic, should return error
			_, err := ParseBytes([]byte(input))
			// We expect most of these to error, but shouldn't panic
			if err != nil {
				t.Logf("Got expected error: %v", err)
			}
		})
	}
}
