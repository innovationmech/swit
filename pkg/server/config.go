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

package server

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/keepalive"

	"github.com/innovationmech/swit/pkg/tracing"
	"github.com/innovationmech/swit/pkg/types"
)

// ServerConfig holds the complete configuration for a base server instance
// It includes transport, discovery, middleware, messaging, and monitoring configuration
type ServerConfig struct {
	ServiceName     string                `yaml:"service_name" json:"service_name"`
	HTTP            HTTPConfig            `yaml:"http" json:"http"`
	GRPC            GRPCConfig            `yaml:"grpc" json:"grpc"`
	Discovery       DiscoveryConfig       `yaml:"discovery" json:"discovery"`
	Middleware      MiddlewareConfig      `yaml:"middleware" json:"middleware"`
	Messaging       MessagingConfig       `yaml:"messaging" json:"messaging"`
	Sentry          SentryConfig          `yaml:"sentry" json:"sentry"`
	Logging         LoggingConfig         `yaml:"logging" json:"logging"`
	Prometheus      PrometheusConfig      `yaml:"prometheus" json:"prometheus"`
	Tracing         tracing.TracingConfig `yaml:"tracing" json:"tracing"`
	ShutdownTimeout time.Duration         `yaml:"shutdown_timeout" json:"shutdown_timeout"`
}

// HTTPConfig holds HTTP transport specific configuration
type HTTPConfig struct {
	Port         string            `yaml:"port" json:"port"`
	Address      string            `yaml:"address" json:"address"`
	EnableReady  bool              `yaml:"enable_ready" json:"enable_ready"`
	Enabled      bool              `yaml:"enabled" json:"enabled"`
	TestMode     bool              `yaml:"test_mode" json:"test_mode"`
	TestPort     string            `yaml:"test_port" json:"test_port"`
	Middleware   HTTPMiddleware    `yaml:"middleware" json:"middleware"`
	ReadTimeout  time.Duration     `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration     `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration     `yaml:"idle_timeout" json:"idle_timeout"`
	Headers      map[string]string `yaml:"headers" json:"headers"`
}

// HTTPMiddleware holds HTTP-specific middleware configuration
type HTTPMiddleware struct {
	EnableCORS      bool              `yaml:"enable_cors" json:"enable_cors"`
	EnableAuth      bool              `yaml:"enable_auth" json:"enable_auth"`
	EnableRateLimit bool              `yaml:"enable_rate_limit" json:"enable_rate_limit"`
	EnableLogging   bool              `yaml:"enable_logging" json:"enable_logging"`
	EnableTimeout   bool              `yaml:"enable_timeout" json:"enable_timeout"`
	CORSConfig      CORSConfig        `yaml:"cors" json:"cors"`
	RateLimitConfig RateLimitConfig   `yaml:"rate_limit" json:"rate_limit"`
	TimeoutConfig   TimeoutConfig     `yaml:"timeout" json:"timeout"`
	CustomHeaders   map[string]string `yaml:"custom_headers" json:"custom_headers"`
}

// CORSConfig holds CORS middleware configuration
type CORSConfig struct {
	AllowOrigins     []string `yaml:"allow_origins" json:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods" json:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers" json:"allow_headers"`
	ExposeHeaders    []string `yaml:"expose_headers" json:"expose_headers"`
	AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials"`
	MaxAge           int      `yaml:"max_age" json:"max_age"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int           `yaml:"requests_per_second" json:"requests_per_second"`
	BurstSize         int           `yaml:"burst_size" json:"burst_size"`
	WindowSize        time.Duration `yaml:"window_size" json:"window_size"`
	KeyFunc           string        `yaml:"key_func" json:"key_func"` // "ip", "user", "custom"
}

// TimeoutConfig holds timeout middleware configuration
type TimeoutConfig struct {
	RequestTimeout time.Duration `yaml:"request_timeout" json:"request_timeout"`
	HandlerTimeout time.Duration `yaml:"handler_timeout" json:"handler_timeout"`
}

// GRPCConfig holds gRPC transport specific configuration
type GRPCConfig struct {
	Port                string                `yaml:"port" json:"port"`
	Address             string                `yaml:"address" json:"address"`
	EnableKeepalive     bool                  `yaml:"enable_keepalive" json:"enable_keepalive"`
	EnableReflection    bool                  `yaml:"enable_reflection" json:"enable_reflection"`
	EnableHealthService bool                  `yaml:"enable_health_service" json:"enable_health_service"`
	Enabled             bool                  `yaml:"enabled" json:"enabled"`
	TestMode            bool                  `yaml:"test_mode" json:"test_mode"`
	TestPort            string                `yaml:"test_port" json:"test_port"`
	MaxRecvMsgSize      int                   `yaml:"max_recv_msg_size" json:"max_recv_msg_size"`
	MaxSendMsgSize      int                   `yaml:"max_send_msg_size" json:"max_send_msg_size"`
	KeepaliveParams     GRPCKeepaliveParams   `yaml:"keepalive_params" json:"keepalive_params"`
	KeepalivePolicy     GRPCKeepalivePolicy   `yaml:"keepalive_policy" json:"keepalive_policy"`
	Interceptors        GRPCInterceptorConfig `yaml:"interceptors" json:"interceptors"`
	TLS                 GRPCTLSConfig         `yaml:"tls" json:"tls"`
}

// GRPCKeepaliveParams holds gRPC keepalive parameters
type GRPCKeepaliveParams struct {
	MaxConnectionIdle     time.Duration `yaml:"max_connection_idle" json:"max_connection_idle"`
	MaxConnectionAge      time.Duration `yaml:"max_connection_age" json:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration `yaml:"max_connection_age_grace" json:"max_connection_age_grace"`
	Time                  time.Duration `yaml:"time" json:"time"`
	Timeout               time.Duration `yaml:"timeout" json:"timeout"`
}

// GRPCKeepalivePolicy holds gRPC keepalive enforcement policy
type GRPCKeepalivePolicy struct {
	MinTime             time.Duration `yaml:"min_time" json:"min_time"`
	PermitWithoutStream bool          `yaml:"permit_without_stream" json:"permit_without_stream"`
}

// GRPCInterceptorConfig holds gRPC interceptor configuration
type GRPCInterceptorConfig struct {
	EnableAuth      bool `yaml:"enable_auth" json:"enable_auth"`
	EnableLogging   bool `yaml:"enable_logging" json:"enable_logging"`
	EnableMetrics   bool `yaml:"enable_metrics" json:"enable_metrics"`
	EnableRecovery  bool `yaml:"enable_recovery" json:"enable_recovery"`
	EnableRateLimit bool `yaml:"enable_rate_limit" json:"enable_rate_limit"`
}

// GRPCTLSConfig holds gRPC TLS configuration
type GRPCTLSConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	CertFile   string `yaml:"cert_file" json:"cert_file"`
	KeyFile    string `yaml:"key_file" json:"key_file"`
	CAFile     string `yaml:"ca_file" json:"ca_file"`
	ServerName string `yaml:"server_name" json:"server_name"`
}

// DiscoveryFailureMode defines how the server should handle service discovery failures
type DiscoveryFailureMode string

const (
	// DiscoveryFailureModeGraceful continues server startup even if discovery registration fails
	// This is useful for development environments or when discovery is not critical
	DiscoveryFailureModeGraceful DiscoveryFailureMode = "graceful"

	// DiscoveryFailureModeFailFast stops server startup if discovery registration fails
	// This is recommended for production environments where discovery is critical
	DiscoveryFailureModeFailFast DiscoveryFailureMode = "fail_fast"

	// DiscoveryFailureModeStrict is similar to fail_fast but also requires discovery to be healthy
	// before allowing registration. This provides the highest level of reliability.
	DiscoveryFailureModeStrict DiscoveryFailureMode = "strict"
)

// DiscoveryConfig holds service discovery configuration
type DiscoveryConfig struct {
	Address             string               `yaml:"address" json:"address"`
	ServiceName         string               `yaml:"service_name" json:"service_name"`
	Tags                []string             `yaml:"tags" json:"tags"`
	Enabled             bool                 `yaml:"enabled" json:"enabled"`
	FailureMode         DiscoveryFailureMode `yaml:"failure_mode" json:"failure_mode"`
	HealthCheckRequired bool                 `yaml:"health_check_required" json:"health_check_required"`
	RegistrationTimeout time.Duration        `yaml:"registration_timeout" json:"registration_timeout"`
}

// MiddlewareConfig holds middleware configuration flags
type MiddlewareConfig struct {
	EnableCORS      bool `yaml:"enable_cors" json:"enable_cors"`
	EnableAuth      bool `yaml:"enable_auth" json:"enable_auth"`
	EnableRateLimit bool `yaml:"enable_rate_limit" json:"enable_rate_limit"`
	EnableLogging   bool `yaml:"enable_logging" json:"enable_logging"`
}

// MessagingConfig holds messaging system configuration
type MessagingConfig struct {
	// Enabled indicates if messaging is enabled
	Enabled bool `yaml:"enabled" json:"enabled"`

	// DefaultBroker specifies the default broker to use
	DefaultBroker string `yaml:"default_broker" json:"default_broker"`

	// Brokers configuration map
	Brokers map[string]BrokerConfig `yaml:"brokers" json:"brokers"`

	// Connection settings
	Connection MessagingConnectionConfig `yaml:"connection" json:"connection"`

	// Security settings
	Security MessagingSecurityConfig `yaml:"security" json:"security"`

	// Performance settings
	Performance MessagingPerformanceConfig `yaml:"performance" json:"performance"`

	// Monitoring settings
	Monitoring MessagingMonitoringConfig `yaml:"monitoring" json:"monitoring"`

	// Shutdown settings for graceful messaging shutdown
	Shutdown MessagingShutdownConfig `yaml:"shutdown" json:"shutdown"`
}

// BrokerConfig defines configuration for a specific message broker
type BrokerConfig struct {
	// Type specifies the broker type (kafka, nats, rabbitmq, inmemory)
	Type string `yaml:"type" json:"type"`

	// Endpoints contains the list of broker endpoints
	Endpoints []string `yaml:"endpoints" json:"endpoints"`

	// Authentication configuration (optional)
	Authentication *MessagingAuthConfig `yaml:"authentication,omitempty" json:"authentication,omitempty"`

	// TLS configuration (optional)
	TLS *MessagingTLSConfig `yaml:"tls,omitempty" json:"tls,omitempty"`

	// Connection-specific settings
	ConnectionSettings map[string]interface{} `yaml:"connection_settings,omitempty" json:"connection_settings,omitempty"`
}

// MessagingConnectionConfig defines messaging connection settings
type MessagingConnectionConfig struct {
	// Timeout for establishing connections
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// KeepAlive interval for connection health checks
	KeepAlive time.Duration `yaml:"keep_alive" json:"keep_alive"`

	// MaxAttempts for connection establishment
	MaxAttempts int `yaml:"max_attempts" json:"max_attempts"`

	// RetryInterval between connection attempts
	RetryInterval time.Duration `yaml:"retry_interval" json:"retry_interval"`

	// PoolSize for connection pooling
	PoolSize int `yaml:"pool_size" json:"pool_size"`

	// IdleTimeout for idle connections
	IdleTimeout time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
}

// MessagingAuthConfig defines messaging authentication settings
type MessagingAuthConfig struct {
	// Type specifies the authentication type (none, sasl, oauth2, apikey)
	Type string `yaml:"type" json:"type"`

	// Username for SASL authentication
	Username string `yaml:"username,omitempty" json:"username,omitempty"`

	// Password for SASL authentication
	Password string `yaml:"password,omitempty" json:"password,omitempty"`

	// Token for token-based authentication
	Token string `yaml:"token,omitempty" json:"token,omitempty"`

	// APIKey for API key authentication
	APIKey string `yaml:"api_key,omitempty" json:"api_key,omitempty"`

	// Mechanism for SASL (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)
	Mechanism string `yaml:"mechanism,omitempty" json:"mechanism,omitempty"`
}

// MessagingTLSConfig defines messaging TLS settings
type MessagingTLSConfig struct {
	// Enabled indicates if TLS is enabled
	Enabled bool `yaml:"enabled" json:"enabled"`

	// CertFile is the path to the client certificate file
	CertFile string `yaml:"cert_file,omitempty" json:"cert_file,omitempty"`

	// KeyFile is the path to the client private key file
	KeyFile string `yaml:"key_file,omitempty" json:"key_file,omitempty"`

	// CAFile is the path to the certificate authority file
	CAFile string `yaml:"ca_file,omitempty" json:"ca_file,omitempty"`

	// ServerName for server name verification
	ServerName string `yaml:"server_name,omitempty" json:"server_name,omitempty"`

	// SkipVerify skips certificate verification (insecure)
	SkipVerify bool `yaml:"skip_verify" json:"skip_verify"`
}

// MessagingSecurityConfig defines messaging security settings
type MessagingSecurityConfig struct {
	// EnableEncryption enables message encryption
	EnableEncryption bool `yaml:"enable_encryption" json:"enable_encryption"`

	// EnableAuthentication enables authentication checks
	EnableAuthentication bool `yaml:"enable_authentication" json:"enable_authentication"`

	// EnableAuthorization enables authorization checks
	EnableAuthorization bool `yaml:"enable_authorization" json:"enable_authorization"`

	// MessageSigning enables message signing
	MessageSigning bool `yaml:"message_signing" json:"message_signing"`
}

// MessagingPerformanceConfig defines messaging performance settings
type MessagingPerformanceConfig struct {
	// BatchSize for batching messages
	BatchSize int `yaml:"batch_size" json:"batch_size"`

	// BatchTimeout for batch flushing
	BatchTimeout time.Duration `yaml:"batch_timeout" json:"batch_timeout"`

	// BufferSize for internal buffering
	BufferSize int `yaml:"buffer_size" json:"buffer_size"`

	// Concurrency level for message processing
	Concurrency int `yaml:"concurrency" json:"concurrency"`

	// PrefetchCount for message prefetching
	PrefetchCount int `yaml:"prefetch_count" json:"prefetch_count"`

	// CompressionEnabled enables message compression
	CompressionEnabled bool `yaml:"compression_enabled" json:"compression_enabled"`
}

// MessagingMonitoringConfig defines messaging monitoring settings
type MessagingMonitoringConfig struct {
	// Enabled indicates if monitoring is enabled
	Enabled bool `yaml:"enabled" json:"enabled"`

	// MetricsEnabled enables metrics collection
	MetricsEnabled bool `yaml:"metrics_enabled" json:"metrics_enabled"`

	// TracingEnabled enables distributed tracing
	TracingEnabled bool `yaml:"tracing_enabled" json:"tracing_enabled"`

	// HealthCheckEnabled enables health checks
	HealthCheckEnabled bool `yaml:"health_check_enabled" json:"health_check_enabled"`

	// HealthCheckInterval for health check operations
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
}

// MessagingShutdownConfig defines graceful shutdown settings for messaging
type MessagingShutdownConfig struct {
	// Timeout is the maximum time to wait for shutdown completion
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// ForceTimeout is the maximum time before forcing termination
	ForceTimeout time.Duration `yaml:"force_timeout" json:"force_timeout"`

	// DrainTimeout is the maximum time to wait for in-flight message completion
	DrainTimeout time.Duration `yaml:"drain_timeout" json:"drain_timeout"`

	// ReportInterval is the interval for reporting shutdown status
	ReportInterval time.Duration `yaml:"report_interval" json:"report_interval"`

	// MaxInflightMessages is the maximum number of messages to track during shutdown
	MaxInflightMessages int `yaml:"max_inflight_messages" json:"max_inflight_messages"`
}

// SentryConfig holds Sentry error monitoring configuration
type SentryConfig struct {
	Enabled              bool              `yaml:"enabled" json:"enabled"`
	DSN                  string            `yaml:"dsn" json:"dsn"`
	Environment          string            `yaml:"environment" json:"environment"`
	Release              string            `yaml:"release" json:"release"`
	SampleRate           float64           `yaml:"sample_rate" json:"sample_rate"`
	TracesSampleRate     float64           `yaml:"traces_sample_rate" json:"traces_sample_rate"`
	AttachStacktrace     bool              `yaml:"attach_stacktrace" json:"attach_stacktrace"`
	EnableTracing        bool              `yaml:"enable_tracing" json:"enable_tracing"`
	Debug                bool              `yaml:"debug" json:"debug"`
	ServerName           string            `yaml:"server_name" json:"server_name"`
	Tags                 map[string]string `yaml:"tags" json:"tags"`
	BeforeSend           bool              `yaml:"before_send" json:"before_send"`
	IntegrateHTTP        bool              `yaml:"integrate_http" json:"integrate_http"`
	IntegrateGRPC        bool              `yaml:"integrate_grpc" json:"integrate_grpc"`
	CapturePanics        bool              `yaml:"capture_panics" json:"capture_panics"`
	MaxBreadcrumbs       int               `yaml:"max_breadcrumbs" json:"max_breadcrumbs"`
	IgnoreErrors         []string          `yaml:"ignore_errors" json:"ignore_errors"`
	HTTPIgnorePaths      []string          `yaml:"http_ignore_paths" json:"http_ignore_paths"`
	HTTPIgnoreStatusCode []int             `yaml:"http_ignore_status_codes" json:"http_ignore_status_codes"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level              string   `yaml:"level" json:"level"`
	Development        bool     `yaml:"development" json:"development"`
	Encoding           string   `yaml:"encoding" json:"encoding"`
	OutputPaths        []string `yaml:"output_paths" json:"output_paths"`
	ErrorOutputPaths   []string `yaml:"error_output_paths" json:"error_output_paths"`
	DisableCaller      bool     `yaml:"disable_caller" json:"disable_caller"`
	DisableStacktrace  bool     `yaml:"disable_stacktrace" json:"disable_stacktrace"`
	SamplingEnabled    bool     `yaml:"sampling_enabled" json:"sampling_enabled"`
	SamplingInitial    int      `yaml:"sampling_initial" json:"sampling_initial"`
	SamplingThereafter int      `yaml:"sampling_thereafter" json:"sampling_thereafter"`
}

// PrometheusConfig is now defined in types package to avoid circular dependencies
// Use types.PrometheusConfig instead
type PrometheusConfig = types.PrometheusConfig
type PrometheusBuckets = types.PrometheusBuckets

// NewServerConfig creates a new ServerConfig with default values
func NewServerConfig() *ServerConfig {
	config := &ServerConfig{}
	config.SetDefaults()
	return config
}

// SetDefaults sets default values for all configuration options
func (c *ServerConfig) SetDefaults() {
	if c.ServiceName == "" {
		c.ServiceName = "swit-service"
	}

	// HTTP defaults
	if c.HTTP.Port == "" {
		c.HTTP.Port = "8080"
	}
	if c.HTTP.Address == "" {
		c.HTTP.Address = ":" + c.HTTP.Port
	}
	c.HTTP.Enabled = true
	c.HTTP.EnableReady = true
	c.HTTP.TestMode = false

	// HTTP timeout defaults
	if c.HTTP.ReadTimeout == 0 {
		c.HTTP.ReadTimeout = 30 * time.Second
	}
	if c.HTTP.WriteTimeout == 0 {
		c.HTTP.WriteTimeout = 30 * time.Second
	}
	if c.HTTP.IdleTimeout == 0 {
		c.HTTP.IdleTimeout = 120 * time.Second
	}

	// HTTP middleware defaults
	c.HTTP.Middleware.EnableCORS = true
	c.HTTP.Middleware.EnableAuth = false
	c.HTTP.Middleware.EnableRateLimit = false
	c.HTTP.Middleware.EnableLogging = true
	c.HTTP.Middleware.EnableTimeout = true

	// CORS defaults - use secure development defaults instead of wildcard
	if len(c.HTTP.Middleware.CORSConfig.AllowOrigins) == 0 {
		// Default to localhost origins for development safety
		// Production deployments should explicitly configure allowed origins
		c.HTTP.Middleware.CORSConfig.AllowOrigins = []string{
			"http://localhost:3000", // Common frontend dev server
			"http://localhost:8080", // Common backend dev server
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8080",
		}
	}
	if len(c.HTTP.Middleware.CORSConfig.AllowMethods) == 0 {
		c.HTTP.Middleware.CORSConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(c.HTTP.Middleware.CORSConfig.AllowHeaders) == 0 {
		c.HTTP.Middleware.CORSConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	}

	// Only enable credentials with explicit origins (never with wildcard)
	// This prevents CORS specification violations and CSRF attacks
	c.HTTP.Middleware.CORSConfig.AllowCredentials = true
	if c.HTTP.Middleware.CORSConfig.MaxAge == 0 {
		c.HTTP.Middleware.CORSConfig.MaxAge = 86400 // 24 hours
	}

	// Rate limit defaults
	if c.HTTP.Middleware.RateLimitConfig.RequestsPerSecond == 0 {
		c.HTTP.Middleware.RateLimitConfig.RequestsPerSecond = 100
	}
	if c.HTTP.Middleware.RateLimitConfig.BurstSize == 0 {
		c.HTTP.Middleware.RateLimitConfig.BurstSize = 200
	}
	if c.HTTP.Middleware.RateLimitConfig.WindowSize == 0 {
		c.HTTP.Middleware.RateLimitConfig.WindowSize = time.Minute
	}
	if c.HTTP.Middleware.RateLimitConfig.KeyFunc == "" {
		c.HTTP.Middleware.RateLimitConfig.KeyFunc = "ip"
	}

	// Timeout defaults
	if c.HTTP.Middleware.TimeoutConfig.RequestTimeout == 0 {
		c.HTTP.Middleware.TimeoutConfig.RequestTimeout = 30 * time.Second
	}
	if c.HTTP.Middleware.TimeoutConfig.HandlerTimeout == 0 {
		c.HTTP.Middleware.TimeoutConfig.HandlerTimeout = 25 * time.Second
	}

	// Initialize headers map if nil
	if c.HTTP.Headers == nil {
		c.HTTP.Headers = make(map[string]string)
	}
	if c.HTTP.Middleware.CustomHeaders == nil {
		c.HTTP.Middleware.CustomHeaders = make(map[string]string)
	}

	// gRPC defaults
	if c.GRPC.Port == "" {
		c.GRPC.Port = c.calculateGRPCPort()
	}
	if c.GRPC.Address == "" {
		c.GRPC.Address = ":" + c.GRPC.Port
	}
	c.GRPC.Enabled = true
	c.GRPC.EnableKeepalive = true
	c.GRPC.EnableReflection = true
	c.GRPC.EnableHealthService = true
	c.GRPC.TestMode = false

	// gRPC message size defaults
	if c.GRPC.MaxRecvMsgSize == 0 {
		c.GRPC.MaxRecvMsgSize = 4 * 1024 * 1024 // 4MB
	}
	if c.GRPC.MaxSendMsgSize == 0 {
		c.GRPC.MaxSendMsgSize = 4 * 1024 * 1024 // 4MB
	}

	// gRPC keepalive parameter defaults
	if c.GRPC.KeepaliveParams.MaxConnectionIdle == 0 {
		c.GRPC.KeepaliveParams.MaxConnectionIdle = 15 * time.Second
	}
	if c.GRPC.KeepaliveParams.MaxConnectionAge == 0 {
		c.GRPC.KeepaliveParams.MaxConnectionAge = 30 * time.Second
	}
	if c.GRPC.KeepaliveParams.MaxConnectionAgeGrace == 0 {
		c.GRPC.KeepaliveParams.MaxConnectionAgeGrace = 5 * time.Second
	}
	if c.GRPC.KeepaliveParams.Time == 0 {
		c.GRPC.KeepaliveParams.Time = 5 * time.Second
	}
	if c.GRPC.KeepaliveParams.Timeout == 0 {
		c.GRPC.KeepaliveParams.Timeout = 1 * time.Second
	}

	// gRPC keepalive policy defaults
	if c.GRPC.KeepalivePolicy.MinTime == 0 {
		c.GRPC.KeepalivePolicy.MinTime = 5 * time.Second
	}
	c.GRPC.KeepalivePolicy.PermitWithoutStream = true

	// gRPC interceptor defaults
	c.GRPC.Interceptors.EnableAuth = false
	c.GRPC.Interceptors.EnableLogging = true
	c.GRPC.Interceptors.EnableMetrics = false
	c.GRPC.Interceptors.EnableRecovery = true
	c.GRPC.Interceptors.EnableRateLimit = false

	// gRPC TLS defaults
	c.GRPC.TLS.Enabled = false

	// Discovery defaults
	if c.Discovery.Address == "" {
		c.Discovery.Address = "127.0.0.1:8500"
	}
	if c.Discovery.ServiceName == "" {
		c.Discovery.ServiceName = c.ServiceName
	}
	c.Discovery.Enabled = true
	if len(c.Discovery.Tags) == 0 {
		c.Discovery.Tags = []string{"v1"}
	}

	// Set discovery failure mode defaults
	if c.Discovery.FailureMode == "" {
		c.Discovery.FailureMode = DiscoveryFailureModeGraceful // Default to graceful for backward compatibility
	}
	c.Discovery.HealthCheckRequired = false // Default to false for backward compatibility
	if c.Discovery.RegistrationTimeout == 0 {
		c.Discovery.RegistrationTimeout = 30 * time.Second // Default timeout for discovery operations
	}

	// Middleware defaults
	c.Middleware.EnableCORS = true
	c.Middleware.EnableAuth = false
	c.Middleware.EnableRateLimit = false
	c.Middleware.EnableLogging = true

	// Messaging defaults
	c.Messaging.Enabled = false // Default to disabled for backward compatibility
	c.Messaging.DefaultBroker = ""
	if c.Messaging.Brokers == nil {
		c.Messaging.Brokers = make(map[string]BrokerConfig)
	}

	// Messaging connection defaults
	if c.Messaging.Connection.Timeout == 0 {
		c.Messaging.Connection.Timeout = 30 * time.Second
	}
	if c.Messaging.Connection.KeepAlive == 0 {
		c.Messaging.Connection.KeepAlive = 30 * time.Second
	}
	if c.Messaging.Connection.MaxAttempts == 0 {
		c.Messaging.Connection.MaxAttempts = 3
	}
	if c.Messaging.Connection.RetryInterval == 0 {
		c.Messaging.Connection.RetryInterval = 5 * time.Second
	}
	if c.Messaging.Connection.PoolSize == 0 {
		c.Messaging.Connection.PoolSize = 10
	}
	if c.Messaging.Connection.IdleTimeout == 0 {
		c.Messaging.Connection.IdleTimeout = 5 * time.Minute
	}

	// Messaging security defaults
	c.Messaging.Security.EnableEncryption = false
	c.Messaging.Security.EnableAuthentication = false
	c.Messaging.Security.EnableAuthorization = false
	c.Messaging.Security.MessageSigning = false

	// Messaging performance defaults
	if c.Messaging.Performance.BatchSize == 0 {
		c.Messaging.Performance.BatchSize = 100
	}
	if c.Messaging.Performance.BatchTimeout == 0 {
		c.Messaging.Performance.BatchTimeout = 100 * time.Millisecond
	}
	if c.Messaging.Performance.BufferSize == 0 {
		c.Messaging.Performance.BufferSize = 1000
	}
	if c.Messaging.Performance.Concurrency == 0 {
		c.Messaging.Performance.Concurrency = 1
	}
	if c.Messaging.Performance.PrefetchCount == 0 {
		c.Messaging.Performance.PrefetchCount = 10
	}
	c.Messaging.Performance.CompressionEnabled = false

	// Messaging monitoring defaults
	c.Messaging.Monitoring.Enabled = true
	c.Messaging.Monitoring.MetricsEnabled = true
	c.Messaging.Monitoring.TracingEnabled = false
	c.Messaging.Monitoring.HealthCheckEnabled = true
	if c.Messaging.Monitoring.HealthCheckInterval == 0 {
		c.Messaging.Monitoring.HealthCheckInterval = 30 * time.Second
	}

	// Apply messaging environment overrides
	c.Messaging.ApplyEnvironmentOverrides()

	// Sentry defaults
	c.Sentry.Enabled = false
	c.Sentry.Environment = "development"
	c.Sentry.SampleRate = 1.0
	c.Sentry.TracesSampleRate = 0.0
	c.Sentry.AttachStacktrace = true
	c.Sentry.EnableTracing = false
	c.Sentry.Debug = false
	c.Sentry.BeforeSend = false
	c.Sentry.IntegrateHTTP = true
	c.Sentry.IntegrateGRPC = true
	c.Sentry.CapturePanics = true
	c.Sentry.MaxBreadcrumbs = 30
	if c.Sentry.Tags == nil {
		c.Sentry.Tags = make(map[string]string)
	}
	if len(c.Sentry.HTTPIgnoreStatusCode) == 0 {
		// Default to not capture client errors (4xx)
		c.Sentry.HTTPIgnoreStatusCode = []int{400, 401, 403, 404}
	}

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Encoding == "" {
		if c.Logging.Development {
			c.Logging.Encoding = "console"
		} else {
			c.Logging.Encoding = "json"
		}
	}
	if len(c.Logging.OutputPaths) == 0 {
		c.Logging.OutputPaths = []string{"stdout"}
	}
	if len(c.Logging.ErrorOutputPaths) == 0 {
		c.Logging.ErrorOutputPaths = []string{"stderr"}
	}
	if c.Logging.SamplingEnabled && c.Logging.SamplingInitial == 0 {
		c.Logging.SamplingInitial = 100
	}
	if c.Logging.SamplingEnabled && c.Logging.SamplingThereafter == 0 {
		c.Logging.SamplingThereafter = 100
	}

	// Prometheus defaults
	c.Prometheus.Enabled = true
	if c.Prometheus.Endpoint == "" {
		c.Prometheus.Endpoint = "/metrics"
	}
	if c.Prometheus.Namespace == "" {
		c.Prometheus.Namespace = "swit"
	}
	if c.Prometheus.Subsystem == "" {
		c.Prometheus.Subsystem = "server"
	}
	if len(c.Prometheus.Buckets.Duration) == 0 {
		c.Prometheus.Buckets.Duration = []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10}
	}
	if len(c.Prometheus.Buckets.Size) == 0 {
		c.Prometheus.Buckets.Size = []float64{100, 1000, 10000, 100000, 1000000}
	}
	if c.Prometheus.Labels == nil {
		c.Prometheus.Labels = make(map[string]string)
	}
	if c.Prometheus.CardinalityLimit == 0 {
		c.Prometheus.CardinalityLimit = 10000 // Default limit to prevent memory exhaustion
	}

	// Tracing configuration
	if c.Tracing.Enabled {
		// Set service name if not specified
		if c.Tracing.ServiceName == "" {
			c.Tracing.ServiceName = c.ServiceName
		}
		// Apply environment variable overrides when tracing is enabled
		c.Tracing.ApplyEnvironmentOverrides()
	} else if c.ServiceName != "swit-service" {
		// For non-default service names, set the tracing service name but don't enable tracing
		if c.Tracing.ServiceName == "" {
			c.Tracing.ServiceName = c.ServiceName
		}
	}

	// Server defaults
	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = 5 * time.Second
	}
}

// Validate validates the configuration and returns any errors
func (c *ServerConfig) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}

	// Validate HTTP configuration
	if c.HTTP.Enabled {
		if c.HTTP.Port == "" {
			return fmt.Errorf("http.port is required when HTTP is enabled")
		}
		if err := c.validatePort(c.HTTP.Port, "http.port"); err != nil {
			return err
		}

		// Validate HTTP timeouts
		if c.HTTP.ReadTimeout <= 0 {
			return fmt.Errorf("http.read_timeout must be positive")
		}
		if c.HTTP.WriteTimeout <= 0 {
			return fmt.Errorf("http.write_timeout must be positive")
		}
		if c.HTTP.IdleTimeout <= 0 {
			return fmt.Errorf("http.idle_timeout must be positive")
		}

		// Validate rate limit configuration
		if c.HTTP.Middleware.EnableRateLimit {
			if c.HTTP.Middleware.RateLimitConfig.RequestsPerSecond <= 0 {
				return fmt.Errorf("http.middleware.rate_limit.requests_per_second must be positive")
			}
			if c.HTTP.Middleware.RateLimitConfig.BurstSize <= 0 {
				return fmt.Errorf("http.middleware.rate_limit.burst_size must be positive")
			}
			if c.HTTP.Middleware.RateLimitConfig.WindowSize <= 0 {
				return fmt.Errorf("http.middleware.rate_limit.window_size must be positive")
			}
			validKeyFuncs := map[string]bool{"ip": true, "user": true, "custom": true}
			if !validKeyFuncs[c.HTTP.Middleware.RateLimitConfig.KeyFunc] {
				return fmt.Errorf("http.middleware.rate_limit.key_func must be one of: ip, user, custom")
			}
		}

		// Validate timeout configuration
		if c.HTTP.Middleware.EnableTimeout {
			if c.HTTP.Middleware.TimeoutConfig.RequestTimeout <= 0 {
				return fmt.Errorf("http.middleware.timeout.request_timeout must be positive")
			}
			if c.HTTP.Middleware.TimeoutConfig.HandlerTimeout <= 0 {
				return fmt.Errorf("http.middleware.timeout.handler_timeout must be positive")
			}
		}

		// Validate CORS configuration
		if c.HTTP.Middleware.EnableCORS {
			if err := c.validateCORSConfig(); err != nil {
				return err
			}
		}
	}

	// Validate gRPC configuration
	if c.GRPC.Enabled {
		if c.GRPC.Port == "" {
			return fmt.Errorf("grpc.port is required when gRPC is enabled")
		}
		if err := c.validatePort(c.GRPC.Port, "grpc.port"); err != nil {
			return err
		}

		// Validate message sizes
		if c.GRPC.MaxRecvMsgSize <= 0 {
			return fmt.Errorf("grpc.max_recv_msg_size must be positive")
		}
		if c.GRPC.MaxSendMsgSize <= 0 {
			return fmt.Errorf("grpc.max_send_msg_size must be positive")
		}

		// Validate keepalive parameters if enabled
		if c.GRPC.EnableKeepalive {
			if c.GRPC.KeepaliveParams.MaxConnectionIdle <= 0 {
				return fmt.Errorf("grpc.keepalive_params.max_connection_idle must be positive")
			}
			if c.GRPC.KeepaliveParams.MaxConnectionAge <= 0 {
				return fmt.Errorf("grpc.keepalive_params.max_connection_age must be positive")
			}
			if c.GRPC.KeepaliveParams.MaxConnectionAgeGrace <= 0 {
				return fmt.Errorf("grpc.keepalive_params.max_connection_age_grace must be positive")
			}
			if c.GRPC.KeepaliveParams.Time <= 0 {
				return fmt.Errorf("grpc.keepalive_params.time must be positive")
			}
			if c.GRPC.KeepaliveParams.Timeout <= 0 {
				return fmt.Errorf("grpc.keepalive_params.timeout must be positive")
			}
			if c.GRPC.KeepalivePolicy.MinTime <= 0 {
				return fmt.Errorf("grpc.keepalive_policy.min_time must be positive")
			}
		}

		// Validate TLS configuration if enabled
		if c.GRPC.TLS.Enabled {
			if c.GRPC.TLS.CertFile == "" {
				return fmt.Errorf("grpc.tls.cert_file is required when TLS is enabled")
			}
			if c.GRPC.TLS.KeyFile == "" {
				return fmt.Errorf("grpc.tls.key_file is required when TLS is enabled")
			}
		}
	}

	// Validate that at least one transport is enabled
	if !c.HTTP.Enabled && !c.GRPC.Enabled {
		return fmt.Errorf("at least one transport (HTTP or gRPC) must be enabled")
	}

	// Validate ports are different if both transports are enabled
	// Allow port 0 (system assigns available ports) which will be different
	if c.HTTP.Enabled && c.GRPC.Enabled && c.HTTP.Port == c.GRPC.Port && c.HTTP.Port != "0" {
		return fmt.Errorf("http.port and grpc.port must be different")
	}

	// Validate discovery configuration
	if c.Discovery.Enabled {
		if c.Discovery.Address == "" {
			return fmt.Errorf("discovery.address is required when discovery is enabled")
		}
		if c.Discovery.ServiceName == "" {
			return fmt.Errorf("discovery.service_name is required when discovery is enabled")
		}

		// Validate failure mode
		validFailureModes := map[DiscoveryFailureMode]bool{
			DiscoveryFailureModeGraceful: true,
			DiscoveryFailureModeFailFast: true,
			DiscoveryFailureModeStrict:   true,
		}
		if !validFailureModes[c.Discovery.FailureMode] {
			return fmt.Errorf("discovery.failure_mode must be one of: graceful, fail_fast, strict")
		}

		// Validate registration timeout
		if c.Discovery.RegistrationTimeout <= 0 {
			return fmt.Errorf("discovery.registration_timeout must be positive")
		}
		if c.Discovery.RegistrationTimeout > 5*time.Minute {
			return fmt.Errorf("discovery.registration_timeout should not exceed 5 minutes")
		}
	}

	// Validate messaging configuration
	if c.Messaging.Enabled {
		// Validate default broker is specified if messaging is enabled
		if c.Messaging.DefaultBroker == "" && len(c.Messaging.Brokers) > 0 {
			return fmt.Errorf("messaging.default_broker is required when brokers are configured")
		}

		// Validate default broker exists in brokers map
		if c.Messaging.DefaultBroker != "" {
			if _, exists := c.Messaging.Brokers[c.Messaging.DefaultBroker]; !exists {
				return fmt.Errorf("messaging.default_broker '%s' not found in brokers configuration", c.Messaging.DefaultBroker)
			}
		}

		// Validate broker configurations
		for name, brokerConfig := range c.Messaging.Brokers {
			if err := c.validateBrokerConfig(name, &brokerConfig); err != nil {
				return fmt.Errorf("messaging.brokers.%s validation failed: %w", name, err)
			}
		}

		// Validate connection configuration
		if c.Messaging.Connection.Timeout <= 0 {
			return fmt.Errorf("messaging.connection.timeout must be positive")
		}
		if c.Messaging.Connection.MaxAttempts <= 0 {
			return fmt.Errorf("messaging.connection.max_attempts must be positive")
		}
		if c.Messaging.Connection.RetryInterval <= 0 {
			return fmt.Errorf("messaging.connection.retry_interval must be positive")
		}
		if c.Messaging.Connection.PoolSize <= 0 {
			return fmt.Errorf("messaging.connection.pool_size must be positive")
		}
		if c.Messaging.Connection.IdleTimeout < 0 {
			return fmt.Errorf("messaging.connection.idle_timeout cannot be negative")
		}

		// Validate performance configuration
		if c.Messaging.Performance.BatchSize <= 0 {
			return fmt.Errorf("messaging.performance.batch_size must be positive")
		}
		if c.Messaging.Performance.BatchTimeout <= 0 {
			return fmt.Errorf("messaging.performance.batch_timeout must be positive")
		}
		if c.Messaging.Performance.BufferSize <= 0 {
			return fmt.Errorf("messaging.performance.buffer_size must be positive")
		}
		if c.Messaging.Performance.Concurrency <= 0 {
			return fmt.Errorf("messaging.performance.concurrency must be positive")
		}
		if c.Messaging.Performance.PrefetchCount <= 0 {
			return fmt.Errorf("messaging.performance.prefetch_count must be positive")
		}

		// Validate monitoring configuration
		if c.Messaging.Monitoring.HealthCheckInterval <= 0 {
			return fmt.Errorf("messaging.monitoring.health_check_interval must be positive")
		}
	}

	// Validate Sentry configuration
	if c.Sentry.Enabled {
		if c.Sentry.DSN == "" {
			return fmt.Errorf("sentry.dsn is required when Sentry is enabled")
		}

		// Validate sample rates
		if c.Sentry.SampleRate < 0.0 || c.Sentry.SampleRate > 1.0 {
			return fmt.Errorf("sentry.sample_rate must be between 0.0 and 1.0")
		}
		if c.Sentry.TracesSampleRate < 0.0 || c.Sentry.TracesSampleRate > 1.0 {
			return fmt.Errorf("sentry.traces_sample_rate must be between 0.0 and 1.0")
		}

		// Validate environment
		if c.Sentry.Environment == "" {
			return fmt.Errorf("sentry.environment cannot be empty when Sentry is enabled")
		}

		// Validate max breadcrumbs
		if c.Sentry.MaxBreadcrumbs < 0 || c.Sentry.MaxBreadcrumbs > 100 {
			return fmt.Errorf("sentry.max_breadcrumbs must be between 0 and 100")
		}

		// Validate HTTP ignore status codes
		for _, code := range c.Sentry.HTTPIgnoreStatusCode {
			if code < 100 || code > 599 {
				return fmt.Errorf("sentry.http_ignore_status_codes contains invalid HTTP status code: %d", code)
			}
		}
	}

	// Validate logging configuration
	// Set default level if empty
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
		"dpanic": true, "panic": true, "fatal": true,
	}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error, dpanic, panic, fatal")
	}

	// Set default encoding if empty
	if c.Logging.Encoding == "" {
		if c.Logging.Development {
			c.Logging.Encoding = "console"
		} else {
			c.Logging.Encoding = "json"
		}
	}
	validEncodings := map[string]bool{"json": true, "console": true}
	if !validEncodings[c.Logging.Encoding] {
		return fmt.Errorf("logging.encoding must be one of: json, console")
	}

	if c.Logging.SamplingEnabled {
		if c.Logging.SamplingInitial <= 0 {
			return fmt.Errorf("logging.sampling_initial must be positive when sampling is enabled")
		}
		if c.Logging.SamplingThereafter <= 0 {
			return fmt.Errorf("logging.sampling_thereafter must be positive when sampling is enabled")
		}
	}

	// Validate Prometheus configuration
	if c.Prometheus.Enabled {
		if c.Prometheus.Endpoint == "" {
			return fmt.Errorf("prometheus.endpoint cannot be empty when Prometheus is enabled")
		}
		if !strings.HasPrefix(c.Prometheus.Endpoint, "/") {
			return fmt.Errorf("prometheus.endpoint must start with '/'")
		}
		if c.Prometheus.Namespace == "" {
			return fmt.Errorf("prometheus.namespace cannot be empty when Prometheus is enabled")
		}
		if c.Prometheus.Subsystem == "" {
			return fmt.Errorf("prometheus.subsystem cannot be empty when Prometheus is enabled")
		}

		// Validate buckets
		if len(c.Prometheus.Buckets.Duration) == 0 {
			return fmt.Errorf("prometheus.buckets.duration cannot be empty")
		}
		if len(c.Prometheus.Buckets.Size) == 0 {
			return fmt.Errorf("prometheus.buckets.size cannot be empty")
		}

		// Validate that buckets are sorted
		for i := 1; i < len(c.Prometheus.Buckets.Duration); i++ {
			if c.Prometheus.Buckets.Duration[i] <= c.Prometheus.Buckets.Duration[i-1] {
				return fmt.Errorf("prometheus.buckets.duration must be sorted in ascending order")
			}
		}
		for i := 1; i < len(c.Prometheus.Buckets.Size); i++ {
			if c.Prometheus.Buckets.Size[i] <= c.Prometheus.Buckets.Size[i-1] {
				return fmt.Errorf("prometheus.buckets.size must be sorted in ascending order")
			}
		}

		// Validate cardinality limit
		if c.Prometheus.CardinalityLimit <= 0 {
			return fmt.Errorf("prometheus.cardinality_limit must be positive")
		}
		if c.Prometheus.CardinalityLimit > 100000 {
			return fmt.Errorf("prometheus.cardinality_limit exceeds recommended maximum of 100000")
		}
	}

	// Validate tracing configuration
	if c.Tracing.Enabled {
		if err := c.Tracing.Validate(); err != nil {
			return fmt.Errorf("tracing configuration invalid: %w", err)
		}
	}

	// Validate shutdown timeout
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown_timeout must be positive")
	}

	return nil
}

// validatePort validates that a port string is a valid port number
func (c *ServerConfig) validatePort(port, fieldName string) error {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("%s must be a valid port number: %w", fieldName, err)
	}
	// Allow port 0 (system assigns available port) for testing and dynamic allocation
	if portNum < 0 || portNum > 65535 {
		return fmt.Errorf("%s must be between 0 and 65535, got %d", fieldName, portNum)
	}
	return nil
}

// calculateGRPCPort calculates a default gRPC port based on HTTP port
// Uses the same logic as the existing transport package
func (c *ServerConfig) calculateGRPCPort() string {
	if c.HTTP.Port == "" {
		return "9080" // Default if no HTTP port set
	}

	httpPort, err := strconv.Atoi(c.HTTP.Port)
	if err != nil {
		return "9080" // Fallback to default
	}

	// Add 1000 to HTTP port for gRPC (e.g., 8080 -> 9080)
	grpcPort := httpPort + 1000
	return strconv.Itoa(grpcPort)
}

// GetHTTPAddress returns the full HTTP address with colon prefix
func (c *ServerConfig) GetHTTPAddress() string {
	if !c.HTTP.Enabled {
		return ""
	}
	return ":" + c.HTTP.Port
}

// GetGRPCAddress returns the full gRPC address with colon prefix
func (c *ServerConfig) GetGRPCAddress() string {
	if !c.GRPC.Enabled {
		return ""
	}
	return ":" + c.GRPC.Port
}

// IsHTTPEnabled returns true if HTTP transport is enabled
func (c *ServerConfig) IsHTTPEnabled() bool {
	return c.HTTP.Enabled
}

// IsGRPCEnabled returns true if gRPC transport is enabled
func (c *ServerConfig) IsGRPCEnabled() bool {
	return c.GRPC.Enabled
}

// IsDiscoveryEnabled returns true if service discovery is enabled
func (c *ServerConfig) IsDiscoveryEnabled() bool {
	return c.Discovery.Enabled
}

// IsMessagingEnabled returns true if messaging system is enabled
func (c *ServerConfig) IsMessagingEnabled() bool {
	return c.Messaging.Enabled
}

// IsDiscoveryFailureModeFailFast returns true if discovery failure mode is fail_fast or strict
func (c *ServerConfig) IsDiscoveryFailureModeFailFast() bool {
	return c.Discovery.FailureMode == DiscoveryFailureModeFailFast ||
		c.Discovery.FailureMode == DiscoveryFailureModeStrict
}

// IsDiscoveryFailureModeGraceful returns true if discovery failure mode is graceful
func (c *ServerConfig) IsDiscoveryFailureModeGraceful() bool {
	return c.Discovery.FailureMode == DiscoveryFailureModeGraceful
}

// IsDiscoveryFailureModeStrict returns true if discovery failure mode is strict
func (c *ServerConfig) IsDiscoveryFailureModeStrict() bool {
	return c.Discovery.FailureMode == DiscoveryFailureModeStrict
}

// GetDiscoveryFailureModeDescription returns a human-readable description of the failure mode
func (c *ServerConfig) GetDiscoveryFailureModeDescription() string {
	switch c.Discovery.FailureMode {
	case DiscoveryFailureModeGraceful:
		return "graceful degradation - server continues without discovery"
	case DiscoveryFailureModeFailFast:
		return "fail-fast - server startup fails if discovery registration fails"
	case DiscoveryFailureModeStrict:
		return "strict mode - requires discovery health check and fails fast"
	default:
		return "unknown failure mode"
	}
}

// toGRPCKeepaliveParams converts config keepalive params to gRPC keepalive params
func (c *ServerConfig) toGRPCKeepaliveParams() *keepalive.ServerParameters {
	return &keepalive.ServerParameters{
		MaxConnectionIdle:     c.GRPC.KeepaliveParams.MaxConnectionIdle,
		MaxConnectionAge:      c.GRPC.KeepaliveParams.MaxConnectionAge,
		MaxConnectionAgeGrace: c.GRPC.KeepaliveParams.MaxConnectionAgeGrace,
		Time:                  c.GRPC.KeepaliveParams.Time,
		Timeout:               c.GRPC.KeepaliveParams.Timeout,
	}
}

// toGRPCKeepalivePolicy converts config keepalive policy to gRPC keepalive policy
func (c *ServerConfig) toGRPCKeepalivePolicy() *keepalive.EnforcementPolicy {
	return &keepalive.EnforcementPolicy{
		MinTime:             c.GRPC.KeepalivePolicy.MinTime,
		PermitWithoutStream: c.GRPC.KeepalivePolicy.PermitWithoutStream,
	}
}

// validateCORSConfig validates CORS configuration for security compliance
func (c *ServerConfig) validateCORSConfig() error {
	cors := c.HTTP.Middleware.CORSConfig

	// Check for dangerous wildcard + credentials combination (CORS spec violation)
	if cors.AllowCredentials {
		for _, origin := range cors.AllowOrigins {
			if origin == "*" {
				return fmt.Errorf("CORS security violation: cannot use wildcard origin '*' with AllowCredentials=true. This violates RFC 6454 and creates security risks. Use explicit origins instead")
			}
		}
	}

	// Validate that origins are not empty when credentials are enabled
	if cors.AllowCredentials && len(cors.AllowOrigins) == 0 {
		return fmt.Errorf("CORS configuration error: AllowOrigins cannot be empty when AllowCredentials=true")
	}

	// Validate origin format (basic URL validation)
	for i, origin := range cors.AllowOrigins {
		if origin == "" {
			return fmt.Errorf("CORS configuration error: origin at index %d cannot be empty", i)
		}

		// Allow wildcard only when credentials are disabled
		if origin == "*" && cors.AllowCredentials {
			return fmt.Errorf("CORS security violation: wildcard origin '*' detected with AllowCredentials=true at index %d", i)
		}

		// Basic URL format validation for non-wildcard origins
		if origin != "*" {
			if len(origin) < 7 || (!strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://")) {
				return fmt.Errorf("CORS configuration error: invalid origin format at index %d: %s (must start with http:// or https://)", i, origin)
			}
		}
	}

	// Validate MaxAge is reasonable (prevent excessive caching)
	if cors.MaxAge < 0 {
		return fmt.Errorf("CORS configuration error: MaxAge cannot be negative")
	}
	if cors.MaxAge > 86400*7 { // 7 days
		return fmt.Errorf("CORS configuration warning: MaxAge %d seconds exceeds recommended maximum of 7 days", cors.MaxAge)
	}

	return nil
}

// validateBrokerConfig validates a specific broker configuration
func (c *ServerConfig) validateBrokerConfig(name string, brokerConfig *BrokerConfig) error {
	if brokerConfig.Type == "" {
		return fmt.Errorf("broker type is required")
	}

	// Validate broker type
	validBrokerTypes := map[string]bool{
		"kafka":    true,
		"nats":     true,
		"rabbitmq": true,
		"inmemory": true,
	}
	if !validBrokerTypes[brokerConfig.Type] {
		return fmt.Errorf("invalid broker type: %s, must be one of: kafka, nats, rabbitmq, inmemory", brokerConfig.Type)
	}

	// Validate endpoints (except for inmemory broker)
	if brokerConfig.Type != "inmemory" {
		if len(brokerConfig.Endpoints) == 0 {
			return fmt.Errorf("endpoints are required for broker type: %s", brokerConfig.Type)
		}

		for i, endpoint := range brokerConfig.Endpoints {
			if endpoint == "" {
				return fmt.Errorf("endpoint %d cannot be empty", i)
			}
		}
	}

	// Validate authentication if provided
	if brokerConfig.Authentication != nil {
		if err := c.validateMessagingAuthConfig(brokerConfig.Authentication); err != nil {
			return fmt.Errorf("authentication validation failed: %w", err)
		}
	}

	// Validate TLS if provided
	if brokerConfig.TLS != nil {
		if err := c.validateMessagingTLSConfig(brokerConfig.TLS); err != nil {
			return fmt.Errorf("TLS validation failed: %w", err)
		}
	}

	return nil
}

// validateMessagingAuthConfig validates messaging authentication configuration
func (c *ServerConfig) validateMessagingAuthConfig(auth *MessagingAuthConfig) error {
	if auth.Type == "" {
		return fmt.Errorf("authentication type is required")
	}

	validAuthTypes := map[string]bool{
		"none":   true,
		"sasl":   true,
		"oauth2": true,
		"apikey": true,
	}
	if !validAuthTypes[auth.Type] {
		return fmt.Errorf("invalid authentication type: %s, must be one of: none, sasl, oauth2, apikey", auth.Type)
	}

	switch auth.Type {
	case "none":
		// No additional validation needed
	case "sasl":
		if auth.Username == "" {
			return fmt.Errorf("username is required for SASL authentication")
		}
		if auth.Password == "" {
			return fmt.Errorf("password is required for SASL authentication")
		}
	case "oauth2":
		if auth.Token == "" {
			return fmt.Errorf("token is required for OAuth2 authentication")
		}
	case "apikey":
		if auth.APIKey == "" {
			return fmt.Errorf("api_key is required for API key authentication")
		}
	}

	return nil
}

// validateMessagingTLSConfig validates messaging TLS configuration
func (c *ServerConfig) validateMessagingTLSConfig(tls *MessagingTLSConfig) error {
	if !tls.Enabled {
		return nil
	}

	// If cert file is provided, key file must also be provided
	if tls.CertFile != "" && tls.KeyFile == "" {
		return fmt.Errorf("key_file is required when cert_file is specified")
	}

	if tls.KeyFile != "" && tls.CertFile == "" {
		return fmt.Errorf("cert_file is required when key_file is specified")
	}

	return nil
}

// GetMessagingDefaultBroker returns the default broker name
func (c *ServerConfig) GetMessagingDefaultBroker() string {
	return c.Messaging.DefaultBroker
}

// GetMessagingBroker returns the configuration for a specific broker
func (c *ServerConfig) GetMessagingBroker(name string) (BrokerConfig, bool) {
	broker, exists := c.Messaging.Brokers[name]
	return broker, exists
}

// ApplyMessagingEnvironmentOverrides applies environment variable overrides to messaging configuration
// Environment variables follow the pattern: SWIT_MESSAGING_*
func (c *MessagingConfig) ApplyEnvironmentOverrides() {
	if enabled := os.Getenv("SWIT_MESSAGING_ENABLED"); enabled != "" {
		if val, err := strconv.ParseBool(enabled); err == nil {
			c.Enabled = val
		}
	}

	if defaultBroker := os.Getenv("SWIT_MESSAGING_DEFAULT_BROKER"); defaultBroker != "" {
		c.DefaultBroker = defaultBroker
	}

	// Connection overrides
	c.applyConnectionOverrides()

	// Security overrides
	c.applySecurityOverrides()

	// Performance overrides
	c.applyPerformanceOverrides()

	// Monitoring overrides
	c.applyMonitoringOverrides()
}

// applyConnectionOverrides applies connection-related environment overrides
func (c *MessagingConfig) applyConnectionOverrides() {
	if timeout := os.Getenv("SWIT_MESSAGING_CONNECTION_TIMEOUT"); timeout != "" {
		if val, err := time.ParseDuration(timeout); err == nil {
			c.Connection.Timeout = val
		}
	}

	if keepAlive := os.Getenv("SWIT_MESSAGING_CONNECTION_KEEP_ALIVE"); keepAlive != "" {
		if val, err := time.ParseDuration(keepAlive); err == nil {
			c.Connection.KeepAlive = val
		}
	}

	if maxAttempts := os.Getenv("SWIT_MESSAGING_CONNECTION_MAX_ATTEMPTS"); maxAttempts != "" {
		if val, err := strconv.Atoi(maxAttempts); err == nil {
			c.Connection.MaxAttempts = val
		}
	}

	if retryInterval := os.Getenv("SWIT_MESSAGING_CONNECTION_RETRY_INTERVAL"); retryInterval != "" {
		if val, err := time.ParseDuration(retryInterval); err == nil {
			c.Connection.RetryInterval = val
		}
	}

	if poolSize := os.Getenv("SWIT_MESSAGING_CONNECTION_POOL_SIZE"); poolSize != "" {
		if val, err := strconv.Atoi(poolSize); err == nil {
			c.Connection.PoolSize = val
		}
	}

	if idleTimeout := os.Getenv("SWIT_MESSAGING_CONNECTION_IDLE_TIMEOUT"); idleTimeout != "" {
		if val, err := time.ParseDuration(idleTimeout); err == nil {
			c.Connection.IdleTimeout = val
		}
	}
}

// applySecurityOverrides applies security-related environment overrides
func (c *MessagingConfig) applySecurityOverrides() {
	if enableEncryption := os.Getenv("SWIT_MESSAGING_SECURITY_ENABLE_ENCRYPTION"); enableEncryption != "" {
		if val, err := strconv.ParseBool(enableEncryption); err == nil {
			c.Security.EnableEncryption = val
		}
	}

	if enableAuth := os.Getenv("SWIT_MESSAGING_SECURITY_ENABLE_AUTHENTICATION"); enableAuth != "" {
		if val, err := strconv.ParseBool(enableAuth); err == nil {
			c.Security.EnableAuthentication = val
		}
	}

	if enableAuthz := os.Getenv("SWIT_MESSAGING_SECURITY_ENABLE_AUTHORIZATION"); enableAuthz != "" {
		if val, err := strconv.ParseBool(enableAuthz); err == nil {
			c.Security.EnableAuthorization = val
		}
	}

	if messageSigning := os.Getenv("SWIT_MESSAGING_SECURITY_MESSAGE_SIGNING"); messageSigning != "" {
		if val, err := strconv.ParseBool(messageSigning); err == nil {
			c.Security.MessageSigning = val
		}
	}
}

// applyPerformanceOverrides applies performance-related environment overrides
func (c *MessagingConfig) applyPerformanceOverrides() {
	if batchSize := os.Getenv("SWIT_MESSAGING_PERFORMANCE_BATCH_SIZE"); batchSize != "" {
		if val, err := strconv.Atoi(batchSize); err == nil {
			c.Performance.BatchSize = val
		}
	}

	if batchTimeout := os.Getenv("SWIT_MESSAGING_PERFORMANCE_BATCH_TIMEOUT"); batchTimeout != "" {
		if val, err := time.ParseDuration(batchTimeout); err == nil {
			c.Performance.BatchTimeout = val
		}
	}

	if bufferSize := os.Getenv("SWIT_MESSAGING_PERFORMANCE_BUFFER_SIZE"); bufferSize != "" {
		if val, err := strconv.Atoi(bufferSize); err == nil {
			c.Performance.BufferSize = val
		}
	}

	if concurrency := os.Getenv("SWIT_MESSAGING_PERFORMANCE_CONCURRENCY"); concurrency != "" {
		if val, err := strconv.Atoi(concurrency); err == nil {
			c.Performance.Concurrency = val
		}
	}

	if prefetchCount := os.Getenv("SWIT_MESSAGING_PERFORMANCE_PREFETCH_COUNT"); prefetchCount != "" {
		if val, err := strconv.Atoi(prefetchCount); err == nil {
			c.Performance.PrefetchCount = val
		}
	}

	if compressionEnabled := os.Getenv("SWIT_MESSAGING_PERFORMANCE_COMPRESSION_ENABLED"); compressionEnabled != "" {
		if val, err := strconv.ParseBool(compressionEnabled); err == nil {
			c.Performance.CompressionEnabled = val
		}
	}
}

// applyMonitoringOverrides applies monitoring-related environment overrides
func (c *MessagingConfig) applyMonitoringOverrides() {
	if enabled := os.Getenv("SWIT_MESSAGING_MONITORING_ENABLED"); enabled != "" {
		if val, err := strconv.ParseBool(enabled); err == nil {
			c.Monitoring.Enabled = val
		}
	}

	if metricsEnabled := os.Getenv("SWIT_MESSAGING_MONITORING_METRICS_ENABLED"); metricsEnabled != "" {
		if val, err := strconv.ParseBool(metricsEnabled); err == nil {
			c.Monitoring.MetricsEnabled = val
		}
	}

	if tracingEnabled := os.Getenv("SWIT_MESSAGING_MONITORING_TRACING_ENABLED"); tracingEnabled != "" {
		if val, err := strconv.ParseBool(tracingEnabled); err == nil {
			c.Monitoring.TracingEnabled = val
		}
	}

	if healthCheckEnabled := os.Getenv("SWIT_MESSAGING_MONITORING_HEALTH_CHECK_ENABLED"); healthCheckEnabled != "" {
		if val, err := strconv.ParseBool(healthCheckEnabled); err == nil {
			c.Monitoring.HealthCheckEnabled = val
		}
	}

	if healthCheckInterval := os.Getenv("SWIT_MESSAGING_MONITORING_HEALTH_CHECK_INTERVAL"); healthCheckInterval != "" {
		if val, err := time.ParseDuration(healthCheckInterval); err == nil {
			c.Monitoring.HealthCheckInterval = val
		}
	}
}
