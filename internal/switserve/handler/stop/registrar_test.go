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

func TestNewStopRouteRegistrar(t *testing.T) {
	var called bool
	r := NewStopRouteRegistrar(func() { called = true })
	assert.NotNil(t, r)
	assert.False(t, called)
}

func TestStopRouteRegistrar_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shutdownCalled := make(chan bool, 1)
	r := NewStopRouteRegistrar(func() { shutdownCalled <- true })

	engine := gin.New()
	rg := engine.Group("")
	err := r.RegisterRoutes(rg)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/stop", nil)

	go func() {
		engine.ServeHTTP(w, req)
	}()

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Server is stopping")

	select {
	case <-shutdownCalled:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("shutdownFunc not called")
	}
}

func TestStopRouteRegistrar_InterfaceMethods(t *testing.T) {
	r := NewStopRouteRegistrar(func() {})
	assert.Equal(t, "stop-service", r.GetName())
	assert.Equal(t, "root", r.GetVersion())
	assert.Equal(t, "", r.GetPrefix())
}
