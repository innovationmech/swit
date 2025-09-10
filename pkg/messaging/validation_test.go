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
	"os"
	"testing"
	"time"
)

func TestNewConfigValidator(t *testing.T) {
	validator := NewConfigValidator()
	if validator == nil {
		t.Fatal("expected validator to be created")
	}
	if validator.profiles == nil {
		t.Fatal("expected profiles map to be initialized")
	}
}

func TestConfigValidator_LoadProfiles(t *testing.T) {
	tests := []struct {
		name      string
		yamlData  string
		expectErr bool
		checkFunc func(*testing.T, *ConfigValidator)
	}{
		{
			name: "valid profiles",
			yamlData: `
development:
  name: development
  overrides:
    Connection.Timeout: "5s"
    Retry.MaxAttempts: 1
production:
  name: production
  inherits: development
  overrides:
    Connection.Timeout: "30s"
    Retry.MaxAttempts: 3
`,
			expectErr: false,
			checkFunc: func(t *testing.T, v *ConfigValidator) {
				if len(v.profiles) != 2 {
					t.Errorf("expected 2 profiles, got %d", len(v.profiles))
				}

				dev, exists := v.GetProfile("development")
				if !exists {
					t.Fatal("development profile not found")
				}
				if dev.Name != "development" {
					t.Errorf("expected name 'development', got %s", dev.Name)
				}

				prod, exists := v.GetProfile("production")
				if !exists {
					t.Fatal("production profile not found")
				}
				if prod.Inherits != "development" {
					t.Errorf("expected inherits 'development', got %s", prod.Inherits)
				}
			},
		},
		{
			name:      "invalid yaml",
			yamlData:  "invalid: yaml: data:",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConfigValidator()
			err := validator.LoadProfiles([]byte(tt.yamlData))

			if tt.expectErr && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, validator)
			}
		})
	}
}

func TestApplyEnvironmentOverrides(t *testing.T) {
	// Test struct for environment overrides
	type TestConfig struct {
		StringField   string        `yaml:"string_field"`
		IntField      int           `yaml:"int_field"`
		BoolField     bool          `yaml:"bool_field"`
		DurationField time.Duration `yaml:"duration_field"`
		NestedConfig  struct {
			NestedString string `yaml:"nested_string"`
			NestedInt    int    `yaml:"nested_int"`
		} `yaml:"nested_config"`
		SliceField []string `yaml:"slice_field"`
	}

	tests := []struct {
		name      string
		envVars   map[string]string
		prefix    string
		expectErr bool
		checkFunc func(*testing.T, *TestConfig)
	}{
		{
			name: "string field override",
			envVars: map[string]string{
				"TEST_STRING_FIELD": "overridden_value",
			},
			prefix:    "TEST",
			expectErr: false,
			checkFunc: func(t *testing.T, config *TestConfig) {
				if config.StringField != "overridden_value" {
					t.Errorf("expected 'overridden_value', got %s", config.StringField)
				}
			},
		},
		{
			name: "int field override",
			envVars: map[string]string{
				"TEST_INT_FIELD": "42",
			},
			prefix:    "TEST",
			expectErr: false,
			checkFunc: func(t *testing.T, config *TestConfig) {
				if config.IntField != 42 {
					t.Errorf("expected 42, got %d", config.IntField)
				}
			},
		},
		{
			name: "bool field override",
			envVars: map[string]string{
				"TEST_BOOL_FIELD": "true",
			},
			prefix:    "TEST",
			expectErr: false,
			checkFunc: func(t *testing.T, config *TestConfig) {
				if !config.BoolField {
					t.Error("expected true, got false")
				}
			},
		},
		{
			name: "duration field override",
			envVars: map[string]string{
				"TEST_DURATION_FIELD": "30s",
			},
			prefix:    "TEST",
			expectErr: false,
			checkFunc: func(t *testing.T, config *TestConfig) {
				if config.DurationField != 30*time.Second {
					t.Errorf("expected 30s, got %v", config.DurationField)
				}
			},
		},
		{
			name: "nested field override",
			envVars: map[string]string{
				"TEST_NESTED_CONFIG_NESTED_STRING": "nested_override",
				"TEST_NESTED_CONFIG_NESTED_INT":    "99",
			},
			prefix:    "TEST",
			expectErr: false,
			checkFunc: func(t *testing.T, config *TestConfig) {
				if config.NestedConfig.NestedString != "nested_override" {
					t.Errorf("expected 'nested_override', got %s", config.NestedConfig.NestedString)
				}
				if config.NestedConfig.NestedInt != 99 {
					t.Errorf("expected 99, got %d", config.NestedConfig.NestedInt)
				}
			},
		},
		{
			name: "slice field override",
			envVars: map[string]string{
				"TEST_SLICE_FIELD": "item1,item2,item3",
			},
			prefix:    "TEST",
			expectErr: false,
			checkFunc: func(t *testing.T, config *TestConfig) {
				expected := []string{"item1", "item2", "item3"}
				if len(config.SliceField) != len(expected) {
					t.Fatalf("expected length %d, got %d", len(expected), len(config.SliceField))
				}
				for i, v := range expected {
					if config.SliceField[i] != v {
						t.Errorf("expected %s at index %d, got %s", v, i, config.SliceField[i])
					}
				}
			},
		},
		{
			name: "invalid bool value",
			envVars: map[string]string{
				"TEST_BOOL_FIELD": "not_a_bool",
			},
			prefix:    "TEST",
			expectErr: true,
		},
		{
			name: "invalid int value",
			envVars: map[string]string{
				"TEST_INT_FIELD": "not_an_int",
			},
			prefix:    "TEST",
			expectErr: true,
		},
		{
			name: "invalid duration value",
			envVars: map[string]string{
				"TEST_DURATION_FIELD": "not_a_duration",
			},
			prefix:    "TEST",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			config := &TestConfig{}
			err := ApplyEnvironmentOverrides(config, tt.prefix)

			if tt.expectErr && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, config)
			}
		})
	}
}

func TestGetBuiltinProfiles(t *testing.T) {
	profiles := GetBuiltinProfiles()

	expectedProfiles := []string{"development", "staging", "production"}
	for _, profileName := range expectedProfiles {
		profile, exists := profiles[profileName]
		if !exists {
			t.Errorf("expected profile %s to exist", profileName)
			continue
		}

		if profile.Name != profileName {
			t.Errorf("expected profile name %s, got %s", profileName, profile.Name)
		}

		// Check that profiles have overrides
		if len(profile.Overrides) == 0 {
			t.Errorf("expected profile %s to have overrides", profileName)
		}
	}

	// Check staging inherits from development
	staging := profiles["staging"]
	if staging.Inherits != "development" {
		t.Errorf("expected staging to inherit from development, got %s", staging.Inherits)
	}
}

func TestValidateStruct(t *testing.T) {
	// Test struct with validation tags
	type TestStruct struct {
		RequiredField  string   `validate:"required"`
		MinLengthField string   `validate:"min=3"`
		MaxLengthField string   `validate:"max=10"`
		OneOfField     string   `validate:"oneof=option1 option2 option3"`
		IntMinField    int      `validate:"min=5"`
		IntMaxField    int      `validate:"max=100"`
		SliceMinField  []string `validate:"min=2"`
	}

	tests := []struct {
		name      string
		config    TestStruct
		expectErr bool
		errSubstr string
	}{
		{
			name: "valid config",
			config: TestStruct{
				RequiredField:  "present",
				MinLengthField: "long_enough",
				MaxLengthField: "short",
				OneOfField:     "option1",
				IntMinField:    10,
				IntMaxField:    50,
				SliceMinField:  []string{"item1", "item2"},
			},
			expectErr: false,
		},
		{
			name: "missing required field",
			config: TestStruct{
				MinLengthField: "long_enough",
				MaxLengthField: "short",
				OneOfField:     "option1",
				IntMinField:    10,
				IntMaxField:    50,
				SliceMinField:  []string{"item1", "item2"},
			},
			expectErr: true,
			errSubstr: "RequiredField is required",
		},
		{
			name: "min length violation",
			config: TestStruct{
				RequiredField:  "present",
				MinLengthField: "xy", // too short
				MaxLengthField: "short",
				OneOfField:     "option1",
				IntMinField:    10,
				IntMaxField:    50,
				SliceMinField:  []string{"item1", "item2"},
			},
			expectErr: true,
			errSubstr: "must have at least 3 elements",
		},
		{
			name: "max length violation",
			config: TestStruct{
				RequiredField:  "present",
				MinLengthField: "long_enough",
				MaxLengthField: "this_is_way_too_long", // too long
				OneOfField:     "option1",
				IntMinField:    10,
				IntMaxField:    50,
				SliceMinField:  []string{"item1", "item2"},
			},
			expectErr: true,
			errSubstr: "must have at most 10 elements",
		},
		{
			name: "oneof violation",
			config: TestStruct{
				RequiredField:  "present",
				MinLengthField: "long_enough",
				MaxLengthField: "short",
				OneOfField:     "invalid_option",
				IntMinField:    10,
				IntMaxField:    50,
				SliceMinField:  []string{"item1", "item2"},
			},
			expectErr: true,
			errSubstr: "must be one of: option1 option2 option3",
		},
		{
			name: "int min violation",
			config: TestStruct{
				RequiredField:  "present",
				MinLengthField: "long_enough",
				MaxLengthField: "short",
				OneOfField:     "option1",
				IntMinField:    2, // too small
				IntMaxField:    50,
				SliceMinField:  []string{"item1", "item2"},
			},
			expectErr: true,
			errSubstr: "must be at least 5",
		},
		{
			name: "int max violation",
			config: TestStruct{
				RequiredField:  "present",
				MinLengthField: "long_enough",
				MaxLengthField: "short",
				OneOfField:     "option1",
				IntMinField:    10,
				IntMaxField:    150, // too large
				SliceMinField:  []string{"item1", "item2"},
			},
			expectErr: true,
			errSubstr: "must be at most 100",
		},
		{
			name: "slice min violation",
			config: TestStruct{
				RequiredField:  "present",
				MinLengthField: "long_enough",
				MaxLengthField: "short",
				OneOfField:     "option1",
				IntMinField:    10,
				IntMaxField:    50,
				SliceMinField:  []string{"item1"}, // too few items
			},
			expectErr: true,
			errSubstr: "must have at least 2 elements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.config)

			if tt.expectErr && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectErr && tt.errSubstr != "" {
				if err == nil || err.Error() == "" {
					t.Fatal("expected error with message")
				}
				// Check for substring in error message
				errMsg := err.Error()
				found := false
				// Simple substring check
				for i := 0; i <= len(errMsg)-len(tt.errSubstr); i++ {
					if errMsg[i:i+len(tt.errSubstr)] == tt.errSubstr {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errSubstr, errMsg)
				}
			}
		})
	}
}

func TestConfigValidator_ValidateWithProfile(t *testing.T) {
	validator := NewConfigValidator()

	// Add test profile
	testProfile := ConfigProfile{
		Name: "test",
		Overrides: map[string]interface{}{
			"Connection.Timeout": "15s",
			"Retry.MaxAttempts":  2,
		},
	}
	validator.AddProfile(testProfile)

	config := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	}
	config.SetDefaults()

	ctx := ValidationContext{
		Profile:     "test",
		Environment: make(map[string]string),
		Strict:      false,
	}

	err := validator.ValidateWithProfile(config, "test", ctx)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}
}

func TestConfigValidator_ProfileInheritance(t *testing.T) {
	validator := NewConfigValidator()

	// Add parent profile
	parentProfile := ConfigProfile{
		Name: "parent",
		Overrides: map[string]interface{}{
			"Connection.Timeout": "10s",
			"Retry.MaxAttempts":  1,
		},
	}
	validator.AddProfile(parentProfile)

	// Add child profile that inherits from parent
	childProfile := ConfigProfile{
		Name:     "child",
		Inherits: "parent",
		Overrides: map[string]interface{}{
			"Connection.Timeout":   "20s", // Override parent value
			"Connection.KeepAlive": "60s", // Add new value
		},
	}
	validator.AddProfile(childProfile)

	config := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	}
	config.SetDefaults()

	ctx := ValidationContext{
		Profile:     "child",
		Environment: make(map[string]string),
		Strict:      false,
	}

	err := validator.ValidateWithProfile(config, "child", ctx)
	if err != nil {
		t.Fatalf("validation with inheritance failed: %v", err)
	}
}
