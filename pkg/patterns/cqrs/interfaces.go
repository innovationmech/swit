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
	"errors"
)

var (
	// ErrCommandHandlerNotFound 未找到命令处理器
	ErrCommandHandlerNotFound = errors.New("command handler not found")
	// ErrQueryHandlerNotFound 未找到查询处理器
	ErrQueryHandlerNotFound = errors.New("query handler not found")
	// ErrInvalidCommand 无效的命令
	ErrInvalidCommand = errors.New("invalid command")
	// ErrInvalidQuery 无效的查询
	ErrInvalidQuery = errors.New("invalid query")
	// ErrHandlerAlreadyRegistered 处理器已注册
	ErrHandlerAlreadyRegistered = errors.New("handler already registered")
)

// Command 定义命令接口
//
// 命令表示系统中的写操作，用于改变系统状态
type Command interface {
	// GetCommandName 返回命令名称，用于路由到对应的处理器
	GetCommandName() string
}

// Query 定义查询接口
//
// 查询表示系统中的读操作，不应改变系统状态
type Query interface {
	// GetQueryName 返回查询名称，用于路由到对应的处理器
	GetQueryName() string
}

// CommandHandler 定义命令处理器接口
//
// 命令处理器负责处理特定类型的命令并执行相应的业务逻辑
type CommandHandler interface {
	// Handle 处理命令
	// 返回错误表示命令处理失败
	Handle(ctx context.Context, cmd Command) error

	// GetCommandName 返回该处理器能够处理的命令名称
	GetCommandName() string
}

// QueryHandler 定义查询处理器接口
//
// 查询处理器负责处理特定类型的查询并返回结果
type QueryHandler interface {
	// Handle 处理查询
	// 返回查询结果和可能的错误
	Handle(ctx context.Context, query Query) (interface{}, error)

	// GetQueryName 返回该处理器能够处理的查询名称
	GetQueryName() string
}

// CommandBus 定义命令总线接口
//
// 命令总线负责将命令路由到对应的处理器
type CommandBus interface {
	// Dispatch 分发命令到对应的处理器
	Dispatch(ctx context.Context, cmd Command) error

	// RegisterHandler 注册命令处理器
	RegisterHandler(handler CommandHandler) error

	// UnregisterHandler 注销命令处理器
	UnregisterHandler(commandName string) error

	// HasHandler 检查是否有处理器处理指定的命令
	HasHandler(commandName string) bool
}

// QueryBus 定义查询总线接口
//
// 查询总线负责将查询路由到对应的处理器
type QueryBus interface {
	// Dispatch 分发查询到对应的处理器
	Dispatch(ctx context.Context, query Query) (interface{}, error)

	// RegisterHandler 注册查询处理器
	RegisterHandler(handler QueryHandler) error

	// UnregisterHandler 注销查询处理器
	UnregisterHandler(queryName string) error

	// HasHandler 检查是否有处理器处理指定的查询
	HasHandler(queryName string) bool
}

// Middleware 定义中间件接口
//
// 中间件可以在命令或查询处理前后执行额外的逻辑（如日志、验证、权限检查等）
type Middleware interface {
	// ExecuteCommand 在命令处理前后执行
	ExecuteCommand(ctx context.Context, cmd Command, next func(context.Context, Command) error) error

	// ExecuteQuery 在查询处理前后执行
	ExecuteQuery(ctx context.Context, query Query, next func(context.Context, Query) (interface{}, error)) (interface{}, error)
}

// CommandMiddleware 定义命令中间件接口
type CommandMiddleware func(ctx context.Context, cmd Command, next func(context.Context, Command) error) error

// QueryMiddleware 定义查询中间件接口
type QueryMiddleware func(ctx context.Context, query Query, next func(context.Context, Query) (interface{}, error)) (interface{}, error)
