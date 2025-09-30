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

// 测试查询实现
type testQuery struct {
	name string
	id   string
}

func (q *testQuery) GetQueryName() string {
	return q.name
}

// 测试查询处理器实现
type testQueryHandler struct {
	queryName  string
	handleFunc func(ctx context.Context, query Query) (interface{}, error)
}

func (h *testQueryHandler) Handle(ctx context.Context, query Query) (interface{}, error) {
	if h.handleFunc != nil {
		return h.handleFunc(ctx, query)
	}
	return nil, nil
}

func (h *testQueryHandler) GetQueryName() string {
	return h.queryName
}

func TestQueryBus_RegisterAndDispatch(t *testing.T) {
	bus := NewQueryBus()
	expectedResult := "test result"

	handler := &testQueryHandler{
		queryName: "test.query",
		handleFunc: func(ctx context.Context, query Query) (interface{}, error) {
			return expectedResult, nil
		},
	}

	// 注册处理器
	err := bus.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	// 检查处理器是否存在
	if !bus.HasHandler("test.query") {
		t.Fatal("Handler should be registered")
	}

	// 分发查询
	query := &testQuery{name: "test.query", id: "123"}
	result, err := bus.Dispatch(context.Background(), query)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	if result != expectedResult {
		t.Fatalf("Expected result %v, got: %v", expectedResult, result)
	}
}

func TestQueryBus_DispatchNotFound(t *testing.T) {
	bus := NewQueryBus()
	query := &testQuery{name: "unknown.query", id: "123"}

	_, err := bus.Dispatch(context.Background(), query)
	if !errors.Is(err, ErrQueryHandlerNotFound) {
		t.Fatalf("Expected ErrQueryHandlerNotFound, got: %v", err)
	}
}

func TestQueryBus_RegisterDuplicate(t *testing.T) {
	bus := NewQueryBus()

	handler1 := &testQueryHandler{queryName: "test.query"}
	handler2 := &testQueryHandler{queryName: "test.query"}

	err := bus.RegisterHandler(handler1)
	if err != nil {
		t.Fatalf("First RegisterHandler failed: %v", err)
	}

	err = bus.RegisterHandler(handler2)
	if !errors.Is(err, ErrHandlerAlreadyRegistered) {
		t.Fatalf("Expected ErrHandlerAlreadyRegistered, got: %v", err)
	}
}

func TestQueryBus_UnregisterHandler(t *testing.T) {
	bus := NewQueryBus()
	handler := &testQueryHandler{queryName: "test.query"}

	err := bus.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	if !bus.HasHandler("test.query") {
		t.Fatal("Handler should be registered")
	}

	err = bus.UnregisterHandler("test.query")
	if err != nil {
		t.Fatalf("UnregisterHandler failed: %v", err)
	}

	if bus.HasHandler("test.query") {
		t.Fatal("Handler should be unregistered")
	}
}

func TestQueryBus_UnregisterNotFound(t *testing.T) {
	bus := NewQueryBus()

	err := bus.UnregisterHandler("unknown.query")
	if !errors.Is(err, ErrQueryHandlerNotFound) {
		t.Fatalf("Expected ErrQueryHandlerNotFound, got: %v", err)
	}
}

func TestQueryBus_InvalidQuery(t *testing.T) {
	bus := NewQueryBus()

	tests := []struct {
		name    string
		query   Query
		wantErr error
	}{
		{
			name:    "nil query",
			query:   nil,
			wantErr: ErrInvalidQuery,
		},
		{
			name:    "empty query name",
			query:   &testQuery{name: "", id: "123"},
			wantErr: ErrInvalidQuery,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := bus.Dispatch(context.Background(), tt.query)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Expected error %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestQueryBus_Middleware(t *testing.T) {
	bus := NewQueryBus().(*queryBus)
	middlewareCalled := false
	expectedResult := "test result"

	handler := &testQueryHandler{
		queryName: "test.query",
		handleFunc: func(ctx context.Context, query Query) (interface{}, error) {
			return expectedResult, nil
		},
	}

	middleware := func(ctx context.Context, query Query, next func(context.Context, Query) (interface{}, error)) (interface{}, error) {
		middlewareCalled = true
		return next(ctx, query)
	}

	bus.AddMiddleware(middleware)
	err := bus.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	query := &testQuery{name: "test.query", id: "123"}
	result, err := bus.Dispatch(context.Background(), query)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	if result != expectedResult {
		t.Fatalf("Expected result %v, got: %v", expectedResult, result)
	}

	if !middlewareCalled {
		t.Fatal("Middleware was not called")
	}
}

func TestQueryBus_MiddlewareChain(t *testing.T) {
	bus := NewQueryBus().(*queryBus)
	var order []string
	expectedResult := "test result"

	handler := &testQueryHandler{
		queryName: "test.query",
		handleFunc: func(ctx context.Context, query Query) (interface{}, error) {
			order = append(order, "handler")
			return expectedResult, nil
		},
	}

	middleware1 := func(ctx context.Context, query Query, next func(context.Context, Query) (interface{}, error)) (interface{}, error) {
		order = append(order, "m1-before")
		result, err := next(ctx, query)
		order = append(order, "m1-after")
		return result, err
	}

	middleware2 := func(ctx context.Context, query Query, next func(context.Context, Query) (interface{}, error)) (interface{}, error) {
		order = append(order, "m2-before")
		result, err := next(ctx, query)
		order = append(order, "m2-after")
		return result, err
	}

	bus.AddMiddleware(middleware1)
	bus.AddMiddleware(middleware2)
	err := bus.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	query := &testQuery{name: "test.query", id: "123"}
	result, err := bus.Dispatch(context.Background(), query)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	if result != expectedResult {
		t.Fatalf("Expected result %v, got: %v", expectedResult, result)
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

func TestQueryBus_HandlerError(t *testing.T) {
	bus := NewQueryBus()
	expectedErr := errors.New("handler error")

	handler := &testQueryHandler{
		queryName: "test.query",
		handleFunc: func(ctx context.Context, query Query) (interface{}, error) {
			return nil, expectedErr
		},
	}

	err := bus.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	query := &testQuery{name: "test.query", id: "123"}
	_, err = bus.Dispatch(context.Background(), query)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Expected error %v, got: %v", expectedErr, err)
	}
}
