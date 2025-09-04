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
	"fmt"
	"time"
)

// TracingConfig represents the complete tracing configuration
type TracingConfig struct {
	// Basic configuration
	Enabled     bool   `yaml:"enabled" mapstructure:"enabled"`
	ServiceName string `yaml:"service_name" mapstructure:"service_name"`

	// Sampling configuration
	Sampling SamplingConfig `yaml:"sampling" mapstructure:"sampling"`

	// Exporter configuration
	Exporter ExporterConfig `yaml:"exporter" mapstructure:"exporter"`

	// Resource attributes
	ResourceAttributes map[string]string `yaml:"resource_attributes" mapstructure:"resource_attributes"`

	// Propagators configuration
	Propagators []string `yaml:"propagators" mapstructure:"propagators"`
}

// SamplingConfig represents sampling strategy configuration
type SamplingConfig struct {
	Type string  `yaml:"type" mapstructure:"type"` // always_on, always_off, traceidratio
	Rate float64 `yaml:"rate" mapstructure:"rate"` // 0.0-1.0 for traceidratio
}

// ExporterConfig represents the exporter configuration
type ExporterConfig struct {
	Type     string            `yaml:"type" mapstructure:"type"` // jaeger, otlp, console
	Endpoint string            `yaml:"endpoint" mapstructure:"endpoint"`
	Headers  map[string]string `yaml:"headers" mapstructure:"headers"`
	Timeout  string            `yaml:"timeout" mapstructure:"timeout"`

	// Jaeger specific configuration
	Jaeger JaegerConfig `yaml:"jaeger" mapstructure:"jaeger"`

	// OTLP specific configuration
	OTLP OTLPConfig `yaml:"otlp" mapstructure:"otlp"`
}

// JaegerConfig represents Jaeger-specific configuration
type JaegerConfig struct {
	AgentEndpoint     string `yaml:"agent_endpoint" mapstructure:"agent_endpoint"`
	CollectorEndpoint string `yaml:"collector_endpoint" mapstructure:"collector_endpoint"`
	Username          string `yaml:"username" mapstructure:"username"`
	Password          string `yaml:"password" mapstructure:"password"`
	RPCTimeout        string `yaml:"rpc_timeout" mapstructure:"rpc_timeout"`
}

// OTLPConfig represents OTLP-specific configuration
type OTLPConfig struct {
	Endpoint    string            `yaml:"endpoint" mapstructure:"endpoint"`
	Insecure    bool              `yaml:"insecure" mapstructure:"insecure"`
	Headers     map[string]string `yaml:"headers" mapstructure:"headers"`
	Compression string            `yaml:"compression" mapstructure:"compression"` // gzip, none
}

// DefaultTracingConfig returns a default tracing configuration
func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		Enabled:     false,
		ServiceName: "swit-service",
		Sampling: SamplingConfig{
			Type: "traceidratio",
			Rate: 0.1,
		},
		Exporter: ExporterConfig{
			Type:    "console",
			Timeout: "10s",
		},
		ResourceAttributes: map[string]string{
			"service.version":        "unknown",
			"deployment.environment": "unknown",
		},
		Propagators: []string{"tracecontext", "baggage"},
	}
}

// Validate validates the tracing configuration
func (c *TracingConfig) Validate() error {
	if !c.Enabled {
		return nil // No need to validate if tracing is disabled
	}

	if c.ServiceName == "" {
		return fmt.Errorf("service_name is required when tracing is enabled")
	}

	if err := c.Sampling.Validate(); err != nil {
		return fmt.Errorf("sampling configuration invalid: %w", err)
	}

	if err := c.Exporter.Validate(); err != nil {
		return fmt.Errorf("exporter configuration invalid: %w", err)
	}

	return nil
}

// Validate validates the sampling configuration
func (s *SamplingConfig) Validate() error {
	switch s.Type {
	case "always_on", "always_off":
		// No additional validation needed
	case "traceidratio":
		if s.Rate < 0.0 || s.Rate > 1.0 {
			return fmt.Errorf("sampling rate must be between 0.0 and 1.0, got %f", s.Rate)
		}
	case "":
		return fmt.Errorf("sampling type is required")
	default:
		return fmt.Errorf("unsupported sampling type: %s", s.Type)
	}
	return nil
}

// Validate validates the exporter configuration
func (e *ExporterConfig) Validate() error {
	switch e.Type {
	case "console":
		// Console exporter doesn't require additional configuration
	case "jaeger":
		if e.Endpoint == "" && e.Jaeger.CollectorEndpoint == "" && e.Jaeger.AgentEndpoint == "" {
			return fmt.Errorf("jaeger exporter requires either endpoint, collector_endpoint, or agent_endpoint")
		}
	case "otlp":
		if e.Endpoint == "" && e.OTLP.Endpoint == "" {
			return fmt.Errorf("otlp exporter requires endpoint")
		}
	case "":
		return fmt.Errorf("exporter type is required")
	default:
		return fmt.Errorf("unsupported exporter type: %s", e.Type)
	}

	if e.Timeout != "" {
		if _, err := time.ParseDuration(e.Timeout); err != nil {
			return fmt.Errorf("invalid timeout format: %w", err)
		}
	}

	return nil
}

// GetTimeout returns the parsed timeout duration or a default value
func (e *ExporterConfig) GetTimeout() time.Duration {
	if e.Timeout == "" {
		return 10 * time.Second
	}

	duration, err := time.ParseDuration(e.Timeout)
	if err != nil {
		return 10 * time.Second
	}

	return duration
}

// Validate validates the Jaeger configuration
func (j *JaegerConfig) Validate() error {
	if j.RPCTimeout != "" {
		if _, err := time.ParseDuration(j.RPCTimeout); err != nil {
			return fmt.Errorf("invalid rpc_timeout format: %w", err)
		}
	}
	return nil
}

// GetRPCTimeout returns the parsed RPC timeout duration or a default value
func (j *JaegerConfig) GetRPCTimeout() time.Duration {
	if j.RPCTimeout == "" {
		return 5 * time.Second
	}

	duration, err := time.ParseDuration(j.RPCTimeout)
	if err != nil {
		return 5 * time.Second
	}

	return duration
}

// Validate validates the OTLP configuration
func (o *OTLPConfig) Validate() error {
	if o.Compression != "" && o.Compression != "gzip" && o.Compression != "none" {
		return fmt.Errorf("unsupported compression type: %s", o.Compression)
	}
	return nil
}
