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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// TestResult represents the result of testing a DSL file.
type TestResult struct {
	// File is the path to the tested file
	File string `json:"file" yaml:"file"`

	// Status is the test status: pass, fail, skip
	Status TestStatus `json:"status" yaml:"status"`

	// Duration is the time taken to test
	Duration time.Duration `json:"duration" yaml:"duration"`

	// SagaID is the ID of the tested Saga
	SagaID string `json:"saga_id,omitempty" yaml:"saga_id,omitempty"`

	// SagaName is the name of the tested Saga
	SagaName string `json:"saga_name,omitempty" yaml:"saga_name,omitempty"`

	// StepCount is the number of steps in the Saga
	StepCount int `json:"step_count,omitempty" yaml:"step_count,omitempty"`

	// Errors contains any validation or parsing errors
	Errors []string `json:"errors,omitempty" yaml:"errors,omitempty"`

	// Warnings contains warnings (non-critical issues)
	Warnings []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`

	// DryRunResults contains results from dry-run simulation (if executed)
	DryRunResults *DryRunResult `json:"dry_run_results,omitempty" yaml:"dry_run_results,omitempty"`
}

// TestStatus represents the test result status.
type TestStatus string

const (
	// TestStatusPass indicates the test passed
	TestStatusPass TestStatus = "pass"

	// TestStatusFail indicates the test failed
	TestStatusFail TestStatus = "fail"

	// TestStatusSkip indicates the test was skipped
	TestStatusSkip TestStatus = "skip"
)

// DryRunResult contains the results of a dry-run simulation.
type DryRunResult struct {
	// ExecutionOrder shows the order in which steps would be executed
	ExecutionOrder []string `json:"execution_order" yaml:"execution_order"`

	// ParallelSteps lists steps that could be executed in parallel
	ParallelSteps map[string][]string `json:"parallel_steps,omitempty" yaml:"parallel_steps,omitempty"`

	// CompensationOrder shows the order in which compensations would be executed
	CompensationOrder []string `json:"compensation_order" yaml:"compensation_order"`

	// EstimatedDuration is the estimated execution time (if timeouts are specified)
	EstimatedDuration time.Duration `json:"estimated_duration" yaml:"estimated_duration"`

	// Warnings contains warnings from the dry-run
	Warnings []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// TestSuite represents a collection of test results.
type TestSuite struct {
	// TotalFiles is the total number of files tested
	TotalFiles int `json:"total_files" yaml:"total_files"`

	// PassedFiles is the number of files that passed
	PassedFiles int `json:"passed_files" yaml:"passed_files"`

	// FailedFiles is the number of files that failed
	FailedFiles int `json:"failed_files" yaml:"failed_files"`

	// SkippedFiles is the number of files that were skipped
	SkippedFiles int `json:"skipped_files" yaml:"skipped_files"`

	// Duration is the total time taken for all tests
	Duration time.Duration `json:"duration" yaml:"duration"`

	// Results contains individual test results
	Results []*TestResult `json:"results" yaml:"results"`
}

// Tester provides testing functionality for DSL files.
type Tester struct {
	parser    *Parser
	validator *Validator
	options   *TesterOptions
}

// TesterOptions configures the Tester behavior.
type TesterOptions struct {
	// StrictMode enables strict validation (treats warnings as errors)
	StrictMode bool

	// DryRun enables dry-run simulation
	DryRun bool

	// Format specifies output format: json, yaml, text
	Format string

	// Verbose enables verbose output
	Verbose bool

	// FailFast stops testing after the first failure
	FailFast bool

	// OutputWriter is where results are written (defaults to stdout)
	OutputWriter io.Writer
}

// NewTester creates a new DSL tester with the given options.
func NewTester(opts *TesterOptions) *Tester {
	if opts == nil {
		opts = &TesterOptions{
			Format:       "text",
			OutputWriter: os.Stdout,
		}
	}

	if opts.OutputWriter == nil {
		opts.OutputWriter = os.Stdout
	}

	return &Tester{
		parser:    NewParser(),
		validator: NewValidator(),
		options:   opts,
	}
}

// TestFile tests a single DSL file.
func (t *Tester) TestFile(path string) *TestResult {
	start := time.Now()

	result := &TestResult{
		File:     path,
		Status:   TestStatusPass,
		Duration: 0,
	}

	// Parse the file
	def, err := t.parser.ParseFile(path)
	if err != nil {
		result.Status = TestStatusFail
		result.Errors = append(result.Errors, fmt.Sprintf("Parse error: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	// Record basic information
	result.SagaID = def.Saga.ID
	result.SagaName = def.Saga.Name
	result.StepCount = len(def.Steps)

	// Validate the definition
	if err := t.validator.Validate(def); err != nil {
		result.Status = TestStatusFail
		if validationErrors, ok := err.(ValidationErrors); ok {
			for _, ve := range validationErrors {
				result.Errors = append(result.Errors, ve.Error())
			}
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("Validation error: %v", err))
		}
		result.Duration = time.Since(start)
		return result
	}

	// Check for warnings
	warnings := t.checkWarnings(def)
	if len(warnings) > 0 {
		result.Warnings = warnings
		if t.options.StrictMode {
			result.Status = TestStatusFail
			result.Errors = append(result.Errors, "Warnings treated as errors in strict mode")
		}
	}

	// Perform dry-run if enabled
	if t.options.DryRun && result.Status == TestStatusPass {
		dryRunResult := t.performDryRun(def)
		result.DryRunResults = dryRunResult
		if len(dryRunResult.Warnings) > 0 && t.options.StrictMode {
			result.Status = TestStatusFail
			result.Errors = append(result.Errors, "Dry-run warnings treated as errors in strict mode")
		}
	}

	result.Duration = time.Since(start)
	return result
}

// TestFiles tests multiple DSL files and returns a test suite.
func (t *Tester) TestFiles(paths []string) *TestSuite {
	start := time.Now()

	suite := &TestSuite{
		TotalFiles: len(paths),
		Results:    make([]*TestResult, 0, len(paths)),
	}

	for _, path := range paths {
		result := t.TestFile(path)
		suite.Results = append(suite.Results, result)

		switch result.Status {
		case TestStatusPass:
			suite.PassedFiles++
		case TestStatusFail:
			suite.FailedFiles++
			if t.options.FailFast {
				suite.Duration = time.Since(start)
				return suite
			}
		case TestStatusSkip:
			suite.SkippedFiles++
		}
	}

	suite.Duration = time.Since(start)
	return suite
}

// TestDirectory tests all DSL files in a directory (recursively).
func (t *Tester) TestDirectory(dir string, pattern string) *TestSuite {
	if pattern == "" {
		pattern = "*.yaml"
	}

	files, err := t.findFiles(dir, pattern)
	if err != nil {
		// Return a suite with error
		return &TestSuite{
			TotalFiles:  0,
			FailedFiles: 1,
			Results: []*TestResult{
				{
					File:   dir,
					Status: TestStatusFail,
					Errors: []string{fmt.Sprintf("Failed to find files: %v", err)},
				},
			},
		}
	}

	return t.TestFiles(files)
}

// findFiles finds all files matching the pattern in a directory.
func (t *Tester) findFiles(dir string, pattern string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}

		if matched {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// checkWarnings checks for potential issues that are not errors.
func (t *Tester) checkWarnings(def *SagaDefinition) []string {
	var warnings []string

	// Check if Saga has no description
	if def.Saga.Description == "" {
		warnings = append(warnings, "Saga has no description")
	}

	// Check if Saga has no version
	if def.Saga.Version == "" {
		warnings = append(warnings, "Saga has no version specified")
	}

	// Check for steps without descriptions
	for i, step := range def.Steps {
		if step.Description == "" {
			warnings = append(warnings, fmt.Sprintf("Step '%s' (index %d) has no description", step.ID, i))
		}

		// Check for steps without compensation
		if step.Compensation == nil {
			warnings = append(warnings, fmt.Sprintf("Step '%s' has no compensation defined", step.ID))
		}

		// Check for steps without timeout
		if step.Timeout == 0 && def.Saga.Timeout == 0 {
			warnings = append(warnings, fmt.Sprintf("Step '%s' has no timeout (and no global timeout)", step.ID))
		}

		// Check for steps without retry policy
		if step.RetryPolicy == nil && def.GlobalRetryPolicy == nil {
			warnings = append(warnings, fmt.Sprintf("Step '%s' has no retry policy (and no global retry policy)", step.ID))
		}
	}

	// Check for unused global configurations
	if def.GlobalRetryPolicy != nil {
		hasStepWithoutRetry := false
		for _, step := range def.Steps {
			if step.RetryPolicy == nil {
				hasStepWithoutRetry = true
				break
			}
		}
		if !hasStepWithoutRetry {
			warnings = append(warnings, "Global retry policy is defined but all steps have their own retry policies")
		}
	}

	return warnings
}

// performDryRun simulates the execution of a Saga and returns the results.
func (t *Tester) performDryRun(def *SagaDefinition) *DryRunResult {
	result := &DryRunResult{
		ExecutionOrder:    make([]string, 0),
		ParallelSteps:     make(map[string][]string),
		CompensationOrder: make([]string, 0),
		Warnings:          make([]string, 0),
	}

	// Build dependency graph
	stepMap := make(map[string]*StepConfig)
	for i := range def.Steps {
		stepMap[def.Steps[i].ID] = &def.Steps[i]
	}

	// Determine execution order using topological sort
	executionOrder, err := t.topologicalSort(def.Steps)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to determine execution order: %v", err))
		return result
	}

	result.ExecutionOrder = executionOrder

	// Determine compensation order (reverse of execution)
	for i := len(executionOrder) - 1; i >= 0; i-- {
		result.CompensationOrder = append(result.CompensationOrder, executionOrder[i])
	}

	// Identify parallel execution opportunities
	levels := t.identifyParallelLevels(def.Steps)
	for level, steps := range levels {
		if len(steps) > 1 {
			result.ParallelSteps[fmt.Sprintf("level_%d", level)] = steps
		}
	}

	// Estimate execution duration
	result.EstimatedDuration = t.estimateDuration(def, stepMap)

	// Check for potential issues
	if len(result.ExecutionOrder) != len(def.Steps) {
		result.Warnings = append(result.Warnings, "Not all steps are reachable in the execution graph")
	}

	return result
}

// topologicalSort performs topological sort on steps based on dependencies.
func (t *Tester) topologicalSort(steps []StepConfig) ([]string, error) {
	// Build adjacency list and in-degree map
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, step := range steps {
		if _, exists := inDegree[step.ID]; !exists {
			inDegree[step.ID] = 0
		}

		for _, dep := range step.Dependencies {
			graph[dep] = append(graph[dep], step.ID)
			inDegree[step.ID]++
		}
	}

	// Kahn's algorithm for topological sort
	var queue []string
	for stepID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, stepID)
		}
	}

	var result []string
	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Decrease in-degree for neighbors
		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check if all nodes were processed
	if len(result) != len(steps) {
		return nil, fmt.Errorf("cycle detected in step dependencies")
	}

	return result, nil
}

// identifyParallelLevels identifies steps that can be executed in parallel.
func (t *Tester) identifyParallelLevels(steps []StepConfig) map[int][]string {
	levels := make(map[int][]string)
	stepLevel := make(map[string]int)

	// Build dependency map
	depMap := make(map[string][]string)
	for _, step := range steps {
		depMap[step.ID] = step.Dependencies
	}

	// Calculate level for each step (BFS-like approach)
	var calculateLevel func(string) int
	calculateLevel = func(stepID string) int {
		if level, exists := stepLevel[stepID]; exists {
			return level
		}

		maxDepLevel := -1
		for _, dep := range depMap[stepID] {
			depLevel := calculateLevel(dep)
			if depLevel > maxDepLevel {
				maxDepLevel = depLevel
			}
		}

		level := maxDepLevel + 1
		stepLevel[stepID] = level
		return level
	}

	// Assign levels to all steps
	for _, step := range steps {
		level := calculateLevel(step.ID)
		levels[level] = append(levels[level], step.ID)
	}

	return levels
}

// estimateDuration estimates the total execution duration based on timeouts.
func (t *Tester) estimateDuration(def *SagaDefinition, stepMap map[string]*StepConfig) time.Duration {
	if def.Saga.Timeout > 0 {
		return time.Duration(def.Saga.Timeout)
	}

	// Calculate based on step timeouts
	var totalDuration time.Duration
	for _, step := range def.Steps {
		if step.Timeout > 0 {
			totalDuration += time.Duration(step.Timeout)
		} else if step.Action.Service != nil && step.Action.Service.Timeout > 0 {
			totalDuration += time.Duration(step.Action.Service.Timeout)
		}
	}

	return totalDuration
}

// PrintResult prints a single test result.
func (t *Tester) PrintResult(result *TestResult) error {
	switch t.options.Format {
	case "json":
		return t.printJSON(result)
	case "yaml":
		return t.printYAML(result)
	default:
		return t.printText(result)
	}
}

// PrintSuite prints a test suite.
func (t *Tester) PrintSuite(suite *TestSuite) error {
	switch t.options.Format {
	case "json":
		return t.printJSON(suite)
	case "yaml":
		return t.printYAML(suite)
	default:
		return t.printSuiteText(suite)
	}
}

// printJSON prints data as JSON.
func (t *Tester) printJSON(data interface{}) error {
	encoder := json.NewEncoder(t.options.OutputWriter)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// printYAML prints data as YAML.
func (t *Tester) printYAML(data interface{}) error {
	encoder := yaml.NewEncoder(t.options.OutputWriter)
	defer encoder.Close()
	return encoder.Encode(data)
}

// printText prints a test result in text format.
func (t *Tester) printText(result *TestResult) error {
	w := t.options.OutputWriter

	// Print header
	fmt.Fprintf(w, "File: %s\n", result.File)
	fmt.Fprintf(w, "Status: %s\n", result.Status)
	fmt.Fprintf(w, "Duration: %s\n", result.Duration)

	if result.SagaID != "" {
		fmt.Fprintf(w, "Saga ID: %s\n", result.SagaID)
		fmt.Fprintf(w, "Saga Name: %s\n", result.SagaName)
		fmt.Fprintf(w, "Step Count: %d\n", result.StepCount)
	}

	// Print errors
	if len(result.Errors) > 0 {
		fmt.Fprintf(w, "\nErrors:\n")
		for i, err := range result.Errors {
			fmt.Fprintf(w, "  %d. %s\n", i+1, err)
		}
	}

	// Print warnings
	if len(result.Warnings) > 0 {
		fmt.Fprintf(w, "\nWarnings:\n")
		for i, warn := range result.Warnings {
			fmt.Fprintf(w, "  %d. %s\n", i+1, warn)
		}
	}

	// Print dry-run results
	if result.DryRunResults != nil {
		fmt.Fprintf(w, "\nDry Run Results:\n")
		fmt.Fprintf(w, "  Execution Order: %s\n", strings.Join(result.DryRunResults.ExecutionOrder, " -> "))
		fmt.Fprintf(w, "  Compensation Order: %s\n", strings.Join(result.DryRunResults.CompensationOrder, " -> "))
		fmt.Fprintf(w, "  Estimated Duration: %s\n", result.DryRunResults.EstimatedDuration)

		if len(result.DryRunResults.ParallelSteps) > 0 {
			fmt.Fprintf(w, "  Parallel Execution Opportunities:\n")
			for level, steps := range result.DryRunResults.ParallelSteps {
				fmt.Fprintf(w, "    %s: %s\n", level, strings.Join(steps, ", "))
			}
		}

		if len(result.DryRunResults.Warnings) > 0 {
			fmt.Fprintf(w, "  Warnings:\n")
			for i, warn := range result.DryRunResults.Warnings {
				fmt.Fprintf(w, "    %d. %s\n", i+1, warn)
			}
		}
	}

	fmt.Fprintln(w, "")
	return nil
}

// printSuiteText prints a test suite in text format.
func (t *Tester) printSuiteText(suite *TestSuite) error {
	w := t.options.OutputWriter

	// Print summary
	fmt.Fprintf(w, "Test Suite Summary\n")
	fmt.Fprintf(w, "==================\n")
	fmt.Fprintf(w, "Total Files: %d\n", suite.TotalFiles)
	fmt.Fprintf(w, "Passed: %d\n", suite.PassedFiles)
	fmt.Fprintf(w, "Failed: %d\n", suite.FailedFiles)
	fmt.Fprintf(w, "Skipped: %d\n", suite.SkippedFiles)
	fmt.Fprintf(w, "Duration: %s\n\n", suite.Duration)

	// Print individual results if verbose
	if t.options.Verbose {
		fmt.Fprintf(w, "Individual Results\n")
		fmt.Fprintf(w, "==================\n\n")
		for _, result := range suite.Results {
			if err := t.printText(result); err != nil {
				return err
			}
		}
	} else {
		// Print only failed tests
		hasFailures := false
		for _, result := range suite.Results {
			if result.Status == TestStatusFail {
				if !hasFailures {
					fmt.Fprintf(w, "Failed Tests\n")
					fmt.Fprintf(w, "============\n\n")
					hasFailures = true
				}
				if err := t.printText(result); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// FormatFile formats a DSL YAML file (rewrites it with consistent formatting).
func FormatFile(path string) error {
	// Parse the file
	def, err := ParseFile(path)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Marshal back to YAML
	data, err := yaml.Marshal(def)
	if err != nil {
		return fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// CheckSyntax performs basic syntax checking on a DSL file.
func CheckSyntax(path string) error {
	// Try to parse the file
	_, err := ParseFile(path)
	return err
}

// ValidateFile validates a DSL file.
func ValidateFile(path string) error {
	// Parse the file
	def, err := ParseFile(path)
	if err != nil {
		return err
	}

	// Validate the definition
	validator := NewValidator()
	return validator.Validate(def)
}

