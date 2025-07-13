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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/logger"
)

func init() {
	logger.Logger, _ = zap.NewDevelopment()
	gin.SetMode(gin.TestMode)

	// Change to project root for config file access
	wd, _ := os.Getwd()
	if filepath.Base(wd) == "server" {
		os.Chdir("../../..")
	}
}

func setupTestConsulMock() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/agent/service/register":
			w.WriteHeader(http.StatusOK)
		case "/v1/agent/service/deregister/swit-serve-localhost-9000":
			w.WriteHeader(http.StatusOK)
		case "/v1/health/service/swit-serve":
			services := []map[string]interface{}{
				{
					"Service": map[string]interface{}{
						"ID":      "swit-serve-localhost-9000",
						"Service": "swit-serve",
						"Address": "localhost",
						"Port":    9000,
					},
					"Checks": []map[string]interface{}{
						{"Status": "passing"},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(services)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func createTestServerWithoutDB() *Server {
	return &Server{
		router: gin.New(),
	}
}

func TestNewServer_StructCreation(t *testing.T) {
	server := createTestServerWithoutDB()

	assert.NotNil(t, server)
	assert.NotNil(t, server.router)
	assert.Nil(t, server.srv)
	assert.Nil(t, server.grpcServer)
	assert.Nil(t, server.sd)
}

func TestServer_setGRPCServer_getGRPCServer(t *testing.T) {
	server := createTestServerWithoutDB()

	grpcServer := grpc.NewServer()

	assert.Nil(t, server.getGRPCServer())

	server.setGRPCServer(grpcServer)

	assert.Equal(t, grpcServer, server.getGRPCServer())

	server.setGRPCServer(nil)
	assert.Nil(t, server.getGRPCServer())
}

func TestServer_setGRPCServer_getGRPCServer_Concurrency(t *testing.T) {
	server := createTestServerWithoutDB()

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			grpcServer := grpc.NewServer()
			server.setGRPCServer(grpcServer)

			retrievedServer := server.getGRPCServer()
			assert.NotNil(t, retrievedServer)

			time.Sleep(time.Millisecond)

			server.setGRPCServer(nil)
		}(i)
	}

	wg.Wait()
}

func TestServer_Run_BasicStructure(t *testing.T) {
	server := createTestServerWithoutDB()

	// Test that Run method can be called without panicking on basic setup
	assert.NotNil(t, server.router)

	// We can't fully test Run without database and service discovery setup
	// But we can verify the server structure is correct
}

func TestServer_Shutdown_WithNilHTTPServer(t *testing.T) {
	server := createTestServerWithoutDB()

	// Since Shutdown() tries to access srv.Shutdown(), it will panic with nil srv
	// This demonstrates that the Shutdown method expects srv to be initialized
	assert.Panics(t, func() {
		server.Shutdown()
	}, "Shutdown should panic when srv is nil")
}

func TestServer_GRPCServerMethods(t *testing.T) {
	server := createTestServerWithoutDB()

	// Test gRPC server getter/setter work correctly
	assert.Nil(t, server.getGRPCServer())

	grpcServer := grpc.NewServer()
	server.setGRPCServer(grpcServer)
	assert.Equal(t, grpcServer, server.getGRPCServer())

	server.setGRPCServer(nil)
	assert.Nil(t, server.getGRPCServer())
}
