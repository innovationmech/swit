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

// Package checker provides quality checking functionality for switctl
package checker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// RegisterChecker registers a new checker with the given name
func (q *QualityChecker) RegisterChecker(name string, checker IndividualChecker) error {
	if name == "" {
		return fmt.Errorf("checker name cannot be empty")
	}

	if checker == nil {
		return fmt.Errorf("checker cannot be nil")
	}

	if _, exists := q.checkers[name]; exists {
		return fmt.Errorf("checker '%s' already registered", name)
	}

	q.checkers[name] = checker
	return nil
}

// UnregisterChecker removes a checker by name
func (q *QualityChecker) UnregisterChecker(name string) error {
	if _, exists := q.checkers[name]; !exists {
		return fmt.Errorf("checker '%s' not found", name)
	}

	delete(q.checkers, name)
	delete(q.results, name)
	return nil
}

// ListCheckers returns a list of all registered checker names
func (q *QualityChecker) ListCheckers() []string {
	names := make([]string, 0, len(q.checkers))
	for name := range q.checkers {
		names = append(names, name)
	}
	return names
}

// RunAllChecks executes all registered checkers and returns a summary
func (q *QualityChecker) RunAllChecks(ctx context.Context) (*CheckerSummary, error) {
	if len(q.checkers) == 0 {
		return &CheckerSummary{
			Results:   make(map[string]interfaces.CheckResult),
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}, nil
	}

	startTime := time.Now()

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, q.timeout)
	defer cancel()

	// Create a semaphore to limit concurrent goroutines
	semaphore := make(chan struct{}, q.maxGoroutines)

	// Results collection
	results := make(map[string]interfaces.CheckResult)
	var resultsMutex sync.Mutex
	var wg sync.WaitGroup

	// Error collection
	var errors []error
	var errorsMutex sync.Mutex

	// Execute all checkers
	for name, checker := range q.checkers {
		wg.Add(1)

		go func(checkerName string, c IndividualChecker) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check cache first if enabled
			if q.enableCache {
				if cachedResult, exists := q.results[checkerName]; exists {
					resultsMutex.Lock()
					results[checkerName] = cachedResult
					resultsMutex.Unlock()
					return
				}
			}

			// Execute the check
			result, err := c.Check(ctxWithTimeout)

			if err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					errorsMutex.Lock()
					errors = append(errors, err)
					errorsMutex.Unlock()
					return
				}

				// For other errors, create an error result
				result = interfaces.CheckResult{
					Name:     checkerName,
					Status:   interfaces.CheckStatusError,
					Message:  err.Error(),
					Duration: 0,
				}
			}

			// Store result
			resultsMutex.Lock()
			results[checkerName] = result
			if q.enableCache {
				q.results[checkerName] = result
			}
			resultsMutex.Unlock()
		}(name, checker)
	}

	wg.Wait()

	endTime := time.Now()

	// Check for context cancellation or timeout errors
	if len(errors) > 0 {
		for _, err := range errors {
			if err == context.Canceled || err == context.DeadlineExceeded {
				return &CheckerSummary{
					Results:   results,
					StartTime: startTime,
					EndTime:   endTime,
					Duration:  endTime.Sub(startTime),
				}, err
			}
		}
	}

	// Aggregate results
	summary := q.aggregateResults(results)
	summary.StartTime = startTime
	summary.EndTime = endTime
	summary.Duration = endTime.Sub(startTime)

	return summary, nil
}

// RunSpecificCheck executes a specific checker by name
func (q *QualityChecker) RunSpecificCheck(ctx context.Context, name string) (interfaces.CheckResult, error) {
	checker, exists := q.checkers[name]
	if !exists {
		return interfaces.CheckResult{}, fmt.Errorf("checker '%s' not found", name)
	}

	// Check cache first if enabled
	if q.enableCache {
		if cachedResult, exists := q.results[name]; exists {
			return cachedResult, nil
		}
	}

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, q.timeout)
	defer cancel()

	result, err := checker.Check(ctxWithTimeout)
	if err != nil {
		return interfaces.CheckResult{}, err
	}

	// Cache result if enabled
	if q.enableCache {
		q.results[name] = result
	}

	return result, nil
}

// aggregateResults creates a summary from individual check results
func (q *QualityChecker) aggregateResults(results map[string]interfaces.CheckResult) *CheckerSummary {
	summary := &CheckerSummary{
		Results: results,
	}

	totalScore := 0
	maxScore := 0
	totalDuration := time.Duration(0)

	for _, result := range results {
		summary.TotalChecks++

		switch result.Status {
		case interfaces.CheckStatusPass:
			summary.PassedChecks++
		case interfaces.CheckStatusFail:
			summary.FailedChecks++
		case interfaces.CheckStatusWarning:
			summary.WarningChecks++
		case interfaces.CheckStatusSkip:
			summary.SkippedChecks++
		case interfaces.CheckStatusError:
			summary.ErrorChecks++
		}

		totalScore += result.Score
		maxScore += result.MaxScore
		totalDuration += result.Duration
	}

	summary.TotalScore = totalScore
	summary.MaxScore = maxScore
	summary.Duration = totalDuration

	if maxScore > 0 {
		summary.OverallScore = float64(totalScore) / float64(maxScore) * 100
	}

	return summary
}

// Close closes the quality checker and all registered checkers
func (q *QualityChecker) Close() error {
	var errors []error

	// Close all checkers
	for name, checker := range q.checkers {
		if err := checker.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close checker '%s': %w", name, err))
		}
	}

	// Clear all maps
	q.checkers = make(map[string]IndividualChecker)
	q.results = make(map[string]interfaces.CheckResult)

	if len(errors) > 0 {
		return fmt.Errorf("errors closing checkers: %v", errors)
	}

	return nil
}

// CheckCodeStyle implements interfaces.QualityChecker
func (q *QualityChecker) CheckCodeStyle() interfaces.CheckResult {
	ctx := context.Background()
	if result, err := q.RunSpecificCheck(ctx, "style"); err == nil {
		return result
	}

	return interfaces.CheckResult{
		Name:    "Code Style Check",
		Status:  interfaces.CheckStatusError,
		Message: "Style checker not available",
	}
}

// RunTests implements interfaces.QualityChecker
func (q *QualityChecker) RunTests() interfaces.TestResult {
	// This would typically integrate with a test runner
	// For now, return a default result
	return interfaces.TestResult{
		TotalTests:  0,
		PassedTests: 0,
		FailedTests: 0,
		Coverage:    0.0,
		Duration:    0,
	}
}

// CheckSecurity implements interfaces.QualityChecker
func (q *QualityChecker) CheckSecurity() interfaces.SecurityResult {
	// This would typically integrate with a security scanner
	// For now, return a default result
	return interfaces.SecurityResult{
		Issues:   []interfaces.SecurityIssue{},
		Severity: "none",
		Score:    100,
		MaxScore: 100,
		Scanned:  0,
		Duration: 0,
	}
}

// CheckCoverage implements interfaces.QualityChecker
func (q *QualityChecker) CheckCoverage() interfaces.CoverageResult {
	// This would typically integrate with a coverage tool
	// For now, return a default result
	return interfaces.CoverageResult{
		TotalLines:   0,
		CoveredLines: 0,
		Coverage:     0.0,
		Threshold:    80.0,
		Packages:     []interfaces.PackageCoverageResult{},
		Duration:     0,
	}
}

// ValidateConfig implements interfaces.QualityChecker
func (q *QualityChecker) ValidateConfig() interfaces.ValidationResult {
	ctx := context.Background()
	if result, err := q.RunSpecificCheck(ctx, "config"); err == nil {
		return interfaces.ValidationResult{
			Valid:    result.Status == interfaces.CheckStatusPass,
			Errors:   []interfaces.ValidationError{},
			Warnings: []interfaces.ValidationError{},
			Duration: result.Duration,
		}
	}

	return interfaces.ValidationResult{
		Valid:    false,
		Errors:   []interfaces.ValidationError{},
		Warnings: []interfaces.ValidationError{},
		Duration: 0,
	}
}

// CheckInterfaces implements interfaces.QualityChecker
func (q *QualityChecker) CheckInterfaces() interfaces.CheckResult {
	ctx := context.Background()
	if result, err := q.RunSpecificCheck(ctx, "interfaces"); err == nil {
		return result
	}

	return interfaces.CheckResult{
		Name:    "Interface Check",
		Status:  interfaces.CheckStatusError,
		Message: "Interface checker not available",
	}
}

// CheckImports implements interfaces.QualityChecker
func (q *QualityChecker) CheckImports() interfaces.CheckResult {
	ctx := context.Background()
	if result, err := q.RunSpecificCheck(ctx, "imports"); err == nil {
		return result
	}

	return interfaces.CheckResult{
		Name:    "Import Check",
		Status:  interfaces.CheckStatusError,
		Message: "Import checker not available",
	}
}

// CheckCopyright implements interfaces.QualityChecker
func (q *QualityChecker) CheckCopyright() interfaces.CheckResult {
	ctx := context.Background()
	if result, err := q.RunSpecificCheck(ctx, "copyright"); err == nil {
		return result
	}

	return interfaces.CheckResult{
		Name:    "Copyright Check",
		Status:  interfaces.CheckStatusError,
		Message: "Copyright checker not available",
	}
}
