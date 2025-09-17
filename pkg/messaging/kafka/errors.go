package kafka

import (
	"github.com/innovationmech/swit/pkg/messaging"
	"strings"
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
