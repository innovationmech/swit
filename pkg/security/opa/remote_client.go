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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/open-policy-agent/opa/rego"
)

// remoteClient 远程 OPA 客户端实现
type remoteClient struct {
	config       *Config
	httpClient   *http.Client
	cache        Cache
	servers      []*serverEndpoint
	loadBalancer LoadBalancer
	healthMgr    *healthManager
	closeCh      chan struct{}
	closeOnce    sync.Once
}

// serverEndpoint 服务器端点
type serverEndpoint struct {
	url             string
	healthy         atomic.Bool
	consecutiveFail atomic.Int32
	consecutiveSucc atomic.Int32
	activeConns     atomic.Int64 // 用于 least_connections 策略
}

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	Select(servers []*serverEndpoint) *serverEndpoint
}

// roundRobinBalancer 轮询负载均衡器
type roundRobinBalancer struct {
	counter atomic.Uint64
}

func (rb *roundRobinBalancer) Select(servers []*serverEndpoint) *serverEndpoint {
	healthy := make([]*serverEndpoint, 0, len(servers))
	for _, s := range servers {
		if s.healthy.Load() {
			healthy = append(healthy, s)
		}
	}

	if len(healthy) == 0 {
		// 如果没有健康的服务器，从所有服务器中轮询（给它们恢复的机会）
		if len(servers) > 0 {
			idx := rb.counter.Add(1) % uint64(len(servers))
			return servers[idx]
		}
		return nil
	}

	idx := rb.counter.Add(1) % uint64(len(healthy))
	return healthy[idx]
}

// randomBalancer 随机负载均衡器
type randomBalancer struct{}

func (rb *randomBalancer) Select(servers []*serverEndpoint) *serverEndpoint {
	healthy := make([]*serverEndpoint, 0, len(servers))
	for _, s := range servers {
		if s.healthy.Load() {
			healthy = append(healthy, s)
		}
	}

	if len(healthy) == 0 {
		// 如果没有健康的服务器，从所有服务器中随机选择（给它们恢复的机会）
		if len(servers) > 0 {
			return servers[rand.Intn(len(servers))]
		}
		return nil
	}

	return healthy[rand.Intn(len(healthy))]
}

// leastConnectionsBalancer 最少连接负载均衡器
type leastConnectionsBalancer struct{}

func (lb *leastConnectionsBalancer) Select(servers []*serverEndpoint) *serverEndpoint {
	healthy := make([]*serverEndpoint, 0, len(servers))
	for _, s := range servers {
		if s.healthy.Load() {
			healthy = append(healthy, s)
		}
	}

	if len(healthy) == 0 {
		// 如果没有健康的服务器，从所有服务器中选择连接数最少的（给它们恢复的机会）
		if len(servers) > 0 {
			var minServer *serverEndpoint
			var minConns int64 = -1
			for _, s := range servers {
				conns := s.activeConns.Load()
				if minConns < 0 || conns < minConns {
					minConns = conns
					minServer = s
				}
			}
			return minServer
		}
		return nil
	}

	var minServer *serverEndpoint
	var minConns int64 = -1
	for _, s := range healthy {
		conns := s.activeConns.Load()
		if minConns < 0 || conns < minConns {
			minConns = conns
			minServer = s
		}
	}

	return minServer
}

// healthManager 健康检查管理器
type healthManager struct {
	client  *remoteClient
	config  *HealthCheckConfig
	closeCh chan struct{}
	wg      sync.WaitGroup
}

// newRemoteClient 创建远程客户端
func newRemoteClient(ctx context.Context, config *Config) (Client, error) {
	client := &remoteClient{
		config:  config,
		closeCh: make(chan struct{}),
	}

	// 创建 HTTP 客户端
	httpClient, err := client.createHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	client.httpClient = httpClient

	// 初始化服务器端点列表
	client.servers = make([]*serverEndpoint, len(config.RemoteConfig.URLs))
	for i, url := range config.RemoteConfig.URLs {
		server := &serverEndpoint{
			url: strings.TrimRight(url, "/"),
		}
		server.healthy.Store(true) // 初始标记为健康
		client.servers[i] = server
	}

	// 创建负载均衡器
	client.loadBalancer = client.createLoadBalancer()

	// 初始化健康检查
	if config.RemoteConfig.HealthCheck != nil && config.RemoteConfig.HealthCheck.Enabled {
		client.healthMgr = &healthManager{
			client:  client,
			config:  config.RemoteConfig.HealthCheck,
			closeCh: make(chan struct{}),
		}
		client.healthMgr.start()
	}

	// 初始化缓存
	if config.CacheConfig != nil && config.CacheConfig.Enabled {
		client.cache, err = NewCache(config.CacheConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache: %w", err)
		}
	}

	return client, nil
}

// createLoadBalancer 创建负载均衡器
func (c *remoteClient) createLoadBalancer() LoadBalancer {
	strategy := c.config.RemoteConfig.LoadBalancing
	if strategy == "" {
		strategy = "round_robin"
	}

	switch strategy {
	case "random":
		return &randomBalancer{}
	case "least_connections":
		return &leastConnectionsBalancer{}
	default: // round_robin
		return &roundRobinBalancer{}
	}
}

// createHTTPClient 创建 HTTP 客户端
func (c *remoteClient) createHTTPClient() (*http.Client, error) {
	transport := &http.Transport{}

	// 配置连接池
	if c.config.RemoteConfig.PoolConfig != nil {
		pool := c.config.RemoteConfig.PoolConfig
		transport.MaxIdleConns = pool.MaxIdleConns
		transport.MaxConnsPerHost = pool.MaxConnsPerHost
		transport.IdleConnTimeout = pool.IdleConnTimeout
	} else {
		// 默认连接池配置
		transport.MaxIdleConns = 100
		transport.MaxConnsPerHost = 100
		transport.IdleConnTimeout = 90 * time.Second
	}

	// 配置 TLS
	if c.config.RemoteConfig.TLSConfig != nil && c.config.RemoteConfig.TLSConfig.Enabled {
		tlsConfig, err := c.createTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		transport.TLSClientConfig = tlsConfig
	}

	return &http.Client{
		Transport: transport,
		Timeout:   c.config.RemoteConfig.Timeout,
	}, nil
}

// createTLSConfig 创建 TLS 配置
func (c *remoteClient) createTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.config.RemoteConfig.TLSConfig.InsecureSkipVerify,
	}

	// 加载 CA 证书
	if c.config.RemoteConfig.TLSConfig.CAFile != "" {
		caCert, err := os.ReadFile(c.config.RemoteConfig.TLSConfig.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// 加载客户端证书
	if c.config.RemoteConfig.TLSConfig.CertFile != "" && c.config.RemoteConfig.TLSConfig.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(
			c.config.RemoteConfig.TLSConfig.CertFile,
			c.config.RemoteConfig.TLSConfig.KeyFile,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// Evaluate 评估策略决策
func (c *remoteClient) Evaluate(ctx context.Context, path string, input interface{}) (*Result, error) {
	// 如果未指定路径，使用默认路径
	if path == "" {
		path = c.config.DefaultDecisionPath
	}

	if path == "" {
		return nil, fmt.Errorf("decision path is required")
	}

	// 检查缓存
	if c.cache != nil {
		if cached, ok := c.cache.Get(ctx, path, input); ok {
			return cached, nil
		}
	}

	// 规范化路径：支持 "rbac/allow" 和 "rbac.allow" 两种格式
	_, slashPath := normalizePath(path)

	requestBody := map[string]interface{}{
		"input": input,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求（带重试和负载均衡）
	var result *Result
	var lastErr error

	maxRetries := c.config.RemoteConfig.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1
	}

	backoffMultiplier := c.config.RemoteConfig.RetryBackoffMultiplier
	if backoffMultiplier == 0 {
		backoffMultiplier = 2.0
	}

	maxWait := c.config.RemoteConfig.RetryMaxWait
	if maxWait == 0 {
		maxWait = 30 * time.Second
	}

	baseWait := time.Second

	for i := 0; i < maxRetries; i++ {
		// 选择服务器
		server := c.loadBalancer.Select(c.servers)
		if server == nil {
			return nil, fmt.Errorf("no available servers")
		}

		// 构建完整 URL
		url := server.url + "/v1/data/" + strings.TrimPrefix(slashPath, "/")

		// 跟踪活动连接数（用于 least_connections 策略）
		server.activeConns.Add(1)
		result, lastErr = c.doRequest(ctx, url, jsonData)
		server.activeConns.Add(-1)

		if lastErr == nil {
			// 成功：更新健康状态
			c.markServerSuccess(server)
			break
		}

		// 失败：更新健康状态
		c.markServerFailure(server)

		// 如果不是最后一次重试，等待后重试
		if i < maxRetries-1 {
			// 指数退避策略
			waitTime := time.Duration(float64(baseWait) * float64(i+1) * backoffMultiplier)
			if waitTime > maxWait {
				waitTime = maxWait
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitTime):
				// 继续重试
			}
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
	}

	// 缓存结果
	if c.cache != nil {
		c.cache.Set(ctx, path, input, result)
	}

	return result, nil
}

// markServerSuccess 标记服务器成功
func (c *remoteClient) markServerSuccess(server *serverEndpoint) {
	server.consecutiveFail.Store(0)
	succCount := server.consecutiveSucc.Add(1)

	// 如果连续成功次数达到阈值，标记为健康
	if c.config.RemoteConfig.HealthCheck != nil {
		threshold := int32(c.config.RemoteConfig.HealthCheck.SuccessThreshold)
		if succCount >= threshold {
			server.healthy.Store(true)
		}
	} else {
		server.healthy.Store(true)
	}
}

// markServerFailure 标记服务器失败
func (c *remoteClient) markServerFailure(server *serverEndpoint) {
	server.consecutiveSucc.Store(0)
	failCount := server.consecutiveFail.Add(1)

	// 如果连续失败次数达到阈值，标记为不健康
	if c.config.RemoteConfig.HealthCheck != nil {
		threshold := int32(c.config.RemoteConfig.HealthCheck.FailureThreshold)
		if failCount >= threshold {
			server.healthy.Store(false)
		}
	} else {
		// 如果没有健康检查配置，使用默认阈值（3次）避免单次失败永久移除服务器
		// 这样可以在瞬时故障时保持服务器在轮询池中，支持自动恢复
		const defaultFailureThreshold = 3
		if failCount >= defaultFailureThreshold {
			server.healthy.Store(false)
		}
	}
}

// doRequest 执行 HTTP 请求
func (c *remoteClient) doRequest(ctx context.Context, url string, jsonData []byte) (*Result, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 添加认证
	if c.config.RemoteConfig.AuthConfig != nil {
		if err := c.addAuth(req); err != nil {
			return nil, fmt.Errorf("failed to add auth: %w", err)
		}
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response struct {
		DecisionID string                 `json:"decision_id,omitempty"`
		Result     interface{}            `json:"result"`
		Metrics    map[string]interface{} `json:"metrics,omitempty"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 构建结果
	result := &Result{
		Decision:   response.Result,
		DecisionID: response.DecisionID,
		Metrics:    &Metrics{},
	}

	// 如果决策是布尔值，设置 Allowed 字段
	if allowed, ok := response.Result.(bool); ok {
		result.Allowed = allowed
	}

	// 提取指标
	if response.Metrics != nil {
		if evalTime, ok := response.Metrics["timer_rego_eval_ns"]; ok {
			if t, ok := evalTime.(float64); ok {
				result.Metrics.TimerEvalNs = int64(t)
			}
		}
		if queryTime, ok := response.Metrics["timer_rego_query_eval_ns"]; ok {
			if t, ok := queryTime.(float64); ok {
				result.Metrics.TimerRegoQueryEvalNs = int64(t)
			}
		}
	}

	return result, nil
}

// addAuth 添加认证信息
func (c *remoteClient) addAuth(req *http.Request) error {
	auth := c.config.RemoteConfig.AuthConfig
	if auth == nil {
		return nil
	}

	switch auth.Type {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	case "basic":
		req.SetBasicAuth(auth.Username, auth.Password)
	case "api_key":
		req.Header.Set(auth.APIKeyHeader, auth.APIKey)
	default:
		return fmt.Errorf("unsupported auth type: %s", auth.Type)
	}

	return nil
}

// Query 执行 Rego 查询（远程模式不支持）
func (c *remoteClient) Query(ctx context.Context, query string, input interface{}) (rego.ResultSet, error) {
	return nil, fmt.Errorf("Query is not supported in remote mode")
}

// PartialEvaluate 执行部分评估（远程模式不支持）
func (c *remoteClient) PartialEvaluate(ctx context.Context, query string, input interface{}) (*PartialResult, error) {
	return nil, fmt.Errorf("PartialEvaluate is not supported in remote mode")
}

// LoadPolicy 加载策略（远程模式不支持）
func (c *remoteClient) LoadPolicy(ctx context.Context, name string, policy string) error {
	return fmt.Errorf("LoadPolicy is not supported in remote mode")
}

// LoadPolicyFromFile 从文件加载策略（远程模式不支持）
func (c *remoteClient) LoadPolicyFromFile(ctx context.Context, path string) error {
	return fmt.Errorf("LoadPolicyFromFile is not supported in remote mode")
}

// LoadData 加载数据（远程模式不支持）
func (c *remoteClient) LoadData(ctx context.Context, path string, data interface{}) error {
	return fmt.Errorf("LoadData is not supported in remote mode")
}

// RemovePolicy 移除策略（远程模式不支持）
func (c *remoteClient) RemovePolicy(ctx context.Context, name string) error {
	return fmt.Errorf("RemovePolicy is not supported in remote mode")
}

// Close 关闭客户端
func (c *remoteClient) Close(ctx context.Context) error {
	c.closeOnce.Do(func() {
		close(c.closeCh)

		// 停止健康检查
		if c.healthMgr != nil {
			c.healthMgr.stop()
		}
	})

	if c.cache != nil {
		return c.cache.Close(ctx)
	}
	return nil
}

// start 启动健康检查
func (hm *healthManager) start() {
	hm.wg.Add(1)
	go hm.run()
}

// stop 停止健康检查
func (hm *healthManager) stop() {
	close(hm.closeCh)
	hm.wg.Wait()
}

// run 运行健康检查循环
func (hm *healthManager) run() {
	defer hm.wg.Done()

	ticker := time.NewTicker(hm.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-hm.closeCh:
			return
		case <-ticker.C:
			hm.checkAllServers()
		}
	}
}

// checkAllServers 检查所有服务器健康状态
func (hm *healthManager) checkAllServers() {
	for _, server := range hm.client.servers {
		// 并发检查所有服务器
		go hm.checkServer(server)
	}
}

// checkServer 检查单个服务器健康状态
func (hm *healthManager) checkServer(server *serverEndpoint) {
	ctx, cancel := context.WithTimeout(context.Background(), hm.config.Timeout)
	defer cancel()

	// 使用 OPA 的健康检查端点
	url := server.url + "/health"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		hm.handleCheckFailure(server)
		return
	}

	resp, err := hm.client.httpClient.Do(req)
	if err != nil {
		hm.handleCheckFailure(server)
		return
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode == http.StatusOK {
		hm.handleCheckSuccess(server)
	} else {
		hm.handleCheckFailure(server)
	}
}

// handleCheckSuccess 处理健康检查成功
func (hm *healthManager) handleCheckSuccess(server *serverEndpoint) {
	hm.client.markServerSuccess(server)
}

// handleCheckFailure 处理健康检查失败
func (hm *healthManager) handleCheckFailure(server *serverEndpoint) {
	hm.client.markServerFailure(server)
}

// IsEmbedded 是否为嵌入式模式
func (c *remoteClient) IsEmbedded() bool {
	return false
}
