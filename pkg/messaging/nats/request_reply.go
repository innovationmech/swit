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
	"context"
	"errors"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/nats-io/nats.go"
)

// RequestReplyClient provides helpers for request-reply over NATS core.
// It converts nats.go errors into framework messaging errors and respects context timeouts.
type RequestReplyClient struct {
	conn    *nats.Conn
	timeout time.Duration
}

// NewRequestReplyClient creates a new RequestReplyClient with default timeout from config.
func NewRequestReplyClient(conn *nats.Conn, cfg *Config) *RequestReplyClient {
	to := 5 * time.Second
	if cfg != nil {
		to = cfg.Timeouts.RequestTimeout()
	}
	return &RequestReplyClient{conn: conn, timeout: to}
}

// Request sends a request to subject using context for deadline/cancel and returns the reply payload.
func (c *RequestReplyClient) Request(ctx context.Context, subject string, data []byte) ([]byte, error) {
	if c == nil || c.conn == nil {
		return nil, messaging.NewConnectionError("nats client not connected", nil)
	}

	// Ensure we have a deadline; if not, apply default per config
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	msg, err := c.conn.RequestWithContext(ctx, subject, data)
	if err != nil {
		// Map common timeout/cancel errors
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, nats.ErrTimeout) {
			return nil, messaging.NewProcessingTimeoutError("nats request timed out", err)
		}
		if errors.Is(err, context.Canceled) {
			return nil, messaging.NewProcessingError("nats request canceled", err)
		}
		return nil, messaging.NewProcessingError("nats request failed", err)
	}
	return msg.Data, nil
}

// RequestMessage sends a messaging.Message and returns a messaging.Message reply.
// Caller is responsible for serialization of message.Payload.
func (c *RequestReplyClient) RequestMessage(ctx context.Context, request *messaging.Message) (*messaging.Message, error) {
	if request == nil {
		return nil, messaging.NewProcessingError("request message cannot be nil", nil)
	}
	data, err := c.Request(ctx, request.Topic, request.Payload)
	if err != nil {
		return nil, err
	}
	reply := &messaging.Message{
		Topic:         request.ReplyTo,
		Payload:       data,
		CorrelationID: request.CorrelationID,
		Timestamp:     time.Now(),
	}
	return reply, nil
}

// RequestReplyServer provides reply handling helpers.
type RequestReplyServer struct {
	conn *nats.Conn
}

// NewRequestReplyServer creates a new RequestReplyServer.
func NewRequestReplyServer(conn *nats.Conn) *RequestReplyServer {
	return &RequestReplyServer{conn: conn}
}

// Handle subscribes subject and invokes handler to produce a response payload per request.
func (s *RequestReplyServer) Handle(subject string, handler func([]byte) ([]byte, error)) (*nats.Subscription, error) {
	if s == nil || s.conn == nil {
		return nil, messaging.NewConnectionError("nats server not connected", nil)
	}
	if subject == "" {
		return nil, messaging.NewConfigError("subject is required", nil)
	}
	if handler == nil {
		return nil, messaging.NewConfigError("handler is required", nil)
	}
	sub, err := s.conn.Subscribe(subject, func(msg *nats.Msg) {
		if msg == nil {
			return
		}
		resp, hErr := handler(msg.Data)
		if hErr != nil {
			// For now, respond with empty to indicate error; users can encode error details
			_ = msg.Respond(nil)
			return
		}
		_ = msg.Respond(resp)
	})
	if err != nil {
		return nil, messaging.NewProcessingError("nats reply subscription failed", err)
	}
	return sub, nil
}
