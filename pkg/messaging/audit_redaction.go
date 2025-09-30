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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// RedactionStrategy defines how sensitive data should be redacted
type RedactionStrategy string

const (
	// RedactionStrategyMask replaces characters with asterisks
	RedactionStrategyMask RedactionStrategy = "mask"
	// RedactionStrategyHash replaces with a SHA256 hash
	RedactionStrategyHash RedactionStrategy = "hash"
	// RedactionStrategyRemove completely removes the field
	RedactionStrategyRemove RedactionStrategy = "remove"
	// RedactionStrategyPartial shows first/last few characters
	RedactionStrategyPartial RedactionStrategy = "partial"
)

// RedactionRule defines a rule for redacting sensitive data
type RedactionRule struct {
	// FieldNames to redact (supports wildcards)
	FieldNames []string `json:"field_names" yaml:"field_names"`

	// PatternType for automatic detection
	PatternType string `json:"pattern_type,omitempty" yaml:"pattern_type,omitempty"`

	// CustomPattern for custom regex matching
	CustomPattern string `json:"custom_pattern,omitempty" yaml:"custom_pattern,omitempty"`

	// Strategy for redaction
	Strategy RedactionStrategy `json:"strategy" yaml:"strategy" default:"mask"`

	// PreserveLength keeps the same length when masking
	PreserveLength bool `json:"preserve_length" yaml:"preserve_length" default:"true"`

	// PartialShow shows N characters at start and end (for partial strategy)
	PartialShowStart int `json:"partial_show_start,omitempty" yaml:"partial_show_start,omitempty" default:"4"`
	PartialShowEnd   int `json:"partial_show_end,omitempty" yaml:"partial_show_end,omitempty" default:"4"`
}

// RedactionConfig defines configuration for data redaction
type RedactionConfig struct {
	// Enabled indicates if redaction is enabled
	Enabled bool `json:"enabled" yaml:"enabled" default:"true"`

	// Rules for redaction
	Rules []RedactionRule `json:"rules" yaml:"rules"`

	// AutoDetect enables automatic detection of sensitive data
	AutoDetect bool `json:"auto_detect" yaml:"auto_detect" default:"true"`

	// DefaultStrategy for auto-detected fields
	DefaultStrategy RedactionStrategy `json:"default_strategy" yaml:"default_strategy" default:"mask"`
}

// Redactor handles redaction of sensitive data in audit events
type Redactor struct {
	config   *RedactionConfig
	patterns map[string]*regexp.Regexp
}

// Common sensitive data patterns
var (
	// Email pattern
	emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	// Credit card pattern (basic)
	creditCardPattern = regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`)

	// Phone number pattern (US format)
	phonePattern = regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`)

	// IP address pattern
	ipPattern = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)

	// JWT token pattern
	jwtPattern = regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`)

	// Common sensitive field names
	sensitiveFieldNames = []string{
		"password", "passwd", "pwd", "secret", "token", "key", "api_key", "apikey",
		"access_token", "refresh_token", "auth_token", "authorization",
		"credit_card", "creditcard", "card_number", "cvv", "ssn",
		"private_key", "privatekey", "credentials", "credential",
	}
)

// NewRedactor creates a new redactor
func NewRedactor(config *RedactionConfig) (*Redactor, error) {
	if config == nil {
		config = &RedactionConfig{Enabled: false}
	}

	redactor := &Redactor{
		config:   config,
		patterns: make(map[string]*regexp.Regexp),
	}

	// Compile custom patterns
	for _, rule := range config.Rules {
		if rule.CustomPattern != "" {
			pattern, err := regexp.Compile(rule.CustomPattern)
			if err != nil {
				return nil, fmt.Errorf("invalid regex pattern '%s': %w", rule.CustomPattern, err)
			}
			redactor.patterns[rule.CustomPattern] = pattern
		}
	}

	return redactor, nil
}

// RedactEvent redacts sensitive data in an audit event
func (r *Redactor) RedactEvent(event *AuditEvent) *AuditEvent {
	if !r.config.Enabled || event == nil {
		return event
	}

	// Create a copy to avoid modifying the original
	redacted := *event

	// Redact headers
	if redacted.Headers != nil {
		redacted.Headers = r.redactMap(redacted.Headers)
	}

	// Redact metadata
	if redacted.Metadata != nil {
		redacted.Metadata = r.redactMetadata(redacted.Metadata)
	}

	// Redact auth context if present
	if redacted.AuthContext != nil && redacted.AuthContext.Credentials != nil {
		redacted.AuthContext = r.redactAuthContext(redacted.AuthContext)
	}

	// Redact message ID if it looks sensitive
	if r.config.AutoDetect && r.isSensitiveValue(redacted.MessageID) {
		redacted.MessageID = r.redactString(redacted.MessageID, r.config.DefaultStrategy)
	}

	return &redacted
}

// redactMap redacts sensitive data in a map
func (r *Redactor) redactMap(data map[string]string) map[string]string {
	result := make(map[string]string, len(data))

	for key, value := range data {
		// Check if field name is sensitive
		if r.isSensitiveFieldName(key) {
			result[key] = r.redactString(value, r.config.DefaultStrategy)
			continue
		}

		// Check if value matches sensitive patterns
		if r.config.AutoDetect && r.isSensitiveValue(value) {
			result[key] = r.redactString(value, r.config.DefaultStrategy)
			continue
		}

		// Check custom rules
		redacted := false
		for _, rule := range r.config.Rules {
			if r.matchesFieldName(key, rule.FieldNames) {
				result[key] = r.redactString(value, rule.Strategy)
				redacted = true
				break
			}
		}

		if !redacted {
			result[key] = value
		}
	}

	return result
}

// redactMetadata redacts sensitive data in metadata
func (r *Redactor) redactMetadata(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(data))

	for key, value := range data {
		// Check if field name is sensitive
		if r.isSensitiveFieldName(key) {
			result[key] = r.redactValue(value, r.config.DefaultStrategy)
			continue
		}

		// Check custom rules
		redacted := false
		for _, rule := range r.config.Rules {
			if r.matchesFieldName(key, rule.FieldNames) {
				result[key] = r.redactValue(value, rule.Strategy)
				redacted = true
				break
			}
		}

		if !redacted {
			// Recursively redact nested objects
			switch v := value.(type) {
			case map[string]interface{}:
				result[key] = r.redactMetadata(v)
			case map[string]string:
				result[key] = r.redactMap(v)
			case string:
				if r.config.AutoDetect && r.isSensitiveValue(v) {
					result[key] = r.redactString(v, r.config.DefaultStrategy)
				} else {
					result[key] = value
				}
			default:
				result[key] = value
			}
		}
	}

	return result
}

// redactAuthContext redacts sensitive data in auth context
func (r *Redactor) redactAuthContext(ctx *AuthContext) *AuthContext {
	if ctx == nil {
		return nil
	}

	redacted := *ctx
	if ctx.Credentials != nil {
		creds := *ctx.Credentials
		// Always redact tokens and secrets
		if creds.Token != "" {
			creds.Token = r.redactString(creds.Token, RedactionStrategyPartial)
		}
		redacted.Credentials = &creds
	}

	return &redacted
}

// redactString redacts a string value based on strategy
func (r *Redactor) redactString(value string, strategy RedactionStrategy) string {
	if value == "" {
		return value
	}

	switch strategy {
	case RedactionStrategyMask:
		return strings.Repeat("*", len(value))

	case RedactionStrategyHash:
		// Simple hash representation (not cryptographic)
		return fmt.Sprintf("<hash-%08x>", hashString(value))

	case RedactionStrategyRemove:
		return "<redacted>"

	case RedactionStrategyPartial:
		if len(value) <= 8 {
			return strings.Repeat("*", len(value))
		}
		start := 4
		end := len(value) - 4
		if end <= start {
			return strings.Repeat("*", len(value))
		}
		return value[:start] + strings.Repeat("*", end-start) + value[end:]

	default:
		return strings.Repeat("*", len(value))
	}
}

// redactValue redacts any value based on strategy
func (r *Redactor) redactValue(value interface{}, strategy RedactionStrategy) interface{} {
	switch v := value.(type) {
	case string:
		return r.redactString(v, strategy)
	case map[string]interface{}:
		return r.redactMetadata(v)
	case map[string]string:
		return r.redactMap(v)
	default:
		// For other types, convert to JSON and redact
		if data, err := json.Marshal(value); err == nil {
			return r.redactString(string(data), strategy)
		}
		return "<redacted>"
	}
}

// isSensitiveFieldName checks if a field name is sensitive
func (r *Redactor) isSensitiveFieldName(name string) bool {
	lowerName := strings.ToLower(name)
	for _, sensitive := range sensitiveFieldNames {
		if strings.Contains(lowerName, sensitive) {
			return true
		}
	}
	return false
}

// isSensitiveValue checks if a value matches sensitive patterns
func (r *Redactor) isSensitiveValue(value string) bool {
	// Check common patterns
	if emailPattern.MatchString(value) {
		return false // Emails are often not sensitive in audit context
	}

	if creditCardPattern.MatchString(value) {
		return true
	}

	if jwtPattern.MatchString(value) {
		return true
	}

	// Check if it looks like a long token or key
	if len(value) > 32 && !strings.Contains(value, " ") {
		return true
	}

	return false
}

// matchesFieldName checks if a field name matches any of the given patterns
func (r *Redactor) matchesFieldName(name string, patterns []string) bool {
	lowerName := strings.ToLower(name)
	for _, pattern := range patterns {
		lowerPattern := strings.ToLower(pattern)

		// Simple wildcard matching
		if strings.Contains(lowerPattern, "*") {
			// Convert wildcard to regex
			regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(lowerPattern), `\*`, ".*") + "$"
			if matched, _ := regexp.MatchString(regexPattern, lowerName); matched {
				return true
			}
		} else if lowerName == lowerPattern || strings.Contains(lowerName, lowerPattern) {
			return true
		}
	}
	return false
}

// hashString creates a simple hash of a string
func hashString(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

// SetDefaults sets default values for redaction config
func (c *RedactionConfig) SetDefaults() {
	c.Enabled = true
	c.AutoDetect = true
	c.DefaultStrategy = RedactionStrategyMask

	// Add default rules for common sensitive fields
	if len(c.Rules) == 0 {
		c.Rules = []RedactionRule{
			{
				FieldNames:       []string{"password", "passwd", "pwd"},
				Strategy:         RedactionStrategyMask,
				PreserveLength:   false,
				PartialShowStart: 0,
				PartialShowEnd:   0,
			},
			{
				FieldNames:       []string{"token", "*_token", "api_key", "apikey", "secret"},
				Strategy:         RedactionStrategyPartial,
				PreserveLength:   true,
				PartialShowStart: 4,
				PartialShowEnd:   4,
			},
			{
				FieldNames:       []string{"credit_card", "card_number", "cvv"},
				Strategy:         RedactionStrategyPartial,
				PreserveLength:   true,
				PartialShowStart: 4,
				PartialShowEnd:   4,
			},
		}
	}
}

// Validate validates the redaction configuration
func (c *RedactionConfig) Validate() error {
	for i, rule := range c.Rules {
		if len(rule.FieldNames) == 0 && rule.PatternType == "" && rule.CustomPattern == "" {
			return fmt.Errorf("rule %d: must specify field_names, pattern_type, or custom_pattern", i)
		}

		if rule.CustomPattern != "" {
			if _, err := regexp.Compile(rule.CustomPattern); err != nil {
				return fmt.Errorf("rule %d: invalid custom_pattern: %w", i, err)
			}
		}

		// Validate strategy
		switch rule.Strategy {
		case RedactionStrategyMask, RedactionStrategyHash, RedactionStrategyRemove, RedactionStrategyPartial:
			// Valid
		default:
			return fmt.Errorf("rule %d: invalid strategy '%s'", i, rule.Strategy)
		}
	}

	return nil
}
