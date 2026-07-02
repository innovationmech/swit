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

// Package msgtest provides framework-level test utilities for the messaging
// system, including interface-compliant mocks, message/config fixtures, and
// assertion helpers.
//
// It is part of the reusable testing toolkit under pkg/testing and supersedes
// the mock and helper portions of pkg/messaging/testutil. Unlike the legacy
// package, every mock in msgtest is verified against the corresponding
// messaging interface with compile-time assertions, so mocks cannot drift
// from the interfaces they simulate.
//
// Typical usage:
//
//	setup := msgtest.NewBrokerSetup()
//	broker := setup.Broker // *msgtest.MockMessageBroker with default expectations
//
//	msg := msgtest.NewTestMessage("id-1", "user.created", nil)
//	err := publisher.Publish(ctx, msg)
package msgtest
