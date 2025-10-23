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

// Package security provides audit report generation functionality.
package security

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// ReportFormat represents the output format for audit reports
type ReportFormat string

const (
	// ReportFormatJSON exports audit data in JSON format
	ReportFormatJSON ReportFormat = "json"
	// ReportFormatCSV exports audit data in CSV format
	ReportFormatCSV ReportFormat = "csv"
	// ReportFormatText exports audit data in plain text format
	ReportFormatText ReportFormat = "text"
	// ReportFormatHTML exports audit data in HTML format
	ReportFormatHTML ReportFormat = "html"
)

// AuditReport represents a generated audit report
type AuditReport struct {
	// Report metadata
	ReportID    string       `json:"report_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	GeneratedAt time.Time    `json:"generated_at"`
	GeneratedBy string       `json:"generated_by,omitempty"`
	Format      ReportFormat `json:"format"`

	// Report parameters
	Filter    *AuditFilter `json:"filter,omitempty"`
	StartTime time.Time    `json:"start_time"`
	EndTime   time.Time    `json:"end_time"`

	// Report data
	Entries    []*AuditEntry    `json:"entries,omitempty"`
	Statistics *AuditStatistics `json:"statistics,omitempty"`
	Timeline   *AuditTimeline   `json:"timeline,omitempty"`

	// Summary information
	Summary string `json:"summary,omitempty"`
}

// AuditReportGenerator generates audit reports
type AuditReportGenerator interface {
	// GenerateReport generates a report based on the provided filter
	GenerateReport(ctx context.Context, config *ReportConfig) (*AuditReport, error)

	// ExportToJSON exports a report to JSON format
	ExportToJSON(report *AuditReport, writer io.Writer) error

	// ExportToCSV exports a report to CSV format
	ExportToCSV(report *AuditReport, writer io.Writer) error

	// ExportToText exports a report to plain text format
	ExportToText(report *AuditReport, writer io.Writer) error

	// ExportToHTML exports a report to HTML format
	ExportToHTML(report *AuditReport, writer io.Writer) error
}

// ReportConfig configures report generation
type ReportConfig struct {
	Title            string
	Description      string
	Filter           *AuditFilter
	IncludeStats     bool
	IncludeEntries   bool
	IncludeTimeline  bool
	TimelineInterval time.Duration
	GeneratedBy      string
}

// DefaultReportGenerator implements AuditReportGenerator
type DefaultReportGenerator struct {
	queryService *AuditQueryService
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(auditLogger AuditLogger) *DefaultReportGenerator {
	return &DefaultReportGenerator{
		queryService: NewAuditQueryService(auditLogger),
	}
}

// GenerateReport generates a report based on the provided filter
func (g *DefaultReportGenerator) GenerateReport(ctx context.Context, config *ReportConfig) (*AuditReport, error) {
	if config == nil {
		return nil, fmt.Errorf("report config is required")
	}

	report := &AuditReport{
		ReportID:    fmt.Sprintf("report-%d", time.Now().UnixNano()),
		Title:       config.Title,
		Description: config.Description,
		GeneratedAt: time.Now(),
		GeneratedBy: config.GeneratedBy,
		Filter:      config.Filter,
	}

	// Determine time range
	if config.Filter != nil {
		if config.Filter.StartTime != nil {
			report.StartTime = *config.Filter.StartTime
		}
		if config.Filter.EndTime != nil {
			report.EndTime = *config.Filter.EndTime
		}
	}

	// If time range not specified, use last 24 hours
	if report.StartTime.IsZero() {
		report.StartTime = time.Now().Add(-24 * time.Hour)
	}
	if report.EndTime.IsZero() {
		report.EndTime = time.Now()
	}

	// Get entries if requested
	if config.IncludeEntries {
		entries, err := g.queryService.Query(ctx, config.Filter)
		if err != nil {
			return nil, fmt.Errorf("failed to query entries: %w", err)
		}
		report.Entries = entries
	}

	// Get statistics if requested
	if config.IncludeStats {
		stats, err := g.queryService.GetStatistics(ctx, config.Filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get statistics: %w", err)
		}
		report.Statistics = stats
	}

	// Get timeline if requested
	if config.IncludeTimeline {
		interval := config.TimelineInterval
		if interval == 0 {
			// Default to 1 hour intervals
			interval = time.Hour
		}

		timeline, err := g.queryService.GetTimeline(ctx, report.StartTime, report.EndTime, interval, config.Filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get timeline: %w", err)
		}
		report.Timeline = timeline
	}

	// Generate summary
	report.Summary = g.generateSummary(report)

	return report, nil
}

// generateSummary generates a text summary of the report
func (g *DefaultReportGenerator) generateSummary(report *AuditReport) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Audit Report: %s\n", report.Title))
	summary.WriteString(fmt.Sprintf("Generated: %s\n", report.GeneratedAt.Format(time.RFC3339)))
	summary.WriteString(fmt.Sprintf("Time Range: %s to %s\n\n",
		report.StartTime.Format(time.RFC3339),
		report.EndTime.Format(time.RFC3339)))

	if report.Statistics != nil {
		summary.WriteString("=== Statistics ===\n")
		summary.WriteString(fmt.Sprintf("Total Entries: %d\n", report.Statistics.TotalEntries))
		summary.WriteString(fmt.Sprintf("Unique Users: %d\n", report.Statistics.UniqueUsers))
		summary.WriteString(fmt.Sprintf("Unique Resources: %d\n", report.Statistics.UniqueResources))
		summary.WriteString(fmt.Sprintf("Error Rate: %.2f%%\n\n", report.Statistics.ErrorRate*100))

		if len(report.Statistics.EntriesByLevel) > 0 {
			summary.WriteString("Entries by Level:\n")
			for level, count := range report.Statistics.EntriesByLevel {
				summary.WriteString(fmt.Sprintf("  %s: %d\n", level, count))
			}
			summary.WriteString("\n")
		}

		if len(report.Statistics.EntriesByCategory) > 0 {
			summary.WriteString("Entries by Category:\n")
			for category, count := range report.Statistics.EntriesByCategory {
				summary.WriteString(fmt.Sprintf("  %s: %d\n", category, count))
			}
			summary.WriteString("\n")
		}
	}

	if len(report.Entries) > 0 {
		summary.WriteString("=== Entries ===\n")
		summary.WriteString(fmt.Sprintf("Total: %d entries\n", len(report.Entries)))
	}

	return summary.String()
}

// ExportToJSON exports a report to JSON format
func (g *DefaultReportGenerator) ExportToJSON(report *AuditReport, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// ExportToCSV exports a report to CSV format
func (g *DefaultReportGenerator) ExportToCSV(report *AuditReport, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header
	header := []string{
		"ID",
		"Timestamp",
		"Level",
		"Action",
		"Resource Type",
		"Resource ID",
		"User ID",
		"Username",
		"Source",
		"IP Address",
		"Old State",
		"New State",
		"Message",
		"Error",
		"Trace ID",
		"Correlation ID",
	}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write entries
	for _, entry := range report.Entries {
		row := []string{
			entry.ID,
			entry.Timestamp.Format(time.RFC3339),
			string(entry.Level),
			string(entry.Action),
			entry.ResourceType,
			entry.ResourceID,
			entry.UserID,
			entry.Username,
			entry.Source,
			entry.IPAddress,
			entry.OldState,
			entry.NewState,
			entry.Message,
			entry.Error,
			entry.TraceID,
			entry.CorrelationID,
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// ExportToText exports a report to plain text format
func (g *DefaultReportGenerator) ExportToText(report *AuditReport, writer io.Writer) error {
	// Write summary
	if _, err := writer.Write([]byte(report.Summary)); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	// Write entries
	if len(report.Entries) > 0 {
		if _, err := writer.Write([]byte("\n=== Detailed Entries ===\n\n")); err != nil {
			return err
		}

		for i, entry := range report.Entries {
			text := formatEntryAsText(entry, i+1)
			if _, err := writer.Write([]byte(text)); err != nil {
				return fmt.Errorf("failed to write entry: %w", err)
			}
			if _, err := writer.Write([]byte("\n")); err != nil {
				return err
			}
		}
	}

	return nil
}

// formatEntryAsText formats an audit entry as plain text
func formatEntryAsText(entry *AuditEntry, index int) string {
	var text strings.Builder

	text.WriteString(fmt.Sprintf("[%d] %s\n", index, entry.ID))
	text.WriteString(fmt.Sprintf("Time: %s\n", entry.Timestamp.Format(time.RFC3339)))
	text.WriteString(fmt.Sprintf("Level: %s | Action: %s\n", entry.Level, entry.Action))

	if entry.ResourceType != "" || entry.ResourceID != "" {
		text.WriteString(fmt.Sprintf("Resource: %s/%s\n", entry.ResourceType, entry.ResourceID))
	}

	if entry.UserID != "" {
		text.WriteString(fmt.Sprintf("User: %s", entry.UserID))
		if entry.Username != "" {
			text.WriteString(fmt.Sprintf(" (%s)", entry.Username))
		}
		text.WriteString("\n")
	}

	if entry.OldState != "" || entry.NewState != "" {
		text.WriteString(fmt.Sprintf("State Change: %s -> %s\n", entry.OldState, entry.NewState))
	}

	text.WriteString(fmt.Sprintf("Message: %s\n", entry.Message))

	if entry.Error != "" {
		text.WriteString(fmt.Sprintf("Error: %s\n", entry.Error))
	}

	if entry.TraceID != "" {
		text.WriteString(fmt.Sprintf("Trace ID: %s\n", entry.TraceID))
	}

	text.WriteString(strings.Repeat("-", 80) + "\n")

	return text.String()
}

// ExportToHTML exports a report to HTML format
func (g *DefaultReportGenerator) ExportToHTML(report *AuditReport, writer io.Writer) error {
	html := generateHTML(report)
	_, err := writer.Write([]byte(html))
	return err
}

// generateHTML generates an HTML representation of the report
func generateHTML(report *AuditReport) string {
	var html strings.Builder

	html.WriteString("<!DOCTYPE html>\n")
	html.WriteString("<html>\n<head>\n")
	html.WriteString(fmt.Sprintf("  <title>%s</title>\n", escapeHTML(report.Title)))
	html.WriteString("  <style>\n")
	html.WriteString(`    body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
    .container { max-width: 1200px; margin: 0 auto; background-color: white; padding: 20px; box-shadow: 0 0 10px rgba(0,0,0,0.1); }
    h1 { color: #333; border-bottom: 2px solid #007bff; padding-bottom: 10px; }
    h2 { color: #555; margin-top: 30px; }
    .metadata { background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
    .metadata p { margin: 5px 0; }
    .statistics { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 15px; margin-bottom: 20px; }
    .stat-card { background-color: #e9ecef; padding: 15px; border-radius: 5px; border-left: 4px solid #007bff; }
    .stat-card h3 { margin: 0 0 10px 0; color: #333; font-size: 14px; }
    .stat-card .value { font-size: 24px; font-weight: bold; color: #007bff; }
    table { width: 100%; border-collapse: collapse; margin-top: 20px; }
    th { background-color: #007bff; color: white; padding: 12px; text-align: left; }
    td { padding: 10px; border-bottom: 1px solid #ddd; }
    tr:hover { background-color: #f8f9fa; }
    .level-info { color: #17a2b8; font-weight: bold; }
    .level-warning { color: #ffc107; font-weight: bold; }
    .level-error { color: #dc3545; font-weight: bold; }
    .level-critical { color: #721c24; font-weight: bold; background-color: #f8d7da; padding: 2px 5px; border-radius: 3px; }
  `)
	html.WriteString("  </style>\n</head>\n<body>\n")
	html.WriteString("  <div class=\"container\">\n")

	// Title and metadata
	html.WriteString(fmt.Sprintf("    <h1>%s</h1>\n", escapeHTML(report.Title)))
	html.WriteString("    <div class=\"metadata\">\n")
	if report.Description != "" {
		html.WriteString(fmt.Sprintf("      <p><strong>Description:</strong> %s</p>\n", escapeHTML(report.Description)))
	}
	html.WriteString(fmt.Sprintf("      <p><strong>Generated:</strong> %s</p>\n", report.GeneratedAt.Format(time.RFC3339)))
	if report.GeneratedBy != "" {
		html.WriteString(fmt.Sprintf("      <p><strong>Generated By:</strong> %s</p>\n", escapeHTML(report.GeneratedBy)))
	}
	html.WriteString(fmt.Sprintf("      <p><strong>Time Range:</strong> %s to %s</p>\n",
		report.StartTime.Format(time.RFC3339),
		report.EndTime.Format(time.RFC3339)))
	html.WriteString("    </div>\n")

	// Statistics
	if report.Statistics != nil {
		html.WriteString("    <h2>Statistics</h2>\n")
		html.WriteString("    <div class=\"statistics\">\n")
		html.WriteString(fmt.Sprintf("      <div class=\"stat-card\"><h3>Total Entries</h3><div class=\"value\">%d</div></div>\n", report.Statistics.TotalEntries))
		html.WriteString(fmt.Sprintf("      <div class=\"stat-card\"><h3>Unique Users</h3><div class=\"value\">%d</div></div>\n", report.Statistics.UniqueUsers))
		html.WriteString(fmt.Sprintf("      <div class=\"stat-card\"><h3>Unique Resources</h3><div class=\"value\">%d</div></div>\n", report.Statistics.UniqueResources))
		html.WriteString(fmt.Sprintf("      <div class=\"stat-card\"><h3>Error Rate</h3><div class=\"value\">%.2f%%</div></div>\n", report.Statistics.ErrorRate*100))
		html.WriteString("    </div>\n")
	}

	// Entries table
	if len(report.Entries) > 0 {
		html.WriteString("    <h2>Audit Entries</h2>\n")
		html.WriteString("    <table>\n")
		html.WriteString("      <thead>\n")
		html.WriteString("        <tr>\n")
		html.WriteString("          <th>Timestamp</th>\n")
		html.WriteString("          <th>Level</th>\n")
		html.WriteString("          <th>Action</th>\n")
		html.WriteString("          <th>Resource</th>\n")
		html.WriteString("          <th>User</th>\n")
		html.WriteString("          <th>Message</th>\n")
		html.WriteString("        </tr>\n")
		html.WriteString("      </thead>\n")
		html.WriteString("      <tbody>\n")

		for _, entry := range report.Entries {
			html.WriteString("        <tr>\n")
			html.WriteString(fmt.Sprintf("          <td>%s</td>\n", entry.Timestamp.Format("2006-01-02 15:04:05")))

			// Level with color coding
			levelClass := fmt.Sprintf("level-%s", strings.ToLower(string(entry.Level)))
			html.WriteString(fmt.Sprintf("          <td class=\"%s\">%s</td>\n", levelClass, entry.Level))

			html.WriteString(fmt.Sprintf("          <td>%s</td>\n", escapeHTML(string(entry.Action))))

			resource := ""
			if entry.ResourceType != "" {
				resource = entry.ResourceType
				if entry.ResourceID != "" {
					resource += "/" + entry.ResourceID
				}
			}
			html.WriteString(fmt.Sprintf("          <td>%s</td>\n", escapeHTML(resource)))

			user := entry.UserID
			if entry.Username != "" {
				user = entry.Username
			}
			html.WriteString(fmt.Sprintf("          <td>%s</td>\n", escapeHTML(user)))

			html.WriteString(fmt.Sprintf("          <td>%s</td>\n", escapeHTML(entry.Message)))
			html.WriteString("        </tr>\n")
		}

		html.WriteString("      </tbody>\n")
		html.WriteString("    </table>\n")
	}

	html.WriteString("  </div>\n")
	html.WriteString("</body>\n</html>\n")

	return html.String()
}

// escapeHTML escapes special HTML characters
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
