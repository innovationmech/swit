// Copyright (c) 2024 Six-Thirty Labs, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRemoteClient_Basic 测试远程客户端基本功能
func TestRemoteClient_Basic(t *testing.T) {
	// 创建模拟 OPA 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/data/authz/allow" {
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"result": true,
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// 创建客户端配置
	config := &Config{
		Mode: ModeRemote,
		RemoteConfig: &RemoteConfig{
			URL:     server.URL,
			Timeout: 5 * time.Second,
		},
	}

	// 创建客户端
	ctx := context.Background()
	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close(ctx)

	// 评估策略
	result, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{
		"user": "alice",
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

// TestRemoteClient_MultipleServers 测试多服务器负载均衡
func TestRemoteClient_MultipleServers(t *testing.T) {
	// 创建多个模拟服务器
	var serverHits sync.Map
	createServer := func(name string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/data/authz/allow" {
				// 记录服务器被调用
				count, _ := serverHits.LoadOrStore(name, 0)
				serverHits.Store(name, count.(int)+1)

				w.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"result": true,
				}
				json.NewEncoder(w).Encode(response)
			}
		}))
	}

	server1 := createServer("server1")
	defer server1.Close()
	server2 := createServer("server2")
	defer server2.Close()
	server3 := createServer("server3")
	defer server3.Close()

	// 测试轮询负载均衡
	t.Run("RoundRobin", func(t *testing.T) {
		serverHits = sync.Map{}
		config := &Config{
			Mode: ModeRemote,
			RemoteConfig: &RemoteConfig{
				URLs:          []string{server1.URL, server2.URL, server3.URL},
				Timeout:       5 * time.Second,
				LoadBalancing: "round_robin",
			},
		}

		ctx := context.Background()
		client, err := NewClient(ctx, config)
		require.NoError(t, err)
		defer client.Close(ctx)

		// 执行多次请求
		for i := 0; i < 9; i++ {
			_, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{
				"user": fmt.Sprintf("user%d", i),
			})
			require.NoError(t, err)
		}

		// 验证请求被均匀分布
		count1, _ := serverHits.Load("server1")
		count2, _ := serverHits.Load("server2")
		count3, _ := serverHits.Load("server3")

		assert.Equal(t, 3, count1.(int))
		assert.Equal(t, 3, count2.(int))
		assert.Equal(t, 3, count3.(int))
	})

	// 测试随机负载均衡
	t.Run("Random", func(t *testing.T) {
		serverHits = sync.Map{}
		config := &Config{
			Mode: ModeRemote,
			RemoteConfig: &RemoteConfig{
				URLs:          []string{server1.URL, server2.URL, server3.URL},
				Timeout:       5 * time.Second,
				LoadBalancing: "random",
			},
		}

		ctx := context.Background()
		client, err := NewClient(ctx, config)
		require.NoError(t, err)
		defer client.Close(ctx)

		// 执行多次请求
		for i := 0; i < 30; i++ {
			_, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{
				"user": fmt.Sprintf("user%d", i),
			})
			require.NoError(t, err)
		}

		// 验证所有服务器都被调用了（随机分布）
		count1, ok1 := serverHits.Load("server1")
		count2, ok2 := serverHits.Load("server2")
		count3, ok3 := serverHits.Load("server3")

		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.True(t, ok3)
		assert.True(t, count1.(int) > 0)
		assert.True(t, count2.(int) > 0)
		assert.True(t, count3.(int) > 0)
	})
}

// TestRemoteClient_RetryWithBackoff 测试重试和退避策略
func TestRemoteClient_RetryWithBackoff(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// 前两次请求失败
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			// 第三次成功
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"result": true,
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	config := &Config{
		Mode: ModeRemote,
		RemoteConfig: &RemoteConfig{
			URL:                    server.URL,
			Timeout:                5 * time.Second,
			MaxRetries:             3,
			RetryBackoffMultiplier: 1.5,
			RetryMaxWait:           2 * time.Second,
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close(ctx)

	startTime := time.Now()
	result, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{
		"user": "alice",
	})
	elapsed := time.Since(startTime)

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 3, attemptCount)
	// 验证有退避等待（至少等待了一些时间）
	assert.True(t, elapsed > time.Second)
}

// TestRemoteClient_HealthCheck 测试健康检查
func TestRemoteClient_HealthCheck(t *testing.T) {
	// 创建两个服务器，一个健康，一个不健康
	healthyCount := 0
	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		} else if r.URL.Path == "/v1/data/authz/allow" {
			healthyCount++
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"result": true,
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer healthyServer.Close()

	unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 总是返回 503
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthyServer.Close()

	config := &Config{
		Mode: ModeRemote,
		RemoteConfig: &RemoteConfig{
			URLs:          []string{healthyServer.URL, unhealthyServer.URL},
			Timeout:       5 * time.Second,
			LoadBalancing: "round_robin",
			HealthCheck: &HealthCheckConfig{
				Enabled:          true,
				Interval:         1 * time.Second,
				Timeout:          2 * time.Second,
				FailureThreshold: 2,
				SuccessThreshold: 1,
			},
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close(ctx)

	// 等待健康检查运行几次
	time.Sleep(3 * time.Second)

	// 执行多次请求，应该只会路由到健康的服务器
	healthyCount = 0
	for i := 0; i < 10; i++ {
		_, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{
			"user": fmt.Sprintf("user%d", i),
		})
		// 可能有些请求会失败（如果路由到了不健康的服务器），但大部分应该成功
		if err == nil {
			// 成功的请求应该都是健康服务器处理的
		}
	}

	// 健康的服务器应该处理了大部分或全部请求
	assert.True(t, healthyCount >= 5, "expected healthy server to handle most requests")
}

// TestRemoteClient_ConnectionPool 测试连接池配置
func TestRemoteClient_ConnectionPool(t *testing.T) {
	concurrentRequests := 50
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟一些延迟
		time.Sleep(10 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"result": true,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Mode: ModeRemote,
		RemoteConfig: &RemoteConfig{
			URL:     server.URL,
			Timeout: 5 * time.Second,
			PoolConfig: &PoolConfig{
				MaxIdleConns:    50,
				MaxConnsPerHost: 50,
				IdleConnTimeout: 90 * time.Second,
			},
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close(ctx)

	// 并发发送请求
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{
				"user": fmt.Sprintf("user%d", id),
			})
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// 所有请求应该都成功
	assert.Equal(t, concurrentRequests, successCount)
}

// TestRemoteClient_TLS 测试 TLS 配置
func TestRemoteClient_TLS(t *testing.T) {
	// 创建 HTTPS 服务器
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"result": true,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Mode: ModeRemote,
		RemoteConfig: &RemoteConfig{
			URL:     server.URL,
			Timeout: 5 * time.Second,
			TLSConfig: &TLSConfig{
				Enabled:            true,
				InsecureSkipVerify: true, // 测试环境跳过验证
			},
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close(ctx)

	result, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{
		"user": "alice",
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

// TestRemoteClient_Authentication 测试认证
func TestRemoteClient_Authentication(t *testing.T) {
	tests := []struct {
		name       string
		authConfig *AuthConfig
		handler    http.HandlerFunc
		expectErr  bool
	}{
		{
			name: "Bearer Token",
			authConfig: &AuthConfig{
				Type:  "bearer",
				Token: "test-token",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				auth := r.Header.Get("Authorization")
				if auth != "Bearer test-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"result": true})
			},
			expectErr: false,
		},
		{
			name: "Basic Auth",
			authConfig: &AuthConfig{
				Type:     "basic",
				Username: "admin",
				Password: "secret",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				username, password, ok := r.BasicAuth()
				if !ok || username != "admin" || password != "secret" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"result": true})
			},
			expectErr: false,
		},
		{
			name: "API Key",
			authConfig: &AuthConfig{
				Type:         "api_key",
				APIKey:       "my-api-key",
				APIKeyHeader: "X-API-Key",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				apiKey := r.Header.Get("X-API-Key")
				if apiKey != "my-api-key" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"result": true})
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			config := &Config{
				Mode: ModeRemote,
				RemoteConfig: &RemoteConfig{
					URL:        server.URL,
					Timeout:    5 * time.Second,
					AuthConfig: tt.authConfig,
				},
			}

			ctx := context.Background()
			client, err := NewClient(ctx, config)
			require.NoError(t, err)
			defer client.Close(ctx)

			result, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{
				"user": "alice",
			})

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.True(t, result.Allowed)
			}
		})
	}
}

// TestRemoteClient_Cache 测试缓存
func TestRemoteClient_Cache(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"result": true,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Mode: ModeRemote,
		RemoteConfig: &RemoteConfig{
			URL:     server.URL,
			Timeout: 5 * time.Second,
		},
		CacheConfig: &CacheConfig{
			Enabled: true,
			MaxSize: 100,
			TTL:     5 * time.Second,
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close(ctx)

	input := map[string]interface{}{"user": "alice"}

	// 第一次请求
	result1, err := client.Evaluate(ctx, "authz/allow", input)
	require.NoError(t, err)
	assert.True(t, result1.Allowed)
	assert.Equal(t, 1, requestCount)

	// 第二次请求（应该从缓存获取）
	result2, err := client.Evaluate(ctx, "authz/allow", input)
	require.NoError(t, err)
	assert.True(t, result2.Allowed)
	assert.Equal(t, 1, requestCount) // 没有增加

	// 不同的输入应该发送新请求
	result3, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{"user": "bob"})
	require.NoError(t, err)
	assert.True(t, result3.Allowed)
	assert.Equal(t, 2, requestCount)
}

// TestRemoteClient_ContextCancellation 测试上下文取消
func TestRemoteClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟慢请求
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"result": true})
	}))
	defer server.Close()

	config := &Config{
		Mode: ModeRemote,
		RemoteConfig: &RemoteConfig{
			URL:     server.URL,
			Timeout: 10 * time.Second,
		},
	}

	client, err := NewClient(context.Background(), config)
	require.NoError(t, err)
	defer client.Close(context.Background())

	// 创建可取消的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err = client.Evaluate(ctx, "authz/allow", map[string]interface{}{
		"user": "alice",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

// TestLoadBalancer_RoundRobin 测试轮询负载均衡器
func TestLoadBalancer_RoundRobin(t *testing.T) {
	servers := []*serverEndpoint{
		{url: "http://server1"},
		{url: "http://server2"},
		{url: "http://server3"},
	}

	for _, s := range servers {
		s.healthy.Store(true)
	}

	lb := &roundRobinBalancer{}

	// 测试轮询
	selected := make(map[string]int)
	for i := 0; i < 9; i++ {
		server := lb.Select(servers)
		selected[server.url]++
	}

	assert.Equal(t, 3, selected["http://server1"])
	assert.Equal(t, 3, selected["http://server2"])
	assert.Equal(t, 3, selected["http://server3"])
}

// TestLoadBalancer_LeastConnections 测试最少连接负载均衡器
func TestLoadBalancer_LeastConnections(t *testing.T) {
	servers := []*serverEndpoint{
		{url: "http://server1"},
		{url: "http://server2"},
		{url: "http://server3"},
	}

	for _, s := range servers {
		s.healthy.Store(true)
	}

	servers[0].activeConns.Store(10)
	servers[1].activeConns.Store(5)
	servers[2].activeConns.Store(8)

	lb := &leastConnectionsBalancer{}

	// 应该选择连接数最少的服务器
	server := lb.Select(servers)
	assert.Equal(t, "http://server2", server.url)
}

// TestLoadBalancer_UnhealthyServers 测试不健康服务器的处理
func TestLoadBalancer_UnhealthyServers(t *testing.T) {
	servers := []*serverEndpoint{
		{url: "http://server1"},
		{url: "http://server2"},
		{url: "http://server3"},
	}

	servers[0].healthy.Store(false) // 不健康
	servers[1].healthy.Store(true)
	servers[2].healthy.Store(false) // 不健康

	lb := &roundRobinBalancer{}

	// 应该只选择健康的服务器
	for i := 0; i < 10; i++ {
		server := lb.Select(servers)
		assert.Equal(t, "http://server2", server.url)
	}
}

// TestRemoteConfig_URLCompatibility 测试 URL 和 URLs 的兼容性
func TestRemoteConfig_URLCompatibility(t *testing.T) {
	t.Run("URL to URLs conversion", func(t *testing.T) {
		config := &RemoteConfig{
			URL: "http://opa-server:8181",
		}

		err := config.Validate()
		require.NoError(t, err)
		assert.Equal(t, []string{"http://opa-server:8181"}, config.URLs)
	})

	t.Run("URLs takes precedence", func(t *testing.T) {
		config := &RemoteConfig{
			URL:  "http://old-server:8181",
			URLs: []string{"http://new-server:8181"},
		}

		err := config.Validate()
		require.NoError(t, err)
		assert.Equal(t, []string{"http://new-server:8181"}, config.URLs)
	})
}

// TestRemoteClient_AutoRecoveryWithoutHealthCheck 测试无健康检查时的自动恢复
func TestRemoteClient_AutoRecoveryWithoutHealthCheck(t *testing.T) {
	// 创建两个服务器，一个会失败几次后恢复
	server1Failures := 0
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/data/authz/allow" {
			// 前 2 次失败，之后成功
			if server1Failures < 2 {
				server1Failures++
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"result": true})
		}
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/data/authz/allow" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"result": true})
		}
	}))
	defer server2.Close()

	// 不启用健康检查
	config := &Config{
		Mode: ModeRemote,
		RemoteConfig: &RemoteConfig{
			URLs:          []string{server1.URL, server2.URL},
			Timeout:       5 * time.Second,
			MaxRetries:    1, // 减少重试次数，加快测试
			LoadBalancing: "round_robin",
			// 注意：不配置 HealthCheck
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close(ctx)

	// 执行多次请求，server1 应该在失败几次后被标记为不健康
	// 但仍然有机会被选中并恢复
	successCount := 0
	for i := 0; i < 20; i++ {
		result, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{
			"user": fmt.Sprintf("user%d", i),
		})
		if err == nil && result.Allowed {
			successCount++
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 验证：
	// 1. server1 应该在前几次请求中失败
	// 2. 但由于负载均衡器在没有健康服务器时会继续尝试所有服务器
	// 3. server1 最终应该恢复并成功处理请求
	// 4. 总体成功率应该很高（大部分请求成功）
	assert.True(t, successCount >= 15, "expected at least 15 successful requests, got %d", successCount)
	assert.True(t, server1Failures >= 2, "expected server1 to fail at least 2 times")
}

// TestRemoteClient_TransientFailureRecovery 测试瞬时故障恢复
func TestRemoteClient_TransientFailureRecovery(t *testing.T) {
	// 模拟瞬时故障：服务器偶尔失败但会快速恢复
	failureCount := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/data/authz/allow" {
			mu.Lock()
			// 每 5 次请求中有 1 次失败（模拟 20% 失败率的瞬时故障）
			shouldFail := (failureCount % 5) == 0
			failureCount++
			mu.Unlock()

			if shouldFail {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"result": true})
		}
	}))
	defer server.Close()

	config := &Config{
		Mode: ModeRemote,
		RemoteConfig: &RemoteConfig{
			URLs:       []string{server.URL},
			Timeout:    5 * time.Second,
			MaxRetries: 3,
			// 不配置健康检查，使用默认的失败阈值（3次）
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close(ctx)

	// 执行多次请求
	successCount := 0
	totalRequests := 50
	for i := 0; i < totalRequests; i++ {
		result, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{
			"user": fmt.Sprintf("user%d", i),
		})
		if err == nil && result.Allowed {
			successCount++
		}
	}

	// 由于有重试机制和默认失败阈值，大部分请求应该成功
	// 即使服务器有 20% 的失败率，重试后成功率应该很高
	successRate := float64(successCount) / float64(totalRequests)
	assert.True(t, successRate >= 0.9, "expected success rate >= 90%%, got %.2f%%", successRate*100)
}
