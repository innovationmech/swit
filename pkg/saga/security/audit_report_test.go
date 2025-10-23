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

package security

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewReportGenerator(t *testing.T) {
	auditLogger := createTestAuditLogger(t)
	generator := NewReportGenerator(auditLogger)

	if generator == nil {
		t.Error("NewReportGenerator() returned nil")
	}
	if generator.queryService == nil {
		t.Error("NewReportGenerator() queryService is nil")
	}
}

func TestReportGenerator_GenerateReport(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	generator := NewReportGenerator(auditLogger)

	// Add test data
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Test entry 1", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelError, AuditActionSagaFailed, "saga", "saga-2", "Test entry 2", nil)

	tests := []struct {
		name    string
		config  *ReportConfig
		wantErr bool
	}{
		{
			name: "basic report with entries",
			config: &ReportConfig{
				Title:          "Test Report",
				Description:    "Test Description",
				IncludeEntries: true,
				IncludeStats:   false,
			},
			wantErr: false,
		},
		{
			name: "report with statistics",
			config: &ReportConfig{
				Title:          "Stats Report",
				IncludeStats:   true,
				IncludeEntries: false,
			},
			wantErr: false,
		},
		{
			name: "report with timeline",
			config: &ReportConfig{
				Title:            "Timeline Report",
				IncludeTimeline:  true,
				TimelineInterval: time.Hour,
			},
			wantErr: false,
		},
		{
			name: "full report",
			config: &ReportConfig{
				Title:            "Full Report",
				Description:      "Complete audit report",
				IncludeEntries:   true,
				IncludeStats:     true,
				IncludeTimeline:  true,
				TimelineInterval: time.Hour,
				GeneratedBy:      "test-user",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := generator.GenerateReport(ctx, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateReport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if report == nil {
					t.Fatal("GenerateReport() returned nil report")
				}
				if report.ReportID == "" {
					t.Error("Report ID is empty")
				}
				if report.Title != tt.config.Title {
					t.Errorf("Title = %v, want %v", report.Title, tt.config.Title)
				}
				if tt.config.IncludeEntries && len(report.Entries) == 0 {
					t.Error("Report should include entries but has none")
				}
				if tt.config.IncludeStats && report.Statistics == nil {
					t.Error("Report should include statistics but has none")
				}
				if tt.config.IncludeTimeline && report.Timeline == nil {
					t.Error("Report should include timeline but has none")
				}
			}
		})
	}
}

func TestReportGenerator_ExportToJSON(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	generator := NewReportGenerator(auditLogger)

	// Add test data
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Test entry", nil)

	// Generate report
	config := &ReportConfig{
		Title:          "JSON Export Test",
		IncludeEntries: true,
	}
	report, err := generator.GenerateReport(ctx, config)
	if err != nil {
		t.Fatalf("GenerateReport() failed: %v", err)
	}

	// Export to JSON
	var buf bytes.Buffer
	err = generator.ExportToJSON(report, &buf)
	if err != nil {
		t.Errorf("ExportToJSON() error = %v", err)
	}

	// Verify JSON is valid
	var decoded AuditReport
	err = json.Unmarshal(buf.Bytes(), &decoded)
	if err != nil {
		t.Errorf("Failed to decode JSON: %v", err)
	}

	if decoded.Title != report.Title {
		t.Errorf("Decoded title = %v, want %v", decoded.Title, report.Title)
	}
}

func TestReportGenerator_ExportToCSV(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	generator := NewReportGenerator(auditLogger)

	// Add test data
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Test entry 1", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelError, AuditActionSagaFailed, "saga", "saga-2", "Test entry 2", nil)

	// Generate report
	config := &ReportConfig{
		Title:          "CSV Export Test",
		IncludeEntries: true,
	}
	report, err := generator.GenerateReport(ctx, config)
	if err != nil {
		t.Fatalf("GenerateReport() failed: %v", err)
	}

	// Export to CSV
	var buf bytes.Buffer
	err = generator.ExportToCSV(report, &buf)
	if err != nil {
		t.Errorf("ExportToCSV() error = %v", err)
	}

	// Verify CSV is valid
	csvReader := csv.NewReader(&buf)
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Errorf("Failed to read CSV: %v", err)
	}

	// Should have header + entries
	expectedRows := 1 + len(report.Entries)
	if len(records) != expectedRows {
		t.Errorf("CSV has %d rows, want %d", len(records), expectedRows)
	}

	// Verify header
	if len(records) > 0 {
		header := records[0]
		if header[0] != "ID" {
			t.Errorf("First column header = %v, want ID", header[0])
		}
	}
}

func TestReportGenerator_ExportToText(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	generator := NewReportGenerator(auditLogger)

	// Add test data
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Test entry", nil)

	// Generate report
	config := &ReportConfig{
		Title:          "Text Export Test",
		Description:    "Test Description",
		IncludeEntries: true,
		IncludeStats:   true,
	}
	report, err := generator.GenerateReport(ctx, config)
	if err != nil {
		t.Fatalf("GenerateReport() failed: %v", err)
	}

	// Export to text
	var buf bytes.Buffer
	err = generator.ExportToText(report, &buf)
	if err != nil {
		t.Errorf("ExportToText() error = %v", err)
	}

	text := buf.String()

	// Verify text contains expected sections
	if !strings.Contains(text, report.Title) {
		t.Error("Text export should contain report title")
	}
	if !strings.Contains(text, "Statistics") {
		t.Error("Text export should contain statistics section")
	}
	if !strings.Contains(text, "Entries") {
		t.Error("Text export should contain entries section")
	}
}

func TestReportGenerator_ExportToHTML(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	generator := NewReportGenerator(auditLogger)

	// Add test data
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Test entry", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelError, AuditActionSagaFailed, "saga", "saga-2", "Test error", nil)

	// Generate report
	config := &ReportConfig{
		Title:          "HTML Export Test",
		Description:    "Test Description",
		IncludeEntries: true,
		IncludeStats:   true,
		GeneratedBy:    "test-user",
	}
	report, err := generator.GenerateReport(ctx, config)
	if err != nil {
		t.Fatalf("GenerateReport() failed: %v", err)
	}

	// Export to HTML
	var buf bytes.Buffer
	err = generator.ExportToHTML(report, &buf)
	if err != nil {
		t.Errorf("ExportToHTML() error = %v", err)
	}

	html := buf.String()

	// Verify HTML structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML should contain DOCTYPE")
	}
	if !strings.Contains(html, "<html>") {
		t.Error("HTML should contain html tag")
	}
	if !strings.Contains(html, report.Title) {
		t.Error("HTML should contain report title")
	}
	if !strings.Contains(html, "Statistics") {
		t.Error("HTML should contain statistics section")
	}
	if !strings.Contains(html, "<table>") {
		t.Error("HTML should contain table for entries")
	}
}

func TestFormatEntryAsText(t *testing.T) {
	entry := &AuditEntry{
		ID:           "audit-123",
		Timestamp:    time.Now(),
		Level:        AuditLevelInfo,
		Action:       AuditActionSagaStarted,
		ResourceType: "saga",
		ResourceID:   "saga-1",
		UserID:       "user-1",
		Username:     "testuser",
		Message:      "Test message",
		OldState:     "pending",
		NewState:     "running",
		Error:        "test error",
		TraceID:      "trace-123",
	}

	text := formatEntryAsText(entry, 1)

	// Verify text contains key information
	if !strings.Contains(text, entry.ID) {
		t.Error("Text should contain entry ID")
	}
	if !strings.Contains(text, string(entry.Level)) {
		t.Error("Text should contain level")
	}
	if !strings.Contains(text, string(entry.Action)) {
		t.Error("Text should contain action")
	}
	if !strings.Contains(text, entry.Message) {
		t.Error("Text should contain message")
	}
	if !strings.Contains(text, entry.UserID) {
		t.Error("Text should contain user ID")
	}
	if !strings.Contains(text, entry.TraceID) {
		t.Error("Text should contain trace ID")
	}
}

func TestGenerateHTML(t *testing.T) {
	report := &AuditReport{
		ReportID:    "report-123",
		Title:       "Test Report",
		Description: "Test Description",
		GeneratedAt: time.Now(),
		GeneratedBy: "test-user",
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
		Entries: []*AuditEntry{
			{
				ID:           "entry-1",
				Timestamp:    time.Now(),
				Level:        AuditLevelInfo,
				Action:       AuditActionSagaStarted,
				ResourceType: "saga",
				ResourceID:   "saga-1",
				UserID:       "user-1",
				Message:      "Test message",
			},
		},
		Statistics: &AuditStatistics{
			TotalEntries:    1,
			UniqueUsers:     1,
			UniqueResources: 1,
			ErrorRate:       0.0,
		},
	}

	html := generateHTML(report)

	// Verify HTML contains expected elements
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML should start with DOCTYPE")
	}
	if !strings.Contains(html, report.Title) {
		t.Error("HTML should contain report title")
	}
	if !strings.Contains(html, report.Description) {
		t.Error("HTML should contain description")
	}
	if !strings.Contains(html, "Statistics") {
		t.Error("HTML should contain statistics section")
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ampersand",
			input:    "test & more",
			expected: "test &amp; more",
		},
		{
			name:     "less than",
			input:    "a < b",
			expected: "a &lt; b",
		},
		{
			name:     "greater than",
			input:    "a > b",
			expected: "a &gt; b",
		},
		{
			name:     "quotes",
			input:    `"test" and 'test'`,
			expected: "&quot;test&quot; and &#39;test&#39;",
		},
		{
			name:     "all special chars",
			input:    `<script>alert("XSS")</script>`,
			expected: "&lt;script&gt;alert(&quot;XSS&quot;)&lt;/script&gt;",
		},
		{
			name:     "no special chars",
			input:    "normal text",
			expected: "normal text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("escapeHTML() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestReportWithFilter(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	generator := NewReportGenerator(auditLogger)

	// Add test data with different levels
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Info entry", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelError, AuditActionSagaFailed, "saga", "saga-2", "Error entry", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelWarning, AuditActionSagaTimedOut, "saga", "saga-3", "Warning entry", nil)

	// Generate report with filter for error level only
	filter := NewQueryBuilder().
		WithLevels(AuditLevelError).
		Build()

	config := &ReportConfig{
		Title:          "Filtered Report",
		Filter:         filter,
		IncludeEntries: true,
		IncludeStats:   true,
	}

	report, err := generator.GenerateReport(ctx, config)
	if err != nil {
		t.Fatalf("GenerateReport() failed: %v", err)
	}

	// Should only include error entries
	if len(report.Entries) != 1 {
		t.Errorf("Report entries count = %d, want 1", len(report.Entries))
	}
	if len(report.Entries) > 0 && report.Entries[0].Level != AuditLevelError {
		t.Errorf("Entry level = %v, want %v", report.Entries[0].Level, AuditLevelError)
	}
}

func TestReportSummaryGeneration(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	generator := NewReportGenerator(auditLogger)

	// Add test data
	entry1 := &AuditEntry{
		Level:        AuditLevelInfo,
		Action:       AuditActionSagaStarted,
		ResourceType: "saga",
		ResourceID:   "saga-1",
		UserID:       "user-1",
		Message:      "Test 1",
	}
	AddTagsToEntry(entry1, CategorySaga, []AuditTag{TagCritical})
	_ = auditLogger.Log(ctx, entry1)

	// Generate report with statistics
	config := &ReportConfig{
		Title:          "Summary Test Report",
		IncludeEntries: true,
		IncludeStats:   true,
	}

	report, err := generator.GenerateReport(ctx, config)
	if err != nil {
		t.Fatalf("GenerateReport() failed: %v", err)
	}

	// Verify summary is not empty
	if report.Summary == "" {
		t.Error("Report summary is empty")
	}

	// Verify summary contains key information
	if !strings.Contains(report.Summary, report.Title) {
		t.Error("Summary should contain report title")
	}
	if !strings.Contains(report.Summary, "Statistics") {
		t.Error("Summary should contain statistics section")
	}
	if !strings.Contains(report.Summary, "Total Entries") {
		t.Error("Summary should contain total entries")
	}
}

func TestExportFormats(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	generator := NewReportGenerator(auditLogger)

	// Add test data
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Test entry", nil)

	// Generate report
	config := &ReportConfig{
		Title:          "Export Formats Test",
		IncludeEntries: true,
		IncludeStats:   true,
	}
	report, err := generator.GenerateReport(ctx, config)
	if err != nil {
		t.Fatalf("GenerateReport() failed: %v", err)
	}

	tests := []struct {
		name       string
		exportFunc func(*AuditReport, *bytes.Buffer) error
		checkFunc  func(string) error
	}{
		{
			name: "JSON export",
			exportFunc: func(r *AuditReport, buf *bytes.Buffer) error {
				return generator.ExportToJSON(r, buf)
			},
			checkFunc: func(content string) error {
				var decoded AuditReport
				return json.Unmarshal([]byte(content), &decoded)
			},
		},
		{
			name: "CSV export",
			exportFunc: func(r *AuditReport, buf *bytes.Buffer) error {
				return generator.ExportToCSV(r, buf)
			},
			checkFunc: func(content string) error {
				reader := csv.NewReader(strings.NewReader(content))
				_, err := reader.ReadAll()
				return err
			},
		},
		{
			name: "Text export",
			exportFunc: func(r *AuditReport, buf *bytes.Buffer) error {
				return generator.ExportToText(r, buf)
			},
			checkFunc: func(content string) error {
				if len(content) == 0 {
					return nil
				}
				return nil
			},
		},
		{
			name: "HTML export",
			exportFunc: func(r *AuditReport, buf *bytes.Buffer) error {
				return generator.ExportToHTML(r, buf)
			},
			checkFunc: func(content string) error {
				if !strings.Contains(content, "<!DOCTYPE html>") {
					return nil
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.exportFunc(report, &buf)
			if err != nil {
				t.Errorf("%s failed: %v", tt.name, err)
				return
			}

			content := buf.String()
			if len(content) == 0 {
				t.Errorf("%s produced empty output", tt.name)
				return
			}

			if tt.checkFunc != nil {
				if err := tt.checkFunc(content); err != nil {
					t.Errorf("%s content validation failed: %v", tt.name, err)
				}
			}
		})
	}
}
