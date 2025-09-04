package config

import (
	"os"
	"strconv"
	"time"
)

// Config represents the application configuration
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Tracing TracingConfig `yaml:"tracing"`
	Payment PaymentConfig `yaml:"payment"`
	Logging LoggingConfig `yaml:"logging"`
}

// ServerConfig contains server configuration
type ServerConfig struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	GRPCPort string `yaml:"grpc_port"`
}

// TracingConfig contains OpenTelemetry tracing configuration
type TracingConfig struct {
	Enabled        bool    `yaml:"enabled"`
	JaegerEndpoint string  `yaml:"jaeger_endpoint"`
	ServiceName    string  `yaml:"service_name"`
	SamplingRate   float64 `yaml:"sampling_rate"`
}

// PaymentConfig contains payment processing configuration
type PaymentConfig struct {
	SimulationMode bool          `yaml:"simulation_mode"`
	Timeout        time.Duration `yaml:"timeout"`
	FailureRate    float64       `yaml:"failure_rate"` // 0.0 to 1.0
	ProcessingTime time.Duration `yaml:"processing_time"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Name:     "payment-service",
			Version:  "1.0.0",
			GRPCPort: "9082",
		},
		Tracing: TracingConfig{
			Enabled:        true,
			JaegerEndpoint: "http://localhost:14268/api/traces",
			ServiceName:    "payment-service",
			SamplingRate:   1.0,
		},
		Payment: PaymentConfig{
			SimulationMode: true,
			Timeout:        30 * time.Second,
			FailureRate:    0.1, // 10% failure rate for demo
			ProcessingTime: 2 * time.Second,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// ApplyEnvironmentOverrides applies environment variable overrides to the configuration
func (c *Config) ApplyEnvironmentOverrides() {
	// Server configuration
	if grpcPort := os.Getenv("SERVER_GRPC_PORT"); grpcPort != "" {
		c.Server.GRPCPort = grpcPort
	}

	// Tracing configuration
	if enabled := os.Getenv("TRACING_ENABLED"); enabled != "" {
		c.Tracing.Enabled = enabled == "true" || enabled == "1"
	}
	if endpoint := os.Getenv("JAEGER_ENDPOINT"); endpoint != "" {
		c.Tracing.JaegerEndpoint = endpoint
	}
	if rate := os.Getenv("TRACING_SAMPLING_RATE"); rate != "" {
		if samplingRate, err := strconv.ParseFloat(rate, 64); err == nil {
			c.Tracing.SamplingRate = samplingRate
		}
	}

	// Payment configuration
	if simMode := os.Getenv("PAYMENT_SIMULATION_MODE"); simMode != "" {
		c.Payment.SimulationMode = simMode == "true" || simMode == "1"
	}
	if timeout := os.Getenv("PAYMENT_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			c.Payment.Timeout = duration
		}
	}

	// Logging configuration
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.Logging.Level = level
	}
}
