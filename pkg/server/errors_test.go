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

package server

import (
	"errors"
	"testing"
)

func TestServerError(t *testing.T) {
	t.Run("NewServerError", func(t *testing.T) {
		err := NewServerError(ErrCodeConfigInvalid, "test message")
		if err.Code != ErrCodeConfigInvalid {
			t.Errorf("Expected code %s, got %s", ErrCodeConfigInvalid, err.Code)
		}
		if err.Message != "test message" {
			t.Errorf("Expected message 'test message', got '%s'", err.Message)
		}
	})

	t.Run("NewServerErrorWithCause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewServerErrorWithCause(ErrCodeTransportStart, "transport failed", cause)
		if err.Cause != cause {
			t.Errorf("Expected cause to be set")
		}
		if err.Unwrap() != cause {
			t.Errorf("Expected Unwrap to return cause")
		}
	})

	t.Run("NewServerErrorWithDetails", func(t *testing.T) {
		err := NewServerErrorWithDetails(ErrCodeConfigValidation, "validation failed", "field 'port' is invalid")
		if err.Details != "field 'port' is invalid" {
			t.Errorf("Expected details to be set")
		}
	})

	t.Run("WrapError", func(t *testing.T) {
		original := errors.New("original error")
		err := WrapError(ErrCodeStartup, "startup failed", original)
		if err.Cause != original {
			t.Errorf("Expected cause to be original error")
		}
		if err.Details != "original error" {
			t.Errorf("Expected details to be original error message")
		}
	})

	t.Run("WrapError with nil", func(t *testing.T) {
		err := WrapError(ErrCodeStartup, "startup failed", nil)
		if err != nil {
			t.Errorf("Expected nil when wrapping nil error")
		}
	})
}

func TestServerErrorMethods(t *testing.T) {
	t.Run("Error method", func(t *testing.T) {
		err := NewServerError(ErrCodeConfigInvalid, "test message")
		expected := "CONFIG_INVALID: test message"
		if err.Error() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("Error method with details", func(t *testing.T) {
		err := NewServerErrorWithDetails(ErrCodeConfigInvalid, "test message", "more details")
		expected := "CONFIG_INVALID: test message - more details"
		if err.Error() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("WithContext", func(t *testing.T) {
		err := NewServerError(ErrCodeConfigInvalid, "test message")
		err.WithContext("service", "test-service")
		if err.Context["service"] != "test-service" {
			t.Errorf("Expected context to be set")
		}
	})

	t.Run("WithOperation", func(t *testing.T) {
		err := NewServerError(ErrCodeConfigInvalid, "test message")
		err.WithOperation("startup")
		if err.Operation != "startup" {
			t.Errorf("Expected operation to be set")
		}
	})
}

func TestErrorHelpers(t *testing.T) {
	t.Run("IsServerError", func(t *testing.T) {
		serverErr := NewServerError(ErrCodeConfigInvalid, "test")
		regularErr := errors.New("regular error")

		if !IsServerError(serverErr) {
			t.Errorf("Expected IsServerError to return true for ServerError")
		}
		if IsServerError(regularErr) {
			t.Errorf("Expected IsServerError to return false for regular error")
		}
	})

	t.Run("GetServerError", func(t *testing.T) {
		serverErr := NewServerError(ErrCodeConfigInvalid, "test")
		regularErr := errors.New("regular error")

		if extracted, ok := GetServerError(serverErr); !ok || extracted != serverErr {
			t.Errorf("Expected GetServerError to extract ServerError")
		}
		if _, ok := GetServerError(regularErr); ok {
			t.Errorf("Expected GetServerError to return false for regular error")
		}
	})

	t.Run("HasErrorCode", func(t *testing.T) {
		serverErr := NewServerError(ErrCodeConfigInvalid, "test")
		regularErr := errors.New("regular error")

		if !HasErrorCode(serverErr, ErrCodeConfigInvalid) {
			t.Errorf("Expected HasErrorCode to return true for matching code")
		}
		if HasErrorCode(serverErr, ErrCodeTransportStart) {
			t.Errorf("Expected HasErrorCode to return false for non-matching code")
		}
		if HasErrorCode(regularErr, ErrCodeConfigInvalid) {
			t.Errorf("Expected HasErrorCode to return false for regular error")
		}
	})
}

func TestPredefinedErrors(t *testing.T) {
	t.Run("Configuration errors", func(t *testing.T) {
		err := ErrConfigInvalid("port must be positive")
		if err.Code != ErrCodeConfigInvalid {
			t.Errorf("Expected code %s", ErrCodeConfigInvalid)
		}

		err = ErrConfigMissing("database_url")
		if err.Code != ErrCodeConfigMissing {
			t.Errorf("Expected code %s", ErrCodeConfigMissing)
		}

		original := errors.New("validation failed")
		err = ErrConfigValidation(original)
		if err.Code != ErrCodeConfigValidation || err.Cause != original {
			t.Errorf("Expected wrapped validation error")
		}
	})

	t.Run("Transport errors", func(t *testing.T) {
		original := errors.New("bind failed")
		err := ErrTransportStart("http", original)
		if err.Code != ErrCodeTransportStart {
			t.Errorf("Expected code %s", ErrCodeTransportStart)
		}

		err = ErrTransportBind(":8080", original)
		if err.Code != ErrCodeTransportBind {
			t.Errorf("Expected code %s", ErrCodeTransportBind)
		}
	})

	t.Run("Discovery errors", func(t *testing.T) {
		original := errors.New("connection refused")
		err := ErrDiscoveryConnect("localhost:8500", original)
		if err.Code != ErrCodeDiscoveryConnect {
			t.Errorf("Expected code %s", ErrCodeDiscoveryConnect)
		}

		err = ErrDiscoveryRegister("test-service", original)
		if err.Code != ErrCodeDiscoveryRegister {
			t.Errorf("Expected code %s", ErrCodeDiscoveryRegister)
		}
	})

	t.Run("Lifecycle errors", func(t *testing.T) {
		original := errors.New("startup failed")
		err := ErrStartup("database", original)
		if err.Code != ErrCodeStartup {
			t.Errorf("Expected code %s", ErrCodeStartup)
		}

		err = ErrTimeout("database_connect", "30s")
		if err.Code != ErrCodeTimeout {
			t.Errorf("Expected code %s", ErrCodeTimeout)
		}
	})

	t.Run("Service errors", func(t *testing.T) {
		original := errors.New("init failed")
		err := ErrServiceInit("user-service", original)
		if err.Code != ErrCodeServiceInit {
			t.Errorf("Expected code %s", ErrCodeServiceInit)
		}
	})

	t.Run("Factory errors", func(t *testing.T) {
		original := errors.New("creation failed")
		err := ErrFactoryCreate("http", original)
		if err.Code != ErrCodeFactoryCreate {
			t.Errorf("Expected code %s", ErrCodeFactoryCreate)
		}

		err = ErrFactoryNotFound("unknown-factory")
		if err.Code != ErrCodeFactoryNotFound {
			t.Errorf("Expected code %s", ErrCodeFactoryNotFound)
		}
	})
}
