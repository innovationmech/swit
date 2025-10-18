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

import "errors"

// Handler configuration errors
var (
	// ErrInvalidHandlerID indicates the handler ID is empty or invalid.
	ErrInvalidHandlerID = errors.New("invalid handler ID: must not be empty")

	// ErrInvalidHandlerName indicates the handler name is empty or invalid.
	ErrInvalidHandlerName = errors.New("invalid handler name: must not be empty")

	// ErrNoTopicsConfigured indicates no topics are configured for the handler.
	ErrNoTopicsConfigured = errors.New("no topics configured: at least one topic is required")

	// ErrInvalidConcurrency indicates the concurrency setting is invalid.
	ErrInvalidConcurrency = errors.New("invalid concurrency: must be greater than 0")

	// ErrInvalidBatchSize indicates the batch size setting is invalid.
	ErrInvalidBatchSize = errors.New("invalid batch size: must be greater than 0")

	// ErrInvalidTimeout indicates the processing timeout is invalid.
	ErrInvalidTimeout = errors.New("invalid timeout: must be greater than 0")
)

// Handler registration errors
var (
	// ErrHandlerAlreadyRegistered indicates a handler with the same ID is already registered.
	ErrHandlerAlreadyRegistered = errors.New("handler already registered with this ID")

	// ErrHandlerNotFound indicates the specified handler was not found.
	ErrHandlerNotFound = errors.New("handler not found")

	// ErrNoHandlersAvailable indicates no handlers are available for processing.
	ErrNoHandlersAvailable = errors.New("no handlers available for event processing")

	// ErrHandlerNotInitialized indicates the handler has not been initialized.
	ErrHandlerNotInitialized = errors.New("handler not initialized")

	// ErrHandlerAlreadyInitialized indicates the handler has already been initialized.
	ErrHandlerAlreadyInitialized = errors.New("handler already initialized")

	// ErrHandlerShutdown indicates the handler has been shut down.
	ErrHandlerShutdown = errors.New("handler has been shut down")
)

// Event processing errors
var (
	// ErrInvalidEvent indicates the event is nil or invalid.
	ErrInvalidEvent = errors.New("invalid event: event must not be nil")

	// ErrInvalidEventType indicates the event type is not supported.
	ErrInvalidEventType = errors.New("invalid event type: not supported by this handler")

	// ErrEventProcessingTimeout indicates event processing exceeded the timeout.
	ErrEventProcessingTimeout = errors.New("event processing timeout")

	// ErrEventDeserializationFailed indicates the event could not be deserialized.
	ErrEventDeserializationFailed = errors.New("event deserialization failed")

	// ErrEventValidationFailed indicates the event failed validation.
	ErrEventValidationFailed = errors.New("event validation failed")

	// ErrEventFilteredOut indicates the event was filtered out by filter rules.
	ErrEventFilteredOut = errors.New("event filtered out by filter rules")
)

// Dead-letter queue errors
var (
	// ErrDeadLetterQueueDisabled indicates the dead-letter queue is not enabled.
	ErrDeadLetterQueueDisabled = errors.New("dead-letter queue is disabled")

	// ErrDeadLetterQueueFull indicates the dead-letter queue is full.
	ErrDeadLetterQueueFull = errors.New("dead-letter queue is full")

	// ErrDeadLetterPublishFailed indicates publishing to the dead-letter queue failed.
	ErrDeadLetterPublishFailed = errors.New("failed to publish to dead-letter queue")
)

// Routing errors
var (
	// ErrInvalidRoutingStrategy indicates the routing strategy is not supported.
	ErrInvalidRoutingStrategy = errors.New("invalid routing strategy")

	// ErrNoMatchingHandlers indicates no handlers match the event criteria.
	ErrNoMatchingHandlers = errors.New("no matching handlers for event")

	// ErrRoutingFailed indicates event routing failed.
	ErrRoutingFailed = errors.New("event routing failed")
)

// Context errors
var (
	// ErrInvalidContext indicates the context is nil or invalid.
	ErrInvalidContext = errors.New("invalid context: must not be nil")

	// ErrContextCancelled indicates the context was cancelled.
	ErrContextCancelled = errors.New("context cancelled")

	// ErrContextDeadlineExceeded indicates the context deadline was exceeded.
	ErrContextDeadlineExceeded = errors.New("context deadline exceeded")
)
