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
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// RoutingInfo contains comprehensive routing information for message delivery.
// It provides abstractions for different routing strategies including topics,
// partitions, queues, and exchanges, enabling support for various message brokers.
type RoutingInfo struct {
	// Topic specifies the logical destination for the message
	Topic string `json:"topic" validate:"required,topic"`

	// Partition specifies the partition within the topic (for partitioned topics)
	Partition *int32 `json:"partition,omitempty" validate:"omitempty,min=0"`

	// PartitionKey is used for deterministic partition assignment
	PartitionKey string `json:"partition_key,omitempty"`

	// Queue specifies the queue name for queue-based routing
	Queue string `json:"queue,omitempty" validate:"omitempty,topic"`

	// Exchange specifies the exchange name for exchange-based routing (RabbitMQ)
	Exchange string `json:"exchange,omitempty"`

	// RoutingKey specifies the routing key for exchange routing
	RoutingKey string `json:"routing_key,omitempty"`

	// Headers contains routing headers for header-based routing
	Headers map[string]string `json:"headers,omitempty"`

	// Strategy specifies the routing strategy to use
	Strategy RoutingStrategy `json:"strategy"`

	// Options contains broker-specific routing options
	Options map[string]interface{} `json:"options,omitempty"`
}

// RoutingStrategy defines different message routing strategies.
type RoutingStrategy int

const (
	// RoutingStrategyDirect routes messages directly to a specific topic/queue
	RoutingStrategyDirect RoutingStrategy = iota

	// RoutingStrategyPartitioned routes messages to partitions within a topic
	RoutingStrategyPartitioned

	// RoutingStrategyRoundRobin distributes messages evenly across available partitions
	RoutingStrategyRoundRobin

	// RoutingStrategyKeyBased routes messages based on partition key hash
	RoutingStrategyKeyBased

	// RoutingStrategyExchange routes messages through an exchange (RabbitMQ style)
	RoutingStrategyExchange

	// RoutingStrategyHeaderBased routes messages based on header values
	RoutingStrategyHeaderBased

	// RoutingStrategyMulticast sends messages to multiple destinations
	RoutingStrategyMulticast

	// RoutingStrategyCustom allows for custom routing logic
	RoutingStrategyCustom
)

// String returns the string representation of RoutingStrategy.
func (rs RoutingStrategy) String() string {
	switch rs {
	case RoutingStrategyDirect:
		return "direct"
	case RoutingStrategyPartitioned:
		return "partitioned"
	case RoutingStrategyRoundRobin:
		return "round_robin"
	case RoutingStrategyKeyBased:
		return "key_based"
	case RoutingStrategyExchange:
		return "exchange"
	case RoutingStrategyHeaderBased:
		return "header_based"
	case RoutingStrategyMulticast:
		return "multicast"
	case RoutingStrategyCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// TopicConfig provides configuration for topic-based routing with
// support for partitioning and naming conventions.
type TopicConfig struct {
	// Name is the topic name
	Name string `json:"name" validate:"required,topic"`

	// Partitions specifies the number of partitions for the topic
	Partitions int32 `json:"partitions" validate:"min=1,max=1000"`

	// ReplicationFactor specifies replication level (Kafka)
	ReplicationFactor int16 `json:"replication_factor,omitempty" validate:"omitempty,min=1,max=10"`

	// RetentionMS specifies message retention time in milliseconds
	RetentionMS int64 `json:"retention_ms,omitempty" validate:"omitempty,min=3600000"` // Min 1 hour

	// CleanupPolicy specifies how to clean up old messages
	CleanupPolicy CleanupPolicy `json:"cleanup_policy"`

	// Config contains additional topic configuration properties
	Config map[string]string `json:"config,omitempty"`
}

// QueueConfig provides configuration for queue-based routing with
// durability and delivery options.
type QueueConfig struct {
	// Name is the queue name
	Name string `json:"name" validate:"required,topic"`

	// Durable specifies if the queue survives broker restarts
	Durable bool `json:"durable"`

	// AutoDelete specifies if the queue is deleted when not in use
	AutoDelete bool `json:"auto_delete"`

	// Exclusive specifies if only one consumer can access the queue
	Exclusive bool `json:"exclusive"`

	// MessageTTL specifies default message TTL in milliseconds
	MessageTTL int64 `json:"message_ttl,omitempty" validate:"omitempty,min=0"`

	// MaxLength specifies maximum number of messages in the queue
	MaxLength int64 `json:"max_length,omitempty" validate:"omitempty,min=1"`

	// DeadLetterExchange specifies dead letter exchange for failed messages
	DeadLetterExchange string `json:"dead_letter_exchange,omitempty"`

	// Arguments contains additional queue arguments
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ExchangeConfig provides configuration for exchange-based routing (RabbitMQ).
type ExchangeConfig struct {
	// Name is the exchange name
	Name string `json:"name" validate:"required"`

	// Type specifies the exchange type (direct, fanout, topic, headers)
	Type ExchangeType `json:"type"`

	// Durable specifies if the exchange survives broker restarts
	Durable bool `json:"durable"`

	// AutoDelete specifies if the exchange is deleted when not in use
	AutoDelete bool `json:"auto_delete"`

	// Internal specifies if the exchange is internal (used by broker)
	Internal bool `json:"internal"`

	// Arguments contains additional exchange arguments
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CleanupPolicy defines how old messages are cleaned up from topics.
type CleanupPolicy int

const (
	// CleanupPolicyDelete deletes old messages based on retention
	CleanupPolicyDelete CleanupPolicy = iota

	// CleanupPolicyCompact keeps only the latest value for each key
	CleanupPolicyCompact

	// CleanupPolicyDeleteAndCompact combines both policies
	CleanupPolicyDeleteAndCompact
)

// String returns the string representation of CleanupPolicy.
func (cp CleanupPolicy) String() string {
	switch cp {
	case CleanupPolicyDelete:
		return "delete"
	case CleanupPolicyCompact:
		return "compact"
	case CleanupPolicyDeleteAndCompact:
		return "delete,compact"
	default:
		return "unknown"
	}
}

// ExchangeType defines different exchange types for RabbitMQ-style routing.
type ExchangeType int

const (
	// ExchangeTypeDirect routes messages with exact routing key match
	ExchangeTypeDirect ExchangeType = iota

	// ExchangeTypeFanout routes messages to all bound queues
	ExchangeTypeFanout

	// ExchangeTypeTopic routes messages based on routing key patterns
	ExchangeTypeTopic

	// ExchangeTypeHeaders routes messages based on header values
	ExchangeTypeHeaders
)

// String returns the string representation of ExchangeType.
func (et ExchangeType) String() string {
	switch et {
	case ExchangeTypeDirect:
		return "direct"
	case ExchangeTypeFanout:
		return "fanout"
	case ExchangeTypeTopic:
		return "topic"
	case ExchangeTypeHeaders:
		return "headers"
	default:
		return "unknown"
	}
}

// PartitionResolver defines the interface for partition resolution strategies.
// Different brokers can implement custom partition resolution logic.
type PartitionResolver interface {
	// ResolvePartition determines the partition number for a given message
	ResolvePartition(routing *RoutingInfo, totalPartitions int32) (int32, error)

	// GetStrategyName returns the name of the partition strategy
	GetStrategyName() string
}

// DefaultPartitionResolver provides standard partition resolution strategies.
type DefaultPartitionResolver struct{}

// ResolvePartition implements partition resolution based on the routing strategy.
func (dpr *DefaultPartitionResolver) ResolvePartition(routing *RoutingInfo, totalPartitions int32) (int32, error) {
	if totalPartitions <= 0 {
		return 0, fmt.Errorf("total partitions must be positive, got: %d", totalPartitions)
	}

	switch routing.Strategy {
	case RoutingStrategyDirect:
		if routing.Partition != nil {
			if *routing.Partition >= totalPartitions {
				return 0, fmt.Errorf("partition %d exceeds total partitions %d", *routing.Partition, totalPartitions)
			}
			return *routing.Partition, nil
		}
		return 0, nil

	case RoutingStrategyKeyBased:
		if routing.PartitionKey == "" {
			return 0, fmt.Errorf("partition key is required for key-based routing")
		}
		return dpr.hashPartition(routing.PartitionKey, totalPartitions), nil

	case RoutingStrategyRoundRobin:
		// For round-robin, the caller should maintain state
		// This returns a hash-based partition as fallback
		return dpr.hashPartition(routing.Topic, totalPartitions), nil

	default:
		return 0, fmt.Errorf("unsupported routing strategy: %s", routing.Strategy.String())
	}
}

// GetStrategyName returns the name of this partition resolver.
func (dpr *DefaultPartitionResolver) GetStrategyName() string {
	return "default"
}

// hashPartition calculates a partition number based on a consistent hash of the key.
func (dpr *DefaultPartitionResolver) hashPartition(key string, totalPartitions int32) int32 {
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	return int32(hasher.Sum32()) % totalPartitions
}

// RoutingValidator provides validation for routing information.
type RoutingValidator struct {
	validator *validator.Validate
}

// NewRoutingValidator creates a new routing validator with custom validation rules.
func NewRoutingValidator() *RoutingValidator {
	v := validator.New()

	// Register custom validations
	v.RegisterValidation("topic", validateTopicName)

	return &RoutingValidator{validator: v}
}

// Validate performs comprehensive validation on routing information.
func (rv *RoutingValidator) Validate(routing *RoutingInfo) error {
	if routing == nil {
		return fmt.Errorf("routing info cannot be nil")
	}

	// Perform struct validation
	if err := rv.validator.Struct(routing); err != nil {
		return &MessageValidationError{
			Field:   getFieldName(err),
			Message: formatValidationError(err),
			Cause:   err,
		}
	}

	// Validate strategy-specific requirements
	return rv.validateStrategyRequirements(routing)
}

// validateStrategyRequirements validates that required fields are present for each strategy.
func (rv *RoutingValidator) validateStrategyRequirements(routing *RoutingInfo) error {
	switch routing.Strategy {
	case RoutingStrategyDirect:
		if routing.Topic == "" && routing.Queue == "" {
			return &MessageValidationError{
				Field:   "topic",
				Message: "either topic or queue is required for direct routing",
			}
		}

	case RoutingStrategyPartitioned:
		if routing.Topic == "" {
			return &MessageValidationError{
				Field:   "topic",
				Message: "topic is required for partitioned routing",
			}
		}

	case RoutingStrategyKeyBased:
		if routing.PartitionKey == "" {
			return &MessageValidationError{
				Field:   "partition_key",
				Message: "partition key is required for key-based routing",
			}
		}

	case RoutingStrategyExchange:
		if routing.Exchange == "" {
			return &MessageValidationError{
				Field:   "exchange",
				Message: "exchange is required for exchange-based routing",
			}
		}

	case RoutingStrategyHeaderBased:
		if len(routing.Headers) == 0 {
			return &MessageValidationError{
				Field:   "headers",
				Message: "routing headers are required for header-based routing",
			}
		}
	}

	return nil
}

// RoutingBuilder provides a fluent interface for constructing routing information.
type RoutingBuilder struct {
	routing *RoutingInfo
}

// NewRoutingBuilder creates a new routing builder with defaults.
func NewRoutingBuilder() *RoutingBuilder {
	return &RoutingBuilder{
		routing: &RoutingInfo{
			Strategy: RoutingStrategyDirect,
			Headers:  make(map[string]string),
			Options:  make(map[string]interface{}),
		},
	}
}

// WithTopic sets the destination topic.
func (rb *RoutingBuilder) WithTopic(topic string) *RoutingBuilder {
	rb.routing.Topic = topic
	return rb
}

// WithPartition sets a specific partition number.
func (rb *RoutingBuilder) WithPartition(partition int32) *RoutingBuilder {
	rb.routing.Partition = &partition
	rb.routing.Strategy = RoutingStrategyPartitioned
	return rb
}

// WithPartitionKey sets the partition key for key-based routing.
func (rb *RoutingBuilder) WithPartitionKey(key string) *RoutingBuilder {
	rb.routing.PartitionKey = key
	rb.routing.Strategy = RoutingStrategyKeyBased
	return rb
}

// WithQueue sets the destination queue.
func (rb *RoutingBuilder) WithQueue(queue string) *RoutingBuilder {
	rb.routing.Queue = queue
	return rb
}

// WithExchange sets the exchange and routing key for exchange-based routing.
func (rb *RoutingBuilder) WithExchange(exchange, routingKey string) *RoutingBuilder {
	rb.routing.Exchange = exchange
	rb.routing.RoutingKey = routingKey
	rb.routing.Strategy = RoutingStrategyExchange
	return rb
}

// WithHeader adds a routing header for header-based routing.
func (rb *RoutingBuilder) WithHeader(key, value string) *RoutingBuilder {
	if rb.routing.Headers == nil {
		rb.routing.Headers = make(map[string]string)
	}
	rb.routing.Headers[key] = value
	rb.routing.Strategy = RoutingStrategyHeaderBased
	return rb
}

// WithStrategy explicitly sets the routing strategy.
func (rb *RoutingBuilder) WithStrategy(strategy RoutingStrategy) *RoutingBuilder {
	rb.routing.Strategy = strategy
	return rb
}

// WithOption adds a broker-specific routing option.
func (rb *RoutingBuilder) WithOption(key string, value interface{}) *RoutingBuilder {
	if rb.routing.Options == nil {
		rb.routing.Options = make(map[string]interface{})
	}
	rb.routing.Options[key] = value
	return rb
}

// Build creates and validates the routing information.
func (rb *RoutingBuilder) Build() (*RoutingInfo, error) {
	validator := NewRoutingValidator()
	if err := validator.Validate(rb.routing); err != nil {
		return nil, err
	}

	// Create a copy to prevent modification after build
	result := *rb.routing
	return &result, nil
}

// TopicMatcher provides pattern matching for topic-based routing.
type TopicMatcher struct {
	patterns map[string]*regexp.Regexp
}

// NewTopicMatcher creates a new topic matcher with compiled patterns.
func NewTopicMatcher() *TopicMatcher {
	return &TopicMatcher{
		patterns: make(map[string]*regexp.Regexp),
	}
}

// AddPattern adds a topic pattern for matching.
// Patterns support wildcards: * matches any single level, # matches multiple levels.
func (tm *TopicMatcher) AddPattern(pattern string) error {
	regexPattern := tm.convertTopicPatternToRegex(pattern)
	compiled, err := regexp.Compile(regexPattern)
	if err != nil {
		return fmt.Errorf("invalid topic pattern '%s': %w", pattern, err)
	}

	tm.patterns[pattern] = compiled
	return nil
}

// Matches checks if a topic matches any of the registered patterns.
func (tm *TopicMatcher) Matches(topic string) []string {
	var matches []string
	for pattern, regex := range tm.patterns {
		if regex.MatchString(topic) {
			matches = append(matches, pattern)
		}
	}
	return matches
}

// convertTopicPatternToRegex converts topic wildcard patterns to regular expressions.
// * matches exactly one level (e.g., "user.*" matches "user.created" but not "user.profile.updated")
// # matches one or more levels (e.g., "user.#" matches "user.created" and "user.profile.updated")
func (tm *TopicMatcher) convertTopicPatternToRegex(pattern string) string {
	// Escape regex special characters except * and #
	escaped := regexp.QuoteMeta(pattern)

	// Replace escaped wildcards with regex equivalents
	escaped = strings.ReplaceAll(escaped, `\*`, `[^.]+`) // * matches one level
	escaped = strings.ReplaceAll(escaped, `\#`, `.+`)    // # matches multiple levels

	// Anchor the pattern to match the entire topic
	return "^" + escaped + "$"
}

// ConsistentHashPartitioner implements consistent hashing for partition assignment.
type ConsistentHashPartitioner struct {
	hashFunction func([]byte) uint64
}

// NewConsistentHashPartitioner creates a new consistent hash partitioner.
func NewConsistentHashPartitioner() *ConsistentHashPartitioner {
	return &ConsistentHashPartitioner{
		hashFunction: func(data []byte) uint64 {
			hasher := fnv.New64a()
			hasher.Write(data)
			return hasher.Sum64()
		},
	}
}

// GetPartition calculates the partition for a given key using consistent hashing.
func (chp *ConsistentHashPartitioner) GetPartition(key []byte, totalPartitions int32) int32 {
	if totalPartitions <= 0 {
		return 0
	}

	hash := chp.hashFunction(key)
	return int32(hash % uint64(totalPartitions))
}

// MD5HashPartitioner implements MD5-based partitioning for compatibility.
type MD5HashPartitioner struct{}

// NewMD5HashPartitioner creates a new MD5-based partitioner.
func NewMD5HashPartitioner() *MD5HashPartitioner {
	return &MD5HashPartitioner{}
}

// GetPartition calculates the partition using MD5 hash.
func (m5p *MD5HashPartitioner) GetPartition(key []byte, totalPartitions int32) int32 {
	if totalPartitions <= 0 {
		return 0
	}

	hash := md5.Sum(key)

	// Use the first 8 bytes of the hash as uint64
	hashValue := binary.BigEndian.Uint64(hash[:8])
	return int32(hashValue % uint64(totalPartitions))
}
