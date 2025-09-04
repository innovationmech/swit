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

package config

import (
	"os"
	"strconv"
	"time"
)

// Config represents the application configuration
type Config struct {
	Server           ServerConfig           `yaml:"server"`
	Database         DatabaseConfig         `yaml:"database"`
	Tracing          TracingConfig          `yaml:"tracing"`
	ExternalServices ExternalServicesConfig `yaml:"external_services"`
	Logging          LoggingConfig          `yaml:"logging"`
}

// ServerConfig contains server configuration
type ServerConfig struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	HTTPPort string `yaml:"http_port"`
	GRPCPort string `yaml:"grpc_port"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

// TracingConfig contains OpenTelemetry tracing configuration
type TracingConfig struct {
	Enabled        bool    `yaml:"enabled"`
	JaegerEndpoint string  `yaml:"jaeger_endpoint"`
	ServiceName    string  `yaml:"service_name"`
	SamplingRate   float64 `yaml:"sampling_rate"`
}

// ExternalServicesConfig contains external service configurations
type ExternalServicesConfig struct {
	PaymentServiceURL   string        `yaml:"payment_service_url"`
	InventoryServiceURL string        `yaml:"inventory_service_url"`
	Timeout             time.Duration `yaml:"timeout"`
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
			Name:     "order-service",
			Version:  "1.0.0",
			HTTPPort: "8081",
			GRPCPort: "9081",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "./data/orders.db",
		},
		Tracing: TracingConfig{
			Enabled:        true,
			JaegerEndpoint: "http://localhost:14268/api/traces",
			ServiceName:    "order-service",
			SamplingRate:   1.0,
		},
		ExternalServices: ExternalServicesConfig{
			PaymentServiceURL:   "localhost:9082",
			InventoryServiceURL: "localhost:9083",
			Timeout:             30 * time.Second,
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
	if httpPort := os.Getenv("SERVER_HTTP_PORT"); httpPort != "" {
		c.Server.HTTPPort = httpPort
	}
	if grpcPort := os.Getenv("SERVER_GRPC_PORT"); grpcPort != "" {
		c.Server.GRPCPort = grpcPort
	}

	// Database configuration
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		c.Database.DSN = dsn
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

	// External services
	if paymentURL := os.Getenv("PAYMENT_SERVICE_URL"); paymentURL != "" {
		c.ExternalServices.PaymentServiceURL = paymentURL
	}
	if inventoryURL := os.Getenv("INVENTORY_SERVICE_URL"); inventoryURL != "" {
		c.ExternalServices.InventoryServiceURL = inventoryURL
	}

	// Logging configuration
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.Logging.Level = level
	}
}
