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

package messaging

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	tagValidator *validator.Validate
)

// GetTagValidator returns a singleton go-playground validator with custom rules registered.
func GetTagValidator() *validator.Validate {
	if tagValidator != nil {
		return tagValidator
	}
	v := validator.New(validator.WithRequiredStructEnabled())

	// Register custom validators
	_ = v.RegisterValidation("notblank", validateNotBlank)
	_ = v.RegisterValidation("endpoint", validateEndpoint)

	tagValidator = v
	return tagValidator
}

// validateNotBlank checks that a string is not just whitespace.
func validateNotBlank(fl validator.FieldLevel) bool {
	if fl.Field().Kind().String() != "string" {
		return true
	}
	return strings.TrimSpace(fl.Field().String()) != ""
}

// validateEndpoint validates broker endpoint strings.
// Accepted forms:
// - host:port (e.g., "localhost:9092")
// - scheme://host[:port] (e.g., "nats://localhost:4222")
func validateEndpoint(fl validator.FieldLevel) bool {
	if fl.Field().Kind().String() != "string" {
		return true
	}
	value := strings.TrimSpace(fl.Field().String())
	if value == "" {
		return false
	}
	if strings.Contains(value, "://") {
		if u, err := url.Parse(value); err == nil && u.Host != "" {
			return true
		}
		return false
	}
	// If no scheme and no colon, treat as host-only and allow (e.g., for inmemory/default cases)
	if !strings.Contains(value, ":") {
		return true
	}
	// Otherwise require host:port
	host, port, err := net.SplitHostPort(value)
	if err != nil || host == "" || port == "" {
		return false
	}
	return true
}

// AggregateValidationError converts validator.ValidationErrors into a single MessagingError
// with a user-friendly aggregated message.
func AggregateValidationError(err error) error {
	if err == nil {
		return nil
	}
	if ve, ok := err.(validator.ValidationErrors); ok {
		var parts []string
		for _, fe := range ve {
			fieldPath := buildFieldPath(fe.StructNamespace())
			msg := renderFieldError(fe, fieldPath)
			parts = append(parts, msg)
		}
		return NewConfigValidationError(strings.Join(parts, "; "), err)
	}
	return err
}

func renderFieldError(fe validator.FieldError, fieldPath string) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s: is required", fieldPath)
	case "min":
		return fmt.Sprintf("%s: must be at least %s", fieldPath, fe.Param())
	case "max":
		return fmt.Sprintf("%s: must be at most %s", fieldPath, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s: must be one of [%s]", fieldPath, fe.Param())
	case "notblank":
		return fmt.Sprintf("%s: must not be blank", fieldPath)
	case "endpoint":
		return fmt.Sprintf("%s: must be a valid endpoint (host:port or scheme://host[:port])", fieldPath)
	case "dive":
		// dive is internal; should not surface
		return fmt.Sprintf("%s: invalid value", fieldPath)
	default:
		// Generic fallback
		if fe.Param() != "" {
			return fmt.Sprintf("%s: validation '%s' failed (param=%s)", fieldPath, fe.Tag(), fe.Param())
		}
		return fmt.Sprintf("%s: validation '%s' failed", fieldPath, fe.Tag())
	}
}

// buildFieldPath converts StructNamespace (e.g., BrokerConfig.Endpoints.0)
// to a user-friendly path (e.g., broker_config.endpoints[0]).
func buildFieldPath(ns string) string {
	if ns == "" {
		return "field"
	}
	// Split by '.' and transform each segment
	segments := strings.Split(ns, ".")
	var out []string
	for _, s := range segments {
		// index segment is numeric
		if isAllDigits(s) {
			// append as [i] to the previous segment if exists
			if len(out) == 0 {
				out = append(out, "["+s+"]")
			} else {
				out[len(out)-1] = out[len(out)-1] + "[" + s + "]"
			}
			continue
		}
		out = append(out, toSnakeLower(s))
	}
	return strings.Join(out, ".")
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

func toSnakeLower(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
