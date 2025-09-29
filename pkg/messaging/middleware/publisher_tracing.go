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


