package rabbitmq

import (
	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/streadway/amqp"
)

// MapAMQPError maps streadway/amqp errors to unified messaging errors.
// It returns an error suitable for classification and retry logic.
func MapAMQPError(op string, err error) error {
	if err == nil {
		return nil
	}

	// amqp.Error carries code and reason
	if aerr, ok := err.(*amqp.Error); ok {
		// Classify by code group per AMQP semantics
		switch aerr.Code {
		// Channel/connection closed/errors → connection lost (retryable)
		case 320, 541:
			return messaging.NewRabbitMQChannelError(0, aerr.Reason, err)

		// Access refused, not allowed → auth/permission (non-retry until fixed)
		case 403, 530:
			return messaging.NewRabbitMQError(messaging.ErrorTypeAuthentication, messaging.ErrCodePermissionDenied).
				Message("RabbitMQ access denied").
				Cause(err).
				Retryable(false).
				Build()

		// Resource locked/limit → resource class with backoff
		case 405, 406, 311: // not allowed/precondition failed/content too large
			return messaging.NewRabbitMQError(messaging.ErrorTypeResource, messaging.ErrCodeResourceExhausted).
				Message("RabbitMQ resource limit or precondition failed").
				Cause(err).
				Retryable(true).
				Build()
		}
	}

	// Generic fallback with broker context
	return messaging.WrapWithBrokerContext(messaging.BrokerTypeRabbitMQ, err, op)
}
