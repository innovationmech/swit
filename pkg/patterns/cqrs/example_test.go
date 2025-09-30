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

package cqrs_test

import (
	"context"
	"fmt"
	"log"

	"github.com/innovationmech/swit/pkg/patterns/cqrs"
)

// CreateOrderCommand 创建订单命令
type CreateOrderCommand struct {
	OrderID    string
	CustomerID string
	Amount     float64
}

func (c *CreateOrderCommand) GetCommandName() string {
	return "order.create"
}

// CreateOrderHandler 创建订单命令处理器
type CreateOrderHandler struct{}

func (h *CreateOrderHandler) Handle(ctx context.Context, cmd cqrs.Command) error {
	orderCmd := cmd.(*CreateOrderCommand)
	// 执行创建订单的业务逻辑
	fmt.Printf("Creating order: %s for customer: %s, amount: %.2f\n",
		orderCmd.OrderID, orderCmd.CustomerID, orderCmd.Amount)
	return nil
}

func (h *CreateOrderHandler) GetCommandName() string {
	return "order.create"
}

// GetOrderQuery 获取订单查询
type GetOrderQuery struct {
	OrderID string
}

func (q *GetOrderQuery) GetQueryName() string {
	return "order.get"
}

// OrderDTO 订单数据传输对象
type OrderDTO struct {
	OrderID    string
	CustomerID string
	Amount     float64
	Status     string
}

// GetOrderHandler 获取订单查询处理器
type GetOrderHandler struct{}

func (h *GetOrderHandler) Handle(ctx context.Context, query cqrs.Query) (interface{}, error) {
	orderQuery := query.(*GetOrderQuery)
	// 从读模型中查询订单
	order := &OrderDTO{
		OrderID:    orderQuery.OrderID,
		CustomerID: "customer-123",
		Amount:     99.99,
		Status:     "pending",
	}
	return order, nil
}

func (h *GetOrderHandler) GetQueryName() string {
	return "order.get"
}

func ExampleCommandBus() {
	// 创建命令总线
	commandBus := cqrs.NewCommandBus()

	// 注册命令处理器
	createOrderHandler := &CreateOrderHandler{}
	if err := commandBus.RegisterHandler(createOrderHandler); err != nil {
		log.Fatal(err)
	}

	// 创建并分发命令
	cmd := &CreateOrderCommand{
		OrderID:    "order-001",
		CustomerID: "customer-123",
		Amount:     99.99,
	}

	if err := commandBus.Dispatch(context.Background(), cmd); err != nil {
		log.Fatal(err)
	}

	// Output:
	// Creating order: order-001 for customer: customer-123, amount: 99.99
}

func ExampleQueryBus() {
	// 创建查询总线
	queryBus := cqrs.NewQueryBus()

	// 注册查询处理器
	getOrderHandler := &GetOrderHandler{}
	if err := queryBus.RegisterHandler(getOrderHandler); err != nil {
		log.Fatal(err)
	}

	// 创建并分发查询
	query := &GetOrderQuery{
		OrderID: "order-001",
	}

	result, err := queryBus.Dispatch(context.Background(), query)
	if err != nil {
		log.Fatal(err)
	}

	order := result.(*OrderDTO)
	fmt.Printf("Order: %s, Customer: %s, Amount: %.2f, Status: %s\n",
		order.OrderID, order.CustomerID, order.Amount, order.Status)

	// Output:
	// Order: order-001, Customer: customer-123, Amount: 99.99, Status: pending
}

// Note: Middleware support example is commented out because AddMiddleware
// is not exposed in the public interface. In production use, you would
// extend the CommandBus interface to include AddMiddleware method.
//
// Example usage would be:
// type CommandBusWithMiddleware interface {
//     cqrs.CommandBus
//     AddMiddleware(middleware cqrs.CommandMiddleware)
// }
