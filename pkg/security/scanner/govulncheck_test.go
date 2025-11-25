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

func TestNewGovulncheckScanner(t *testing.T) {
	scanner := NewGovulncheckScanner()
	if scanner == nil {
		t.Fatal("NewGovulncheckScanner returned nil")
	}

	if scanner.Name() != "govulncheck" {
		t.Errorf("Name() = %s, want govulncheck", scanner.Name())
	}
}

func TestGovulncheckScanner_parseCVSSScore(t *testing.T) {
	scanner := NewGovulncheckScanner()

	tests := []struct {
		name     string
		scoreStr string
		expected float64
	}{
		{
			name:     "critical - high impact all categories, easily exploitable",
			scoreStr: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			expected: 9.8,
		},
		{
			name:     "high - high impact all categories, requires privileges",
			scoreStr: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H",
			expected: 8.1,
		},
		{
			name:     "high - high confidentiality impact only",
			scoreStr: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N",
			expected: 7.5,
		},
		{
			name:     "high - high integrity impact only",
			scoreStr: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:H/A:N",
			expected: 7.5,
		},
		{
			name:     "high - high availability impact only",
			scoreStr: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H",
			expected: 7.5,
		},
		{
			name:     "low - low impact all categories",
			scoreStr: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:U/C:L/I:L/A:L",
			expected: 3.7,
		},
		{
			name:     "medium - no impact (default)",
			scoreStr: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:N",
			expected: 5.0,
		},
		{
			name:     "empty string",
			scoreStr: "",
			expected: 5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanner.parseCVSSScore(tt.scoreStr)
			if result != tt.expected {
				t.Errorf("parseCVSSScore(%s) = %f, want %f", tt.scoreStr, result, tt.expected)
			}
		})
	}
}

func TestGovulncheckScanner_mapCVSSSeverity(t *testing.T) {
	scanner := NewGovulncheckScanner()

	tests := []struct {
		name       string
		severities []struct {
			Type  string  `json:"type"`
			Score string  `json:"score"`
			Value float64 `json:"-"`
		}
		expected Severity
	}{
		{
			name: "critical severity",
			severities: []struct {
				Type  string  `json:"type"`
				Score string  `json:"score"`
				Value float64 `json:"-"`
			}{
				{Type: "CVSS_V3", Score: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"},
			},
			expected: SeverityCritical,
		},
		{
			name: "high severity",
			severities: []struct {
				Type  string  `json:"type"`
				Score string  `json:"score"`
				Value float64 `json:"-"`
			}{
				{Type: "CVSS_V3", Score: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H"},
			},
			expected: SeverityHigh,
		},
		{
			name: "medium severity",
			severities: []struct {
				Type  string  `json:"type"`
				Score string  `json:"score"`
				Value float64 `json:"-"`
			}{
				{Type: "CVSS_V3", Score: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:N"},
			},
			expected: SeverityMedium,
		},
		{
			name: "low severity - low impact all categories",
			severities: []struct {
				Type  string  `json:"type"`
				Score string  `json:"score"`
				Value float64 `json:"-"`
			}{
				{Type: "CVSS_V3", Score: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:U/C:L/I:L/A:L"},
			},
			expected: SeverityLow,
		},
		{
			name:       "info severity - empty",
			severities: nil,
			expected:   SeverityInfo,
		},
		{
			name: "non-CVSS_V3 type ignored",
			severities: []struct {
				Type  string  `json:"type"`
				Score string  `json:"score"`
				Value float64 `json:"-"`
			}{
				{Type: "CVSS_V2", Score: "AV:N/AC:L/Au:N/C:C/I:C/A:C"},
			},
			expected: SeverityInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanner.mapCVSSSeverity(tt.severities)
			if result != tt.expected {
				t.Errorf("mapCVSSSeverity() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGovulncheckScanner_parseGovulncheckOutput(t *testing.T) {
	scanner := NewGovulncheckScanner()
	startTime := time.Now()

	tests := []struct {
		name     string
		output   []byte
		validate func(*testing.T, *ScanResult)
	}{
		{
			name:   "empty output",
			output: []byte(""),
			validate: func(t *testing.T, result *ScanResult) {
				if result.Tool != "govulncheck" {
					t.Errorf("Tool = %s, want govulncheck", result.Tool)
				}
				if len(result.Findings) != 0 {
					t.Errorf("Findings count = %d, want 0", len(result.Findings))
				}
			},
		},
		{
			name: "config message only",
			output: []byte(`{"config":{"scanner_name":"govulncheck","scanner_version":"v1.0.0","db":"https://vuln.go.dev","db_last_modified":"2024-01-01T00:00:00Z"}}
`),
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 0 {
					t.Errorf("Findings count = %d, want 0", len(result.Findings))
				}
			},
		},
		{
			name: "vulnerability with fix available",
			output: []byte(`{"vulnerability":{"osv":{"id":"GO-2023-0001","summary":"Test vulnerability","details":"A test vulnerability","severity":[{"type":"CVSS_V3","score":"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"}],"database_specific":{"url":"https://pkg.go.dev/vuln/GO-2023-0001"}},"modules":[{"path":"github.com/test/pkg","found_version":"v1.0.0","fixed_version":"v1.0.1","packages":[{"path":"github.com/test/pkg","symbols":[{"name":"VulnerableFunc"}]}]}]}}
`),
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 1 {
					t.Errorf("Findings count = %d, want 1", len(result.Findings))
					return
				}
				finding := result.Findings[0]
				if finding.ID != "GO-2023-0001" {
					t.Errorf("Finding ID = %s, want GO-2023-0001", finding.ID)
				}
				if finding.Severity != SeverityCritical {
					t.Errorf("Finding Severity = %s, want %s", finding.Severity, SeverityCritical)
				}
				if result.Summary.CriticalCount != 1 {
					t.Errorf("CriticalCount = %d, want 1", result.Summary.CriticalCount)
				}
			},
		},
		{
			name: "vulnerability without fix",
			output: []byte(`{"vulnerability":{"osv":{"id":"GO-2023-0002","summary":"No fix vulnerability","details":"A vulnerability without fix","severity":[{"type":"CVSS_V3","score":"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:L"}],"database_specific":{"url":""}},"modules":[{"path":"github.com/test/pkg2","found_version":"v2.0.0","fixed_version":"","packages":[]}]}}
`),
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 1 {
					t.Errorf("Findings count = %d, want 1", len(result.Findings))
					return
				}
				finding := result.Findings[0]
				if finding.ID != "GO-2023-0002" {
					t.Errorf("Finding ID = %s, want GO-2023-0002", finding.ID)
				}
				if finding.Severity != SeverityHigh {
					t.Errorf("Finding Severity = %s, want %s", finding.Severity, SeverityHigh)
				}
			},
		},
		{
			name: "multiple vulnerabilities",
			output: []byte(`{"vulnerability":{"osv":{"id":"GO-2023-0003","summary":"Vuln 1","details":"Details 1","severity":[{"type":"CVSS_V3","score":"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"}],"database_specific":{"url":""}},"modules":[{"path":"pkg1","found_version":"v1.0.0","fixed_version":"v1.0.1","packages":[]}]}}
{"vulnerability":{"osv":{"id":"GO-2023-0004","summary":"Vuln 2","details":"Details 2","severity":[{"type":"CVSS_V3","score":"CVSS:3.1/AV:N/AC:H/PR:H/UI:R/S:U/C:L/I:L/A:L"}],"database_specific":{"url":""}},"modules":[{"path":"pkg2","found_version":"v2.0.0","fixed_version":"v2.0.1","packages":[]}]}}
`),
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 2 {
					t.Errorf("Findings count = %d, want 2", len(result.Findings))
				}
				if result.Summary.TotalFindings != 2 {
					t.Errorf("TotalFindings = %d, want 2", result.Summary.TotalFindings)
				}
			},
		},
		{
			name: "vulnerability affecting multiple modules",
			output: []byte(`{"vulnerability":{"osv":{"id":"GO-2023-0005","summary":"Multi-module vuln","details":"Details","severity":[{"type":"CVSS_V3","score":"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:N"}],"database_specific":{"url":""}},"modules":[{"path":"pkg1","found_version":"v1.0.0","fixed_version":"v1.0.1","packages":[]},{"path":"pkg2","found_version":"v1.0.0","fixed_version":"v1.0.1","packages":[]}]}}
`),
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 2 {
					t.Errorf("Findings count = %d, want 2 (one per module)", len(result.Findings))
				}
			},
		},
		{
			name: "invalid JSON lines ignored",
			output: []byte(`invalid json line
{"vulnerability":{"osv":{"id":"GO-2023-0006","summary":"Valid vuln","details":"Details","severity":[],"database_specific":{"url":""}},"modules":[{"path":"pkg","found_version":"v1.0.0","fixed_version":"v1.0.1","packages":[]}]}}
another invalid line
`),
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 1 {
					t.Errorf("Findings count = %d, want 1", len(result.Findings))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := scanner.parseGovulncheckOutput(tt.output, startTime)
			if err != nil {
				t.Errorf("parseGovulncheckOutput() error = %v", err)
				return
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestGovulncheckScanner_convertVulnerabilityToFindings(t *testing.T) {
	scanner := NewGovulncheckScanner()

	vuln := &struct {
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
	}{
		OSV: struct {
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
		}{
			ID:      "GO-2023-TEST",
			Summary: "Test vulnerability summary",
			Details: "Test vulnerability details",
			Severity: []struct {
				Type  string  `json:"type"`
				Score string  `json:"score"`
				Value float64 `json:"-"`
			}{
				{Type: "CVSS_V3", Score: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"},
			},
			DatabaseSpecific: struct {
				URL string `json:"url"`
			}{
				URL: "https://pkg.go.dev/vuln/GO-2023-TEST",
			},
		},
		Modules: []struct {
			Path         string `json:"path"`
			FoundVersion string `json:"found_version"`
			FixedVersion string `json:"fixed_version"`
			Packages     []struct {
				Path    string `json:"path"`
				Symbols []struct {
					Name string `json:"name"`
				} `json:"symbols"`
			} `json:"packages"`
		}{
			{
				Path:         "github.com/test/module",
				FoundVersion: "v1.0.0",
				FixedVersion: "v1.0.1",
			},
		},
	}

	findings := scanner.convertVulnerabilityToFindings(vuln)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}

	finding := findings[0]

	if finding.ID != "GO-2023-TEST" {
		t.Errorf("ID = %s, want GO-2023-TEST", finding.ID)
	}

	if finding.Severity != SeverityCritical {
		t.Errorf("Severity = %s, want %s", finding.Severity, SeverityCritical)
	}

	if finding.Category != "Dependency Vulnerability" {
		t.Errorf("Category = %s, want Dependency Vulnerability", finding.Category)
	}

	if finding.Title != "Test vulnerability summary" {
		t.Errorf("Title = %s, want Test vulnerability summary", finding.Title)
	}

	if finding.File != "go.mod" {
		t.Errorf("File = %s, want go.mod", finding.File)
	}

	if len(finding.References) < 2 {
		t.Errorf("Expected at least 2 references, got %d", len(finding.References))
	}
}

