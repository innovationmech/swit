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
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/open-policy-agent/opa/rego"
)

// remoteClient 远程 OPA 客户端实现
type remoteClient struct {
	config     *Config
	httpClient *http.Client
	cache      Cache
}

// newRemoteClient 创建远程客户端
func newRemoteClient(ctx context.Context, config *Config) (Client, error) {
	client := &remoteClient{
		config: config,
	}

	// 创建 HTTP 客户端
	httpClient, err := client.createHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	client.httpClient = httpClient

	// 初始化缓存
	if config.CacheConfig != nil && config.CacheConfig.Enabled {
		client.cache, err = NewCache(config.CacheConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache: %w", err)
		}
	}

	return client, nil
}

// createHTTPClient 创建 HTTP 客户端
func (c *remoteClient) createHTTPClient() (*http.Client, error) {
	transport := &http.Transport{}

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

	// 构建请求 - 使用斜杠分隔的路径
	url := strings.TrimRight(c.config.RemoteConfig.URL, "/") + "/v1/data/" + strings.TrimPrefix(slashPath, "/")

	requestBody := map[string]interface{}{
		"input": input,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求（带重试）
	var result *Result
	var lastErr error

	maxRetries := c.config.RemoteConfig.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1
	}

	for i := 0; i < maxRetries; i++ {
		result, lastErr = c.doRequest(ctx, url, jsonData)
		if lastErr == nil {
			break
		}

		// 如果不是最后一次重试，等待后重试
		if i < maxRetries-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Second * time.Duration(i+1)):
				// 指数退避
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

// LoadPolicy 加载策略（远程模式不支持）
func (c *remoteClient) LoadPolicy(ctx context.Context, name string, policy string) error {
	return fmt.Errorf("LoadPolicy is not supported in remote mode")
}

// RemovePolicy 移除策略（远程模式不支持）
func (c *remoteClient) RemovePolicy(ctx context.Context, name string) error {
	return fmt.Errorf("RemovePolicy is not supported in remote mode")
}

// Close 关闭客户端
func (c *remoteClient) Close(ctx context.Context) error {
	if c.cache != nil {
		return c.cache.Close(ctx)
	}
	return nil
}

// IsEmbedded 是否为嵌入式模式
func (c *remoteClient) IsEmbedded() bool {
	return false
}
