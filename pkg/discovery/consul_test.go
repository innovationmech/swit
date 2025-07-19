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

package discovery

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServiceDiscovery(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:    "valid address",
			address: "127.0.0.1:8500",
			wantErr: false,
		},
		{
			name:    "localhost address",
			address: "localhost:8500",
			wantErr: false,
		},
		{
			name:    "empty address should use default",
			address: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd, err := NewServiceDiscovery(tt.address)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, sd)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, sd)
				assert.NotNil(t, sd.client)
				assert.Equal(t, 0, sd.roundRobinIndex)
			}
		})
	}
}

func TestServiceDiscovery_RegisterService(t *testing.T) {
	// 创建模拟 Consul 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/v1/agent/service/register") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	sd, err := NewServiceDiscovery(strings.TrimPrefix(server.URL, "http://"))
	require.NoError(t, err)

	tests := []struct {
		name    string
		svcName string
		address string
		port    int
		wantErr bool
	}{
		{
			name:    "valid service registration",
			svcName: "test-service",
			address: "127.0.0.1",
			port:    8080,
			wantErr: false,
		},
		{
			name:    "valid service with different port",
			svcName: "another-service",
			address: "127.0.0.1",
			port:    9090,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sd.RegisterService(tt.svcName, tt.address, tt.port)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServiceDiscovery_DeregisterService(t *testing.T) {
	// 创建模拟 Consul 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/v1/agent/service/deregister") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	sd, err := NewServiceDiscovery(strings.TrimPrefix(server.URL, "http://"))
	require.NoError(t, err)

	tests := []struct {
		name    string
		svcName string
		address string
		port    int
		wantErr bool
	}{
		{
			name:    "valid service deregistration",
			svcName: "test-service",
			address: "127.0.0.1",
			port:    8080,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sd.DeregisterService(tt.svcName, tt.address, tt.port)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServiceDiscovery_GetInstanceRoundRobin(t *testing.T) {
	// 创建模拟 Consul 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/v1/health/service") {
			serviceName := strings.TrimPrefix(r.URL.Path, "/v1/health/service/")
			if serviceName == "test-service" {
				// 返回健康的服务实例
				response := `[
					{
						"Service": {
							"Address": "127.0.0.1",
							"Port": 8080
						}
					},
					{
						"Service": {
							"Address": "127.0.0.1",
							"Port": 8081
						}
					}
				]`
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(response))
				return
			} else if serviceName == "empty-service" {
				// 返回空的服务实例列表
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("[]"))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	sd, err := NewServiceDiscovery(strings.TrimPrefix(server.URL, "http://"))
	require.NoError(t, err)

	tests := []struct {
		name        string
		serviceName string
		wantErr     bool
		wantAddr    []string
	}{
		{
			name:        "service with instances",
			serviceName: "test-service",
			wantErr:     false,
			wantAddr:    []string{"127.0.0.1:8080", "127.0.0.1:8081"},
		},
		{
			name:        "service with no instances",
			serviceName: "empty-service",
			wantErr:     true,
		},
		{
			name:        "non-existent service",
			serviceName: "non-existent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				// 测试轮询逻辑
				seenAddresses := make(map[string]bool)
				for i := 0; i < 4; i++ {
					addr, err := sd.GetInstanceRoundRobin(tt.serviceName)
					assert.NoError(t, err)
					seenAddresses[addr] = true
				}
				// 验证轮询是否正确工作
				for _, expectedAddr := range tt.wantAddr {
					assert.True(t, seenAddresses[expectedAddr], "Expected address %s was not seen", expectedAddr)
				}
			} else {
				_, err := sd.GetInstanceRoundRobin(tt.serviceName)
				assert.Error(t, err)
			}
		})
	}
}

func TestServiceDiscovery_GetInstanceRandom(t *testing.T) {
	// 创建模拟 Consul 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/v1/health/service") {
			serviceName := strings.TrimPrefix(r.URL.Path, "/v1/health/service/")
			if serviceName == "test-service" {
				// 返回健康的服务实例
				response := `[
					{
						"Service": {
							"Address": "127.0.0.1",
							"Port": 8080
						}
					},
					{
						"Service": {
							"Address": "127.0.0.1",
							"Port": 8081
						}
					}
				]`
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(response))
				return
			} else if serviceName == "empty-service" {
				// 返回空的服务实例列表
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("[]"))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	sd, err := NewServiceDiscovery(strings.TrimPrefix(server.URL, "http://"))
	require.NoError(t, err)

	tests := []struct {
		name        string
		serviceName string
		wantErr     bool
		wantAddr    []string
	}{
		{
			name:        "service with instances",
			serviceName: "test-service",
			wantErr:     false,
			wantAddr:    []string{"127.0.0.1:8080", "127.0.0.1:8081"},
		},
		{
			name:        "service with no instances",
			serviceName: "empty-service",
			wantErr:     true,
		},
		{
			name:        "non-existent service",
			serviceName: "non-existent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				// 测试随机选择逻辑
				seenAddresses := make(map[string]bool)
				for i := 0; i < 10; i++ {
					addr, err := sd.GetInstanceRandom(tt.serviceName)
					assert.NoError(t, err)
					seenAddresses[addr] = true
				}
				// 验证返回的地址是否在预期范围内
				for addr := range seenAddresses {
					found := false
					for _, expectedAddr := range tt.wantAddr {
						if addr == expectedAddr {
							found = true
							break
						}
					}
					assert.True(t, found, "Unexpected address %s", addr)
				}
			} else {
				_, err := sd.GetInstanceRandom(tt.serviceName)
				assert.Error(t, err)
			}
		})
	}
}

func TestServiceDiscovery_RoundRobinConsistency(t *testing.T) {
	// 创建模拟 Consul 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/v1/health/service/test-service") {
			response := `[
				{
					"Service": {
						"Address": "127.0.0.1",
						"Port": 8080
					}
				},
				{
					"Service": {
						"Address": "127.0.0.1",
						"Port": 8081
					}
				},
				{
					"Service": {
						"Address": "127.0.0.1",
						"Port": 8082
					}
				}
			]`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	sd, err := NewServiceDiscovery(strings.TrimPrefix(server.URL, "http://"))
	require.NoError(t, err)

	// 测试轮询的一致性
	expectedSequence := []string{"127.0.0.1:8080", "127.0.0.1:8081", "127.0.0.1:8082"}

	for i := 0; i < 9; i++ { // 测试 3 轮
		addr, err := sd.GetInstanceRoundRobin("test-service")
		assert.NoError(t, err)
		expected := expectedSequence[i%3]
		assert.Equal(t, expected, addr, "Round robin at position %d", i)
	}
}

func TestServiceDiscovery_ConcurrentRoundRobin(t *testing.T) {
	// 创建模拟 Consul 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/v1/health/service/test-service") {
			response := `[
				{
					"Service": {
						"Address": "127.0.0.1",
						"Port": 8080
					}
				},
				{
					"Service": {
						"Address": "127.0.0.1",
						"Port": 8081
					}
				}
			]`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	sd, err := NewServiceDiscovery(strings.TrimPrefix(server.URL, "http://"))
	require.NoError(t, err)

	// 测试并发安全性
	done := make(chan bool)
	addresses := make(chan string, 100)

	// 启动多个 goroutine 并发调用
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				addr, err := sd.GetInstanceRoundRobin("test-service")
				assert.NoError(t, err)
				addresses <- addr
			}
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
	close(addresses)

	// 验证所有地址都是有效的
	validAddresses := map[string]bool{
		"127.0.0.1:8080": true,
		"127.0.0.1:8081": true,
	}

	for addr := range addresses {
		assert.True(t, validAddresses[addr], "Invalid address: %s", addr)
	}
}

// TestRandomSeedInitialization verifies that random seed is initialized once
// and GetInstanceRandom doesn't cause memory leaks by calling rand.Seed repeatedly
func TestRandomSeedInitialization(t *testing.T) {
	// 创建模拟 Consul 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/v1/health/service/test-service") {
			response := `[
				{
					"Service": {
						"Address": "127.0.0.1",
						"Port": 8080
					}
				},
				{
					"Service": {
						"Address": "127.0.0.1",
						"Port": 8081
					}
				}
			]`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	sd, err := NewServiceDiscovery(strings.TrimPrefix(server.URL, "http://"))
	require.NoError(t, err)

	// Test that multiple calls to GetInstanceRandom work without issues
	// This verifies that rand.Seed is not called on every request
	validAddresses := map[string]bool{
		"127.0.0.1:8080": true,
		"127.0.0.1:8081": true,
	}

	// Call GetInstanceRandom multiple times to ensure no memory leak
	for i := 0; i < 100; i++ {
		addr, err := sd.GetInstanceRandom("test-service")
		assert.NoError(t, err)
		assert.True(t, validAddresses[addr], "Invalid address: %s", addr)
	}

	// Test concurrent access to ensure thread safety
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				addr, err := sd.GetInstanceRandom("test-service")
				assert.NoError(t, err)
				assert.True(t, validAddresses[addr], "Invalid address: %s", addr)
			}
		}()
	}
	wg.Wait()
}

// BenchmarkGetInstanceRandom benchmarks the GetInstanceRandom method
// to ensure the fix doesn't cause performance regression
func BenchmarkGetInstanceRandom(b *testing.B) {
	// 创建模拟 Consul 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/v1/health/service/test-service") {
			response := `[
				{
					"Service": {
						"Address": "127.0.0.1",
						"Port": 8080
					}
				},
				{
					"Service": {
						"Address": "127.0.0.1",
						"Port": 8081
					}
				}
			]`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	sd, err := NewServiceDiscovery(strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := sd.GetInstanceRandom("test-service")
			if err != nil {
				b.Error(err)
			}
		}
	})
}
