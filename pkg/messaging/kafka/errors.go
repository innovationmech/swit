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
	"strings"

	"github.com/innovationmech/swit/pkg/messaging"
)

// MapKafkaError maps kafka-go errors to unified messaging errors.
func MapKafkaError(op string, err error) error {
	if err == nil {
		return nil
	}

	// Avoid importing kafka-go: use heuristics based on error strings to keep deps minimal.
	s := err.Error()
	if containsAnyLower(s, []string{"timeout", "timed out"}) {
		return messaging.NewKafkaError(messaging.ErrorTypeConnection, messaging.ErrCodeConnectionTimeout).
			Message("Kafka timeout").
			Cause(err).
			Retryable(true).
			Build()
	}
	if containsAnyLower(s, []string{"temporary", "temporarily", "connection reset", "connection refused"}) {
		return messaging.NewKafkaError(messaging.ErrorTypeConnection, messaging.ErrCodeConnectionFailed).
			Message("Kafka temporary error").
			Cause(err).
			Retryable(true).
			Build()
	}
	// Default
	return messaging.WrapWithBrokerContext(messaging.BrokerTypeKafka, err, op)
}

func containsAnyLower(haystack string, needles []string) bool {
	if haystack == "" || len(needles) == 0 {
		return false
	}
	hl := strings.ToLower(haystack)
	for _, n := range needles {
		if n == "" {
			continue
		}
		if strings.Contains(hl, strings.ToLower(n)) {
			return true
		}
	}
	return false
}
