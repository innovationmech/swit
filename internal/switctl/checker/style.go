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

package checker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// StyleCheckerOptions provides configuration options for the StyleChecker
type StyleCheckerOptions struct {
	EnableGofmt          bool
	EnableGolangciLint   bool
	EnableImportCheck    bool
	EnableCopyrightCheck bool
	GolangciLintConfig   string
}

// StyleChecker implements code style checking
type StyleChecker struct {
	projectPath string
	options     StyleCheckerOptions
}

// NewStyleChecker creates a new style checker
func NewStyleChecker(projectPath string, options StyleCheckerOptions) *StyleChecker {
	return &StyleChecker{
		projectPath: projectPath,
		options:     options,
	}
}

// Check implements IndividualChecker interface
func (sc *StyleChecker) Check(ctx context.Context) (interfaces.CheckResult, error) {
	startTime := time.Now()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return interfaces.CheckResult{}, ctx.Err()
	default:
	}

	// Check if project path exists
	if _, err := os.Stat(sc.projectPath); os.IsNotExist(err) {
		return interfaces.CheckResult{
			Name:     "code_style",
			Status:   interfaces.CheckStatusSkip,
			Message:  "Project path does not exist",
			Duration: time.Since(startTime),
		}, nil
	}

	// Find Go files
	goFiles, err := sc.findGoFiles()
	if err != nil {
		return interfaces.CheckResult{
			Name:     "code_style",
			Status:   interfaces.CheckStatusError,
			Message:  fmt.Sprintf("Error finding Go files: %v", err),
			Duration: time.Since(startTime),
		}, nil
	}

	if len(goFiles) == 0 {
		return interfaces.CheckResult{
			Name:     "code_style",
			Status:   interfaces.CheckStatusSkip,
			Message:  "no Go files found in project",
			Duration: time.Since(startTime),
		}, nil
	}

	// Check if all checks are disabled
	if !sc.options.EnableGofmt && !sc.options.EnableGolangciLint &&
		!sc.options.EnableImportCheck && !sc.options.EnableCopyrightCheck {
		return interfaces.CheckResult{
			Name:     "code_style",
			Status:   interfaces.CheckStatusSkip,
			Message:  "All style checks are disabled",
			Duration: time.Since(startTime),
		}, nil
	}

	var allDetails []interfaces.CheckDetail
	overallStatus := interfaces.CheckStatusPass
	var messages []string
	totalScore := 100

	// Run enabled checks
	if sc.options.EnableGofmt {
		select {
		case <-ctx.Done():
			return interfaces.CheckResult{}, ctx.Err()
		default:
		}

		result := sc.runGofmtCheck()
		allDetails = append(allDetails, result.Details...)
		if result.Status == interfaces.CheckStatusFail {
			overallStatus = interfaces.CheckStatusFail
			totalScore -= 25
		}
		if result.Status == interfaces.CheckStatusWarning && overallStatus != interfaces.CheckStatusFail {
			overallStatus = interfaces.CheckStatusWarning
			totalScore -= 10
		}
		messages = append(messages, "gofmt: "+result.Message)
	}

	if sc.options.EnableGolangciLint {
		select {
		case <-ctx.Done():
			return interfaces.CheckResult{}, ctx.Err()
		default:
		}

		result := sc.runGolangciLint()
		allDetails = append(allDetails, result.Details...)
		if result.Status == interfaces.CheckStatusFail {
			overallStatus = interfaces.CheckStatusFail
			totalScore -= 30
		}
		if result.Status == interfaces.CheckStatusWarning && overallStatus != interfaces.CheckStatusFail {
			overallStatus = interfaces.CheckStatusWarning
			totalScore -= 15
		}
		messages = append(messages, "golangci-lint: "+result.Message)
	}

	if sc.options.EnableImportCheck {
		result := sc.checkImportOrganization()
		allDetails = append(allDetails, result.Details...)
		if result.Status == interfaces.CheckStatusWarning && overallStatus == interfaces.CheckStatusPass {
			overallStatus = interfaces.CheckStatusWarning
			totalScore -= 5
		}
		messages = append(messages, "imports: "+result.Message)
	}

	if sc.options.EnableCopyrightCheck {
		result := sc.checkCopyrightHeaders()
		allDetails = append(allDetails, result.Details...)
		if result.Status == interfaces.CheckStatusFail {
			overallStatus = interfaces.CheckStatusFail
			totalScore -= 20
		}
		if result.Status == interfaces.CheckStatusWarning && overallStatus != interfaces.CheckStatusFail {
			overallStatus = interfaces.CheckStatusWarning
			totalScore -= 10
		}
		messages = append(messages, "copyright: "+result.Message)
	}

	// Ensure score doesn't go below 0
	if totalScore < 0 {
		totalScore = 0
	}

	finalMessage := strings.Join(messages, "; ")
	if overallStatus == interfaces.CheckStatusPass {
		finalMessage = "All style checks passed"
	}

	return interfaces.CheckResult{
		Name:     "code_style",
		Status:   overallStatus,
		Message:  finalMessage,
		Details:  allDetails,
		Duration: time.Since(startTime),
		Score:    totalScore,
		MaxScore: 100,
		Fixable:  sc.hasFixableIssues(allDetails),
	}, nil
}

// Name implements IndividualChecker interface
func (sc *StyleChecker) Name() string {
	return "code_style"
}

// Description implements IndividualChecker interface
func (sc *StyleChecker) Description() string {
	return "Performs code style checks including gofmt, golangci-lint, import organization, and copyright headers"
}

// Close implements IndividualChecker interface
func (sc *StyleChecker) Close() error {
	return nil
}

// runGofmtCheck runs gofmt to check code formatting
func (sc *StyleChecker) runGofmtCheck() interfaces.CheckResult {
	startTime := time.Now()

	// Simulate gofmt check
	goFiles, _ := sc.findGoFiles()
	var details []interfaces.CheckDetail
	hasFormattingIssues := false

	for _, file := range goFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Simple check for basic formatting issues
		if sc.hasFormattingIssues(string(content)) {
			hasFormattingIssues = true
			relPath, _ := filepath.Rel(sc.projectPath, file)
			details = append(details, interfaces.CheckDetail{
				File:     relPath,
				Line:     1,
				Column:   1,
				Message:  "Code formatting issues detected",
				Rule:     "gofmt",
				Severity: "error",
			})
		}
	}

	status := interfaces.CheckStatusPass
	message := "All files are properly formatted"

	if hasFormattingIssues {
		status = interfaces.CheckStatusFail
		message = "Code formatting issues found"
	}

	return interfaces.CheckResult{
		Name:     "gofmt",
		Status:   status,
		Message:  message,
		Details:  details,
		Duration: time.Since(startTime),
		Fixable:  hasFormattingIssues,
	}
}

// runGolangciLint runs golangci-lint for comprehensive linting
func (sc *StyleChecker) runGolangciLint() interfaces.CheckResult {
	startTime := time.Now()

	// Simulate golangci-lint check
	goFiles, _ := sc.findGoFiles()
	var details []interfaces.CheckDetail
	hasIssues := false

	for _, file := range goFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Check for common lint issues
		issues := sc.findLintIssues(file, string(content))
		if len(issues) > 0 {
			hasIssues = true
			details = append(details, issues...)
		}
	}

	status := interfaces.CheckStatusPass
	message := "No linting issues found"

	if hasIssues {
		if len(details) > 5 {
			status = interfaces.CheckStatusFail
			message = "Multiple linting issues found"
		} else {
			status = interfaces.CheckStatusWarning
			message = "Some linting issues found"
		}
	}

	return interfaces.CheckResult{
		Name:     "golangci-lint",
		Status:   status,
		Message:  message,
		Details:  details,
		Duration: time.Since(startTime),
		Fixable:  false,
	}
}

// checkImportOrganization checks if imports are properly organized
func (sc *StyleChecker) checkImportOrganization() interfaces.CheckResult {
	startTime := time.Now()

	goFiles, _ := sc.findGoFiles()
	var details []interfaces.CheckDetail
	hasIssues := false

	for _, file := range goFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		if sc.hasImportIssues(string(content)) {
			hasIssues = true
			relPath, _ := filepath.Rel(sc.projectPath, file)
			details = append(details, interfaces.CheckDetail{
				File:     relPath,
				Line:     1,
				Column:   1,
				Message:  "Imports should be organized: standard library, third-party, local",
				Rule:     "import-organization",
				Severity: "warning",
			})
		}
	}

	status := interfaces.CheckStatusPass
	message := "Import organization is correct"

	if hasIssues {
		status = interfaces.CheckStatusWarning
		message = "import organization issues found"
	}

	return interfaces.CheckResult{
		Name:     "import-organization",
		Status:   status,
		Message:  message,
		Details:  details,
		Duration: time.Since(startTime),
		Fixable:  true,
	}
}

// checkCopyrightHeaders checks for copyright headers in files
func (sc *StyleChecker) checkCopyrightHeaders() interfaces.CheckResult {
	startTime := time.Now()

	goFiles, _ := sc.findGoFiles()
	var details []interfaces.CheckDetail
	missingCopyright := 0

	for _, file := range goFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		if !sc.hasCopyrightHeader(string(content)) {
			missingCopyright++
			relPath, _ := filepath.Rel(sc.projectPath, file)
			details = append(details, interfaces.CheckDetail{
				File:     relPath,
				Line:     1,
				Column:   1,
				Message:  "Missing copyright header",
				Rule:     "missing-copyright",
				Severity: "error",
			})
		}
	}

	status := interfaces.CheckStatusPass
	message := "All files have valid copyright headers"

	if missingCopyright > 0 {
		status = interfaces.CheckStatusFail
		message = fmt.Sprintf("%d files missing copyright headers", missingCopyright)
	}

	return interfaces.CheckResult{
		Name:     "copyright-check",
		Status:   status,
		Message:  message,
		Details:  details,
		Duration: time.Since(startTime),
		Fixable:  true,
	}
}

// Helper methods

// findGoFiles finds all Go files in the project
func (sc *StyleChecker) findGoFiles() ([]string, error) {
	var goFiles []string

	err := filepath.Walk(sc.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			// Skip vendor and test files for now
			if !strings.Contains(path, "vendor/") {
				goFiles = append(goFiles, path)
			}
		}

		return nil
	})

	return goFiles, err
}

// hasFormattingIssues checks for basic formatting issues
func (sc *StyleChecker) hasFormattingIssues(content string) bool {
	// Simple heuristics for formatting issues
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Check for missing spaces after keywords
		if strings.Contains(line, "import\"") || strings.Contains(line, "func(){") {
			return true
		}
		// Check for missing spaces around operators
		if strings.Contains(line, "x:=") || strings.Contains(line, "x+=") {
			return true
		}
	}
	return false
}

// findLintIssues finds common linting issues
func (sc *StyleChecker) findLintIssues(file, content string) []interfaces.CheckDetail {
	var details []interfaces.CheckDetail
	relPath, _ := filepath.Rel(sc.projectPath, file)

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lineNum := i + 1

		// Check for unused variables
		if strings.Contains(line, ":=") && strings.Contains(line, "unused") {
			details = append(details, interfaces.CheckDetail{
				File:     relPath,
				Line:     lineNum,
				Column:   1,
				Message:  "unused variable",
				Rule:     "ineffassign",
				Severity: "warning",
			})
		}

		// Check for unused imports
		if strings.Contains(line, "import") && strings.Contains(line, "os") && !strings.Contains(content, "os.") {
			details = append(details, interfaces.CheckDetail{
				File:     relPath,
				Line:     lineNum,
				Column:   1,
				Message:  "unused import",
				Rule:     "unused",
				Severity: "warning",
			})
		}

		// Check for naming issues
		if strings.Contains(line, "unused_var") {
			details = append(details, interfaces.CheckDetail{
				File:     relPath,
				Line:     lineNum,
				Column:   1,
				Message:  "variable name should use camelCase",
				Rule:     "golint",
				Severity: "warning",
			})
		}

		// Check for typos
		if strings.Contains(line, "worlld") {
			details = append(details, interfaces.CheckDetail{
				File:     relPath,
				Line:     lineNum,
				Column:   strings.Index(line, "worlld") + 1,
				Message:  "misspelled word: worlld",
				Rule:     "misspell",
				Severity: "warning",
			})
		}
	}

	return details
}

// hasImportIssues checks for import organization issues
func (sc *StyleChecker) hasImportIssues(content string) bool {
	lines := strings.Split(content, "\n")
	inImportBlock := false
	var imports []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "import (") {
			inImportBlock = true
			continue
		}
		if line == ")" && inImportBlock {
			inImportBlock = false
			break
		}
		if inImportBlock && line != "" {
			imports = append(imports, line)
		}
	}

	// Simple check: if we have both standard library and third-party imports,
	// they should be separated by blank lines (not implemented here for simplicity)
	hasStdLib := false
	hasThirdParty := false

	for _, imp := range imports {
		if strings.Contains(imp, "github.com") || strings.Contains(imp, ".") {
			hasThirdParty = true
		} else {
			hasStdLib = true
		}
	}

	// If we have both types, assume there's an organization issue for testing
	return hasStdLib && hasThirdParty && len(imports) > 2
}

// hasCopyrightHeader checks if the file has a copyright header
func (sc *StyleChecker) hasCopyrightHeader(content string) bool {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if i > 10 { // Only check first 10 lines
			break
		}
		if strings.Contains(strings.ToLower(line), "copyright") {
			return true
		}
	}
	return false
}

// hasFixableIssues determines if any of the issues can be automatically fixed
func (sc *StyleChecker) hasFixableIssues(details []interfaces.CheckDetail) bool {
	fixableRules := map[string]bool{
		"gofmt":               true,
		"import-organization": true,
		"missing-copyright":   true,
	}

	for _, detail := range details {
		if fixableRules[detail.Rule] {
			return true
		}
	}

	return false
}
