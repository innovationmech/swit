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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewConfigManager(t *testing.T) {
	manager := NewConfigManager("development")
	if manager == nil {
		t.Fatal("expected config manager to be created")
	}
	if manager.profile != "development" {
		t.Errorf("expected profile 'development', got %s", manager.profile)
	}
	if manager.validator == nil {
		t.Fatal("expected validator to be initialized")
	}
}

func TestConfigManager_SetBasePath(t *testing.T) {
	manager := NewConfigManager("development")
	testPath := "/test/path"
	manager.SetBasePath(testPath)

	if manager.basePath != testPath {
		t.Errorf("expected base path %s, got %s", testPath, manager.basePath)
	}
}

func TestConfigManager_LoadBrokerConfig_FromYAML(t *testing.T) {
	// Create a temporary YAML config file
	yamlContent := `
type: inmemory
endpoints:
  - localhost:9092
  - localhost:9093
connection:
  timeout: 15s
  keep_alive: 45s
  max_attempts: 2
  pool_size: 15
retry:
  max_attempts: 2
  initial_delay: 2s
  max_delay: 45s
monitoring:
  enabled: true
  metrics_interval: 45s
`

	tempFile := createTempFile(t, "config.yaml", yamlContent)
	defer os.Remove(tempFile)

	manager := NewConfigManager("development")
	manager.SetBasePath(filepath.Dir(tempFile))

	config, err := manager.LoadBrokerConfig(filepath.Base(tempFile))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify loaded values
	if config.Type != BrokerTypeInMemory {
		t.Errorf("expected type inmemory, got %s", config.Type)
	}
	if len(config.Endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(config.Endpoints))
	}
	if config.Connection.Timeout != 15*time.Second {
		t.Errorf("expected timeout 15s, got %v", config.Connection.Timeout)
	}
	if config.Connection.KeepAlive != 45*time.Second {
		t.Errorf("expected keep alive 45s, got %v", config.Connection.KeepAlive)
	}
	if config.Retry.MaxAttempts != 2 {
		t.Errorf("expected max attempts 2, got %d", config.Retry.MaxAttempts)
	}
}

func TestConfigManager_LoadBrokerConfig_FromJSON(t *testing.T) {
	// Create a temporary JSON config file
	jsonContent := `{
  "type": "kafka",
  "endpoints": ["localhost:9092"],
  "connection": {
    "timeout": "20s",
    "keep_alive": "60s",
    "max_attempts": 3,
    "pool_size": 20
  },
  "retry": {
    "max_attempts": 3,
    "initial_delay": "1s",
    "max_delay": "30s",
    "multiplier": 2.0,
    "jitter": 0.1
  },
  "monitoring": {
    "enabled": true,
    "metrics_interval": "60s"
  }
}`

	tempFile := createTempFile(t, "config.json", jsonContent)
	defer os.Remove(tempFile)

	manager := NewConfigManager("production")
	manager.SetBasePath(filepath.Dir(tempFile))

	config, err := manager.LoadBrokerConfig(filepath.Base(tempFile))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify loaded values
	if config.Type != BrokerTypeKafka {
		t.Errorf("expected type kafka, got %s", config.Type)
	}
	if config.Connection.Timeout != 20*time.Second {
		t.Errorf("expected timeout 20s, got %v", config.Connection.Timeout)
	}
	if config.Retry.Multiplier != 2.0 {
		t.Errorf("expected multiplier 2.0, got %f", config.Retry.Multiplier)
	}
}

func TestConfigManager_LoadBrokerConfig_WithEnvironmentOverrides(t *testing.T) {
	// Set environment variables
	envVars := map[string]string{
		"MESSAGING_TYPE":                        "nats",
		"MESSAGING_ENDPOINTS":                   "nats://localhost:4222",
		"MESSAGING_CONNECTION_TIMEOUT":          "25s",
		"MESSAGING_CONNECTION_MAX_ATTEMPTS":     "5",
		"MESSAGING_RETRY_MAX_ATTEMPTS":          "5",
		"MESSAGING_MONITORING_ENABLED":          "true",
		"MESSAGING_MONITORING_METRICS_INTERVAL": "90s",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
	}
	defer func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
	}()

	// Create basic config file
	yamlContent := `
type: inmemory
endpoints:
  - localhost:9092
`

	tempFile := createTempFile(t, "config.yaml", yamlContent)
	defer os.Remove(tempFile)

	manager := NewConfigManager("development")
	manager.SetBasePath(filepath.Dir(tempFile))

	config, err := manager.LoadBrokerConfig(filepath.Base(tempFile))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify environment overrides were applied
	if config.Type != BrokerTypeNATS {
		t.Errorf("expected type nats (from env), got %s", config.Type)
	}
	if config.Connection.Timeout != 25*time.Second {
		t.Errorf("expected timeout 25s (from env), got %v", config.Connection.Timeout)
	}
	if config.Connection.MaxAttempts != 5 {
		t.Errorf("expected max attempts 5 (from env), got %d", config.Connection.MaxAttempts)
	}
}

func TestConfigManager_LoadPublisherConfig(t *testing.T) {
	yamlContent := `
topic: test-topic
routing:
  strategy: hash
  partition_key: user_id
batching:
  enabled: true
  max_messages: 200
  max_bytes: 2097152
  flush_interval: 200ms
confirmation:
  required: true
  timeout: 10s
  retries: 5
timeout:
  publish: 60s
  flush: 45s
  close: 30s
`

	tempFile := createTempFile(t, "publisher.yaml", yamlContent)
	defer os.Remove(tempFile)

	manager := NewConfigManager("development")
	manager.SetBasePath(filepath.Dir(tempFile))

	config, err := manager.LoadPublisherConfig(filepath.Base(tempFile))
	if err != nil {
		t.Fatalf("failed to load publisher config: %v", err)
	}

	// Verify loaded values
	if config.Topic != "test-topic" {
		t.Errorf("expected topic 'test-topic', got %s", config.Topic)
	}
	if config.Routing.Strategy != "hash" {
		t.Errorf("expected strategy 'hash', got %s", config.Routing.Strategy)
	}
	if config.Routing.PartitionKey != "user_id" {
		t.Errorf("expected partition key 'user_id', got %s", config.Routing.PartitionKey)
	}
	if config.Batching.MaxMessages != 200 {
		t.Errorf("expected max messages 200, got %d", config.Batching.MaxMessages)
	}
	if config.Confirmation.Required != true {
		t.Error("expected confirmation required to be true")
	}
}

func TestConfigManager_LoadSubscriberConfig(t *testing.T) {
	yamlContent := `
topics:
  - topic1
  - topic2
consumer_group: test-group
type: exclusive
concurrency: 4
prefetch_count: 50
processing:
  max_processing_time: 60s
  ack_mode: manual
  max_in_flight: 200
  ordered: true
dead_letter:
  enabled: true
  topic: dead-letter-topic
  max_retries: 5
  ttl: 336h
offset:
  initial: earliest
  auto_commit: false
  interval: 10s
`

	tempFile := createTempFile(t, "subscriber.yaml", yamlContent)
	defer os.Remove(tempFile)

	manager := NewConfigManager("development")
	manager.SetBasePath(filepath.Dir(tempFile))

	config, err := manager.LoadSubscriberConfig(filepath.Base(tempFile))
	if err != nil {
		t.Fatalf("failed to load subscriber config: %v", err)
	}

	// Verify loaded values
	if len(config.Topics) != 2 {
		t.Errorf("expected 2 topics, got %d", len(config.Topics))
	}
	if config.ConsumerGroup != "test-group" {
		t.Errorf("expected consumer group 'test-group', got %s", config.ConsumerGroup)
	}
	if config.Type != SubscriptionExclusive {
		t.Errorf("expected type exclusive, got %s", config.Type)
	}
	if config.Concurrency != 4 {
		t.Errorf("expected concurrency 4, got %d", config.Concurrency)
	}
	if config.Processing.AckMode != AckModeManual {
		t.Errorf("expected ack mode manual, got %s", config.Processing.AckMode)
	}
	if config.Processing.Ordered != true {
		t.Error("expected ordered processing to be true")
	}
	if config.DeadLetter.Topic != "dead-letter-topic" {
		t.Errorf("expected dead letter topic 'dead-letter-topic', got %s", config.DeadLetter.Topic)
	}
	if config.Offset.Initial != OffsetEarliest {
		t.Errorf("expected offset initial earliest, got %s", config.Offset.Initial)
	}
}

func TestMergeConfigs(t *testing.T) {
	base := &BrokerConfig{
		Type:      BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
		Connection: ConnectionConfig{
			Timeout:     10 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
		},
		Retry: RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
		},
	}

	override := &BrokerConfig{
		Type:      BrokerTypeNATS,
		Endpoints: []string{"nats://localhost:4222"},
		Connection: ConnectionConfig{
			Timeout:  20 * time.Second,
			PoolSize: 20,
		},
		Retry: RetryConfig{
			MaxAttempts: 5,
		},
	}

	merged, err := MergeConfigs(base, override)
	if err != nil {
		t.Fatalf("failed to merge configs: %v", err)
	}

	// Verify overridden values
	if merged.Type != BrokerTypeNATS {
		t.Errorf("expected type nats, got %s", merged.Type)
	}
	if len(merged.Endpoints) != 1 || merged.Endpoints[0] != "nats://localhost:4222" {
		t.Errorf("expected endpoints to be overridden")
	}
	if merged.Connection.Timeout != 20*time.Second {
		t.Errorf("expected timeout 20s, got %v", merged.Connection.Timeout)
	}
	if merged.Connection.PoolSize != 20 {
		t.Errorf("expected pool size 20, got %d", merged.Connection.PoolSize)
	}

	// Verify preserved values from base
	if merged.Connection.KeepAlive != 30*time.Second {
		t.Errorf("expected keep alive to be preserved, got %v", merged.Connection.KeepAlive)
	}
	if merged.Connection.MaxAttempts != 3 {
		t.Errorf("expected max attempts to be preserved, got %d", merged.Connection.MaxAttempts)
	}

	// Verify merged retry values
	if merged.Retry.MaxAttempts != 5 {
		t.Errorf("expected retry max attempts 5, got %d", merged.Retry.MaxAttempts)
	}
	if merged.Retry.InitialDelay != 1*time.Second {
		t.Errorf("expected initial delay to be preserved, got %v", merged.Retry.InitialDelay)
	}
}

func TestMergeConfigs_NilHandling(t *testing.T) {
	base := &BrokerConfig{
		Type:      BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
	}

	// Test with nil override
	merged, err := MergeConfigs(base, nil)
	if err != nil {
		t.Fatalf("failed to merge configs: %v", err)
	}
	if merged != base {
		t.Error("expected merged config to be same as base when override is nil")
	}

	// Test with nil base
	override := &BrokerConfig{
		Type:      BrokerTypeNATS,
		Endpoints: []string{"nats://localhost:4222"},
	}

	merged, err = MergeConfigs(nil, override)
	if err != nil {
		t.Fatalf("failed to merge configs: %v", err)
	}
	if merged != override {
		t.Error("expected merged config to be same as override when base is nil")
	}

	// Test with both nil
	merged, err = MergeConfigs(nil, nil)
	if err != nil {
		t.Fatalf("failed to merge configs: %v", err)
	}
	if merged != nil {
		t.Error("expected merged config to be nil when both are nil")
	}
}

func TestConfigManager_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		configType  string
		content     string
		expectError string
	}{
		{
			name:       "invalid broker type",
			configType: "broker",
			content: `
type: invalid_broker_type
endpoints:
  - localhost:9092
`,
			expectError: "invalid broker type",
		},
		{
			name:       "missing endpoints",
			configType: "broker",
			content: `
type: kafka
endpoints: []
`,
			expectError: "at least one endpoint must be specified",
		},
		{
			name:       "invalid publisher strategy",
			configType: "publisher",
			content: `
topic: test-topic
routing:
  strategy: invalid_strategy
`,
			expectError: "invalid routing strategy",
		},
		{
			name:       "missing subscriber topics",
			configType: "subscriber",
			content: `
consumer_group: test-group
topics: []
`,
			expectError: "subscriber topics are required",
		},
		{
			name:       "missing subscriber consumer group",
			configType: "subscriber",
			content: `
topics:
  - topic1
`,
			expectError: "subscriber consumer_group is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := createTempFile(t, "config.yaml", tt.content)
			defer os.Remove(tempFile)

			manager := NewConfigManager("development")
			manager.SetBasePath(filepath.Dir(tempFile))

			var err error
			switch tt.configType {
			case "broker":
				_, err = manager.LoadBrokerConfig(filepath.Base(tempFile))
			case "publisher":
				_, err = manager.LoadPublisherConfig(filepath.Base(tempFile))
			case "subscriber":
				_, err = manager.LoadSubscriberConfig(filepath.Base(tempFile))
			}

			if err == nil {
				t.Fatal("expected validation error but got none")
			}

			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.expectError)) {
				t.Errorf("expected error containing '%s', got '%s'", tt.expectError, err.Error())
			}
		})
	}
}

func TestConfigManager_UnsupportedFileFormat(t *testing.T) {
	// Create a file with unsupported extension
	content := `type: kafka`
	tempFile := createTempFile(t, "config.txt", content)
	defer os.Remove(tempFile)

	manager := NewConfigManager("development")
	manager.SetBasePath(filepath.Dir(tempFile))

	_, err := manager.LoadBrokerConfig(filepath.Base(tempFile))
	if err == nil {
		t.Fatal("expected error for unsupported file format")
	}

	if !strings.Contains(err.Error(), "unsupported configuration file format") {
		t.Errorf("expected error about unsupported format, got: %v", err)
	}
}

func TestConfigManager_FileNotFound(t *testing.T) {
	manager := NewConfigManager("development")

	_, err := manager.LoadBrokerConfig("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}

	if !strings.Contains(err.Error(), "configuration file not found") {
		t.Errorf("expected error about file not found, got: %v", err)
	}
}

// Helper function to create temporary files for testing
func createTempFile(t *testing.T, name, content string) string {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, name)

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	return filePath
}
