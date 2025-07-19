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

package health

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func TestNewServiceRegistrar(t *testing.T) {
	mockService := NewService()
	registrar := NewServiceRegistrar(mockService)

	assert.NotNil(t, registrar)
	assert.NotNil(t, registrar.service)
	assert.Equal(t, "health", registrar.GetName())
}

func TestServiceRegistrar_GetName(t *testing.T) {
	mockService := NewService()
	registrar := NewServiceRegistrar(mockService)

	name := registrar.GetName()

	assert.Equal(t, "health", name)
}

func TestServiceRegistrar_RegisterGRPC(t *testing.T) {
	logger.Logger = zap.NewNop()

	mockService := NewService()
	registrar := NewServiceRegistrar(mockService)
	server := grpc.NewServer()

	err := registrar.RegisterGRPC(server)

	assert.NoError(t, err)
}

func TestServiceRegistrar_RegisterHTTP(t *testing.T) {
	logger.Logger = zap.NewNop()

	mockService := NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()

	err := registrar.RegisterHTTP(router)

	assert.NoError(t, err)
}

func TestServiceRegistrar_healthCheckHTTP_Success(t *testing.T) {
	mockService := NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()
	router.GET("/health", registrar.healthCheckHTTP)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
	assert.Contains(t, w.Body.String(), "timestamp")
}

func TestServiceRegistrar_healthCheckHTTP_Integration(t *testing.T) {
	logger.Logger = zap.NewNop()

	mockService := NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()

	err := registrar.RegisterHTTP(router)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
	assert.Contains(t, w.Body.String(), "server")
	assert.Contains(t, w.Body.String(), "version")
}
