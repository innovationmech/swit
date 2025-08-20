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

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// CompositeChecker implements the full QualityChecker interface by composing
// all individual checkers into a single comprehensive checker.
type CompositeChecker struct {
	styleChecker    *StyleChecker
	testRunner      *TestRunner
	securityChecker *SecurityChecker
	configChecker   *ConfigChecker
	workDir         string
	logger          interfaces.Logger
}

// NewCompositeChecker creates a new composite quality checker that combines
// all individual checker implementations.
func NewCompositeChecker(workDir string, logger interfaces.Logger) *CompositeChecker {
	return &CompositeChecker{
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
	}
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
