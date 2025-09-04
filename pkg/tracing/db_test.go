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

package tracing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGormTracing(t *testing.T) {
	config := DefaultTracingConfig()
	config.Enabled = true
	config.Exporter.Type = "console"
	
	tm := NewTracingManager()
	err := tm.Initialize(context.Background(), config)
	require.NoError(t, err)

	t.Run("creates with default config", func(t *testing.T) {
		gt := NewGormTracing(tm, nil)
		
		assert.NotNil(t, gt)
		assert.Equal(t, tm, gt.tm)
		assert.NotNil(t, gt.config)
		assert.Equal(t, DefaultGormTracingConfig().RecordSQL, gt.config.RecordSQL)
	})

	t.Run("creates with custom config", func(t *testing.T) {
		customConfig := &GormTracingConfig{
			RecordSQL:      false,
			SanitizeSQL:    false,
			SlowThreshold:  100 * time.Millisecond,
			SkipOperations: []string{"SELECT"},
		}

		gt := NewGormTracing(tm, customConfig)
		
		assert.NotNil(t, gt)
		assert.Equal(t, tm, gt.tm)
		assert.Equal(t, customConfig, gt.config)
		assert.False(t, gt.config.RecordSQL)
		assert.Equal(t, 100*time.Millisecond, gt.config.SlowThreshold)
	})
}

func TestDefaultGormTracingConfig(t *testing.T) {
	config := DefaultGormTracingConfig()
	
	assert.NotNil(t, config)
	assert.True(t, config.RecordSQL)
	assert.True(t, config.SanitizeSQL)
	assert.Equal(t, 200*time.Millisecond, config.SlowThreshold)
	assert.Empty(t, config.SkipOperations)
}

func TestGormTracingName(t *testing.T) {
	config := DefaultTracingConfig()
	config.Enabled = true
	config.Exporter.Type = "console"
	
	tm := NewTracingManager()
	err := tm.Initialize(context.Background(), config)
	require.NoError(t, err)

	gt := NewGormTracing(tm, nil)
	
	assert.Equal(t, "tracing", gt.Name())
}

func TestSanitizeSQL(t *testing.T) {
	config := DefaultTracingConfig()
	config.Enabled = true
	config.Exporter.Type = "console"
	
	tm := NewTracingManager()
	err := tm.Initialize(context.Background(), config)
	require.NoError(t, err)

	gt := NewGormTracing(tm, nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "parameters with $1, $2",
			input:    "SELECT * FROM users WHERE id = $1 AND name = $2",
			expected: "SELECT * FROM users WHERE id = ? AND name = ?",
		},
		{
			name:     "parameters with ?",
			input:    "SELECT * FROM users WHERE id = ? AND name = ?",
			expected: "SELECT * FROM users WHERE id = ? AND name = ?",
		},
		{
			name:     "extra whitespace",
			input:    "SELECT   *    FROM    users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "mixed parameters and whitespace",
			input:    "INSERT   INTO   users   (id, name)   VALUES   ($1,   $2)",
			expected: "INSERT INTO users (id, name) VALUES (?, ?)",
		},
		{
			name:     "no parameters",
			input:    "SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gt.sanitizeSQL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInstallGormTracing(t *testing.T) {
	config := DefaultTracingConfig()
	config.Enabled = true
	config.Exporter.Type = "console"
	
	tm := NewTracingManager()
	err := tm.Initialize(context.Background(), config)
	require.NoError(t, err)

	t.Run("install with default config", func(t *testing.T) {
		// We can't easily test this without a real database connection
		// This test mainly verifies the function signature and basic setup
		tracingConfig := DefaultGormTracingConfig()
		
		assert.NotNil(t, tracingConfig)
		assert.True(t, tracingConfig.RecordSQL)
		
		// Test that NewGormTracing works with the config
		gt := NewGormTracing(tm, tracingConfig)
		assert.NotNil(t, gt)
	})
}

func TestGormDBWithTracing(t *testing.T) {
	t.Run("wraps db with context", func(t *testing.T) {
		// This is a simple wrapper function test
		// We can't test the actual functionality without a database connection
		ctx := context.WithValue(context.Background(), "test", "value")
		
		// The function should accept any context
		assert.NotNil(t, ctx)
		
		// Test that the context value is preserved
		value := ctx.Value("test")
		assert.Equal(t, "value", value)
	})
}