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
