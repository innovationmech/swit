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
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// default implementation delegates to segmentio/kafka-go if available.
// To avoid hard dependency in this step, we provide a thin adapter with a build tag in future.

// wireDefaultWriterFactory sets a no-op placeholder. Real factory will be set in init() when kafka-go is available.
func wireDefaultWriterFactory() {
	if writerFactory == nil {
		writerFactory = func(brokerCfg *messaging.BrokerConfig, topic string, pubCfg *messaging.PublisherConfig) (kafkaWriter, error) {
			// minimal in-memory fake to keep tests compiling until kafka-go integration is added
			return &fakeWriter{}, nil
		}
	}
}

type fakeWriter struct{}

func (f *fakeWriter) WriteMessages(ctx context.Context, msgs ...KafkaMessage) error { return nil }
func (f *fakeWriter) Close() error                                                  { return nil }

func init() {
	wireDefaultWriterFactory()
}

// helper to translate framework config batching into writer settings will be added alongside kafka-go integration.
func _unused_silence() time.Duration { return time.Duration(0) }
