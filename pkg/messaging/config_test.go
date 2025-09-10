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
	"testing"
	"time"
)

// TestBrokerConfigValidation tests BrokerConfig validation.
func TestBrokerConfigValidation(t *testing.T) {
	// Test valid config
	validConfig := &BrokerConfig{
		Type:      BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
		Connection: ConnectionConfig{
			Timeout:     10 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    10,
			IdleTimeout: 5 * time.Minute,
		},
		Retry: RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
	}

	err := validConfig.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}

	// Test invalid broker type
	invalidTypeConfig := *validConfig
	invalidTypeConfig.Type = BrokerType("invalid")
	err = invalidTypeConfig.Validate()
	if err == nil {
		t.Error("Expected error for invalid broker type")
	}
	if !IsConfigurationError(err) {
		t.Errorf("Expected configuration error, got: %v", err)
	}

	// Test empty endpoints
	emptyEndpointsConfig := *validConfig
	emptyEndpointsConfig.Endpoints = []string{}
	err = emptyEndpointsConfig.Validate()
	if err == nil {
		t.Error("Expected error for empty endpoints")
	}

	// Test empty endpoint string
	emptyEndpointConfig := *validConfig
	emptyEndpointConfig.Endpoints = []string{"localhost:9092", ""}
	err = emptyEndpointConfig.Validate()
	if err == nil {
		t.Error("Expected error for empty endpoint string")
	}
}

// TestConnectionConfigValidation tests ConnectionConfig validation.
func TestConnectionConfigValidation(t *testing.T) {
	// Test valid config
	validConfig := &ConnectionConfig{
		Timeout:     10 * time.Second,
		KeepAlive:   30 * time.Second,
		MaxAttempts: 3,
		PoolSize:    10,
		IdleTimeout: 5 * time.Minute,
	}

	err := validConfig.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}

	// Test invalid timeout
	invalidTimeout := *validConfig
	invalidTimeout.Timeout = 0
	err = invalidTimeout.Validate()
	if err == nil {
		t.Error("Expected error for zero timeout")
	}

	// Test negative keep alive
	negativeKeepAlive := *validConfig
	negativeKeepAlive.KeepAlive = -1 * time.Second
	err = negativeKeepAlive.Validate()
	if err == nil {
		t.Error("Expected error for negative keep alive")
	}

	// Test invalid max attempts
	invalidMaxAttempts := *validConfig
	invalidMaxAttempts.MaxAttempts = 0
	err = invalidMaxAttempts.Validate()
	if err == nil {
		t.Error("Expected error for zero max attempts")
	}

	// Test invalid pool size
	invalidPoolSize := *validConfig
	invalidPoolSize.PoolSize = 0
	err = invalidPoolSize.Validate()
	if err == nil {
		t.Error("Expected error for zero pool size")
	}

	// Test negative idle timeout
	negativeIdleTimeout := *validConfig
	negativeIdleTimeout.IdleTimeout = -1 * time.Second
	err = negativeIdleTimeout.Validate()
	if err == nil {
		t.Error("Expected error for negative idle timeout")
	}
}

// TestAuthConfigValidation tests AuthConfig validation.
func TestAuthConfigValidation(t *testing.T) {
	// Test None auth
	noneAuth := &AuthConfig{Type: AuthTypeNone}
	err := noneAuth.Validate()
	if err != nil {
		t.Errorf("Expected no error for none auth, got: %v", err)
	}

	// Test SASL auth - valid
	saslAuth := &AuthConfig{
		Type:     AuthTypeSASL,
		Username: "user",
		Password: "pass",
	}
	err = saslAuth.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid SASL auth, got: %v", err)
	}

	// Test SASL auth - missing username
	saslNoUsername := &AuthConfig{
		Type:     AuthTypeSASL,
		Password: "pass",
	}
	err = saslNoUsername.Validate()
	if err == nil {
		t.Error("Expected error for SASL auth without username")
	}

	// Test SASL auth - missing password
	saslNoPassword := &AuthConfig{
		Type:     AuthTypeSASL,
		Username: "user",
	}
	err = saslNoPassword.Validate()
	if err == nil {
		t.Error("Expected error for SASL auth without password")
	}

	// Test OAuth2 auth - valid
	oauth2Auth := &AuthConfig{
		Type:         AuthTypeOAuth2,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenURL:     "https://example.com/token",
	}
	err = oauth2Auth.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid OAuth2 auth, got: %v", err)
	}

	// Test OAuth2 auth - missing client ID
	oauth2NoClientID := &AuthConfig{
		Type:         AuthTypeOAuth2,
		ClientSecret: "client-secret",
		TokenURL:     "https://example.com/token",
	}
	err = oauth2NoClientID.Validate()
	if err == nil {
		t.Error("Expected error for OAuth2 auth without client ID")
	}

	// Test API Key auth - valid
	apiKeyAuth := &AuthConfig{
		Type:   AuthTypeAPIKey,
		APIKey: "api-key-123",
	}
	err = apiKeyAuth.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid API key auth, got: %v", err)
	}

	// Test API Key auth - missing key
	apiKeyNoKey := &AuthConfig{Type: AuthTypeAPIKey}
	err = apiKeyNoKey.Validate()
	if err == nil {
		t.Error("Expected error for API key auth without key")
	}

	// Test unsupported auth type
	unsupportedAuth := &AuthConfig{Type: AuthType("unsupported")}
	err = unsupportedAuth.Validate()
	if err == nil {
		t.Error("Expected error for unsupported auth type")
	}
}

// TestTLSConfigValidation tests TLSConfig validation.
func TestTLSConfigValidation(t *testing.T) {
	// Test disabled TLS
	disabledTLS := &TLSConfig{Enabled: false}
	err := disabledTLS.Validate()
	if err != nil {
		t.Errorf("Expected no error for disabled TLS, got: %v", err)
	}

	// Test enabled TLS without certs
	enabledTLS := &TLSConfig{Enabled: true}
	err = enabledTLS.Validate()
	if err != nil {
		t.Errorf("Expected no error for enabled TLS without certs, got: %v", err)
	}

	// Test with cert file but no key file
	certOnlyTLS := &TLSConfig{
		Enabled:  true,
		CertFile: "cert.pem",
	}
	err = certOnlyTLS.Validate()
	if err == nil {
		t.Error("Expected error for cert file without key file")
	}

	// Test with key file but no cert file
	keyOnlyTLS := &TLSConfig{
		Enabled: true,
		KeyFile: "key.pem",
	}
	err = keyOnlyTLS.Validate()
	if err == nil {
		t.Error("Expected error for key file without cert file")
	}

	// Test with both cert and key files
	validTLS := &TLSConfig{
		Enabled:  true,
		CertFile: "cert.pem",
		KeyFile:  "key.pem",
	}
	err = validTLS.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid TLS config, got: %v", err)
	}
}

// TestRetryConfigValidation tests RetryConfig validation.
func TestRetryConfigValidation(t *testing.T) {
	// Test valid config
	validConfig := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	err := validConfig.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid retry config, got: %v", err)
	}

	// Test negative max attempts
	negativeAttempts := *validConfig
	negativeAttempts.MaxAttempts = -1
	err = negativeAttempts.Validate()
	if err == nil {
		t.Error("Expected error for negative max attempts")
	}

	// Test negative initial delay
	negativeInitialDelay := *validConfig
	negativeInitialDelay.InitialDelay = -1 * time.Second
	err = negativeInitialDelay.Validate()
	if err == nil {
		t.Error("Expected error for negative initial delay")
	}

	// Test negative max delay
	negativeMaxDelay := *validConfig
	negativeMaxDelay.MaxDelay = -1 * time.Second
	err = negativeMaxDelay.Validate()
	if err == nil {
		t.Error("Expected error for negative max delay")
	}

	// Test max delay less than initial delay
	invalidDelayOrder := *validConfig
	invalidDelayOrder.MaxDelay = 500 * time.Millisecond
	invalidDelayOrder.InitialDelay = 1 * time.Second
	err = invalidDelayOrder.Validate()
	if err == nil {
		t.Error("Expected error for max delay less than initial delay")
	}

	// Test invalid multiplier
	invalidMultiplier := *validConfig
	invalidMultiplier.Multiplier = 1.0
	err = invalidMultiplier.Validate()
	if err == nil {
		t.Error("Expected error for multiplier <= 1.0")
	}

	// Test invalid jitter - negative
	negativeJitter := *validConfig
	negativeJitter.Jitter = -0.1
	err = negativeJitter.Validate()
	if err == nil {
		t.Error("Expected error for negative jitter")
	}

	// Test invalid jitter - greater than 1
	largeJitter := *validConfig
	largeJitter.Jitter = 1.1
	err = largeJitter.Validate()
	if err == nil {
		t.Error("Expected error for jitter > 1.0")
	}
}

// TestBrokerConfigWithNestedValidation tests BrokerConfig validation with nested configs.
func TestBrokerConfigWithNestedValidation(t *testing.T) {
	config := &BrokerConfig{
		Type:      BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
		Connection: ConnectionConfig{
			Timeout: 0, // Invalid
		},
		Authentication: &AuthConfig{
			Type: AuthTypeSASL,
			// Missing username and password
		},
		TLS: &TLSConfig{
			Enabled:  true,
			CertFile: "cert.pem",
			// Missing key file
		},
		Retry: RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     500 * time.Millisecond, // Less than initial delay
			Multiplier:   2.0,
			Jitter:       0.1,
		},
	}

	// Should fail due to multiple nested validation errors
	err := config.Validate()
	if err == nil {
		t.Error("Expected error for config with multiple nested validation issues")
	}

	// Fix connection config
	config.Connection.Timeout = 10 * time.Second
	config.Connection.KeepAlive = 30 * time.Second
	config.Connection.MaxAttempts = 3
	config.Connection.PoolSize = 10
	config.Connection.IdleTimeout = 5 * time.Minute

	err = config.Validate()
	if err == nil {
		t.Error("Expected error for remaining validation issues")
	}

	// Fix auth config
	config.Authentication.Username = "user"
	config.Authentication.Password = "pass"

	err = config.Validate()
	if err == nil {
		t.Error("Expected error for remaining TLS and retry issues")
	}

	// Fix TLS config
	config.TLS.KeyFile = "key.pem"

	err = config.Validate()
	if err == nil {
		t.Error("Expected error for remaining retry issue")
	}

	// Fix retry config
	config.Retry.MaxDelay = 30 * time.Second

	err = config.Validate()
	if err != nil {
		t.Errorf("Expected no error after fixing all issues, got: %v", err)
	}
}

// TestBrokerConfigSetDefaults tests that SetDefaults applies correct default values.
func TestBrokerConfigSetDefaults(t *testing.T) {
	config := &BrokerConfig{
		Type:      BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
	}

	// Apply defaults
	config.SetDefaults()

	// Check connection defaults
	if config.Connection.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout to be 10s, got %v", config.Connection.Timeout)
	}
	if config.Connection.KeepAlive != 30*time.Second {
		t.Errorf("Expected default keep alive to be 30s, got %v", config.Connection.KeepAlive)
	}
	if config.Connection.MaxAttempts != 3 {
		t.Errorf("Expected default max attempts to be 3, got %d", config.Connection.MaxAttempts)
	}
	if config.Connection.PoolSize != 10 {
		t.Errorf("Expected default pool size to be 10, got %d", config.Connection.PoolSize)
	}
	if config.Connection.IdleTimeout != 5*time.Minute {
		t.Errorf("Expected default idle timeout to be 5m, got %v", config.Connection.IdleTimeout)
	}

	// Check retry defaults
	if config.Retry.MaxAttempts != 3 {
		t.Errorf("Expected default retry max attempts to be 3, got %d", config.Retry.MaxAttempts)
	}
	if config.Retry.InitialDelay != 1*time.Second {
		t.Errorf("Expected default initial delay to be 1s, got %v", config.Retry.InitialDelay)
	}
	if config.Retry.MaxDelay != 30*time.Second {
		t.Errorf("Expected default max delay to be 30s, got %v", config.Retry.MaxDelay)
	}
	if config.Retry.Multiplier != 2.0 {
		t.Errorf("Expected default multiplier to be 2.0, got %f", config.Retry.Multiplier)
	}
	if config.Retry.Jitter != 0.1 {
		t.Errorf("Expected default jitter to be 0.1, got %f", config.Retry.Jitter)
	}

	// Check monitoring defaults
	if !config.Monitoring.Enabled {
		t.Error("Expected monitoring to be enabled by default")
	}
	if config.Monitoring.MetricsInterval != 30*time.Second {
		t.Errorf("Expected default metrics interval to be 30s, got %v", config.Monitoring.MetricsInterval)
	}
	if config.Monitoring.HealthCheckInterval != 30*time.Second {
		t.Errorf("Expected default health check interval to be 30s, got %v", config.Monitoring.HealthCheckInterval)
	}
	if config.Monitoring.HealthCheckTimeout != 5*time.Second {
		t.Errorf("Expected default health check timeout to be 5s, got %v", config.Monitoring.HealthCheckTimeout)
	}
}
