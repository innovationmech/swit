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

package rabbitmq

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/innovationmech/swit/pkg/messaging"
)

var validExchangeTypes = map[string]struct{}{
	"direct":  {},
	"fanout":  {},
	"topic":   {},
	"headers": {},
}

// validateTopology performs semantic validation for RabbitMQ topology.
// It returns adapter-style validation issues that can be merged into the
// adapter validation result.
func validateTopology(t TopologyConfig) (
	errs []messaging.AdapterValidationError,
	warns []messaging.AdapterValidationWarning,
	suggs []messaging.AdapterValidationSuggestion,
) {
	// Normalize and collect declared entities for reference checks
	exchanges := make(map[string]ExchangeConfig, len(t.Exchanges))
	for key, ex := range t.Exchanges {
		name := strings.TrimSpace(ex.Name)
		if name == "" {
			name = strings.TrimSpace(key)
		}

		if name == "" {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    fmt.Sprintf("Topology.Exchanges[%s].name", key),
				Message:  "exchange name cannot be empty",
				Code:     "RABBITMQ_TOPOLOGY_EXCHANGE_NAME_EMPTY",
				Severity: messaging.AdapterValidationSeverityError,
			})
			continue
		}

		if _, ok := validExchangeTypes[strings.ToLower(strings.TrimSpace(ex.Type))]; !ok {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    fmt.Sprintf("Topology.Exchanges[%s].type", name),
				Message:  fmt.Sprintf("invalid exchange type: %s (allowed: direct, fanout, topic, headers)", ex.Type),
				Code:     "RABBITMQ_TOPOLOGY_EXCHANGE_TYPE_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}

		if !ex.Durable {
			suggs = append(suggs, messaging.AdapterValidationSuggestion{
				Field:          fmt.Sprintf("Topology.Exchanges[%s].durable", name),
				Message:        "consider setting durable=true for production",
				SuggestedValue: true,
			})
		}

		exchanges[name] = ex
	}

	queues := make(map[string]QueueConfig, len(t.Queues))
	for key, q := range t.Queues {
		name := strings.TrimSpace(q.Name)
		if name == "" {
			name = strings.TrimSpace(key)
		}
		if name == "" {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    fmt.Sprintf("Topology.Queues[%s].name", key),
				Message:  "queue name cannot be empty",
				Code:     "RABBITMQ_TOPOLOGY_QUEUE_NAME_EMPTY",
				Severity: messaging.AdapterValidationSeverityError,
			})
			continue
		}

		if !q.Durable {
			suggs = append(suggs, messaging.AdapterValidationSuggestion{
				Field:          fmt.Sprintf("Topology.Queues[%s].durable", name),
				Message:        "consider setting durable=true for production",
				SuggestedValue: true,
			})
		}

		if ttl, ok := extractInt64(q.Arguments, "x-message-ttl"); ok {
			if ttl <= 0 {
				errs = append(errs, messaging.AdapterValidationError{
					Field:    fmt.Sprintf("Topology.Queues[%s].arguments.x-message-ttl", name),
					Message:  "x-message-ttl must be positive",
					Code:     "RABBITMQ_TOPOLOGY_INVALID_TTL",
					Severity: messaging.AdapterValidationSeverityError,
				})
			}
		}

		queues[name] = q
	}

	for i, b := range t.Bindings {
		if strings.TrimSpace(b.Exchange) == "" {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    fmt.Sprintf("Topology.Bindings[%d].exchange", i),
				Message:  "binding exchange cannot be empty",
				Code:     "RABBITMQ_TOPOLOGY_BINDING_EXCHANGE_EMPTY",
				Severity: messaging.AdapterValidationSeverityError,
			})
		} else if _, ok := exchanges[b.Exchange]; !ok {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    fmt.Sprintf("Topology.Bindings[%d].exchange", i),
				Message:  fmt.Sprintf("binding references unknown exchange: %s", b.Exchange),
				Code:     "RABBITMQ_TOPOLOGY_UNKNOWN_EXCHANGE",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}

		if strings.TrimSpace(b.Queue) == "" {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    fmt.Sprintf("Topology.Bindings[%d].queue", i),
				Message:  "binding queue cannot be empty",
				Code:     "RABBITMQ_TOPOLOGY_BINDING_QUEUE_EMPTY",
				Severity: messaging.AdapterValidationSeverityError,
			})
		} else if _, ok := queues[b.Queue]; !ok {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    fmt.Sprintf("Topology.Bindings[%d].queue", i),
				Message:  fmt.Sprintf("binding references unknown queue: %s", b.Queue),
				Code:     "RABBITMQ_TOPOLOGY_UNKNOWN_QUEUE",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
	}

	return errs, warns, suggs
}

// validateRabbitConfiguration performs additional adapter-level validation:
// - Enforce amqp/amqps URL scheme for endpoints
// - Warn if heartbeat is too low for production
func validateRabbitConfiguration(base *messaging.BrokerConfig, cfg *Config) (
	[]messaging.AdapterValidationError,
	[]messaging.AdapterValidationWarning,
	[]messaging.AdapterValidationSuggestion,
) {
	var errs []messaging.AdapterValidationError
	var warns []messaging.AdapterValidationWarning
	var suggs []messaging.AdapterValidationSuggestion

	// Endpoint scheme validation: require amqp:// or amqps://
	for i, ep := range base.Endpoints {
		s := strings.TrimSpace(ep)
		if !strings.HasPrefix(s, "amqp://") && !strings.HasPrefix(s, "amqps://") {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    fmt.Sprintf("Endpoints[%d]", i),
				Message:  "endpoint must start with amqp:// or amqps://",
				Code:     "RABBITMQ_ENDPOINT_SCHEME_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
			continue
		}
		if u, err := url.Parse(s); err != nil || u.Host == "" {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    fmt.Sprintf("Endpoints[%d]", i),
				Message:  "invalid AMQP endpoint URL",
				Code:     "RABBITMQ_ENDPOINT_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
	}

	if cfg != nil {
		// Heartbeat warn when < 10s
		if cfg.Timeouts.Heartbeat > 0 && cfg.Timeouts.Heartbeat < messaging.Duration(10_000_000_000) { // 10s
			warns = append(warns, messaging.AdapterValidationWarning{
				Field:   "Extra.rabbitmq.timeouts.heartbeat",
				Message: "heartbeat is set very low; consider >= 10s to avoid churn",
				Code:    "RABBITMQ_HEARTBEAT_TOO_LOW",
			})
		}
	}

	return errs, warns, suggs
}

func extractInt64(m map[string]interface{}, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint:
		return int64(n), true
	case uint8:
		return int64(n), true
	case uint16:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		if n > ^uint64(0)>>1 {
			return int64(^uint64(0) >> 1), true
		}
		return int64(n), true
	case float32:
		return int64(n), true
	case float64:
		return int64(n), true
	case string:
		// Best-effort: attempt to parse numeric strings
		// Keep it simple: ignore parse errors and treat as not found
		return 0, false
	default:
		return 0, false
	}
}
