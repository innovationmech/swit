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
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

var (
	// ErrInvalidPolicyConfig is returned when the policy config is invalid.
	ErrInvalidPolicyConfig = errors.New("invalid policy config")

	// ErrPolicyNotFound is returned when a policy is not found.
	ErrPolicyNotFound = errors.New("policy not found")
)

// RecoveryPolicyConfig defines the recovery policy configuration.
// It supports global defaults, per-definition policies, and per-error-type policies.
type RecoveryPolicyConfig struct {
	// DefaultPolicy is the default recovery policy for all Sagas.
	DefaultPolicy *PolicyRule `json:"default" yaml:"default"`

	// ByDefinition maps definition IDs to their specific recovery policies.
	ByDefinition map[string]*PolicyRule `json:"by_definition" yaml:"by_definition"`

	// ByErrorType maps error types to their specific recovery policies.
	ByErrorType map[string]*PolicyRule `json:"by_error_type" yaml:"by_error_type"`

	// EnablePolicyOverride allows definition policies to override error type policies.
	// If false, error type policies take precedence.
	EnablePolicyOverride bool `json:"enable_policy_override" yaml:"enable_policy_override"`
}

// PolicyRule defines a single recovery policy rule.
type PolicyRule struct {
	// Strategy is the recovery strategy to use.
	Strategy RecoveryStrategyType `json:"strategy" yaml:"strategy"`

	// MaxRetryAttempts is the maximum number of retry attempts (for retry strategy).
	MaxRetryAttempts int `json:"max_retry_attempts" yaml:"max_retry_attempts"`

	// RetryDelay is the delay between retry attempts.
	RetryDelay time.Duration `json:"retry_delay" yaml:"retry_delay"`

	// Timeout is the timeout for the recovery operation.
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// RequiresManualIntervention indicates if manual intervention is required.
	RequiresManualIntervention bool `json:"requires_manual_intervention" yaml:"requires_manual_intervention"`
}

// PolicyManager manages recovery policies.
type PolicyManager struct {
	config *RecoveryPolicyConfig
	mu     sync.RWMutex
	logger *zap.Logger
}

// NewPolicyManager creates a new PolicyManager.
func NewPolicyManager(config *RecoveryPolicyConfig, logger *zap.Logger) *PolicyManager {
	if config == nil {
		config = DefaultRecoveryPolicyConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	return &PolicyManager{
		config: config,
		logger: logger.With(zap.String("component", "policy_manager")),
	}
}

// DefaultRecoveryPolicyConfig returns a default recovery policy configuration.
func DefaultRecoveryPolicyConfig() *RecoveryPolicyConfig {
	return &RecoveryPolicyConfig{
		DefaultPolicy: &PolicyRule{
			Strategy:         RecoveryStrategyRetry,
			MaxRetryAttempts: 3,
			RetryDelay:       5 * time.Second,
			Timeout:          5 * time.Minute,
		},
		ByDefinition:         make(map[string]*PolicyRule),
		ByErrorType:          make(map[string]*PolicyRule),
		EnablePolicyOverride: true,
	}
}

// ValidatePolicyConfig validates the recovery policy configuration.
func ValidatePolicyConfig(config *RecoveryPolicyConfig) error {
	if config == nil {
		return ErrInvalidPolicyConfig
	}

	// Validate default policy
	if config.DefaultPolicy == nil {
		return fmt.Errorf("%w: default policy is required", ErrInvalidPolicyConfig)
	}

	if err := validatePolicyRule(config.DefaultPolicy); err != nil {
		return fmt.Errorf("%w: default policy invalid: %v", ErrInvalidPolicyConfig, err)
	}

	// Validate by-definition policies
	for defID, rule := range config.ByDefinition {
		if err := validatePolicyRule(rule); err != nil {
			return fmt.Errorf("%w: policy for definition %s invalid: %v", ErrInvalidPolicyConfig, defID, err)
		}
	}

	// Validate by-error-type policies
	for errType, rule := range config.ByErrorType {
		if err := validatePolicyRule(rule); err != nil {
			return fmt.Errorf("%w: policy for error type %s invalid: %v", ErrInvalidPolicyConfig, errType, err)
		}
	}

	return nil
}

// validatePolicyRule validates a single policy rule.
func validatePolicyRule(rule *PolicyRule) error {
	if rule == nil {
		return errors.New("policy rule is nil")
	}

	if rule.Strategy == "" {
		return errors.New("strategy is required")
	}

	if rule.Strategy == RecoveryStrategyRetry {
		if rule.MaxRetryAttempts <= 0 {
			return errors.New("max_retry_attempts must be positive for retry strategy")
		}
		if rule.RetryDelay < 0 {
			return errors.New("retry_delay cannot be negative")
		}
	}

	if rule.Timeout < 0 {
		return errors.New("timeout cannot be negative")
	}

	return nil
}

// GetPolicy returns the appropriate recovery policy for a Saga.
// It applies the policy selection priority:
// 1. Error type policy (if error type is provided)
// 2. Definition policy (if EnablePolicyOverride is true)
// 3. Definition policy (always if no error type)
// 4. Default policy
func (pm *PolicyManager) GetPolicy(definitionID string, errorType saga.ErrorType) *PolicyRule {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Try error type policy first (if provided and override is disabled)
	if errorType != "" && !pm.config.EnablePolicyOverride {
		if rule, exists := pm.config.ByErrorType[string(errorType)]; exists {
			pm.logger.Debug("using error type policy",
				zap.String("definition_id", definitionID),
				zap.String("error_type", string(errorType)),
				zap.String("strategy", string(rule.Strategy)),
			)
			return rule
		}
	}

	// Try definition policy
	if definitionID != "" {
		if rule, exists := pm.config.ByDefinition[definitionID]; exists {
			pm.logger.Debug("using definition policy",
				zap.String("definition_id", definitionID),
				zap.String("strategy", string(rule.Strategy)),
			)
			return rule
		}
	}

	// Try error type policy (if override is enabled)
	if errorType != "" && pm.config.EnablePolicyOverride {
		if rule, exists := pm.config.ByErrorType[string(errorType)]; exists {
			pm.logger.Debug("using error type policy",
				zap.String("definition_id", definitionID),
				zap.String("error_type", string(errorType)),
				zap.String("strategy", string(rule.Strategy)),
			)
			return rule
		}
	}

	// Fall back to default policy
	pm.logger.Debug("using default policy",
		zap.String("definition_id", definitionID),
		zap.String("strategy", string(pm.config.DefaultPolicy.Strategy)),
	)
	return pm.config.DefaultPolicy
}

// UpdateConfig updates the recovery policy configuration at runtime.
func (pm *PolicyManager) UpdateConfig(config *RecoveryPolicyConfig) error {
	if err := ValidatePolicyConfig(config); err != nil {
		return err
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.config = config

	pm.logger.Info("recovery policy config updated",
		zap.String("default_strategy", string(config.DefaultPolicy.Strategy)),
		zap.Int("by_definition_count", len(config.ByDefinition)),
		zap.Int("by_error_type_count", len(config.ByErrorType)),
	)

	return nil
}

// GetConfig returns a copy of the current policy configuration.
func (pm *PolicyManager) GetConfig() *RecoveryPolicyConfig {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Deep copy to prevent external modification
	configCopy := &RecoveryPolicyConfig{
		DefaultPolicy:        copyPolicyRule(pm.config.DefaultPolicy),
		ByDefinition:         make(map[string]*PolicyRule),
		ByErrorType:          make(map[string]*PolicyRule),
		EnablePolicyOverride: pm.config.EnablePolicyOverride,
	}

	for k, v := range pm.config.ByDefinition {
		configCopy.ByDefinition[k] = copyPolicyRule(v)
	}

	for k, v := range pm.config.ByErrorType {
		configCopy.ByErrorType[k] = copyPolicyRule(v)
	}

	return configCopy
}

// SetDefinitionPolicy sets a policy for a specific definition.
func (pm *PolicyManager) SetDefinitionPolicy(definitionID string, rule *PolicyRule) error {
	if definitionID == "" {
		return errors.New("definition ID cannot be empty")
	}

	if err := validatePolicyRule(rule); err != nil {
		return fmt.Errorf("invalid policy rule: %w", err)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.config.ByDefinition[definitionID] = rule

	pm.logger.Info("definition policy set",
		zap.String("definition_id", definitionID),
		zap.String("strategy", string(rule.Strategy)),
	)

	return nil
}

// SetErrorTypePolicy sets a policy for a specific error type.
func (pm *PolicyManager) SetErrorTypePolicy(errorType saga.ErrorType, rule *PolicyRule) error {
	if errorType == "" {
		return errors.New("error type cannot be empty")
	}

	if err := validatePolicyRule(rule); err != nil {
		return fmt.Errorf("invalid policy rule: %w", err)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.config.ByErrorType[string(errorType)] = rule

	pm.logger.Info("error type policy set",
		zap.String("error_type", string(errorType)),
		zap.String("strategy", string(rule.Strategy)),
	)

	return nil
}

// RemoveDefinitionPolicy removes a policy for a specific definition.
func (pm *PolicyManager) RemoveDefinitionPolicy(definitionID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.config.ByDefinition, definitionID)

	pm.logger.Info("definition policy removed",
		zap.String("definition_id", definitionID),
	)
}

// RemoveErrorTypePolicy removes a policy for a specific error type.
func (pm *PolicyManager) RemoveErrorTypePolicy(errorType saga.ErrorType) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.config.ByErrorType, string(errorType))

	pm.logger.Info("error type policy removed",
		zap.String("error_type", string(errorType)),
	)
}

// copyPolicyRule creates a deep copy of a policy rule.
func copyPolicyRule(rule *PolicyRule) *PolicyRule {
	if rule == nil {
		return nil
	}

	return &PolicyRule{
		Strategy:                   rule.Strategy,
		MaxRetryAttempts:           rule.MaxRetryAttempts,
		RetryDelay:                 rule.RetryDelay,
		Timeout:                    rule.Timeout,
		RequiresManualIntervention: rule.RequiresManualIntervention,
	}
}

// selectRecoveryStrategy selects the appropriate recovery strategy based on the policy.
func (rm *RecoveryManager) selectRecoveryStrategy(ctx context.Context, sagaInst saga.SagaInstance, errorType saga.ErrorType) (RecoveryStrategy, error) {
	// Get policy from policy manager if available
	var policyRule *PolicyRule
	if rm.policyManager != nil {
		policyRule = rm.policyManager.GetPolicy(sagaInst.GetDefinitionID(), errorType)
	}

	// If no policy or manual intervention required, return nil
	if policyRule != nil && policyRule.RequiresManualIntervention {
		rm.logger.Info("manual intervention required for saga",
			zap.String("saga_id", sagaInst.GetID()),
			zap.String("definition_id", sagaInst.GetDefinitionID()),
		)
		return nil, errors.New("manual intervention required")
	}

	// Select strategy based on policy or default logic
	var strategyType RecoveryStrategyType
	if policyRule != nil {
		strategyType = policyRule.Strategy
	} else {
		// Default strategy selection logic
		switch sagaInst.GetState() {
		case saga.StateRunning, saga.StateStepCompleted:
			strategyType = RecoveryStrategyRetry
		case saga.StateCompensating:
			strategyType = RecoveryStrategyContinue
		default:
			strategyType = RecoveryStrategyRetry
		}
	}

	// Create strategy based on type
	switch strategyType {
	case RecoveryStrategyRetry:
		maxAttempts := 3
		if policyRule != nil && policyRule.MaxRetryAttempts > 0 {
			maxAttempts = policyRule.MaxRetryAttempts
		}
		return NewRetryRecoveryStrategy(maxAttempts), nil

	case RecoveryStrategyCompensate:
		return NewCompensateRecoveryStrategy(), nil

	case RecoveryStrategyMarkFailed:
		return NewMarkFailedRecoveryStrategy(), nil

	case RecoveryStrategyContinue:
		return NewContinueRecoveryStrategy(), nil

	default:
		return nil, fmt.Errorf("unsupported recovery strategy: %s", strategyType)
	}
}
