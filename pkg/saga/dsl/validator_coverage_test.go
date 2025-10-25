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

package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateRetryPolicies_CoverageBoost tests additional scenarios for retry policies
func TestValidateRetryPolicies_CoverageBoost(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		shouldErr bool
	}{
		{
			name: "linear backoff with increment",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
global_retry_policy:
  type: linear_backoff
  max_attempts: 3
  initial_delay: 100ms
  increment: 50ms
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			shouldErr: false,
		},
		{
			name: "fixed delay retry",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
global_retry_policy:
  type: fixed_delay
  max_attempts: 5
  initial_delay: 2s
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			shouldErr: false,
		},
		{
			name: "no retry type",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
global_retry_policy:
  type: no_retry
  max_attempts: 1
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestValidateCompensation_CoverageBoost tests additional compensation scenarios
func TestValidateCompensation_CoverageBoost(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		shouldErr bool
	}{
		{
			name: "global compensation with parallel strategy",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
global_compensation:
  strategy: parallel
  timeout: 2m
  max_attempts: 3
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
    compensation:
      type: custom
      action:
        service:
          name: test-service
          method: DELETE
`,
			shouldErr: false,
		},
		{
			name: "global compensation with best effort",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
global_compensation:
  strategy: best_effort
  timeout: 1m
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			shouldErr: false,
		},
		{
			name: "compensation with on_failure ignore",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
    compensation:
      type: custom
      action:
        service:
          name: test-service
          method: DELETE
      on_failure:
        action: ignore
`,
			shouldErr: false,
		},
		{
			name: "compensation with on_failure retry",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
    compensation:
      type: custom
      action:
        service:
          name: test-service
          method: DELETE
      on_failure:
        action: retry
        max_retries: 3
`,
			shouldErr: false,
		},
		{
			name: "compensation with on_failure fail",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
    compensation:
      type: custom
      action:
        service:
          name: test-service
          method: DELETE
      on_failure:
        action: fail
`,
			shouldErr: false,
		},
		{
			name: "compensation with on_failure alert",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
    compensation:
      type: custom
      action:
        service:
          name: test-service
          method: DELETE
      on_failure:
        action: alert
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestValidateTimeouts_CoverageBoost tests additional timeout scenarios
func TestValidateTimeouts_CoverageBoost(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		shouldErr bool
	}{
		{
			name: "global compensation timeout minimum",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
global_compensation:
  strategy: sequential
  timeout: 1s
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			shouldErr: false,
		},
		{
			name: "service timeout minimum valid",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
        timeout: 100ms
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestFormatValidatorMessage_AllTags tests all validator tag types
func TestFormatValidatorMessage_AllTags(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		shouldErr bool
		contains  []string
	}{
		{
			name: "oneof validation",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
  mode: invalid-mode
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			shouldErr: true,
			contains:  []string{"validation"},
		},
		{
			name: "min validation",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
global_retry_policy:
  type: exponential_backoff
  max_attempts: 0
  initial_delay: 1s
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			shouldErr: true,
			contains:  []string{"validation"},
		},
		{
			name: "max validation - large but valid",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
global_retry_policy:
  type: exponential_backoff
  max_attempts: 150
  initial_delay: 1s
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			shouldErr: false, // 150 is valid, just large
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			if tt.shouldErr {
				require.Error(t, err)
				for _, expected := range tt.contains {
					assert.Contains(t, err.Error(), expected)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestValidateStepAction_CoverageBoost tests all step action types for coverage
func TestValidateStepAction_CoverageBoost(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		shouldErr bool
	}{
		{
			name: "http type without service",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: http
    action:
      function:
        name: test-func
        handler: pkg.Handler
`,
			shouldErr: true,
		},
		{
			name: "grpc type without service",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: grpc
    action:
      function:
        name: test-func
        handler: pkg.Handler
`,
			shouldErr: true,
		},
		{
			name: "function without handler",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: function
    action:
      function:
        name: test-func
`,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestValidateRetryPolicy_EdgeCases tests edge cases for retry policy validation
func TestValidateRetryPolicy_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		shouldErr bool
	}{
		{
			name: "linear backoff with small increment is allowed",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
global_retry_policy:
  type: linear_backoff
  max_attempts: 3
  initial_delay: 100ms
  increment: 5ms
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			shouldErr: false, // Small increment is allowed
		},
		{
			name: "fixed delay with small initial delay is allowed",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
global_retry_policy:
  type: fixed_delay
  max_attempts: 3
  initial_delay: 5ms
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			shouldErr: false, // Small delay is allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestValidateGlobalCompensation_EdgeCases tests edge cases for global compensation
func TestValidateGlobalCompensation_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		shouldErr bool
	}{
		{
			name: "global compensation with zero max attempts is allowed",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
global_compensation:
  strategy: sequential
  max_attempts: 0
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
`,
			shouldErr: false, // Zero max_attempts is allowed (unlimited or default behavior)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestValidateCompensationAction_EdgeCases tests edge cases for compensation actions
func TestValidateCompensationAction_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		shouldErr bool
	}{
		{
			name: "compensation with zero max attempts is allowed",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
    compensation:
      type: custom
      max_attempts: 0
      action:
        service:
          name: test-service
          method: DELETE
`,
			shouldErr: false,
		},
		{
			name: "compensation with short timeout is allowed",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
    compensation:
      type: custom
      timeout: 50ms
      action:
        service:
          name: test-service
          method: DELETE
`,
			shouldErr: false,
		},
		{
			name: "compensation with automatic type and strategy",
			yaml: `
saga:
  id: test-saga
  name: Test Saga
steps:
  - id: step1
    name: Step 1
    type: service
    action:
      service:
        name: test-service
        method: POST
    compensation:
      type: automatic
      strategy: parallel
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
