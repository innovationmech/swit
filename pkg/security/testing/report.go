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

package testing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Severity levels for security findings.
const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityMedium   = "MEDIUM"
	SeverityLow      = "LOW"
	SeverityInfo     = "INFO"
)

// SecurityAuditReport represents a comprehensive security audit report.
type SecurityAuditReport struct {
	// Metadata
	Title       string    `json:"title"`
	Version     string    `json:"version"`
	GeneratedAt time.Time `json:"generated_at"`
	GeneratedBy string    `json:"generated_by"`

	// Summary
	Summary ReportSummary `json:"summary"`

	// Test Results by Category
	OWASPResults       []SecurityFindingResult `json:"owasp_results"`
	PenetrationResults []SecurityFindingResult `json:"penetration_results"`
	FuzzingResults     []SecurityFindingResult `json:"fuzzing_results"`
	TimingResults      []SecurityFindingResult `json:"timing_results"`
	DoSResults         []SecurityFindingResult `json:"dos_results"`

	// Recommendations
	Recommendations []Recommendation `json:"recommendations"`

	// Compliance Status
	ComplianceStatus ComplianceStatus `json:"compliance_status"`
}

// ReportSummary provides an overview of the security audit.
type ReportSummary struct {
	TotalTests       int           `json:"total_tests"`
	PassedTests      int           `json:"passed_tests"`
	FailedTests      int           `json:"failed_tests"`
	SkippedTests     int           `json:"skipped_tests"`
	CriticalFindings int           `json:"critical_findings"`
	HighFindings     int           `json:"high_findings"`
	MediumFindings   int           `json:"medium_findings"`
	LowFindings      int           `json:"low_findings"`
	InfoFindings     int           `json:"info_findings"`
	OverallScore     float64       `json:"overall_score"`
	RiskLevel        string        `json:"risk_level"`
	TestDuration     time.Duration `json:"test_duration"`
	Coverage         TestCoverage  `json:"coverage"`
}

// TestCoverage represents the coverage of security tests.
type TestCoverage struct {
	OWASPTop10        float64 `json:"owasp_top_10"`
	Authentication    float64 `json:"authentication"`
	Authorization     float64 `json:"authorization"`
	InputValidation   float64 `json:"input_validation"`
	Cryptography      float64 `json:"cryptography"`
	SessionManagement float64 `json:"session_management"`
}

// SecurityFindingResult represents a single security test result.
type SecurityFindingResult struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Category    string                 `json:"category"`
	Severity    string                 `json:"severity"`
	Status      string                 `json:"status"` // PASSED, FAILED, SKIPPED
	Description string                 `json:"description"`
	Impact      string                 `json:"impact"`
	Remediation string                 `json:"remediation"`
	References  []string               `json:"references"`
	Evidence    string                 `json:"evidence,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	TestedAt    time.Time              `json:"tested_at"`
}

// Recommendation represents a security recommendation.
type Recommendation struct {
	Priority    string `json:"priority"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Effort      string `json:"effort"` // LOW, MEDIUM, HIGH
	Impact      string `json:"impact"` // LOW, MEDIUM, HIGH
}

// ComplianceStatus represents compliance with security standards.
type ComplianceStatus struct {
	OWASPTop10 ComplianceDetail `json:"owasp_top_10"`
	CWETop25   ComplianceDetail `json:"cwe_top_25"`
	SANSTop25  ComplianceDetail `json:"sans_top_25"`
}

// ComplianceDetail provides details about compliance with a standard.
type ComplianceDetail struct {
	Standard    string  `json:"standard"`
	Version     string  `json:"version"`
	Compliant   bool    `json:"compliant"`
	Score       float64 `json:"score"`
	PassedItems int     `json:"passed_items"`
	TotalItems  int     `json:"total_items"`
	Details     string  `json:"details"`
}

// SecurityAuditReportBuilder builds security audit reports.
type SecurityAuditReportBuilder struct {
	report *SecurityAuditReport
}

// NewSecurityAuditReportBuilder creates a new report builder.
func NewSecurityAuditReportBuilder() *SecurityAuditReportBuilder {
	return &SecurityAuditReportBuilder{
		report: &SecurityAuditReport{
			Title:       "Security Audit Report",
			Version:     "1.0",
			GeneratedAt: time.Now(),
			GeneratedBy: "swit-security-testing",
			Summary: ReportSummary{
				Coverage: TestCoverage{},
			},
			OWASPResults:       []SecurityFindingResult{},
			PenetrationResults: []SecurityFindingResult{},
			FuzzingResults:     []SecurityFindingResult{},
			TimingResults:      []SecurityFindingResult{},
			DoSResults:         []SecurityFindingResult{},
			Recommendations:    []Recommendation{},
		},
	}
}

// SetTitle sets the report title.
func (b *SecurityAuditReportBuilder) SetTitle(title string) *SecurityAuditReportBuilder {
	b.report.Title = title
	return b
}

// SetVersion sets the report version.
func (b *SecurityAuditReportBuilder) SetVersion(version string) *SecurityAuditReportBuilder {
	b.report.Version = version
	return b
}

// AddOWASPResult adds an OWASP test result.
func (b *SecurityAuditReportBuilder) AddOWASPResult(result SecurityFindingResult) *SecurityAuditReportBuilder {
	result.TestedAt = time.Now()
	b.report.OWASPResults = append(b.report.OWASPResults, result)
	b.updateSummary(result)
	return b
}

// AddPenetrationResult adds a penetration test result.
func (b *SecurityAuditReportBuilder) AddPenetrationResult(result SecurityFindingResult) *SecurityAuditReportBuilder {
	result.TestedAt = time.Now()
	b.report.PenetrationResults = append(b.report.PenetrationResults, result)
	b.updateSummary(result)
	return b
}

// AddFuzzingResult adds a fuzzing test result.
func (b *SecurityAuditReportBuilder) AddFuzzingResult(result SecurityFindingResult) *SecurityAuditReportBuilder {
	result.TestedAt = time.Now()
	b.report.FuzzingResults = append(b.report.FuzzingResults, result)
	b.updateSummary(result)
	return b
}

// AddTimingResult adds a timing test result.
func (b *SecurityAuditReportBuilder) AddTimingResult(result SecurityFindingResult) *SecurityAuditReportBuilder {
	result.TestedAt = time.Now()
	b.report.TimingResults = append(b.report.TimingResults, result)
	b.updateSummary(result)
	return b
}

// AddDoSResult adds a DoS test result.
func (b *SecurityAuditReportBuilder) AddDoSResult(result SecurityFindingResult) *SecurityAuditReportBuilder {
	result.TestedAt = time.Now()
	b.report.DoSResults = append(b.report.DoSResults, result)
	b.updateSummary(result)
	return b
}

// AddRecommendation adds a recommendation.
func (b *SecurityAuditReportBuilder) AddRecommendation(rec Recommendation) *SecurityAuditReportBuilder {
	b.report.Recommendations = append(b.report.Recommendations, rec)
	return b
}

// SetTestDuration sets the total test duration.
func (b *SecurityAuditReportBuilder) SetTestDuration(duration time.Duration) *SecurityAuditReportBuilder {
	b.report.Summary.TestDuration = duration
	return b
}

// SetCoverage sets the test coverage.
func (b *SecurityAuditReportBuilder) SetCoverage(coverage TestCoverage) *SecurityAuditReportBuilder {
	b.report.Summary.Coverage = coverage
	return b
}

func (b *SecurityAuditReportBuilder) updateSummary(result SecurityFindingResult) {
	b.report.Summary.TotalTests++

	switch result.Status {
	case "PASSED":
		b.report.Summary.PassedTests++
	case "FAILED":
		b.report.Summary.FailedTests++
		switch result.Severity {
		case SeverityCritical:
			b.report.Summary.CriticalFindings++
		case SeverityHigh:
			b.report.Summary.HighFindings++
		case SeverityMedium:
			b.report.Summary.MediumFindings++
		case SeverityLow:
			b.report.Summary.LowFindings++
		case SeverityInfo:
			b.report.Summary.InfoFindings++
		}
	case "SKIPPED":
		b.report.Summary.SkippedTests++
	}
}

// Build finalizes and returns the report.
func (b *SecurityAuditReportBuilder) Build() *SecurityAuditReport {
	// Calculate overall score
	b.calculateOverallScore()

	// Determine risk level
	b.determineRiskLevel()

	// Calculate compliance status
	b.calculateComplianceStatus()

	// Generate recommendations if not already added
	if len(b.report.Recommendations) == 0 {
		b.generateRecommendations()
	}

	return b.report
}

func (b *SecurityAuditReportBuilder) calculateOverallScore() {
	if b.report.Summary.TotalTests == 0 {
		b.report.Summary.OverallScore = 100.0
		return
	}

	// Base score from pass rate
	passRate := float64(b.report.Summary.PassedTests) / float64(b.report.Summary.TotalTests) * 100

	// Deductions for findings
	deductions := float64(b.report.Summary.CriticalFindings)*20 +
		float64(b.report.Summary.HighFindings)*10 +
		float64(b.report.Summary.MediumFindings)*5 +
		float64(b.report.Summary.LowFindings)*2

	score := passRate - deductions
	if score < 0 {
		score = 0
	}

	b.report.Summary.OverallScore = score
}

func (b *SecurityAuditReportBuilder) determineRiskLevel() {
	if b.report.Summary.CriticalFindings > 0 {
		b.report.Summary.RiskLevel = "CRITICAL"
	} else if b.report.Summary.HighFindings > 0 {
		b.report.Summary.RiskLevel = "HIGH"
	} else if b.report.Summary.MediumFindings > 0 {
		b.report.Summary.RiskLevel = "MEDIUM"
	} else if b.report.Summary.LowFindings > 0 {
		b.report.Summary.RiskLevel = "LOW"
	} else {
		b.report.Summary.RiskLevel = "MINIMAL"
	}
}

func (b *SecurityAuditReportBuilder) calculateComplianceStatus() {
	// Calculate OWASP Top 10 compliance
	owaspPassed := 0
	owaspTotal := len(b.report.OWASPResults)
	for _, r := range b.report.OWASPResults {
		if r.Status == "PASSED" {
			owaspPassed++
		}
	}

	owaspScore := 0.0
	if owaspTotal > 0 {
		owaspScore = float64(owaspPassed) / float64(owaspTotal) * 100
	}

	b.report.ComplianceStatus.OWASPTop10 = ComplianceDetail{
		Standard:    "OWASP Top 10",
		Version:     "2021",
		Compliant:   owaspScore >= 80,
		Score:       owaspScore,
		PassedItems: owaspPassed,
		TotalItems:  owaspTotal,
		Details:     fmt.Sprintf("%d of %d OWASP Top 10 tests passed", owaspPassed, owaspTotal),
	}

	// Simplified CWE and SANS compliance (would need actual mapping in production)
	b.report.ComplianceStatus.CWETop25 = ComplianceDetail{
		Standard:    "CWE Top 25",
		Version:     "2023",
		Compliant:   b.report.Summary.OverallScore >= 80,
		Score:       b.report.Summary.OverallScore,
		PassedItems: b.report.Summary.PassedTests,
		TotalItems:  b.report.Summary.TotalTests,
	}

	b.report.ComplianceStatus.SANSTop25 = ComplianceDetail{
		Standard:    "SANS Top 25",
		Version:     "2023",
		Compliant:   b.report.Summary.OverallScore >= 80,
		Score:       b.report.Summary.OverallScore,
		PassedItems: b.report.Summary.PassedTests,
		TotalItems:  b.report.Summary.TotalTests,
	}
}

func (b *SecurityAuditReportBuilder) generateRecommendations() {
	// Generate recommendations based on findings
	if b.report.Summary.CriticalFindings > 0 {
		b.report.Recommendations = append(b.report.Recommendations, Recommendation{
			Priority:    "CRITICAL",
			Title:       "Address Critical Security Vulnerabilities",
			Description: fmt.Sprintf("There are %d critical security vulnerabilities that require immediate attention.", b.report.Summary.CriticalFindings),
			Category:    "Security",
			Effort:      "HIGH",
			Impact:      "HIGH",
		})
	}

	if b.report.Summary.HighFindings > 0 {
		b.report.Recommendations = append(b.report.Recommendations, Recommendation{
			Priority:    "HIGH",
			Title:       "Remediate High-Severity Issues",
			Description: fmt.Sprintf("There are %d high-severity security issues that should be addressed in the next sprint.", b.report.Summary.HighFindings),
			Category:    "Security",
			Effort:      "MEDIUM",
			Impact:      "HIGH",
		})
	}

	// Add general recommendations
	b.report.Recommendations = append(b.report.Recommendations, Recommendation{
		Priority:    "MEDIUM",
		Title:       "Implement Security Headers",
		Description: "Ensure all security headers (CSP, X-Frame-Options, etc.) are properly configured.",
		Category:    "Configuration",
		Effort:      "LOW",
		Impact:      "MEDIUM",
	})

	b.report.Recommendations = append(b.report.Recommendations, Recommendation{
		Priority:    "MEDIUM",
		Title:       "Enable Rate Limiting",
		Description: "Implement rate limiting on all API endpoints to prevent abuse.",
		Category:    "DoS Protection",
		Effort:      "LOW",
		Impact:      "HIGH",
	})

	b.report.Recommendations = append(b.report.Recommendations, Recommendation{
		Priority:    "LOW",
		Title:       "Regular Security Audits",
		Description: "Schedule regular security audits and penetration testing.",
		Category:    "Process",
		Effort:      "MEDIUM",
		Impact:      "HIGH",
	})
}

// ============================================================================
// Report Output Methods
// ============================================================================

// ToJSON converts the report to JSON format.
func (r *SecurityAuditReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// ToHTML converts the report to HTML format.
func (r *SecurityAuditReport) ToHTML() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 { color: #333; border-bottom: 2px solid #4CAF50; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        .summary-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }
        .summary-card {
            background: #f9f9f9;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
        }
        .summary-card h3 { margin: 0; font-size: 2em; }
        .summary-card p { margin: 5px 0 0; color: #666; }
        .critical { color: #d32f2f; }
        .high { color: #f57c00; }
        .medium { color: #fbc02d; }
        .low { color: #4caf50; }
        .info { color: #2196f3; }
        .passed { background-color: #e8f5e9; }
        .failed { background-color: #ffebee; }
        .skipped { background-color: #fff3e0; }
        table {
            width: 100%%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th { background-color: #4CAF50; color: white; }
        tr:hover { background-color: #f5f5f5; }
        .severity-badge {
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 0.85em;
            font-weight: bold;
        }
        .severity-critical { background: #d32f2f; color: white; }
        .severity-high { background: #f57c00; color: white; }
        .severity-medium { background: #fbc02d; color: black; }
        .severity-low { background: #4caf50; color: white; }
        .severity-info { background: #2196f3; color: white; }
        .score-meter {
            width: 100%%;
            height: 20px;
            background: #e0e0e0;
            border-radius: 10px;
            overflow: hidden;
        }
        .score-fill {
            height: 100%%;
            border-radius: 10px;
            transition: width 0.5s ease;
        }
        .recommendation {
            background: #f5f5f5;
            padding: 15px;
            margin: 10px 0;
            border-left: 4px solid #4CAF50;
            border-radius: 4px;
        }
        .recommendation.critical { border-left-color: #d32f2f; }
        .recommendation.high { border-left-color: #f57c00; }
        .recommendation.medium { border-left-color: #fbc02d; }
        .recommendation.low { border-left-color: #4caf50; }
        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #ddd;
            text-align: center;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>%s</h1>
        <p><strong>Generated:</strong> %s</p>
        <p><strong>Version:</strong> %s</p>

        <h2>Executive Summary</h2>
        <div class="summary-grid">
            <div class="summary-card">
                <h3>%d</h3>
                <p>Total Tests</p>
            </div>
            <div class="summary-card passed">
                <h3>%d</h3>
                <p>Passed</p>
            </div>
            <div class="summary-card failed">
                <h3>%d</h3>
                <p>Failed</p>
            </div>
            <div class="summary-card">
                <h3 class="critical">%d</h3>
                <p>Critical</p>
            </div>
            <div class="summary-card">
                <h3 class="high">%d</h3>
                <p>High</p>
            </div>
            <div class="summary-card">
                <h3 class="medium">%d</h3>
                <p>Medium</p>
            </div>
        </div>

        <h3>Overall Security Score</h3>
        <div class="score-meter">
            <div class="score-fill" style="width: %.1f%%; background: %s;"></div>
        </div>
        <p style="text-align: center; font-size: 1.5em; font-weight: bold;">%.1f%% - Risk Level: %s</p>
`, r.Title, r.Title, r.GeneratedAt.Format(time.RFC3339), r.Version,
		r.Summary.TotalTests, r.Summary.PassedTests, r.Summary.FailedTests,
		r.Summary.CriticalFindings, r.Summary.HighFindings, r.Summary.MediumFindings,
		r.Summary.OverallScore, getScoreColor(r.Summary.OverallScore), r.Summary.OverallScore, r.Summary.RiskLevel))

	// OWASP Results
	if len(r.OWASPResults) > 0 {
		sb.WriteString(`
        <h2>OWASP Top 10 Test Results</h2>
        <table>
            <tr>
                <th>Test</th>
                <th>Category</th>
                <th>Severity</th>
                <th>Status</th>
                <th>Description</th>
            </tr>
`)
		for _, result := range r.OWASPResults {
			sb.WriteString(fmt.Sprintf(`            <tr class="%s">
                <td>%s</td>
                <td>%s</td>
                <td><span class="severity-badge severity-%s">%s</span></td>
                <td>%s</td>
                <td>%s</td>
            </tr>
`, strings.ToLower(result.Status), result.Name, result.Category,
				strings.ToLower(result.Severity), result.Severity, result.Status, result.Description))
		}
		sb.WriteString(`        </table>
`)
	}

	// Penetration Test Results
	if len(r.PenetrationResults) > 0 {
		sb.WriteString(`
        <h2>Penetration Test Results</h2>
        <table>
            <tr>
                <th>Test</th>
                <th>Category</th>
                <th>Severity</th>
                <th>Status</th>
                <th>Description</th>
            </tr>
`)
		for _, result := range r.PenetrationResults {
			sb.WriteString(fmt.Sprintf(`            <tr class="%s">
                <td>%s</td>
                <td>%s</td>
                <td><span class="severity-badge severity-%s">%s</span></td>
                <td>%s</td>
                <td>%s</td>
            </tr>
`, strings.ToLower(result.Status), result.Name, result.Category,
				strings.ToLower(result.Severity), result.Severity, result.Status, result.Description))
		}
		sb.WriteString(`        </table>
`)
	}

	// Recommendations
	if len(r.Recommendations) > 0 {
		sb.WriteString(`
        <h2>Recommendations</h2>
`)
		for _, rec := range r.Recommendations {
			sb.WriteString(fmt.Sprintf(`        <div class="recommendation %s">
            <strong>%s</strong> - %s
            <p>%s</p>
            <p><em>Effort: %s | Impact: %s</em></p>
        </div>
`, strings.ToLower(rec.Priority), rec.Priority, rec.Title, rec.Description, rec.Effort, rec.Impact))
		}
	}

	// Compliance Status
	sb.WriteString(fmt.Sprintf(`
        <h2>Compliance Status</h2>
        <table>
            <tr>
                <th>Standard</th>
                <th>Version</th>
                <th>Score</th>
                <th>Status</th>
            </tr>
            <tr>
                <td>%s</td>
                <td>%s</td>
                <td>%.1f%%</td>
                <td>%s</td>
            </tr>
            <tr>
                <td>%s</td>
                <td>%s</td>
                <td>%.1f%%</td>
                <td>%s</td>
            </tr>
            <tr>
                <td>%s</td>
                <td>%s</td>
                <td>%.1f%%</td>
                <td>%s</td>
            </tr>
        </table>
`,
		r.ComplianceStatus.OWASPTop10.Standard, r.ComplianceStatus.OWASPTop10.Version,
		r.ComplianceStatus.OWASPTop10.Score, getComplianceStatus(r.ComplianceStatus.OWASPTop10.Compliant),
		r.ComplianceStatus.CWETop25.Standard, r.ComplianceStatus.CWETop25.Version,
		r.ComplianceStatus.CWETop25.Score, getComplianceStatus(r.ComplianceStatus.CWETop25.Compliant),
		r.ComplianceStatus.SANSTop25.Standard, r.ComplianceStatus.SANSTop25.Version,
		r.ComplianceStatus.SANSTop25.Score, getComplianceStatus(r.ComplianceStatus.SANSTop25.Compliant)))

	sb.WriteString(fmt.Sprintf(`
        <div class="footer">
            <p>Generated by %s | Test Duration: %v</p>
        </div>
    </div>
</body>
</html>
`, r.GeneratedBy, r.Summary.TestDuration))

	return sb.String()
}

// ToMarkdown converts the report to Markdown format.
func (r *SecurityAuditReport) ToMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", r.Title))
	sb.WriteString(fmt.Sprintf("**Generated:** %s  \n", r.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Version:** %s  \n\n", r.Version))

	sb.WriteString("## Executive Summary\n\n")
	sb.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	sb.WriteString(fmt.Sprintf("|--------|-------|\n"))
	sb.WriteString(fmt.Sprintf("| Total Tests | %d |\n", r.Summary.TotalTests))
	sb.WriteString(fmt.Sprintf("| Passed | %d |\n", r.Summary.PassedTests))
	sb.WriteString(fmt.Sprintf("| Failed | %d |\n", r.Summary.FailedTests))
	sb.WriteString(fmt.Sprintf("| Critical Findings | %d |\n", r.Summary.CriticalFindings))
	sb.WriteString(fmt.Sprintf("| High Findings | %d |\n", r.Summary.HighFindings))
	sb.WriteString(fmt.Sprintf("| Medium Findings | %d |\n", r.Summary.MediumFindings))
	sb.WriteString(fmt.Sprintf("| Overall Score | %.1f%% |\n", r.Summary.OverallScore))
	sb.WriteString(fmt.Sprintf("| Risk Level | %s |\n\n", r.Summary.RiskLevel))

	if len(r.OWASPResults) > 0 {
		sb.WriteString("## OWASP Top 10 Results\n\n")
		sb.WriteString("| Test | Severity | Status | Description |\n")
		sb.WriteString("|------|----------|--------|-------------|\n")
		for _, result := range r.OWASPResults {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				result.Name, result.Severity, result.Status, result.Description))
		}
		sb.WriteString("\n")
	}

	if len(r.Recommendations) > 0 {
		sb.WriteString("## Recommendations\n\n")
		for _, rec := range r.Recommendations {
			sb.WriteString(fmt.Sprintf("### %s: %s\n\n", rec.Priority, rec.Title))
			sb.WriteString(fmt.Sprintf("%s\n\n", rec.Description))
			sb.WriteString(fmt.Sprintf("- **Category:** %s\n", rec.Category))
			sb.WriteString(fmt.Sprintf("- **Effort:** %s\n", rec.Effort))
			sb.WriteString(fmt.Sprintf("- **Impact:** %s\n\n", rec.Impact))
		}
	}

	return sb.String()
}

// SaveToFile saves the report to a file.
func (r *SecurityAuditReport) SaveToFile(path string) error {
	// Determine format from extension
	ext := strings.ToLower(filepath.Ext(path))

	var content []byte
	var err error

	switch ext {
	case ".json":
		content, err = r.ToJSON()
	case ".html", ".htm":
		content = []byte(r.ToHTML())
	case ".md", ".markdown":
		content = []byte(r.ToMarkdown())
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, content, 0644)
}

// Helper functions

func getScoreColor(score float64) string {
	if score >= 80 {
		return "#4caf50"
	} else if score >= 60 {
		return "#fbc02d"
	} else if score >= 40 {
		return "#f57c00"
	}
	return "#d32f2f"
}

func getComplianceStatus(compliant bool) string {
	if compliant {
		return "✅ Compliant"
	}
	return "❌ Non-Compliant"
}
