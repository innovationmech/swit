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
	"time"
)

// Message represents a message in the messaging system.
// It contains all necessary information for routing, processing, and tracing messages.
type Message struct {
	// ID is the unique message identifier
	ID string `json:"id"`

	// Topic is the destination topic/queue name
	Topic string `json:"topic"`

	// Key is an optional routing/partition key for message ordering
	Key []byte `json:"key,omitempty"`

	// Payload is the actual message content
	Payload []byte `json:"payload"`

	// Headers contains message metadata as key-value pairs
	Headers map[string]string `json:"headers,omitempty"`

	// Timestamp indicates when the message was created
	Timestamp time.Time `json:"timestamp"`

	// CorrelationID is used for request-response patterns and distributed tracing
	CorrelationID string `json:"correlation_id,omitempty"`

	// ReplyTo specifies the topic for responses in request-response patterns
	ReplyTo string `json:"reply_to,omitempty"`

	// Priority sets the message priority (if supported by the broker)
	Priority int `json:"priority,omitempty"`

	// TTL specifies the time-to-live in seconds (if supported by the broker)
	TTL int `json:"ttl,omitempty"`

	// DeliveryAttempt tracks how many times delivery has been attempted
	DeliveryAttempt int `json:"delivery_attempt,omitempty"`

	// BrokerMetadata contains broker-specific metadata (not serialized)
	BrokerMetadata interface{} `json:"-"`
}

// PublishCallback defines the callback function signature for asynchronous publishing.
// The callback receives confirmation details on success or an error on failure.
type PublishCallback func(confirmation *PublishConfirmation, err error)

// PublishConfirmation contains details about a successful message publish operation.
type PublishConfirmation struct {
	// MessageID is the unique identifier assigned by the broker
	MessageID string `json:"message_id"`

	// Partition is the partition number where the message was stored
	Partition int32 `json:"partition"`

	// Offset is the position of the message within the partition
	Offset int64 `json:"offset"`

	// Timestamp is when the message was accepted by the broker
	Timestamp time.Time `json:"timestamp"`

	// Metadata contains additional broker-specific confirmation details
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SeekPosition defines where to seek in the message stream.
type SeekPosition struct {
	// Type specifies the seek type (beginning, end, offset, timestamp, position)
	Type SeekType `json:"type"`

	// Offset is the specific offset to seek to (used with SeekTypeOffset)
	Offset int64 `json:"offset,omitempty"`

	// Timestamp is the time to seek to (used with SeekTypeTimestamp)
	Timestamp time.Time `json:"timestamp,omitempty"`

	// Position is a broker-specific position identifier
	Position string `json:"position,omitempty"`

	// Topic specifies the topic to seek within (if different from subscription)
	Topic string `json:"topic,omitempty"`

	// Partition specifies the partition to seek within (if applicable)
	Partition int32 `json:"partition,omitempty"`
}

// SeekType defines the different types of seek operations.
type SeekType int

const (
	// SeekTypeBeginning seeks to the beginning of the stream
	SeekTypeBeginning SeekType = iota

	// SeekTypeEnd seeks to the end of the stream
	SeekTypeEnd

	// SeekTypeOffset seeks to a specific offset
	SeekTypeOffset

	// SeekTypeTimestamp seeks to a specific timestamp
	SeekTypeTimestamp

	// SeekTypePosition seeks to a broker-specific position
	SeekTypePosition
)

// String returns the string representation of SeekType.
func (st SeekType) String() string {
	switch st {
	case SeekTypeBeginning:
		return "beginning"
	case SeekTypeEnd:
		return "end"
	case SeekTypeOffset:
		return "offset"
	case SeekTypeTimestamp:
		return "timestamp"
	case SeekTypePosition:
		return "position"
	default:
		return "unknown"
	}
}
