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

// validateKafkaConfiguration performs Kafka-specific semantic validation.
// It augments adapter validation with endpoint format checks, auth constraints,
// and producer settings compatibility.
func validateKafkaConfiguration(base *messaging.BrokerConfig, kcfg *Config) (
	[]messaging.AdapterValidationError,
	[]messaging.AdapterValidationWarning,
	[]messaging.AdapterValidationSuggestion,
) {
	var errs []messaging.AdapterValidationError
	var warns []messaging.AdapterValidationWarning
	var suggs []messaging.AdapterValidationSuggestion

	if base == nil {
		errs = append(errs, messaging.AdapterValidationError{
			Field:    "",
			Message:  "configuration cannot be nil",
			Code:     "KAFKA_CONFIG_NIL",
			Severity: messaging.AdapterValidationSeverityError,
		})
		return errs, warns, suggs
	}

	// Endpoints must be host:port for kafka-go's kafka.TCP(...)
	for i, ep := range base.Endpoints {
		epTrim := strings.TrimSpace(ep)
		if strings.Contains(epTrim, "://") {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Endpoints[" + intToString(i) + "]",
				Message:  "endpoint must be in host:port form (no URL scheme)",
				Code:     "KAFKA_ENDPOINT_FORMAT_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
	}

	// Auth constraints: Kafka adapter supports only SASL (or none)
	if base.Authentication != nil {
		switch base.Authentication.Type {
		case messaging.AuthTypeNone:
			// ok
		case messaging.AuthTypeSASL:
			mech := strings.TrimSpace(base.Authentication.Mechanism)
			if mech == "" || strings.EqualFold(mech, "AUTO") {
				// default to PLAIN, but require credentials
				if base.Authentication.Username == "" || base.Authentication.Password == "" {
					errs = append(errs, messaging.AdapterValidationError{
						Field:    "Authentication",
						Message:  "SASL requires username and password",
						Code:     "KAFKA_SASL_CREDENTIALS_REQUIRED",
						Severity: messaging.AdapterValidationSeverityError,
					})
				}
			} else {
				// Supported: PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
				switch strings.ToUpper(mech) {
				case "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512":
					if base.Authentication.Username == "" || base.Authentication.Password == "" {
						errs = append(errs, messaging.AdapterValidationError{
							Field:    "Authentication",
							Message:  "SASL requires username and password",
							Code:     "KAFKA_SASL_CREDENTIALS_REQUIRED",
							Severity: messaging.AdapterValidationSeverityError,
						})
					}
				default:
					errs = append(errs, messaging.AdapterValidationError{
						Field:    "Authentication.Mechanism",
						Message:  "unsupported SASL mechanism (allowed: PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)",
						Code:     "KAFKA_SASL_MECHANISM_UNSUPPORTED",
						Severity: messaging.AdapterValidationSeverityError,
					})
				}
			}
		default:
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Authentication.Type",
				Message:  "unsupported auth type for Kafka (allowed: none, sasl)",
				Code:     "KAFKA_AUTH_UNSUPPORTED",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
	}

	// Producer semantics
	if kcfg != nil {
		// If idempotent requested, require acks=all for stronger guarantees
		if kcfg.Producer.Idempotent && !strings.EqualFold(kcfg.Producer.Acks, "all") {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.kafka.producer",
				Message:  "idempotent=true requires producer.acks=all",
				Code:     "KAFKA_IDEMPOTENCE_ACKS_INCOMPATIBLE",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
		// Warn when acks=none due to potential data loss
		if strings.EqualFold(kcfg.Producer.Acks, "none") {
			warns = append(warns, messaging.AdapterValidationWarning{
				Field:   "Extra.kafka.producer.acks",
				Message: "acks=none may lead to message loss; consider 'leader' or 'all'",
				Code:    "KAFKA_ACKS_NONE_RISKS_LOSS",
			})
		}
	}

	return errs, warns, suggs
}

func intToString(i int) string {
	// minimal allocation helper without importing strconv everywhere in this file
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	n := i
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = digits[n%10]
		n /= 10
	}
	return string(buf[pos:])
}
