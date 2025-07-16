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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewStopHandler(t *testing.T) {
	var called bool
	h := NewHandler(func() { called = true })
	assert.NotNil(t, h)
	assert.False(t, called)
}

func TestStopHandler_Stop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shutdownCalled := make(chan bool, 1)
	h := NewHandler(func() {
		select {
		case shutdownCalled <- true:
		default:
		}
	})

	r := gin.New()
	r.POST("/stop", h.Stop)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/stop", nil)

	r.ServeHTTP(w, req)

	// 检查响应
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Server is stopping")

	// 检查 shutdownFunc 是否被调用
	select {
	case <-shutdownCalled:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("shutdownFunc not called")
	}
}

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shutdownCalled := make(chan bool, 1)
	r := gin.New()
	RegisterRoutes(r, func() {
		select {
		case shutdownCalled <- true:
		default:
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/stop", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Server is stopping")

	select {
	case <-shutdownCalled:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("shutdownFunc not called")
	}
}
