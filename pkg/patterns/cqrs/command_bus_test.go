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
	"testing"
)

// 测试命令实现
type testCommand struct {
	name string
	data string
}

func (c *testCommand) GetCommandName() string {
	return c.name
}

// 测试命令处理器实现
type testCommandHandler struct {
	commandName string
	handleFunc  func(ctx context.Context, cmd Command) error
}

func (h *testCommandHandler) Handle(ctx context.Context, cmd Command) error {
	if h.handleFunc != nil {
		return h.handleFunc(ctx, cmd)
	}
	return nil
}

func (h *testCommandHandler) GetCommandName() string {
	return h.commandName
}

func TestCommandBus_RegisterAndDispatch(t *testing.T) {
	bus := NewCommandBus()
	executed := false

	handler := &testCommandHandler{
		commandName: "test.command",
		handleFunc: func(ctx context.Context, cmd Command) error {
			executed = true
			return nil
		},
	}

	// 注册处理器
	err := bus.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	// 检查处理器是否存在
	if !bus.HasHandler("test.command") {
		t.Fatal("Handler should be registered")
	}

	// 分发命令
	cmd := &testCommand{name: "test.command", data: "test"}
	err = bus.Dispatch(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	if !executed {
		t.Fatal("Command handler was not executed")
	}
}

func TestCommandBus_DispatchNotFound(t *testing.T) {
	bus := NewCommandBus()
	cmd := &testCommand{name: "unknown.command", data: "test"}

	err := bus.Dispatch(context.Background(), cmd)
	if !errors.Is(err, ErrCommandHandlerNotFound) {
		t.Fatalf("Expected ErrCommandHandlerNotFound, got: %v", err)
	}
}

func TestCommandBus_RegisterDuplicate(t *testing.T) {
	bus := NewCommandBus()

	handler1 := &testCommandHandler{commandName: "test.command"}
	handler2 := &testCommandHandler{commandName: "test.command"}

	err := bus.RegisterHandler(handler1)
	if err != nil {
		t.Fatalf("First RegisterHandler failed: %v", err)
	}

	err = bus.RegisterHandler(handler2)
	if !errors.Is(err, ErrHandlerAlreadyRegistered) {
		t.Fatalf("Expected ErrHandlerAlreadyRegistered, got: %v", err)
	}
}

func TestCommandBus_UnregisterHandler(t *testing.T) {
	bus := NewCommandBus()
	handler := &testCommandHandler{commandName: "test.command"}

	err := bus.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	if !bus.HasHandler("test.command") {
		t.Fatal("Handler should be registered")
	}

	err = bus.UnregisterHandler("test.command")
	if err != nil {
		t.Fatalf("UnregisterHandler failed: %v", err)
	}

	if bus.HasHandler("test.command") {
		t.Fatal("Handler should be unregistered")
	}
}

func TestCommandBus_UnregisterNotFound(t *testing.T) {
	bus := NewCommandBus()

	err := bus.UnregisterHandler("unknown.command")
	if !errors.Is(err, ErrCommandHandlerNotFound) {
		t.Fatalf("Expected ErrCommandHandlerNotFound, got: %v", err)
	}
}

func TestCommandBus_InvalidCommand(t *testing.T) {
	bus := NewCommandBus()

	tests := []struct {
		name    string
		cmd     Command
		wantErr error
	}{
		{
			name:    "nil command",
			cmd:     nil,
			wantErr: ErrInvalidCommand,
		},
		{
			name:    "empty command name",
			cmd:     &testCommand{name: "", data: "test"},
			wantErr: ErrInvalidCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bus.Dispatch(context.Background(), tt.cmd)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Expected error %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestCommandBus_Middleware(t *testing.T) {
	bus := NewCommandBus().(*commandBus)
	executed := false
	middlewareCalled := false

	handler := &testCommandHandler{
		commandName: "test.command",
		handleFunc: func(ctx context.Context, cmd Command) error {
			executed = true
			return nil
		},
	}

	middleware := func(ctx context.Context, cmd Command, next func(context.Context, Command) error) error {
		middlewareCalled = true
		return next(ctx, cmd)
	}

	bus.AddMiddleware(middleware)
	err := bus.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	cmd := &testCommand{name: "test.command", data: "test"}
	err = bus.Dispatch(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	if !executed {
		t.Fatal("Command handler was not executed")
	}

	if !middlewareCalled {
		t.Fatal("Middleware was not called")
	}
}

func TestCommandBus_MiddlewareChain(t *testing.T) {
	bus := NewCommandBus().(*commandBus)
	var order []string

	handler := &testCommandHandler{
		commandName: "test.command",
		handleFunc: func(ctx context.Context, cmd Command) error {
			order = append(order, "handler")
			return nil
		},
	}

	middleware1 := func(ctx context.Context, cmd Command, next func(context.Context, Command) error) error {
		order = append(order, "m1-before")
		err := next(ctx, cmd)
		order = append(order, "m1-after")
		return err
	}

	middleware2 := func(ctx context.Context, cmd Command, next func(context.Context, Command) error) error {
		order = append(order, "m2-before")
		err := next(ctx, cmd)
		order = append(order, "m2-after")
		return err
	}

	bus.AddMiddleware(middleware1)
	bus.AddMiddleware(middleware2)
	err := bus.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	cmd := &testCommand{name: "test.command", data: "test"}
	err = bus.Dispatch(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	if len(order) != len(expected) {
		t.Fatalf("Expected order length %d, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("Expected order[%d] = %s, got %s", i, v, order[i])
		}
	}
}

func TestCommandBus_HandlerError(t *testing.T) {
	bus := NewCommandBus()
	expectedErr := errors.New("handler error")

	handler := &testCommandHandler{
		commandName: "test.command",
		handleFunc: func(ctx context.Context, cmd Command) error {
			return expectedErr
		},
	}

	err := bus.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	cmd := &testCommand{name: "test.command", data: "test"}
	err = bus.Dispatch(context.Background(), cmd)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Expected error %v, got: %v", expectedErr, err)
	}
}
