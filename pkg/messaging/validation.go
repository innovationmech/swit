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

package messaging

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigValidator provides validation functionality for messaging configurations
type ConfigValidator struct {
	profiles map[string]ConfigProfile
}

// ConfigProfile represents a configuration profile (dev, staging, prod)
type ConfigProfile struct {
	Name      string                 `yaml:"name" json:"name"`
	Overrides map[string]interface{} `yaml:"overrides,omitempty" json:"overrides,omitempty"`
	Inherits  string                 `yaml:"inherits,omitempty" json:"inherits,omitempty"`
}

// ValidationContext provides context for validation operations
type ValidationContext struct {
	Profile     string
	Environment map[string]string
	Strict      bool
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		profiles: make(map[string]ConfigProfile),
	}
}

// LoadProfiles loads configuration profiles from YAML data
func (v *ConfigValidator) LoadProfiles(data []byte) error {
	var profiles map[string]ConfigProfile
	if err := yaml.Unmarshal(data, &profiles); err != nil {
		return fmt.Errorf("failed to unmarshal profiles: %w", err)
	}

	for name, profile := range profiles {
		profile.Name = name
		v.profiles[name] = profile
	}

	return nil
}

// LoadProfilesFromFile loads configuration profiles from a YAML file
func (v *ConfigValidator) LoadProfilesFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read profiles file %s: %w", filename, err)
	}

	return v.LoadProfiles(data)
}

// AddProfile adds a configuration profile
func (v *ConfigValidator) AddProfile(profile ConfigProfile) {
	v.profiles[profile.Name] = profile
}

// GetProfile retrieves a configuration profile by name
func (v *ConfigValidator) GetProfile(name string) (ConfigProfile, bool) {
	profile, exists := v.profiles[name]
	return profile, exists
}

// ApplyEnvironmentOverrides applies environment variable overrides to a configuration
func ApplyEnvironmentOverrides(config interface{}, prefix string) error {
	return applyEnvironmentOverridesToValue(reflect.ValueOf(config), prefix, "")
}

// applyEnvironmentOverridesToValue recursively applies environment overrides
func applyEnvironmentOverridesToValue(value reflect.Value, prefix, path string) error {
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		return applyEnvironmentOverridesToStruct(value, prefix, path)
	case reflect.Map:
		return applyEnvironmentOverridesToMap(value, prefix, path)
	case reflect.Slice:
		return applyEnvironmentOverridesToSlice(value, prefix, path)
	}

	return nil
}

// applyEnvironmentOverridesToStruct applies environment overrides to struct fields
func applyEnvironmentOverridesToStruct(value reflect.Value, prefix, path string) error {
	structType := value.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := value.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		// Get the field name for environment variable lookup
		fieldName := field.Name
		if yamlTag := field.Tag.Get("yaml"); yamlTag != "" && yamlTag != "-" {
			tagParts := strings.Split(yamlTag, ",")
			if tagParts[0] != "" {
				fieldName = tagParts[0]
			}
		}

		// Build the full path
		fullPath := fieldName
		if path != "" {
			fullPath = path + "_" + fieldName
		}

		// Build environment variable name (handle nested fields properly)
		envVar := strings.ToUpper(prefix + "_" + strings.Replace(fullPath, ".", "_", -1))

		if envValue, exists := os.LookupEnv(envVar); exists {
			if err := setFieldFromEnvValue(fieldValue, envValue, envVar); err != nil {
				return err
			}
			continue
		}

		// Recursively process nested structures
		if err := applyEnvironmentOverridesToValue(fieldValue, prefix, fullPath); err != nil {
			return err
		}
	}

	return nil
}

// applyEnvironmentOverridesToMap applies environment overrides to map values
func applyEnvironmentOverridesToMap(value reflect.Value, prefix, path string) error {
	if value.IsNil() {
		return nil
	}

	for _, key := range value.MapKeys() {
		mapValue := value.MapIndex(key)
		keyStr := fmt.Sprintf("%v", key.Interface())

		fullPath := keyStr
		if path != "" {
			fullPath = path + "_" + keyStr
		}

		if err := applyEnvironmentOverridesToValue(mapValue, prefix, fullPath); err != nil {
			return err
		}
	}

	return nil
}

// applyEnvironmentOverridesToSlice applies environment overrides to slice elements
func applyEnvironmentOverridesToSlice(value reflect.Value, prefix, path string) error {
	for i := 0; i < value.Len(); i++ {
		element := value.Index(i)
		fullPath := fmt.Sprintf("%s_%d", path, i)

		if err := applyEnvironmentOverridesToValue(element, prefix, fullPath); err != nil {
			return err
		}
	}

	return nil
}

// setFieldFromEnvValue sets a struct field value from an environment variable string
func setFieldFromEnvValue(fieldValue reflect.Value, envValue, envVar string) error {
	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(envValue)
	case reflect.Bool:
		if val, err := strconv.ParseBool(envValue); err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", envVar, envValue)
		} else {
			fieldValue.SetBool(val)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if fieldValue.Type() == reflect.TypeOf(time.Duration(0)) {
			if val, err := time.ParseDuration(envValue); err != nil {
				return fmt.Errorf("invalid duration value for %s: %s", envVar, envValue)
			} else {
				fieldValue.SetInt(int64(val))
			}
		} else {
			if val, err := strconv.ParseInt(envValue, 10, 64); err != nil {
				return fmt.Errorf("invalid integer value for %s: %s", envVar, envValue)
			} else {
				fieldValue.SetInt(val)
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val, err := strconv.ParseUint(envValue, 10, 64); err != nil {
			return fmt.Errorf("invalid unsigned integer value for %s: %s", envVar, envValue)
		} else {
			fieldValue.SetUint(val)
		}
	case reflect.Float32, reflect.Float64:
		if val, err := strconv.ParseFloat(envValue, 64); err != nil {
			return fmt.Errorf("invalid float value for %s: %s", envVar, envValue)
		} else {
			fieldValue.SetFloat(val)
		}
	case reflect.Slice:
		if fieldValue.Type().Elem().Kind() == reflect.String {
			values := strings.Split(envValue, ",")
			for i, val := range values {
				values[i] = strings.TrimSpace(val)
			}
			fieldValue.Set(reflect.ValueOf(values))
		}
	default:
		return fmt.Errorf("unsupported field type %s for environment variable %s", fieldValue.Type(), envVar)
	}

	return nil
}

// ValidateWithProfile validates configuration using a specific profile
func (v *ConfigValidator) ValidateWithProfile(config interface{}, profileName string, ctx ValidationContext) error {
	// First apply base validation
	if validator, ok := config.(interface{ Validate() error }); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("base validation failed: %w", err)
		}
	}

	// Apply profile-specific validation if profile exists
	if profileName != "" {
		if err := v.applyProfileValidation(config, profileName, ctx); err != nil {
			return fmt.Errorf("profile validation failed: %w", err)
		}
	}

	return nil
}

// applyProfileValidation applies profile-specific validation rules
func (v *ConfigValidator) applyProfileValidation(config interface{}, profileName string, ctx ValidationContext) error {
	profile, exists := v.profiles[profileName]
	if !exists {
		if ctx.Strict {
			return fmt.Errorf("profile %s not found", profileName)
		}
		return nil
	}

	// Apply inherited profile validation first
	if profile.Inherits != "" {
		if err := v.applyProfileValidation(config, profile.Inherits, ctx); err != nil {
			return err
		}
	}

	// Apply profile-specific overrides and validation
	return v.applyProfileOverrides(config, profile, ctx)
}

// applyProfileOverrides applies configuration overrides from a profile
func (v *ConfigValidator) applyProfileOverrides(config interface{}, profile ConfigProfile, ctx ValidationContext) error {
	configValue := reflect.ValueOf(config)
	if configValue.Kind() == reflect.Ptr {
		configValue = configValue.Elem()
	}

	for key, value := range profile.Overrides {
		if err := v.setConfigValue(configValue, key, value); err != nil {
			return fmt.Errorf("failed to apply override %s: %w", key, err)
		}
	}

	return nil
}

// setConfigValue sets a configuration value using dot notation
func (v *ConfigValidator) setConfigValue(configValue reflect.Value, key string, value interface{}) error {
	keys := strings.Split(key, ".")
	current := configValue

	// Navigate to the target field
	for i, k := range keys[:len(keys)-1] {
		field := current.FieldByName(strings.Title(k))
		if !field.IsValid() {
			return fmt.Errorf("field %s not found in path %s", k, strings.Join(keys[:i+1], "."))
		}

		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				// Create new instance if nil
				field.Set(reflect.New(field.Type().Elem()))
			}
			field = field.Elem()
		}
		current = field
	}

	// Set the final value
	finalKey := keys[len(keys)-1]
	field := current.FieldByName(strings.Title(finalKey))
	if !field.IsValid() {
		return fmt.Errorf("field %s not found", finalKey)
	}

	if !field.CanSet() {
		return fmt.Errorf("field %s cannot be set", finalKey)
	}

	// Convert and set the value
	return v.setReflectValue(field, value)
}

// setReflectValue sets a reflect.Value with type conversion
func (v *ConfigValidator) setReflectValue(field reflect.Value, value interface{}) error {
	valueReflect := reflect.ValueOf(value)

	if valueReflect.Type().ConvertibleTo(field.Type()) {
		field.Set(valueReflect.Convert(field.Type()))
		return nil
	}

	// Handle special cases
	switch field.Kind() {
	case reflect.String:
		field.SetString(fmt.Sprintf("%v", value))
	case reflect.Bool:
		if b, ok := value.(bool); ok {
			field.SetBool(b)
		} else {
			return fmt.Errorf("cannot convert %v to bool", value)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			if d, err := time.ParseDuration(fmt.Sprintf("%v", value)); err != nil {
				return fmt.Errorf("cannot convert %v to duration: %w", value, err)
			} else {
				field.SetInt(int64(d))
			}
		} else {
			if i, err := strconv.ParseInt(fmt.Sprintf("%v", value), 10, 64); err != nil {
				return fmt.Errorf("cannot convert %v to int: %w", value, err)
			} else {
				field.SetInt(i)
			}
		}
	case reflect.Float32, reflect.Float64:
		if f, err := strconv.ParseFloat(fmt.Sprintf("%v", value), 64); err != nil {
			return fmt.Errorf("cannot convert %v to float: %w", value, err)
		} else {
			field.SetFloat(f)
		}
	default:
		return fmt.Errorf("unsupported field type %s", field.Type())
	}

	return nil
}

// GetBuiltinProfiles returns built-in configuration profiles for common environments
func GetBuiltinProfiles() map[string]ConfigProfile {
	return map[string]ConfigProfile{
		"development": {
			Name: "development",
			Overrides: map[string]interface{}{
				"Connection.Timeout":         "5s",
				"Connection.KeepAlive":       "15s",
				"Connection.MaxAttempts":     1,
				"Retry.MaxAttempts":          1,
				"Monitoring.Enabled":         true,
				"Monitoring.MetricsInterval": "10s",
			},
		},
		"staging": {
			Name:     "staging",
			Inherits: "development",
			Overrides: map[string]interface{}{
				"Connection.Timeout":         "10s",
				"Connection.KeepAlive":       "30s",
				"Connection.MaxAttempts":     2,
				"Retry.MaxAttempts":          2,
				"Monitoring.MetricsInterval": "30s",
			},
		},
		"production": {
			Name: "production",
			Overrides: map[string]interface{}{
				"Connection.Timeout":         "30s",
				"Connection.KeepAlive":       "60s",
				"Connection.MaxAttempts":     3,
				"Connection.PoolSize":        20,
				"Retry.MaxAttempts":          3,
				"Retry.InitialDelay":         "1s",
				"Retry.MaxDelay":             "60s",
				"Monitoring.Enabled":         true,
				"Monitoring.MetricsInterval": "60s",
			},
		},
	}
}

// ValidateStruct performs comprehensive struct validation using reflection and tags
func ValidateStruct(s interface{}) error {
	return validateStructValue(reflect.ValueOf(s), "")
}

// validateStructValue recursively validates struct fields
func validateStructValue(v reflect.Value, path string) error {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		fieldPath := field.Name
		if path != "" {
			fieldPath = path + "." + field.Name
		}

		// Check validation tags
		if err := validateFieldTags(fieldValue, field, fieldPath); err != nil {
			return err
		}

		// Recursively validate nested structs
		if err := validateStructValue(fieldValue, fieldPath); err != nil {
			return err
		}
	}

	return nil
}

// validateFieldTags validates field based on struct tags
func validateFieldTags(fieldValue reflect.Value, field reflect.StructField, path string) error {
	validateTag := field.Tag.Get("validate")
	if validateTag == "" {
		return nil
	}

	rules := strings.Split(validateTag, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if err := applyValidationRule(fieldValue, rule, path); err != nil {
			return err
		}
	}

	return nil
}

// applyValidationRule applies a single validation rule
func applyValidationRule(fieldValue reflect.Value, rule, path string) error {
	parts := strings.SplitN(rule, "=", 2)
	ruleName := parts[0]

	switch ruleName {
	case "required":
		return validateRequired(fieldValue, path)
	case "min":
		if len(parts) != 2 {
			return fmt.Errorf("min rule requires a value")
		}
		return validateMin(fieldValue, parts[1], path)
	case "max":
		if len(parts) != 2 {
			return fmt.Errorf("max rule requires a value")
		}
		return validateMax(fieldValue, parts[1], path)
	case "oneof":
		if len(parts) != 2 {
			return fmt.Errorf("oneof rule requires values")
		}
		return validateOneOf(fieldValue, parts[1], path)
	default:
		return fmt.Errorf("unknown validation rule: %s", ruleName)
	}
}

// validateRequired checks if a required field has a value
func validateRequired(fieldValue reflect.Value, path string) error {
	switch fieldValue.Kind() {
	case reflect.String:
		if fieldValue.String() == "" {
			return NewConfigError(fmt.Sprintf("field %s is required", path), nil)
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if fieldValue.Len() == 0 {
			return NewConfigError(fmt.Sprintf("field %s is required and cannot be empty", path), nil)
		}
	case reflect.Ptr, reflect.Interface:
		if fieldValue.IsNil() {
			return NewConfigError(fmt.Sprintf("field %s is required", path), nil)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if fieldValue.Int() == 0 {
			return NewConfigError(fmt.Sprintf("field %s is required", path), nil)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if fieldValue.Uint() == 0 {
			return NewConfigError(fmt.Sprintf("field %s is required", path), nil)
		}
	case reflect.Float32, reflect.Float64:
		if fieldValue.Float() == 0 {
			return NewConfigError(fmt.Sprintf("field %s is required", path), nil)
		}
	case reflect.Bool:
		// Boolean false is considered a valid value
	}

	return nil
}

// validateMin validates minimum value/length constraints
func validateMin(fieldValue reflect.Value, minStr, path string) error {
	switch fieldValue.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		min, err := strconv.Atoi(minStr)
		if err != nil {
			return fmt.Errorf("invalid min value: %s", minStr)
		}
		if fieldValue.Len() < min {
			return NewConfigError(fmt.Sprintf("field %s must have at least %d elements", path, min), nil)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		min, err := strconv.ParseInt(minStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid min value: %s", minStr)
		}
		if fieldValue.Int() < min {
			return NewConfigError(fmt.Sprintf("field %s must be at least %d", path, min), nil)
		}
	case reflect.Float32, reflect.Float64:
		min, err := strconv.ParseFloat(minStr, 64)
		if err != nil {
			return fmt.Errorf("invalid min value: %s", minStr)
		}
		if fieldValue.Float() < min {
			return NewConfigError(fmt.Sprintf("field %s must be at least %f", path, min), nil)
		}
	}

	return nil
}

// validateMax validates maximum value/length constraints
func validateMax(fieldValue reflect.Value, maxStr, path string) error {
	switch fieldValue.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		max, err := strconv.Atoi(maxStr)
		if err != nil {
			return fmt.Errorf("invalid max value: %s", maxStr)
		}
		if fieldValue.Len() > max {
			return NewConfigError(fmt.Sprintf("field %s must have at most %d elements", path, max), nil)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		max, err := strconv.ParseInt(maxStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid max value: %s", maxStr)
		}
		if fieldValue.Int() > max {
			return NewConfigError(fmt.Sprintf("field %s must be at most %d", path, max), nil)
		}
	case reflect.Float32, reflect.Float64:
		max, err := strconv.ParseFloat(maxStr, 64)
		if err != nil {
			return fmt.Errorf("invalid max value: %s", maxStr)
		}
		if fieldValue.Float() > max {
			return NewConfigError(fmt.Sprintf("field %s must be at most %f", path, max), nil)
		}
	}

	return nil
}

// validateOneOf validates that the field value is one of the allowed values
func validateOneOf(fieldValue reflect.Value, allowedStr, path string) error {
	allowed := strings.Split(allowedStr, " ")
	value := fmt.Sprintf("%v", fieldValue.Interface())

	for _, a := range allowed {
		if value == a {
			return nil
		}
	}

	return NewConfigError(fmt.Sprintf("field %s must be one of: %s", path, allowedStr), nil)
}
