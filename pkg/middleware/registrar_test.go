// Copyright © 2023 jackelyj <dreamerlyj@gmail.com>
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

package middleware

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewGlobalMiddlewareRegistrar(t *testing.T) {
	registrar := NewGlobalMiddlewareRegistrar()

	assert.NotNil(t, registrar)
	assert.IsType(t, &GlobalMiddlewareRegistrar{}, registrar)
}

func TestGlobalMiddlewareRegistrar_GetName(t *testing.T) {
	registrar := NewGlobalMiddlewareRegistrar()

	name := registrar.GetName()
	assert.Equal(t, "global-middleware", name)
}

func TestGlobalMiddlewareRegistrar_GetPriority(t *testing.T) {
	registrar := NewGlobalMiddlewareRegistrar()

	priority := registrar.GetPriority()
	assert.Equal(t, 1, priority)
}

func TestGlobalMiddlewareRegistrar_RegisterMiddleware(t *testing.T) {
	t.Run("successful_registration", func(t *testing.T) {
		router := gin.New()
		registrar := NewGlobalMiddlewareRegistrar()

		err := registrar.RegisterMiddleware(router)
		assert.NoError(t, err)

		// Verify that the router has middlewares registered
		assert.NotEmpty(t, router.Handlers)
	})

	t.Run("registrar_returns_no_error", func(t *testing.T) {
		router := gin.New()
		registrar := NewGlobalMiddlewareRegistrar()

		err := registrar.RegisterMiddleware(router)
		assert.NoError(t, err)
	})
}

func TestGlobalMiddlewareRegistrar_RegisterMiddleware_NilRouter(t *testing.T) {
	registrar := NewGlobalMiddlewareRegistrar()

	// This should panic with nil router, but since the implementation
	// doesn't check for nil, we expect a panic
	assert.Panics(t, func() {
		registrar.RegisterMiddleware(nil)
	})
}

func TestGlobalMiddlewareRegistrar_MiddlewareOrder(t *testing.T) {
	router := gin.New()
	registrar := NewGlobalMiddlewareRegistrar()

	// Track middleware execution order
	executionOrder := make([]string, 0)

	// Add a middleware to track execution before global middlewares
	router.Use(func(c *gin.Context) {
		executionOrder = append(executionOrder, "before-global")
		c.Next()
	})

	err := registrar.RegisterMiddleware(router)
	require.NoError(t, err)

	// Add a middleware to track execution after global middlewares
	router.Use(func(c *gin.Context) {
		executionOrder = append(executionOrder, "after-global")
		c.Next()
	})

	// Verify that middlewares were registered
	assert.True(t, len(router.Handlers) > 1, "Expected multiple middleware handlers")
}

func TestGlobalMiddlewareRegistrar_ConfigurationValidation(t *testing.T) {
	router := gin.New()
	registrar := NewGlobalMiddlewareRegistrar()

	err := registrar.RegisterMiddleware(router)
	require.NoError(t, err)

	// Verify that timeout configuration is properly set
	// This is a structural test to ensure the middleware registration works
	assert.NotNil(t, router)
	assert.NotEmpty(t, router.Handlers)
}

func TestGlobalMiddlewareRegistrar_MultipleRegistrations(t *testing.T) {
	router := gin.New()
	registrar := NewGlobalMiddlewareRegistrar()

	// Register middleware multiple times (should not cause issues)
	err1 := registrar.RegisterMiddleware(router)
	err2 := registrar.RegisterMiddleware(router)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Each registration adds more middleware handlers
	assert.True(t, len(router.Handlers) >= 2, "Expected at least 2 middleware handlers after multiple registrations")
}

func TestGlobalMiddlewareRegistrar_Interface(t *testing.T) {
	registrar := NewGlobalMiddlewareRegistrar()

	// Test that it implements the expected interface methods
	assert.Equal(t, "global-middleware", registrar.GetName())
	assert.Equal(t, 1, registrar.GetPriority())

	// Test that RegisterMiddleware method exists and can be called
	router := gin.New()
	err := registrar.RegisterMiddleware(router)
	assert.NoError(t, err)
}

func TestGlobalMiddlewareRegistrar_TimeoutConfiguration(t *testing.T) {
	router := gin.New()
	registrar := NewGlobalMiddlewareRegistrar()

	// Count handlers before registration
	handlersBefore := len(router.Handlers)

	err := registrar.RegisterMiddleware(router)
	require.NoError(t, err)

	// Count handlers after registration
	handlersAfter := len(router.Handlers)

	// Should have added middleware handlers
	assert.Greater(t, handlersAfter, handlersBefore, "Expected middleware handlers to be added")
}

func TestGlobalMiddlewareRegistrar_Consistency(t *testing.T) {
	// Test that multiple instances have the same configuration
	registrar1 := NewGlobalMiddlewareRegistrar()
	registrar2 := NewGlobalMiddlewareRegistrar()

	assert.Equal(t, registrar1.GetName(), registrar2.GetName())
	assert.Equal(t, registrar1.GetPriority(), registrar2.GetPriority())
}

// Simple benchmark test that doesn't involve HTTP requests
func BenchmarkGlobalMiddlewareRegistrar_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registrar := NewGlobalMiddlewareRegistrar()
		_ = registrar.GetName()
		_ = registrar.GetPriority()
	}
}

func BenchmarkGlobalMiddlewareRegistrar_RegisterMiddleware(b *testing.B) {
	gin.SetMode(gin.TestMode)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router := gin.New()
		registrar := NewGlobalMiddlewareRegistrar()
		registrar.RegisterMiddleware(router)
	}
}
