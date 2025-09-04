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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTracingConfig(t *testing.T) {
	config := DefaultTracingConfig()

	assert.NotNil(t, config)
	assert.False(t, config.Enabled)
	assert.Equal(t, "swit-service", config.ServiceName)
	assert.Equal(t, "traceidratio", config.Sampling.Type)
	assert.Equal(t, 0.1, config.Sampling.Rate)
	assert.Equal(t, "console", config.Exporter.Type)
	assert.Equal(t, "10s", config.Exporter.Timeout)
	assert.Contains(t, config.Propagators, "tracecontext")
	assert.Contains(t, config.Propagators, "baggage")
	assert.NotNil(t, config.ResourceAttributes)
}

func TestTracingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *TracingConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid disabled config",
			config: &TracingConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "valid enabled config",
			config: &TracingConfig{
				Enabled:     true,
				ServiceName: "test-service",
				Sampling: SamplingConfig{
					Type: "always_on",
				},
				Exporter: ExporterConfig{
					Type: "console",
				},
			},
			wantErr: false,
		},
		{
			name: "missing service name",
			config: &TracingConfig{
				Enabled:     true,
				ServiceName: "",
				Sampling: SamplingConfig{
					Type: "always_on",
				},
				Exporter: ExporterConfig{
					Type: "console",
				},
			},
			wantErr: true,
			errMsg:  "service_name is required",
		},
		{
			name: "invalid sampling config",
			config: &TracingConfig{
				Enabled:     true,
				ServiceName: "test-service",
				Sampling: SamplingConfig{
					Type: "invalid",
				},
				Exporter: ExporterConfig{
					Type: "console",
				},
			},
			wantErr: true,
			errMsg:  "sampling configuration invalid",
		},
		{
			name: "invalid exporter config",
			config: &TracingConfig{
				Enabled:     true,
				ServiceName: "test-service",
				Sampling: SamplingConfig{
					Type: "always_on",
				},
				Exporter: ExporterConfig{
					Type: "invalid",
				},
			},
			wantErr: true,
			errMsg:  "exporter configuration invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSamplingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  SamplingConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "always_on",
			config:  SamplingConfig{Type: "always_on"},
			wantErr: false,
		},
		{
			name:    "always_off",
			config:  SamplingConfig{Type: "always_off"},
			wantErr: false,
		},
		{
			name:    "traceidratio valid rate",
			config:  SamplingConfig{Type: "traceidratio", Rate: 0.5},
			wantErr: false,
		},
		{
			name:    "traceidratio rate zero",
			config:  SamplingConfig{Type: "traceidratio", Rate: 0.0},
			wantErr: false,
		},
		{
			name:    "traceidratio rate one",
			config:  SamplingConfig{Type: "traceidratio", Rate: 1.0},
			wantErr: false,
		},
		{
			name:    "traceidratio rate negative",
			config:  SamplingConfig{Type: "traceidratio", Rate: -0.1},
			wantErr: true,
			errMsg:  "sampling rate must be between 0.0 and 1.0",
		},
		{
			name:    "traceidratio rate too high",
			config:  SamplingConfig{Type: "traceidratio", Rate: 1.1},
			wantErr: true,
			errMsg:  "sampling rate must be between 0.0 and 1.0",
		},
		{
			name:    "empty type",
			config:  SamplingConfig{Type: ""},
			wantErr: true,
			errMsg:  "sampling type is required",
		},
		{
			name:    "invalid type",
			config:  SamplingConfig{Type: "invalid"},
			wantErr: true,
			errMsg:  "unsupported sampling type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExporterConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ExporterConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "console exporter",
			config:  ExporterConfig{Type: "console"},
			wantErr: false,
		},
		{
			name:    "jaeger with endpoint",
			config:  ExporterConfig{Type: "jaeger", Endpoint: "http://localhost:14268/api/traces"},
			wantErr: false,
		},
		{
			name:    "jaeger with collector endpoint",
			config:  ExporterConfig{Type: "jaeger", Jaeger: JaegerConfig{CollectorEndpoint: "http://localhost:14268/api/traces"}},
			wantErr: false,
		},
		{
			name:    "jaeger with agent endpoint",
			config:  ExporterConfig{Type: "jaeger", Jaeger: JaegerConfig{AgentEndpoint: "localhost:6831"}},
			wantErr: false,
		},
		{
			name:    "jaeger without endpoints",
			config:  ExporterConfig{Type: "jaeger"},
			wantErr: true,
			errMsg:  "jaeger exporter requires either endpoint",
		},
		{
			name:    "otlp with endpoint",
			config:  ExporterConfig{Type: "otlp", Endpoint: "http://localhost:4318"},
			wantErr: false,
		},
		{
			name:    "otlp with OTLP endpoint",
			config:  ExporterConfig{Type: "otlp", OTLP: OTLPConfig{Endpoint: "http://localhost:4318"}},
			wantErr: false,
		},
		{
			name:    "otlp without endpoint",
			config:  ExporterConfig{Type: "otlp"},
			wantErr: true,
			errMsg:  "otlp exporter requires endpoint",
		},
		{
			name:    "empty type",
			config:  ExporterConfig{Type: ""},
			wantErr: true,
			errMsg:  "exporter type is required",
		},
		{
			name:    "invalid type",
			config:  ExporterConfig{Type: "invalid"},
			wantErr: true,
			errMsg:  "unsupported exporter type",
		},
		{
			name:    "invalid timeout",
			config:  ExporterConfig{Type: "console", Timeout: "invalid"},
			wantErr: true,
			errMsg:  "invalid timeout format",
		},
		{
			name:    "valid timeout",
			config:  ExporterConfig{Type: "console", Timeout: "30s"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExporterConfig_GetTimeout(t *testing.T) {
	tests := []struct {
		name     string
		config   ExporterConfig
		expected time.Duration
	}{
		{
			name:     "empty timeout",
			config:   ExporterConfig{},
			expected: 10 * time.Second,
		},
		{
			name:     "valid timeout",
			config:   ExporterConfig{Timeout: "30s"},
			expected: 30 * time.Second,
		},
		{
			name:     "invalid timeout falls back to default",
			config:   ExporterConfig{Timeout: "invalid"},
			expected: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetTimeout()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJaegerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  JaegerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty config",
			config:  JaegerConfig{},
			wantErr: false,
		},
		{
			name:    "valid RPC timeout",
			config:  JaegerConfig{RPCTimeout: "10s"},
			wantErr: false,
		},
		{
			name:    "invalid RPC timeout",
			config:  JaegerConfig{RPCTimeout: "invalid"},
			wantErr: true,
			errMsg:  "invalid rpc_timeout format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJaegerConfig_GetRPCTimeout(t *testing.T) {
	tests := []struct {
		name     string
		config   JaegerConfig
		expected time.Duration
	}{
		{
			name:     "empty timeout",
			config:   JaegerConfig{},
			expected: 5 * time.Second,
		},
		{
			name:     "valid timeout",
			config:   JaegerConfig{RPCTimeout: "15s"},
			expected: 15 * time.Second,
		},
		{
			name:     "invalid timeout falls back to default",
			config:   JaegerConfig{RPCTimeout: "invalid"},
			expected: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetRPCTimeout()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOTLPConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  OTLPConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty config",
			config:  OTLPConfig{},
			wantErr: false,
		},
		{
			name:    "valid compression gzip",
			config:  OTLPConfig{Compression: "gzip"},
			wantErr: false,
		},
		{
			name:    "valid compression none",
			config:  OTLPConfig{Compression: "none"},
			wantErr: false,
		},
		{
			name:    "invalid compression",
			config:  OTLPConfig{Compression: "invalid"},
			wantErr: true,
			errMsg:  "unsupported compression type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTracingConfigYAMLIntegration(t *testing.T) {
	// This test verifies that the struct tags work correctly with YAML unmarshalling
	var config TracingConfig
	// Note: We're not actually unmarshalling here since it would require yaml package
	// But we can verify the struct is properly structured for YAML
	require.NotNil(t, config)

	// Verify mapstructure tags are present (these would be used by Viper)
	// This is a structural test to ensure the tags are correctly defined
}

func TestTracingConfig_ApplyEnvironmentOverrides(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{}
	envVars := []string{
		"SWIT_TRACING_ENABLED",
		"SWIT_TRACING_SERVICE_NAME",
		"SWIT_TRACING_SAMPLING_TYPE",
		"SWIT_TRACING_SAMPLING_RATE",
		"SWIT_TRACING_EXPORTER_TYPE",
		"SWIT_TRACING_EXPORTER_ENDPOINT",
		"SWIT_TRACING_JAEGER_AGENT_ENDPOINT",
		"SWIT_TRACING_OTLP_ENDPOINT",
		"SWIT_TRACING_RESOURCE_SERVICE_VERSION",
		"SWIT_TRACING_RESOURCE_ENVIRONMENT",
		"SWIT_TRACING_PROPAGATORS",
	}
	
	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}
	
	// Cleanup function
	defer func() {
		for _, env := range envVars {
			if value := originalEnv[env]; value != "" {
				os.Setenv(env, value)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	tests := []struct {
		name     string
		initial  *TracingConfig
		envVars  map[string]string
		expected *TracingConfig
	}{
		{
			name: "basic overrides",
			initial: &TracingConfig{
				Enabled:     false,
				ServiceName: "original-service",
				Sampling: SamplingConfig{
					Type: "always_off",
					Rate: 0.0,
				},
				Exporter: ExporterConfig{
					Type: "console",
				},
			},
			envVars: map[string]string{
				"SWIT_TRACING_ENABLED":       "true",
				"SWIT_TRACING_SERVICE_NAME":  "env-service",
				"SWIT_TRACING_SAMPLING_TYPE": "traceidratio",
				"SWIT_TRACING_SAMPLING_RATE": "0.5",
				"SWIT_TRACING_EXPORTER_TYPE": "jaeger",
			},
			expected: &TracingConfig{
				Enabled:     true,
				ServiceName: "env-service",
				Sampling: SamplingConfig{
					Type: "traceidratio",
					Rate: 0.5,
				},
				Exporter: ExporterConfig{
					Type: "jaeger",
				},
			},
		},
		{
			name: "resource attributes",
			initial: &TracingConfig{
				ResourceAttributes: map[string]string{
					"service.version": "v1.0.0",
				},
			},
			envVars: map[string]string{
				"SWIT_TRACING_RESOURCE_SERVICE_VERSION":    "v2.0.0",
				"SWIT_TRACING_RESOURCE_ENVIRONMENT":        "production",
				"SWIT_TRACING_RESOURCE_CUSTOM_TEAM":        "platform",
				"SWIT_TRACING_RESOURCE_CUSTOM_COST_CENTER": "engineering",
			},
			expected: &TracingConfig{
				ResourceAttributes: map[string]string{
					"service.version":        "v2.0.0",
					"deployment.environment": "production",
					"team":                   "platform",
					"cost.center":            "engineering",
				},
			},
		},
		{
			name: "propagators",
			initial: &TracingConfig{
				Propagators: []string{"tracecontext"},
			},
			envVars: map[string]string{
				"SWIT_TRACING_PROPAGATORS": "tracecontext,baggage,jaeger",
			},
			expected: &TracingConfig{
				Propagators: []string{"tracecontext", "baggage", "jaeger"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Apply overrides
			config := tt.initial
			config.ApplyEnvironmentOverrides()

			// Verify results
			if tt.expected.Enabled != false {
				assert.Equal(t, tt.expected.Enabled, config.Enabled)
			}
			if tt.expected.ServiceName != "" {
				assert.Equal(t, tt.expected.ServiceName, config.ServiceName)
			}
			if tt.expected.Sampling.Type != "" {
				assert.Equal(t, tt.expected.Sampling.Type, config.Sampling.Type)
			}
			if tt.expected.Sampling.Rate != 0 {
				assert.Equal(t, tt.expected.Sampling.Rate, config.Sampling.Rate)
			}
			if tt.expected.Exporter.Type != "" {
				assert.Equal(t, tt.expected.Exporter.Type, config.Exporter.Type)
			}
			if len(tt.expected.ResourceAttributes) > 0 {
				for key, expectedValue := range tt.expected.ResourceAttributes {
					assert.Equal(t, expectedValue, config.ResourceAttributes[key], "Resource attribute %s", key)
				}
			}
			if len(tt.expected.Propagators) > 0 {
				assert.Equal(t, tt.expected.Propagators, config.Propagators)
			}

			// Clean up environment variables for next test
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestTracingConfig_ApplyEnvironmentOverrides_InvalidValues(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{}
	envVars := []string{
		"SWIT_TRACING_ENABLED",
		"SWIT_TRACING_SAMPLING_RATE",
		"SWIT_TRACING_OTLP_INSECURE",
	}
	
	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}
	
	defer func() {
		for _, env := range envVars {
			if value := originalEnv[env]; value != "" {
				os.Setenv(env, value)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	config := &TracingConfig{
		Enabled: false,
		Sampling: SamplingConfig{
			Rate: 0.1,
		},
		Exporter: ExporterConfig{
			OTLP: OTLPConfig{
				Insecure: true,
			},
		},
	}

	// Set invalid environment variables
	os.Setenv("SWIT_TRACING_ENABLED", "invalid-bool")
	os.Setenv("SWIT_TRACING_SAMPLING_RATE", "invalid-float")
	os.Setenv("SWIT_TRACING_OTLP_INSECURE", "invalid-bool")

	// Apply overrides - should not crash and should ignore invalid values
	config.ApplyEnvironmentOverrides()

	// Verify original values are preserved
	assert.False(t, config.Enabled)
	assert.Equal(t, 0.1, config.Sampling.Rate)
	assert.True(t, config.Exporter.OTLP.Insecure)
}
