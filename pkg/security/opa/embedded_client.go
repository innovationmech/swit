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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/storage/inmem"
)

// embeddedClient 嵌入式 OPA 客户端实现
type embeddedClient struct {
	config          *Config
	store           storage.Store
	modules         map[string]*ast.Module
	compiledQueries map[string]rego.PreparedEvalQuery // 编译缓存
	mu              sync.RWMutex
	cache           Cache
}

// newEmbeddedClient 创建嵌入式客户端
func newEmbeddedClient(ctx context.Context, config *Config) (Client, error) {
	client := &embeddedClient{
		config:          config,
		modules:         make(map[string]*ast.Module),
		compiledQueries: make(map[string]rego.PreparedEvalQuery),
		store:           inmem.New(),
	}

	// 初始化缓存
	if config.CacheConfig != nil && config.CacheConfig.Enabled {
		var err error
		client.cache, err = NewCache(config.CacheConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache: %w", err)
		}
	}

	// 如果指定了策略目录，加载所有策略
	if config.EmbeddedConfig.PolicyDir != "" {
		if err := client.loadPoliciesFromDir(ctx, config.EmbeddedConfig.PolicyDir); err != nil {
			return nil, fmt.Errorf("failed to load policies from directory: %w", err)
		}
	}

	return client, nil
}

// loadPoliciesFromDir 从目录加载所有策略
func (c *embeddedClient) loadPoliciesFromDir(ctx context.Context, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理 .rego 文件
		if info.IsDir() || filepath.Ext(path) != ".rego" {
			return nil
		}

		// 读取策略文件
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read policy file %s: %w", path, err)
		}

		// 使用相对路径作为模块名
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			relPath = filepath.Base(path)
		}

		// 加载策略
		if err := c.LoadPolicy(ctx, relPath, string(content)); err != nil {
			return fmt.Errorf("failed to load policy %s: %w", relPath, err)
		}

		return nil
	})
}

// Evaluate 评估策略决策
func (c *embeddedClient) Evaluate(ctx context.Context, path string, input interface{}) (*Result, error) {
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
	dotPath, _ := normalizePath(path)

	// 构建查询 - 使用完整的赋值查询避免未绑定变量错误
	query := fmt.Sprintf("result = data.%s", dotPath)

	// 使用 rego.Module 逐个添加模块
	opts := []func(*rego.Rego){
		rego.Query(query),
		rego.Store(c.store),
		rego.Input(input),
	}

	c.mu.RLock()
	for name, m := range c.modules {
		opts = append(opts, rego.Module(name, m.String()))
	}
	c.mu.RUnlock()

	r := rego.New(opts...)

	// 准备查询
	pq, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}

	// 执行评估
	rs, err := pq.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policy: %w", err)
	}

	// 构建结果
	result := &Result{
		Metrics: &Metrics{},
	}

	if len(rs) > 0 && len(rs[0].Bindings) > 0 {
		// 提取决策结果 - 从绑定中获取 result 变量
		if decision, ok := rs[0].Bindings["result"]; ok {
			result.Decision = decision

			// 如果决策是布尔值，设置 Allowed 字段
			if allowed, ok := decision.(bool); ok {
				result.Allowed = allowed
			}
		} else if len(rs[0].Expressions) > 0 {
			// 回退到表达式结果
			result.Decision = rs[0].Expressions[0].Value

			// 如果决策是布尔值，设置 Allowed 字段
			if allowed, ok := result.Decision.(bool); ok {
				result.Allowed = allowed
			}
		}

		// 注：OPA v1 API 中 Result 不直接暴露 Metrics
		// 如需详细指标，可通过其他方式获取
	}

	// 缓存结果
	if c.cache != nil {
		c.cache.Set(ctx, path, input, result)
	}

	return result, nil
}

// Query 执行 Rego 查询
func (c *embeddedClient) Query(ctx context.Context, query string, input interface{}) (rego.ResultSet, error) {
	// 使用 rego.Module 逐个添加模块
	opts := []func(*rego.Rego){
		rego.Query(query),
		rego.Store(c.store),
		rego.Input(input),
	}

	c.mu.RLock()
	for name, m := range c.modules {
		opts = append(opts, rego.Module(name, m.String()))
	}
	c.mu.RUnlock()

	r := rego.New(opts...)

	pq, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}

	rs, err := pq.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate query: %w", err)
	}

	return rs, nil
}

// PartialEvaluate 执行部分评估
func (c *embeddedClient) PartialEvaluate(ctx context.Context, query string, input interface{}) (*PartialResult, error) {
	// 使用 rego.Module 逐个添加模块
	opts := []func(*rego.Rego){
		rego.Query(query),
		rego.Store(c.store),
	}

	if input != nil {
		opts = append(opts, rego.Input(input))
	}

	c.mu.RLock()
	for name, m := range c.modules {
		opts = append(opts, rego.Module(name, m.String()))
	}
	c.mu.RUnlock()

	r := rego.New(opts...)

	// 执行部分评估
	pq, err := r.Partial(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to partial evaluate: %w", err)
	}

	result := &PartialResult{
		Queries: make([]interface{}, len(pq.Queries)),
		Support: make([]interface{}, len(pq.Support)),
	}

	// 转换查询
	for i, q := range pq.Queries {
		result.Queries[i] = q.String()
	}

	// 转换支持规则
	for i, s := range pq.Support {
		result.Support[i] = s.String()
	}

	return result, nil
}

// LoadPolicy 加载策略
func (c *embeddedClient) LoadPolicy(ctx context.Context, name string, policy string) error {
	// 解析策略
	module, err := ast.ParseModule(name, policy)
	if err != nil {
		return fmt.Errorf("failed to parse policy: %w", err)
	}

	// 编译策略
	compiler := ast.NewCompiler()
	c.mu.RLock()
	allModules := make(map[string]*ast.Module)
	for k, v := range c.modules {
		allModules[k] = v
	}
	c.mu.RUnlock()
	allModules[name] = module

	if compiler.Compile(allModules); compiler.Failed() {
		return fmt.Errorf("failed to compile policy: %v", compiler.Errors)
	}

	// 存储模块
	c.mu.Lock()
	c.modules[name] = module
	// 清除编译缓存（策略已变更）
	c.compiledQueries = make(map[string]rego.PreparedEvalQuery)
	c.mu.Unlock()

	// 清除决策缓存（策略已变更）
	if c.cache != nil {
		c.cache.Clear(ctx)
	}

	return nil
}

// LoadPolicyFromFile 从文件加载策略
func (c *embeddedClient) LoadPolicyFromFile(ctx context.Context, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	// 使用文件名作为模块名
	name := filepath.Base(path)
	return c.LoadPolicy(ctx, name, string(content))
}

// LoadData 加载数据到 OPA store
func (c *embeddedClient) LoadData(ctx context.Context, path string, data interface{}) error {
	if c.store == nil {
		return fmt.Errorf("store not initialized")
	}

	// 规范化路径：确保以 / 开头，但避免重复斜杠
	normalizedPath := path
	if !strings.HasPrefix(normalizedPath, "/") {
		normalizedPath = "/" + normalizedPath
	}

	// 解析路径为 storage.Path
	storagePath, ok := storage.ParsePath(normalizedPath)
	if !ok {
		return fmt.Errorf("invalid storage path: %s", path)
	}

	// 写入数据 - WriteOne 内部会管理事务
	if err := storage.WriteOne(ctx, c.store, storage.AddOp, storagePath, data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	// 清除缓存（数据已变更）
	if c.cache != nil {
		c.cache.Clear(ctx)
	}

	return nil
}

// RemovePolicy 移除策略
func (c *embeddedClient) RemovePolicy(ctx context.Context, name string) error {
	c.mu.Lock()
	delete(c.modules, name)
	c.mu.Unlock()

	// 清除缓存（策略已变更）
	if c.cache != nil {
		c.cache.Clear(ctx)
	}

	return nil
}

// Close 关闭客户端
func (c *embeddedClient) Close(ctx context.Context) error {
	if c.cache != nil {
		return c.cache.Close(ctx)
	}
	return nil
}

// IsEmbedded 是否为嵌入式模式
func (c *embeddedClient) IsEmbedded() bool {
	return true
}
