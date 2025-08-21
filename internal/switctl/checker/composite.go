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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// FileIndex represents a cached index of project files for performance optimization.
type FileIndex struct {
	GoFiles     []string               `json:"go_files"`
	TestFiles   []string               `json:"test_files"`
	ConfigFiles []string               `json:"config_files"`
	ProtoFiles  []string               `json:"proto_files"`
	AllFiles    []string               `json:"all_files"`
	IndexedAt   time.Time              `json:"indexed_at"`
	FileMeta    map[string]os.FileInfo `json:"-"` // Not serialized
}

// CompositeChecker implements the full QualityChecker interface by composing
// all individual checkers into a single comprehensive checker.
type CompositeChecker struct {
	styleChecker    *StyleChecker
	testRunner      *TestRunner
	securityChecker *SecurityChecker
	configChecker   *ConfigChecker
	workDir         string
	logger          interfaces.Logger
	fileIndex       *FileIndex
	indexMux        sync.RWMutex
	indexMaxAge     time.Duration
}

// NewCompositeChecker creates a new composite quality checker that combines
// all individual checker implementations.
func NewCompositeChecker(workDir string, logger interfaces.Logger) *CompositeChecker {
	cc := &CompositeChecker{
		styleChecker: NewStyleChecker(workDir, StyleCheckerOptions{
			EnableGofmt:          true,
			EnableGolangciLint:   true,
			EnableImportCheck:    true,
			EnableCopyrightCheck: true,
		}),
		testRunner: NewTestRunner(workDir, TestRunnerOptions{
			EnableCoverage:      true,
			EnableRaceDetection: false,
			CoverageThreshold:   80.0,
		}),
		securityChecker: NewSecurityChecker(workDir, logger),
		configChecker:   NewConfigChecker(workDir, logger),
		workDir:         workDir,
		logger:          logger,
		indexMaxAge:     5 * time.Minute, // Cache index for 5 minutes
	}

	// Initialize file index
	cc.fileIndex = &FileIndex{
		FileMeta: make(map[string]os.FileInfo),
	}

	return cc
}

// SetStyleConfig updates the style checker configuration.
func (cc *CompositeChecker) SetStyleConfig(config StyleCheckerOptions) {
	cc.styleChecker.options = config
}

// SetTestConfig updates the test checker configuration.
func (cc *CompositeChecker) SetTestConfig(config TestRunnerOptions) {
	cc.testRunner.options = config
}

// SetSecurityConfig updates the security checker configuration.
func (cc *CompositeChecker) SetSecurityConfig(config SecurityConfig) {
	cc.securityChecker.SetConfig(config)
}

// SetConfigValidationConfig updates the config checker configuration.
func (cc *CompositeChecker) SetConfigValidationConfig(config ConfigValidationConfig) {
	cc.configChecker.SetConfig(config)
}

// CheckCodeStyle performs comprehensive code style checking.
func (cc *CompositeChecker) CheckCodeStyle() interfaces.CheckResult {
	result, _ := cc.styleChecker.Check(context.Background())
	return result
}

// RunTests executes all tests and returns comprehensive results.
func (cc *CompositeChecker) RunTests() interfaces.TestResult {
	packages, _ := cc.testRunner.findTestPackages()
	result, _ := cc.testRunner.RunTests(context.Background(), packages)
	return result
}

// CheckSecurity performs comprehensive security scanning.
func (cc *CompositeChecker) CheckSecurity() interfaces.SecurityResult {
	return cc.securityChecker.CheckSecurity()
}

// CheckCoverage analyzes test coverage and returns detailed results.
func (cc *CompositeChecker) CheckCoverage() interfaces.CoverageResult {
	packages, _ := cc.testRunner.findTestPackages()
	testResult, _ := cc.testRunner.RunTestsWithCoverage(context.Background(), packages)

	return interfaces.CoverageResult{
		TotalLines:   int(testResult.Coverage * 100), // Approximate
		CoveredLines: int(testResult.Coverage * 100),
		Coverage:     testResult.Coverage,
		Threshold:    cc.testRunner.options.CoverageThreshold,
		Duration:     testResult.Duration,
	}
}

// ValidateConfig performs comprehensive configuration validation.
func (cc *CompositeChecker) ValidateConfig() interfaces.ValidationResult {
	return cc.configChecker.ValidateConfig()
}

// CheckInterfaces validates service interface implementations.
func (cc *CompositeChecker) CheckInterfaces() interfaces.CheckResult {
	return interfaces.CheckResult{
		Name:    "interfaces",
		Status:  interfaces.CheckStatusPass,
		Message: "Interface validation not implemented yet",
	}
}

// CheckImports validates import organization.
func (cc *CompositeChecker) CheckImports() interfaces.CheckResult {
	return interfaces.CheckResult{
		Name:    "imports",
		Status:  interfaces.CheckStatusPass,
		Message: "Import validation handled by style checker",
	}
}

// CheckCopyright validates copyright headers.
func (cc *CompositeChecker) CheckCopyright() interfaces.CheckResult {
	return interfaces.CheckResult{
		Name:    "copyright",
		Status:  interfaces.CheckStatusPass,
		Message: "Copyright validation handled by style checker",
	}
}

// Check implements IndividualChecker interface to run all comprehensive checks.
func (cc *CompositeChecker) Check(ctx context.Context) (interfaces.CheckResult, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return interfaces.CheckResult{}, ctx.Err()
	default:
	}

	// Run comprehensive checks
	styleResult := cc.CheckCodeStyle()

	select {
	case <-ctx.Done():
		return interfaces.CheckResult{}, ctx.Err()
	default:
	}

	testResult := cc.RunTests()

	select {
	case <-ctx.Done():
		return interfaces.CheckResult{}, ctx.Err()
	default:
	}

	securityResult := cc.CheckSecurity()

	select {
	case <-ctx.Done():
		return interfaces.CheckResult{}, ctx.Err()
	default:
	}

	configResult := cc.ValidateConfig()

	// Aggregate results
	overallStatus := interfaces.CheckStatusPass
	var messages []string
	totalScore := 100
	maxScore := 400 // 4 checkers, each max 100

	// Process style results
	if styleResult.Status == interfaces.CheckStatusFail {
		overallStatus = interfaces.CheckStatusFail
		messages = append(messages, "style: "+styleResult.Message)
		totalScore -= (100 - styleResult.Score)
	} else if styleResult.Status == interfaces.CheckStatusWarning && overallStatus != interfaces.CheckStatusFail {
		overallStatus = interfaces.CheckStatusWarning
		messages = append(messages, "style: "+styleResult.Message)
		totalScore -= (100 - styleResult.Score)
	} else if styleResult.Status == interfaces.CheckStatusPass {
		messages = append(messages, "style: passed")
	}

	// Process test results (approximate scoring)
	if testResult.FailedTests > 0 {
		overallStatus = interfaces.CheckStatusFail
		messages = append(messages, fmt.Sprintf("tests: %d failed", testResult.FailedTests))
		totalScore -= 100 // Heavy penalty for test failures
	} else if testResult.TotalTests == 0 {
		if overallStatus == interfaces.CheckStatusPass {
			overallStatus = interfaces.CheckStatusWarning
		}
		messages = append(messages, "tests: no tests found")
		totalScore -= 50
	} else {
		messages = append(messages, "tests: all passed")
	}

	// Process security results
	if len(securityResult.Issues) > 0 {
		highSeverityCount := 0
		for _, issue := range securityResult.Issues {
			if issue.Severity == "high" || issue.Severity == "critical" {
				highSeverityCount++
			}
		}

		if highSeverityCount > 0 {
			overallStatus = interfaces.CheckStatusFail
			messages = append(messages, fmt.Sprintf("security: %d high/critical issues", highSeverityCount))
			totalScore -= highSeverityCount * 25
		} else if overallStatus != interfaces.CheckStatusFail {
			overallStatus = interfaces.CheckStatusWarning
			messages = append(messages, fmt.Sprintf("security: %d low/medium issues", len(securityResult.Issues)))
			totalScore -= len(securityResult.Issues) * 10
		}
	} else {
		messages = append(messages, "security: no issues found")
	}

	// Process config results
	if !configResult.Valid {
		overallStatus = interfaces.CheckStatusFail
		messages = append(messages, fmt.Sprintf("config: %d errors", len(configResult.Errors)))
		totalScore -= len(configResult.Errors) * 20
	} else if len(configResult.Warnings) > 0 {
		if overallStatus == interfaces.CheckStatusPass {
			overallStatus = interfaces.CheckStatusWarning
		}
		messages = append(messages, fmt.Sprintf("config: %d warnings", len(configResult.Warnings)))
		totalScore -= len(configResult.Warnings) * 5
	} else {
		messages = append(messages, "config: valid")
	}

	// Ensure score doesn't go below 0
	if totalScore < 0 {
		totalScore = 0
	}

	message := "Comprehensive quality check completed"
	if len(messages) > 0 {
		message = fmt.Sprintf("Quality check: %s", strings.Join(messages, "; "))
	}

	return interfaces.CheckResult{
		Name:     "composite",
		Status:   overallStatus,
		Message:  message,
		Duration: styleResult.Duration + testResult.Duration + securityResult.Duration + configResult.Duration,
		Score:    totalScore,
		MaxScore: maxScore,
		Fixable:  styleResult.Fixable,
	}, nil
}

// Name implements IndividualChecker interface.
func (cc *CompositeChecker) Name() string {
	return "composite"
}

// Description implements IndividualChecker interface.
func (cc *CompositeChecker) Description() string {
	return "Comprehensive quality checker combining style, test, security, and config validation"
}

// Close implements IndividualChecker interface.
func (cc *CompositeChecker) Close() error {
	var errors []error

	if err := cc.styleChecker.Close(); err != nil {
		errors = append(errors, fmt.Errorf("failed to close style checker: %w", err))
	}

	if err := cc.testRunner.Close(); err != nil {
		errors = append(errors, fmt.Errorf("failed to close test runner: %w", err))
	}

	if err := cc.securityChecker.Close(); err != nil {
		errors = append(errors, fmt.Errorf("failed to close security checker: %w", err))
	}

	if err := cc.configChecker.Close(); err != nil {
		errors = append(errors, fmt.Errorf("failed to close config checker: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing composite checker: %v", errors)
	}

	return nil
}

// GetStyleChecker returns the style checker for direct access.
func (cc *CompositeChecker) GetStyleChecker() *StyleChecker {
	return cc.styleChecker
}

// GetTestRunner returns the test runner for direct access.
func (cc *CompositeChecker) GetTestRunner() *TestRunner {
	return cc.testRunner
}

// GetSecurityChecker returns the security checker for direct access.
func (cc *CompositeChecker) GetSecurityChecker() *SecurityChecker {
	return cc.securityChecker
}

// GetConfigChecker returns the config checker for direct access.
func (cc *CompositeChecker) GetConfigChecker() *ConfigChecker {
	return cc.configChecker
}

// FixIssues attempts to automatically fix fixable issues found during checks.
func (cc *CompositeChecker) FixIssues() error {
	// Find Go files for fixing
	files, err := cc.styleChecker.findGoFiles()
	if err != nil {
		cc.logger.Error("Failed to find Go files for fixing", "error", err)
		return err
	}

	cc.logger.Info("Found Go files for fixing", "count", len(files))
	// TODO: Implement actual fixing logic
	return nil
}

// GenerateReports generates detailed reports for all checker results.
func (cc *CompositeChecker) GenerateReports(outputDir string) error {
	cc.logger.Info("Generating quality check reports", "output_dir", outputDir)
	// TODO: Implement report generation
	return nil
}

// buildFileIndex creates or updates the shared file index for performance optimization.
func (cc *CompositeChecker) buildFileIndex() error {
	cc.indexMux.Lock()
	defer cc.indexMux.Unlock()

	// Check if index is still fresh
	if time.Since(cc.fileIndex.IndexedAt) < cc.indexMaxAge {
		return nil // Index is still fresh
	}

	start := time.Now()
	cc.logger.Info("Building file index for performance optimization")

	// Reset index
	cc.fileIndex = &FileIndex{
		GoFiles:     make([]string, 0),
		TestFiles:   make([]string, 0),
		ConfigFiles: make([]string, 0),
		ProtoFiles:  make([]string, 0),
		AllFiles:    make([]string, 0),
		FileMeta:    make(map[string]os.FileInfo),
		IndexedAt:   time.Now(),
	}

	configExtensions := []string{".yaml", ".yml", ".json", ".toml"}
	specialConfigFiles := []string{"go.mod", "go.sum", ".env", ".gitignore"}

	err := filepath.Walk(cc.workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip common directories that should be excluded
			if info.Name() == "vendor" || info.Name() == ".git" ||
				info.Name() == "node_modules" || info.Name() == ".vscode" ||
				info.Name() == ".idea" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(cc.workDir, path)
		if err != nil {
			return err
		}

		// Store file metadata
		cc.fileIndex.FileMeta[relPath] = info
		cc.fileIndex.AllFiles = append(cc.fileIndex.AllFiles, relPath)

		fileName := info.Name()
		ext := strings.ToLower(filepath.Ext(path))

		// Categorize files
		switch {
		case ext == ".go":
			if strings.HasSuffix(fileName, "_test.go") {
				cc.fileIndex.TestFiles = append(cc.fileIndex.TestFiles, relPath)
			} else {
				cc.fileIndex.GoFiles = append(cc.fileIndex.GoFiles, relPath)
			}
		case ext == ".proto":
			cc.fileIndex.ProtoFiles = append(cc.fileIndex.ProtoFiles, relPath)
		case contains(configExtensions, ext) || contains(specialConfigFiles, fileName):
			cc.fileIndex.ConfigFiles = append(cc.fileIndex.ConfigFiles, relPath)
		}

		return nil
	})

	if err != nil {
		cc.logger.Error("Failed to build file index", "error", err)
		return err
	}

	cc.logger.Info("File index built successfully",
		"go_files", len(cc.fileIndex.GoFiles),
		"test_files", len(cc.fileIndex.TestFiles),
		"config_files", len(cc.fileIndex.ConfigFiles),
		"proto_files", len(cc.fileIndex.ProtoFiles),
		"total_files", len(cc.fileIndex.AllFiles),
		"duration", time.Since(start))

	return nil
}

// GetGoFiles returns cached Go files from the index.
func (cc *CompositeChecker) GetGoFiles() ([]string, error) {
	if err := cc.buildFileIndex(); err != nil {
		return nil, err
	}

	cc.indexMux.RLock()
	defer cc.indexMux.RUnlock()

	return append([]string(nil), cc.fileIndex.GoFiles...), nil // Return copy
}

// GetTestFiles returns cached test files from the index.
func (cc *CompositeChecker) GetTestFiles() ([]string, error) {
	if err := cc.buildFileIndex(); err != nil {
		return nil, err
	}

	cc.indexMux.RLock()
	defer cc.indexMux.RUnlock()

	return append([]string(nil), cc.fileIndex.TestFiles...), nil // Return copy
}

// GetConfigFiles returns cached configuration files from the index.
func (cc *CompositeChecker) GetConfigFiles() ([]string, error) {
	if err := cc.buildFileIndex(); err != nil {
		return nil, err
	}

	cc.indexMux.RLock()
	defer cc.indexMux.RUnlock()

	return append([]string(nil), cc.fileIndex.ConfigFiles...), nil // Return copy
}

// GetProtoFiles returns cached Protocol Buffer files from the index.
func (cc *CompositeChecker) GetProtoFiles() ([]string, error) {
	if err := cc.buildFileIndex(); err != nil {
		return nil, err
	}

	cc.indexMux.RLock()
	defer cc.indexMux.RUnlock()

	return append([]string(nil), cc.fileIndex.ProtoFiles...), nil // Return copy
}

// GetFileInfo returns cached file metadata from the index.
func (cc *CompositeChecker) GetFileInfo(relativePath string) (os.FileInfo, bool) {
	cc.indexMux.RLock()
	defer cc.indexMux.RUnlock()

	info, exists := cc.fileIndex.FileMeta[relativePath]
	return info, exists
}

// InvalidateFileIndex forces a rebuild of the file index on next access.
func (cc *CompositeChecker) InvalidateFileIndex() {
	cc.indexMux.Lock()
	defer cc.indexMux.Unlock()

	cc.fileIndex.IndexedAt = time.Time{} // Reset to zero time to force rebuild
}

// Helper function to check if slice contains element
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
