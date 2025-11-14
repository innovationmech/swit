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

// TrivyScanner implements the ScanTool interface for Trivy vulnerability scanner.
type TrivyScanner struct {
	scanFS bool // If true, scan filesystem; if false, scan container images
}

// NewTrivyScanner creates a new TrivyScanner instance.
func NewTrivyScanner() *TrivyScanner {
	return &TrivyScanner{
		scanFS: true, // Default to filesystem scanning
	}
}

// Name returns the name of the scanning tool.
func (t *TrivyScanner) Name() string {
	return "trivy"
}

// IsAvailable checks if trivy is installed and available.
func (t *TrivyScanner) IsAvailable() bool {
	_, err := exec.LookPath("trivy")
	return err == nil
}

// SetScanFS sets whether to scan filesystem (true) or container images (false).
func (t *TrivyScanner) SetScanFS(scanFS bool) {
	t.scanFS = scanFS
}

// Scan performs a vulnerability scan using Trivy.
func (t *TrivyScanner) Scan(ctx context.Context, target string) (*ScanResult, error) {
	startTime := time.Now()

	// Build trivy command
	var args []string
	if t.scanFS {
		args = []string{"fs", "--format", "json", "--scanners", "vuln,misconfig,secret"}
	} else {
		args = []string{"image", "--format", "json"}
	}

	// Add target
	args = append(args, target)

	// Execute trivy
	cmd := exec.CommandContext(ctx, "trivy", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, fmt.Errorf("failed to execute trivy: %w", err)
		}
		// trivy returns non-zero exit code when vulnerabilities are found
	}

	// Parse trivy output
	result, err := t.parseTrivyOutput(output, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trivy output: %w", err)
	}

	return result, nil
}

// trivyOutput represents the JSON output structure from Trivy.
type trivyOutput struct {
	SchemaVersion int    `json:"SchemaVersion"`
	ArtifactName  string `json:"ArtifactName"`
	ArtifactType  string `json:"ArtifactType"`
	Results       []struct {
		Target          string `json:"Target"`
		Class           string `json:"Class"`
		Type            string `json:"Type"`
		Vulnerabilities []struct {
			VulnerabilityID  string   `json:"VulnerabilityID"`
			PkgName          string   `json:"PkgName"`
			InstalledVersion string   `json:"InstalledVersion"`
			FixedVersion     string   `json:"FixedVersion"`
			Title            string   `json:"Title"`
			Description      string   `json:"Description"`
			Severity         string   `json:"Severity"`
			References       []string `json:"References"`
			CVSS             struct {
				Nvd struct {
					V3Score float64 `json:"V3Score"`
				} `json:"nvd"`
			} `json:"CVSS"`
		} `json:"Vulnerabilities"`
		Misconfigurations []struct {
			Type        string   `json:"Type"`
			ID          string   `json:"ID"`
			Title       string   `json:"Title"`
			Description string   `json:"Description"`
			Message     string   `json:"Message"`
			Resolution  string   `json:"Resolution"`
			Severity    string   `json:"Severity"`
			PrimaryURL  string   `json:"PrimaryURL"`
			References  []string `json:"References"`
		} `json:"Misconfigurations"`
		Secrets []struct {
			RuleID   string `json:"RuleID"`
			Category string `json:"Category"`
			Title    string `json:"Title"`
			Severity string `json:"Severity"`
			Match    string `json:"Match"`
		} `json:"Secrets"`
	} `json:"Results"`
}

// parseTrivyOutput parses the JSON output from Trivy and converts it to ScanResult.
func (t *TrivyScanner) parseTrivyOutput(output []byte, startTime time.Time) (*ScanResult, error) {
	var trivyResult trivyOutput

	if err := json.Unmarshal(output, &trivyResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trivy output: %w", err)
	}

	result := &ScanResult{
		Tool:      t.Name(),
		Timestamp: startTime,
		Duration:  time.Since(startTime),
		Findings:  make([]SecurityFinding, 0),
		Summary:   ScanSummary{},
	}

	// Process all results
	for _, res := range trivyResult.Results {
		// Process vulnerabilities
		for _, vuln := range res.Vulnerabilities {
			severity := t.mapTrivySeverity(vuln.Severity)

			remediation := fmt.Sprintf("Update %s from version %s to %s or later",
				vuln.PkgName, vuln.InstalledVersion, vuln.FixedVersion)
			if vuln.FixedVersion == "" {
				remediation = fmt.Sprintf("No fix available yet for %s (current version: %s)",
					vuln.PkgName, vuln.InstalledVersion)
			}

			finding := SecurityFinding{
				ID:          vuln.VulnerabilityID,
				Severity:    severity,
				Category:    "Dependency Vulnerability",
				Title:       vuln.Title,
				Description: fmt.Sprintf("%s\n\nPackage: %s@%s\nTarget: %s", vuln.Description, vuln.PkgName, vuln.InstalledVersion, res.Target),
				File:        res.Target,
				Remediation: remediation,
				CVSS:        vuln.CVSS.Nvd.V3Score,
				References:  vuln.References,
			}

			result.Findings = append(result.Findings, finding)
			t.updateSummary(&result.Summary, severity)
		}

		// Process misconfigurations
		for _, misconfig := range res.Misconfigurations {
			severity := t.mapTrivySeverity(misconfig.Severity)

			finding := SecurityFinding{
				ID:          misconfig.ID,
				Severity:    severity,
				Category:    "Misconfiguration",
				Title:       misconfig.Title,
				Description: fmt.Sprintf("%s\n\n%s\nTarget: %s", misconfig.Description, misconfig.Message, res.Target),
				File:        res.Target,
				Remediation: misconfig.Resolution,
				References:  append([]string{misconfig.PrimaryURL}, misconfig.References...),
			}

			result.Findings = append(result.Findings, finding)
			t.updateSummary(&result.Summary, severity)
		}

		// Process secrets
		for _, secret := range res.Secrets {
			severity := t.mapTrivySeverity(secret.Severity)

			finding := SecurityFinding{
				ID:          secret.RuleID,
				Severity:    severity,
				Category:    "Secret Exposure",
				Title:       secret.Title,
				Description: fmt.Sprintf("Potential secret found: %s\nMatch: %s\nTarget: %s", secret.Category, secret.Match, res.Target),
				File:        res.Target,
				Remediation: "Remove hardcoded secrets and use environment variables or secret management systems.",
			}

			result.Findings = append(result.Findings, finding)
			t.updateSummary(&result.Summary, severity)
		}
	}

	return result, nil
}

// mapTrivySeverity maps Trivy severity levels to our standard severity levels.
func (t *TrivyScanner) mapTrivySeverity(trivySev string) Severity {
	switch strings.ToUpper(trivySev) {
	case "CRITICAL":
		return SeverityCritical
	case "HIGH":
		return SeverityHigh
	case "MEDIUM":
		return SeverityMedium
	case "LOW":
		return SeverityLow
	default:
		return SeverityInfo
	}
}

// updateSummary updates the summary counts based on severity.
func (t *TrivyScanner) updateSummary(summary *ScanSummary, severity Severity) {
	summary.TotalFindings++
	switch severity {
	case SeverityCritical:
		summary.CriticalCount++
	case SeverityHigh:
		summary.HighCount++
	case SeverityMedium:
		summary.MediumCount++
	case SeverityLow:
		summary.LowCount++
	case SeverityInfo:
		summary.InfoCount++
	}
}
