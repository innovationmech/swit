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

// Package scanner provides security scanning capabilities for code analysis.
package scanner

import (
	"context"
	"fmt"
	"time"
)

// Severity represents the severity level of a security finding.
type Severity string

const (
	// SeverityCritical represents critical severity findings.
	SeverityCritical Severity = "critical"
	// SeverityHigh represents high severity findings.
	SeverityHigh Severity = "high"
	// SeverityMedium represents medium severity findings.
	SeverityMedium Severity = "medium"
	// SeverityLow represents low severity findings.
	SeverityLow Severity = "low"
	// SeverityInfo represents informational findings.
	SeverityInfo Severity = "info"
)

// ScanTool defines the interface for security scanning tools.
type ScanTool interface {
	// Name returns the name of the scanning tool.
	Name() string
	// Scan performs a security scan on the target directory.
	Scan(ctx context.Context, target string) (*ScanResult, error)
	// IsAvailable checks if the tool is available in the system.
	IsAvailable() bool
}

// SecurityFinding represents a single security issue found during scanning.
type SecurityFinding struct {
	// ID is the unique identifier of the finding (e.g., rule ID).
	ID string `json:"id"`
	// Severity is the severity level of the finding.
	Severity Severity `json:"severity"`
	// Category is the category of the security issue.
	Category string `json:"category"`
	// Title is a short title of the finding.
	Title string `json:"title"`
	// Description provides detailed information about the finding.
	Description string `json:"description"`
	// File is the file path where the issue was found.
	File string `json:"file"`
	// Line is the line number where the issue was found.
	Line int `json:"line"`
	// Column is the column number where the issue was found (optional).
	Column int `json:"column,omitempty"`
	// Code is the code snippet related to the finding (optional).
	Code string `json:"code,omitempty"`
	// Remediation provides guidance on how to fix the issue.
	Remediation string `json:"remediation,omitempty"`
	// CWE is the Common Weakness Enumeration identifier (optional).
	CWE string `json:"cwe,omitempty"`
	// CVSS is the Common Vulnerability Scoring System score (optional).
	CVSS float64 `json:"cvss,omitempty"`
	// References contains URLs to related documentation or advisories.
	References []string `json:"references,omitempty"`
}

// ScanSummary provides aggregated statistics about scan results.
type ScanSummary struct {
	// TotalFindings is the total number of findings.
	TotalFindings int `json:"total_findings"`
	// CriticalCount is the count of critical severity findings.
	CriticalCount int `json:"critical_count"`
	// HighCount is the count of high severity findings.
	HighCount int `json:"high_count"`
	// MediumCount is the count of medium severity findings.
	MediumCount int `json:"medium_count"`
	// LowCount is the count of low severity findings.
	LowCount int `json:"low_count"`
	// InfoCount is the count of informational findings.
	InfoCount int `json:"info_count"`
	// FilesScanned is the number of files that were scanned.
	FilesScanned int `json:"files_scanned"`
}

// ScanResult represents the result of a security scan.
type ScanResult struct {
	// Tool is the name of the scanning tool that produced this result.
	Tool string `json:"tool"`
	// Timestamp is when the scan was performed.
	Timestamp time.Time `json:"timestamp"`
	// Duration is how long the scan took.
	Duration time.Duration `json:"duration"`
	// Findings contains all security findings discovered.
	Findings []SecurityFinding `json:"findings"`
	// Summary provides aggregated statistics.
	Summary ScanSummary `json:"summary"`
	// Error contains any error that occurred during scanning (optional).
	Error string `json:"error,omitempty"`
}

// ScannerConfig represents the configuration for the security scanner.
type ScannerConfig struct {
	// Enabled indicates whether security scanning is enabled.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Tools is the list of scanning tools to use (e.g., "gosec", "govulncheck", "trivy").
	Tools []string `yaml:"tools" json:"tools"`
	// OutputDir is the directory where scan reports will be written.
	OutputDir string `yaml:"output_dir" json:"output_dir"`
	// FailOnHigh determines if the scanner should fail on high severity findings.
	FailOnHigh bool `yaml:"fail_on_high" json:"fail_on_high"`
	// FailOnMedium determines if the scanner should fail on medium severity findings.
	FailOnMedium bool `yaml:"fail_on_medium" json:"fail_on_medium"`
	// FailOnCritical determines if the scanner should fail on critical severity findings.
	FailOnCritical bool `yaml:"fail_on_critical" json:"fail_on_critical"`
	// Excludes contains patterns of files/directories to exclude from scanning.
	Excludes []string `yaml:"excludes" json:"excludes"`
	// Timeout is the maximum duration for a scan operation.
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
	// Parallel determines if tools should run in parallel.
	Parallel bool `yaml:"parallel" json:"parallel"`
}

// DefaultConfig returns the default scanner configuration.
func DefaultConfig() *ScannerConfig {
	return &ScannerConfig{
		Enabled:        true,
		Tools:          []string{"gosec", "govulncheck"},
		OutputDir:      "_output/security",
		FailOnHigh:     true,
		FailOnMedium:   false,
		FailOnCritical: true,
		Excludes:       []string{"vendor/", "_output/", ".git/"},
		Timeout:        10 * time.Minute,
		Parallel:       true,
	}
}

// SecurityScanner is the main security scanner that coordinates multiple scanning tools.
type SecurityScanner struct {
	config *ScannerConfig
	tools  map[string]ScanTool
}

// NewSecurityScanner creates a new SecurityScanner instance.
func NewSecurityScanner(config *ScannerConfig) (*SecurityScanner, error) {
	if config == nil {
		config = DefaultConfig()
	}

	scanner := &SecurityScanner{
		config: config,
		tools:  make(map[string]ScanTool),
	}

	// Register available tools
	scanner.registerTools()

	return scanner, nil
}

// registerTools registers all available scanning tools.
func (s *SecurityScanner) registerTools() {
	// Register gosec scanner
	gosecTool := NewGosecScanner()
	s.tools[gosecTool.Name()] = gosecTool

	// Register govulncheck scanner
	vulncheckTool := NewGovulncheckScanner()
	s.tools[vulncheckTool.Name()] = vulncheckTool

	// Register trivy scanner (optional)
	trivyTool := NewTrivyScanner()
	s.tools[trivyTool.Name()] = trivyTool
}

// Scan performs security scanning using configured tools.
func (s *SecurityScanner) Scan(ctx context.Context, target string) ([]*ScanResult, error) {
	if !s.config.Enabled {
		return nil, fmt.Errorf("security scanning is disabled")
	}

	var results []*ScanResult
	var errList []error

	// Determine which tools to use
	toolsToRun := s.getEnabledTools()
	if len(toolsToRun) == 0 {
		return nil, fmt.Errorf("no scanning tools are available or enabled")
	}

	// Create context with timeout
	scanCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if s.config.Parallel {
		// Run tools in parallel
		results, errList = s.scanParallel(scanCtx, target, toolsToRun)
	} else {
		// Run tools sequentially
		results, errList = s.scanSequential(scanCtx, target, toolsToRun)
	}

	// Check for errors
	if len(errList) > 0 {
		return results, fmt.Errorf("scanning completed with %d error(s): %v", len(errList), errList[0])
	}

	return results, nil
}

// getEnabledTools returns the list of enabled and available tools.
func (s *SecurityScanner) getEnabledTools() []ScanTool {
	var tools []ScanTool

	for _, toolName := range s.config.Tools {
		tool, exists := s.tools[toolName]
		if !exists {
			continue
		}
		if tool.IsAvailable() {
			tools = append(tools, tool)
		}
	}

	return tools
}

// scanSequential runs scanning tools sequentially.
func (s *SecurityScanner) scanSequential(ctx context.Context, target string, tools []ScanTool) ([]*ScanResult, []error) {
	var results []*ScanResult
	var errors []error

	for _, tool := range tools {
		select {
		case <-ctx.Done():
			errors = append(errors, ctx.Err())
			return results, errors
		default:
		}

		result, err := tool.Scan(ctx, target)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", tool.Name(), err))
			continue
		}
		results = append(results, result)
	}

	return results, errors
}

// scanParallel runs scanning tools in parallel.
func (s *SecurityScanner) scanParallel(ctx context.Context, target string, tools []ScanTool) ([]*ScanResult, []error) {
	type scanOutput struct {
		result *ScanResult
		err    error
	}

	resultChan := make(chan scanOutput, len(tools))

	// Launch scans
	for _, tool := range tools {
		go func(t ScanTool) {
			result, err := t.Scan(ctx, target)
			resultChan <- scanOutput{result: result, err: err}
		}(tool)
	}

	// Collect results
	var results []*ScanResult
	var errors []error

	for i := 0; i < len(tools); i++ {
		output := <-resultChan
		if output.err != nil {
			errors = append(errors, output.err)
		} else if output.result != nil {
			results = append(results, output.result)
		}
	}

	return results, errors
}

// ShouldFail determines if the scan results should cause a failure based on configuration.
func (s *SecurityScanner) ShouldFail(results []*ScanResult) bool {
	for _, result := range results {
		if s.config.FailOnCritical && result.Summary.CriticalCount > 0 {
			return true
		}
		if s.config.FailOnHigh && result.Summary.HighCount > 0 {
			return true
		}
		if s.config.FailOnMedium && result.Summary.MediumCount > 0 {
			return true
		}
	}
	return false
}

// GetTotalFindings returns the total number of findings across all results.
func GetTotalFindings(results []*ScanResult) int {
	total := 0
	for _, result := range results {
		total += result.Summary.TotalFindings
	}
	return total
}

// GetFindingsBySeverity aggregates findings by severity level across all results.
func GetFindingsBySeverity(results []*ScanResult) map[Severity]int {
	counts := make(map[Severity]int)

	for _, result := range results {
		counts[SeverityCritical] += result.Summary.CriticalCount
		counts[SeverityHigh] += result.Summary.HighCount
		counts[SeverityMedium] += result.Summary.MediumCount
		counts[SeverityLow] += result.Summary.LowCount
		counts[SeverityInfo] += result.Summary.InfoCount
	}

	return counts
}

