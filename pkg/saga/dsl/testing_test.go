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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewTester(t *testing.T) {
	tests := []struct {
		name string
		opts *TesterOptions
		want string
	}{
		{
			name: "default options",
			opts: nil,
			want: "text",
		},
		{
			name: "custom format",
			opts: &TesterOptions{Format: "json"},
			want: "json",
		},
		{
			name: "yaml format",
			opts: &TesterOptions{Format: "yaml"},
			want: "yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := NewTester(tt.opts)
			if tester == nil {
				t.Fatal("NewTester returned nil")
			}
			if tester.options.Format != tt.want {
				t.Errorf("Format = %v, want %v", tester.options.Format, tt.want)
			}
		})
	}
}

func TestTestFile_ValidSaga(t *testing.T) {
	// Create a temporary valid YAML file
	tmpFile := createTempFile(t, "valid-saga.yaml", `
saga:
  id: test-saga
  name: Test Saga
  description: A test saga
  version: "1.0.0"
  mode: orchestration
  timeout: 30s

steps:
  - id: step1
    name: First Step
    type: service
    action:
      service:
        name: test-service
        method: POST
    timeout: 10s
    compensation:
      type: automatic
`)
	defer os.Remove(tmpFile)

	tester := NewTester(&TesterOptions{Format: "text"})
	result := tester.TestFile(tmpFile)

	if result.Status != TestStatusPass {
		t.Errorf("Expected status pass, got %v", result.Status)
		for _, err := range result.Errors {
			t.Logf("Error: %s", err)
		}
	}

	if result.SagaID != "test-saga" {
		t.Errorf("Expected saga_id 'test-saga', got %v", result.SagaID)
	}

	if result.StepCount != 1 {
		t.Errorf("Expected 1 step, got %d", result.StepCount)
	}
}

func TestTestFile_InvalidSyntax(t *testing.T) {
	// Create a temporary file with invalid YAML
	tmpFile := createTempFile(t, "invalid-syntax.yaml", `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    invalid_yaml: [unclosed bracket
`)
	defer os.Remove(tmpFile)

	tester := NewTester(&TesterOptions{Format: "text"})
	result := tester.TestFile(tmpFile)

	if result.Status != TestStatusFail {
		t.Errorf("Expected status fail, got %v", result.Status)
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors, got none")
	}
}

func TestTestFile_ValidationErrors(t *testing.T) {
	// Create a file with validation errors
	tmpFile := createTempFile(t, "validation-errors.yaml", `
saga:
  id: ""
  name: Test Saga

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: ""
        method: POST
`)
	defer os.Remove(tmpFile)

	tester := NewTester(&TesterOptions{Format: "text"})
	result := tester.TestFile(tmpFile)

	if result.Status != TestStatusFail {
		t.Errorf("Expected status fail, got %v", result.Status)
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors, got none")
	}
}

func TestTestFile_WithWarnings(t *testing.T) {
	tmpFile := createTempFile(t, "warnings.yaml", `
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
`)
	defer os.Remove(tmpFile)

	tester := NewTester(&TesterOptions{
		Format:     "text",
		StrictMode: false,
	})
	result := tester.TestFile(tmpFile)

	if result.Status != TestStatusPass {
		t.Errorf("Expected status pass (non-strict), got %v", result.Status)
	}

	if len(result.Warnings) == 0 {
		t.Error("Expected warnings, got none")
	}
}

func TestTestFile_StrictModeWithWarnings(t *testing.T) {
	tmpFile := createTempFile(t, "strict-warnings.yaml", `
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
`)
	defer os.Remove(tmpFile)

	tester := NewTester(&TesterOptions{
		Format:     "text",
		StrictMode: true,
	})
	result := tester.TestFile(tmpFile)

	if result.Status != TestStatusFail {
		t.Errorf("Expected status fail in strict mode with warnings, got %v", result.Status)
	}
}

func TestTestFile_WithDryRun(t *testing.T) {
	tmpFile := createTempFile(t, "dry-run.yaml", `
saga:
  id: test-saga
  name: Test Saga
  mode: orchestration
  timeout: 1m

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: service1
        method: POST
    timeout: 10s
    compensation:
      type: automatic

  - id: step2
    name: Step 2
    type: service
    dependencies:
      - step1
    action:
      service:
        name: service2
        method: POST
    timeout: 10s
    compensation:
      type: automatic
`)
	defer os.Remove(tmpFile)

	tester := NewTester(&TesterOptions{
		Format: "text",
		DryRun: true,
	})
	result := tester.TestFile(tmpFile)

	if result.Status != TestStatusPass {
		t.Errorf("Expected status pass, got %v", result.Status)
		for _, err := range result.Errors {
			t.Logf("Error: %s", err)
		}
	}

	if result.DryRunResults == nil {
		t.Fatal("Expected dry-run results, got nil")
	}

	if len(result.DryRunResults.ExecutionOrder) != 2 {
		t.Errorf("Expected 2 steps in execution order, got %d", len(result.DryRunResults.ExecutionOrder))
	}

	if result.DryRunResults.ExecutionOrder[0] != "step1" {
		t.Errorf("Expected first step to be 'step1', got %v", result.DryRunResults.ExecutionOrder[0])
	}

	if result.DryRunResults.ExecutionOrder[1] != "step2" {
		t.Errorf("Expected second step to be 'step2', got %v", result.DryRunResults.ExecutionOrder[1])
	}

	if len(result.DryRunResults.CompensationOrder) != 2 {
		t.Errorf("Expected 2 steps in compensation order, got %d", len(result.DryRunResults.CompensationOrder))
	}

	if result.DryRunResults.CompensationOrder[0] != "step2" {
		t.Errorf("Expected first compensation to be 'step2', got %v", result.DryRunResults.CompensationOrder[0])
	}
}

func TestTestFile_ParallelSteps(t *testing.T) {
	tmpFile := createTempFile(t, "parallel.yaml", `
saga:
  id: test-saga
  name: Test Saga with Parallel Steps

steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: service1
        method: POST
    compensation:
      type: automatic

  - id: step2
    name: Step 2 (parallel with step1)
    type: service
    action:
      service:
        name: service2
        method: POST
    compensation:
      type: automatic

  - id: step3
    name: Step 3 (depends on step1 and step2)
    type: service
    dependencies:
      - step1
      - step2
    action:
      service:
        name: service3
        method: POST
    compensation:
      type: automatic
`)
	defer os.Remove(tmpFile)

	tester := NewTester(&TesterOptions{
		Format: "text",
		DryRun: true,
	})
	result := tester.TestFile(tmpFile)

	if result.Status != TestStatusPass {
		t.Errorf("Expected status pass, got %v", result.Status)
	}

	if result.DryRunResults == nil {
		t.Fatal("Expected dry-run results, got nil")
	}

	// step1 and step2 should be at level 0 (can run in parallel)
	if len(result.DryRunResults.ParallelSteps) == 0 {
		t.Error("Expected parallel steps to be identified")
	}
}

func TestTestFiles_Multiple(t *testing.T) {
	// Create multiple test files
	tmpFile1 := createTempFile(t, "test1.yaml", `
saga:
  id: saga1
  name: Saga 1
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: svc1
        method: POST
    compensation:
      type: automatic
`)

	tmpFile2 := createTempFile(t, "test2.yaml", `
saga:
  id: saga2
  name: Saga 2
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: svc2
        method: POST
    compensation:
      type: automatic
`)

	defer os.Remove(tmpFile1)
	defer os.Remove(tmpFile2)

	tester := NewTester(&TesterOptions{Format: "text"})
	suite := tester.TestFiles([]string{tmpFile1, tmpFile2})

	if suite.TotalFiles != 2 {
		t.Errorf("Expected 2 total files, got %d", suite.TotalFiles)
	}

	if suite.PassedFiles != 2 {
		t.Errorf("Expected 2 passed files, got %d", suite.PassedFiles)
	}

	if suite.FailedFiles != 0 {
		t.Errorf("Expected 0 failed files, got %d", suite.FailedFiles)
	}
}

func TestTestFiles_WithFailures(t *testing.T) {
	tmpFile1 := createTempFile(t, "valid.yaml", `
saga:
  id: saga1
  name: Saga 1
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: svc1
        method: POST
    compensation:
      type: automatic
`)

	tmpFile2 := createTempFile(t, "invalid.yaml", `
saga:
  id: ""
  name: Saga 2
steps: []
`)

	defer os.Remove(tmpFile1)
	defer os.Remove(tmpFile2)

	tester := NewTester(&TesterOptions{Format: "text"})
	suite := tester.TestFiles([]string{tmpFile1, tmpFile2})

	if suite.PassedFiles != 1 {
		t.Errorf("Expected 1 passed file, got %d", suite.PassedFiles)
	}

	if suite.FailedFiles != 1 {
		t.Errorf("Expected 1 failed file, got %d", suite.FailedFiles)
	}
}

func TestTestFiles_FailFast(t *testing.T) {
	tmpFile1 := createTempFile(t, "invalid1.yaml", `saga: invalid`)
	tmpFile2 := createTempFile(t, "invalid2.yaml", `saga: invalid`)

	defer os.Remove(tmpFile1)
	defer os.Remove(tmpFile2)

	tester := NewTester(&TesterOptions{
		Format:   "text",
		FailFast: true,
	})
	suite := tester.TestFiles([]string{tmpFile1, tmpFile2})

	// Should stop after first failure
	if len(suite.Results) != 1 {
		t.Errorf("Expected 1 result (fail-fast), got %d", len(suite.Results))
	}
}

func TestPrintResult_TextFormat(t *testing.T) {
	result := &TestResult{
		File:      "test.yaml",
		Status:    TestStatusPass,
		Duration:  100 * time.Millisecond,
		SagaID:    "test-saga",
		SagaName:  "Test Saga",
		StepCount: 2,
		Warnings:  []string{"warning 1", "warning 2"},
	}

	var buf bytes.Buffer
	tester := NewTester(&TesterOptions{
		Format:       "text",
		OutputWriter: &buf,
	})

	err := tester.PrintResult(result)
	if err != nil {
		t.Fatalf("PrintResult failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test.yaml") {
		t.Error("Output should contain filename")
	}
	if !strings.Contains(output, "pass") {
		t.Error("Output should contain status")
	}
	if !strings.Contains(output, "test-saga") {
		t.Error("Output should contain saga ID")
	}
}

func TestPrintResult_JSONFormat(t *testing.T) {
	result := &TestResult{
		File:      "test.yaml",
		Status:    TestStatusPass,
		Duration:  100 * time.Millisecond,
		SagaID:    "test-saga",
		SagaName:  "Test Saga",
		StepCount: 2,
	}

	var buf bytes.Buffer
	tester := NewTester(&TesterOptions{
		Format:       "json",
		OutputWriter: &buf,
	})

	err := tester.PrintResult(result)
	if err != nil {
		t.Fatalf("PrintResult failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"file"`) {
		t.Error("JSON output should contain 'file' field")
	}
	if !strings.Contains(output, `"status"`) {
		t.Error("JSON output should contain 'status' field")
	}
}

func TestPrintResult_YAMLFormat(t *testing.T) {
	result := &TestResult{
		File:      "test.yaml",
		Status:    TestStatusPass,
		Duration:  100 * time.Millisecond,
		SagaID:    "test-saga",
		SagaName:  "Test Saga",
		StepCount: 2,
	}

	var buf bytes.Buffer
	tester := NewTester(&TesterOptions{
		Format:       "yaml",
		OutputWriter: &buf,
	})

	err := tester.PrintResult(result)
	if err != nil {
		t.Fatalf("PrintResult failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "file:") {
		t.Error("YAML output should contain 'file:' field")
	}
	if !strings.Contains(output, "status:") {
		t.Error("YAML output should contain 'status:' field")
	}
}

func TestPrintSuite_TextFormat(t *testing.T) {
	suite := &TestSuite{
		TotalFiles:   3,
		PassedFiles:  2,
		FailedFiles:  1,
		SkippedFiles: 0,
		Duration:     500 * time.Millisecond,
		Results: []*TestResult{
			{File: "test1.yaml", Status: TestStatusPass},
			{File: "test2.yaml", Status: TestStatusPass},
			{File: "test3.yaml", Status: TestStatusFail, Errors: []string{"error"}},
		},
	}

	var buf bytes.Buffer
	tester := NewTester(&TesterOptions{
		Format:       "text",
		OutputWriter: &buf,
	})

	err := tester.PrintSuite(suite)
	if err != nil {
		t.Fatalf("PrintSuite failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Total Files: 3") {
		t.Error("Output should contain total files count")
	}
	if !strings.Contains(output, "Passed: 2") {
		t.Error("Output should contain passed count")
	}
	if !strings.Contains(output, "Failed: 1") {
		t.Error("Output should contain failed count")
	}
}

func TestCheckWarnings(t *testing.T) {
	tests := []struct {
		name        string
		def         *SagaDefinition
		wantCount   int
		wantContain string
	}{
		{
			name: "no warnings",
			def: &SagaDefinition{
				Saga: SagaConfig{
					ID:          "test",
					Name:        "Test",
					Description: "Description",
					Version:     "1.0.0",
				},
				Steps: []StepConfig{
					{
						ID:          "step1",
						Name:        "Step 1",
						Description: "Step description",
						Type:        StepTypeService,
						Timeout:     Duration(10 * time.Second),
						Compensation: &CompensationAction{
							Type: CompensationTypeAutomatic,
						},
						RetryPolicy: &RetryPolicy{
							Type:        "fixed_delay",
							MaxAttempts: 3,
						},
						Action: ActionConfig{
							Service: &ServiceConfig{
								Name:   "svc",
								Method: "POST",
							},
						},
					},
				},
			},
			wantCount: 0,
		},
		{
			name: "missing description",
			def: &SagaDefinition{
				Saga: SagaConfig{
					ID:   "test",
					Name: "Test",
				},
				Steps: []StepConfig{
					{
						ID:   "step1",
						Name: "Step 1",
						Type: StepTypeService,
						Action: ActionConfig{
							Service: &ServiceConfig{
								Name:   "svc",
								Method: "POST",
							},
						},
					},
				},
			},
			wantCount:   6, // saga desc, saga version, step desc, step compensation, step timeout, step retry
			wantContain: "no description",
		},
		{
			name: "missing compensation",
			def: &SagaDefinition{
				Saga: SagaConfig{
					ID:          "test",
					Name:        "Test",
					Description: "Desc",
					Version:     "1.0",
				},
				Steps: []StepConfig{
					{
						ID:          "step1",
						Name:        "Step 1",
						Description: "Step",
						Type:        StepTypeService,
						Action: ActionConfig{
							Service: &ServiceConfig{
								Name:   "svc",
								Method: "POST",
							},
						},
					},
				},
			},
			wantCount:   3, // compensation, timeout, retry
			wantContain: "no compensation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := NewTester(nil)
			warnings := tester.checkWarnings(tt.def)

			if len(warnings) != tt.wantCount {
				t.Errorf("Expected %d warnings, got %d", tt.wantCount, len(warnings))
				for i, w := range warnings {
					t.Logf("Warning %d: %s", i+1, w)
				}
			}

			if tt.wantContain != "" {
				found := false
				for _, w := range warnings {
					if strings.Contains(w, tt.wantContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected warning containing '%s', but not found", tt.wantContain)
				}
			}
		})
	}
}

func TestTopologicalSort(t *testing.T) {
	tests := []struct {
		name    string
		steps   []StepConfig
		want    []string
		wantErr bool
	}{
		{
			name: "linear dependencies",
			steps: []StepConfig{
				{ID: "step1"},
				{ID: "step2", Dependencies: []string{"step1"}},
				{ID: "step3", Dependencies: []string{"step2"}},
			},
			want:    []string{"step1", "step2", "step3"},
			wantErr: false,
		},
		{
			name: "parallel steps",
			steps: []StepConfig{
				{ID: "step1"},
				{ID: "step2"},
				{ID: "step3", Dependencies: []string{"step1", "step2"}},
			},
			want:    []string{"step1", "step2", "step3"}, // or step2, step1, step3
			wantErr: false,
		},
		{
			name: "no dependencies",
			steps: []StepConfig{
				{ID: "step1"},
				{ID: "step2"},
			},
			want:    []string{"step1", "step2"}, // or any order
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := NewTester(nil)
			got, err := tester.topologicalSort(tt.steps)

			if (err != nil) != tt.wantErr {
				t.Errorf("topologicalSort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(got) != len(tt.want) {
				t.Errorf("topologicalSort() got %d steps, want %d", len(got), len(tt.want))
			}
		})
	}
}

func TestFormatFile(t *testing.T) {
	tmpFile := createTempFile(t, "format-test.yaml", `
saga:
  id: test
  name: Test
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: svc
        method: POST
    compensation:
      type: automatic
`)
	defer os.Remove(tmpFile)

	err := FormatFile(tmpFile)
	if err != nil {
		t.Fatalf("FormatFile failed: %v", err)
	}

	// Verify file still exists and is valid
	_, err = os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("File does not exist after formatting: %v", err)
	}

	// Verify it can be parsed
	_, err = ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("File cannot be parsed after formatting: %v", err)
	}
}

func TestCheckSyntax(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid syntax",
			content: `
saga:
  id: test
  name: Test
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: svc
        method: POST
    compensation:
      type: automatic
`,
			wantErr: false,
		},
		{
			name: "invalid syntax",
			content: `
saga:
  id: test
  invalid yaml [
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempFile(t, "syntax-test.yaml", tt.content)
			defer os.Remove(tmpFile)

			err := CheckSyntax(tmpFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid file",
			content: `
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
      type: automatic
`,
			wantErr: false,
		},
		{
			name: "invalid file - empty saga id",
			content: `
saga:
  id: ""
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: svc
        method: POST
    compensation:
      type: automatic
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempFile(t, "validate-test.yaml", tt.content)
			defer os.Remove(tmpFile)

			err := ValidateFile(tmpFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to create temporary test files
func createTempFile(t *testing.T, name, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, name)

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return tmpFile
}
