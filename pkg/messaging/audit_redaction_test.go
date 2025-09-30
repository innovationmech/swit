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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactionConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *RedactionConfig
		expectError bool
	}{
		{
			name: "Valid config",
			config: &RedactionConfig{
				Enabled:    true,
				AutoDetect: true,
				Rules: []RedactionRule{
					{
						FieldNames: []string{"password"},
						Strategy:   RedactionStrategyMask,
					},
				},
			},
			expectError: false,
		},
		{
			name: "Invalid strategy",
			config: &RedactionConfig{
				Enabled: true,
				Rules: []RedactionRule{
					{
						FieldNames: []string{"password"},
						Strategy:   "invalid",
					},
				},
			},
			expectError: true,
		},
		{
			name: "Invalid custom pattern",
			config: &RedactionConfig{
				Enabled: true,
				Rules: []RedactionRule{
					{
						CustomPattern: "[invalid(regex",
						Strategy:      RedactionStrategyMask,
					},
				},
			},
			expectError: true,
		},
		{
			name: "Rule without any matching criteria",
			config: &RedactionConfig{
				Enabled: true,
				Rules: []RedactionRule{
					{
						Strategy: RedactionStrategyMask,
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedactionConfig_SetDefaults(t *testing.T) {
	config := &RedactionConfig{}
	config.SetDefaults()

	assert.True(t, config.Enabled)
	assert.True(t, config.AutoDetect)
	assert.Equal(t, RedactionStrategyMask, config.DefaultStrategy)
	assert.NotEmpty(t, config.Rules)
}

func TestRedactor_RedactString(t *testing.T) {
	config := &RedactionConfig{
		Enabled: true,
	}
	redactor, err := NewRedactor(config)
	require.NoError(t, err)

	tests := []struct {
		name     string
		value    string
		strategy RedactionStrategy
		expected func(string) bool
	}{
		{
			name:     "Mask strategy",
			value:    "password123",
			strategy: RedactionStrategyMask,
			expected: func(s string) bool { return s == "***********" },
		},
		{
			name:     "Hash strategy",
			value:    "secret",
			strategy: RedactionStrategyHash,
			expected: func(s string) bool { return strings.HasPrefix(s, "<hash-") },
		},
		{
			name:     "Remove strategy",
			value:    "secret",
			strategy: RedactionStrategyRemove,
			expected: func(s string) bool { return s == "<redacted>" },
		},
		{
			name:     "Partial strategy",
			value:    "verylongsecret",
			strategy: RedactionStrategyPartial,
			expected: func(s string) bool {
				return strings.HasPrefix(s, "very") && strings.HasSuffix(s, "cret") && strings.Contains(s, "****")
			},
		},
		{
			name:     "Partial strategy - short string",
			value:    "short",
			strategy: RedactionStrategyPartial,
			expected: func(s string) bool { return s == "*****" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.redactString(tt.value, tt.strategy)
			assert.True(t, tt.expected(result), "Result: %s", result)
		})
	}
}

func TestRedactor_RedactMap(t *testing.T) {
	config := &RedactionConfig{
		Enabled:    true,
		AutoDetect: true,
		Rules: []RedactionRule{
			{
				FieldNames: []string{"password", "api_key"},
				Strategy:   RedactionStrategyMask,
			},
		},
	}
	redactor, err := NewRedactor(config)
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]bool // key -> should be redacted
	}{
		{
			name: "Password field",
			input: map[string]string{
				"username": "user123",
				"password": "secret",
			},
			expected: map[string]bool{
				"username": false,
				"password": true,
			},
		},
		{
			name: "API key field",
			input: map[string]string{
				"name":    "service",
				"api_key": "sk_test_123456789",
			},
			expected: map[string]bool{
				"name":    false,
				"api_key": true,
			},
		},
		{
			name: "Token field",
			input: map[string]string{
				"user":  "test",
				"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc",
			},
			expected: map[string]bool{
				"user":  false,
				"token": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.redactMap(tt.input)
			for key, shouldBeRedacted := range tt.expected {
				if shouldBeRedacted {
					assert.NotEqual(t, tt.input[key], result[key], "Field %s should be redacted", key)
					assert.Contains(t, result[key], "*", "Redacted field should contain asterisks")
				} else {
					assert.Equal(t, tt.input[key], result[key], "Field %s should not be redacted", key)
				}
			}
		})
	}
}

func TestRedactor_RedactEvent(t *testing.T) {
	config := &RedactionConfig{
		Enabled:    true,
		AutoDetect: true,
	}
	config.SetDefaults()
	redactor, err := NewRedactor(config)
	require.NoError(t, err)

	t.Run("Redact headers", func(t *testing.T) {
		event := &AuditEvent{
			Type:      AuditEventMessagePublished,
			Timestamp: time.Now(),
			MessageID: "msg-123",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
				"X-Api-Key":     "sk_live_123456789",
			},
		}

		redacted := redactor.RedactEvent(event)

		assert.NotNil(t, redacted)
		assert.Equal(t, "application/json", redacted.Headers["Content-Type"])
		assert.NotEqual(t, event.Headers["Authorization"], redacted.Headers["Authorization"])
		assert.NotEqual(t, event.Headers["X-Api-Key"], redacted.Headers["X-Api-Key"])
	})

	t.Run("Redact metadata", func(t *testing.T) {
		event := &AuditEvent{
			Type:      AuditEventMessagePublished,
			Timestamp: time.Now(),
			MessageID: "msg-123",
			Metadata: map[string]interface{}{
				"service":  "test-service",
				"password": "secret123",
				"api_key":  "key_123",
			},
		}

		redacted := redactor.RedactEvent(event)

		assert.NotNil(t, redacted)
		assert.Equal(t, "test-service", redacted.Metadata["service"])
		assert.NotEqual(t, event.Metadata["password"], redacted.Metadata["password"])
		assert.NotEqual(t, event.Metadata["api_key"], redacted.Metadata["api_key"])
	})

	t.Run("Redact auth context", func(t *testing.T) {
		event := &AuditEvent{
			Type:      AuditEventAuthSuccess,
			Timestamp: time.Now(),
			MessageID: "msg-123",
			AuthContext: &AuthContext{
				Credentials: &AuthCredentials{
					UserID: "user-123",
					Token:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature",
					Scopes: []string{"read", "write"},
				},
				Timestamp: time.Now(),
				Source:    "test",
			},
		}

		redacted := redactor.RedactEvent(event)

		assert.NotNil(t, redacted)
		assert.NotNil(t, redacted.AuthContext)
		assert.NotNil(t, redacted.AuthContext.Credentials)
		assert.Equal(t, "user-123", redacted.AuthContext.Credentials.UserID)
		assert.NotEqual(t, event.AuthContext.Credentials.Token, redacted.AuthContext.Credentials.Token)
		assert.Contains(t, redacted.AuthContext.Credentials.Token, "*")
	})

	t.Run("Do not modify original event", func(t *testing.T) {
		originalToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test"
		event := &AuditEvent{
			Type:      AuditEventMessagePublished,
			Timestamp: time.Now(),
			MessageID: "msg-123",
			Headers: map[string]string{
				"Authorization": originalToken,
			},
		}

		redacted := redactor.RedactEvent(event)

		// Original event should remain unchanged
		assert.Equal(t, originalToken, event.Headers["Authorization"])
		// Redacted event should have masked token
		assert.NotEqual(t, originalToken, redacted.Headers["Authorization"])
	})
}

func TestRedactor_IsSensitiveFieldName(t *testing.T) {
	config := &RedactionConfig{Enabled: true}
	redactor, err := NewRedactor(config)
	require.NoError(t, err)

	tests := []struct {
		name      string
		fieldName string
		expected  bool
	}{
		{"Password", "password", true},
		{"API Key", "api_key", true},
		{"Token", "access_token", true},
		{"Secret", "client_secret", true},
		{"Credit Card", "credit_card", true},
		{"Normal field", "username", false},
		{"Email", "email", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.isSensitiveFieldName(tt.fieldName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactor_IsSensitiveValue(t *testing.T) {
	config := &RedactionConfig{Enabled: true}
	redactor, err := NewRedactor(config)
	require.NoError(t, err)

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"Credit card", "1234-5678-9012-3456", true},
		{"JWT token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc", true},
		{"Long token", "sk_live_" + strings.Repeat("x", 30), true},
		{"Email", "user@example.com", false},
		{"Normal text", "hello world", false},
		{"Short string", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.isSensitiveValue(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactor_CustomRules(t *testing.T) {
	config := &RedactionConfig{
		Enabled: true,
		Rules: []RedactionRule{
			{
				FieldNames: []string{"custom_*"},
				Strategy:   RedactionStrategyPartial,
			},
			{
				CustomPattern: `\btest-\d+\b`,
				Strategy:      RedactionStrategyHash,
			},
		},
	}

	redactor, err := NewRedactor(config)
	require.NoError(t, err)

	t.Run("Wildcard field name matching", func(t *testing.T) {
		data := map[string]string{
			"custom_field1": "value1",
			"custom_field2": "value2",
			"other_field":   "value3",
		}

		result := redactor.redactMap(data)

		assert.NotEqual(t, data["custom_field1"], result["custom_field1"])
		assert.NotEqual(t, data["custom_field2"], result["custom_field2"])
		assert.Equal(t, data["other_field"], result["other_field"])
	})
}

func TestHashString(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Simple string", "hello"},
		{"Complex string", "complex-string-123"},
		{"Empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hashString(tt.input)
			hash2 := hashString(tt.input)

			// Same input should produce same hash
			assert.Equal(t, hash1, hash2)

			// Different inputs should (likely) produce different hashes
			if tt.input != "" {
				differentHash := hashString(tt.input + "x")
				assert.NotEqual(t, hash1, differentHash)
			}
		})
	}
}
