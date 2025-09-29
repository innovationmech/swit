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

import "sync"

// HandlerChain defines an ordered collection of MessageHandler steps.
//
// The chain guarantees iteration order consistency (insertion order) and is
// intended to be executed by a ChainExecutor with deterministic semantics.
// It is separate from MiddlewareChain to represent business handler steps
// rather than cross-cutting middleware concerns.
type HandlerChain interface {
	// Add appends one or more handlers to the end of the chain.
	Add(steps ...MessageHandler)

	// Prepend inserts one or more handlers at the beginning of the chain,
	// preserving the relative order of the inserted steps.
	Prepend(steps ...MessageHandler)

	// Steps returns a snapshot copy of the handlers in order.
	Steps() []MessageHandler

	// Len returns the number of handlers in the chain.
	Len() int
}

// DefaultHandlerChain is a threadsafe implementation of HandlerChain.
type DefaultHandlerChain struct {
	mu    sync.RWMutex
	steps []MessageHandler
}

// NewHandlerChain creates a new DefaultHandlerChain with optional initial steps.
func NewHandlerChain(steps ...MessageHandler) *DefaultHandlerChain {
	copied := make([]MessageHandler, len(steps))
	copy(copied, steps)
	return &DefaultHandlerChain{steps: copied}
}

// Add appends one or more handlers to the end of the chain.
func (c *DefaultHandlerChain) Add(steps ...MessageHandler) {
	c.mu.Lock()
	c.steps = append(c.steps, steps...)
	c.mu.Unlock()
}

// Prepend inserts one or more handlers at the beginning of the chain.
func (c *DefaultHandlerChain) Prepend(steps ...MessageHandler) {
	c.mu.Lock()
	c.steps = append(append([]MessageHandler(nil), steps...), c.steps...)
	c.mu.Unlock()
}

// Steps returns a snapshot copy of the handlers in order.
func (c *DefaultHandlerChain) Steps() []MessageHandler {
	c.mu.RLock()
	out := make([]MessageHandler, len(c.steps))
	copy(out, c.steps)
	c.mu.RUnlock()
	return out
}

// Len returns the number of handlers in the chain.
func (c *DefaultHandlerChain) Len() int {
	c.mu.RLock()
	n := len(c.steps)
	c.mu.RUnlock()
	return n
}
