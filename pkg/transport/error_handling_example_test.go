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
	"errors"
	"fmt"
	"log"
	"time"
)

// ExampleManager_Stop demonstrates the improved error handling in Stop method
func ExampleManager_Stop() {
	// Create transport manager
	manager := NewManager()

	// Create some mock transports - one will fail during stop
	httpTransport := NewMockTransport("http", ":8080")
	grpcTransport := NewMockTransport("grpc", ":9090")

	// Simulate stop failure for HTTP transport
	httpTransport.SetStopError(errors.New("connection cleanup failed"))

	// Register transports
	manager.Register(httpTransport)
	manager.Register(grpcTransport)

	// Start all transports
	ctx := context.Background()
	if err := manager.Start(ctx); err != nil {
		log.Printf("Failed to start transports: %v", err)
		return
	}

	// Stop all transports - this will now collect all errors
	err := manager.Stop(5 * time.Second)

	if err != nil {
		// Check if it's a transport stop error
		if IsStopError(err) {
			// Extract individual transport errors
			stopErrors := ExtractStopErrors(err)
			fmt.Printf("Transport stop completed with %d errors:\n", len(stopErrors))

			for _, stopErr := range stopErrors {
				fmt.Printf("- %s transport failed: %v\n", stopErr.TransportName, stopErr.Err)
			}

			// You can also check specific transports
			if multiErr, ok := err.(*MultiError); ok {
				if httpErr := multiErr.GetErrorByTransport("http"); httpErr != nil {
					fmt.Printf("HTTP transport had issues: %v\n", httpErr.Err)
				}
				if grpcErr := multiErr.GetErrorByTransport("grpc"); grpcErr == nil {
					fmt.Printf("GRPC transport stopped successfully\n")
				}
			}
		} else {
			fmt.Printf("Non-transport error occurred: %v\n", err)
		}
	} else {
		fmt.Println("All transports stopped successfully")
	}

	// Output:
	// Transport stop completed with 1 errors:
	// - http transport failed: connection cleanup failed
	// HTTP transport had issues: connection cleanup failed
	// GRPC transport stopped successfully
}

// ExampleMultiError demonstrates working with multiple transport errors
func ExampleMultiError() {
	// Create a MultiError with several transport errors
	multiErr := &MultiError{
		Errors: []TransportError{
			{
				TransportName: "http",
				Err:           errors.New("port already in use"),
			},
			{
				TransportName: "grpc",
				Err:           errors.New("TLS certificate invalid"),
			},
			{
				TransportName: "websocket",
				Err:           errors.New("websocket upgrade failed"),
			},
		},
	}

	// Print overall error message
	fmt.Printf("Combined error: %v\n", multiErr)

	// Check if there are any errors
	if multiErr.HasErrors() {
		fmt.Printf("Found %d transport errors\n", len(multiErr.Errors))
	}

	// Access specific transport errors
	if httpErr := multiErr.GetErrorByTransport("http"); httpErr != nil {
		fmt.Printf("HTTP transport error: %v\n", httpErr.Err)
	}

	// Try to unwrap (only works for single errors)
	if unwrapped := multiErr.Unwrap(); unwrapped != nil {
		fmt.Printf("Unwrapped error: %v\n", unwrapped)
	} else {
		fmt.Println("Cannot unwrap multiple errors")
	}

	// Output:
	// Combined error: multiple transport errors: transport 'http': port already in use; transport 'grpc': TLS certificate invalid; transport 'websocket': websocket upgrade failed
	// Found 3 transport errors
	// HTTP transport error: port already in use
	// Cannot unwrap multiple errors
}

// ExampleTransportError demonstrates working with individual transport errors
func ExampleTransportError() {
	// Create a single transport error
	transportErr := &TransportError{
		TransportName: "grpc",
		Err:           errors.New("failed to bind to port 9090"),
	}

	fmt.Printf("Transport error: %v\n", transportErr)
	fmt.Printf("Failed transport: %s\n", transportErr.TransportName)
	fmt.Printf("Underlying error: %v\n", transportErr.Err)

	// Output:
	// Transport error: transport 'grpc': failed to bind to port 9090
	// Failed transport: grpc
	// Underlying error: failed to bind to port 9090
}
