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

package stop

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockStopService implements StopService for testing
type mockStopService struct {
	shouldFail bool
	called     bool
}

func (m *mockStopService) InitiateShutdown(ctx context.Context) (*ShutdownStatus, error) {
	m.called = true
	if m.shouldFail {
		return nil, assert.AnError
	}
	return &ShutdownStatus{
		Status:    "stopping",
		Message:   "Server is shutting down",
		Timestamp: time.Now().Unix(),
	}, nil
}

func TestNewStopHandler(t *testing.T) {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
	defer logger.Logger.Sync()

	service := &mockStopService{}
	h := NewHandler(service)
	assert.NotNil(t, h)
	assert.False(t, service.called)
}

func TestStopHandler_StopServer(t *testing.T) {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
	defer logger.Logger.Sync()

	gin.SetMode(gin.TestMode)
	service := &mockStopService{}
	h := NewHandler(service)

	r := gin.New()
	r.POST("/stop", h.StopServer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/stop", nil)

	r.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "stopping")

	// Check service was called
	assert.True(t, service.called)
}

func TestStopRouteRegistration(t *testing.T) {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
	defer logger.Logger.Sync()

	gin.SetMode(gin.TestMode)
	service := &mockStopService{}
	r := gin.New()
	handler := NewHandler(service)
	r.POST("/stop", handler.StopServer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/stop", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "stopping")
	assert.True(t, service.called)
}

func TestStopHandler_ServiceError(t *testing.T) {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
	defer logger.Logger.Sync()

	gin.SetMode(gin.TestMode)
	service := &mockStopService{shouldFail: true}
	h := NewHandler(service)

	r := gin.New()
	r.POST("/stop", h.StopServer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/stop", nil)

	r.ServeHTTP(w, req)

	// Should return error response
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "error")
	assert.True(t, service.called)
}
