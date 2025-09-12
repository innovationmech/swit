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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSecurityManager_NewSecurityManager(t *testing.T) {
	logger := zap.NewNop()

	config := &SecurityManagerConfig{
		DefaultPolicy: &SecurityPolicy{
			Name: "default",
			Encryption: &EncryptionConfig{
				Enabled:   true,
				Algorithm: EncryptionAlgorithmA256GCM,
			},
			Signing: &SigningConfig{
				Enabled:   true,
				Algorithm: SigningAlgorithmHS256,
			},
		},
		Logger: logger,
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)
	require.NotNil(t, sm)
	assert.NotNil(t, sm.encryptor)
	assert.NotNil(t, sm.signer)
	assert.NotNil(t, sm.auditor)
}

func TestSecurityManager_PolicyManagement(t *testing.T) {
	logger := zap.NewNop()

	config := &SecurityManagerConfig{
		DefaultPolicy: &SecurityPolicy{
			Name: "default",
		},
		Logger: logger,
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)

	// Test adding a policy
	policy := &SecurityPolicy{
		Name: "test-policy",
		Encryption: &EncryptionConfig{
			Enabled:             true,
			Algorithm:           EncryptionAlgorithmA256GCM,
			KeySize:             32,             // 32 bytes for AES-256
			KeyRotationInterval: 24 * time.Hour, // 24 hours
		},
	}

	err = sm.AddPolicy("test-policy", policy)
	require.NoError(t, err)

	// Test getting policy
	retrievedPolicy, exists := sm.GetPolicy("test-policy")
	require.True(t, exists)
	assert.Equal(t, "test-policy", retrievedPolicy.Name)

	// Test removing policy
	err = sm.RemovePolicy("test-policy")
	require.NoError(t, err)

	// Test that policy was removed
	_, exists = sm.GetPolicy("test-policy")
	assert.False(t, exists)

	// Test that default policy cannot be removed
	err = sm.RemovePolicy("default")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove default security policy")
}

func TestSecurityManager_GetPolicyForMessage(t *testing.T) {
	logger := zap.NewNop()

	config := &SecurityManagerConfig{
		DefaultPolicy: &SecurityPolicy{
			Name: "default",
		},
		Logger: logger,
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)

	// Add topic-specific policy
	topicPolicy := &SecurityPolicy{
		Name:    "topic-specific",
		Enabled: true,
		Encryption: &EncryptionConfig{
			Enabled:             true,
			Algorithm:           EncryptionAlgorithmA256GCM,
			KeySize:             32,             // 32 bytes for AES-256
			KeyRotationInterval: 24 * time.Hour, // 24 hours
		},
		Topics: []string{"secure-topic"},
	}

	err = sm.AddPolicy("topic-specific", topicPolicy)
	require.NoError(t, err)

	// Test message with topic-specific policy
	message := &Message{
		ID:      "test-message-id",
		Topic:   "secure-topic",
		Payload: []byte("test payload"),
		Headers: map[string]string{
			"message_type": "test",
		},
		Timestamp: time.Now(),
	}

	policy, err := sm.GetPolicyForMessage(message)
	require.NoError(t, err)
	assert.Equal(t, "topic-specific", policy.Name)

	// Test message with default policy
	message.Topic = "regular-topic"
	policy, err = sm.GetPolicyForMessage(message)
	require.NoError(t, err)
	assert.Equal(t, "default", policy.Name)
}

func TestSecurityManager_GetMetrics(t *testing.T) {
	logger := zap.NewNop()

	config := &SecurityManagerConfig{
		DefaultPolicy: &SecurityPolicy{
			Name: "default",
			Encryption: &EncryptionConfig{
				Enabled:   true,
				Algorithm: EncryptionAlgorithmA256GCM,
			},
		},
		Logger: logger,
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)

	// Get metrics
	metrics := sm.GetMetrics()
	require.NotNil(t, metrics)
	assert.NotNil(t, metrics.CollectedAt)
}

func TestSecurityPolicy_Validate(t *testing.T) {
	// Test valid policy
	validPolicy := &SecurityPolicy{
		Name: "test-policy",
		Encryption: &EncryptionConfig{
			Enabled:             true,
			Algorithm:           EncryptionAlgorithmA256GCM,
			KeySize:             32,             // 32 bytes for AES-256
			KeyRotationInterval: 24 * time.Hour, // 24 hours
		},
	}
	err := validPolicy.Validate()
	require.NoError(t, err)

	// Test policy with empty name
	invalidPolicy := &SecurityPolicy{
		Name: "",
	}
	err = invalidPolicy.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy name is required")

	// Test policy with invalid encryption algorithm
	invalidPolicy = &SecurityPolicy{
		Name: "test-policy",
		Encryption: &EncryptionConfig{
			Enabled:   true,
			Algorithm: "invalid-algorithm",
		},
	}
	err = invalidPolicy.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid encryption algorithm")
}

func TestInMemoryKeyStore(t *testing.T) {
	store := NewInMemoryKeyStore()
	ctx := context.Background()

	// Test SetKey and GetKey
	err := store.SetKey(ctx, "test-key", []byte("test-value"))
	require.NoError(t, err)

	retrievedKey, err := store.GetKey(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-value"), retrievedKey)

	// Test GetKey for non-existent key
	_, err = store.GetKey(ctx, "non-existent-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")

	// Test ListKeys
	keys, err := store.ListKeys(ctx)
	require.NoError(t, err)
	assert.Contains(t, keys, "test-key")

	// Test DeleteKey
	err = store.DeleteKey(ctx, "test-key")
	require.NoError(t, err)

	// Verify key was deleted
	_, err = store.GetKey(ctx, "test-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

func TestEncryptionCache(t *testing.T) {
	cache := NewEncryptionCache()

	// Test Set and Get
	cache.Set("test-key", []byte("test-value"))

	retrievedValue, exists := cache.Get("test-key")
	assert.True(t, exists)
	assert.Equal(t, []byte("test-value"), retrievedValue)

	// Test Get for non-existent key
	_, exists = cache.Get("non-existent-key")
	assert.False(t, exists)
}

func TestSecurityAuditor(t *testing.T) {
	config := &AuditConfig{
		Enabled: true,
	}
	auditor := NewSecurityAuditor(config)

	ctx := context.Background()

	// Test logging security events
	event := &SecurityEvent{
		Type:        SecurityEventTypeEncrypt,
		MessageID:   "test-message-id",
		PolicyName:  "test-policy",
		Timestamp:   time.Now(),
		Success:     true,
		MessageType: "test",
	}

	auditor.LogSecurityEvent(ctx, event)

	// Test getting metrics
	metrics := auditor.GetMetrics()
	require.NotNil(t, metrics)
	assert.Equal(t, int64(1), metrics.EncryptionCount)

	// Test getting events
	events := auditor.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, event.Type, events[0].Type)
}

func TestSecurityManager_Compression(t *testing.T) {
	logger := zap.NewNop()

	config := &SecurityManagerConfig{
		DefaultPolicy: &SecurityPolicy{
			Name: "test-policy",
			Encryption: &EncryptionConfig{
				Enabled:   true,
				Algorithm: EncryptionAlgorithmA256GCM,
				Compression: &CompressionConfig{
					Enabled:   true,
					Algorithm: CompressionAlgorithmGzip,
					Threshold: 100, // Compress payloads larger than 100 bytes
				},
			},
		},
		Logger: logger,
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)

	// Test compression method
	testData := []byte("This is a test string that should be compressed")

	compressed, err := sm.compressMessage(testData, "gzip")
	require.NoError(t, err)

	// Compressed data should be different from original
	assert.NotEqual(t, testData, compressed)

	// Test decompression
	decompressed, err := sm.decompressMessage(compressed, "gzip")
	require.NoError(t, err)
	assert.Equal(t, testData, decompressed)
}
