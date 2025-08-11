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
	"testing"
)

func TestInitLogger(t *testing.T) {
	originalLogger := Logger

	defer func() {
		Logger = originalLogger
	}()

	Logger = nil

	InitLogger()

	if Logger == nil {
		t.Error("InitLogger() failed: Logger is nil after initialization")
	}

	if Logger.Core() == nil {
		t.Error("InitLogger() failed: Logger core is nil")
	}
}

func TestInitLoggerMultipleCalls(t *testing.T) {
	originalLogger := Logger

	defer func() {
		Logger = originalLogger
	}()

	ResetLogger() // Use proper reset function

	InitLogger()
	firstLogger := Logger

	InitLogger()
	secondLogger := Logger

	if firstLogger == nil || secondLogger == nil {
		t.Error("InitLogger() failed: Logger is nil after multiple calls")
	}

	// Multiple calls should return the same logger instance (race-safe behavior)
	if firstLogger != secondLogger {
		t.Error("InitLogger() should return the same logger instance on multiple calls for thread safety")
	}
}

func TestLoggerGlobalVariable(t *testing.T) {
	originalLogger := Logger

	defer func() {
		Logger = originalLogger
	}()

	ResetLogger() // Use proper reset function

	if Logger != nil {
		t.Error("Logger should be nil initially")
	}

	InitLogger()

	if Logger == nil {
		t.Error("Logger should not be nil after InitLogger()")
	}

	Logger.Info("test message")

	if Logger == nil {
		t.Error("Logger should still be available after logging")
	}
}

func TestLoggerProductionConfig(t *testing.T) {
	originalLogger := Logger

	defer func() {
		Logger = originalLogger
	}()

	Logger = nil

	InitLogger()

	if Logger == nil {
		t.Fatal("Logger should not be nil after InitLogger()")
	}

	Logger.Info("test info message")
	Logger.Error("test error message")
	Logger.Debug("test debug message")
	Logger.Warn("test warning message")
}

func BenchmarkInitLogger(b *testing.B) {
	originalLogger := Logger

	defer func() {
		Logger = originalLogger
	}()

	for i := 0; i < b.N; i++ {
		Logger = nil
		InitLogger()
	}
}
