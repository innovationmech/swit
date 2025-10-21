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
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/saga"
)

// MessageConverter defines the interface for converting between different message formats.
// It provides bidirectional conversion between SagaEvent, SagaMessageProtocol,
// and messaging.Message formats.
type MessageConverter interface {
	// ToProtocolMessage converts a SagaEvent to SagaMessageProtocol format
	ToProtocolMessage(event *saga.SagaEvent) (*SagaMessageProtocol, error)

	// FromProtocolMessage converts a SagaMessageProtocol to SagaEvent
	FromProtocolMessage(msg *SagaMessageProtocol) (*saga.SagaEvent, error)

	// ToMessagingMessage converts a SagaEvent to messaging.Message format
	ToMessagingMessage(event *saga.SagaEvent) (*messaging.Message, error)

	// FromMessagingMessage converts a messaging.Message to SagaEvent
	FromMessagingMessage(msg *messaging.Message) (*saga.SagaEvent, error)

	// ProtocolToMessaging converts SagaMessageProtocol to messaging.Message
	ProtocolToMessaging(protocol *SagaMessageProtocol) (*messaging.Message, error)

	// MessagingToProtocol converts messaging.Message to SagaMessageProtocol
	MessagingToProtocol(msg *messaging.Message) (*SagaMessageProtocol, error)
}

// StandardMessageConverter implements MessageConverter with standard conversion logic.
type StandardMessageConverter struct {
	// serializer is used to serialize/deserialize event payloads
	serializer SagaEventSerializer

	// topicRouter handles topic routing
	topicRouter TopicRouter

	// validator validates protocol messages
	validator ProtocolValidator

	// versionComparer handles version comparison
	versionComparer ProtocolVersionComparer
}

// MessageConverterConfig contains configuration for message converter.
type MessageConverterConfig struct {
	// SerializerType specifies the serialization format ("json" or "protobuf")
	SerializerType string

	// EnableValidation enables message validation
	EnableValidation bool

	// StrictMode enables strict validation
	StrictMode bool
}

// NewStandardMessageConverter creates a new standard message converter.
func NewStandardMessageConverter(config *MessageConverterConfig) (*StandardMessageConverter, error) {
	if config == nil {
		config = &MessageConverterConfig{
			SerializerType:   "json",
			EnableValidation: true,
			StrictMode:       false,
		}
	}

	// Create serializer
	var serializer SagaEventSerializer
	switch config.SerializerType {
	case "json":
		serializer = NewJSONSagaEventSerializer()
	case "protobuf":
		serializer = NewProtobufSagaEventSerializer()
	default:
		return nil, fmt.Errorf("unsupported serializer type: %s", config.SerializerType)
	}

	// Create validator
	validator := NewStandardProtocolValidator()
	validator.StrictMode = config.StrictMode

	return &StandardMessageConverter{
		serializer:      serializer,
		topicRouter:     NewStandardTopicRouter(),
		validator:       validator,
		versionComparer: NewStandardVersionComparer(),
	}, nil
}

// ToProtocolMessage implements MessageConverter.ToProtocolMessage.
func (c *StandardMessageConverter) ToProtocolMessage(event *saga.SagaEvent) (*SagaMessageProtocol, error) {
	if event == nil {
		return nil, fmt.Errorf("event cannot be nil")
	}

	// Create protocol message
	protocol := &SagaMessageProtocol{
		Version:        CurrentProtocolVersion,
		MessageType:    MessageTypeSagaEvent,
		SagaID:         event.SagaID,
		EventID:        event.ID,
		EventType:      string(event.Type),
		Timestamp:      event.Timestamp,
		Headers:        BuildFromSagaEvent(event),
		Payload:        event.Data,
		StepID:         event.StepID,
		CorrelationID:  event.CorrelationID,
		Source:         event.Source,
		Service:        event.Service,
		ServiceVersion: event.ServiceVersion,
		TraceID:        event.TraceID,
		SpanID:         event.SpanID,
		ParentSpanID:   event.ParentSpanID,
		Attempt:        event.Attempt,
		MaxAttempts:    event.MaxAttempts,
		Duration:       event.Duration,
		Metadata:       event.Metadata,
	}

	// Use event version if specified
	if event.Version != "" {
		protocol.Version = event.Version
	}

	return protocol, nil
}

// FromProtocolMessage implements MessageConverter.FromProtocolMessage.
func (c *StandardMessageConverter) FromProtocolMessage(msg *SagaMessageProtocol) (*saga.SagaEvent, error) {
	if msg == nil {
		return nil, fmt.Errorf("protocol message cannot be nil")
	}

	// Validate protocol message
	if err := c.validator.Validate(msg); err != nil {
		return nil, fmt.Errorf("protocol validation failed: %w", err)
	}

	// Create Saga event
	event := &saga.SagaEvent{
		ID:             msg.EventID,
		SagaID:         msg.SagaID,
		StepID:         msg.StepID,
		Type:           saga.SagaEventType(msg.EventType),
		Version:        msg.Version,
		Timestamp:      msg.Timestamp,
		CorrelationID:  msg.CorrelationID,
		Data:           msg.Payload,
		Source:         msg.Source,
		Service:        msg.Service,
		ServiceVersion: msg.ServiceVersion,
		TraceID:        msg.TraceID,
		SpanID:         msg.SpanID,
		ParentSpanID:   msg.ParentSpanID,
		Attempt:        msg.Attempt,
		MaxAttempts:    msg.MaxAttempts,
		Duration:       msg.Duration,
		Metadata:       msg.Metadata,
	}

	return event, nil
}

// ToMessagingMessage implements MessageConverter.ToMessagingMessage.
func (c *StandardMessageConverter) ToMessagingMessage(event *saga.SagaEvent) (*messaging.Message, error) {
	if event == nil {
		return nil, fmt.Errorf("event cannot be nil")
	}

	// Serialize event data
	data, err := c.serializer.Serialize(context.Background(), event)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize event: %w", err)
	}

	// Determine topic
	topic := c.topicRouter.GetTopicForEvent(event.Type)

	// Build headers
	headers := BuildFromSagaEvent(event)
	headers[HeaderContentType] = c.serializer.ContentType()

	// Create messaging message
	msg := &messaging.Message{
		ID:            event.ID,
		Topic:         topic,
		Payload:       data,
		Timestamp:     event.Timestamp,
		CorrelationID: event.CorrelationID,
		Headers:       headers,
	}

	return msg, nil
}

// FromMessagingMessage implements MessageConverter.FromMessagingMessage.
func (c *StandardMessageConverter) FromMessagingMessage(msg *messaging.Message) (*saga.SagaEvent, error) {
	if msg == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	// Deserialize event
	event, err := c.serializer.Deserialize(context.Background(), msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize event: %w", err)
	}

	// Enrich event with headers
	c.enrichEventFromHeaders(event, msg.Headers)

	// Use message ID if event ID is not set
	if event.ID == "" {
		event.ID = msg.ID
	}

	// Use message timestamp if event timestamp is zero
	if event.Timestamp.IsZero() {
		event.Timestamp = msg.Timestamp
	}

	// Set correlation ID from message
	if event.CorrelationID == "" {
		event.CorrelationID = msg.CorrelationID
	}

	return event, nil
}

// ProtocolToMessaging implements MessageConverter.ProtocolToMessaging.
func (c *StandardMessageConverter) ProtocolToMessaging(protocol *SagaMessageProtocol) (*messaging.Message, error) {
	if protocol == nil {
		return nil, fmt.Errorf("protocol message cannot be nil")
	}

	// Validate protocol message
	if err := c.validator.Validate(protocol); err != nil {
		return nil, fmt.Errorf("protocol validation failed: %w", err)
	}

	// Convert protocol to Saga event first
	event, err := c.FromProtocolMessage(protocol)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protocol to event: %w", err)
	}

	// Then convert event to messaging message
	return c.ToMessagingMessage(event)
}

// MessagingToProtocol implements MessageConverter.MessagingToProtocol.
func (c *StandardMessageConverter) MessagingToProtocol(msg *messaging.Message) (*SagaMessageProtocol, error) {
	if msg == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	// Try to extract protocol info from headers first
	protocol := &SagaMessageProtocol{
		EventID:   msg.ID,
		Timestamp: msg.Timestamp,
		Headers:   msg.Headers,
	}

	// Extract from headers
	if sagaID := msg.Headers[HeaderSagaID]; sagaID != "" {
		protocol.SagaID = sagaID
	}

	if eventType := msg.Headers[HeaderEventType]; eventType != "" {
		protocol.EventType = eventType
	}

	if version := msg.Headers[HeaderVersion]; version != "" {
		protocol.Version = version
	} else {
		protocol.Version = CurrentProtocolVersion
	}

	if messageType := msg.Headers["message_type"]; messageType != "" {
		protocol.MessageType = messageType
	} else {
		protocol.MessageType = MessageTypeSagaEvent
	}

	if stepID := msg.Headers[HeaderStepID]; stepID != "" {
		protocol.StepID = stepID
	}

	if correlation := msg.Headers[HeaderCorrelation]; correlation != "" {
		protocol.CorrelationID = correlation
	} else {
		protocol.CorrelationID = msg.CorrelationID
	}

	if source := msg.Headers[HeaderSource]; source != "" {
		protocol.Source = source
	}

	if service := msg.Headers[HeaderService]; service != "" {
		protocol.Service = service
	}

	if serviceVersion := msg.Headers[HeaderServiceVersion]; serviceVersion != "" {
		protocol.ServiceVersion = serviceVersion
	}

	if traceID := msg.Headers[HeaderTraceID]; traceID != "" {
		protocol.TraceID = traceID
	}

	if spanID := msg.Headers[HeaderSpanID]; spanID != "" {
		protocol.SpanID = spanID
	}

	if parentSpanID := msg.Headers[HeaderParentSpanID]; parentSpanID != "" {
		protocol.ParentSpanID = parentSpanID
	}

	if attempt := msg.Headers[HeaderAttempt]; attempt != "" {
		if val, err := strconv.Atoi(attempt); err == nil {
			protocol.Attempt = val
		}
	}

	if maxAttempts := msg.Headers[HeaderMaxAttempts]; maxAttempts != "" {
		if val, err := strconv.Atoi(maxAttempts); err == nil {
			protocol.MaxAttempts = val
		}
	}

	if duration := msg.Headers[HeaderDuration]; duration != "" {
		if val, err := strconv.ParseInt(duration, 10, 64); err == nil {
			protocol.Duration = time.Duration(val) * time.Millisecond
		}
	}

	// Deserialize payload
	if len(msg.Payload) > 0 {
		var payload interface{}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			// If JSON unmarshal fails, store raw bytes
			protocol.Payload = msg.Payload
		} else {
			protocol.Payload = payload
		}
	}

	return protocol, nil
}

// enrichEventFromHeaders enriches a SagaEvent with information from message headers.
func (c *StandardMessageConverter) enrichEventFromHeaders(event *saga.SagaEvent, headers map[string]string) {
	if headers == nil {
		return
	}

	if sagaID := headers[HeaderSagaID]; sagaID != "" {
		event.SagaID = sagaID
	}

	if eventType := headers[HeaderEventType]; eventType != "" {
		event.Type = saga.SagaEventType(eventType)
	}

	if version := headers[HeaderVersion]; version != "" {
		event.Version = version
	}

	if stepID := headers[HeaderStepID]; stepID != "" {
		event.StepID = stepID
	}

	if correlation := headers[HeaderCorrelation]; correlation != "" {
		event.CorrelationID = correlation
	}

	if source := headers[HeaderSource]; source != "" {
		event.Source = source
	}

	if service := headers[HeaderService]; service != "" {
		event.Service = service
	}

	if serviceVersion := headers[HeaderServiceVersion]; serviceVersion != "" {
		event.ServiceVersion = serviceVersion
	}

	if traceID := headers[HeaderTraceID]; traceID != "" {
		event.TraceID = traceID
	}

	if spanID := headers[HeaderSpanID]; spanID != "" {
		event.SpanID = spanID
	}

	if parentSpanID := headers[HeaderParentSpanID]; parentSpanID != "" {
		event.ParentSpanID = parentSpanID
	}

	if timestamp := headers[HeaderTimestamp]; timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
			event.Timestamp = ts
		}
	}

	if attempt := headers[HeaderAttempt]; attempt != "" {
		if val, err := strconv.Atoi(attempt); err == nil {
			event.Attempt = val
		}
	}

	if maxAttempts := headers[HeaderMaxAttempts]; maxAttempts != "" {
		if val, err := strconv.Atoi(maxAttempts); err == nil {
			event.MaxAttempts = val
		}
	}

	if duration := headers[HeaderDuration]; duration != "" {
		if val, err := strconv.ParseInt(duration, 10, 64); err == nil {
			event.Duration = time.Duration(val) * time.Millisecond
		}
	}
}

// BatchMessageConverter provides batch conversion capabilities.
type BatchMessageConverter struct {
	converter MessageConverter
}

// NewBatchMessageConverter creates a new batch message converter.
func NewBatchMessageConverter(converter MessageConverter) *BatchMessageConverter {
	return &BatchMessageConverter{
		converter: converter,
	}
}

// ToMessagingMessages converts multiple SagaEvents to messaging.Messages.
func (b *BatchMessageConverter) ToMessagingMessages(events []*saga.SagaEvent) ([]*messaging.Message, error) {
	if events == nil {
		return nil, fmt.Errorf("events cannot be nil")
	}

	messages := make([]*messaging.Message, 0, len(events))
	for i, event := range events {
		msg, err := b.converter.ToMessagingMessage(event)
		if err != nil {
			return nil, fmt.Errorf("failed to convert event at index %d: %w", i, err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// FromMessagingMessages converts multiple messaging.Messages to SagaEvents.
func (b *BatchMessageConverter) FromMessagingMessages(messages []*messaging.Message) ([]*saga.SagaEvent, error) {
	if messages == nil {
		return nil, fmt.Errorf("messages cannot be nil")
	}

	events := make([]*saga.SagaEvent, 0, len(messages))
	for i, msg := range messages {
		event, err := b.converter.FromMessagingMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to convert message at index %d: %w", i, err)
		}
		events = append(events, event)
	}

	return events, nil
}

// ToProtocolMessages converts multiple SagaEvents to SagaMessageProtocols.
func (b *BatchMessageConverter) ToProtocolMessages(events []*saga.SagaEvent) ([]*SagaMessageProtocol, error) {
	if events == nil {
		return nil, fmt.Errorf("events cannot be nil")
	}

	protocols := make([]*SagaMessageProtocol, 0, len(events))
	for i, event := range events {
		protocol, err := b.converter.ToProtocolMessage(event)
		if err != nil {
			return nil, fmt.Errorf("failed to convert event at index %d: %w", i, err)
		}
		protocols = append(protocols, protocol)
	}

	return protocols, nil
}

// FromProtocolMessages converts multiple SagaMessageProtocols to SagaEvents.
func (b *BatchMessageConverter) FromProtocolMessages(protocols []*SagaMessageProtocol) ([]*saga.SagaEvent, error) {
	if protocols == nil {
		return nil, fmt.Errorf("protocols cannot be nil")
	}

	events := make([]*saga.SagaEvent, 0, len(protocols))
	for i, protocol := range protocols {
		event, err := b.converter.FromProtocolMessage(protocol)
		if err != nil {
			return nil, fmt.Errorf("failed to convert protocol at index %d: %w", i, err)
		}
		events = append(events, event)
	}

	return events, nil
}
