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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewReporter(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir)

	if reporter == nil {
		t.Fatal("NewReporter returned nil")
	}

	if reporter.outputDir != tmpDir {
		t.Errorf("Expected outputDir to be %s, got %s", tmpDir, reporter.outputDir)
	}
}

func TestReporter_GenerateReport_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir)

	results := []*ScanResult{
		{
			Tool:      "test",
			Timestamp: time.Now(),
			Findings:  []SecurityFinding{},
			Summary:   ScanSummary{},
		},
	}

	_, err := reporter.GenerateReport(results, "unsupported")
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

func TestReporter_GenerateJSONReport(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir)

	results := []*ScanResult{
		{
			Tool:      "gosec",
			Timestamp: time.Now(),
			Duration:  time.Second,
			Findings: []SecurityFinding{
				{
					ID:          "G101",
					Severity:    SeverityHigh,
					Category:    "Code Analysis",
					Title:       "Hardcoded credentials",
					Description: "Potential hardcoded credentials",
					File:        "test.go",
					Line:        10,
					Remediation: "Use environment variables",
				},
			},
			Summary: ScanSummary{
				TotalFindings: 1,
				HighCount:     1,
			},
		},
	}

	outputPath, err := reporter.GenerateReport(results, ReportFormatJSON)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Report file was not created: %s", outputPath)
	}

	// Verify file content is valid JSON
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	var parsedResults []*ScanResult
	if err := json.Unmarshal(data, &parsedResults); err != nil {
		t.Errorf("Report is not valid JSON: %v", err)
	}

	if len(parsedResults) != 1 {
		t.Errorf("Expected 1 result in report, got %d", len(parsedResults))
	}
}

func TestReporter_GenerateHTMLReport(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir)

	results := []*ScanResult{
		{
			Tool:      "gosec",
			Timestamp: time.Now(),
			Duration:  time.Second,
			Findings: []SecurityFinding{
				{
					ID:          "G101",
					Severity:    SeverityHigh,
					Category:    "Code Analysis",
					Title:       "Hardcoded credentials",
					Description: "Potential hardcoded credentials",
					File:        "test.go",
					Line:        10,
					Remediation: "Use environment variables",
				},
			},
			Summary: ScanSummary{
				TotalFindings: 1,
				HighCount:     1,
				FilesScanned:  5,
			},
		},
	}

	outputPath, err := reporter.GenerateReport(results, ReportFormatHTML)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Report file was not created: %s", outputPath)
	}

	// Verify file content is HTML
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Error("HTML report does not contain DOCTYPE")
	}

	if !strings.Contains(content, "Security Scan Report") {
		t.Error("HTML report does not contain title")
	}

	if !strings.Contains(content, "G101") {
		t.Error("HTML report does not contain finding ID")
	}
}

func TestReporter_GenerateTextReport(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir)

	results := []*ScanResult{
		{
			Tool:      "gosec",
			Timestamp: time.Now(),
			Duration:  time.Second,
			Findings: []SecurityFinding{
				{
					ID:          "G101",
					Severity:    SeverityHigh,
					Category:    "Code Analysis",
					Title:       "Hardcoded credentials",
					Description: "Potential hardcoded credentials",
					File:        "test.go",
					Line:        10,
					Remediation: "Use environment variables",
				},
			},
			Summary: ScanSummary{
				TotalFindings: 1,
				HighCount:     1,
				FilesScanned:  5,
			},
		},
	}

	outputPath, err := reporter.GenerateReport(results, ReportFormatText)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Report file was not created: %s", outputPath)
	}

	// Verify file content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "SECURITY SCAN REPORT") {
		t.Error("Text report does not contain title")
	}

	if !strings.Contains(content, "G101") {
		t.Error("Text report does not contain finding ID")
	}

	if !strings.Contains(content, "SUMMARY") {
		t.Error("Text report does not contain summary section")
	}
}

func TestReporter_GenerateSARIFReport(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir)

	results := []*ScanResult{
		{
			Tool:      "gosec",
			Timestamp: time.Now(),
			Duration:  time.Second,
			Findings: []SecurityFinding{
				{
					ID:          "G101",
					Severity:    SeverityHigh,
					Category:    "Code Analysis",
					Title:       "Hardcoded credentials",
					Description: "Potential hardcoded credentials",
					File:        "test.go",
					Line:        10,
					Column:      5,
					Remediation: "Use environment variables",
				},
			},
			Summary: ScanSummary{
				TotalFindings: 1,
				HighCount:     1,
			},
		},
	}

	outputPath, err := reporter.GenerateReport(results, ReportFormatSARIF)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Report file was not created: %s", outputPath)
	}

	// Verify file content is valid SARIF JSON
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	var sarif map[string]interface{}
	if err := json.Unmarshal(data, &sarif); err != nil {
		t.Errorf("Report is not valid JSON: %v", err)
	}

	// Verify SARIF structure
	if version, ok := sarif["version"].(string); !ok || version != "2.1.0" {
		t.Error("SARIF version is not 2.1.0")
	}

	if _, ok := sarif["$schema"].(string); !ok {
		t.Error("SARIF schema is missing")
	}

	if runs, ok := sarif["runs"].([]interface{}); !ok || len(runs) == 0 {
		t.Error("SARIF runs array is missing or empty")
	}
}

func TestReporter_MapSeverityToSARIFLevel(t *testing.T) {
	reporter := NewReporter("")

	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityCritical, "error"},
		{SeverityHigh, "error"},
		{SeverityMedium, "warning"},
		{SeverityLow, "note"},
		{SeverityInfo, "note"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			level := reporter.mapSeverityToSARIFLevel(tt.severity)
			if level != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, level)
			}
		})
	}
}

func TestReporter_GetToolURI(t *testing.T) {
	reporter := NewReporter("")

	tests := []struct {
		tool     string
		hasURI   bool
	}{
		{"gosec", true},
		{"govulncheck", true},
		{"trivy", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			uri := reporter.getToolURI(tt.tool)
			if tt.hasURI && uri == "" {
				t.Errorf("Expected URI for tool %s, got empty string", tt.tool)
			}
			if !tt.hasURI && uri != "" {
				t.Errorf("Expected empty URI for unknown tool, got %s", uri)
			}
		})
	}
}

func TestReporter_GenerateReport_WithMultipleFindings(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir)

	results := []*ScanResult{
		{
			Tool:      "gosec",
			Timestamp: time.Now(),
			Duration:  time.Second,
			Findings: []SecurityFinding{
				{
					ID:          "G101",
					Severity:    SeverityCritical,
					Category:    "Code Analysis",
					Title:       "Hardcoded credentials",
					Description: "Potential hardcoded credentials",
					File:        "test1.go",
					Line:        10,
				},
				{
					ID:          "G104",
					Severity:    SeverityHigh,
					Category:    "Code Analysis",
					Title:       "Unhandled error",
					Description: "Error return value not checked",
					File:        "test2.go",
					Line:        25,
				},
			},
			Summary: ScanSummary{
				TotalFindings: 2,
				CriticalCount: 1,
				HighCount:     1,
				FilesScanned:  10,
			},
		},
		{
			Tool:      "govulncheck",
			Timestamp: time.Now(),
			Duration:  2 * time.Second,
			Findings: []SecurityFinding{
				{
					ID:          "GO-2023-1234",
					Severity:    SeverityHigh,
					Category:    "Dependency Vulnerability",
					Title:       "Vulnerability in example.com/pkg",
					Description: "Known vulnerability",
					File:        "go.mod",
					CVSS:        7.5,
				},
			},
			Summary: ScanSummary{
				TotalFindings: 1,
				HighCount:     1,
			},
		},
	}

	// Test all formats
	formats := []ReportFormat{
		ReportFormatJSON,
		ReportFormatHTML,
		ReportFormatSARIF,
		ReportFormatText,
	}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			outputPath, err := reporter.GenerateReport(results, format)
			if err != nil {
				t.Fatalf("GenerateReport failed for format %s: %v", format, err)
			}

			// Verify file exists
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Errorf("Report file was not created: %s", outputPath)
			}

			// Verify file is not empty
			info, err := os.Stat(outputPath)
			if err != nil {
				t.Fatalf("Failed to stat file: %v", err)
			}

			if info.Size() == 0 {
				t.Error("Report file is empty")
			}

			// Verify file extension
			ext := filepath.Ext(outputPath)
			expectedExt := "." + string(format)
			if ext != expectedExt {
				t.Errorf("Expected file extension %s, got %s", expectedExt, ext)
			}
		})
	}
}

func TestReporter_GenerateReport_NoFindings(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir)

	results := []*ScanResult{
		{
			Tool:      "gosec",
			Timestamp: time.Now(),
			Duration:  time.Second,
			Findings:  []SecurityFinding{},
			Summary: ScanSummary{
				TotalFindings: 0,
				FilesScanned:  10,
			},
		},
	}

	outputPath, err := reporter.GenerateReport(results, ReportFormatHTML)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	// Verify file exists and contains success message
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "No security findings detected") && !strings.Contains(content, "✅") {
		t.Error("HTML report for no findings should contain success message")
	}
}

