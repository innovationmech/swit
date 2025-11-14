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

package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GosecScanner implements the ScanTool interface for gosec security scanner.
type GosecScanner struct {
	excludeRules []string
	includeRules []string
}

// NewGosecScanner creates a new GosecScanner instance.
func NewGosecScanner() *GosecScanner {
	return &GosecScanner{
		excludeRules: []string{},
		includeRules: []string{},
	}
}

// Name returns the name of the scanning tool.
func (g *GosecScanner) Name() string {
	return "gosec"
}

// IsAvailable checks if gosec is installed and available.
func (g *GosecScanner) IsAvailable() bool {
	_, err := exec.LookPath("gosec")
	return err == nil
}

// SetExcludeRules sets the rules to exclude from scanning.
func (g *GosecScanner) SetExcludeRules(rules []string) {
	g.excludeRules = rules
}

// SetIncludeRules sets the rules to include in scanning.
func (g *GosecScanner) SetIncludeRules(rules []string) {
	g.includeRules = rules
}

// Scan performs a security scan using gosec.
func (g *GosecScanner) Scan(ctx context.Context, target string) (*ScanResult, error) {
	startTime := time.Now()

	// Build gosec command
	args := []string{"-fmt=json", "-no-fail"}

	// Add exclude rules if specified
	if len(g.excludeRules) > 0 {
		args = append(args, "-exclude="+strings.Join(g.excludeRules, ","))
	}

	// Add include rules if specified
	if len(g.includeRules) > 0 {
		args = append(args, "-include="+strings.Join(g.includeRules, ","))
	}

	// Add target
	args = append(args, target)

	// Execute gosec
	cmd := exec.CommandContext(ctx, "gosec", args...)
	output, err := cmd.CombinedOutput()

	// gosec returns non-zero exit code when issues are found, which is expected
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			// This is a real error, not just findings
			return nil, fmt.Errorf("failed to execute gosec: %w", err)
		}
	}

	// Parse gosec output
	result, err := g.parseGosecOutput(output, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gosec output: %w", err)
	}

	return result, nil
}

// gosecOutput represents the JSON output structure from gosec.
type gosecOutput struct {
	Issues []struct {
		Severity   string `json:"severity"`
		Confidence string `json:"confidence"`
		CWE        struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"cwe"`
		RuleID  string `json:"rule_id"`
		Details string `json:"details"`
		File    string `json:"file"`
		Code    string `json:"code"`
		Line    string `json:"line"`
		Column  string `json:"column"`
	} `json:"Issues"`
	Stats struct {
		Files int `json:"files"`
		Lines int `json:"lines"`
	} `json:"Stats"`
}

// parseGosecOutput parses the JSON output from gosec and converts it to ScanResult.
func (g *GosecScanner) parseGosecOutput(output []byte, startTime time.Time) (*ScanResult, error) {
	var gosecResult gosecOutput

	if err := json.Unmarshal(output, &gosecResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gosec output: %w", err)
	}

	result := &ScanResult{
		Tool:      g.Name(),
		Timestamp: startTime,
		Duration:  time.Since(startTime),
		Findings:  make([]SecurityFinding, 0, len(gosecResult.Issues)),
		Summary: ScanSummary{
			FilesScanned: gosecResult.Stats.Files,
		},
	}

	// Convert gosec issues to SecurityFindings
	for _, issue := range gosecResult.Issues {
		line, _ := strconv.Atoi(issue.Line)
		column, _ := strconv.Atoi(issue.Column)

		severity := g.mapGosecSeverity(issue.Severity)

		finding := SecurityFinding{
			ID:          issue.RuleID,
			Severity:    severity,
			Category:    "Code Analysis",
			Title:       fmt.Sprintf("Gosec Rule: %s", issue.RuleID),
			Description: issue.Details,
			File:        issue.File,
			Line:        line,
			Column:      column,
			Code:        issue.Code,
			CWE:         issue.CWE.ID,
			Remediation: g.getRemediationAdvice(issue.RuleID),
			References:  []string{issue.CWE.URL},
		}

		result.Findings = append(result.Findings, finding)

		// Update summary counts
		result.Summary.TotalFindings++
		switch severity {
		case SeverityCritical:
			result.Summary.CriticalCount++
		case SeverityHigh:
			result.Summary.HighCount++
		case SeverityMedium:
			result.Summary.MediumCount++
		case SeverityLow:
			result.Summary.LowCount++
		case SeverityInfo:
			result.Summary.InfoCount++
		}
	}

	return result, nil
}

// mapGosecSeverity maps gosec severity levels to our standard severity levels.
func (g *GosecScanner) mapGosecSeverity(gosecSev string) Severity {
	switch strings.ToLower(gosecSev) {
	case "high":
		return SeverityHigh
	case "medium":
		return SeverityMedium
	case "low":
		return SeverityLow
	default:
		return SeverityInfo
	}
}

// getRemediationAdvice provides remediation advice based on gosec rule ID.
func (g *GosecScanner) getRemediationAdvice(ruleID string) string {
	// Common remediation advice for gosec rules
	remediations := map[string]string{
		"G101": "Avoid hardcoding credentials. Use environment variables or secure secret management systems.",
		"G102": "Avoid using unsafe network bindings. Bind to specific interfaces instead of 0.0.0.0.",
		"G103": "Avoid using unsafe packages. Review the use of unsafe pointer operations.",
		"G104": "Always check for errors. Unhandled errors can lead to unexpected behavior.",
		"G106": "Use crypto/ssh package for SSH connections instead of third-party implementations.",
		"G107": "Avoid passing user input directly to URL.Parse. Validate and sanitize input first.",
		"G108": "Use profiling endpoints carefully. Ensure they are not exposed in production.",
		"G109": "Avoid integer overflow issues. Use appropriate data types and bounds checking.",
		"G110": "Avoid potential DoS vulnerabilities. Implement proper resource limits.",
		"G201": "Avoid SQL injection. Use parameterized queries or prepared statements.",
		"G202": "Avoid SQL injection in string concatenation. Use parameterized queries.",
		"G203": "Avoid unescaped HTML in templates. Use html/template package properly.",
		"G204": "Avoid command injection. Validate and sanitize command arguments.",
		"G301": "Avoid creating files/directories with overly permissive permissions.",
		"G302": "Avoid using files with overly permissive permissions.",
		"G303": "Avoid creating world-readable or world-writable temporary files.",
		"G304": "Avoid file path traversal vulnerabilities. Validate and sanitize file paths.",
		"G305": "Avoid extracting zip files without validating paths (zip slip vulnerability).",
		"G306": "Avoid writing sensitive data to files with insecure permissions.",
		"G401": "Avoid using weak cryptographic algorithms (MD5, SHA1).",
		"G402": "Use secure TLS/SSL configuration. Avoid insecure cipher suites.",
		"G403": "Use strong encryption algorithms. Avoid DES, RC4, and other weak ciphers.",
		"G404": "Use crypto/rand for generating random numbers, not math/rand.",
		"G501": "Avoid importing blacklisted crypto packages (MD5, DES, RC4).",
		"G502": "Avoid importing blacklisted crypto packages (MD4, RIPEMD160).",
		"G503": "Avoid importing blacklisted crypto packages (SHA1).",
		"G504": "Avoid importing net/http/cgi which has known security issues.",
		"G505": "Avoid importing blacklisted crypto packages (DES).",
		"G601": "Avoid implicit memory aliasing in for loops. Use explicit copy or index.",
	}

	if advice, exists := remediations[ruleID]; exists {
		return advice
	}

	return "Review the security finding and follow best practices to remediate the issue."
}

