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
	"testing"
	"time"
)

func TestNewTrivyScanner(t *testing.T) {
	scanner := NewTrivyScanner()
	if scanner == nil {
		t.Fatal("NewTrivyScanner returned nil")
	}

	if scanner.Name() != "trivy" {
		t.Errorf("Name() = %s, want trivy", scanner.Name())
	}

	if !scanner.scanFS {
		t.Error("scanFS should be true by default")
	}
}

func TestTrivyScanner_SetScanFS(t *testing.T) {
	scanner := NewTrivyScanner()

	scanner.SetScanFS(false)
	if scanner.scanFS {
		t.Error("scanFS should be false after SetScanFS(false)")
	}

	scanner.SetScanFS(true)
	if !scanner.scanFS {
		t.Error("scanFS should be true after SetScanFS(true)")
	}
}

func TestTrivyScanner_mapTrivySeverity(t *testing.T) {
	scanner := NewTrivyScanner()

	tests := []struct {
		input    string
		expected Severity
	}{
		{"CRITICAL", SeverityCritical},
		{"critical", SeverityCritical},
		{"Critical", SeverityCritical},
		{"HIGH", SeverityHigh},
		{"high", SeverityHigh},
		{"High", SeverityHigh},
		{"MEDIUM", SeverityMedium},
		{"medium", SeverityMedium},
		{"Medium", SeverityMedium},
		{"LOW", SeverityLow},
		{"low", SeverityLow},
		{"Low", SeverityLow},
		{"UNKNOWN", SeverityInfo},
		{"", SeverityInfo},
		{"invalid", SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := scanner.mapTrivySeverity(tt.input)
			if result != tt.expected {
				t.Errorf("mapTrivySeverity(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTrivyScanner_updateSummary(t *testing.T) {
	scanner := NewTrivyScanner()

	tests := []struct {
		severity         Severity
		expectedCritical int
		expectedHigh     int
		expectedMedium   int
		expectedLow      int
		expectedInfo     int
	}{
		{SeverityCritical, 1, 0, 0, 0, 0},
		{SeverityHigh, 0, 1, 0, 0, 0},
		{SeverityMedium, 0, 0, 1, 0, 0},
		{SeverityLow, 0, 0, 0, 1, 0},
		{SeverityInfo, 0, 0, 0, 0, 1},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			summary := &ScanSummary{}
			scanner.updateSummary(summary, tt.severity)

			if summary.TotalFindings != 1 {
				t.Errorf("TotalFindings = %d, want 1", summary.TotalFindings)
			}
			if summary.CriticalCount != tt.expectedCritical {
				t.Errorf("CriticalCount = %d, want %d", summary.CriticalCount, tt.expectedCritical)
			}
			if summary.HighCount != tt.expectedHigh {
				t.Errorf("HighCount = %d, want %d", summary.HighCount, tt.expectedHigh)
			}
			if summary.MediumCount != tt.expectedMedium {
				t.Errorf("MediumCount = %d, want %d", summary.MediumCount, tt.expectedMedium)
			}
			if summary.LowCount != tt.expectedLow {
				t.Errorf("LowCount = %d, want %d", summary.LowCount, tt.expectedLow)
			}
			if summary.InfoCount != tt.expectedInfo {
				t.Errorf("InfoCount = %d, want %d", summary.InfoCount, tt.expectedInfo)
			}
		})
	}
}

func TestTrivyScanner_parseTrivyOutput(t *testing.T) {
	scanner := NewTrivyScanner()
	startTime := time.Now()

	tests := []struct {
		name     string
		output   []byte
		wantErr  bool
		validate func(*testing.T, *ScanResult)
	}{
		{
			name: "valid output with no findings",
			output: []byte(`{
				"SchemaVersion": 2,
				"ArtifactName": "test-project",
				"ArtifactType": "filesystem",
				"Results": []
			}`),
			wantErr: false,
			validate: func(t *testing.T, result *ScanResult) {
				if result.Tool != "trivy" {
					t.Errorf("Tool = %s, want trivy", result.Tool)
				}
				if len(result.Findings) != 0 {
					t.Errorf("Findings count = %d, want 0", len(result.Findings))
				}
			},
		},
		{
			name: "valid output with vulnerabilities",
			output: []byte(`{
				"SchemaVersion": 2,
				"ArtifactName": "test-project",
				"ArtifactType": "filesystem",
				"Results": [
					{
						"Target": "go.mod",
						"Class": "lang-pkgs",
						"Type": "gomod",
						"Vulnerabilities": [
							{
								"VulnerabilityID": "CVE-2023-1234",
								"PkgName": "github.com/example/pkg",
								"InstalledVersion": "1.0.0",
								"FixedVersion": "1.0.1",
								"Title": "Example vulnerability",
								"Description": "A vulnerability in example package",
								"Severity": "HIGH",
								"References": ["https://nvd.nist.gov/vuln/detail/CVE-2023-1234"],
								"CVSS": {"nvd": {"V3Score": 7.5}}
							}
						]
					}
				]
			}`),
			wantErr: false,
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 1 {
					t.Errorf("Findings count = %d, want 1", len(result.Findings))
				}
				if result.Summary.HighCount != 1 {
					t.Errorf("HighCount = %d, want 1", result.Summary.HighCount)
				}
				finding := result.Findings[0]
				if finding.ID != "CVE-2023-1234" {
					t.Errorf("Finding ID = %s, want CVE-2023-1234", finding.ID)
				}
				if finding.Category != "Dependency Vulnerability" {
					t.Errorf("Finding Category = %s, want Dependency Vulnerability", finding.Category)
				}
			},
		},
		{
			name: "valid output with misconfigurations",
			output: []byte(`{
				"SchemaVersion": 2,
				"ArtifactName": "test-project",
				"ArtifactType": "filesystem",
				"Results": [
					{
						"Target": "Dockerfile",
						"Class": "config",
						"Type": "dockerfile",
						"Misconfigurations": [
							{
								"Type": "Dockerfile Security Check",
								"ID": "DS002",
								"Title": "Image user should not be 'root'",
								"Description": "Running as root user",
								"Message": "Specify a user in the Dockerfile",
								"Resolution": "Add USER instruction",
								"Severity": "MEDIUM",
								"PrimaryURL": "https://example.com/ds002",
								"References": ["https://docs.docker.com/develop/develop-images/dockerfile_best-practices/"]
							}
						]
					}
				]
			}`),
			wantErr: false,
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 1 {
					t.Errorf("Findings count = %d, want 1", len(result.Findings))
				}
				if result.Summary.MediumCount != 1 {
					t.Errorf("MediumCount = %d, want 1", result.Summary.MediumCount)
				}
				finding := result.Findings[0]
				if finding.Category != "Misconfiguration" {
					t.Errorf("Finding Category = %s, want Misconfiguration", finding.Category)
				}
			},
		},
		{
			name: "valid output with secrets",
			output: []byte(`{
				"SchemaVersion": 2,
				"ArtifactName": "test-project",
				"ArtifactType": "filesystem",
				"Results": [
					{
						"Target": "config.yaml",
						"Class": "secret",
						"Type": "secret",
						"Secrets": [
							{
								"RuleID": "aws-access-key-id",
								"Category": "AWS",
								"Title": "AWS Access Key ID",
								"Severity": "CRITICAL",
								"Match": "AKIAIOSFODNN7EXAMPLE"
							}
						]
					}
				]
			}`),
			wantErr: false,
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 1 {
					t.Errorf("Findings count = %d, want 1", len(result.Findings))
				}
				if result.Summary.CriticalCount != 1 {
					t.Errorf("CriticalCount = %d, want 1", result.Summary.CriticalCount)
				}
				finding := result.Findings[0]
				if finding.Category != "Secret Exposure" {
					t.Errorf("Finding Category = %s, want Secret Exposure", finding.Category)
				}
			},
		},
		{
			name: "valid output with mixed findings",
			output: []byte(`{
				"SchemaVersion": 2,
				"ArtifactName": "test-project",
				"ArtifactType": "filesystem",
				"Results": [
					{
						"Target": "go.mod",
						"Class": "lang-pkgs",
						"Type": "gomod",
						"Vulnerabilities": [
							{
								"VulnerabilityID": "CVE-2023-0001",
								"PkgName": "pkg1",
								"InstalledVersion": "1.0.0",
								"FixedVersion": "",
								"Title": "No fix available",
								"Description": "A vulnerability without fix",
								"Severity": "LOW",
								"References": [],
								"CVSS": {}
							}
						],
						"Misconfigurations": [
							{
								"Type": "Config",
								"ID": "CFG001",
								"Title": "Config issue",
								"Description": "A config issue",
								"Message": "Fix the config",
								"Resolution": "Update config",
								"Severity": "HIGH",
								"PrimaryURL": "",
								"References": []
							}
						],
						"Secrets": [
							{
								"RuleID": "generic-api-key",
								"Category": "Generic",
								"Title": "Generic API Key",
								"Severity": "MEDIUM",
								"Match": "api_key=secret123"
							}
						]
					}
				]
			}`),
			wantErr: false,
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 3 {
					t.Errorf("Findings count = %d, want 3", len(result.Findings))
				}
				if result.Summary.TotalFindings != 3 {
					t.Errorf("TotalFindings = %d, want 3", result.Summary.TotalFindings)
				}
				if result.Summary.HighCount != 1 {
					t.Errorf("HighCount = %d, want 1", result.Summary.HighCount)
				}
				if result.Summary.MediumCount != 1 {
					t.Errorf("MediumCount = %d, want 1", result.Summary.MediumCount)
				}
				if result.Summary.LowCount != 1 {
					t.Errorf("LowCount = %d, want 1", result.Summary.LowCount)
				}
			},
		},
		{
			name:    "invalid JSON",
			output:  []byte(`{invalid json`),
			wantErr: true,
		},
		{
			name:   "empty output",
			output: []byte(`{}`),
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 0 {
					t.Errorf("Findings count = %d, want 0", len(result.Findings))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := scanner.parseTrivyOutput(tt.output, startTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTrivyOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestTrivyScanner_parseTrivyOutputVulnerabilityRemediation(t *testing.T) {
	scanner := NewTrivyScanner()
	startTime := time.Now()

	// Test with fixed version available
	outputWithFix := []byte(`{
		"SchemaVersion": 2,
		"Results": [
			{
				"Target": "go.mod",
				"Vulnerabilities": [
					{
						"VulnerabilityID": "CVE-2023-0001",
						"PkgName": "example-pkg",
						"InstalledVersion": "1.0.0",
						"FixedVersion": "1.0.1",
						"Title": "Test",
						"Description": "Test",
						"Severity": "HIGH"
					}
				]
			}
		]
	}`)

	result, err := scanner.parseTrivyOutput(outputWithFix, startTime)
	if err != nil {
		t.Fatalf("parseTrivyOutput() error = %v", err)
	}

	if len(result.Findings) != 1 {
		t.Fatalf("Findings count = %d, want 1", len(result.Findings))
	}

	if result.Findings[0].Remediation == "" {
		t.Error("Remediation should not be empty when fix is available")
	}

	// Test without fixed version
	outputWithoutFix := []byte(`{
		"SchemaVersion": 2,
		"Results": [
			{
				"Target": "go.mod",
				"Vulnerabilities": [
					{
						"VulnerabilityID": "CVE-2023-0002",
						"PkgName": "example-pkg2",
						"InstalledVersion": "2.0.0",
						"FixedVersion": "",
						"Title": "Test2",
						"Description": "Test2",
						"Severity": "MEDIUM"
					}
				]
			}
		]
	}`)

	result2, err := scanner.parseTrivyOutput(outputWithoutFix, startTime)
	if err != nil {
		t.Fatalf("parseTrivyOutput() error = %v", err)
	}

	if len(result2.Findings) != 1 {
		t.Fatalf("Findings count = %d, want 1", len(result2.Findings))
	}

	if result2.Findings[0].Remediation == "" {
		t.Error("Remediation should not be empty even when no fix is available")
	}
}

