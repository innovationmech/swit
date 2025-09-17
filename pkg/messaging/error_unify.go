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
	"errors"
)

// ToBaseMessagingError converts any supported error into *BaseMessagingError for unified processing.
//
// Behavior:
// - If err is already *BaseMessagingError, it is returned as-is
// - If err is *BrokerSpecificError, its embedded BaseMessagingError is returned
// - If err is legacy *MessagingError, it is converted to *BaseMessagingError using a best-effort mapping
// - Otherwise returns nil (caller can fall back to generic handling)
func ToBaseMessagingError(err error) *BaseMessagingError {
	if err == nil {
		return nil
	}

	var be *BaseMessagingError
	if errors.As(err, &be) {
		return be
	}

	var bse *BrokerSpecificError
	if errors.As(err, &bse) {
		return bse.BaseMessagingError
	}

	var le *MessagingError
	if errors.As(err, &le) {
		return convertLegacyToBase(le)
	}

	return nil
}

// convertLegacyToBase maps legacy MessagingError to the new BaseMessagingError taxonomy.
func convertLegacyToBase(le *MessagingError) *BaseMessagingError {
	if le == nil {
		return nil
	}

	// Default mapping
	mappedType := ErrorTypeInternal
	mappedCode := ErrCodeInternal

	switch le.Code {
	// Connection
	case ErrConnectionFailed:
		mappedType, mappedCode = ErrorTypeConnection, ErrCodeConnectionFailed
	case ErrConnectionTimeout:
		mappedType, mappedCode = ErrorTypeConnection, ErrCodeConnectionTimeout
	case ErrConnectionLost:
		mappedType, mappedCode = ErrorTypeConnection, ErrCodeConnectionLost
	case ErrDisconnectionFailed:
		mappedType, mappedCode = ErrorTypeConnection, ErrCodeDisconnectionFailed

	// Auth
	case ErrAuthenticationFailed:
		mappedType, mappedCode = ErrorTypeAuthentication, ErrCodeAuthFailed
	case ErrAuthorizationDenied:
		mappedType, mappedCode = ErrorTypeAuthentication, ErrCodePermissionDenied
	case ErrInvalidCredentials:
		mappedType, mappedCode = ErrorTypeAuthentication, ErrCodeInvalidCredentials

	// Config
	case ErrInvalidConfig:
		mappedType, mappedCode = ErrorTypeConfiguration, ErrCodeInvalidConfig
	case ErrUnsupportedBroker:
		mappedType, mappedCode = ErrorTypeConfiguration, ErrCodeUnsupportedBroker
	case ErrConfigValidation:
		mappedType, mappedCode = ErrorTypeConfiguration, ErrCodeConfigValidation

	// Publish
	case ErrPublishFailed:
		mappedType, mappedCode = ErrorTypePublishing, ErrCodePublishFailed
	case ErrPublishTimeout:
		mappedType, mappedCode = ErrorTypePublishing, ErrCodePublishTimeout
	case ErrMessageTooLarge:
		mappedType, mappedCode = ErrorTypePublishing, ErrCodeMessageTooLarge
	case ErrTopicNotFound:
		mappedType, mappedCode = ErrorTypePublishing, ErrCodeTopicNotFound
	case ErrBatchPublishFailed:
		mappedType, mappedCode = ErrorTypePublishing, ErrCodeBatchPublishFailed
	case ErrPublisherClosed:
		mappedType, mappedCode = ErrorTypePublishing, ErrCodePublisherClosed

	// Subscription
	case ErrSubscriptionFailed:
		mappedType, mappedCode = ErrorTypeSubscription, ErrCodeSubscriptionFailed
	case ErrConsumerGroupError:
		mappedType, mappedCode = ErrorTypeSubscription, ErrCodeConsumerGroupError
	case ErrRebalancing:
		mappedType, mappedCode = ErrorTypeSubscription, ErrCodeRebalancing
	case ErrSubscriberClosed:
		mappedType, mappedCode = ErrorTypeSubscription, ErrCodeSubscriberClosed
	case ErrUnsubscribeFailed:
		mappedType, mappedCode = ErrorTypeSubscription, ErrCodeUnsubscribeFailed

	// Processing
	case ErrProcessingFailed:
		mappedType, mappedCode = ErrorTypeProcessing, ErrCodeProcessingFailed
	case ErrProcessingTimeout:
		mappedType, mappedCode = ErrorTypeProcessing, ErrCodeProcessingTimeout
	case ErrInvalidMessage:
		mappedType, mappedCode = ErrorTypeProcessing, ErrCodeInvalidMessage
	case ErrMessageDeserialization:
		mappedType, mappedCode = ErrorTypeSerialization, ErrCodeDeserializationFailed
	case ErrMiddlewareFailure:
		mappedType, mappedCode = ErrorTypeProcessing, ErrCodeMiddlewareFailure

	// Transaction
	case ErrTransactionFailed:
		mappedType, mappedCode = ErrorTypeTransaction, ErrCodeTransactionFailed
	case ErrTransactionAborted:
		mappedType, mappedCode = ErrorTypeTransaction, ErrCodeTransactionAborted
	case ErrTransactionTimeout:
		mappedType, mappedCode = ErrorTypeTransaction, ErrCodeTransactionTimeout
	case ErrTransactionNotSupported:
		mappedType, mappedCode = ErrorTypeTransaction, ErrCodeTransactionNotSupported

	// Health
	case ErrHealthCheckFailed:
		mappedType, mappedCode = ErrorTypeHealth, ErrCodeHealthCheckFailed
	case ErrHealthCheckTimeout:
		mappedType, mappedCode = ErrorTypeHealth, ErrCodeHealthCheckTimeout

	// Seek/position
	case ErrSeekFailed:
		mappedType, mappedCode = ErrorTypeSubscription, ErrCodeSeekFailed
	case ErrSeekNotSupported:
		mappedType, mappedCode = ErrorTypeSubscription, ErrCodeSeekNotSupported
	case ErrInvalidPosition:
		mappedType, mappedCode = ErrorTypeSubscription, ErrCodeInvalidPosition

	// Resource
	case ErrResourceExhausted:
		mappedType, mappedCode = ErrorTypeResource, ErrCodeResourceExhausted
	case ErrRateLimitExceeded:
		mappedType, mappedCode = ErrorTypeResource, ErrCodeRateLimitExceeded
	case ErrQueueFull:
		mappedType, mappedCode = ErrorTypeResource, ErrCodeQueueFull

	// Internal
	case ErrInternal:
		mappedType, mappedCode = ErrorTypeInternal, ErrCodeInternal
	case ErrNotImplemented:
		mappedType, mappedCode = ErrorTypeInternal, ErrCodeNotImplemented
	case ErrOperationAborted:
		mappedType, mappedCode = ErrorTypeInternal, ErrCodeOperationAborted
	}

	return &BaseMessagingError{
		Type:      mappedType,
		Code:      mappedCode,
		Message:   le.Message,
		Severity:  SeverityMedium,
		Retryable: le.Retryable,
		Context:   ErrorContext{},
		Cause:     le.Cause,
		Timestamp: le.Timestamp,
	}
}

// NormalizeError ensures an error is represented as *BaseMessagingError; if not convertible,
// it wraps it into a generic internal error with provided context.
func NormalizeError(err error, errType MessagingErrorType, code string, message string) *BaseMessagingError {
	if err == nil {
		return nil
	}
	if be := ToBaseMessagingError(err); be != nil {
		return be
	}
	// Fallback wrapper
	return NewError(errType, code).
		Message(message).
		Cause(err).
		Severity(SeverityMedium).
		Retryable(true).
		Build()
}

// WrapAsBrokerError wraps any error into a BrokerSpecificError for the given broker type and operation.
func WrapAsBrokerError(brokerType BrokerType, operation string, err error) error {
	if err == nil {
		return nil
	}
	// Reuse existing helper if available
	return WrapWithBrokerContext(brokerType, err, operation)
}
