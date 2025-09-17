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
