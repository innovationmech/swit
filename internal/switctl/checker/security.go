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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// SecurityChecker implements security vulnerability scanning.
type SecurityChecker struct {
	workDir string
	logger  interfaces.Logger
	config  SecurityConfig
}

// SecurityConfig represents configuration for security checking.
type SecurityConfig struct {
	GosecEnabled      bool     `yaml:"gosec_enabled" json:"gosec_enabled"`
	NancyEnabled      bool     `yaml:"nancy_enabled" json:"nancy_enabled"`
	TrivyEnabled      bool     `yaml:"trivy_enabled" json:"trivy_enabled"`
	SeverityThreshold string   `yaml:"severity_threshold" json:"severity_threshold"`
	ExcludeRules      []string `yaml:"exclude_rules" json:"exclude_rules"`
	IncludeRules      []string `yaml:"include_rules" json:"include_rules"`
	ExcludePatterns   []string `yaml:"exclude_patterns" json:"exclude_patterns"`
	ReportFormat      string   `yaml:"report_format" json:"report_format"`
	MaxIssues         int      `yaml:"max_issues" json:"max_issues"`
	Timeout           string   `yaml:"timeout" json:"timeout"`
	FailOnMedium      bool     `yaml:"fail_on_medium" json:"fail_on_medium"`
	FailOnHigh        bool     `yaml:"fail_on_high" json:"fail_on_high"`
	CustomRulesPath   string   `yaml:"custom_rules_path" json:"custom_rules_path"`
}

// NewSecurityChecker creates a new security checker.
func NewSecurityChecker(workDir string, logger interfaces.Logger) *SecurityChecker {
	return &SecurityChecker{
		workDir: workDir,
		logger:  logger,
		config:  defaultSecurityConfig(),
	}
}

// SetConfig updates the security checker configuration.
func (sc *SecurityChecker) SetConfig(config SecurityConfig) {
	sc.config = config
}

// Check implements IndividualChecker interface.
func (sc *SecurityChecker) Check(ctx context.Context) (interfaces.CheckResult, error) {
	startTime := time.Now()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return interfaces.CheckResult{}, ctx.Err()
	default:
	}

	// Run the security check
	securityResult := sc.CheckSecurity()

	// Convert SecurityResult to CheckResult
	status := interfaces.CheckStatusPass
	message := "Security scan completed successfully"

	if len(securityResult.Issues) > 0 {
		highCount := 0
		mediumCount := 0
		for _, issue := range securityResult.Issues {
			if issue.Severity == "high" || issue.Severity == "critical" {
				highCount++
			} else if issue.Severity == "medium" {
				mediumCount++
			}
		}

		if highCount > 0 {
			status = interfaces.CheckStatusFail
			message = fmt.Sprintf("Found %d high/critical and %d medium security issues", highCount, mediumCount)
		} else if mediumCount > 0 {
			status = interfaces.CheckStatusWarning
			message = fmt.Sprintf("Found %d medium security issues", mediumCount)
		}
	}

	// Convert security issues to check details
	var details []interfaces.CheckDetail
	for _, issue := range securityResult.Issues {
		details = append(details, interfaces.CheckDetail{
			File:     issue.File,
			Line:     issue.Line,
			Message:  issue.Description,
			Rule:     issue.ID,
			Severity: issue.Severity,
			Context:  issue,
		})
	}

	return interfaces.CheckResult{
		Name:     "security",
		Status:   status,
		Message:  message,
		Details:  details,
		Duration: time.Since(startTime),
		Score:    securityResult.Score,
		MaxScore: securityResult.MaxScore,
		Fixable:  false, // Security issues are generally not automatically fixable
	}, nil
}

// Name returns the name of this checker.
func (sc *SecurityChecker) Name() string {
	return "security"
}

// Description returns the description of this checker.
func (sc *SecurityChecker) Description() string {
	return "Performs security vulnerability scanning using gosec, Nancy, and Trivy"
}

// Close cleans up any resources used by the checker.
func (sc *SecurityChecker) Close() error {
	// No resources to clean up for SecurityChecker
	return nil
}

// CheckSecurity performs comprehensive security scanning.
func (sc *SecurityChecker) CheckSecurity() interfaces.SecurityResult {
	start := time.Now()
	result := interfaces.SecurityResult{
		Issues:   make([]interfaces.SecurityIssue, 0),
		Severity: "none",
		MaxScore: 100,
		Score:    100,
	}

	var allIssues []interfaces.SecurityIssue
	filesScanned := 0

	// Run gosec security scanner
	if sc.config.GosecEnabled {
		gosecIssues, scanned := sc.runGosec()
		allIssues = append(allIssues, gosecIssues...)
		filesScanned += scanned
	}

	// Run Nancy for dependency vulnerability scanning
	if sc.config.NancyEnabled {
		nancyIssues := sc.runNancy()
		allIssues = append(allIssues, nancyIssues...)
	}

	// Run Trivy for container and dependency scanning
	if sc.config.TrivyEnabled {
		trivyIssues := sc.runTrivy()
		allIssues = append(allIssues, trivyIssues...)
	}

	// Run custom security checks
	customIssues, customScanned := sc.runCustomChecks()
	allIssues = append(allIssues, customIssues...)
	filesScanned += customScanned

	// Filter and prioritize issues
	result.Issues = sc.filterAndPrioritizeIssues(allIssues)
	result.Scanned = filesScanned
	result.Severity = sc.calculateOverallSeverity(result.Issues)
	result.Score = sc.calculateSecurityScore(result.Issues)
	result.Duration = time.Since(start)

	return result
}

// runGosec executes gosec security scanner.
func (sc *SecurityChecker) runGosec() ([]interfaces.SecurityIssue, int) {
	var issues []interfaces.SecurityIssue

	// Check if gosec is available
	if !sc.isCommandAvailable("gosec") {
		sc.logger.Warn("gosec not found, skipping security scan")
		return issues, 0
	}

	// Build gosec command
	args := []string{"-fmt", "json", "./..."}

	// Add excluded rules
	if len(sc.config.ExcludeRules) > 0 {
		args = append(args, "-exclude", strings.Join(sc.config.ExcludeRules, ","))
	}

	// Add included rules
	if len(sc.config.IncludeRules) > 0 {
		args = append(args, "-include", strings.Join(sc.config.IncludeRules, ","))
	}

	cmd := exec.Command("gosec", args...)
	cmd.Dir = sc.workDir

	_, cancel := context.WithTimeout(context.Background(), sc.getTimeoutDuration())
	defer cancel()

	output, err := cmd.Output()
	if err != nil {
		// gosec returns non-zero exit code when issues are found
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = exitErr.Stderr
			if len(output) == 0 {
				// Try to get stdout if stderr is empty
				if stdoutData, stdoutErr := cmd.Output(); stdoutErr == nil {
					output = stdoutData
				}
			}
		} else {
			sc.logger.Warn("gosec execution failed", "error", err)
			return issues, 0
		}
	}

	// Parse gosec output
	issues, filesScanned := sc.parseGosecOutput(string(output))
	return issues, filesScanned
}

// parseGosecOutput parses gosec JSON output.
func (sc *SecurityChecker) parseGosecOutput(output string) ([]interfaces.SecurityIssue, int) {
	var issues []interfaces.SecurityIssue
	var gosecResult struct {
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

	if err := json.Unmarshal([]byte(output), &gosecResult); err != nil {
		sc.logger.Warn("Failed to parse gosec output", "error", err)
		return issues, 0
	}

	// Convert gosec issues to our format
	for _, issue := range gosecResult.Issues {
		line, _ := strconv.Atoi(issue.Line)

		severity := strings.ToLower(issue.Severity)

		securityIssue := interfaces.SecurityIssue{
			ID:          issue.RuleID,
			Title:       fmt.Sprintf("Security issue: %s", issue.RuleID),
			Description: issue.Details,
			Severity:    severity,
			File:        issue.File,
			Line:        line,
			CWE:         issue.CWE.ID,
			Fix:         sc.getSuggestionForRule(issue.RuleID),
		}

		issues = append(issues, securityIssue)
	}

	return issues, gosecResult.Stats.Files
}

// runNancy executes Nancy for dependency vulnerability scanning.
func (sc *SecurityChecker) runNancy() []interfaces.SecurityIssue {
	var issues []interfaces.SecurityIssue

	// Check if nancy is available
	if !sc.isCommandAvailable("nancy") {
		sc.logger.Warn("nancy not found, skipping dependency scan")
		return issues
	}

	// Check if go.sum exists
	goSumPath := filepath.Join(sc.workDir, "go.sum")
	if _, err := os.Stat(goSumPath); os.IsNotExist(err) {
		sc.logger.Info("go.sum not found, skipping nancy scan")
		return issues
	}

	cmd := exec.Command("nancy", "sleuth", "--loud", "--output", "json")
	cmd.Dir = sc.workDir

	// Pipe go list output to nancy
	goListCmd := exec.Command("go", "list", "-json", "-m", "all")
	goListCmd.Dir = sc.workDir

	goListOutput, err := goListCmd.Output()
	if err != nil {
		sc.logger.Warn("Failed to get module list for nancy", "error", err)
		return issues
	}

	cmd.Stdin = strings.NewReader(string(goListOutput))

	_, cancel := context.WithTimeout(context.Background(), sc.getTimeoutDuration())
	defer cancel()

	output, err := cmd.Output()
	if err != nil {
		sc.logger.Warn("nancy execution failed", "error", err)
		return issues
	}

	// Parse nancy output
	issues = sc.parseNancyOutput(string(output))
	return issues
}

// parseNancyOutput parses Nancy JSON output.
func (sc *SecurityChecker) parseNancyOutput(output string) []interfaces.SecurityIssue {
	var issues []interfaces.SecurityIssue

	// Nancy output format is different, adapt as needed
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to parse as JSON
		var nancyIssue map[string]interface{}
		if err := json.Unmarshal([]byte(line), &nancyIssue); err != nil {
			continue
		}

		// Extract vulnerability information
		if vuln, ok := nancyIssue["vulnerability"].(map[string]interface{}); ok {
			issue := interfaces.SecurityIssue{
				ID:          fmt.Sprintf("NANCY-%s", vuln["id"]),
				Title:       fmt.Sprintf("Dependency vulnerability: %s", vuln["title"]),
				Description: fmt.Sprintf("%s", vuln["description"]),
				Severity:    sc.mapNancySeverity(fmt.Sprintf("%v", vuln["cvss_score"])),
			}

			if cvss, ok := vuln["cvss_score"].(float64); ok {
				issue.CVSS = cvss
			}

			issues = append(issues, issue)
		}
	}

	return issues
}

// runTrivy executes Trivy for comprehensive vulnerability scanning.
func (sc *SecurityChecker) runTrivy() []interfaces.SecurityIssue {
	var issues []interfaces.SecurityIssue

	// Check if trivy is available
	if !sc.isCommandAvailable("trivy") {
		sc.logger.Warn("trivy not found, skipping vulnerability scan")
		return issues
	}

	// Scan filesystem
	cmd := exec.Command("trivy", "fs", "--format", "json", "--security-checks", "vuln", ".")
	cmd.Dir = sc.workDir

	_, cancel := context.WithTimeout(context.Background(), sc.getTimeoutDuration())
	defer cancel()

	output, err := cmd.Output()
	if err != nil {
		sc.logger.Warn("trivy execution failed", "error", err)
		return issues
	}

	// Parse trivy output
	issues = sc.parseTrivyOutput(string(output))
	return issues
}

// parseTrivyOutput parses Trivy JSON output.
func (sc *SecurityChecker) parseTrivyOutput(output string) []interfaces.SecurityIssue {
	var issues []interfaces.SecurityIssue
	var trivyResult struct {
		Results []struct {
			Target          string `json:"Target"`
			Vulnerabilities []struct {
				VulnerabilityID string `json:"VulnerabilityID"`
				PkgName         string `json:"PkgName"`
				Title           string `json:"Title"`
				Description     string `json:"Description"`
				Severity        string `json:"Severity"`
				CVSS            struct {
					Score float64 `json:"Score"`
				} `json:"CVSS"`
				References   []string `json:"References"`
				FixedVersion string   `json:"FixedVersion"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}

	if err := json.Unmarshal([]byte(output), &trivyResult); err != nil {
		sc.logger.Warn("Failed to parse trivy output", "error", err)
		return issues
	}

	// Convert trivy vulnerabilities to our format
	for _, result := range trivyResult.Results {
		for _, vuln := range result.Vulnerabilities {
			issue := interfaces.SecurityIssue{
				ID:          vuln.VulnerabilityID,
				Title:       vuln.Title,
				Description: vuln.Description,
				Severity:    strings.ToLower(vuln.Severity),
				CVSS:        vuln.CVSS.Score,
				Fix:         fmt.Sprintf("Update %s to version %s or later", vuln.PkgName, vuln.FixedVersion),
			}

			// Note: URL field not available in SecurityIssue struct
			// Could be added to description or fix field if needed

			issues = append(issues, issue)
		}
	}

	return issues
}

// runCustomChecks runs custom security checks specific to the project.
func (sc *SecurityChecker) runCustomChecks() ([]interfaces.SecurityIssue, int) {
	var issues []interfaces.SecurityIssue
	filesScanned := 0

	// Find Go files to scan
	files, err := sc.findGoFiles()
	if err != nil {
		sc.logger.Error("Failed to find Go files for custom security checks", "error", err)
		return issues, 0
	}

	for _, file := range files {
		if sc.isExcludedFile(file) {
			continue
		}

		fileIssues := sc.scanFileForSecurityIssues(file)
		issues = append(issues, fileIssues...)
		filesScanned++
	}

	return issues, filesScanned
}

// scanFileForSecurityIssues scans a single file for security issues.
func (sc *SecurityChecker) scanFileForSecurityIssues(filename string) []interfaces.SecurityIssue {
	var issues []interfaces.SecurityIssue

	fullPath := filepath.Join(sc.workDir, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return issues
	}

	fileContent := string(content)
	lines := strings.Split(fileContent, "\n")

	// Check for common security anti-patterns
	for lineNum, line := range lines {
		lineNum++ // Line numbers start at 1

		// Check for hardcoded secrets
		if sc.containsHardcodedSecrets(line) {
			issues = append(issues, interfaces.SecurityIssue{
				ID:          "HARDCODED_SECRET",
				Title:       "Hardcoded secret detected",
				Description: "Potential hardcoded secret or sensitive information found",
				Severity:    "high",
				File:        filename,
				Line:        lineNum,
				Fix:         "Move secrets to environment variables or secure configuration",
			})
		}

		// Check for SQL injection vulnerabilities
		if sc.containsSQLInjectionRisk(line) {
			issues = append(issues, interfaces.SecurityIssue{
				ID:          "SQL_INJECTION_RISK",
				Title:       "SQL injection risk",
				Description: "Potential SQL injection vulnerability detected",
				Severity:    "high",
				File:        filename,
				Line:        lineNum,
				Fix:         "Use parameterized queries or prepared statements",
			})
		}

		// Check for unsafe HTTP usage
		if sc.containsUnsafeHTTP(line) {
			issues = append(issues, interfaces.SecurityIssue{
				ID:          "UNSAFE_HTTP",
				Title:       "Insecure HTTP usage",
				Description: "HTTP used instead of HTTPS for sensitive operations",
				Severity:    "medium",
				File:        filename,
				Line:        lineNum,
				Fix:         "Use HTTPS for all sensitive communications",
			})
		}

		// Check for weak cryptographic practices
		if sc.containsWeakCrypto(line) {
			issues = append(issues, interfaces.SecurityIssue{
				ID:          "WEAK_CRYPTO",
				Title:       "Weak cryptographic practice",
				Description: "Usage of weak or deprecated cryptographic algorithms",
				Severity:    "medium",
				File:        filename,
				Line:        lineNum,
				Fix:         "Use strong, modern cryptographic algorithms",
			})
		}
	}

	return issues
}

// containsHardcodedSecrets checks for potential hardcoded secrets.
func (sc *SecurityChecker) containsHardcodedSecrets(line string) bool {
	secretPatterns := []string{
		`password\s*[:=]+\s*["'][^"']{3,}["']`,
		`secret\s*[:=]+\s*["'][^"']{8,}["']`,
		`token\s*[:=]+\s*["'][^"']{10,}["']`,
		`api[_-]?key\s*[:=]+\s*["'][^"']{8,}["']`,
		`private[_-]?key\s*[:=]+`,
	}

	for _, pattern := range secretPatterns {
		if matched, _ := regexp.MatchString("(?i)"+pattern, line); matched {
			// Exclude common test/example values
			if !strings.Contains(strings.ToLower(line), "test") &&
				!strings.Contains(strings.ToLower(line), "example") &&
				!strings.Contains(strings.ToLower(line), "dummy") {
				return true
			}
		}
	}

	return false
}

// containsSQLInjectionRisk checks for SQL injection vulnerabilities.
func (sc *SecurityChecker) containsSQLInjectionRisk(line string) bool {
	// Look for string concatenation with user input in SQL queries
	sqlPatterns := []string{
		`(?i)(select|insert|update|delete).*\+.*`,
		`(?i)fmt\.sprintf.*select`,
		`(?i)fmt\.sprintf.*insert`,
		`(?i)fmt\.sprintf.*update`,
		`(?i)fmt\.sprintf.*delete`,
	}

	for _, pattern := range sqlPatterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}

// containsUnsafeHTTP checks for insecure HTTP usage.
func (sc *SecurityChecker) containsUnsafeHTTP(line string) bool {
	httpPatterns := []string{
		`"http://[^"]*"`,
		`'http://[^']*'`,
	}

	for _, pattern := range httpPatterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			// Allow localhost and development URLs
			if !strings.Contains(line, "localhost") &&
				!strings.Contains(line, "127.0.0.1") &&
				!strings.Contains(line, "example.com") {
				return true
			}
		}
	}

	return false
}

// containsWeakCrypto checks for weak cryptographic practices.
func (sc *SecurityChecker) containsWeakCrypto(line string) bool {
	weakCryptoPatterns := []string{
		`md5\.New\(\)`,
		`sha1\.New\(\)`,
		`des\.New`,
		`rc4\.New`,
		`rand\.Read\(\)`, // Should use crypto/rand
	}

	for _, pattern := range weakCryptoPatterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}

// filterAndPrioritizeIssues filters and sorts issues by severity.
func (sc *SecurityChecker) filterAndPrioritizeIssues(issues []interfaces.SecurityIssue) []interfaces.SecurityIssue {
	var filteredIssues []interfaces.SecurityIssue

	// Filter by severity threshold
	threshold := sc.getSeverityLevel(sc.config.SeverityThreshold)

	for _, issue := range issues {
		issueLevel := sc.getSeverityLevel(issue.Severity)
		if issueLevel >= threshold {
			filteredIssues = append(filteredIssues, issue)
		}
	}

	// Sort by severity (highest first)
	sort.Slice(filteredIssues, func(i, j int) bool {
		return sc.getSeverityLevel(filteredIssues[i].Severity) >
			sc.getSeverityLevel(filteredIssues[j].Severity)
	})

	// Apply max issues limit
	if sc.config.MaxIssues > 0 && len(filteredIssues) > sc.config.MaxIssues {
		filteredIssues = filteredIssues[:sc.config.MaxIssues]
	}

	return filteredIssues
}

// calculateOverallSeverity determines the overall severity level.
func (sc *SecurityChecker) calculateOverallSeverity(issues []interfaces.SecurityIssue) string {
	if len(issues) == 0 {
		return "none"
	}

	highCount := 0
	mediumCount := 0

	for _, issue := range issues {
		switch issue.Severity {
		case "critical", "high":
			highCount++
		case "medium":
			mediumCount++
		}
	}

	if highCount > 0 {
		return "high"
	}
	if mediumCount > 0 {
		return "medium"
	}

	return "low"
}

// calculateSecurityScore calculates a security score based on issues found.
func (sc *SecurityChecker) calculateSecurityScore(issues []interfaces.SecurityIssue) int {
	if len(issues) == 0 {
		return 100
	}

	score := 100

	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			score -= 30
		case "high":
			score -= 20
		case "medium":
			score -= 10
		case "low":
			score -= 5
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

// getSeverityLevel returns numeric severity level for comparison.
func (sc *SecurityChecker) getSeverityLevel(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// getSuggestionForRule returns fix suggestion for a given rule.
func (sc *SecurityChecker) getSuggestionForRule(ruleID string) string {
	suggestions := map[string]string{
		"G101": "Remove hardcoded credentials and use environment variables",
		"G102": "Use secure binding for network services",
		"G103": "Avoid unsafe block in audit mode",
		"G104": "Handle error return values properly",
		"G201": "Use parameterized queries to prevent SQL injection",
		"G301": "Set appropriate file permissions",
		"G302": "Use secure file permissions for sensitive files",
		"G303": "Ensure secure directory permissions",
		"G401": "Use strong cryptographic hash functions",
		"G501": "Use strong cryptographic algorithms",
		"G505": "Use strong cryptographic key generation",
	}

	if suggestion, exists := suggestions[ruleID]; exists {
		return suggestion
	}

	return "Review and fix the security issue according to best practices"
}

// mapNancySeverity maps Nancy CVSS score to severity level.
func (sc *SecurityChecker) mapNancySeverity(cvssStr string) string {
	cvss, err := strconv.ParseFloat(cvssStr, 64)
	if err != nil {
		return "low"
	}

	if cvss >= 9.0 {
		return "critical"
	} else if cvss >= 7.0 {
		return "high"
	} else if cvss >= 4.0 {
		return "medium"
	}

	return "low"
}

// findGoFiles finds all Go source files.
func (sc *SecurityChecker) findGoFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(sc.workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip vendor and other directories
			if info.Name() == "vendor" || info.Name() == ".git" ||
				info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, ".go") && !sc.isGeneratedFile(path) {
			relPath, err := filepath.Rel(sc.workDir, path)
			if err == nil {
				files = append(files, relPath)
			}
		}

		return nil
	})

	return files, err
}

// isExcludedFile checks if a file should be excluded.
func (sc *SecurityChecker) isExcludedFile(file string) bool {
	for _, pattern := range sc.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(file)); matched {
			return true
		}
	}
	return false
}

// isGeneratedFile checks if a file is generated.
func (sc *SecurityChecker) isGeneratedFile(filename string) bool {
	generatedPatterns := []string{
		".pb.go",
		".pb.gw.go",
		"_gen.go",
		"generated.go",
	}

	for _, pattern := range generatedPatterns {
		if strings.HasSuffix(filename, pattern) {
			return true
		}
	}

	return false
}

// isCommandAvailable checks if a command is available.
func (sc *SecurityChecker) isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// getTimeoutDuration parses timeout configuration.
func (sc *SecurityChecker) getTimeoutDuration() time.Duration {
	if sc.config.Timeout == "" {
		return 5 * time.Minute
	}

	if duration, err := time.ParseDuration(sc.config.Timeout); err == nil {
		return duration
	}

	return 5 * time.Minute
}

// Placeholder implementations for QualityChecker interface compatibility
func (sc *SecurityChecker) CheckCodeStyle() interfaces.CheckResult {
	return interfaces.CheckResult{Name: "code_style", Status: interfaces.CheckStatusSkip}
}

func (sc *SecurityChecker) RunTests() interfaces.TestResult {
	return interfaces.TestResult{}
}

func (sc *SecurityChecker) CheckCoverage() interfaces.CoverageResult {
	return interfaces.CoverageResult{}
}

func (sc *SecurityChecker) ValidateConfig() interfaces.ValidationResult {
	return interfaces.ValidationResult{Valid: true}
}

func (sc *SecurityChecker) CheckInterfaces() interfaces.CheckResult {
	return interfaces.CheckResult{Name: "interfaces", Status: interfaces.CheckStatusSkip}
}

func (sc *SecurityChecker) CheckImports() interfaces.CheckResult {
	return interfaces.CheckResult{Name: "imports", Status: interfaces.CheckStatusSkip}
}

func (sc *SecurityChecker) CheckCopyright() interfaces.CheckResult {
	return interfaces.CheckResult{Name: "copyright", Status: interfaces.CheckStatusSkip}
}

// defaultSecurityConfig returns default security checking configuration.
func defaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		GosecEnabled:      true,
		NancyEnabled:      false, // Requires additional setup
		TrivyEnabled:      false, // Requires additional setup
		SeverityThreshold: "medium",
		ReportFormat:      "json",
		MaxIssues:         50,
		Timeout:           "5m",
		FailOnMedium:      false,
		FailOnHigh:        true,
		ExcludePatterns: []string{
			"vendor/*",
			"*.pb.go",
			"*_test.go", // Test files may have intentionally insecure code
		},
	}
}
