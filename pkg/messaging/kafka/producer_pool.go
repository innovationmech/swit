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

package kafka

import (
    "context"
    "errors"
    "sync"
    "time"

    "github.com/innovationmech/swit/pkg/messaging"
)

// kafkaWriter abstracts kafka-go Writer for testability.
type kafkaWriter interface {
    WriteMessages(ctx context.Context, msgs ...KafkaMessage) error
    Close() error
}

// KafkaMessage is a minimal message abstraction to decouple tests from kafka-go API.
// In production, this will be converted to github.com/segmentio/kafka-go.Message.
type KafkaMessage struct {
    Topic     string
    Key       []byte
    Value     []byte
    Headers   map[string]string
    Timestamp time.Time
}

// writerFactory creates a writer for a given topic using broker/publisher config.
var writerFactory func(brokerCfg *messaging.BrokerConfig, topic string, pubCfg *messaging.PublisherConfig) (kafkaWriter, error)

// publishOp represents a queued publish request processed by workers.
type publishOp struct {
    ctx        context.Context
    topic      string
    batch      []KafkaMessage
    single     *KafkaMessage
    callback   messaging.PublishCallback
    resultChan chan error
    confirmCh  chan *messaging.PublishConfirmation
}

// topicProducer manages a queue and workers for a specific topic.
type topicProducer struct {
    writer     kafkaWriter
    queue      chan publishOp
    workers    int
    wg         sync.WaitGroup
    closed     bool
    closeOnce  sync.Once
    closeMutex sync.RWMutex
}

func newTopicProducer(writer kafkaWriter, queueSize, workers int) *topicProducer {
    tp := &topicProducer{
        writer:  writer,
        queue:   make(chan publishOp, queueSize),
        workers: workers,
    }
    for i := 0; i < workers; i++ {
        tp.wg.Add(1)
        go tp.workerLoop()
    }
    return tp
}

func (tp *topicProducer) workerLoop() {
    defer tp.wg.Done()
    for op := range tp.queue {
        var err error
        if op.single != nil {
            err = tp.writer.WriteMessages(op.ctx, *op.single)
        } else if len(op.batch) > 0 {
            err = tp.writer.WriteMessages(op.ctx, op.batch...)
        }

        if op.resultChan != nil {
            op.resultChan <- err
            close(op.resultChan)
        }

        if op.callback != nil || op.confirmCh != nil {
            if err != nil {
                if op.callback != nil {
                    op.callback(nil, messaging.NewPublishError("async publish failed", err))
                }
                if op.confirmCh != nil {
                    close(op.confirmCh)
                }
            } else {
                // Build a basic confirmation (partition/offset unknown here)
                conf := &messaging.PublishConfirmation{
                    MessageID: "",
                    Partition: -1,
                    Offset:    -1,
                    Timestamp: time.Now(),
                    Metadata:  map[string]string{"topic": op.topic},
                }
                if op.callback != nil {
                    op.callback(conf, nil)
                }
                if op.confirmCh != nil {
                    op.confirmCh <- conf
                    close(op.confirmCh)
                }
            }
        }
    }
}

func (tp *topicProducer) enqueue(ctx context.Context, op publishOp) error {
    tp.closeMutex.RLock()
    closed := tp.closed
    tp.closeMutex.RUnlock()
    if closed {
        return messaging.ErrPublisherAlreadyClosed
    }

    select {
    case tp.queue <- op:
        return nil
    case <-ctx.Done():
        return messaging.NewPublishError("producer queue backpressure: enqueue timeout", ctx.Err())
    }
}

func (tp *topicProducer) close() error {
    var err error
    tp.closeOnce.Do(func() {
        tp.closeMutex.Lock()
        tp.closed = true
        tp.closeMutex.Unlock()
        close(tp.queue)
        tp.wg.Wait()
        if tp.writer != nil {
            err = tp.writer.Close()
        }
    })
    return err
}

// producerPool manages topic-level producers sharing configuration.
type producerPool struct {
    brokerCfg *messaging.BrokerConfig
    mu        sync.RWMutex
    topics    map[string]*topicProducer
    closed    bool
}

func newProducerPool(brokerCfg *messaging.BrokerConfig, pubCfg *messaging.PublisherConfig) (*producerPool, error) {
    if writerFactory == nil {
        return nil, errors.New("kafka writer factory is not initialized")
    }
    return &producerPool{
        brokerCfg: brokerCfg,
        topics:    make(map[string]*topicProducer),
    }, nil
}

func (p *producerPool) getOrCreateTopicProducer(topic string, pubCfg *messaging.PublisherConfig) (*topicProducer, error) {
    p.mu.RLock()
    if tp, ok := p.topics[topic]; ok {
        p.mu.RUnlock()
        return tp, nil
    }
    p.mu.RUnlock()

    p.mu.Lock()
    defer p.mu.Unlock()
    if tp, ok := p.topics[topic]; ok {
        return tp, nil
    }

    w, err := writerFactory(p.brokerCfg, topic, pubCfg)
    if err != nil {
        return nil, err
    }
    queueSize := pubCfg.QueueSize
    if queueSize <= 0 {
        queueSize = 10000
    }
    workers := pubCfg.Workers
    if workers <= 0 {
        workers = 1
    }
    tp := newTopicProducer(w, queueSize, workers)
    p.topics[topic] = tp
    return tp, nil
}

func (p *producerPool) PublishSync(ctx context.Context, topic string, msg KafkaMessage, pubCfg *messaging.PublisherConfig) error {
    if p.closed {
        return messaging.ErrPublisherAlreadyClosed
    }
    tp, err := p.getOrCreateTopicProducer(topic, pubCfg)
    if err != nil {
        return err
    }
    res := make(chan error, 1)
    op := publishOp{ctx: ctx, topic: topic, single: &msg, resultChan: res}
    if err := tp.enqueue(ctx, op); err != nil {
        return err
    }
    return <-res
}

func (p *producerPool) PublishBatchSync(ctx context.Context, topic string, msgs []KafkaMessage, pubCfg *messaging.PublisherConfig) error {
    if p.closed {
        return messaging.ErrPublisherAlreadyClosed
    }
    tp, err := p.getOrCreateTopicProducer(topic, pubCfg)
    if err != nil {
        return err
    }
    res := make(chan error, 1)
    op := publishOp{ctx: ctx, topic: topic, batch: msgs, resultChan: res}
    if err := tp.enqueue(ctx, op); err != nil {
        return err
    }
    return <-res
}

func (p *producerPool) PublishAsync(ctx context.Context, topic string, msg KafkaMessage, pubCfg *messaging.PublisherConfig, cb messaging.PublishCallback) error {
    if p.closed {
        return messaging.ErrPublisherAlreadyClosed
    }
    tp, err := p.getOrCreateTopicProducer(topic, pubCfg)
    if err != nil {
        return err
    }
    op := publishOp{ctx: ctx, topic: topic, single: &msg, callback: cb}
    return tp.enqueue(ctx, op)
}

func (p *producerPool) PublishWithConfirm(ctx context.Context, topic string, msg KafkaMessage, pubCfg *messaging.PublisherConfig) (*messaging.PublishConfirmation, error) {
    if p.closed {
        return nil, messaging.ErrPublisherAlreadyClosed
    }
    tp, err := p.getOrCreateTopicProducer(topic, pubCfg)
    if err != nil {
        return nil, err
    }
    confCh := make(chan *messaging.PublishConfirmation, 1)
    res := make(chan error, 1)
    op := publishOp{ctx: ctx, topic: topic, single: &msg, resultChan: res, confirmCh: confCh}
    if err := tp.enqueue(ctx, op); err != nil {
        return nil, err
    }
    if err := <-res; err != nil {
        return nil, err
    }
    conf, ok := <-confCh
    if !ok {
        return nil, messaging.NewPublishError("failed to get confirmation", nil)
    }
    return conf, nil
}

func (p *producerPool) Close() error {
    p.mu.Lock()
    defer p.mu.Unlock()
    if p.closed {
        return nil
    }
    p.closed = true
    for topic, tp := range p.topics {
        _ = tp.close()
        delete(p.topics, topic)
    }
    return nil
}


