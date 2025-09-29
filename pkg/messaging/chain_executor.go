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
)

// ChainExecutor defines how a HandlerChain is executed.
//
// The base semantics are:
// - Deterministic ordering (chain order)
// - Short-circuit on first error when ShortCircuit is true
// - ErrorPolicy decides how errors are surfaced/collected
// - Context must flow through steps
type ChainExecutor interface {
	// Execute runs the chain against the provided message.
	// The executor should honor ordering and short-circuit when configured.
	Execute(ctx context.Context, chain HandlerChain, message *Message) error

	// WithOptions returns a shallow-copied executor with provided options.
	WithOptions(opts ChainExecutorOptions) ChainExecutor
}

// ChainExecutorOptions controls execution behavior.
type ChainExecutorOptions struct {
	// ShortCircuit stops on first error if true.
	ShortCircuit bool

	// ErrorPolicy controls error surfacing.
	ErrorPolicy ErrorPolicy
}

// BaseChainExecutor provides a default implementation of ChainExecutor.
// It executes handlers sequentially, preserves order, and adheres to options.
type BaseChainExecutor struct {
	opts ChainExecutorOptions
}

// NewBaseChainExecutor creates a new BaseChainExecutor with defaults.
func NewBaseChainExecutor() *BaseChainExecutor {
	return &BaseChainExecutor{
		opts: ChainExecutorOptions{ShortCircuit: true, ErrorPolicy: ErrorPolicyStopOnFirst},
	}
}

// WithOptions returns a shallow-copied executor with the given options.
func (e *BaseChainExecutor) WithOptions(opts ChainExecutorOptions) ChainExecutor {
	cp := *e
	cp.opts = opts
	return &cp
}

// Execute runs the chain steps in order with ordering and error short-circuit.
func (e *BaseChainExecutor) Execute(ctx context.Context, chain HandlerChain, message *Message) error {
	if chain == nil || chain.Len() == 0 {
		return nil
	}

	var collected []error
	for _, step := range chain.Steps() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := step.Handle(ctx, message); err != nil {
			switch e.opts.ErrorPolicy {
			case ErrorPolicyIgnoreErrors:
				// swallow
			case ErrorPolicyCollectAll:
				collected = append(collected, err)
			default: // ErrorPolicyStopOnFirst
				if e.opts.ShortCircuit {
					return err
				}
				collected = append(collected, err)
			}
		}
	}

	if len(collected) == 0 {
		return nil
	}
	// For simplicity: return first error when not StopOnFirst collected earlier
	return collected[0]
}
