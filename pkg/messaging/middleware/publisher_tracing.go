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

package middleware

import (
	"context"

	"github.com/innovationmech/swit/pkg/messaging"
)

// PublisherTracingMiddleware injects propagation headers on publish path and
// creates PRODUCER spans when a Tracer is provided.
type PublisherTracingMiddleware struct {
	tracer Tracer
}

func NewPublisherTracingMiddleware(tracer Tracer) *PublisherTracingMiddleware {
	if tracer == nil {
		tracer = NewNoOpTracer()
	}
	return &PublisherTracingMiddleware{tracer: tracer}
}

func (m *PublisherTracingMiddleware) Name() string { return "publisher-tracing" }

func (m *PublisherTracingMiddleware) WrapPublish(next messaging.PublishFunc) messaging.PublishFunc {
	return func(ctx context.Context, message *messaging.Message) error {
		// Start PRODUCER span, inject headers then call next
		spanName := "publish_message"
		ctx2, span := m.tracer.StartSpan(ctx, spanName, WithSpanKind(SpanKindProducer))
		if message != nil {
			if message.Headers == nil {
				message.Headers = make(map[string]string)
			}
			m.tracer.Inject(ctx2, message.Headers)
		}
		err := next(ctx2, message)
		if err != nil {
			span.SetStatus(SpanStatusError, err.Error())
		} else {
			span.SetStatus(SpanStatusOK, "")
		}
		span.End()
		return err
	}
}

func (m *PublisherTracingMiddleware) WrapPublishBatch(next messaging.PublishBatchFunc) messaging.PublishBatchFunc {
	return func(ctx context.Context, messages []*messaging.Message) error {
		spanName := "publish_messages_batch"
		ctx2, span := m.tracer.StartSpan(ctx, spanName, WithSpanKind(SpanKindProducer))
		for _, msg := range messages {
			if msg == nil {
				continue
			}
			if msg.Headers == nil {
				msg.Headers = make(map[string]string)
			}
			m.tracer.Inject(ctx2, msg.Headers)
		}
		err := next(ctx2, messages)
		if err != nil {
			span.SetStatus(SpanStatusError, err.Error())
		} else {
			span.SetStatus(SpanStatusOK, "")
		}
		span.End()
		return err
	}
}
