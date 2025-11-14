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
	"strings"
	"time"
)

// GovulncheckScanner implements the ScanTool interface for govulncheck vulnerability scanner.
type GovulncheckScanner struct{}

// NewGovulncheckScanner creates a new GovulncheckScanner instance.
func NewGovulncheckScanner() *GovulncheckScanner {
	return &GovulncheckScanner{}
}

// Name returns the name of the scanning tool.
func (g *GovulncheckScanner) Name() string {
	return "govulncheck"
}

// IsAvailable checks if govulncheck is installed and available.
func (g *GovulncheckScanner) IsAvailable() bool {
	_, err := exec.LookPath("govulncheck")
	return err == nil
}

// Scan performs a vulnerability scan using govulncheck.
func (g *GovulncheckScanner) Scan(ctx context.Context, target string) (*ScanResult, error) {
	startTime := time.Now()

	// Build govulncheck command with JSON output
	args := []string{"-json"}

	// Add target
	args = append(args, target)

	// Execute govulncheck
	cmd := exec.CommandContext(ctx, "govulncheck", args...)
	output, err := cmd.CombinedOutput()

	// govulncheck returns non-zero exit code when vulnerabilities are found
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			// This is a real error, not just findings
			return nil, fmt.Errorf("failed to execute govulncheck: %w", err)
		}
	}

	// Parse govulncheck output
	result, err := g.parseGovulncheckOutput(output, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse govulncheck output: %w", err)
	}

	return result, nil
}

// govulncheckMessage represents a single message from govulncheck JSON output.
type govulncheckMessage struct {
	Vulnerability *struct {
		OSV struct {
			ID       string `json:"id"`
			Summary  string `json:"summary"`
			Details  string `json:"details"`
			Severity []struct {
				Type  string  `json:"type"`
				Score string  `json:"score"`
				Value float64 `json:"-"` // Parsed from Score
			} `json:"severity"`
			DatabaseSpecific struct {
				URL string `json:"url"`
			} `json:"database_specific"`
		} `json:"osv"`
		Modules []struct {
			Path         string `json:"path"`
			FoundVersion string `json:"found_version"`
			FixedVersion string `json:"fixed_version"`
			Packages     []struct {
				Path    string `json:"path"`
				Symbols []struct {
					Name string `json:"name"`
				} `json:"symbols"`
			} `json:"packages"`
		} `json:"modules"`
	} `json:"vulnerability,omitempty"`
	Config *struct {
		ScannerName    string `json:"scanner_name"`
		ScannerVersion string `json:"scanner_version"`
		DB             string `json:"db"`
		DBLastModified string `json:"db_last_modified"`
	} `json:"config,omitempty"`
}

// parseGovulncheckOutput parses the JSON output from govulncheck and converts it to ScanResult.
func (g *GovulncheckScanner) parseGovulncheckOutput(output []byte, startTime time.Time) (*ScanResult, error) {
	result := &ScanResult{
		Tool:      g.Name(),
		Timestamp: startTime,
		Duration:  time.Since(startTime),
		Findings:  make([]SecurityFinding, 0),
		Summary:   ScanSummary{},
	}

	// govulncheck outputs one JSON object per line
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var msg govulncheckMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Skip lines that can't be parsed
			continue
		}

		// Process vulnerability messages
		if msg.Vulnerability != nil {
			findings := g.convertVulnerabilityToFindings(msg.Vulnerability)
			result.Findings = append(result.Findings, findings...)

			// Update summary counts
			for _, finding := range findings {
				result.Summary.TotalFindings++
				switch finding.Severity {
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
		}
	}

	return result, nil
}

// convertVulnerabilityToFindings converts a govulncheck vulnerability to SecurityFindings.
func (g *GovulncheckScanner) convertVulnerabilityToFindings(vuln *struct {
	OSV struct {
		ID       string `json:"id"`
		Summary  string `json:"summary"`
		Details  string `json:"details"`
		Severity []struct {
			Type  string  `json:"type"`
			Score string  `json:"score"`
			Value float64 `json:"-"`
		} `json:"severity"`
		DatabaseSpecific struct {
			URL string `json:"url"`
		} `json:"database_specific"`
	} `json:"osv"`
	Modules []struct {
		Path         string `json:"path"`
		FoundVersion string `json:"found_version"`
		FixedVersion string `json:"fixed_version"`
		Packages     []struct {
			Path    string `json:"path"`
			Symbols []struct {
				Name string `json:"name"`
			} `json:"symbols"`
		} `json:"packages"`
	} `json:"modules"`
}) []SecurityFinding {
	var findings []SecurityFinding

	// Determine severity based on CVSS score
	severity := g.mapCVSSSeverity(vuln.OSV.Severity)

	// Get CVSS score if available
	var cvss float64
	for _, sev := range vuln.OSV.Severity {
		if sev.Type == "CVSS_V3" {
			// Parse CVSS score from string like "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
			cvss = g.parseCVSSScore(sev.Score)
			break
		}
	}

	// Create a finding for each affected module
	for _, module := range vuln.Modules {
		remediation := fmt.Sprintf("Update %s from version %s to %s or later",
			module.Path, module.FoundVersion, module.FixedVersion)

		if module.FixedVersion == "" {
			remediation = fmt.Sprintf("No fix available yet for %s (current version: %s). Monitor for updates.",
				module.Path, module.FoundVersion)
		}

		var references []string
		if vuln.OSV.DatabaseSpecific.URL != "" {
			references = append(references, vuln.OSV.DatabaseSpecific.URL)
		}
		references = append(references, fmt.Sprintf("https://pkg.go.dev/vuln/%s", vuln.OSV.ID))

		finding := SecurityFinding{
			ID:          vuln.OSV.ID,
			Severity:    severity,
			Category:    "Dependency Vulnerability",
			Title:       vuln.OSV.Summary,
			Description: fmt.Sprintf("%s\n\nAffected module: %s@%s", vuln.OSV.Details, module.Path, module.FoundVersion),
			File:        "go.mod",
			Remediation: remediation,
			CVSS:        cvss,
			References:  references,
		}

		findings = append(findings, finding)
	}

	return findings
}

// mapCVSSSeverity maps CVSS score to severity level.
func (g *GovulncheckScanner) mapCVSSSeverity(severities []struct {
	Type  string  `json:"type"`
	Score string  `json:"score"`
	Value float64 `json:"-"`
}) Severity {
	// Find CVSS score
	var cvss float64
	for _, sev := range severities {
		if sev.Type == "CVSS_V3" {
			cvss = g.parseCVSSScore(sev.Score)
			break
		}
	}

	// Map CVSS score to severity
	// CVSS v3.0 Ratings:
	// None: 0.0
	// Low: 0.1-3.9
	// Medium: 4.0-6.9
	// High: 7.0-8.9
	// Critical: 9.0-10.0
	switch {
	case cvss >= 9.0:
		return SeverityCritical
	case cvss >= 7.0:
		return SeverityHigh
	case cvss >= 4.0:
		return SeverityMedium
	case cvss > 0:
		return SeverityLow
	default:
		return SeverityInfo
	}
}

// parseCVSSScore parses CVSS score from string format.
func (g *GovulncheckScanner) parseCVSSScore(scoreStr string) float64 {
	// CVSS score string format: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
	// We need to calculate the actual score based on the vector
	// For simplicity, we'll map common patterns to scores
	// In production, you'd want to use a proper CVSS calculator

	// Common mappings based on impact levels
	if strings.Contains(scoreStr, "C:H") && strings.Contains(scoreStr, "I:H") && strings.Contains(scoreStr, "A:H") {
		// High impact across all categories
		if strings.Contains(scoreStr, "PR:N") && strings.Contains(scoreStr, "UI:N") {
			return 9.8 // Critical - easily exploitable
		}
		return 8.1 // High
	}

	if strings.Contains(scoreStr, "C:H") || strings.Contains(scoreStr, "I:H") || strings.Contains(scoreStr, "A:H") {
		// High impact in at least one category
		return 7.5
	}

	if strings.Contains(scoreStr, "C:L") && strings.Contains(scoreStr, "I:L") && strings.Contains(scoreStr, "A:L") {
		// Low impact
		return 3.7
	}

	// Default to medium
	return 5.0
}
