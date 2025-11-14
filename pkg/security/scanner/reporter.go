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
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReportFormat represents the format of the security scan report.
type ReportFormat string

const (
	// ReportFormatJSON represents JSON format.
	ReportFormatJSON ReportFormat = "json"
	// ReportFormatHTML represents HTML format.
	ReportFormatHTML ReportFormat = "html"
	// ReportFormatSARIF represents SARIF format.
	ReportFormatSARIF ReportFormat = "sarif"
	// ReportFormatText represents plain text format.
	ReportFormatText ReportFormat = "text"
)

// Reporter generates security scan reports in various formats.
type Reporter struct {
	outputDir string
}

// NewReporter creates a new Reporter instance.
func NewReporter(outputDir string) *Reporter {
	return &Reporter{
		outputDir: outputDir,
	}
}

// GenerateReport generates a report from scan results in the specified format.
func (r *Reporter) GenerateReport(results []*ScanResult, format ReportFormat) (string, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("security-report-%s.%s", timestamp, format)
	outputPath := filepath.Join(r.outputDir, filename)

	var err error
	switch format {
	case ReportFormatJSON:
		err = r.generateJSONReport(results, outputPath)
	case ReportFormatHTML:
		err = r.generateHTMLReport(results, outputPath)
	case ReportFormatSARIF:
		err = r.generateSARIFReport(results, outputPath)
	case ReportFormatText:
		err = r.generateTextReport(results, outputPath)
	default:
		return "", fmt.Errorf("unsupported report format: %s", format)
	}

	if err != nil {
		return "", err
	}

	return outputPath, nil
}

// generateJSONReport generates a JSON format report.
func (r *Reporter) generateJSONReport(results []*ScanResult, outputPath string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}

// generateHTMLReport generates an HTML format report.
func (r *Reporter) generateHTMLReport(results []*ScanResult, outputPath string) error {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security Scan Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #007bff; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        .summary { display: flex; gap: 20px; margin: 20px 0; flex-wrap: wrap; }
        .summary-card { flex: 1; min-width: 150px; padding: 15px; border-radius: 6px; text-align: center; }
        .critical { background-color: #dc3545; color: white; }
        .high { background-color: #fd7e14; color: white; }
        .medium { background-color: #ffc107; color: black; }
        .low { background-color: #28a745; color: white; }
        .info { background-color: #17a2b8; color: white; }
        .summary-card h3 { margin: 0; font-size: 2em; }
        .summary-card p { margin: 5px 0 0 0; }
        .tool-section { margin: 30px 0; padding: 20px; background: #f8f9fa; border-left: 4px solid #007bff; }
        .finding { margin: 15px 0; padding: 15px; background: white; border-left: 4px solid; border-radius: 4px; }
        .finding.critical { border-left-color: #dc3545; }
        .finding.high { border-left-color: #fd7e14; }
        .finding.medium { border-left-color: #ffc107; }
        .finding.low { border-left-color: #28a745; }
        .finding.info { border-left-color: #17a2b8; }
        .finding-header { display: flex; justify-content: space-between; align-items: start; margin-bottom: 10px; }
        .finding-title { font-weight: bold; font-size: 1.1em; }
        .severity-badge { padding: 4px 12px; border-radius: 12px; font-size: 0.85em; font-weight: bold; text-transform: uppercase; }
        .severity-badge.critical { background: #dc3545; color: white; }
        .severity-badge.high { background: #fd7e14; color: white; }
        .severity-badge.medium { background: #ffc107; color: black; }
        .severity-badge.low { background: #28a745; color: white; }
        .severity-badge.info { background: #17a2b8; color: white; }
        .finding-meta { color: #666; font-size: 0.9em; margin: 5px 0; }
        .finding-description { margin: 10px 0; line-height: 1.6; }
        .finding-remediation { margin: 10px 0; padding: 10px; background: #e7f3ff; border-radius: 4px; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; font-family: 'Courier New', monospace; }
        .timestamp { color: #666; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔒 Security Scan Report</h1>
        <p class="timestamp">Generated: {{.Timestamp}}</p>
        
        <h2>Summary</h2>
        <div class="summary">
            <div class="summary-card critical">
                <h3>{{.TotalCritical}}</h3>
                <p>Critical</p>
            </div>
            <div class="summary-card high">
                <h3>{{.TotalHigh}}</h3>
                <p>High</p>
            </div>
            <div class="summary-card medium">
                <h3>{{.TotalMedium}}</h3>
                <p>Medium</p>
            </div>
            <div class="summary-card low">
                <h3>{{.TotalLow}}</h3>
                <p>Low</p>
            </div>
            <div class="summary-card info">
                <h3>{{.TotalInfo}}</h3>
                <p>Info</p>
            </div>
        </div>

        {{range .Results}}
        <div class="tool-section">
            <h2>🔧 {{.Tool}}</h2>
            <p><strong>Scan Duration:</strong> {{.Duration}}</p>
            <p><strong>Files Scanned:</strong> {{.Summary.FilesScanned}}</p>
            <p><strong>Total Findings:</strong> {{.Summary.TotalFindings}}</p>

            {{if .Findings}}
            {{range .Findings}}
            <div class="finding {{.Severity}}">
                <div class="finding-header">
                    <div class="finding-title">{{.Title}}</div>
                    <span class="severity-badge {{.Severity}}">{{.Severity}}</span>
                </div>
                <div class="finding-meta">
                    <strong>ID:</strong> {{.ID}} | 
                    <strong>Category:</strong> {{.Category}} | 
                    {{if .File}}<strong>File:</strong> <code>{{.File}}</code>{{end}}
                    {{if .Line}} (Line {{.Line}}){{end}}
                </div>
                <div class="finding-description">{{.Description}}</div>
                {{if .Remediation}}
                <div class="finding-remediation">
                    <strong>💡 Remediation:</strong> {{.Remediation}}
                </div>
                {{end}}
                {{if .CWE}}
                <div class="finding-meta"><strong>CWE:</strong> {{.CWE}}</div>
                {{end}}
                {{if .CVSS}}
                <div class="finding-meta"><strong>CVSS Score:</strong> {{.CVSS}}</div>
                {{end}}
            </div>
            {{end}}
            {{else}}
            <p>✅ No security findings detected.</p>
            {{end}}
        </div>
        {{end}}
    </div>
</body>
</html>`

	// Prepare template data
	data := struct {
		Timestamp     string
		TotalCritical int
		TotalHigh     int
		TotalMedium   int
		TotalLow      int
		TotalInfo     int
		Results       []*ScanResult
	}{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Results:   results,
	}

	// Calculate totals
	severityCounts := GetFindingsBySeverity(results)
	data.TotalCritical = severityCounts[SeverityCritical]
	data.TotalHigh = severityCounts[SeverityHigh]
	data.TotalMedium = severityCounts[SeverityMedium]
	data.TotalLow = severityCounts[SeverityLow]
	data.TotalInfo = severityCounts[SeverityInfo]

	// Parse and execute template
	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := t.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// generateSARIFReport generates a SARIF format report.
func (r *Reporter) generateSARIFReport(results []*ScanResult, outputPath string) error {
	// SARIF (Static Analysis Results Interchange Format) v2.1.0
	sarif := map[string]interface{}{
		"version": "2.1.0",
		"$schema": "https://json.schemastore.org/sarif-2.1.0.json",
		"runs":    []interface{}{},
	}

	for _, result := range results {
		run := map[string]interface{}{
			"tool": map[string]interface{}{
				"driver": map[string]interface{}{
					"name":           result.Tool,
					"informationUri": r.getToolURI(result.Tool),
				},
			},
			"results": []interface{}{},
		}

		for _, finding := range result.Findings {
			sarifResult := map[string]interface{}{
				"ruleId": finding.ID,
				"level":  r.mapSeverityToSARIFLevel(finding.Severity),
				"message": map[string]interface{}{
					"text": finding.Description,
				},
			}

			// Add location if available
			if finding.File != "" {
				sarifResult["locations"] = []interface{}{
					map[string]interface{}{
						"physicalLocation": map[string]interface{}{
							"artifactLocation": map[string]interface{}{
								"uri": finding.File,
							},
							"region": map[string]interface{}{
								"startLine":   finding.Line,
								"startColumn": finding.Column,
							},
						},
					},
				}
			}

			run["results"] = append(run["results"].([]interface{}), sarifResult)
		}

		sarif["runs"] = append(sarif["runs"].([]interface{}), run)
	}

	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SARIF: %w", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}

// generateTextReport generates a plain text format report.
func (r *Reporter) generateTextReport(results []*ScanResult, outputPath string) error {
	var sb strings.Builder

	sb.WriteString("=================================================\n")
	sb.WriteString("           SECURITY SCAN REPORT\n")
	sb.WriteString("=================================================\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Summary
	severityCounts := GetFindingsBySeverity(results)
	totalFindings := GetTotalFindings(results)

	sb.WriteString("SUMMARY\n")
	sb.WriteString("-------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("Total Findings:  %d\n", totalFindings))
	sb.WriteString(fmt.Sprintf("  Critical:      %d\n", severityCounts[SeverityCritical]))
	sb.WriteString(fmt.Sprintf("  High:          %d\n", severityCounts[SeverityHigh]))
	sb.WriteString(fmt.Sprintf("  Medium:        %d\n", severityCounts[SeverityMedium]))
	sb.WriteString(fmt.Sprintf("  Low:           %d\n", severityCounts[SeverityLow]))
	sb.WriteString(fmt.Sprintf("  Info:          %d\n", severityCounts[SeverityInfo]))
	sb.WriteString("\n")

	// Detailed findings
	for _, result := range results {
		sb.WriteString(fmt.Sprintf("\n=== %s ===\n", strings.ToUpper(result.Tool)))
		sb.WriteString(fmt.Sprintf("Duration: %s\n", result.Duration))
		sb.WriteString(fmt.Sprintf("Files Scanned: %d\n", result.Summary.FilesScanned))
		sb.WriteString(fmt.Sprintf("Findings: %d\n\n", result.Summary.TotalFindings))

		if len(result.Findings) == 0 {
			sb.WriteString("✓ No security findings detected.\n")
			continue
		}

		for i, finding := range result.Findings {
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, strings.ToUpper(string(finding.Severity)), finding.Title))
			sb.WriteString(fmt.Sprintf("   ID: %s\n", finding.ID))
			sb.WriteString(fmt.Sprintf("   Category: %s\n", finding.Category))
			if finding.File != "" {
				sb.WriteString(fmt.Sprintf("   Location: %s", finding.File))
				if finding.Line > 0 {
					sb.WriteString(fmt.Sprintf(":%d", finding.Line))
				}
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("   Description: %s\n", finding.Description))
			if finding.Remediation != "" {
				sb.WriteString(fmt.Sprintf("   Remediation: %s\n", finding.Remediation))
			}
			sb.WriteString("\n")
		}
	}

	return os.WriteFile(outputPath, []byte(sb.String()), 0644)
}

// mapSeverityToSARIFLevel maps our severity levels to SARIF levels.
func (r *Reporter) mapSeverityToSARIFLevel(severity Severity) string {
	switch severity {
	case SeverityCritical, SeverityHigh:
		return "error"
	case SeverityMedium:
		return "warning"
	case SeverityLow, SeverityInfo:
		return "note"
	default:
		return "none"
	}
}

// getToolURI returns the documentation URI for a scanning tool.
func (r *Reporter) getToolURI(tool string) string {
	uris := map[string]string{
		"gosec":       "https://github.com/securego/gosec",
		"govulncheck": "https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck",
		"trivy":       "https://github.com/aquasecurity/trivy",
	}

	if uri, ok := uris[tool]; ok {
		return uri
	}
	return ""
}
