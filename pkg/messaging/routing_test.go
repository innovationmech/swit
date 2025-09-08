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
	"testing"
)

// TestRoutingValidator tests the routing validation functionality.
func TestRoutingValidator(t *testing.T) {
	validator := NewRoutingValidator()

	tests := []struct {
		name       string
		routing    *RoutingInfo
		wantError  bool
		errorField string
	}{
		{
			name: "valid direct routing",
			routing: &RoutingInfo{
				Topic:    "test.topic",
				Strategy: RoutingStrategyDirect,
				Headers:  make(map[string]string),
				Options:  make(map[string]interface{}),
			},
			wantError: false,
		},
		{
			name: "valid partitioned routing",
			routing: &RoutingInfo{
				Topic:     "test.topic",
				Partition: int32Ptr(0),
				Strategy:  RoutingStrategyPartitioned,
				Headers:   make(map[string]string),
				Options:   make(map[string]interface{}),
			},
			wantError: false,
		},
		{
			name: "valid key-based routing",
			routing: &RoutingInfo{
				Topic:        "test.topic",
				PartitionKey: "user-123",
				Strategy:     RoutingStrategyKeyBased,
				Headers:      make(map[string]string),
				Options:      make(map[string]interface{}),
			},
			wantError: false,
		},
		{
			name: "valid exchange routing",
			routing: &RoutingInfo{
				Exchange:   "user.exchange",
				RoutingKey: "user.created",
				Strategy:   RoutingStrategyExchange,
				Headers:    make(map[string]string),
				Options:    make(map[string]interface{}),
			},
			wantError: false,
		},
		{
			name: "valid header-based routing",
			routing: &RoutingInfo{
				Headers: map[string]string{
					"user-type": "premium",
					"region":    "us-east",
				},
				Strategy: RoutingStrategyHeaderBased,
				Options:  make(map[string]interface{}),
			},
			wantError: false,
		},
		{
			name: "direct routing without topic or queue",
			routing: &RoutingInfo{
				Strategy: RoutingStrategyDirect,
				Headers:  make(map[string]string),
				Options:  make(map[string]interface{}),
			},
			wantError:  true,
			errorField: "topic",
		},
		{
			name: "partitioned routing without topic",
			routing: &RoutingInfo{
				Strategy: RoutingStrategyPartitioned,
				Headers:  make(map[string]string),
				Options:  make(map[string]interface{}),
			},
			wantError:  true,
			errorField: "topic",
		},
		{
			name: "key-based routing without partition key",
			routing: &RoutingInfo{
				Topic:    "test.topic",
				Strategy: RoutingStrategyKeyBased,
				Headers:  make(map[string]string),
				Options:  make(map[string]interface{}),
			},
			wantError:  true,
			errorField: "partition_key",
		},
		{
			name: "exchange routing without exchange",
			routing: &RoutingInfo{
				RoutingKey: "user.created",
				Strategy:   RoutingStrategyExchange,
				Headers:    make(map[string]string),
				Options:    make(map[string]interface{}),
			},
			wantError:  true,
			errorField: "exchange",
		},
		{
			name: "header-based routing without headers",
			routing: &RoutingInfo{
				Strategy: RoutingStrategyHeaderBased,
				Headers:  make(map[string]string), // Empty headers
				Options:  make(map[string]interface{}),
			},
			wantError:  true,
			errorField: "headers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.routing)

			if tt.wantError {
				if err == nil {
					t.Error("expected validation error, got nil")
					return
				}

				if validErr, ok := err.(*MessageValidationError); ok {
					if tt.errorField != "" && validErr.Field != tt.errorField {
						t.Errorf("expected error field %s, got %s", tt.errorField, validErr.Field)
					}
				} else {
					t.Errorf("expected MessageValidationError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected validation error: %v", err)
				}
			}
		})
	}
}

// TestRoutingStrategy tests the RoutingStrategy string representation.
func TestRoutingStrategy(t *testing.T) {
	tests := []struct {
		strategy RoutingStrategy
		expected string
	}{
		{RoutingStrategyDirect, "direct"},
		{RoutingStrategyPartitioned, "partitioned"},
		{RoutingStrategyRoundRobin, "round_robin"},
		{RoutingStrategyKeyBased, "key_based"},
		{RoutingStrategyExchange, "exchange"},
		{RoutingStrategyHeaderBased, "header_based"},
		{RoutingStrategyMulticast, "multicast"},
		{RoutingStrategyCustom, "custom"},
		{RoutingStrategy(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.strategy.String()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestCleanupPolicy tests the CleanupPolicy string representation.
func TestCleanupPolicy(t *testing.T) {
	tests := []struct {
		policy   CleanupPolicy
		expected string
	}{
		{CleanupPolicyDelete, "delete"},
		{CleanupPolicyCompact, "compact"},
		{CleanupPolicyDeleteAndCompact, "delete,compact"},
		{CleanupPolicy(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.policy.String()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestExchangeType tests the ExchangeType string representation.
func TestExchangeType(t *testing.T) {
	tests := []struct {
		exchangeType ExchangeType
		expected     string
	}{
		{ExchangeTypeDirect, "direct"},
		{ExchangeTypeFanout, "fanout"},
		{ExchangeTypeTopic, "topic"},
		{ExchangeTypeHeaders, "headers"},
		{ExchangeType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.exchangeType.String()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestDefaultPartitionResolver tests the default partition resolution.
func TestDefaultPartitionResolver(t *testing.T) {
	resolver := &DefaultPartitionResolver{}

	tests := []struct {
		name              string
		routing           *RoutingInfo
		totalPartitions   int32
		expectedPartition int32
		wantError         bool
	}{
		{
			name: "direct routing with specific partition",
			routing: &RoutingInfo{
				Strategy:  RoutingStrategyDirect,
				Partition: int32Ptr(2),
			},
			totalPartitions:   5,
			expectedPartition: 2,
			wantError:         false,
		},
		{
			name: "direct routing without partition",
			routing: &RoutingInfo{
				Strategy: RoutingStrategyDirect,
			},
			totalPartitions:   5,
			expectedPartition: 0,
			wantError:         false,
		},
		{
			name: "key-based routing",
			routing: &RoutingInfo{
				Strategy:     RoutingStrategyKeyBased,
				PartitionKey: "user-123",
			},
			totalPartitions: 5,
			wantError:       false,
			// We can't predict exact partition due to hashing, just verify no error
		},
		{
			name: "partition exceeds total",
			routing: &RoutingInfo{
				Strategy:  RoutingStrategyDirect,
				Partition: int32Ptr(10),
			},
			totalPartitions: 5,
			wantError:       true,
		},
		{
			name: "key-based without partition key",
			routing: &RoutingInfo{
				Strategy: RoutingStrategyKeyBased,
			},
			totalPartitions: 5,
			wantError:       true,
		},
		{
			name: "invalid total partitions",
			routing: &RoutingInfo{
				Strategy: RoutingStrategyDirect,
			},
			totalPartitions: 0,
			wantError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partition, err := resolver.ResolvePartition(tt.routing, tt.totalPartitions)

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}

				// For direct routing with specific partition, verify exact match
				if tt.routing.Strategy == RoutingStrategyDirect && tt.routing.Partition != nil {
					if partition != tt.expectedPartition {
						t.Errorf("expected partition %d, got %d", tt.expectedPartition, partition)
					}
				}

				// For all strategies, verify partition is within valid range
				if partition < 0 || partition >= tt.totalPartitions {
					t.Errorf("partition %d is out of range [0, %d)", partition, tt.totalPartitions)
				}
			}
		})
	}
}

// TestRoutingBuilder tests the routing builder functionality.
func TestRoutingBuilder(t *testing.T) {
	t.Run("build direct routing", func(t *testing.T) {
		routing, err := NewRoutingBuilder().
			WithTopic("test.topic").
			Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if routing.Topic != "test.topic" {
			t.Errorf("expected topic 'test.topic', got %s", routing.Topic)
		}

		if routing.Strategy != RoutingStrategyDirect {
			t.Errorf("expected strategy %v, got %v", RoutingStrategyDirect, routing.Strategy)
		}
	})

	t.Run("build partitioned routing", func(t *testing.T) {
		routing, err := NewRoutingBuilder().
			WithTopic("test.topic").
			WithPartition(3).
			Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if routing.Partition == nil || *routing.Partition != 3 {
			t.Errorf("expected partition 3, got %v", routing.Partition)
		}

		if routing.Strategy != RoutingStrategyPartitioned {
			t.Errorf("expected strategy %v, got %v", RoutingStrategyPartitioned, routing.Strategy)
		}
	})

	t.Run("build key-based routing", func(t *testing.T) {
		routing, err := NewRoutingBuilder().
			WithTopic("test.topic").
			WithPartitionKey("user-123").
			Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if routing.PartitionKey != "user-123" {
			t.Errorf("expected partition key 'user-123', got %s", routing.PartitionKey)
		}

		if routing.Strategy != RoutingStrategyKeyBased {
			t.Errorf("expected strategy %v, got %v", RoutingStrategyKeyBased, routing.Strategy)
		}
	})

	t.Run("build exchange routing", func(t *testing.T) {
		routing, err := NewRoutingBuilder().
			WithExchange("user.exchange", "user.created").
			Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if routing.Exchange != "user.exchange" {
			t.Errorf("expected exchange 'user.exchange', got %s", routing.Exchange)
		}

		if routing.RoutingKey != "user.created" {
			t.Errorf("expected routing key 'user.created', got %s", routing.RoutingKey)
		}

		if routing.Strategy != RoutingStrategyExchange {
			t.Errorf("expected strategy %v, got %v", RoutingStrategyExchange, routing.Strategy)
		}
	})

	t.Run("build header-based routing", func(t *testing.T) {
		routing, err := NewRoutingBuilder().
			WithHeader("user-type", "premium").
			WithHeader("region", "us-east").
			Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if routing.Headers["user-type"] != "premium" {
			t.Errorf("expected header 'premium', got %s", routing.Headers["user-type"])
		}

		if routing.Headers["region"] != "us-east" {
			t.Errorf("expected header 'us-east', got %s", routing.Headers["region"])
		}

		if routing.Strategy != RoutingStrategyHeaderBased {
			t.Errorf("expected strategy %v, got %v", RoutingStrategyHeaderBased, routing.Strategy)
		}
	})

	t.Run("build with options", func(t *testing.T) {
		routing, err := NewRoutingBuilder().
			WithTopic("test.topic").
			WithOption("timeout", 5000).
			WithOption("retry", true).
			Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if routing.Options["timeout"] != 5000 {
			t.Errorf("expected option 5000, got %v", routing.Options["timeout"])
		}

		if routing.Options["retry"] != true {
			t.Errorf("expected option true, got %v", routing.Options["retry"])
		}
	})
}

// TestTopicMatcher tests the topic pattern matching functionality.
func TestTopicMatcher(t *testing.T) {
	matcher := NewTopicMatcher()

	// Add patterns
	patterns := []string{
		"user.*",
		"user.#",
		"order.payment.*",
		"system.*.critical",
		"*.events",
	}

	for _, pattern := range patterns {
		if err := matcher.AddPattern(pattern); err != nil {
			t.Fatalf("failed to add pattern %s: %v", pattern, err)
		}
	}

	tests := []struct {
		topic           string
		expectedMatches []string
	}{
		{
			topic:           "user.created",
			expectedMatches: []string{"user.*", "user.#"},
		},
		{
			topic:           "user.profile.updated",
			expectedMatches: []string{"user.#"},
		},
		{
			topic:           "order.payment.completed",
			expectedMatches: []string{"order.payment.*"},
		},
		{
			topic:           "system.database.critical",
			expectedMatches: []string{"system.*.critical"},
		},
		{
			topic:           "notification.events",
			expectedMatches: []string{"*.events"},
		},
		{
			topic:           "unmatched.topic",
			expectedMatches: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			matches := matcher.Matches(tt.topic)

			if len(matches) != len(tt.expectedMatches) {
				t.Errorf("expected %d matches, got %d: %v", len(tt.expectedMatches), len(matches), matches)
				return
			}

			// Check that all expected patterns are present
			for _, expected := range tt.expectedMatches {
				found := false
				for _, match := range matches {
					if match == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected pattern %s not found in matches: %v", expected, matches)
				}
			}
		})
	}
}

// TestConsistentHashPartitioner tests the consistent hash partitioner.
func TestConsistentHashPartitioner(t *testing.T) {
	partitioner := NewConsistentHashPartitioner()

	tests := []struct {
		name            string
		key             []byte
		totalPartitions int32
		expectedValid   bool
	}{
		{
			name:            "valid partitioning",
			key:             []byte("user-123"),
			totalPartitions: 5,
			expectedValid:   true,
		},
		{
			name:            "single partition",
			key:             []byte("test"),
			totalPartitions: 1,
			expectedValid:   true,
		},
		{
			name:            "zero partitions",
			key:             []byte("test"),
			totalPartitions: 0,
			expectedValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partition := partitioner.GetPartition(tt.key, tt.totalPartitions)

			if tt.expectedValid {
				if partition < 0 || partition >= tt.totalPartitions {
					t.Errorf("partition %d is out of range [0, %d)", partition, tt.totalPartitions)
				}
			} else {
				if partition != 0 {
					t.Errorf("expected partition 0 for invalid input, got %d", partition)
				}
			}
		})
	}

	// Test consistency: same key should always map to same partition
	key := []byte("consistent-test")
	totalPartitions := int32(10)

	firstResult := partitioner.GetPartition(key, totalPartitions)
	for i := 0; i < 100; i++ {
		result := partitioner.GetPartition(key, totalPartitions)
		if result != firstResult {
			t.Errorf("inconsistent partitioning: first=%d, iteration %d=%d", firstResult, i, result)
			break
		}
	}
}

// TestMD5HashPartitioner tests the MD5 hash partitioner.
func TestMD5HashPartitioner(t *testing.T) {
	partitioner := NewMD5HashPartitioner()

	tests := []struct {
		name            string
		key             []byte
		totalPartitions int32
		expectedValid   bool
	}{
		{
			name:            "valid partitioning",
			key:             []byte("user-456"),
			totalPartitions: 8,
			expectedValid:   true,
		},
		{
			name:            "single partition",
			key:             []byte("test"),
			totalPartitions: 1,
			expectedValid:   true,
		},
		{
			name:            "zero partitions",
			key:             []byte("test"),
			totalPartitions: 0,
			expectedValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partition := partitioner.GetPartition(tt.key, tt.totalPartitions)

			if tt.expectedValid {
				if partition < 0 || partition >= tt.totalPartitions {
					t.Errorf("partition %d is out of range [0, %d)", partition, tt.totalPartitions)
				}
			} else {
				if partition != 0 {
					t.Errorf("expected partition 0 for invalid input, got %d", partition)
				}
			}
		})
	}

	// Test consistency: same key should always map to same partition
	key := []byte("md5-test")
	totalPartitions := int32(16)

	firstResult := partitioner.GetPartition(key, totalPartitions)
	for i := 0; i < 100; i++ {
		result := partitioner.GetPartition(key, totalPartitions)
		if result != firstResult {
			t.Errorf("inconsistent partitioning: first=%d, iteration %d=%d", firstResult, i, result)
			break
		}
	}
}

// BenchmarkDefaultPartitionResolver benchmarks partition resolution.
func BenchmarkDefaultPartitionResolver(b *testing.B) {
	resolver := &DefaultPartitionResolver{}
	routing := &RoutingInfo{
		Strategy:     RoutingStrategyKeyBased,
		PartitionKey: "benchmark-key",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.ResolvePartition(routing, 16)
	}
}

// BenchmarkConsistentHashPartitioner benchmarks consistent hash partitioning.
func BenchmarkConsistentHashPartitioner(b *testing.B) {
	partitioner := NewConsistentHashPartitioner()
	key := []byte("benchmark-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = partitioner.GetPartition(key, 16)
	}
}

// Helper function to create int32 pointer
func int32Ptr(v int32) *int32 {
	return &v
}
