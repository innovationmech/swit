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
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseBytes_EmptyFile tests parsing of completely empty file
func TestParseBytes_EmptyFile(t *testing.T) {
	_, err := ParseBytes([]byte{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty YAML content")
}

// TestParseBytes_OnlyWhitespace tests parsing of file with only whitespace
func TestParseBytes_OnlyWhitespace(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"spaces", "   "},
		{"tabs", "\t\t\t"},
		{"newlines", "\n\n\n"},
		{"mixed whitespace", "  \t\n  \t\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.content))
			require.Error(t, err)
			// Empty YAML or unmarshal error expected
		})
	}
}

// TestParseBytes_MinimalValid tests parsing of minimal valid DSL
func TestParseBytes_MinimalValid(t *testing.T) {
	yamlContent := `
saga:
  id: min
  name: Minimal

steps:
  - id: s1
    name: Step
    type: service
    action:
      service:
        name: svc
        method: GET
`

	def, err := ParseBytes([]byte(yamlContent))
	require.NoError(t, err)
	require.NotNil(t, def)

	assert.Equal(t, "min", def.Saga.ID)
	assert.Equal(t, "Minimal", def.Saga.Name)
	assert.Len(t, def.Steps, 1)
	assert.Equal(t, "s1", def.Steps[0].ID)
}

// TestParseBytes_LargeDSL tests parsing of large DSL with many steps
func TestParseBytes_LargeDSL(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(`
saga:
  id: large-saga
  name: Large Saga with Many Steps
  version: "1.0.0"

steps:
`)

	// Generate 100 steps
	for i := 1; i <= 100; i++ {
		buf.WriteString(fmt.Sprintf(`  - id: step%d
    name: Step %d
    type: service
    action:
      service:
        name: service-%d
        method: POST
`, i, i, i))
		if i > 1 {
			buf.WriteString(fmt.Sprintf("    dependencies: [step%d]\n", i-1))
		}
	}

	start := time.Now()
	def, err := ParseBytes(buf.Bytes())
	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, def)
	assert.Equal(t, "large-saga", def.Saga.ID)
	assert.Len(t, def.Steps, 100)

	// Should parse in reasonable time (< 1 second)
	assert.Less(t, duration, 1*time.Second,
		"parsing 100 steps took %v, expected < 1s", duration)

	t.Logf("Parsed 100 steps in %v", duration)
}

// TestParseBytes_DeeplyNestedDependencies tests parsing of deeply nested step dependencies
func TestParseBytes_DeeplyNestedDependencies(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(`
saga:
  id: deep-saga
  name: Deeply Nested Dependencies

steps:
`)

	// Create a chain of 50 dependent steps
	depth := 50
	for i := 1; i <= depth; i++ {
		buf.WriteString(fmt.Sprintf(`  - id: step%d
    name: Step %d
    type: service
    action:
      service:
        name: service-%d
        method: POST
`, i, i, i))
		if i > 1 {
			buf.WriteString(fmt.Sprintf("    dependencies: [step%d]\n", i-1))
		}
	}

	def, err := ParseBytes(buf.Bytes())
	require.NoError(t, err)
	require.NotNil(t, def)
	assert.Len(t, def.Steps, depth)

	// Verify dependency chain
	for i := 1; i < depth; i++ {
		assert.Contains(t, def.Steps[i].Dependencies, fmt.Sprintf("step%d", i))
	}
}

// TestParseBytes_SpecialCharacters tests parsing of special characters in various fields
func TestParseBytes_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
		checkFunc   func(*testing.T, *SagaDefinition)
	}{
		{
			name: "special chars in description",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
  description: "Special chars: @#$%^&*()_+-=[]{}|;':,.<>?/"

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			expectError: false,
			checkFunc: func(t *testing.T, def *SagaDefinition) {
				assert.Contains(t, def.Saga.Description, "@#$%^&*()")
			},
		},
		{
			name: "quotes in strings",
			yaml: `
saga:
  id: test-saga
  name: 'Saga with "quotes"'

steps:
  - id: step1
    name: "Step with 'quotes'"
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			expectError: false,
			checkFunc: func(t *testing.T, def *SagaDefinition) {
				assert.Contains(t, def.Saga.Name, "quotes")
				assert.Contains(t, def.Steps[0].Name, "quotes")
			},
		},
		{
			name: "newlines in multiline strings",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
  description: |
    This is a multiline
    description with
    multiple lines

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			expectError: false,
			checkFunc: func(t *testing.T, def *SagaDefinition) {
				assert.Contains(t, def.Saga.Description, "\n")
				assert.Contains(t, def.Saga.Description, "multiline")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := ParseBytes([]byte(tt.yaml))
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.checkFunc != nil {
					tt.checkFunc(t, def)
				}
			}
		})
	}
}

// TestParseBytes_UnicodeSupport tests parsing of Unicode characters
func TestParseBytes_UnicodeSupport(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "Chinese characters",
			yaml: `
saga:
  id: test-saga
  name: 测试Saga
  description: 这是一个中文描述

steps:
  - id: step1
    name: 步骤1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
		},
		{
			name: "Japanese characters",
			yaml: `
saga:
  id: test-saga
  name: テストサガ
  description: これは日本語の説明です

steps:
  - id: step1
    name: ステップ1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
		},
		{
			name: "Arabic characters",
			yaml: `
saga:
  id: test-saga
  name: اختبار ساغا
  description: هذا وصف باللغة العربية

steps:
  - id: step1
    name: خطوة 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
		},
		{
			name: "Emoji",
			yaml: `
saga:
  id: test-saga
  name: Test Saga 🚀
  description: Processing order 📦 with payment 💳

steps:
  - id: step1
    name: Validate ✅
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
		},
		{
			name: "Mixed Unicode",
			yaml: `
saga:
  id: test-saga
  name: Test测试サガ
  description: English, 中文, 日本語, العربية

steps:
  - id: step1
    name: Step 1 步骤 ステップ
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := ParseBytes([]byte(tt.yaml))
			require.NoError(t, err)
			require.NotNil(t, def)

			// Verify Unicode content is preserved
			assert.NotEmpty(t, def.Saga.Name)
			assert.NotEmpty(t, def.Steps[0].Name)
		})
	}
}

// TestParseBytes_MaxFieldLengths tests parsing with maximum field lengths
func TestParseBytes_MaxFieldLengths(t *testing.T) {
	// Test very long description
	longDescription := strings.Repeat("a", 10000)
	yamlContent := fmt.Sprintf(`
saga:
  id: test-saga
  name: Test Saga
  description: %s

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`, longDescription)

	def, err := ParseBytes([]byte(yamlContent))
	require.NoError(t, err)
	assert.Len(t, def.Saga.Description, 10000)
}

// TestParseBytes_ComplexDependencyGraph tests parsing of complex dependency graphs
func TestParseBytes_ComplexDependencyGraph(t *testing.T) {
	yamlContent := `
saga:
  id: complex-saga
  name: Complex Dependency Graph

steps:
  - id: step1
    name: Step 1
    type: service
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
    dependencies: [step1]
    action:
      service:
        name: service-3
        method: POST

  - id: step4
    name: Step 4
    type: service
    dependencies: [step2, step3]
    action:
      service:
        name: service-4
        method: POST

  - id: step5
    name: Step 5
    type: service
    dependencies: [step1, step4]
    action:
      service:
        name: service-5
        method: POST
`

	def, err := ParseBytes([]byte(yamlContent))
	require.NoError(t, err)
	require.NotNil(t, def)

	// Verify complex dependencies
	assert.Len(t, def.Steps[0].Dependencies, 0)                                    // step1: no deps
	assert.Equal(t, []string{"step1"}, def.Steps[1].Dependencies)                  // step2: depends on step1
	assert.Equal(t, []string{"step1"}, def.Steps[2].Dependencies)                  // step3: depends on step1
	assert.ElementsMatch(t, []string{"step2", "step3"}, def.Steps[3].Dependencies) // step4: depends on step2, step3
	assert.ElementsMatch(t, []string{"step1", "step4"}, def.Steps[4].Dependencies) // step5: depends on step1, step4
}

// TestParseFile_VeryLargeFile tests parsing of very large DSL file
func TestParseFile_VeryLargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large-saga.yaml")

	var buf bytes.Buffer
	buf.WriteString(`
saga:
  id: very-large-saga
  name: Very Large Saga
  version: "1.0.0"
  description: |
    This is a very large saga definition
    used for performance testing.

steps:
`)

	// Generate 500 steps with various configurations
	for i := 1; i <= 500; i++ {
		buf.WriteString(fmt.Sprintf(`  - id: step%d
    name: Step %d
    description: This is step number %d in the saga
    type: service
    timeout: 30s
    action:
      service:
        name: service-%d
        method: POST
        path: /api/v1/step%d
        timeout: 20s
`, i, i, i, i, i))

		if i > 1 && i%10 != 0 {
			buf.WriteString(fmt.Sprintf("    dependencies: [step%d]\n", i-1))
		}

		// Add retry policy for some steps
		if i%5 == 0 {
			buf.WriteString(`    retry_policy:
      type: exponential_backoff
      max_attempts: 3
      initial_delay: 1s
      max_delay: 30s
      multiplier: 2.0
`)
		}

		// Add compensation for some steps
		if i%7 == 0 {
			buf.WriteString(`    compensation:
      type: custom
      strategy: sequential
      action:
        service:
          name: service-%d
          method: DELETE
` + fmt.Sprintf("          path: /api/v1/step%d\n", i))
		}
	}

	err := os.WriteFile(testFile, buf.Bytes(), 0644)
	require.NoError(t, err)

	// Measure file size
	fileInfo, err := os.Stat(testFile)
	require.NoError(t, err)
	t.Logf("Test file size: %d bytes (%.2f KB)", fileInfo.Size(), float64(fileInfo.Size())/1024)

	// Parse and measure time
	start := time.Now()
	def, err := ParseFile(testFile, WithBasePath(tmpDir))
	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, def)
	assert.Equal(t, "very-large-saga", def.Saga.ID)
	assert.Len(t, def.Steps, 500)

	// Performance requirement: should parse in < 5 seconds
	assert.Less(t, duration, 5*time.Second,
		"parsing 500 steps took %v, expected < 5s", duration)

	t.Logf("Parsed 500 steps (%.2f KB) in %v", float64(fileInfo.Size())/1024, duration)
}

// TestParseReader_ErrorReading tests error handling when reader fails
func TestParseReader_ErrorReading(t *testing.T) {
	// Create a reader that always returns an error
	errReader := &errorReader{err: fmt.Errorf("read error")}

	_, err := ParseReader(errReader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read from reader")
	assert.Contains(t, err.Error(), "read error")
}

// errorReader is a reader that always returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

// TestParseBytes_AllStepTypes tests parsing all supported step types
func TestParseBytes_AllStepTypes(t *testing.T) {
	yamlContent := `
saga:
  id: all-types-saga
  name: All Step Types

steps:
  - id: service-step
    name: Service Step
    type: service
    action:
      service:
        name: test-service
        method: POST

  - id: http-step
    name: HTTP Step
    type: http
    action:
      service:
        name: http-service
        endpoint: http://example.com
        method: GET

  - id: grpc-step
    name: gRPC Step
    type: grpc
    action:
      service:
        name: grpc-service
        endpoint: example.com:9090
        method: TestMethod

  - id: function-step
    name: Function Step
    type: function
    action:
      function:
        name: test-function
        handler: pkg.Handler

  - id: message-step
    name: Message Step
    type: message
    action:
      message:
        topic: test-topic
        broker: kafka

  - id: custom-step
    name: Custom Step
    type: custom
    action:
      custom:
        type: my-custom-type
        config:
          key: value
`

	def, err := ParseBytes([]byte(yamlContent))
	require.NoError(t, err)
	require.NotNil(t, def)

	assert.Len(t, def.Steps, 6)
	assert.Equal(t, StepTypeService, def.Steps[0].Type)
	assert.Equal(t, StepTypeHTTP, def.Steps[1].Type)
	assert.Equal(t, StepTypeGRPC, def.Steps[2].Type)
	assert.Equal(t, StepTypeFunction, def.Steps[3].Type)
	assert.Equal(t, StepTypeMessage, def.Steps[4].Type)
	assert.Equal(t, StepTypeCustom, def.Steps[5].Type)
}

// TestParseBytes_AllCompensationTypes tests parsing all compensation types
func TestParseBytes_AllCompensationTypes(t *testing.T) {
	yamlContent := `
saga:
  id: comp-types-saga
  name: Compensation Types

steps:
  - id: step1
    name: Skip Compensation
    type: service
    action:
      service:
        name: service-1
        method: POST
    compensation:
      type: skip

  - id: step2
    name: Automatic Compensation
    type: service
    action:
      service:
        name: service-2
        method: POST
    compensation:
      type: automatic
      strategy: sequential

  - id: step3
    name: Custom Compensation
    type: service
    action:
      service:
        name: service-3
        method: POST
    compensation:
      type: custom
      strategy: parallel
      timeout: 1m
      max_attempts: 3
      action:
        service:
          name: service-3
          method: DELETE
      on_failure:
        action: retry
        max_retries: 2
`

	def, err := ParseBytes([]byte(yamlContent))
	require.NoError(t, err)
	require.NotNil(t, def)

	assert.Len(t, def.Steps, 3)
	assert.Equal(t, CompensationTypeSkip, def.Steps[0].Compensation.Type)
	assert.Equal(t, CompensationTypeAutomatic, def.Steps[1].Compensation.Type)
	assert.Equal(t, CompensationTypeCustom, def.Steps[2].Compensation.Type)
	assert.Equal(t, CompensationStrategyParallel, def.Steps[2].Compensation.Strategy)
}

// TestParseBytes_ConcurrentParsing tests concurrent parsing is safe
func TestParseBytes_ConcurrentParsing(t *testing.T) {
	yamlContent := `
saga:
  id: concurrent-saga
  name: Concurrent Test

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`

	// Run 100 concurrent parses
	concurrency := 100
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := ParseBytes([]byte(yamlContent))
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check no errors occurred
	for err := range errors {
		t.Errorf("concurrent parsing error: %v", err)
	}
}

// TestParseError_Unwrap tests ParseError unwrapping
func TestParseError_Unwrap(t *testing.T) {
	causeErr := fmt.Errorf("underlying error")
	parseErr := &ParseError{
		Message: "parse failed",
		File:    "test.yaml",
		Line:    10,
		Cause:   causeErr,
	}

	unwrapped := parseErr.Unwrap()
	assert.Equal(t, causeErr, unwrapped)
}

// TestWithMaxIncludeDepth tests WithMaxIncludeDepth option
func TestWithMaxIncludeDepth(t *testing.T) {
	parser := NewParser(WithMaxIncludeDepth(5))
	assert.Equal(t, 5, parser.maxIncludeDepth)
}

// TestWithLogger tests WithLogger option
func TestWithLogger(t *testing.T) {
	customLogger := &testLogger{}
	parser := NewParser(WithLogger(customLogger))
	assert.Equal(t, customLogger, parser.logger)
}

// testLogger is a test implementation of Logger interface
type testLogger struct {
	messages []string
}

func (l *testLogger) Debug(msg string, args ...interface{}) {
	l.messages = append(l.messages, fmt.Sprintf("DEBUG: "+msg, args...))
}

func (l *testLogger) Info(msg string, args ...interface{}) {
	l.messages = append(l.messages, fmt.Sprintf("INFO: "+msg, args...))
}

func (l *testLogger) Warn(msg string, args ...interface{}) {
	l.messages = append(l.messages, fmt.Sprintf("WARN: "+msg, args...))
}

func (l *testLogger) Error(msg string, args ...interface{}) {
	l.messages = append(l.messages, fmt.Sprintf("ERROR: "+msg, args...))
}

// TestDefaultLogger_Error tests the Error method of defaultLogger
func TestDefaultLogger_Error(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-log-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	logger := &defaultLogger{
		logger: newTestStdLogger(tmpFile),
	}

	logger.Error("test error message", "key", "value")

	// Read the file contents
	tmpFile.Seek(0, 0)
	content, err := io.ReadAll(tmpFile)
	require.NoError(t, err)

	assert.Contains(t, string(content), "[ERROR]")
	assert.Contains(t, string(content), "test error message")
}

// newTestStdLogger creates a log.Logger that writes to the given writer
func newTestStdLogger(w io.Writer) *log.Logger {
	return log.New(w, "", log.LstdFlags)
}

// BenchmarkParseBytes_SmallBoundary benchmarks parsing small DSL
func BenchmarkParseBytes_SmallBoundary(b *testing.B) {
	yamlContent := []byte(`
saga:
  id: bench-saga
  name: Benchmark Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseBytes(yamlContent)
	}
}

// BenchmarkParseBytes_MediumBoundary benchmarks parsing medium-sized DSL
func BenchmarkParseBytes_MediumBoundary(b *testing.B) {
	var buf bytes.Buffer
	buf.WriteString(`
saga:
  id: bench-saga
  name: Benchmark Saga

steps:
`)

	for i := 1; i <= 20; i++ {
		buf.WriteString(fmt.Sprintf(`  - id: step%d
    name: Step %d
    type: service
    action:
      service:
        name: service-%d
        method: POST
`, i, i, i))
	}

	yamlContent := buf.Bytes()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseBytes(yamlContent)
	}
}

// BenchmarkParseBytes_ConcurrentBoundary benchmarks concurrent parsing
func BenchmarkParseBytes_ConcurrentBoundary(b *testing.B) {
	yamlContent := []byte(`
saga:
  id: bench-saga
  name: Benchmark Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = ParseBytes(yamlContent)
		}
	})
}
