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
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	// sagaIDPattern validates Saga ID format: alphanumeric, dashes, underscores
	sagaIDPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-_]*[a-zA-Z0-9])?$`)

	// stepIDPattern validates Step ID format
	stepIDPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-_]*[a-zA-Z0-9])?$`)

	// serviceNamePattern validates service name format
	serviceNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
)

// ValidationError represents a validation error with detailed context.
type ValidationError struct {
	Field   string // Field path (e.g., "saga.id", "steps[0].name")
	Message string // Error message
	Value   string // Invalid value
	Rule    string // Validation rule that failed
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("validation error at '%s': %s (value: %s)", e.Field, e.Message, e.Value)
	}
	return fmt.Sprintf("validation error at '%s': %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []*ValidationError

// Error implements the error interface.
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "no validation errors"
	}
	if len(ve) == 1 {
		return ve[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("validation failed with %d error(s):\n", len(ve)))
	for i, err := range ve {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// Validator provides comprehensive validation for Saga definitions.
type Validator struct {
	structValidator *validator.Validate
	customRules     []CustomValidationRule
}

// CustomValidationRule is a function that validates a Saga definition.
type CustomValidationRule func(*SagaDefinition) ValidationErrors

// NewValidator creates a new Validator with default settings.
func NewValidator() *Validator {
	v := &Validator{
		structValidator: validator.New(validator.WithRequiredStructEnabled()),
		customRules:     make([]CustomValidationRule, 0),
	}

	// Register default custom rules
	v.RegisterRule(validateSagaConfig)
	v.RegisterRule(validateSteps)
	v.RegisterRule(validateStepDependencies)
	v.RegisterRule(validateTimeouts)
	v.RegisterRule(validateRetryPolicies)
	v.RegisterRule(validateCompensations)

	return v
}

// RegisterRule adds a custom validation rule.
func (v *Validator) RegisterRule(rule CustomValidationRule) {
	v.customRules = append(v.customRules, rule)
}

// Validate validates a Saga definition against all rules.
func (v *Validator) Validate(def *SagaDefinition) error {
	if def == nil {
		return &ValidationError{
			Field:   "saga_definition",
			Message: "saga definition is nil",
		}
	}

	var allErrors ValidationErrors

	// 1. Struct tag validation
	if err := v.structValidator.Struct(def); err != nil {
		allErrors = append(allErrors, v.convertValidatorErrors(err)...)
	}

	// 2. Custom validation rules
	for _, rule := range v.customRules {
		if errs := rule(def); len(errs) > 0 {
			allErrors = append(allErrors, errs...)
		}
	}

	if len(allErrors) > 0 {
		return allErrors
	}

	return nil
}

// convertValidatorErrors converts validator.ValidationErrors to ValidationErrors.
func (v *Validator) convertValidatorErrors(err error) ValidationErrors {
	var errors ValidationErrors

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			ve := &ValidationError{
				Field:   e.Field(),
				Rule:    e.Tag(),
				Message: formatValidatorMessage(e),
			}
			if e.Value() != nil {
				ve.Value = fmt.Sprintf("%v", e.Value())
			}
			errors = append(errors, ve)
		}
	}

	return errors
}

// formatValidatorMessage formats a validator error into a readable message.
func formatValidatorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "field is required"
	case "min":
		return fmt.Sprintf("value must be at least %s", e.Param())
	case "max":
		return fmt.Sprintf("value must be at most %s", e.Param())
	case "oneof":
		return fmt.Sprintf("value must be one of: %s", e.Param())
	case "dive":
		return "nested validation failed"
	default:
		msg := fmt.Sprintf("validation failed on '%s' tag", e.Tag())
		if e.Param() != "" {
			msg += fmt.Sprintf(" (param: %s)", e.Param())
		}
		return msg
	}
}

// validateSagaConfig validates the Saga configuration.
func validateSagaConfig(def *SagaDefinition) ValidationErrors {
	var errors ValidationErrors

	// Validate Saga ID format
	if def.Saga.ID == "" {
		errors = append(errors, &ValidationError{
			Field:   "saga.id",
			Message: "saga ID is required",
			Rule:    "required",
		})
	} else if !sagaIDPattern.MatchString(def.Saga.ID) {
		errors = append(errors, &ValidationError{
			Field:   "saga.id",
			Message: "saga ID must contain only alphanumeric characters, dashes, and underscores, and must start and end with alphanumeric characters",
			Value:   def.Saga.ID,
			Rule:    "pattern",
		})
	}

	// Validate Saga name
	if def.Saga.Name == "" {
		errors = append(errors, &ValidationError{
			Field:   "saga.name",
			Message: "saga name is required",
			Rule:    "required",
		})
	} else if len(def.Saga.Name) > 100 {
		errors = append(errors, &ValidationError{
			Field:   "saga.name",
			Message: "saga name must not exceed 100 characters",
			Value:   def.Saga.Name,
			Rule:    "max_length",
		})
	}

	// Validate execution mode
	if def.Saga.Mode != "" {
		validModes := []ExecutionMode{
			ExecutionModeOrchestration,
			ExecutionModeChoreography,
			ExecutionModeHybrid,
		}
		valid := false
		for _, mode := range validModes {
			if def.Saga.Mode == mode {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, &ValidationError{
				Field:   "saga.mode",
				Message: "invalid execution mode",
				Value:   string(def.Saga.Mode),
				Rule:    "oneof",
			})
		}
	}

	return errors
}

// validateSteps validates step configurations.
func validateSteps(def *SagaDefinition) ValidationErrors {
	var errors ValidationErrors

	// At least one step is required
	if len(def.Steps) == 0 {
		errors = append(errors, &ValidationError{
			Field:   "steps",
			Message: "at least one step is required",
			Rule:    "min",
		})
		return errors
	}

	// Validate each step
	stepIDs := make(map[string]bool)
	for i, step := range def.Steps {
		fieldPrefix := fmt.Sprintf("steps[%d]", i)

		// Validate step ID uniqueness
		if step.ID == "" {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.id", fieldPrefix),
				Message: "step ID is required",
				Rule:    "required",
			})
		} else {
			if !stepIDPattern.MatchString(step.ID) {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("%s.id", fieldPrefix),
					Message: "step ID must contain only alphanumeric characters, dashes, and underscores",
					Value:   step.ID,
					Rule:    "pattern",
				})
			}

			if stepIDs[step.ID] {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("%s.id", fieldPrefix),
					Message: "duplicate step ID",
					Value:   step.ID,
					Rule:    "unique",
				})
			}
			stepIDs[step.ID] = true
		}

		// Validate step name
		if step.Name == "" {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.name", fieldPrefix),
				Message: "step name is required",
				Rule:    "required",
			})
		}

		// Validate step type
		validTypes := []StepType{
			StepTypeService,
			StepTypeFunction,
			StepTypeHTTP,
			StepTypeGRPC,
			StepTypeMessage,
			StepTypeCustom,
		}
		valid := false
		for _, t := range validTypes {
			if step.Type == t {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.type", fieldPrefix),
				Message: "invalid step type",
				Value:   string(step.Type),
				Rule:    "oneof",
			})
		}

		// Validate action configuration based on step type
		errors = append(errors, validateStepAction(&step, fieldPrefix)...)
	}

	return errors
}

// validateStepAction validates action configuration based on step type.
func validateStepAction(step *StepConfig, fieldPrefix string) ValidationErrors {
	var errors ValidationErrors

	switch step.Type {
	case StepTypeService, StepTypeHTTP, StepTypeGRPC:
		if step.Action.Service == nil {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.action.service", fieldPrefix),
				Message: fmt.Sprintf("service action is required for step type '%s'", step.Type),
				Rule:    "required",
			})
		} else {
			errors = append(errors, validateServiceConfig(step.Action.Service, fmt.Sprintf("%s.action.service", fieldPrefix))...)
		}

	case StepTypeFunction:
		if step.Action.Function == nil {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.action.function", fieldPrefix),
				Message: "function action is required for step type 'function'",
				Rule:    "required",
			})
		} else {
			errors = append(errors, validateFunctionConfig(step.Action.Function, fmt.Sprintf("%s.action.function", fieldPrefix))...)
		}

	case StepTypeMessage:
		if step.Action.Message == nil {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.action.message", fieldPrefix),
				Message: "message action is required for step type 'message'",
				Rule:    "required",
			})
		} else {
			errors = append(errors, validateMessageConfig(step.Action.Message, fmt.Sprintf("%s.action.message", fieldPrefix))...)
		}

	case StepTypeCustom:
		if len(step.Action.Custom) == 0 {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.action.custom", fieldPrefix),
				Message: "custom action is required for step type 'custom'",
				Rule:    "required",
			})
		}
	}

	return errors
}

// validateServiceConfig validates service action configuration.
func validateServiceConfig(svc *ServiceConfig, fieldPrefix string) ValidationErrors {
	var errors ValidationErrors

	if svc.Name == "" {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.name", fieldPrefix),
			Message: "service name is required",
			Rule:    "required",
		})
	} else if !serviceNamePattern.MatchString(svc.Name) {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.name", fieldPrefix),
			Message: "service name must contain only lowercase alphanumeric characters and dashes",
			Value:   svc.Name,
			Rule:    "pattern",
		})
	}

	if svc.Method == "" {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.method", fieldPrefix),
			Message: "service method is required",
			Rule:    "required",
		})
	}

	return errors
}

// validateFunctionConfig validates function action configuration.
func validateFunctionConfig(fn *FunctionConfig, fieldPrefix string) ValidationErrors {
	var errors ValidationErrors

	if fn.Name == "" {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.name", fieldPrefix),
			Message: "function name is required",
			Rule:    "required",
		})
	}

	if fn.Handler == "" {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.handler", fieldPrefix),
			Message: "function handler is required",
			Rule:    "required",
		})
	}

	return errors
}

// validateMessageConfig validates message action configuration.
func validateMessageConfig(msg *MessageConfig, fieldPrefix string) ValidationErrors {
	var errors ValidationErrors

	if msg.Topic == "" {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.topic", fieldPrefix),
			Message: "message topic is required",
			Rule:    "required",
		})
	}

	return errors
}

// validateStepDependencies validates step dependency relationships.
func validateStepDependencies(def *SagaDefinition) ValidationErrors {
	var errors ValidationErrors

	// Build step ID map
	stepIDs := make(map[string]bool)
	for _, step := range def.Steps {
		stepIDs[step.ID] = true
	}

	// Validate dependencies reference existing steps
	for i, step := range def.Steps {
		for _, depID := range step.Dependencies {
			if !stepIDs[depID] {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("steps[%d].dependencies", i),
					Message: fmt.Sprintf("step depends on non-existent step '%s'", depID),
					Value:   depID,
					Rule:    "exists",
				})
			}
		}
	}

	// Validate no circular dependencies
	if err := checkCircularDependencies(def); err != nil {
		errors = append(errors, err)
	}

	return errors
}

// checkCircularDependencies detects circular dependencies using DFS.
func checkCircularDependencies(def *SagaDefinition) *ValidationError {
	// Build dependency graph
	graph := make(map[string][]string)
	for _, step := range def.Steps {
		graph[step.ID] = step.Dependencies
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) (bool, []string)
	hasCycle = func(stepID string) (bool, []string) {
		visited[stepID] = true
		recStack[stepID] = true

		for _, dep := range graph[stepID] {
			if !visited[dep] {
				if cycle, path := hasCycle(dep); cycle {
					return true, append([]string{stepID}, path...)
				}
			} else if recStack[dep] {
				return true, []string{stepID, dep}
			}
		}

		recStack[stepID] = false
		return false, nil
	}

	for stepID := range graph {
		if !visited[stepID] {
			if cycle, path := hasCycle(stepID); cycle {
				return &ValidationError{
					Field:   "steps.dependencies",
					Message: fmt.Sprintf("circular dependency detected: %s", strings.Join(path, " -> ")),
					Rule:    "no_cycles",
				}
			}
		}
	}

	return nil
}

// validateTimeouts validates timeout configurations.
func validateTimeouts(def *SagaDefinition) ValidationErrors {
	var errors ValidationErrors

	// Validate Saga timeout
	if def.Saga.Timeout > 0 {
		if time.Duration(def.Saga.Timeout) < 1*time.Second {
			errors = append(errors, &ValidationError{
				Field:   "saga.timeout",
				Message: "saga timeout must be at least 1 second",
				Value:   def.Saga.Timeout.String(),
				Rule:    "min",
			})
		}

		if time.Duration(def.Saga.Timeout) > 24*time.Hour {
			errors = append(errors, &ValidationError{
				Field:   "saga.timeout",
				Message: "saga timeout must not exceed 24 hours",
				Value:   def.Saga.Timeout.String(),
				Rule:    "max",
			})
		}
	}

	// Validate step timeouts
	for i, step := range def.Steps {
		if step.Timeout > 0 {
			if time.Duration(step.Timeout) < 1*time.Second {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("steps[%d].timeout", i),
					Message: "step timeout must be at least 1 second",
					Value:   step.Timeout.String(),
					Rule:    "min",
				})
			}

			// Step timeout should not exceed Saga timeout
			if def.Saga.Timeout > 0 && step.Timeout > def.Saga.Timeout {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("steps[%d].timeout", i),
					Message: fmt.Sprintf("step timeout (%s) exceeds saga timeout (%s)", step.Timeout.String(), def.Saga.Timeout.String()),
					Value:   step.Timeout.String(),
					Rule:    "max",
				})
			}
		}

		// Validate service action timeout
		if step.Action.Service != nil && step.Action.Service.Timeout > 0 {
			if time.Duration(step.Action.Service.Timeout) < 100*time.Millisecond {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("steps[%d].action.service.timeout", i),
					Message: "service timeout must be at least 100 milliseconds",
					Value:   step.Action.Service.Timeout.String(),
					Rule:    "min",
				})
			}

			// Service timeout should not exceed step timeout
			if step.Timeout > 0 && step.Action.Service.Timeout > step.Timeout {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("steps[%d].action.service.timeout", i),
					Message: fmt.Sprintf("service timeout (%s) exceeds step timeout (%s)", step.Action.Service.Timeout.String(), step.Timeout.String()),
					Value:   step.Action.Service.Timeout.String(),
					Rule:    "max",
				})
			}
		}
	}

	// Validate global compensation timeout
	if def.GlobalCompensation != nil && def.GlobalCompensation.Timeout > 0 {
		if time.Duration(def.GlobalCompensation.Timeout) < 1*time.Second {
			errors = append(errors, &ValidationError{
				Field:   "global_compensation.timeout",
				Message: "global compensation timeout must be at least 1 second",
				Value:   def.GlobalCompensation.Timeout.String(),
				Rule:    "min",
			})
		}
	}

	return errors
}

// validateRetryPolicies validates retry policy configurations.
func validateRetryPolicies(def *SagaDefinition) ValidationErrors {
	var errors ValidationErrors

	// Validate global retry policy
	if def.GlobalRetryPolicy != nil {
		errors = append(errors, validateRetryPolicy(def.GlobalRetryPolicy, "global_retry_policy")...)
	}

	// Validate step retry policies
	for i, step := range def.Steps {
		if step.RetryPolicy != nil {
			errors = append(errors, validateRetryPolicy(step.RetryPolicy, fmt.Sprintf("steps[%d].retry_policy", i))...)
		}
	}

	return errors
}

// validateRetryPolicy validates a single retry policy.
func validateRetryPolicy(policy *RetryPolicy, fieldPrefix string) ValidationErrors {
	var errors ValidationErrors

	// Validate retry type
	validTypes := []string{"fixed_delay", "exponential_backoff", "linear_backoff", "no_retry"}
	valid := false
	for _, t := range validTypes {
		if policy.Type == t {
			valid = true
			break
		}
	}
	if !valid {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.type", fieldPrefix),
			Message: "invalid retry type",
			Value:   policy.Type,
			Rule:    "oneof",
		})
	}

	// Validate max attempts
	if policy.MaxAttempts < 1 {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.max_attempts", fieldPrefix),
			Message: "max attempts must be at least 1",
			Value:   fmt.Sprintf("%d", policy.MaxAttempts),
			Rule:    "min",
		})
	} else if policy.MaxAttempts > 100 {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.max_attempts", fieldPrefix),
			Message: "max attempts must not exceed 100",
			Value:   fmt.Sprintf("%d", policy.MaxAttempts),
			Rule:    "max",
		})
	}

	// Validate delays for specific retry types
	switch policy.Type {
	case "exponential_backoff":
		if policy.InitialDelay > 0 && time.Duration(policy.InitialDelay) < 10*time.Millisecond {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.initial_delay", fieldPrefix),
				Message: "initial delay must be at least 10 milliseconds",
				Value:   policy.InitialDelay.String(),
				Rule:    "min",
			})
		}

		if policy.Multiplier > 0 && policy.Multiplier < 1 {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.multiplier", fieldPrefix),
				Message: "multiplier must be at least 1",
				Value:   fmt.Sprintf("%.2f", policy.Multiplier),
				Rule:    "min",
			})
		} else if policy.Multiplier > 10 {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.multiplier", fieldPrefix),
				Message: "multiplier must not exceed 10",
				Value:   fmt.Sprintf("%.2f", policy.Multiplier),
				Rule:    "max",
			})
		}

		// Validate max_delay > initial_delay
		if policy.MaxDelay > 0 && policy.InitialDelay > 0 && policy.MaxDelay < policy.InitialDelay {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.max_delay", fieldPrefix),
				Message: fmt.Sprintf("max delay (%s) must be greater than initial delay (%s)", policy.MaxDelay.String(), policy.InitialDelay.String()),
				Value:   policy.MaxDelay.String(),
				Rule:    "gt",
			})
		}

	case "linear_backoff":
		if policy.InitialDelay > 0 && time.Duration(policy.InitialDelay) < 10*time.Millisecond {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.initial_delay", fieldPrefix),
				Message: "initial delay must be at least 10 milliseconds",
				Value:   policy.InitialDelay.String(),
				Rule:    "min",
			})
		}

		if policy.Increment > 0 && time.Duration(policy.Increment) < 10*time.Millisecond {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.increment", fieldPrefix),
				Message: "increment must be at least 10 milliseconds",
				Value:   policy.Increment.String(),
				Rule:    "min",
			})
		}

	case "fixed_delay":
		if policy.InitialDelay > 0 && time.Duration(policy.InitialDelay) < 10*time.Millisecond {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.initial_delay", fieldPrefix),
				Message: "initial delay must be at least 10 milliseconds",
				Value:   policy.InitialDelay.String(),
				Rule:    "min",
			})
		}
	}

	return errors
}

// validateCompensations validates compensation configurations.
func validateCompensations(def *SagaDefinition) ValidationErrors {
	var errors ValidationErrors

	// Validate global compensation
	if def.GlobalCompensation != nil {
		errors = append(errors, validateGlobalCompensation(def.GlobalCompensation)...)
	}

	// Validate step compensations
	for i, step := range def.Steps {
		if step.Compensation != nil {
			errors = append(errors, validateCompensationAction(step.Compensation, fmt.Sprintf("steps[%d].compensation", i))...)
		}
	}

	return errors
}

// validateGlobalCompensation validates global compensation configuration.
func validateGlobalCompensation(comp *CompensationConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate strategy
	validStrategies := []CompensationStrategy{
		CompensationStrategySequential,
		CompensationStrategyParallel,
		CompensationStrategyBestEffort,
	}
	valid := false
	for _, s := range validStrategies {
		if comp.Strategy == s {
			valid = true
			break
		}
	}
	if !valid {
		errors = append(errors, &ValidationError{
			Field:   "global_compensation.strategy",
			Message: "invalid compensation strategy",
			Value:   string(comp.Strategy),
			Rule:    "oneof",
		})
	}

	// Validate max attempts
	if comp.MaxAttempts > 0 && comp.MaxAttempts < 1 {
		errors = append(errors, &ValidationError{
			Field:   "global_compensation.max_attempts",
			Message: "max attempts must be at least 1",
			Value:   fmt.Sprintf("%d", comp.MaxAttempts),
			Rule:    "min",
		})
	}

	return errors
}

// validateCompensationAction validates compensation action configuration.
func validateCompensationAction(comp *CompensationAction, fieldPrefix string) ValidationErrors {
	var errors ValidationErrors

	// Validate compensation type
	validTypes := []CompensationType{
		CompensationTypeAutomatic,
		CompensationTypeCustom,
		CompensationTypeSkip,
	}
	valid := false
	for _, t := range validTypes {
		if comp.Type == t {
			valid = true
			break
		}
	}
	if !valid {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.type", fieldPrefix),
			Message: "invalid compensation type",
			Value:   string(comp.Type),
			Rule:    "oneof",
		})
	}

	// Custom compensation requires an action
	if comp.Type == CompensationTypeCustom && comp.Action == nil {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.action", fieldPrefix),
			Message: "custom compensation requires an action configuration",
			Rule:    "required",
		})
	}

	// Skip compensation should not have an action
	if comp.Type == CompensationTypeSkip && comp.Action != nil {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.action", fieldPrefix),
			Message: "skip compensation should not have an action configuration",
			Rule:    "conflict",
		})
	}

	// Validate compensation strategy
	if comp.Strategy != "" {
		validStrategies := []CompensationStrategy{
			CompensationStrategySequential,
			CompensationStrategyParallel,
			CompensationStrategyBestEffort,
		}
		valid := false
		for _, s := range validStrategies {
			if comp.Strategy == s {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.strategy", fieldPrefix),
				Message: "invalid compensation strategy",
				Value:   string(comp.Strategy),
				Rule:    "oneof",
			})
		}
	}

	// Validate max attempts
	if comp.MaxAttempts > 0 && comp.MaxAttempts < 1 {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.max_attempts", fieldPrefix),
			Message: "max attempts must be at least 1",
			Value:   fmt.Sprintf("%d", comp.MaxAttempts),
			Rule:    "min",
		})
	}

	// Validate timeout
	if comp.Timeout > 0 && time.Duration(comp.Timeout) < 100*time.Millisecond {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.timeout", fieldPrefix),
			Message: "compensation timeout must be at least 100 milliseconds",
			Value:   comp.Timeout.String(),
			Rule:    "min",
		})
	}

	// Validate on_failure policy
	if comp.OnFailure != nil {
		validActions := []string{"ignore", "retry", "fail", "alert"}
		valid := false
		for _, a := range validActions {
			if comp.OnFailure.Action == a {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.on_failure.action", fieldPrefix),
				Message: "invalid on_failure action",
				Value:   comp.OnFailure.Action,
				Rule:    "oneof",
			})
		}
	}

	return errors
}
