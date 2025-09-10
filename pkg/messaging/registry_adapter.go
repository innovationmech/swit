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

package messaging

import (
	"context"

	"github.com/innovationmech/swit/pkg/server"
)

// Ensure the adapter implements the server-side interface at compile time.
var _ server.EventHandlerRegistry = (*serverEventHandlerRegistryAdapter)(nil)

// serverEventHandlerRegistryAdapter adapts the messaging EventHandlerRegistry
// to the server.EventHandlerRegistry interface without creating import cycles.
type serverEventHandlerRegistryAdapter struct {
	inner *EventHandlerRegistry
}

// NewServerEventHandlerRegistryAdapter creates a server.EventHandlerRegistry backed by
// the messaging coordinator. It validates and registers handlers via the inner registry.
func NewServerEventHandlerRegistryAdapter(coordinator MessagingCoordinator) server.EventHandlerRegistry {
	return &serverEventHandlerRegistryAdapter{inner: NewEventHandlerRegistry(coordinator)}
}

// RegisterEventHandler wraps a server.EventHandler as a messaging.EventHandler and registers it.
func (a *serverEventHandlerRegistryAdapter) RegisterEventHandler(handler server.EventHandler) error {
	if handler == nil {
		return a.inner.RegisterEventHandler(nil)
	}
	wrapped := &serverToMessagingEventHandler{delegate: handler}
	return a.inner.RegisterEventHandlerWithValidation(wrapped)
}

// UnregisterEventHandler delegates to the inner registry.
func (a *serverEventHandlerRegistryAdapter) UnregisterEventHandler(handlerID string) error {
	return a.inner.UnregisterEventHandler(handlerID)
}

// GetRegisteredHandlers delegates to the inner registry.
func (a *serverEventHandlerRegistryAdapter) GetRegisteredHandlers() []string {
	return a.inner.GetRegisteredHandlers()
}

// serverToMessagingEventHandler adapts a server.EventHandler to messaging.EventHandler.
type serverToMessagingEventHandler struct {
	delegate server.EventHandler
}

func (h *serverToMessagingEventHandler) GetHandlerID() string { return h.delegate.GetHandlerID() }
func (h *serverToMessagingEventHandler) GetTopics() []string  { return h.delegate.GetTopics() }
func (h *serverToMessagingEventHandler) GetBrokerRequirement() string {
	return h.delegate.GetBrokerRequirement()
}

func (h *serverToMessagingEventHandler) Initialize(ctx context.Context) error {
	return h.delegate.Initialize(ctx)
}

func (h *serverToMessagingEventHandler) Shutdown(ctx context.Context) error {
	return h.delegate.Shutdown(ctx)
}

// Handle adapts the messaging message to the server handler that accepts interface{}.
func (h *serverToMessagingEventHandler) Handle(ctx context.Context, message *Message) error {
	return h.delegate.Handle(ctx, message)
}

// OnError adapts the server handler's generic return value to a messaging ErrorAction.
// If the return value is not an ErrorAction (or is nil), default to ErrorActionRetry.
func (h *serverToMessagingEventHandler) OnError(ctx context.Context, message *Message, err error) ErrorAction {
	v := h.delegate.OnError(ctx, message, err)
	if act, ok := v.(ErrorAction); ok {
		return act
	}
	return ErrorActionRetry
}
