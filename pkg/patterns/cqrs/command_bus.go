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

// commandBus 命令总线的内存实现
type commandBus struct {
	mu          sync.RWMutex
	handlers    map[string]CommandHandler
	middlewares []CommandMiddleware
}

// NewCommandBus 创建一个新的命令总线
func NewCommandBus() CommandBus {
	return &commandBus{
		handlers:    make(map[string]CommandHandler),
		middlewares: make([]CommandMiddleware, 0),
	}
}

// Dispatch 分发命令到对应的处理器
func (b *commandBus) Dispatch(ctx context.Context, cmd Command) error {
	if cmd == nil {
		return ErrInvalidCommand
	}

	commandName := cmd.GetCommandName()
	if commandName == "" {
		return fmt.Errorf("%w: command name is empty", ErrInvalidCommand)
	}

	b.mu.RLock()
	handler, ok := b.handlers[commandName]
	b.mu.RUnlock()

	if !ok {
		return fmt.Errorf("%w: %s", ErrCommandHandlerNotFound, commandName)
	}

	// 执行中间件链
	return b.executeMiddlewareChain(ctx, cmd, handler)
}

// executeMiddlewareChain 执行中间件链
func (b *commandBus) executeMiddlewareChain(ctx context.Context, cmd Command, handler CommandHandler) error {
	// 构建处理器链（从后往前）
	next := func(ctx context.Context, cmd Command) error {
		return handler.Handle(ctx, cmd)
	}

	// 应用中间件（从后往前）
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		middleware := b.middlewares[i]
		nextFunc := next
		next = func(ctx context.Context, cmd Command) error {
			return middleware(ctx, cmd, nextFunc)
		}
	}

	return next(ctx, cmd)
}

// RegisterHandler 注册命令处理器
func (b *commandBus) RegisterHandler(handler CommandHandler) error {
	if handler == nil {
		return ErrInvalidCommand
	}

	commandName := handler.GetCommandName()
	if commandName == "" {
		return fmt.Errorf("%w: command name is empty", ErrInvalidCommand)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.handlers[commandName]; exists {
		return fmt.Errorf("%w: %s", ErrHandlerAlreadyRegistered, commandName)
	}

	b.handlers[commandName] = handler
	return nil
}

// UnregisterHandler 注销命令处理器
func (b *commandBus) UnregisterHandler(commandName string) error {
	if commandName == "" {
		return fmt.Errorf("%w: command name is empty", ErrInvalidCommand)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.handlers[commandName]; !exists {
		return fmt.Errorf("%w: %s", ErrCommandHandlerNotFound, commandName)
	}

	delete(b.handlers, commandName)
	return nil
}

// HasHandler 检查是否有处理器处理指定的命令
func (b *commandBus) HasHandler(commandName string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	_, exists := b.handlers[commandName]
	return exists
}

// AddMiddleware 添加命令中间件
func (b *commandBus) AddMiddleware(middleware CommandMiddleware) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.middlewares = append(b.middlewares, middleware)
}
