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

package server_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/innovationmech/swit/pkg/testing/servertest"
)

// TestServerWithServertestToolkit demonstrates and verifies usage of the
// reusable pkg/testing/servertest toolkit for framework server testing.
func TestServerWithServertestToolkit(t *testing.T) {
	handler := servertest.NewHTTPHandler("toolkit-service")
	handler.AddRoute(http.MethodGet, "/toolkit/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	deps := servertest.NewDependencyContainer()
	deps.AddService("cache", "fake-cache")

	registrar := servertest.NewServiceRegistrar()
	registrar.AddHTTPHandler(handler)
	registrar.AddGRPCService(servertest.NewGRPCService("toolkit-grpc", nil))
	registrar.AddHealthCheck(servertest.NewHealthCheck("toolkit-service", true))

	srv := servertest.StartServer(t, servertest.NewServerConfig("toolkit-test"), registrar, deps)

	baseURL := servertest.HTTPBaseURL(srv)
	servertest.WaitForHTTPReady(t, baseURL+"/toolkit/ping", 5*time.Second)

	resp, err := http.Get(baseURL + "/toolkit/ping")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	service, err := deps.GetService("cache")
	require.NoError(t, err)
	assert.Equal(t, "fake-cache", service)

	assert.NotEmpty(t, srv.GetHTTPAddress())
	assert.NotEmpty(t, srv.GetGRPCAddress())
}
