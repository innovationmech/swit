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

package transport

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestHTTPNetworkService_SerialStartStopWithWaitReady tests the Start/Stop functionality
// with WaitReady in a serial manner to avoid race conditions.
func TestHTTPNetworkService_SerialStartStopWithWaitReady(t *testing.T) {
	config := &HTTPTransportConfig{
		Address:     ":0",
		EnableReady: true,
	}
	transport := NewHTTPNetworkServiceWithConfig(config)

	ctx := context.Background()
	const numIterations = 10

	// Perform multiple start/stop cycles serially
	for i := 0; i < numIterations; i++ {
		// Start the service
		err := transport.Start(ctx)
		require.NoError(t, err, "iteration %d: start failed", i)

		// Wait for ready with timeout to avoid hanging
		select {
		case <-transport.WaitReady():
			// Service is ready
		case <-time.After(100 * time.Millisecond):
			t.Errorf("iteration %d: WaitReady timed out", i)
			return
		}

		// Stop the service
		err = transport.Stop(ctx)
		require.NoError(t, err, "iteration %d: stop failed", i)

		// Small delay between iterations
		time.Sleep(time.Millisecond)
	}
}

// TestHTTPNetworkService_WaitReadyAfterStop tests that WaitReady returns
// a closed channel when called after the service has been stopped.
func TestHTTPNetworkService_WaitReadyAfterStop(t *testing.T) {
	config := &HTTPTransportConfig{
		Address:     ":0",
		EnableReady: true,
	}
	transport := NewHTTPNetworkServiceWithConfig(config)

	ctx := context.Background()

	// Start the service
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Wait for ready
	<-transport.WaitReady()

	// Stop the service
	err = transport.Stop(ctx)
	require.NoError(t, err)

	// WaitReady should return a closed channel immediately
	select {
	case <-transport.WaitReady():
		// Expected: should return immediately with a closed channel
	case <-time.After(10 * time.Millisecond):
		t.Error("WaitReady should return immediately after stop")
	}
}

// TestHTTPNetworkService_WaitReadyBeforeStart tests that WaitReady returns
// a closed channel when called before the service has been started.
func TestHTTPNetworkService_WaitReadyBeforeStart(t *testing.T) {
	config := &HTTPTransportConfig{
		Address:     ":0",
		EnableReady: true,
	}
	transport := NewHTTPNetworkServiceWithConfig(config)

	// WaitReady should return a closed channel immediately
	select {
	case <-transport.WaitReady():
		// Expected: should return immediately with a closed channel
	case <-time.After(10 * time.Millisecond):
		t.Error("WaitReady should return immediately before start")
	}
}
