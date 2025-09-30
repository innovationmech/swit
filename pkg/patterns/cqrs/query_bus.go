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

package cqrs

import (
	"context"
	"fmt"
	"sync"
)

// queryBus 查询总线的内存实现
type queryBus struct {
	mu          sync.RWMutex
	handlers    map[string]QueryHandler
	middlewares []QueryMiddleware
}

// NewQueryBus 创建一个新的查询总线
func NewQueryBus() QueryBus {
	return &queryBus{
		handlers:    make(map[string]QueryHandler),
		middlewares: make([]QueryMiddleware, 0),
	}
}

// Dispatch 分发查询到对应的处理器
func (b *queryBus) Dispatch(ctx context.Context, query Query) (interface{}, error) {
	if query == nil {
		return nil, ErrInvalidQuery
	}

	queryName := query.GetQueryName()
	if queryName == "" {
		return nil, fmt.Errorf("%w: query name is empty", ErrInvalidQuery)
	}

	b.mu.RLock()
	handler, ok := b.handlers[queryName]
	b.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrQueryHandlerNotFound, queryName)
	}

	// 执行中间件链
	return b.executeMiddlewareChain(ctx, query, handler)
}

// executeMiddlewareChain 执行中间件链
func (b *queryBus) executeMiddlewareChain(ctx context.Context, query Query, handler QueryHandler) (interface{}, error) {
	// 构建处理器链（从后往前）
	next := func(ctx context.Context, query Query) (interface{}, error) {
		return handler.Handle(ctx, query)
	}

	// 应用中间件（从后往前）
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		middleware := b.middlewares[i]
		nextFunc := next
		next = func(ctx context.Context, query Query) (interface{}, error) {
			return middleware(ctx, query, nextFunc)
		}
	}

	return next(ctx, query)
}

// RegisterHandler 注册查询处理器
func (b *queryBus) RegisterHandler(handler QueryHandler) error {
	if handler == nil {
		return ErrInvalidQuery
	}

	queryName := handler.GetQueryName()
	if queryName == "" {
		return fmt.Errorf("%w: query name is empty", ErrInvalidQuery)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.handlers[queryName]; exists {
		return fmt.Errorf("%w: %s", ErrHandlerAlreadyRegistered, queryName)
	}

	b.handlers[queryName] = handler
	return nil
}

// UnregisterHandler 注销查询处理器
func (b *queryBus) UnregisterHandler(queryName string) error {
	if queryName == "" {
		return fmt.Errorf("%w: query name is empty", ErrInvalidQuery)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.handlers[queryName]; !exists {
		return fmt.Errorf("%w: %s", ErrQueryHandlerNotFound, queryName)
	}

	delete(b.handlers, queryName)
	return nil
}

// HasHandler 检查是否有处理器处理指定的查询
func (b *queryBus) HasHandler(queryName string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	_, exists := b.handlers[queryName]
	return exists
}

// AddMiddleware 添加查询中间件
func (b *queryBus) AddMiddleware(middleware QueryMiddleware) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.middlewares = append(b.middlewares, middleware)
}
