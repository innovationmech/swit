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

package state

import (
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDefaultRecoveryPolicyConfig(t *testing.T) {
	config := DefaultRecoveryPolicyConfig()
	assert.NotNil(t, config)
	assert.NotNil(t, config.DefaultPolicy)
	assert.Equal(t, RecoveryStrategyRetry, config.DefaultPolicy.Strategy)
	assert.Equal(t, 3, config.DefaultPolicy.MaxRetryAttempts)
	assert.Equal(t, 5*time.Second, config.DefaultPolicy.RetryDelay)
	assert.NotNil(t, config.ByDefinition)
	assert.NotNil(t, config.ByErrorType)
	assert.True(t, config.EnablePolicyOverride)
}

func TestValidatePolicyConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *RecoveryPolicyConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &RecoveryPolicyConfig{
				DefaultPolicy: &PolicyRule{
					Strategy:         RecoveryStrategyRetry,
					MaxRetryAttempts: 3,
					RetryDelay:       5 * time.Second,
				},
				ByDefinition: make(map[string]*PolicyRule),
				ByErrorType:  make(map[string]*PolicyRule),
			},
			wantErr: false,
		},
		{
			name: "missing default policy",
			config: &RecoveryPolicyConfig{
				ByDefinition: make(map[string]*PolicyRule),
				ByErrorType:  make(map[string]*PolicyRule),
			},
			wantErr: true,
		},
		{
			name: "invalid retry strategy without max attempts",
			config: &RecoveryPolicyConfig{
				DefaultPolicy: &PolicyRule{
					Strategy:         RecoveryStrategyRetry,
					MaxRetryAttempts: 0,
					RetryDelay:       5 * time.Second,
				},
				ByDefinition: make(map[string]*PolicyRule),
				ByErrorType:  make(map[string]*PolicyRule),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePolicyConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePolicyRule(t *testing.T) {
	tests := []struct {
		name    string
		rule    *PolicyRule
		wantErr bool
	}{
		{
			name:    "nil rule",
			rule:    nil,
			wantErr: true,
		},
		{
			name: "valid retry rule",
			rule: &PolicyRule{
				Strategy:         RecoveryStrategyRetry,
				MaxRetryAttempts: 3,
				RetryDelay:       5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid compensate rule",
			rule: &PolicyRule{
				Strategy: RecoveryStrategyCompensate,
			},
			wantErr: false,
		},
		{
			name: "missing strategy",
			rule: &PolicyRule{
				MaxRetryAttempts: 3,
			},
			wantErr: true,
		},
		{
			name: "retry with zero max attempts",
			rule: &PolicyRule{
				Strategy:         RecoveryStrategyRetry,
				MaxRetryAttempts: 0,
			},
			wantErr: true,
		},
		{
			name: "negative retry delay",
			rule: &PolicyRule{
				Strategy:         RecoveryStrategyRetry,
				MaxRetryAttempts: 3,
				RetryDelay:       -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePolicyRule(tt.rule)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPolicyManager_GetPolicy(t *testing.T) {
	config := &RecoveryPolicyConfig{
		DefaultPolicy: &PolicyRule{
			Strategy:         RecoveryStrategyRetry,
			MaxRetryAttempts: 3,
			RetryDelay:       5 * time.Second,
		},
		ByDefinition: map[string]*PolicyRule{
			"order-saga": {
				Strategy: RecoveryStrategyCompensate,
			},
		},
		ByErrorType: map[string]*PolicyRule{
			string(saga.ErrorTypeTimeout): {
				Strategy:         RecoveryStrategyRetry,
				MaxRetryAttempts: 5,
				RetryDelay:       10 * time.Second,
			},
		},
		EnablePolicyOverride: true,
	}

	pm := NewPolicyManager(config, zap.NewNop())

	tests := []struct {
		name           string
		definitionID   string
		errorType      saga.ErrorType
		expectedPolicy RecoveryStrategyType
	}{
		{
			name:           "get definition policy",
			definitionID:   "order-saga",
			errorType:      "",
			expectedPolicy: RecoveryStrategyCompensate,
		},
		{
			name:           "get error type policy with override enabled",
			definitionID:   "",
			errorType:      saga.ErrorTypeTimeout,
			expectedPolicy: RecoveryStrategyRetry,
		},
		{
			name:           "get default policy",
			definitionID:   "unknown-saga",
			errorType:      "",
			expectedPolicy: RecoveryStrategyRetry,
		},
		{
			name:           "definition overrides error type",
			definitionID:   "order-saga",
			errorType:      saga.ErrorTypeTimeout,
			expectedPolicy: RecoveryStrategyCompensate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := pm.GetPolicy(tt.definitionID, tt.errorType)
			assert.NotNil(t, policy)
			assert.Equal(t, tt.expectedPolicy, policy.Strategy)
		})
	}
}

func TestPolicyManager_UpdateConfig(t *testing.T) {
	pm := NewPolicyManager(nil, zap.NewNop())

	newConfig := &RecoveryPolicyConfig{
		DefaultPolicy: &PolicyRule{
			Strategy:         RecoveryStrategyCompensate,
			MaxRetryAttempts: 5,
		},
		ByDefinition:         make(map[string]*PolicyRule),
		ByErrorType:          make(map[string]*PolicyRule),
		EnablePolicyOverride: false,
	}

	err := pm.UpdateConfig(newConfig)
	assert.NoError(t, err)

	// Verify config was updated
	config := pm.GetConfig()
	assert.Equal(t, RecoveryStrategyCompensate, config.DefaultPolicy.Strategy)
	assert.Equal(t, 5, config.DefaultPolicy.MaxRetryAttempts)
	assert.False(t, config.EnablePolicyOverride)
}

func TestPolicyManager_UpdateConfig_Invalid(t *testing.T) {
	pm := NewPolicyManager(nil, zap.NewNop())

	invalidConfig := &RecoveryPolicyConfig{
		DefaultPolicy: nil, // Missing default policy
	}

	err := pm.UpdateConfig(invalidConfig)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidPolicyConfig)
}

func TestPolicyManager_SetDefinitionPolicy(t *testing.T) {
	pm := NewPolicyManager(nil, zap.NewNop())

	rule := &PolicyRule{
		Strategy:         RecoveryStrategyRetry,
		MaxRetryAttempts: 5,
		RetryDelay:       10 * time.Second,
	}

	err := pm.SetDefinitionPolicy("test-saga", rule)
	assert.NoError(t, err)

	// Verify policy was set
	policy := pm.GetPolicy("test-saga", "")
	assert.Equal(t, RecoveryStrategyRetry, policy.Strategy)
	assert.Equal(t, 5, policy.MaxRetryAttempts)
}

func TestPolicyManager_SetDefinitionPolicy_Invalid(t *testing.T) {
	pm := NewPolicyManager(nil, zap.NewNop())

	// Empty definition ID
	err := pm.SetDefinitionPolicy("", &PolicyRule{Strategy: RecoveryStrategyRetry})
	assert.Error(t, err)

	// Invalid rule
	err = pm.SetDefinitionPolicy("test-saga", &PolicyRule{})
	assert.Error(t, err)
}

func TestPolicyManager_SetErrorTypePolicy(t *testing.T) {
	pm := NewPolicyManager(nil, zap.NewNop())

	rule := &PolicyRule{
		Strategy:         RecoveryStrategyRetry,
		MaxRetryAttempts: 3,
	}

	err := pm.SetErrorTypePolicy(saga.ErrorTypeTimeout, rule)
	assert.NoError(t, err)

	// Verify policy was set
	policy := pm.GetPolicy("", saga.ErrorTypeTimeout)
	assert.Equal(t, RecoveryStrategyRetry, policy.Strategy)
}

func TestPolicyManager_RemoveDefinitionPolicy(t *testing.T) {
	pm := NewPolicyManager(nil, zap.NewNop())

	// Set a policy
	rule := &PolicyRule{
		Strategy:         RecoveryStrategyRetry,
		MaxRetryAttempts: 3,
	}
	err := pm.SetDefinitionPolicy("test-saga", rule)
	require.NoError(t, err)

	// Remove the policy
	pm.RemoveDefinitionPolicy("test-saga")

	// Verify policy was removed (should fall back to default)
	policy := pm.GetPolicy("test-saga", "")
	assert.Equal(t, pm.GetConfig().DefaultPolicy.Strategy, policy.Strategy)
}

func TestPolicyManager_RemoveErrorTypePolicy(t *testing.T) {
	pm := NewPolicyManager(nil, zap.NewNop())

	// Set a policy
	rule := &PolicyRule{
		Strategy:         RecoveryStrategyRetry,
		MaxRetryAttempts: 5,
	}
	err := pm.SetErrorTypePolicy(saga.ErrorTypeTimeout, rule)
	require.NoError(t, err)

	// Remove the policy
	pm.RemoveErrorTypePolicy(saga.ErrorTypeTimeout)

	// Verify policy was removed (should fall back to default)
	policy := pm.GetPolicy("", saga.ErrorTypeTimeout)
	assert.Equal(t, pm.GetConfig().DefaultPolicy.Strategy, policy.Strategy)
}

func TestPolicyManager_GetConfig(t *testing.T) {
	pm := NewPolicyManager(nil, zap.NewNop())

	config := pm.GetConfig()
	assert.NotNil(t, config)
	assert.NotNil(t, config.DefaultPolicy)
	assert.NotNil(t, config.ByDefinition)
	assert.NotNil(t, config.ByErrorType)
}

func TestPolicyRule_Copy(t *testing.T) {
	original := &PolicyRule{
		Strategy:                   RecoveryStrategyRetry,
		MaxRetryAttempts:           3,
		RetryDelay:                 5 * time.Second,
		Timeout:                    1 * time.Minute,
		RequiresManualIntervention: false,
	}

	copy := copyPolicyRule(original)
	assert.NotNil(t, copy)
	assert.Equal(t, original.Strategy, copy.Strategy)
	assert.Equal(t, original.MaxRetryAttempts, copy.MaxRetryAttempts)
	assert.Equal(t, original.RetryDelay, copy.RetryDelay)
	assert.Equal(t, original.Timeout, copy.Timeout)
	assert.Equal(t, original.RequiresManualIntervention, copy.RequiresManualIntervention)

	// Verify it's a deep copy
	copy.MaxRetryAttempts = 10
	assert.Equal(t, 3, original.MaxRetryAttempts)
}

func TestPolicyRule_CopyNil(t *testing.T) {
	copy := copyPolicyRule(nil)
	assert.Nil(t, copy)
}
