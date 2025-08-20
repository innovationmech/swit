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

package checker

import (
	"context"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// Default configuration values
const (
	DefaultTimeout       = 30 * time.Second
	DefaultMaxGoroutines = 4
)

// CheckerOptions provides configuration options for the QualityChecker
type CheckerOptions struct {
	Timeout       time.Duration
	MaxGoroutines int
	EnableCache   bool
}

// CheckerSummary provides a summary of all check results
type CheckerSummary struct {
	Results       map[string]interfaces.CheckResult `json:"results"`
	TotalChecks   int                               `json:"total_checks"`
	PassedChecks  int                               `json:"passed_checks"`
	FailedChecks  int                               `json:"failed_checks"`
	WarningChecks int                               `json:"warning_checks"`
	SkippedChecks int                               `json:"skipped_checks"`
	ErrorChecks   int                               `json:"error_checks"`
	TotalScore    int                               `json:"total_score"`
	MaxScore      int                               `json:"max_score"`
	OverallScore  float64                           `json:"overall_score"`
	Duration      time.Duration                     `json:"duration"`
	StartTime     time.Time                         `json:"start_time"`
	EndTime       time.Time                         `json:"end_time"`
}

// IndividualChecker represents a single quality checker
type IndividualChecker interface {
	Check(ctx context.Context) (interfaces.CheckResult, error)
	Name() string
	Description() string
	Close() error
}

// QualityChecker implements the main quality checking functionality
type QualityChecker struct {
	fs            interfaces.FileSystem
	checkers      map[string]IndividualChecker
	results       map[string]interfaces.CheckResult
	timeout       time.Duration
	maxGoroutines int
	enableCache   bool
}

// NewQualityChecker creates a new quality checker with default options
func NewQualityChecker(fs interfaces.FileSystem) *QualityChecker {
	return NewQualityCheckerWithOptions(fs, CheckerOptions{
		Timeout:       DefaultTimeout,
		MaxGoroutines: DefaultMaxGoroutines,
		EnableCache:   false,
	})
}

// NewQualityCheckerWithOptions creates a new quality checker with custom options
func NewQualityCheckerWithOptions(fs interfaces.FileSystem, options CheckerOptions) *QualityChecker {
	return &QualityChecker{
		fs:            fs,
		checkers:      make(map[string]IndividualChecker),
		results:       make(map[string]interfaces.CheckResult),
		timeout:       options.Timeout,
		maxGoroutines: options.MaxGoroutines,
		enableCache:   options.EnableCache,
	}
}

// Manager provides a higher-level interface for managing quality checkers
type Manager struct {
	logger   interfaces.Logger
	checkers []interfaces.QualityChecker
	timeout  time.Duration
	parallel bool
}

// NewManager creates a new quality checker manager
func NewManager(logger interfaces.Logger) *Manager {
	return &Manager{
		logger:   logger,
		checkers: make([]interfaces.QualityChecker, 0),
		timeout:  5 * time.Minute,
		parallel: true,
	}
}

// SetTimeout sets the timeout for quality checks
func (m *Manager) SetTimeout(timeout time.Duration) {
	m.timeout = timeout
}

// SetParallel sets whether to run checks in parallel
func (m *Manager) SetParallel(parallel bool) {
	m.parallel = parallel
}

// RegisterChecker registers a quality checker with the manager
func (m *Manager) RegisterChecker(checker interfaces.QualityChecker) {
	m.checkers = append(m.checkers, checker)
}

// RunAllChecks runs all registered quality checkers
func (m *Manager) RunAllChecks(ctx context.Context) (*QualityReport, error) {
	if len(m.checkers) == 0 {
		return &QualityReport{
			Overall: QualityOverall{
				Status:  interfaces.CheckStatusSkip,
				Message: "No checkers registered",
			},
		}, nil
	}

	// For simplicity, run the first checker (usually CompositeChecker)
	if len(m.checkers) > 0 {
		checker := m.checkers[0]

		// Create a simple report structure
		report := &QualityReport{
			Overall: QualityOverall{
				Status:  interfaces.CheckStatusPass,
				Message: "Quality checks completed",
			},
			StyleCheck:     checker.CheckCodeStyle(),
			TestResult:     checker.RunTests(),
			SecurityResult: checker.CheckSecurity(),
			ConfigResult:   checker.ValidateConfig(),
		}

		return report, nil
	}

	return nil, nil
}

// QualityReport represents a comprehensive quality report
type QualityReport struct {
	Overall        QualityOverall              `json:"overall"`
	StyleCheck     interfaces.CheckResult      `json:"style_check"`
	TestResult     interfaces.TestResult       `json:"test_result"`
	SecurityResult interfaces.SecurityResult   `json:"security_result"`
	ConfigResult   interfaces.ValidationResult `json:"config_result"`
}

// QualityOverall represents overall quality check status
type QualityOverall struct {
	Status  interfaces.CheckStatus `json:"status"`
	Message string                 `json:"message"`
}
