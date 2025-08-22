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

package logger

import (
	"go.uber.org/zap"
	"sync"
)

var (
	// Logger is the global logger for the application.
	Logger *zap.Logger
	// mu protects Logger from concurrent access
	mu sync.RWMutex
	// initialized tracks whether logger has been initialized
	initialized bool
)

// InitLogger initializes the global logger safely to prevent race conditions.
func InitLogger() {
	mu.Lock()
	defer mu.Unlock()

	// Only initialize if not already done
	if !initialized || Logger == nil {
		var err error
		Logger, err = zap.NewProduction()
		if err != nil {
			panic(err)
		}
		initialized = true
	}
}

// GetLogger returns the global logger, initializing it if necessary.
func GetLogger() *zap.Logger {
	mu.RLock()
	if initialized && Logger != nil {
		defer mu.RUnlock()
		return Logger
	}
	mu.RUnlock()

	// Initialize logger if not done yet
	InitLogger()
	
	mu.RLock()
	defer mu.RUnlock()
	return Logger
}

// ResetLogger resets the logger for testing purposes.
// This should only be used in tests.
func ResetLogger() {
	mu.Lock()
	defer mu.Unlock()

	if Logger != nil {
		Logger.Sync() // Flush any pending log entries
	}
	Logger = nil
	initialized = false
}
