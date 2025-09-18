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

package benchmark

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/stretchr/testify/require"
)

type stubFactory struct {
	broker messaging.MessageBroker
}

func (f *stubFactory) CreateBroker(_ *messaging.BrokerConfig) (messaging.MessageBroker, error) {
	return f.broker, nil
}

func (f *stubFactory) GetSupportedBrokerTypes() []messaging.BrokerType {
	return []messaging.BrokerType{messaging.BrokerTypeInMemory}
}

func (f *stubFactory) ValidateConfig(_ *messaging.BrokerConfig) error { return nil }

type fakeBroker struct {
	latency       time.Duration
	failuresLeft  int32
	connected     bool
	failOnConnect bool
}

func (b *fakeBroker) Connect(_ context.Context) error {
	if b.failOnConnect {
		return errors.New("connect failure")
	}
	b.connected = true
	return nil
}

func (b *fakeBroker) Disconnect(_ context.Context) error {
	b.connected = false
	return nil
}

func (b *fakeBroker) Close() error { b.connected = false; return nil }

func (b *fakeBroker) IsConnected() bool { return b.connected }

func (b *fakeBroker) CreatePublisher(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
	if config.Topic == "" {
		return nil, errors.New("topic required")
	}
	return &fakePublisher{broker: b}, nil
}

func (b *fakeBroker) CreateSubscriber(messaging.SubscriberConfig) (messaging.EventSubscriber, error) {
	return nil, errors.New("not implemented")
}

func (b *fakeBroker) HealthCheck(context.Context) (*messaging.HealthStatus, error) {
	return &messaging.HealthStatus{Status: messaging.HealthStatusHealthy}, nil
}

func (b *fakeBroker) GetMetrics() *messaging.BrokerMetrics { return &messaging.BrokerMetrics{} }

func (b *fakeBroker) GetCapabilities() *messaging.BrokerCapabilities {
	return &messaging.BrokerCapabilities{}
}

type fakePublisher struct {
	broker *fakeBroker
	calls  int32
}

func (p *fakePublisher) Publish(ctx context.Context, _ *messaging.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	time.Sleep(p.broker.latency)
	if atomic.LoadInt32(&p.broker.failuresLeft) > 0 {
		atomic.AddInt32(&p.broker.failuresLeft, -1)
		return errors.New("injected failure")
	}
	atomic.AddInt32(&p.calls, 1)
	return nil
}

func (p *fakePublisher) PublishBatch(ctx context.Context, batch []*messaging.Message) error {
	for range batch {
		if err := p.Publish(ctx, &messaging.Message{}); err != nil {
			return err
		}
	}
	return nil
}

func (p *fakePublisher) PublishWithConfirm(ctx context.Context, m *messaging.Message) (*messaging.PublishConfirmation, error) {
	if err := p.Publish(ctx, m); err != nil {
		return nil, err
	}
	return &messaging.PublishConfirmation{}, nil
}

func (p *fakePublisher) PublishAsync(ctx context.Context, m *messaging.Message, callback messaging.PublishCallback) error {
	go func() {
		err := p.Publish(ctx, m)
		callback(&messaging.PublishConfirmation{}, err)
	}()
	return nil
}

func (p *fakePublisher) BeginTransaction(context.Context) (messaging.Transaction, error) {
	return nil, errors.New("not supported")
}

func (p *fakePublisher) Flush(context.Context) error { return nil }

func (p *fakePublisher) Close() error { return nil }

func (p *fakePublisher) GetMetrics() *messaging.PublisherMetrics {
	return &messaging.PublisherMetrics{}
}

func TestSuiteRunSuccess(t *testing.T) {
	broker := &fakeBroker{latency: 2 * time.Millisecond}
	factory := &stubFactory{broker: broker}

	suite := Suite{
		Factory: factory,
		Targets: []Target{
			{
				Name: "fake",
				Config: &messaging.BrokerConfig{
					Type:      messaging.BrokerTypeInMemory,
					Endpoints: []string{"localhost"},
				},
			},
		},
		Workloads: []Workload{{
			Name:        "single",
			Messages:    20,
			MessageSize: 128,
			Publishers:  2,
			BatchSize:   1,
		}},
	}

	report, err := suite.Run(context.Background())
	require.NoError(t, err)
	require.Len(t, report.Results, 1)

	result := report.Results[0]
	require.Equal(t, "single", result.WorkloadName)
	require.Equal(t, 20, result.Metrics.TotalMessages)
	require.Equal(t, 20, result.Metrics.Successful)
	require.Equal(t, 0, result.Metrics.Failed)
	require.Greater(t, result.Metrics.Throughput, 0.0)
	require.True(t, result.Metrics.AverageLatency >= broker.latency)
	require.Zero(t, result.Metrics.ErrorRate())
}

func TestSuiteRunRecordsFailures(t *testing.T) {
	broker := &fakeBroker{latency: time.Millisecond, failuresLeft: 5}
	factory := &stubFactory{broker: broker}

	suite := Suite{
		Factory: factory,
		Targets: []Target{
			{
				Name: "faulty",
				Config: &messaging.BrokerConfig{
					Type:      messaging.BrokerTypeInMemory,
					Endpoints: []string{"localhost"},
				},
			},
		},
		Workloads: []Workload{{
			Name:        "failure",
			Messages:    10,
			MessageSize: 256,
			Publishers:  1,
			BatchSize:   1,
		}},
	}

	report, err := suite.Run(context.Background())
	require.NoError(t, err)
	require.Len(t, report.Results, 1)

	result := report.Results[0]
	require.Equal(t, 10, result.Metrics.TotalMessages)
	require.Equal(t, 5, result.Metrics.Successful)
	require.Equal(t, 5, result.Metrics.Failed)
	require.NotEmpty(t, result.Metrics.ErrorSummaries)
	require.Greater(t, result.Metrics.ErrorRate(), 0.0)
}

func TestFailureOnConnect(t *testing.T) {
	broker := &fakeBroker{failOnConnect: true}
	factory := &stubFactory{broker: broker}

	suite := Suite{
		Factory: factory,
		Targets: []Target{
			{
				Name: "unreachable",
				Config: &messaging.BrokerConfig{
					Type:      messaging.BrokerTypeInMemory,
					Endpoints: []string{"localhost"},
				},
			},
		},
		Workloads: []Workload{{Name: "noop", Messages: 1, MessageSize: 16, Publishers: 1}},
	}

	report, err := suite.Run(context.Background())
	require.NoError(t, err)
	require.Len(t, report.Results, 1)
	require.Error(t, report.Results[0].Error)
}
