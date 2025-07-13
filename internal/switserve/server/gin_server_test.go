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
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

func init() {
	logger.Logger, _ = zap.NewDevelopment()
}

func createTestServerForGin() *Server {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	return &Server{
		router: router,
	}
}

func TestServer_runGinServer(t *testing.T) {
	tests := []struct {
		name        string
		addr        string
		expectError bool
	}{
		{
			name:        "valid address",
			addr:        ":0",
			expectError: false,
		},
		{
			name:        "invalid address",
			addr:        "invalid:addr:format",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := createTestServerForGin()
			var wg sync.WaitGroup
			ch := make(chan struct{})

			wg.Add(1)
			go s.runGinServer(tt.addr, &wg, ch)

			if !tt.expectError {
				select {
				case <-ch:
				case <-time.After(2 * time.Second):
					t.Fatal("channel was not closed within timeout")
				}

				require.NotNil(t, s.srv)
				assert.Equal(t, tt.addr, s.srv.Addr)
				assert.Equal(t, s.router, s.srv.Handler)

				err := s.srv.Shutdown(context.Background())
				assert.NoError(t, err)
			} else {
				select {
				case <-ch:
					t.Fatal("channel should not be closed for invalid address")
				case <-time.After(100 * time.Millisecond):
				}
			}

			wg.Wait()
		})
	}
}

func TestServer_runGinServer_ServerCreation(t *testing.T) {
	s := createTestServerForGin()
	var wg sync.WaitGroup
	ch := make(chan struct{})

	wg.Add(1)
	go s.runGinServer(":0", &wg, ch)

	<-ch

	assert.NotNil(t, s.srv)
	assert.Equal(t, ":0", s.srv.Addr)
	assert.Equal(t, s.router, s.srv.Handler)

	err := s.srv.Shutdown(context.Background())
	assert.NoError(t, err)

	wg.Wait()
}

func TestServer_runGinServer_ChannelClosure(t *testing.T) {
	s := createTestServerForGin()
	var wg sync.WaitGroup
	ch := make(chan struct{})

	wg.Add(1)
	go s.runGinServer(":0", &wg, ch)

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("channel was not closed within timeout")
	}

	err := s.srv.Shutdown(context.Background())
	assert.NoError(t, err)

	wg.Wait()
}

func TestServer_runGinServer_HTTPRequests(t *testing.T) {
	s := createTestServerForGin()
	var wg sync.WaitGroup
	ch := make(chan struct{})

	wg.Add(1)
	go s.runGinServer(":0", &wg, ch)

	<-ch

	listener, err := net.Listen("tcp", s.srv.Addr)
	require.NoError(t, err)
	actualAddr := listener.Addr().String()
	listener.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/test", actualAddr))
	if err == nil {
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	err = s.srv.Shutdown(context.Background())
	assert.NoError(t, err)

	wg.Wait()
}

func TestServer_runGinServer_WaitGroupDone(t *testing.T) {
	s := createTestServerForGin()
	var wg sync.WaitGroup
	ch := make(chan struct{})

	wg.Add(1)
	go s.runGinServer(":0", &wg, ch)

	<-ch

	err := s.srv.Shutdown(context.Background())
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("WaitGroup.Done() was not called within timeout")
	}
}

func TestServer_runGinServer_InvalidPort(t *testing.T) {
	s := createTestServerForGin()
	var wg sync.WaitGroup
	ch := make(chan struct{})

	wg.Add(1)
	go s.runGinServer(":99999", &wg, ch)

	select {
	case <-ch:
		t.Fatal("channel should not be closed for invalid port")
	case <-time.After(100 * time.Millisecond):
	}

	wg.Wait()
}

func TestServer_runGinServer_ConcurrentCalls(t *testing.T) {
	s1 := createTestServerForGin()
	s2 := createTestServerForGin()

	var wg sync.WaitGroup
	ch1 := make(chan struct{})
	ch2 := make(chan struct{})

	wg.Add(2)
	go s1.runGinServer(":0", &wg, ch1)
	go s2.runGinServer(":0", &wg, ch2)

	<-ch1
	<-ch2

	assert.NotNil(t, s1.srv)
	assert.NotNil(t, s2.srv)

	err1 := s1.srv.Shutdown(context.Background())
	err2 := s2.srv.Shutdown(context.Background())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	wg.Wait()
}
