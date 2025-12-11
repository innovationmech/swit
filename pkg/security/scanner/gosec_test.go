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

func TestNewGosecScanner(t *testing.T) {
	scanner := NewGosecScanner()
	if scanner == nil {
		t.Fatal("NewGosecScanner returned nil")
	}

	if scanner.Name() != "gosec" {
		t.Errorf("Name() = %s, want gosec", scanner.Name())
	}

	if len(scanner.excludeRules) != 0 {
		t.Errorf("excludeRules should be empty, got %d", len(scanner.excludeRules))
	}

	if len(scanner.includeRules) != 0 {
		t.Errorf("includeRules should be empty, got %d", len(scanner.includeRules))
	}
}

func TestGosecScanner_SetExcludeRules(t *testing.T) {
	scanner := NewGosecScanner()
	rules := []string{"G101", "G102", "G103"}
	scanner.SetExcludeRules(rules)

	if len(scanner.excludeRules) != len(rules) {
		t.Errorf("excludeRules length = %d, want %d", len(scanner.excludeRules), len(rules))
	}

	for i, rule := range scanner.excludeRules {
		if rule != rules[i] {
			t.Errorf("excludeRules[%d] = %s, want %s", i, rule, rules[i])
		}
	}
}

func TestGosecScanner_SetIncludeRules(t *testing.T) {
	scanner := NewGosecScanner()
	rules := []string{"G201", "G202"}
	scanner.SetIncludeRules(rules)

	if len(scanner.includeRules) != len(rules) {
		t.Errorf("includeRules length = %d, want %d", len(scanner.includeRules), len(rules))
	}

	for i, rule := range scanner.includeRules {
		if rule != rules[i] {
			t.Errorf("includeRules[%d] = %s, want %s", i, rule, rules[i])
		}
	}
}

func TestGosecScanner_mapGosecSeverity(t *testing.T) {
	scanner := NewGosecScanner()

	tests := []struct {
		input    string
		expected Severity
	}{
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
			result := scanner.mapGosecSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("mapGosecSeverity(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGosecScanner_getRemediationAdvice(t *testing.T) {
	scanner := NewGosecScanner()

	tests := []struct {
		ruleID   string
		contains string
	}{
		{"G101", "hardcoding credentials"},
		{"G102", "unsafe network bindings"},
		{"G103", "unsafe packages"},
		{"G104", "check for errors"},
		{"G106", "crypto/ssh"},
		{"G107", "URL.Parse"},
		{"G108", "profiling endpoints"},
		{"G109", "integer overflow"},
		{"G110", "DoS"},
		{"G201", "SQL injection"},
		{"G202", "SQL injection"},
		{"G203", "unescaped HTML"},
		{"G204", "command injection"},
		{"G301", "permissive permissions"},
		{"G302", "permissive permissions"},
		{"G303", "temporary files"},
		{"G304", "file path traversal"},
		{"G305", "zip slip"},
		{"G306", "insecure permissions"},
		{"G401", "weak cryptographic"},
		{"G402", "TLS/SSL"},
		{"G403", "encryption algorithms"},
		{"G404", "crypto/rand"},
		{"G501", "crypto packages"},
		{"G502", "crypto packages"},
		{"G503", "crypto packages"},
		{"G504", "net/http/cgi"},
		{"G505", "crypto packages"},
		{"G601", "memory aliasing"},
		{"UNKNOWN", "best practices"},
	}

	for _, tt := range tests {
		t.Run(tt.ruleID, func(t *testing.T) {
			advice := scanner.getRemediationAdvice(tt.ruleID)
			if advice == "" {
				t.Errorf("getRemediationAdvice(%s) returned empty string", tt.ruleID)
			}
		})
	}
}

func TestGosecScanner_parseGosecOutput(t *testing.T) {
	scanner := NewGosecScanner()
	startTime := time.Now()

	tests := []struct {
		name     string
		output   []byte
		wantErr  bool
		validate func(*testing.T, *ScanResult)
	}{
		{
			name: "valid output with no issues",
			output: []byte(`{
				"Issues": [],
				"Stats": {
					"files": 10,
					"lines": 500
				}
			}`),
			wantErr: false,
			validate: func(t *testing.T, result *ScanResult) {
				if result.Tool != "gosec" {
					t.Errorf("Tool = %s, want gosec", result.Tool)
				}
				if len(result.Findings) != 0 {
					t.Errorf("Findings count = %d, want 0", len(result.Findings))
				}
				if result.Summary.FilesScanned != 10 {
					t.Errorf("FilesScanned = %d, want 10", result.Summary.FilesScanned)
				}
			},
		},
		{
			name: "valid output with issues",
			output: []byte(`{
				"Issues": [
					{
						"severity": "HIGH",
						"confidence": "HIGH",
						"cwe": {"id": "CWE-798", "url": "https://cwe.mitre.org/data/definitions/798.html"},
						"rule_id": "G101",
						"details": "Potential hardcoded credentials",
						"file": "/path/to/file.go",
						"code": "password := \"secret\"",
						"line": "10",
						"column": "5"
					},
					{
						"severity": "MEDIUM",
						"confidence": "MEDIUM",
						"cwe": {"id": "CWE-89", "url": "https://cwe.mitre.org/data/definitions/89.html"},
						"rule_id": "G201",
						"details": "SQL query construction using string concatenation",
						"file": "/path/to/db.go",
						"code": "query := \"SELECT * FROM users WHERE id = \" + id",
						"line": "25",
						"column": "10"
					}
				],
				"Stats": {
					"files": 5,
					"lines": 200
				}
			}`),
			wantErr: false,
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 2 {
					t.Errorf("Findings count = %d, want 2", len(result.Findings))
				}
				if result.Summary.HighCount != 1 {
					t.Errorf("HighCount = %d, want 1", result.Summary.HighCount)
				}
				if result.Summary.MediumCount != 1 {
					t.Errorf("MediumCount = %d, want 1", result.Summary.MediumCount)
				}
				if result.Summary.TotalFindings != 2 {
					t.Errorf("TotalFindings = %d, want 2", result.Summary.TotalFindings)
				}
			},
		},
		{
			name: "valid output with low severity",
			output: []byte(`{
				"Issues": [
					{
						"severity": "LOW",
						"confidence": "LOW",
						"cwe": {"id": "CWE-000", "url": ""},
						"rule_id": "G601",
						"details": "Implicit memory aliasing",
						"file": "/path/to/file.go",
						"code": "for _, v := range items { go func() { fmt.Println(v) }() }",
						"line": "15",
						"column": "1"
					}
				],
				"Stats": {
					"files": 1,
					"lines": 50
				}
			}`),
			wantErr: false,
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 1 {
					t.Errorf("Findings count = %d, want 1", len(result.Findings))
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
			name:    "empty output",
			output:  []byte(`{}`),
			wantErr: false,
			validate: func(t *testing.T, result *ScanResult) {
				if len(result.Findings) != 0 {
					t.Errorf("Findings count = %d, want 0", len(result.Findings))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := scanner.parseGosecOutput(tt.output, startTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGosecOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestGosecScanner_parseGosecOutputSeverityCounts(t *testing.T) {
	scanner := NewGosecScanner()
	startTime := time.Now()

	output := []byte(`{
		"Issues": [
			{"severity": "HIGH", "confidence": "HIGH", "cwe": {"id": "", "url": ""}, "rule_id": "G101", "details": "", "file": "", "code": "", "line": "1", "column": "1"},
			{"severity": "HIGH", "confidence": "HIGH", "cwe": {"id": "", "url": ""}, "rule_id": "G102", "details": "", "file": "", "code": "", "line": "2", "column": "1"},
			{"severity": "MEDIUM", "confidence": "MEDIUM", "cwe": {"id": "", "url": ""}, "rule_id": "G201", "details": "", "file": "", "code": "", "line": "3", "column": "1"},
			{"severity": "MEDIUM", "confidence": "MEDIUM", "cwe": {"id": "", "url": ""}, "rule_id": "G202", "details": "", "file": "", "code": "", "line": "4", "column": "1"},
			{"severity": "MEDIUM", "confidence": "MEDIUM", "cwe": {"id": "", "url": ""}, "rule_id": "G203", "details": "", "file": "", "code": "", "line": "5", "column": "1"},
			{"severity": "LOW", "confidence": "LOW", "cwe": {"id": "", "url": ""}, "rule_id": "G601", "details": "", "file": "", "code": "", "line": "6", "column": "1"},
			{"severity": "INFO", "confidence": "LOW", "cwe": {"id": "", "url": ""}, "rule_id": "G000", "details": "", "file": "", "code": "", "line": "7", "column": "1"}
		],
		"Stats": {"files": 1, "lines": 100}
	}`)

	result, err := scanner.parseGosecOutput(output, startTime)
	if err != nil {
		t.Fatalf("parseGosecOutput() error = %v", err)
	}

	if result.Summary.HighCount != 2 {
		t.Errorf("HighCount = %d, want 2", result.Summary.HighCount)
	}
	if result.Summary.MediumCount != 3 {
		t.Errorf("MediumCount = %d, want 3", result.Summary.MediumCount)
	}
	if result.Summary.LowCount != 1 {
		t.Errorf("LowCount = %d, want 1", result.Summary.LowCount)
	}
	if result.Summary.InfoCount != 1 {
		t.Errorf("InfoCount = %d, want 1", result.Summary.InfoCount)
	}
	if result.Summary.TotalFindings != 7 {
		t.Errorf("TotalFindings = %d, want 7", result.Summary.TotalFindings)
	}
}


