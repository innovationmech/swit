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
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if len(config.Tools) == 0 {
		t.Error("Expected Tools to have default values")
	}

	if config.OutputDir == "" {
		t.Error("Expected OutputDir to have a default value")
	}

	if !config.FailOnHigh {
		t.Error("Expected FailOnHigh to be true by default")
	}

	if !config.FailOnCritical {
		t.Error("Expected FailOnCritical to be true by default")
	}
}

func TestNewSecurityScanner(t *testing.T) {
	tests := []struct {
		name   string
		config *ScannerConfig
	}{
		{
			name:   "with nil config",
			config: nil,
		},
		{
			name:   "with default config",
			config: DefaultConfig(),
		},
		{
			name: "with custom config",
			config: &ScannerConfig{
				Enabled:   true,
				Tools:     []string{"gosec"},
				OutputDir: "/tmp/security",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner, err := NewSecurityScanner(tt.config)
			if err != nil {
				t.Fatalf("NewSecurityScanner failed: %v", err)
			}

			if scanner == nil {
				t.Fatal("NewSecurityScanner returned nil scanner")
			}

			if scanner.config == nil {
				t.Error("Scanner config is nil")
			}

			if scanner.tools == nil {
				t.Error("Scanner tools map is nil")
			}
		})
	}
}

func TestSecurityScanner_getEnabledTools(t *testing.T) {
	config := &ScannerConfig{
		Enabled: true,
		Tools:   []string{"gosec", "govulncheck", "unknown-tool"},
	}

	scanner, err := NewSecurityScanner(config)
	if err != nil {
		t.Fatalf("NewSecurityScanner failed: %v", err)
	}

	tools := scanner.getEnabledTools()

	// In CI, tools might not be installed, so we just verify the function returns successfully
	// The slice might be empty if no tools are installed, which is acceptable

	// Verify that if tools are returned, they are valid registered tools
	for _, tool := range tools {
		name := tool.Name()
		if name != "gosec" && name != "govulncheck" && name != "trivy" {
			t.Errorf("Unexpected tool: %s", name)
		}
		// Verify the tool reports availability correctly
		if !tool.IsAvailable() {
			t.Errorf("Tool %s returned by getEnabledTools but reports as not available", name)
		}
	}

	// If we have tools available, verify the function filters correctly
	// by checking that unknown-tool is not included
	for _, tool := range tools {
		if tool.Name() == "unknown-tool" {
			t.Error("getEnabledTools should not return unknown tools")
		}
	}
}

func TestSecurityScanner_ShouldFail(t *testing.T) {
	tests := []struct {
		name     string
		config   *ScannerConfig
		results  []*ScanResult
		expected bool
	}{
		{
			name: "no findings",
			config: &ScannerConfig{
				FailOnCritical: true,
				FailOnHigh:     true,
				FailOnMedium:   false,
			},
			results: []*ScanResult{
				{
					Summary: ScanSummary{
						TotalFindings: 0,
					},
				},
			},
			expected: false,
		},
		{
			name: "critical findings with fail on critical",
			config: &ScannerConfig{
				FailOnCritical: true,
				FailOnHigh:     false,
				FailOnMedium:   false,
			},
			results: []*ScanResult{
				{
					Summary: ScanSummary{
						CriticalCount: 1,
					},
				},
			},
			expected: true,
		},
		{
			name: "high findings with fail on high",
			config: &ScannerConfig{
				FailOnCritical: false,
				FailOnHigh:     true,
				FailOnMedium:   false,
			},
			results: []*ScanResult{
				{
					Summary: ScanSummary{
						HighCount: 2,
					},
				},
			},
			expected: true,
		},
		{
			name: "medium findings with fail on medium",
			config: &ScannerConfig{
				FailOnCritical: false,
				FailOnHigh:     false,
				FailOnMedium:   true,
			},
			results: []*ScanResult{
				{
					Summary: ScanSummary{
						MediumCount: 3,
					},
				},
			},
			expected: true,
		},
		{
			name: "medium findings without fail on medium",
			config: &ScannerConfig{
				FailOnCritical: true,
				FailOnHigh:     true,
				FailOnMedium:   false,
			},
			results: []*ScanResult{
				{
					Summary: ScanSummary{
						MediumCount: 5,
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := &SecurityScanner{
				config: tt.config,
			}

			result := scanner.ShouldFail(tt.results)
			if result != tt.expected {
				t.Errorf("ShouldFail() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetTotalFindings(t *testing.T) {
	results := []*ScanResult{
		{
			Summary: ScanSummary{
				TotalFindings: 5,
			},
		},
		{
			Summary: ScanSummary{
				TotalFindings: 3,
			},
		},
		{
			Summary: ScanSummary{
				TotalFindings: 7,
			},
		},
	}

	total := GetTotalFindings(results)
	expected := 15

	if total != expected {
		t.Errorf("GetTotalFindings() = %d, expected %d", total, expected)
	}
}

func TestGetFindingsBySeverity(t *testing.T) {
	results := []*ScanResult{
		{
			Summary: ScanSummary{
				CriticalCount: 2,
				HighCount:     3,
				MediumCount:   5,
				LowCount:      7,
				InfoCount:     11,
			},
		},
		{
			Summary: ScanSummary{
				CriticalCount: 1,
				HighCount:     2,
				MediumCount:   3,
				LowCount:      4,
				InfoCount:     5,
			},
		},
	}

	counts := GetFindingsBySeverity(results)

	expected := map[Severity]int{
		SeverityCritical: 3,
		SeverityHigh:     5,
		SeverityMedium:   8,
		SeverityLow:      11,
		SeverityInfo:     16,
	}

	for severity, expectedCount := range expected {
		if counts[severity] != expectedCount {
			t.Errorf("Severity %s: got %d, expected %d", severity, counts[severity], expectedCount)
		}
	}
}

func TestSecurityScanner_Scan_Disabled(t *testing.T) {
	config := &ScannerConfig{
		Enabled: false,
	}

	scanner, err := NewSecurityScanner(config)
	if err != nil {
		t.Fatalf("NewSecurityScanner failed: %v", err)
	}

	ctx := context.Background()
	results, err := scanner.Scan(ctx, "./...")

	if err == nil {
		t.Error("Expected error when scanning is disabled")
	}

	if results != nil {
		t.Error("Expected nil results when scanning is disabled")
	}
}

func TestSecurityScanner_Scan_NoTools(t *testing.T) {
	config := &ScannerConfig{
		Enabled: true,
		Tools:   []string{}, // No tools
	}

	scanner, err := NewSecurityScanner(config)
	if err != nil {
		t.Fatalf("NewSecurityScanner failed: %v", err)
	}

	ctx := context.Background()
	results, err := scanner.Scan(ctx, "./...")

	if err == nil {
		t.Error("Expected error when no tools are available")
	}

	if results != nil {
		t.Error("Expected nil results when no tools are available")
	}
}

func TestSecurityScanner_Scan_ContextCancellation(t *testing.T) {
	config := &ScannerConfig{
		Enabled: true,
		Tools:   []string{"gosec"},
		Timeout: 1 * time.Second,
	}

	scanner, err := NewSecurityScanner(config)
	if err != nil {
		t.Fatalf("NewSecurityScanner failed: %v", err)
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// This should fail quickly due to cancelled context
	_, err = scanner.Scan(ctx, "./...")

	// We expect an error (either context cancelled or no tools available)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

// Mock tool for testing
type mockTool struct {
	name      string
	available bool
	result    *ScanResult
	err       error
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) IsAvailable() bool {
	return m.available
}

func (m *mockTool) Scan(ctx context.Context, target string) (*ScanResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestSecurityScanner_scanSequential(t *testing.T) {
	config := DefaultConfig()
	scanner := &SecurityScanner{
		config: config,
		tools:  make(map[string]ScanTool),
	}

	mockResult := &ScanResult{
		Tool:      "mock",
		Timestamp: time.Now(),
		Findings:  []SecurityFinding{},
		Summary:   ScanSummary{TotalFindings: 0},
	}

	tools := []ScanTool{
		&mockTool{
			name:      "mock1",
			available: true,
			result:    mockResult,
		},
		&mockTool{
			name:      "mock2",
			available: true,
			result:    mockResult,
		},
	}

	ctx := context.Background()
	results, errors := scanner.scanSequential(ctx, "./...", tools)

	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d", len(errors))
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestSecurityScanner_scanParallel(t *testing.T) {
	config := DefaultConfig()
	scanner := &SecurityScanner{
		config: config,
		tools:  make(map[string]ScanTool),
	}

	mockResult := &ScanResult{
		Tool:      "mock",
		Timestamp: time.Now(),
		Findings:  []SecurityFinding{},
		Summary:   ScanSummary{TotalFindings: 0},
	}

	tools := []ScanTool{
		&mockTool{
			name:      "mock1",
			available: true,
			result:    mockResult,
		},
		&mockTool{
			name:      "mock2",
			available: true,
			result:    mockResult,
		},
	}

	ctx := context.Background()
	results, errors := scanner.scanParallel(ctx, "./...", tools)

	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d", len(errors))
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}
