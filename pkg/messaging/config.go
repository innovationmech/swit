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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
    "runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration is a custom type that can unmarshal duration strings from JSON/YAML
type Duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

// MarshalYAML implements yaml.Marshaler
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// MarshalJSON implements json.Marshaler
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// String returns the string representation
func (d Duration) String() string {
	return time.Duration(d).String()
}

// ToDuration converts to time.Duration
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

// BrokerConfig defines the configuration for a message broker connection.
type BrokerConfig struct {
	// Type specifies the broker type (kafka, nats, rabbitmq, inmemory)
	Type BrokerType `yaml:"type" json:"type" validate:"required,oneof=kafka nats rabbitmq inmemory"`

	// Endpoints contains the list of broker endpoints
	Endpoints []string `yaml:"endpoints" json:"endpoints" validate:"required,min=1"`

	// Connection configuration
	Connection ConnectionConfig `yaml:"connection" json:"connection"`

	// Authentication configuration (optional)
	Authentication *AuthConfig `yaml:"authentication,omitempty" json:"authentication,omitempty"`

	// TLS configuration (optional)
	TLS *TLSConfig `yaml:"tls,omitempty" json:"tls,omitempty"`

	// Retry configuration
	Retry RetryConfig `yaml:"retry" json:"retry"`

	// Monitoring configuration
	Monitoring MonitoringConfig `yaml:"monitoring" json:"monitoring"`

	// Audit logging configuration
	Audit *MessagingAuditConfig `yaml:"audit,omitempty" json:"audit,omitempty"`

	// Broker-specific configuration
	Extra map[string]interface{} `yaml:"extra,omitempty" json:"extra,omitempty"`
}

// Validate validates the broker configuration.
func (c *BrokerConfig) Validate() error {
	if !c.Type.IsValid() {
		return NewConfigError(fmt.Sprintf("invalid broker type: %s", c.Type), nil)
	}

	if len(c.Endpoints) == 0 {
		return NewConfigError("at least one endpoint must be specified", nil)
	}

	for i, endpoint := range c.Endpoints {
		if endpoint == "" {
			return NewConfigError(fmt.Sprintf("endpoint %d cannot be empty", i), nil)
		}
	}

	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("connection config validation failed: %w", err)
	}

	if c.Authentication != nil {
		if err := c.Authentication.Validate(); err != nil {
			return fmt.Errorf("authentication config validation failed: %w", err)
		}
	}

	if c.TLS != nil {
		if err := c.TLS.Validate(); err != nil {
			return fmt.Errorf("TLS config validation failed: %w", err)
		}
	}

	if err := c.Retry.Validate(); err != nil {
		return fmt.Errorf("retry config validation failed: %w", err)
	}

	if c.Audit != nil {
		if err := c.Audit.Validate(); err != nil {
			return fmt.Errorf("audit config validation failed: %w", err)
		}
	}

	return nil
}

// SetDefaults sets default values for the broker configuration.
func (c *BrokerConfig) SetDefaults() {
	c.Connection.SetDefaults()
	c.Retry.SetDefaults()
	c.Monitoring.SetDefaults()

	if c.Audit != nil {
		c.Audit.SetDefaults()
	}
}

// ConnectionConfig defines connection-specific settings.
type ConnectionConfig struct {
	// Timeout for establishing connections
	Timeout time.Duration `yaml:"timeout" json:"timeout" default:"10s"`

	// KeepAlive interval for connection health checks
	KeepAlive time.Duration `yaml:"keep_alive" json:"keep_alive" default:"30s"`

	// MaxAttempts for connection establishment
	MaxAttempts int `yaml:"max_attempts" json:"max_attempts" default:"3"`

	// PoolSize for connection pooling
	PoolSize int `yaml:"pool_size" json:"pool_size" default:"10"`

	// IdleTimeout for idle connections
	IdleTimeout time.Duration `yaml:"idle_timeout" json:"idle_timeout" default:"5m"`
}

// UnmarshalJSON implements json.Unmarshaler for ConnectionConfig
func (c *ConnectionConfig) UnmarshalJSON(data []byte) error {
	// Use a temporary struct with string fields for duration parsing
	type TempConnectionConfig struct {
		Timeout     string `json:"timeout"`
		KeepAlive   string `json:"keep_alive"`
		MaxAttempts int    `json:"max_attempts"`
		PoolSize    int    `json:"pool_size"`
		IdleTimeout string `json:"idle_timeout"`
	}

	var temp TempConnectionConfig
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Parse durations
	if temp.Timeout != "" {
		timeout, err := time.ParseDuration(temp.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout duration: %w", err)
		}
		c.Timeout = timeout
	}

	if temp.KeepAlive != "" {
		keepAlive, err := time.ParseDuration(temp.KeepAlive)
		if err != nil {
			return fmt.Errorf("invalid keep_alive duration: %w", err)
		}
		c.KeepAlive = keepAlive
	}

	if temp.IdleTimeout != "" {
		idleTimeout, err := time.ParseDuration(temp.IdleTimeout)
		if err != nil {
			return fmt.Errorf("invalid idle_timeout duration: %w", err)
		}
		c.IdleTimeout = idleTimeout
	}

	// Set other fields
	c.MaxAttempts = temp.MaxAttempts
	c.PoolSize = temp.PoolSize

	return nil
}

// Validate validates the connection configuration.
func (c *ConnectionConfig) Validate() error {
	if c.Timeout <= 0 {
		return NewConfigError("connection timeout must be positive", nil)
	}

	if c.KeepAlive < 0 {
		return NewConfigError("keep alive cannot be negative", nil)
	}

	if c.MaxAttempts <= 0 {
		return NewConfigError("max attempts must be positive", nil)
	}

	if c.PoolSize <= 0 {
		return NewConfigError("pool size must be positive", nil)
	}

	if c.IdleTimeout < 0 {
		return NewConfigError("idle timeout cannot be negative", nil)
	}

	return nil
}

// SetDefaults sets default values for the connection configuration.
func (c *ConnectionConfig) SetDefaults() {
	if c.Timeout == 0 {
		c.Timeout = 10 * time.Second
	}
	if c.KeepAlive == 0 {
		c.KeepAlive = 30 * time.Second
	}
	if c.MaxAttempts == 0 {
		c.MaxAttempts = 3
	}
	if c.PoolSize == 0 {
		c.PoolSize = 10
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = 5 * time.Minute
	}
}

// AuthType represents the authentication type.
type AuthType string

const (
	// AuthTypeNone indicates no authentication
	AuthTypeNone AuthType = "none"

	// AuthTypeSASL indicates SASL authentication
	AuthTypeSASL AuthType = "sasl"

	// AuthTypeOAuth2 indicates OAuth 2.0 authentication
	AuthTypeOAuth2 AuthType = "oauth2"

	// AuthTypeAPIKey indicates API key authentication
	AuthTypeAPIKey AuthType = "apikey"

	// AuthTypeJWT indicates JWT token authentication
	AuthTypeJWT AuthType = "jwt"

	// AuthTypeCertificate indicates certificate-based authentication
	AuthTypeCertificate AuthType = "certificate"
)

// AuthConfig defines authentication settings.
type AuthConfig struct {
	// Type specifies the authentication type
	Type AuthType `yaml:"type" json:"type" validate:"required,oneof=none sasl oauth2 apikey jwt certificate"`

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

	// OAuth2-specific settings
	ClientID     string   `yaml:"client_id,omitempty" json:"client_id,omitempty"`
	ClientSecret string   `yaml:"client_secret,omitempty" json:"client_secret,omitempty"`
	TokenURL     string   `yaml:"token_url,omitempty" json:"token_url,omitempty"`
	Scopes       []string `yaml:"scopes,omitempty" json:"scopes,omitempty"`

	// Certificate authentication settings
	CertFile string `yaml:"cert_file,omitempty" json:"cert_file,omitempty"`
	KeyFile  string `yaml:"key_file,omitempty" json:"key_file,omitempty"`
}

// Validate validates the authentication configuration.
func (c *AuthConfig) Validate() error {
	switch c.Type {
	case AuthTypeNone:
		// No validation needed
	case AuthTypeSASL:
		if c.Username == "" {
			return NewConfigError("username is required for SASL authentication", nil)
		}
		if c.Password == "" {
			return NewConfigError("password is required for SASL authentication", nil)
		}
	case AuthTypeOAuth2:
		if c.ClientID == "" {
			return NewConfigError("client_id is required for OAuth2 authentication", nil)
		}
		if c.ClientSecret == "" {
			return NewConfigError("client_secret is required for OAuth2 authentication", nil)
		}
		if c.TokenURL == "" {
			return NewConfigError("token_url is required for OAuth2 authentication", nil)
		}
	case AuthTypeAPIKey:
		if c.APIKey == "" {
			return NewConfigError("api_key is required for API key authentication", nil)
		}
	case AuthTypeJWT:
		if c.Token == "" {
			return NewConfigError("token is required for JWT authentication", nil)
		}
	case AuthTypeCertificate:
		// Certificate validation would require additional certificate-specific fields
		// This is a basic validation
		if c.CertFile == "" && c.KeyFile == "" {
			return NewConfigError("certificate or key file is required for certificate authentication", nil)
		}
	default:
		return NewConfigError(fmt.Sprintf("unsupported auth type: %s", c.Type), nil)
	}

	return nil
}

// TLSConfig defines TLS settings.
type TLSConfig struct {
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

// Validate validates the TLS configuration.
func (c *TLSConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	// If cert file is provided, key file must also be provided
	if c.CertFile != "" && c.KeyFile == "" {
		return NewConfigError("key_file is required when cert_file is specified", nil)
	}

	if c.KeyFile != "" && c.CertFile == "" {
		return NewConfigError("cert_file is required when key_file is specified", nil)
	}

	return nil
}

// RetryConfig defines retry settings.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int `yaml:"max_attempts" json:"max_attempts" default:"3"`

	// InitialDelay is the initial delay between retries
	InitialDelay time.Duration `yaml:"initial_delay" json:"initial_delay" default:"1s"`

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration `yaml:"max_delay" json:"max_delay" default:"30s"`

	// Multiplier for exponential backoff
	Multiplier float64 `yaml:"multiplier" json:"multiplier" default:"2.0"`

	// Jitter adds randomness to delays to prevent thundering herd
	Jitter float64 `yaml:"jitter" json:"jitter" default:"0.1"`
}

// UnmarshalJSON implements json.Unmarshaler for RetryConfig
func (r *RetryConfig) UnmarshalJSON(data []byte) error {
	// Use a temporary struct with string fields for duration parsing
	type TempRetryConfig struct {
		MaxAttempts  int     `json:"max_attempts"`
		InitialDelay string  `json:"initial_delay"`
		MaxDelay     string  `json:"max_delay"`
		Multiplier   float64 `json:"multiplier"`
		Jitter       float64 `json:"jitter"`
	}

	var temp TempRetryConfig
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Parse durations
	if temp.InitialDelay != "" {
		initialDelay, err := time.ParseDuration(temp.InitialDelay)
		if err != nil {
			return fmt.Errorf("invalid initial_delay duration: %w", err)
		}
		r.InitialDelay = initialDelay
	}

	if temp.MaxDelay != "" {
		maxDelay, err := time.ParseDuration(temp.MaxDelay)
		if err != nil {
			return fmt.Errorf("invalid max_delay duration: %w", err)
		}
		r.MaxDelay = maxDelay
	}

	// Set other fields
	r.MaxAttempts = temp.MaxAttempts
	r.Multiplier = temp.Multiplier
	r.Jitter = temp.Jitter

	return nil
}

// Validate validates the retry configuration.
func (c *RetryConfig) Validate() error {
	if c.MaxAttempts < 0 {
		return NewConfigError("max attempts cannot be negative", nil)
	}

	if c.InitialDelay < 0 {
		return NewConfigError("initial delay cannot be negative", nil)
	}

	if c.MaxDelay < 0 {
		return NewConfigError("max delay cannot be negative", nil)
	}

	if c.MaxDelay < c.InitialDelay {
		return NewConfigError("max delay cannot be less than initial delay", nil)
	}

	if c.Multiplier <= 1.0 {
		return NewConfigError("multiplier must be greater than 1.0", nil)
	}

	if c.Jitter < 0 || c.Jitter > 1.0 {
		return NewConfigError("jitter must be between 0.0 and 1.0", nil)
	}

	return nil
}

// SetDefaults sets default values for the retry configuration.
func (c *RetryConfig) SetDefaults() {
	if c.MaxAttempts == 0 {
		c.MaxAttempts = 3
	}
	if c.InitialDelay == 0 {
		c.InitialDelay = 1 * time.Second
	}
	if c.MaxDelay == 0 {
		c.MaxDelay = 30 * time.Second
	}
	if c.Multiplier == 0 {
		c.Multiplier = 2.0
	}
	if c.Jitter == 0 {
		c.Jitter = 0.1
	}
}

// MonitoringConfig defines monitoring and metrics settings.
type MonitoringConfig struct {
	// Enabled indicates if monitoring is enabled
	Enabled bool `yaml:"enabled" json:"enabled" default:"true"`

	// MetricsInterval for metrics collection
	MetricsInterval time.Duration `yaml:"metrics_interval" json:"metrics_interval" default:"30s"`

	// HealthCheckInterval for health checks
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval" default:"30s"`

	// HealthCheckTimeout for health check operations
	HealthCheckTimeout time.Duration `yaml:"health_check_timeout" json:"health_check_timeout" default:"5s"`
}

// UnmarshalJSON implements json.Unmarshaler for MonitoringConfig
func (m *MonitoringConfig) UnmarshalJSON(data []byte) error {
	// Use a temporary struct with string fields for duration parsing
	type TempMonitoringConfig struct {
		Enabled             bool   `json:"enabled"`
		MetricsInterval     string `json:"metrics_interval"`
		HealthCheckInterval string `json:"health_check_interval"`
		HealthCheckTimeout  string `json:"health_check_timeout"`
	}

	var temp TempMonitoringConfig
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Parse durations
	if temp.MetricsInterval != "" {
		metricsInterval, err := time.ParseDuration(temp.MetricsInterval)
		if err != nil {
			return fmt.Errorf("invalid metrics_interval duration: %w", err)
		}
		m.MetricsInterval = metricsInterval
	}

	if temp.HealthCheckInterval != "" {
		healthCheckInterval, err := time.ParseDuration(temp.HealthCheckInterval)
		if err != nil {
			return fmt.Errorf("invalid health_check_interval duration: %w", err)
		}
		m.HealthCheckInterval = healthCheckInterval
	}

	if temp.HealthCheckTimeout != "" {
		healthCheckTimeout, err := time.ParseDuration(temp.HealthCheckTimeout)
		if err != nil {
			return fmt.Errorf("invalid health_check_timeout duration: %w", err)
		}
		m.HealthCheckTimeout = healthCheckTimeout
	}

	// Set other fields
	m.Enabled = temp.Enabled

	return nil
}

// SetDefaults sets default values for the monitoring configuration.
func (c *MonitoringConfig) SetDefaults() {
	c.Enabled = true
	if c.MetricsInterval == 0 {
		c.MetricsInterval = 30 * time.Second
	}
	if c.HealthCheckInterval == 0 {
		c.HealthCheckInterval = 30 * time.Second
	}
	if c.HealthCheckTimeout == 0 {
		c.HealthCheckTimeout = 5 * time.Second
	}
}

// PublisherConfig defines publisher-specific configuration.
type PublisherConfig struct {
	// Topic is the destination topic/queue
	Topic string `yaml:"topic" json:"topic" validate:"required"`

	// Routing configuration
	Routing RoutingConfig `yaml:"routing" json:"routing"`

	// Batching configuration
	Batching BatchingConfig `yaml:"batching" json:"batching"`

	// Compression type
	Compression CompressionType `yaml:"compression" json:"compression" default:"none"`

	// Async publishing
	Async bool `yaml:"async" json:"async" default:"false"`

	// Transactional publishing
	Transactional bool `yaml:"transactional" json:"transactional" default:"false"`

	// Confirmation settings
	Confirmation ConfirmationConfig `yaml:"confirmation" json:"confirmation"`

	// Retry configuration
	Retry RetryConfig `yaml:"retry" json:"retry"`

	// Timeout settings
	Timeout TimeoutConfig `yaml:"timeout" json:"timeout"`

	// Workers controls the number of parallel workers in the producer pool
	// for high-throughput publishing. Defaults to the number of CPUs.
	Workers int `yaml:"workers" json:"workers"`

	// QueueSize controls the internal queue capacity for asynchronous
	// publishing and back-pressure. Defaults to 10000.
	QueueSize int `yaml:"queue_size" json:"queue_size"`
}

// RoutingConfig defines message routing settings.
type RoutingConfig struct {
	// Strategy for routing messages (round_robin, hash, random)
	Strategy string `yaml:"strategy" json:"strategy" default:"round_robin"`

	// PartitionKey field name for hash-based routing
	PartitionKey string `yaml:"partition_key,omitempty" json:"partition_key,omitempty"`
}

// BatchingConfig defines batching settings.
type BatchingConfig struct {
	// Enabled indicates if batching is enabled
	Enabled bool `yaml:"enabled" json:"enabled" default:"true"`

	// MaxMessages per batch
	MaxMessages int `yaml:"max_messages" json:"max_messages" default:"100"`

	// MaxBytes per batch
	MaxBytes int `yaml:"max_bytes" json:"max_bytes" default:"1048576"`

	// FlushInterval for automatic batch flushing
	FlushInterval time.Duration `yaml:"flush_interval" json:"flush_interval" default:"100ms"`
}

// ConfirmationConfig defines confirmation settings.
type ConfirmationConfig struct {
	// Required indicates if confirmations are required
	Required bool `yaml:"required" json:"required" default:"false"`

	// Timeout for confirmation
	Timeout time.Duration `yaml:"timeout" json:"timeout" default:"5s"`

	// Retries for confirmation
	Retries int `yaml:"retries" json:"retries" default:"3"`
}

// TimeoutConfig defines timeout settings.
type TimeoutConfig struct {
	// Publish timeout
	Publish time.Duration `yaml:"publish" json:"publish" default:"30s"`

	// Flush timeout
	Flush time.Duration `yaml:"flush" json:"flush" default:"30s"`

	// Close timeout
	Close time.Duration `yaml:"close" json:"close" default:"30s"`
}

// SubscriberConfig defines subscriber-specific configuration.
type SubscriberConfig struct {
	// Topics to subscribe to
	Topics []string `yaml:"topics" json:"topics" validate:"required,min=1"`

	// ConsumerGroup name
	ConsumerGroup string `yaml:"consumer_group" json:"consumer_group" validate:"required"`

	// Subscription type (shared, exclusive, failover)
	Type SubscriptionType `yaml:"type" json:"type" default:"shared"`

	// Concurrency level
	Concurrency int `yaml:"concurrency" json:"concurrency" default:"1"`

	// PrefetchCount for message prefetching
	PrefetchCount int `yaml:"prefetch_count" json:"prefetch_count" default:"10"`

	// Processing configuration
	Processing ProcessingConfig `yaml:"processing" json:"processing"`

	// DeadLetter configuration
	DeadLetter DeadLetterConfig `yaml:"dead_letter" json:"dead_letter"`

	// Offset management
	Offset OffsetConfig `yaml:"offset" json:"offset"`

	// Retry configuration
	Retry RetryConfig `yaml:"retry" json:"retry"`
}

// SubscriptionType defines subscription types.
type SubscriptionType string

const (
	// SubscriptionShared allows multiple consumers to share the subscription
	SubscriptionShared SubscriptionType = "shared"

	// SubscriptionExclusive allows only one consumer per subscription
	SubscriptionExclusive SubscriptionType = "exclusive"

	// SubscriptionFailover provides failover behavior for consumers
	SubscriptionFailover SubscriptionType = "failover"
)

// ProcessingConfig defines message processing settings.
type ProcessingConfig struct {
	// MaxProcessingTime per message
	MaxProcessingTime time.Duration `yaml:"max_processing_time" json:"max_processing_time" default:"30s"`

	// AckMode for acknowledgments (auto, manual)
	AckMode AckMode `yaml:"ack_mode" json:"ack_mode" default:"auto"`

	// MaxInFlight messages
	MaxInFlight int `yaml:"max_in_flight" json:"max_in_flight" default:"100"`

	// Ordered processing
	Ordered bool `yaml:"ordered" json:"ordered" default:"false"`
}

// AckMode defines acknowledgment modes.
type AckMode string

const (
	// AckModeAuto automatically acknowledges messages after processing
	AckModeAuto AckMode = "auto"

	// AckModeManual requires explicit message acknowledgment
	AckModeManual AckMode = "manual"
)

// DeadLetterConfig defines dead letter settings.
type DeadLetterConfig struct {
	// Enabled indicates if dead letter is enabled
	Enabled bool `yaml:"enabled" json:"enabled" default:"true"`

	// Topic for dead letter messages
	Topic string `yaml:"topic" json:"topic"`

	// MaxRetries before sending to dead letter
	MaxRetries int `yaml:"max_retries" json:"max_retries" default:"3"`

	// TTL for dead letter messages
	TTL time.Duration `yaml:"ttl" json:"ttl" default:"168h"`
}

// OffsetConfig defines offset management settings.
type OffsetConfig struct {
	// Initial position when starting (earliest, latest)
	Initial OffsetPosition `yaml:"initial" json:"initial" default:"latest"`

	// AutoCommit enables automatic offset commits
	AutoCommit bool `yaml:"auto_commit" json:"auto_commit" default:"true"`

	// Interval for automatic offset commits
	Interval time.Duration `yaml:"interval" json:"interval" default:"5s"`
}

// OffsetPosition defines initial offset positions.
type OffsetPosition string

const (
	// OffsetEarliest starts from the earliest available message
	OffsetEarliest OffsetPosition = "earliest"

	// OffsetLatest starts from the latest message
	OffsetLatest OffsetPosition = "latest"
)

// ConfigManager provides hierarchical configuration management with validation
type ConfigManager struct {
	validator *ConfigValidator
	profile   string
	basePath  string
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(profile string) *ConfigManager {
	validator := NewConfigValidator()

	// Load built-in profiles
	for _, profile := range GetBuiltinProfiles() {
		validator.AddProfile(profile)
	}

	return &ConfigManager{
		validator: validator,
		profile:   profile,
		basePath:  ".",
	}
}

// SetBasePath sets the base path for configuration file searches
func (cm *ConfigManager) SetBasePath(path string) {
	cm.basePath = path
}

// LoadBrokerConfig loads and validates broker configuration from various sources
func (cm *ConfigManager) LoadBrokerConfig(filename string) (*BrokerConfig, error) {
	config := &BrokerConfig{}

	// Set defaults
	config.SetDefaults()

	// Load from file if specified
	if filename != "" {
		if err := cm.loadFromFile(config, filename); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Apply environment variable overrides
	if err := ApplyEnvironmentOverrides(config, "MESSAGING"); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Validate with profile
	ctx := ValidationContext{
		Profile:     cm.profile,
		Environment: make(map[string]string),
		Strict:      false,
	}

	if err := cm.validator.ValidateWithProfile(config, cm.profile, ctx); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// LoadPublisherConfig loads and validates publisher configuration
func (cm *ConfigManager) LoadPublisherConfig(filename string) (*PublisherConfig, error) {
	config := &PublisherConfig{}

	// Set defaults
	setPublisherDefaults(config)

	// Load from file if specified
	if filename != "" {
		if err := cm.loadFromFile(config, filename); err != nil {
			return nil, fmt.Errorf("failed to load publisher config from file: %w", err)
		}
	}

	// Apply environment variable overrides
	if err := ApplyEnvironmentOverrides(config, "MESSAGING_PUBLISHER"); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Validate configuration
	if err := cm.validatePublisherConfig(config); err != nil {
		return nil, fmt.Errorf("publisher configuration validation failed: %w", err)
	}

	return config, nil
}

// LoadSubscriberConfig loads and validates subscriber configuration
func (cm *ConfigManager) LoadSubscriberConfig(filename string) (*SubscriberConfig, error) {
	config := &SubscriberConfig{}

	// Set defaults
	setSubscriberDefaults(config)

	// Load from file if specified
	if filename != "" {
		if err := cm.loadFromFile(config, filename); err != nil {
			return nil, fmt.Errorf("failed to load subscriber config from file: %w", err)
		}
	}

	// Apply environment variable overrides
	if err := ApplyEnvironmentOverrides(config, "MESSAGING_SUBSCRIBER"); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Validate configuration
	if err := cm.validateSubscriberConfig(config); err != nil {
		return nil, fmt.Errorf("subscriber configuration validation failed: %w", err)
	}

	return config, nil
}

// loadFromFile loads configuration from a YAML or JSON file
func (cm *ConfigManager) loadFromFile(config interface{}, filename string) error {
	// Resolve file path
	filePath := filename
	if !filepath.IsAbs(filename) {
		filePath = filepath.Join(cm.basePath, filename)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", filePath)
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Determine file format and unmarshal
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to unmarshal YAML configuration: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to unmarshal JSON configuration: %w", err)
		}
	default:
		return fmt.Errorf("unsupported configuration file format: %s", ext)
	}

	return nil
}

// setPublisherDefaults sets default values for publisher configuration
func setPublisherDefaults(config *PublisherConfig) {
	if config.Routing.Strategy == "" {
		config.Routing.Strategy = "round_robin"
	}

	config.Batching.Enabled = true
	if config.Batching.MaxMessages == 0 {
		config.Batching.MaxMessages = 100
	}
	if config.Batching.MaxBytes == 0 {
		config.Batching.MaxBytes = 1048576 // 1MB
	}
	if config.Batching.FlushInterval == 0 {
		config.Batching.FlushInterval = 100 * time.Millisecond
	}

	if config.Confirmation.Timeout == 0 {
		config.Confirmation.Timeout = 5 * time.Second
	}
	if config.Confirmation.Retries == 0 {
		config.Confirmation.Retries = 3
	}

	if config.Timeout.Publish == 0 {
		config.Timeout.Publish = 30 * time.Second
	}
	if config.Timeout.Flush == 0 {
		config.Timeout.Flush = 30 * time.Second
	}
	if config.Timeout.Close == 0 {
		config.Timeout.Close = 30 * time.Second
	}

	config.Retry.SetDefaults()

	// Defaults for producer pool
	if config.Workers == 0 {
		config.Workers = runtime.NumCPU()
	}
	if config.QueueSize == 0 {
		config.QueueSize = 10000
	}
}

// setSubscriberDefaults sets default values for subscriber configuration
func setSubscriberDefaults(config *SubscriberConfig) {
	if config.Type == "" {
		config.Type = SubscriptionShared
	}
	if config.Concurrency == 0 {
		config.Concurrency = 1
	}
	if config.PrefetchCount == 0 {
		config.PrefetchCount = 10
	}

	if config.Processing.MaxProcessingTime == 0 {
		config.Processing.MaxProcessingTime = 30 * time.Second
	}
	if config.Processing.AckMode == "" {
		config.Processing.AckMode = AckModeAuto
	}
	if config.Processing.MaxInFlight == 0 {
		config.Processing.MaxInFlight = 100
	}

	config.DeadLetter.Enabled = true
	if config.DeadLetter.MaxRetries == 0 {
		config.DeadLetter.MaxRetries = 3
	}
	if config.DeadLetter.TTL == 0 {
		config.DeadLetter.TTL = 168 * time.Hour // 7 days
	}

	if config.Offset.Initial == "" {
		config.Offset.Initial = OffsetLatest
	}
	config.Offset.AutoCommit = true
	if config.Offset.Interval == 0 {
		config.Offset.Interval = 5 * time.Second
	}

	config.Retry.SetDefaults()
}

// validatePublisherConfig validates publisher configuration
func (cm *ConfigManager) validatePublisherConfig(config *PublisherConfig) error {
	if config.Topic == "" {
		return NewConfigError("publisher topic is required", nil)
	}

	validStrategies := []string{"round_robin", "hash", "random"}
	strategyValid := false
	for _, strategy := range validStrategies {
		if config.Routing.Strategy == strategy {
			strategyValid = true
			break
		}
	}
	if !strategyValid {
		return NewConfigError(fmt.Sprintf("invalid routing strategy: %s, must be one of: %s",
			config.Routing.Strategy, strings.Join(validStrategies, ", ")), nil)
	}

	if config.Batching.Enabled {
		if config.Batching.MaxMessages <= 0 {
			return NewConfigError("batch max_messages must be positive when batching is enabled", nil)
		}
		if config.Batching.MaxBytes <= 0 {
			return NewConfigError("batch max_bytes must be positive when batching is enabled", nil)
		}
		if config.Batching.FlushInterval <= 0 {
			return NewConfigError("batch flush_interval must be positive when batching is enabled", nil)
		}
	}

	if config.Workers <= 0 {
		return NewConfigError("publisher workers must be positive", nil)
	}
	if config.QueueSize <= 0 {
		return NewConfigError("publisher queue_size must be positive", nil)
	}

	return nil
}

// validateSubscriberConfig validates subscriber configuration
func (cm *ConfigManager) validateSubscriberConfig(config *SubscriberConfig) error {
	if len(config.Topics) == 0 {
		return NewConfigError("subscriber topics are required", nil)
	}

	if config.ConsumerGroup == "" {
		return NewConfigError("subscriber consumer_group is required", nil)
	}

	validTypes := []SubscriptionType{SubscriptionShared, SubscriptionExclusive, SubscriptionFailover}
	typeValid := false
	for _, validType := range validTypes {
		if config.Type == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return NewConfigError(fmt.Sprintf("invalid subscription type: %s", config.Type), nil)
	}

	if config.Concurrency <= 0 {
		return NewConfigError("subscriber concurrency must be positive", nil)
	}

	if config.PrefetchCount <= 0 {
		return NewConfigError("subscriber prefetch_count must be positive", nil)
	}

	return nil
}

// LoadProfilesFromFile loads configuration profiles from a file
func (cm *ConfigManager) LoadProfilesFromFile(filename string) error {
	return cm.validator.LoadProfilesFromFile(filename)
}

// GetValidator returns the underlying configuration validator
func (cm *ConfigManager) GetValidator() *ConfigValidator {
	return cm.validator
}

// MergeConfigs merges multiple configurations with the second one taking precedence
func MergeConfigs(base, override *BrokerConfig) (*BrokerConfig, error) {
	if base == nil {
		return override, nil
	}
	if override == nil {
		return base, nil
	}

	// Create a deep copy of the base configuration
	result := &BrokerConfig{}
	*result = *base

	// Override with non-zero values from override config
	if override.Type != "" {
		result.Type = override.Type
	}
	if len(override.Endpoints) > 0 {
		result.Endpoints = override.Endpoints
	}
	if override.Connection != (ConnectionConfig{}) {
		result.Connection = mergeConnectionConfigs(result.Connection, override.Connection)
	}
	if override.Authentication != nil {
		result.Authentication = override.Authentication
	}
	if override.TLS != nil {
		result.TLS = override.TLS
	}
	if override.Retry != (RetryConfig{}) {
		result.Retry = mergeRetryConfigs(result.Retry, override.Retry)
	}
	if override.Monitoring != (MonitoringConfig{}) {
		result.Monitoring = mergeMonitoringConfigs(result.Monitoring, override.Monitoring)
	}
	if len(override.Extra) > 0 {
		if result.Extra == nil {
			result.Extra = make(map[string]interface{})
		}
		for k, v := range override.Extra {
			result.Extra[k] = v
		}
	}

	return result, nil
}

// mergeConnectionConfigs merges connection configurations
func mergeConnectionConfigs(base, override ConnectionConfig) ConnectionConfig {
	result := base
	if override.Timeout != 0 {
		result.Timeout = override.Timeout
	}
	if override.KeepAlive != 0 {
		result.KeepAlive = override.KeepAlive
	}
	if override.MaxAttempts != 0 {
		result.MaxAttempts = override.MaxAttempts
	}
	if override.PoolSize != 0 {
		result.PoolSize = override.PoolSize
	}
	if override.IdleTimeout != 0 {
		result.IdleTimeout = override.IdleTimeout
	}
	return result
}

// mergeRetryConfigs merges retry configurations
func mergeRetryConfigs(base, override RetryConfig) RetryConfig {
	result := base
	if override.MaxAttempts != 0 {
		result.MaxAttempts = override.MaxAttempts
	}
	if override.InitialDelay != 0 {
		result.InitialDelay = override.InitialDelay
	}
	if override.MaxDelay != 0 {
		result.MaxDelay = override.MaxDelay
	}
	if override.Multiplier != 0 {
		result.Multiplier = override.Multiplier
	}
	if override.Jitter != 0 {
		result.Jitter = override.Jitter
	}
	return result
}

// mergeMonitoringConfigs merges monitoring configurations
func mergeMonitoringConfigs(base, override MonitoringConfig) MonitoringConfig {
	result := base
	if override.MetricsInterval != 0 {
		result.MetricsInterval = override.MetricsInterval
	}
	if override.HealthCheckInterval != 0 {
		result.HealthCheckInterval = override.HealthCheckInterval
	}
	if override.HealthCheckTimeout != 0 {
		result.HealthCheckTimeout = override.HealthCheckTimeout
	}
	// Enabled is boolean, so we check if it's explicitly set
	result.Enabled = override.Enabled
	return result
}
