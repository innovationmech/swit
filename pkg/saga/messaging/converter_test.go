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

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStandardMessageConverter(t *testing.T) {
	tests := []struct {
		name      string
		config    *MessageConverterConfig
		wantError bool
	}{
		{
			name: "valid JSON config",
			config: &MessageConverterConfig{
				SerializerType:   "json",
				EnableValidation: true,
				StrictMode:       false,
			},
			wantError: false,
		},
		{
			name: "valid protobuf config",
			config: &MessageConverterConfig{
				SerializerType:   "protobuf",
				EnableValidation: true,
				StrictMode:       false,
			},
			wantError: false,
		},
		{
			name:      "nil config uses defaults",
			config:    nil,
			wantError: false,
		},
		{
			name: "unsupported serializer type",
			config: &MessageConverterConfig{
				SerializerType: "invalid",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := NewStandardMessageConverter(tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, converter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, converter)
			}
		})
	}
}

func TestStandardMessageConverter_ToProtocolMessage(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	now := time.Now()
	event := &saga.SagaEvent{
		ID:            "event-123",
		SagaID:        "saga-456",
		StepID:        "step-789",
		Type:          saga.EventSagaStarted,
		Version:       ProtocolVersion1_1,
		Timestamp:     now,
		CorrelationID: "corr-001",
		Data: map[string]interface{}{
			"order_id": "order-123",
		},
		Source:         "order-service",
		Service:        "payment-service",
		ServiceVersion: "1.0.0",
		TraceID:        "trace-001",
		SpanID:         "span-001",
		ParentSpanID:   "parent-span-001",
		Attempt:        2,
		MaxAttempts:    3,
		Duration:       100 * time.Millisecond,
		Metadata: map[string]interface{}{
			"custom_key": "custom_value",
		},
	}

	protocol, err := converter.ToProtocolMessage(event)
	require.NoError(t, err)
	assert.NotNil(t, protocol)

	assert.Equal(t, ProtocolVersion1_1, protocol.Version)
	assert.Equal(t, MessageTypeSagaEvent, protocol.MessageType)
	assert.Equal(t, "saga-456", protocol.SagaID)
	assert.Equal(t, "event-123", protocol.EventID)
	assert.Equal(t, string(saga.EventSagaStarted), protocol.EventType)
	assert.Equal(t, now, protocol.Timestamp)
	assert.Equal(t, "step-789", protocol.StepID)
	assert.Equal(t, "corr-001", protocol.CorrelationID)
	assert.Equal(t, "order-service", protocol.Source)
	assert.Equal(t, "payment-service", protocol.Service)
	assert.Equal(t, "1.0.0", protocol.ServiceVersion)
	assert.Equal(t, "trace-001", protocol.TraceID)
	assert.Equal(t, "span-001", protocol.SpanID)
	assert.Equal(t, "parent-span-001", protocol.ParentSpanID)
	assert.Equal(t, 2, protocol.Attempt)
	assert.Equal(t, 3, protocol.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, protocol.Duration)
	assert.NotNil(t, protocol.Payload)
	assert.NotNil(t, protocol.Headers)
	assert.NotNil(t, protocol.Metadata)
}

func TestStandardMessageConverter_ToProtocolMessage_NilEvent(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	protocol, err := converter.ToProtocolMessage(nil)
	assert.Error(t, err)
	assert.Nil(t, protocol)
}

func TestStandardMessageConverter_FromProtocolMessage(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	now := time.Now()
	protocol := &SagaMessageProtocol{
		Version:     ProtocolVersion1_1,
		MessageType: MessageTypeSagaEvent,
		SagaID:      "saga-456",
		EventID:     "event-123",
		EventType:   string(saga.EventSagaStarted),
		Timestamp:   now,
		Headers: map[string]string{
			HeaderSagaID:    "saga-456",
			HeaderEventID:   "event-123",
			HeaderEventType: string(saga.EventSagaStarted),
			HeaderTimestamp: now.Format(time.RFC3339Nano),
		},
		Payload: map[string]interface{}{
			"order_id": "order-123",
		},
		StepID:         "step-789",
		CorrelationID:  "corr-001",
		Source:         "order-service",
		Service:        "payment-service",
		ServiceVersion: "1.0.0",
		TraceID:        "trace-001",
		SpanID:         "span-001",
		ParentSpanID:   "parent-span-001",
		Attempt:        2,
		MaxAttempts:    3,
		Duration:       100 * time.Millisecond,
	}

	event, err := converter.FromProtocolMessage(protocol)
	require.NoError(t, err)
	assert.NotNil(t, event)

	assert.Equal(t, "event-123", event.ID)
	assert.Equal(t, "saga-456", event.SagaID)
	assert.Equal(t, "step-789", event.StepID)
	assert.Equal(t, saga.EventSagaStarted, event.Type)
	assert.Equal(t, ProtocolVersion1_1, event.Version)
	assert.Equal(t, now, event.Timestamp)
	assert.Equal(t, "corr-001", event.CorrelationID)
	assert.Equal(t, "order-service", event.Source)
	assert.Equal(t, "payment-service", event.Service)
	assert.Equal(t, "1.0.0", event.ServiceVersion)
	assert.Equal(t, "trace-001", event.TraceID)
	assert.Equal(t, "span-001", event.SpanID)
	assert.Equal(t, "parent-span-001", event.ParentSpanID)
	assert.Equal(t, 2, event.Attempt)
	assert.Equal(t, 3, event.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, event.Duration)
}

func TestStandardMessageConverter_FromProtocolMessage_NilProtocol(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	event, err := converter.FromProtocolMessage(nil)
	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestStandardMessageConverter_FromProtocolMessage_InvalidProtocol(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	// Missing required fields
	protocol := &SagaMessageProtocol{
		Version: ProtocolVersion1_1,
		SagaID:  "saga-456",
		// Missing EventID, EventType, MessageType, Timestamp
	}

	event, err := converter.FromProtocolMessage(protocol)
	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestStandardMessageConverter_ToMessagingMessage(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	now := time.Now()
	event := &saga.SagaEvent{
		ID:            "event-123",
		SagaID:        "saga-456",
		Type:          saga.EventSagaStarted,
		Version:       ProtocolVersion1_1,
		Timestamp:     now,
		CorrelationID: "corr-001",
		Data: map[string]interface{}{
			"order_id": "order-123",
		},
	}

	msg, err := converter.ToMessagingMessage(event)
	require.NoError(t, err)
	assert.NotNil(t, msg)

	assert.Equal(t, "event-123", msg.ID)
	assert.Equal(t, "saga.events.saga.started", msg.Topic)
	assert.Equal(t, now, msg.Timestamp)
	assert.Equal(t, "corr-001", msg.CorrelationID)
	assert.NotNil(t, msg.Payload)
	assert.NotNil(t, msg.Headers)

	// Check headers
	assert.Equal(t, "saga-456", msg.Headers[HeaderSagaID])
	assert.Equal(t, "event-123", msg.Headers[HeaderEventID])
	assert.Equal(t, string(saga.EventSagaStarted), msg.Headers[HeaderEventType])
}

func TestStandardMessageConverter_ToMessagingMessage_NilEvent(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	msg, err := converter.ToMessagingMessage(nil)
	assert.Error(t, err)
	assert.Nil(t, msg)
}

func TestStandardMessageConverter_FromMessagingMessage(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	// First create a message from an event
	now := time.Now()
	originalEvent := &saga.SagaEvent{
		ID:            "event-123",
		SagaID:        "saga-456",
		Type:          saga.EventSagaStarted,
		Version:       ProtocolVersion1_1,
		Timestamp:     now,
		CorrelationID: "corr-001",
		Data: map[string]interface{}{
			"order_id": "order-123",
		},
	}

	msg, err := converter.ToMessagingMessage(originalEvent)
	require.NoError(t, err)

	// Now convert back to event
	event, err := converter.FromMessagingMessage(msg)
	require.NoError(t, err)
	assert.NotNil(t, event)

	assert.Equal(t, "event-123", event.ID)
	assert.Equal(t, "saga-456", event.SagaID)
	assert.Equal(t, saga.EventSagaStarted, event.Type)
	assert.Equal(t, ProtocolVersion1_1, event.Version)
	assert.Equal(t, "corr-001", event.CorrelationID)
}

func TestStandardMessageConverter_FromMessagingMessage_NilMessage(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	event, err := converter.FromMessagingMessage(nil)
	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestStandardMessageConverter_ProtocolToMessaging(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	now := time.Now()
	protocol := &SagaMessageProtocol{
		Version:     ProtocolVersion1_1,
		MessageType: MessageTypeSagaEvent,
		SagaID:      "saga-456",
		EventID:     "event-123",
		EventType:   string(saga.EventSagaStarted),
		Timestamp:   now,
		Headers: map[string]string{
			HeaderSagaID:    "saga-456",
			HeaderEventID:   "event-123",
			HeaderEventType: string(saga.EventSagaStarted),
			HeaderTimestamp: now.Format(time.RFC3339Nano),
		},
	}

	msg, err := converter.ProtocolToMessaging(protocol)
	require.NoError(t, err)
	assert.NotNil(t, msg)

	assert.Equal(t, "event-123", msg.ID)
	assert.Equal(t, "saga.events.saga.started", msg.Topic)
	assert.NotNil(t, msg.Payload)
}

func TestStandardMessageConverter_MessagingToProtocol(t *testing.T) {
	now := time.Now()
	msg := &messaging.Message{
		ID:            "event-123",
		Topic:         "saga.events.saga.started",
		Timestamp:     now,
		CorrelationID: "corr-001",
		Headers: map[string]string{
			HeaderSagaID:    "saga-456",
			HeaderEventID:   "event-123",
			HeaderEventType: string(saga.EventSagaStarted),
			HeaderVersion:   ProtocolVersion1_1,
			HeaderTimestamp: now.Format(time.RFC3339Nano),
		},
		Payload: []byte(`{"order_id":"order-123"}`),
	}

	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	protocol, err := converter.MessagingToProtocol(msg)
	require.NoError(t, err)
	assert.NotNil(t, protocol)

	assert.Equal(t, "event-123", protocol.EventID)
	assert.Equal(t, "saga-456", protocol.SagaID)
	assert.Equal(t, string(saga.EventSagaStarted), protocol.EventType)
	assert.Equal(t, ProtocolVersion1_1, protocol.Version)
	assert.Equal(t, "corr-001", protocol.CorrelationID)
}

func TestStandardMessageConverter_MessagingToProtocol_NilMessage(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	protocol, err := converter.MessagingToProtocol(nil)
	assert.Error(t, err)
	assert.Nil(t, protocol)
}

func TestStandardMessageConverter_RoundTrip(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	now := time.Now()
	originalEvent := &saga.SagaEvent{
		ID:            "event-123",
		SagaID:        "saga-456",
		StepID:        "step-789",
		Type:          saga.EventSagaStarted,
		Version:       ProtocolVersion1_1,
		Timestamp:     now,
		CorrelationID: "corr-001",
		Data: map[string]interface{}{
			"order_id": "order-123",
		},
		Source:      "order-service",
		TraceID:     "trace-001",
		Attempt:     1,
		MaxAttempts: 3,
	}

	// Event -> Protocol -> Event
	protocol, err := converter.ToProtocolMessage(originalEvent)
	require.NoError(t, err)

	event1, err := converter.FromProtocolMessage(protocol)
	require.NoError(t, err)

	assert.Equal(t, originalEvent.ID, event1.ID)
	assert.Equal(t, originalEvent.SagaID, event1.SagaID)
	assert.Equal(t, originalEvent.Type, event1.Type)

	// Event -> Messaging -> Event
	msg, err := converter.ToMessagingMessage(originalEvent)
	require.NoError(t, err)

	event2, err := converter.FromMessagingMessage(msg)
	require.NoError(t, err)

	assert.Equal(t, originalEvent.ID, event2.ID)
	assert.Equal(t, originalEvent.SagaID, event2.SagaID)
	assert.Equal(t, originalEvent.Type, event2.Type)
}

func TestBatchMessageConverter_ToMessagingMessages(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	batch := NewBatchMessageConverter(converter)

	events := []*saga.SagaEvent{
		{
			ID:        "event-1",
			SagaID:    "saga-1",
			Type:      saga.EventSagaStarted,
			Timestamp: time.Now(),
		},
		{
			ID:        "event-2",
			SagaID:    "saga-2",
			Type:      saga.EventSagaCompleted,
			Timestamp: time.Now(),
		},
	}

	messages, err := batch.ToMessagingMessages(events)
	require.NoError(t, err)
	assert.Len(t, messages, 2)

	assert.Equal(t, "event-1", messages[0].ID)
	assert.Equal(t, "event-2", messages[1].ID)
}

func TestBatchMessageConverter_ToMessagingMessages_NilEvents(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	batch := NewBatchMessageConverter(converter)

	messages, err := batch.ToMessagingMessages(nil)
	assert.Error(t, err)
	assert.Nil(t, messages)
}

func TestBatchMessageConverter_FromMessagingMessages(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	batch := NewBatchMessageConverter(converter)

	// First create messages
	events := []*saga.SagaEvent{
		{
			ID:        "event-1",
			SagaID:    "saga-1",
			Type:      saga.EventSagaStarted,
			Timestamp: time.Now(),
		},
		{
			ID:        "event-2",
			SagaID:    "saga-2",
			Type:      saga.EventSagaCompleted,
			Timestamp: time.Now(),
		},
	}

	messages, err := batch.ToMessagingMessages(events)
	require.NoError(t, err)

	// Now convert back
	resultEvents, err := batch.FromMessagingMessages(messages)
	require.NoError(t, err)
	assert.Len(t, resultEvents, 2)

	assert.Equal(t, "event-1", resultEvents[0].ID)
	assert.Equal(t, "event-2", resultEvents[1].ID)
}

func TestBatchMessageConverter_ToProtocolMessages(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	batch := NewBatchMessageConverter(converter)

	events := []*saga.SagaEvent{
		{
			ID:        "event-1",
			SagaID:    "saga-1",
			Type:      saga.EventSagaStarted,
			Timestamp: time.Now(),
		},
		{
			ID:        "event-2",
			SagaID:    "saga-2",
			Type:      saga.EventSagaCompleted,
			Timestamp: time.Now(),
		},
	}

	protocols, err := batch.ToProtocolMessages(events)
	require.NoError(t, err)
	assert.Len(t, protocols, 2)

	assert.Equal(t, "event-1", protocols[0].EventID)
	assert.Equal(t, "event-2", protocols[1].EventID)
}

func TestBatchMessageConverter_FromProtocolMessages(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	batch := NewBatchMessageConverter(converter)

	now := time.Now()
	protocols := []*SagaMessageProtocol{
		{
			Version:     ProtocolVersion1_1,
			MessageType: MessageTypeSagaEvent,
			SagaID:      "saga-1",
			EventID:     "event-1",
			EventType:   string(saga.EventSagaStarted),
			Timestamp:   now,
			Headers: map[string]string{
				HeaderSagaID:    "saga-1",
				HeaderEventID:   "event-1",
				HeaderEventType: string(saga.EventSagaStarted),
				HeaderTimestamp: now.Format(time.RFC3339Nano),
			},
		},
		{
			Version:     ProtocolVersion1_1,
			MessageType: MessageTypeSagaEvent,
			SagaID:      "saga-2",
			EventID:     "event-2",
			EventType:   string(saga.EventSagaCompleted),
			Timestamp:   now,
			Headers: map[string]string{
				HeaderSagaID:    "saga-2",
				HeaderEventID:   "event-2",
				HeaderEventType: string(saga.EventSagaCompleted),
				HeaderTimestamp: now.Format(time.RFC3339Nano),
			},
		},
	}

	events, err := batch.FromProtocolMessages(protocols)
	require.NoError(t, err)
	assert.Len(t, events, 2)

	assert.Equal(t, "event-1", events[0].ID)
	assert.Equal(t, "event-2", events[1].ID)
}

func TestBatchMessageConverter_EmptyBatch(t *testing.T) {
	converter, err := NewStandardMessageConverter(nil)
	require.NoError(t, err)

	batch := NewBatchMessageConverter(converter)

	// Empty events slice
	messages, err := batch.ToMessagingMessages([]*saga.SagaEvent{})
	require.NoError(t, err)
	assert.Empty(t, messages)

	// Empty messages slice
	events, err := batch.FromMessagingMessages([]*messaging.Message{})
	require.NoError(t, err)
	assert.Empty(t, events)
}
