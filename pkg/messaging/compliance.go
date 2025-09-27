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

package messaging

import (
	"fmt"
	"strings"
	"time"
)

// ComplianceConfig defines compliance-related settings for auditing, retention and reporting
// across messaging brokers. This focuses on configuration schema and validation, and can be
// extended by runtime components for enforcement and metrics emission.
type ComplianceConfig struct {
	// Enabled toggles compliance features
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Retention controls audit trail/data retention.
	Retention *ComplianceRetentionConfig `yaml:"retention,omitempty" json:"retention,omitempty"`

	// Redaction controls scrubbing of sensitive fields for logs/audit.
	Redaction *ComplianceRedactionConfig `yaml:"redaction,omitempty" json:"redaction,omitempty"`

	// Residency controls region residency restrictions.
	Residency *ComplianceResidencyConfig `yaml:"residency,omitempty" json:"residency,omitempty"`

	// Reporting controls how compliance information integrates with monitoring.
	Reporting *ComplianceReportingConfig `yaml:"reporting,omitempty" json:"reporting,omitempty"`
}

// ComplianceRetentionConfig defines retention behavior for audit/compliance trails.
type ComplianceRetentionConfig struct {
	// Enabled toggles retention management (does not perform deletion by itself here; used for policy checks).
	Enabled bool `yaml:"enabled" json:"enabled"`

	// DefaultPeriod controls default retention period for audit logs and compliance artifacts.
	DefaultPeriod time.Duration `yaml:"default_period" json:"default_period"`

	// TopicOverrides allows per-topic retention overrides (topic -> duration).
	TopicOverrides map[string]time.Duration `yaml:"topic_overrides,omitempty" json:"topic_overrides,omitempty"`
}

// ComplianceRedactionConfig defines how sensitive content is scrubbed for compliance.
type ComplianceRedactionConfig struct {
	// Enabled toggles redaction in audit events and logs.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Fields indicates header keys that should be redacted when included in audit events.
	Fields []string `yaml:"fields,omitempty" json:"fields,omitempty"`

	// PayloadEnabled indicates if message payload should be redacted entirely in audit events.
	PayloadEnabled bool `yaml:"payload_enabled" json:"payload_enabled"`
}

// ComplianceResidencyConfig defines region residency constraints.
type ComplianceResidencyConfig struct {
	// Enforce toggles enforcement checks for residency.
	Enforce bool `yaml:"enforce" json:"enforce"`

	// AllowedRegions lists regions where data may reside.
	AllowedRegions []string `yaml:"allowed_regions,omitempty" json:"allowed_regions,omitempty"`
}

// ComplianceReportingConfig defines integration with monitoring/metrics.
type ComplianceReportingConfig struct {
	// Enabled toggles reporting compliance metrics or logs.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// MetricsEnabled emits compliance metrics via the monitoring subsystem.
	MetricsEnabled bool `yaml:"metrics_enabled" json:"metrics_enabled"`

	// LogEnabled emits compliance summaries/violations to logs.
	LogEnabled bool `yaml:"log_enabled" json:"log_enabled"`

	// Interval for periodic reporting/aggregation.
	Interval time.Duration `yaml:"interval" json:"interval"`
}

// Validate validates the compliance configuration.
func (c *ComplianceConfig) Validate() error {
	if c == nil || !c.Enabled {
		return nil
	}

	if c.Retention != nil {
		if err := c.Retention.Validate(); err != nil {
			return fmt.Errorf("invalid compliance.retention config: %w", err)
		}
	}
	if c.Redaction != nil {
		if err := c.Redaction.Validate(); err != nil {
			return fmt.Errorf("invalid compliance.redaction config: %w", err)
		}
	}
	if c.Residency != nil {
		if err := c.Residency.Validate(); err != nil {
			return fmt.Errorf("invalid compliance.residency config: %w", err)
		}
	}
	if c.Reporting != nil {
		if err := c.Reporting.Validate(); err != nil {
			return fmt.Errorf("invalid compliance.reporting config: %w", err)
		}
	}
	return nil
}

// SetDefaults sets default values for compliance configuration.
func (c *ComplianceConfig) SetDefaults() {
	if c == nil {
		return
	}
	if c.Retention == nil {
		c.Retention = &ComplianceRetentionConfig{}
	}
	c.Retention.SetDefaults()

	if c.Redaction == nil {
		c.Redaction = &ComplianceRedactionConfig{}
	}
	c.Redaction.SetDefaults()

	if c.Residency == nil {
		c.Residency = &ComplianceResidencyConfig{}
	}
	c.Residency.SetDefaults()

	if c.Reporting == nil {
		c.Reporting = &ComplianceReportingConfig{}
	}
	c.Reporting.SetDefaults()
}

// Validate validates retention settings.
func (r *ComplianceRetentionConfig) Validate() error {
	if !r.Enabled {
		return nil
	}
	if r.DefaultPeriod <= 0 {
		return fmt.Errorf("default_period must be positive when retention is enabled")
	}
	for topic, d := range r.TopicOverrides {
		if strings.TrimSpace(topic) == "" {
			return fmt.Errorf("topic_overrides contains empty topic name")
		}
		if d <= 0 {
			return fmt.Errorf("topic_overrides[%s] must be positive duration", topic)
		}
	}
	return nil
}

// SetDefaults sets sane defaults for retention.
func (r *ComplianceRetentionConfig) SetDefaults() {
	if r.DefaultPeriod == 0 {
		r.DefaultPeriod = 168 * time.Hour // 7d
	}
	if r.TopicOverrides == nil {
		r.TopicOverrides = make(map[string]time.Duration)
	}
}

// Validate validates redaction settings.
func (r *ComplianceRedactionConfig) Validate() error {
	if !r.Enabled {
		return nil
	}
	// No illegal values; fields can be empty. PayloadEnabled is a boolean.
	return nil
}

// SetDefaults sets sane defaults for redaction.
func (r *ComplianceRedactionConfig) SetDefaults() {
	// Default to not redact payload; redact no headers by default.
}

// Validate validates residency settings.
func (r *ComplianceResidencyConfig) Validate() error {
	if !r.Enforce {
		return nil
	}
	if len(r.AllowedRegions) == 0 {
		return fmt.Errorf("allowed_regions must be provided when residency.enforce is true")
	}
	return nil
}

// SetDefaults sets sane defaults for residency.
func (r *ComplianceResidencyConfig) SetDefaults() {
	// No opinionated defaults.
}

// Validate validates reporting settings.
func (r *ComplianceReportingConfig) Validate() error {
	if !r.Enabled {
		return nil
	}
	if r.Interval <= 0 {
		return fmt.Errorf("reporting.interval must be positive when reporting is enabled")
	}
	return nil
}

// SetDefaults sets sane defaults for reporting.
func (r *ComplianceReportingConfig) SetDefaults() {
	if r.Interval == 0 {
		r.Interval = 1 * time.Minute
	}
}
