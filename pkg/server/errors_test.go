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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ServerError
		expected string
	}{
		{
			name: "basic error",
			err: &ServerError{
				Code:    ErrCodeConfigInvalid,
				Message: "invalid configuration",
			},
			expected: "[CONFIG_INVALID] invalid configuration",
		},
		{
			name: "error with details",
			err: &ServerError{
				Code:    ErrCodeTransportFailed,
				Message: "transport initialization failed",
				Details: "port already in use",
			},
			expected: "[TRANSPORT_FAILED] transport initialization failed: port already in use ()",
		},
		{
			name: "error with operation",
			err: &ServerError{
				Code:      ErrCodeServiceRegFailed,
				Message:   "service registration failed",
				Operation: "RegisterHTTPHandler",
			},
			expected: "[SERVICE_REGISTRATION_FAILED] service registration failed (RegisterHTTPHandler)",
		},
		{
			name: "error with details and operation",
			err: &ServerError{
				Code:      ErrCodeDiscoveryFailed,
				Message:   "discovery registration failed",
				Details:   "consul unavailable",
				Operation: "RegisterWithDiscovery",
			},
			expected: "[DISCOVERY_FAILED] discovery registration failed (consul unavailable) (RegisterWithDiscovery)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestServerError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &ServerError{
		Code:    ErrCodeConfigInvalid,
		Message: "config error",
		Cause:   cause,
	}

	assert.Equal(t, cause, err.Unwrap())
}

func TestServerError_WithContext(t *testing.T) {
	err := &ServerError{
		Code:    ErrCodeTransportFailed,
		Message: "transport failed",
	}

	result := err.WithContext("port", "8080").WithContext("protocol", "http")

	assert.Equal(t, "8080", result.Context["port"])
	assert.Equal(t, "http", result.Context["protocol"])
	assert.Same(t, err, result) // Should return the same instance
}

func TestServerError_WithOperation(t *testing.T) {
	err := &ServerError{
		Code:    ErrCodeServiceRegFailed,
		Message: "registration failed",
	}

	result := err.WithOperation("StartServer")

	assert.Equal(t, "StartServer", result.Operation)
	assert.Same(t, err, result) // Should return the same instance
}

func TestNewServerError(t *testing.T) {
	err := NewServerError(ErrCodeConfigInvalid, "invalid config", CategoryConfig)

	assert.Equal(t, ErrCodeConfigInvalid, err.Code)
	assert.Equal(t, "invalid config", err.Message)
	assert.Equal(t, CategoryConfig, err.Category)
	assert.NotNil(t, err.Context)
	assert.Empty(t, err.Context)
}

func TestNewServerErrorWithDetails(t *testing.T) {
	err := NewServerErrorWithDetails(ErrCodeTransportFailed, "transport failed", "port in use", CategoryTransport)

	assert.Equal(t, ErrCodeTransportFailed, err.Code)
	assert.Equal(t, "transport failed", err.Message)
	assert.Equal(t, "port in use", err.Details)
	assert.Equal(t, CategoryTransport, err.Category)
}

func TestNewServerErrorWithCause(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewServerErrorWithCause(ErrCodeDependencyFailed, "dependency failed", CategoryDependency, cause)

	assert.Equal(t, ErrCodeDependencyFailed, err.Code)
	assert.Equal(t, "dependency failed", err.Message)
	assert.Equal(t, CategoryDependency, err.Category)
	assert.Equal(t, cause, err.Cause)
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := WrapError(originalErr, ErrCodeLifecycleFailed, "lifecycle failed", CategoryLifecycle)

	assert.Equal(t, ErrCodeLifecycleFailed, wrappedErr.Code)
	assert.Equal(t, "lifecycle failed", wrappedErr.Message)
	assert.Equal(t, CategoryLifecycle, wrappedErr.Category)
	assert.Equal(t, originalErr, wrappedErr.Cause)
}

func TestConfigurationErrorFactories(t *testing.T) {
	t.Run("ErrConfigInvalid", func(t *testing.T) {
		err := ErrConfigInvalid("invalid port")
		assert.Equal(t, ErrCodeConfigInvalid, err.Code)
		assert.Equal(t, "invalid port", err.Message)
		assert.Equal(t, CategoryConfig, err.Category)
	})

	t.Run("ErrConfigInvalidWithCause", func(t *testing.T) {
		cause := errors.New("parse error")
		err := ErrConfigInvalidWithCause("config parse failed", cause)
		assert.Equal(t, ErrCodeConfigInvalid, err.Code)
		assert.Equal(t, "config parse failed", err.Message)
		assert.Equal(t, CategoryConfig, err.Category)
		assert.Equal(t, cause, err.Cause)
	})
}

func TestTransportErrorFactories(t *testing.T) {
	t.Run("ErrTransportFailed", func(t *testing.T) {
		err := ErrTransportFailed("HTTP transport failed")
		assert.Equal(t, ErrCodeTransportFailed, err.Code)
		assert.Equal(t, "HTTP transport failed", err.Message)
		assert.Equal(t, CategoryTransport, err.Category)
	})

	t.Run("ErrTransportFailedWithCause", func(t *testing.T) {
		cause := errors.New("bind error")
		err := ErrTransportFailedWithCause("failed to bind port", cause)
		assert.Equal(t, ErrCodeTransportFailed, err.Code)
		assert.Equal(t, "failed to bind port", err.Message)
		assert.Equal(t, CategoryTransport, err.Category)
		assert.Equal(t, cause, err.Cause)
	})
}

func TestServiceRegistrationErrorFactories(t *testing.T) {
	t.Run("ErrServiceRegFailed", func(t *testing.T) {
		err := ErrServiceRegFailed("handler registration failed")
		assert.Equal(t, ErrCodeServiceRegFailed, err.Code)
		assert.Equal(t, "handler registration failed", err.Message)
		assert.Equal(t, CategoryService, err.Category)
	})

	t.Run("ErrServiceRegFailedWithCause", func(t *testing.T) {
		cause := errors.New("route conflict")
		err := ErrServiceRegFailedWithCause("route registration failed", cause)
		assert.Equal(t, ErrCodeServiceRegFailed, err.Code)
		assert.Equal(t, "route registration failed", err.Message)
		assert.Equal(t, CategoryService, err.Category)
		assert.Equal(t, cause, err.Cause)
	})
}

func TestDiscoveryErrorFactories(t *testing.T) {
	t.Run("ErrDiscoveryFailed", func(t *testing.T) {
		err := ErrDiscoveryFailed("consul registration failed")
		assert.Equal(t, ErrCodeDiscoveryFailed, err.Code)
		assert.Equal(t, "consul registration failed", err.Message)
		assert.Equal(t, CategoryDiscovery, err.Category)
	})

	t.Run("ErrDiscoveryFailedWithCause", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := ErrDiscoveryFailedWithCause("consul unavailable", cause)
		assert.Equal(t, ErrCodeDiscoveryFailed, err.Code)
		assert.Equal(t, "consul unavailable", err.Message)
		assert.Equal(t, CategoryDiscovery, err.Category)
		assert.Equal(t, cause, err.Cause)
	})
}

func TestDependencyErrorFactories(t *testing.T) {
	t.Run("ErrDependencyFailed", func(t *testing.T) {
		err := ErrDependencyFailed("database connection failed")
		assert.Equal(t, ErrCodeDependencyFailed, err.Code)
		assert.Equal(t, "database connection failed", err.Message)
		assert.Equal(t, CategoryDependency, err.Category)
	})

	t.Run("ErrDependencyFailedWithCause", func(t *testing.T) {
		cause := errors.New("connection timeout")
		err := ErrDependencyFailedWithCause("DB initialization failed", cause)
		assert.Equal(t, ErrCodeDependencyFailed, err.Code)
		assert.Equal(t, "DB initialization failed", err.Message)
		assert.Equal(t, CategoryDependency, err.Category)
		assert.Equal(t, cause, err.Cause)
	})
}

func TestLifecycleErrorFactories(t *testing.T) {
	t.Run("ErrLifecycleFailed", func(t *testing.T) {
		err := ErrLifecycleFailed("startup sequence failed")
		assert.Equal(t, ErrCodeLifecycleFailed, err.Code)
		assert.Equal(t, "startup sequence failed", err.Message)
		assert.Equal(t, CategoryLifecycle, err.Category)
	})

	t.Run("ErrLifecycleFailedWithCause", func(t *testing.T) {
		cause := errors.New("resource allocation failed")
		err := ErrLifecycleFailedWithCause("initialization failed", cause)
		assert.Equal(t, ErrCodeLifecycleFailed, err.Code)
		assert.Equal(t, "initialization failed", err.Message)
		assert.Equal(t, CategoryLifecycle, err.Category)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("ErrServerAlreadyStarted", func(t *testing.T) {
		err := ErrServerAlreadyStarted()
		assert.Equal(t, ErrCodeServerAlreadyStarted, err.Code)
		assert.Equal(t, "server is already started", err.Message)
		assert.Equal(t, CategoryLifecycle, err.Category)
	})

	t.Run("ErrServerNotStarted", func(t *testing.T) {
		err := ErrServerNotStarted()
		assert.Equal(t, ErrCodeServerNotStarted, err.Code)
		assert.Equal(t, "server is not started", err.Message)
		assert.Equal(t, CategoryLifecycle, err.Category)
	})
}

func TestMiddlewareErrorFactories(t *testing.T) {
	t.Run("ErrMiddlewareFailed", func(t *testing.T) {
		err := ErrMiddlewareFailed("CORS middleware failed")
		assert.Equal(t, ErrCodeMiddlewareFailed, err.Code)
		assert.Equal(t, "CORS middleware failed", err.Message)
		assert.Equal(t, CategoryMiddleware, err.Category)
	})

	t.Run("ErrMiddlewareFailedWithCause", func(t *testing.T) {
		cause := errors.New("invalid configuration")
		err := ErrMiddlewareFailedWithCause("auth middleware setup failed", cause)
		assert.Equal(t, ErrCodeMiddlewareFailed, err.Code)
		assert.Equal(t, "auth middleware setup failed", err.Message)
		assert.Equal(t, CategoryMiddleware, err.Category)
		assert.Equal(t, cause, err.Cause)
	})
}

func TestShutdownErrorFactories(t *testing.T) {
	t.Run("ErrShutdownTimeout", func(t *testing.T) {
		err := ErrShutdownTimeout("shutdown timeout exceeded")
		assert.Equal(t, ErrCodeShutdownTimeout, err.Code)
		assert.Equal(t, "shutdown timeout exceeded", err.Message)
		assert.Equal(t, CategoryLifecycle, err.Category)
	})

	t.Run("ErrShutdownTimeoutWithCause", func(t *testing.T) {
		cause := errors.New("transport shutdown failed")
		err := ErrShutdownTimeoutWithCause("graceful shutdown failed", cause)
		assert.Equal(t, ErrCodeShutdownTimeout, err.Code)
		assert.Equal(t, "graceful shutdown failed", err.Message)
		assert.Equal(t, CategoryLifecycle, err.Category)
		assert.Equal(t, cause, err.Cause)
	})
}

func TestErrorUtilityFunctions(t *testing.T) {
	t.Run("IsServerError", func(t *testing.T) {
		serverErr := ErrConfigInvalid("test error")
		regularErr := errors.New("regular error")

		assert.True(t, IsServerError(serverErr))
		assert.False(t, IsServerError(regularErr))
		assert.False(t, IsServerError(nil))
	})

	t.Run("GetServerError", func(t *testing.T) {
		serverErr := ErrTransportFailed("test error")
		regularErr := errors.New("regular error")

		result := GetServerError(serverErr)
		assert.NotNil(t, result)
		assert.Equal(t, ErrCodeTransportFailed, result.Code)

		result = GetServerError(regularErr)
		assert.Nil(t, result)

		result = GetServerError(nil)
		assert.Nil(t, result)
	})

	t.Run("IsErrorCategory", func(t *testing.T) {
		configErr := ErrConfigInvalid("config error")
		transportErr := ErrTransportFailed("transport error")
		regularErr := errors.New("regular error")

		assert.True(t, IsErrorCategory(configErr, CategoryConfig))
		assert.False(t, IsErrorCategory(configErr, CategoryTransport))
		assert.True(t, IsErrorCategory(transportErr, CategoryTransport))
		assert.False(t, IsErrorCategory(regularErr, CategoryConfig))
	})

	t.Run("IsErrorCode", func(t *testing.T) {
		configErr := ErrConfigInvalid("config error")
		transportErr := ErrTransportFailed("transport error")
		regularErr := errors.New("regular error")

		assert.True(t, IsErrorCode(configErr, ErrCodeConfigInvalid))
		assert.False(t, IsErrorCode(configErr, ErrCodeTransportFailed))
		assert.True(t, IsErrorCode(transportErr, ErrCodeTransportFailed))
		assert.False(t, IsErrorCode(regularErr, ErrCodeConfigInvalid))
	})

	t.Run("GetErrorContext", func(t *testing.T) {
		err := ErrServiceRegFailed("registration failed").WithContext("service", "auth").WithContext("port", "8080")
		regularErr := errors.New("regular error")

		context := GetErrorContext(err)
		require.NotNil(t, context)
		assert.Equal(t, "auth", context["service"])
		assert.Equal(t, "8080", context["port"])

		context = GetErrorContext(regularErr)
		assert.Nil(t, context)
	})

	t.Run("GetErrorCategory", func(t *testing.T) {
		configErr := ErrConfigInvalid("config error")
		regularErr := errors.New("regular error")

		category := GetErrorCategory(configErr)
		assert.Equal(t, CategoryConfig, category)

		category = GetErrorCategory(regularErr)
		assert.Equal(t, ErrorCategory(""), category)
	})
}

func TestErrorWrapping(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := ErrTransportFailedWithCause("transport failed", originalErr)

	// Test that errors.Is works
	assert.True(t, errors.Is(wrappedErr, originalErr))

	// Test that errors.As works
	var serverErr *ServerError
	assert.True(t, errors.As(wrappedErr, &serverErr))
	assert.Equal(t, ErrCodeTransportFailed, serverErr.Code)

	// Test unwrapping
	assert.Equal(t, originalErr, errors.Unwrap(wrappedErr))
}

func TestErrorChaining(t *testing.T) {
	// Create a chain of errors
	rootErr := errors.New("root cause")
	middlewareErr := ErrMiddlewareFailedWithCause("middleware setup failed", rootErr)
	serverErr := ErrLifecycleFailedWithCause("server startup failed", middlewareErr)

	// Test that we can find the root cause
	assert.True(t, errors.Is(serverErr, rootErr))
	assert.True(t, errors.Is(serverErr, middlewareErr))

	// Test that we can extract specific error types
	var extractedServerErr *ServerError
	assert.True(t, errors.As(serverErr, &extractedServerErr))
	assert.Equal(t, ErrCodeLifecycleFailed, extractedServerErr.Code)

	// Test unwrapping chain
	unwrapped := errors.Unwrap(serverErr)
	assert.Equal(t, middlewareErr, unwrapped)

	unwrapped = errors.Unwrap(unwrapped)
	assert.Equal(t, rootErr, unwrapped)
}
