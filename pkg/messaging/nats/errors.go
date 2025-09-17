package nats

import (
	"github.com/innovationmech/swit/pkg/messaging"
	n "github.com/nats-io/nats.go"
)

// MapNATSError maps NATS client errors into unified messaging errors.
func MapNATSError(op string, err error) error {
	if err == nil {
		return nil
	}

	switch err {
	case n.ErrTimeout:
		return messaging.NewNATSError(messaging.ErrorTypeConnection, messaging.ErrCodeConnectionTimeout).
			Message("NATS operation timeout").
			Cause(err).
			Retryable(true).
			Build()
	case n.ErrAuthorization, n.ErrNoResponders, n.ErrNoServers:
		// Authorization → non-retryable until fixed; no servers → retryable
		retryable := err == n.ErrNoServers
		code := messaging.ErrCodePermissionDenied
		et := messaging.ErrorTypeAuthentication
		if retryable {
			et = messaging.ErrorTypeConnection
			code = messaging.ErrCodeConnectionFailed
		}
		return messaging.NewNATSError(et, code).
			Message("NATS error").
			Cause(err).
			Retryable(retryable).
			Build()
	}

	return messaging.WrapWithBrokerContext(messaging.BrokerTypeNATS, err, op)
}
