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

package testing

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// ==========================
// TestLogger
// ==========================

// LogLevel represents the severity level of a log entry.
type LogLevel int

const (
	// LogLevelDebug is for detailed debugging information.
	LogLevelDebug LogLevel = iota
	// LogLevelInfo is for informational messages.
	LogLevelInfo
	// LogLevelWarn is for warning messages.
	LogLevelWarn
	// LogLevelError is for error messages.
	LogLevelError
)

// String returns the string representation of a log level.
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a single log entry.
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	Context   map[string]interface{}
}

// TestLogger provides a logger that collects logs for testing purposes.
type TestLogger struct {
	mu      sync.RWMutex
	entries []LogEntry
	level   LogLevel
	output  io.Writer
	prefix  string
}

// NewTestLogger creates a new test logger.
func NewTestLogger() *TestLogger {
	return &TestLogger{
		entries: make([]LogEntry, 0),
		level:   LogLevelInfo,
		output:  nil, // No output by default
		prefix:  "",
	}
}

// NewTestLoggerWithOutput creates a new test logger with output.
func NewTestLoggerWithOutput(output io.Writer) *TestLogger {
	return &TestLogger{
		entries: make([]LogEntry, 0),
		level:   LogLevelInfo,
		output:  output,
		prefix:  "",
	}
}

// SetLevel sets the minimum log level.
func (l *TestLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput sets the output writer for log messages.
func (l *TestLogger) SetOutput(output io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = output
}

// SetPrefix sets a prefix for all log messages.
func (l *TestLogger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

// Debug logs a debug message.
func (l *TestLogger) Debug(message string, context ...map[string]interface{}) {
	l.log(LogLevelDebug, message, context...)
}

// Info logs an info message.
func (l *TestLogger) Info(message string, context ...map[string]interface{}) {
	l.log(LogLevelInfo, message, context...)
}

// Warn logs a warning message.
func (l *TestLogger) Warn(message string, context ...map[string]interface{}) {
	l.log(LogLevelWarn, message, context...)
}

// Error logs an error message.
func (l *TestLogger) Error(message string, context ...map[string]interface{}) {
	l.log(LogLevelError, message, context...)
}

// Debugf logs a formatted debug message.
func (l *TestLogger) Debugf(format string, args ...interface{}) {
	l.log(LogLevelDebug, fmt.Sprintf(format, args...))
}

// Infof logs a formatted info message.
func (l *TestLogger) Infof(format string, args ...interface{}) {
	l.log(LogLevelInfo, fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning message.
func (l *TestLogger) Warnf(format string, args ...interface{}) {
	l.log(LogLevelWarn, fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message.
func (l *TestLogger) Errorf(format string, args ...interface{}) {
	l.log(LogLevelError, fmt.Sprintf(format, args...))
}

// log is the internal logging function.
func (l *TestLogger) log(level LogLevel, message string, context ...map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if level is enabled
	if level < l.level {
		return
	}

	// Create log entry
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Context:   ctx,
	}

	// Store entry
	l.entries = append(l.entries, entry)

	// Write to output if configured
	if l.output != nil {
		l.writeEntry(entry)
	}
}

// writeEntry writes a log entry to the output.
func (l *TestLogger) writeEntry(entry LogEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05.000")
	prefix := l.prefix
	if prefix != "" {
		prefix = prefix + " "
	}

	line := fmt.Sprintf("[%s] %s%s: %s", timestamp, prefix, entry.Level, entry.Message)

	if len(entry.Context) > 0 {
		line += fmt.Sprintf(" %v", entry.Context)
	}

	line += "\n"

	_, _ = l.output.Write([]byte(line))
}

// GetEntries returns all collected log entries.
func (l *TestLogger) GetEntries() []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return a copy
	entries := make([]LogEntry, len(l.entries))
	copy(entries, l.entries)
	return entries
}

// GetEntriesByLevel returns all log entries with the specified level.
func (l *TestLogger) GetEntriesByLevel(level LogLevel) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var result []LogEntry
	for _, entry := range l.entries {
		if entry.Level == level {
			result = append(result, entry)
		}
	}
	return result
}

// GetEntriesCount returns the total number of log entries.
func (l *TestLogger) GetEntriesCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries)
}

// GetEntriesCountByLevel returns the number of log entries with the specified level.
func (l *TestLogger) GetEntriesCountByLevel(level LogLevel) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	count := 0
	for _, entry := range l.entries {
		if entry.Level == level {
			count++
		}
	}
	return count
}

// HasErrors returns true if any error-level entries were logged.
func (l *TestLogger) HasErrors() bool {
	return l.GetEntriesCountByLevel(LogLevelError) > 0
}

// HasWarnings returns true if any warning-level entries were logged.
func (l *TestLogger) HasWarnings() bool {
	return l.GetEntriesCountByLevel(LogLevelWarn) > 0
}

// Clear clears all collected log entries.
func (l *TestLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = make([]LogEntry, 0)
}

// String returns a string representation of all log entries.
func (l *TestLogger) String() string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var buf bytes.Buffer
	for _, entry := range l.entries {
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05.000")
		line := fmt.Sprintf("[%s] %s: %s\n", timestamp, entry.Level, entry.Message)
		buf.WriteString(line)
	}
	return buf.String()
}

// ==========================
// Buffer Logger
// ==========================

// BufferedLogger is a logger that writes to an in-memory buffer.
type BufferedLogger struct {
	*TestLogger
	buffer *bytes.Buffer
}

// NewBufferedLogger creates a new buffered logger.
func NewBufferedLogger() *BufferedLogger {
	buffer := &bytes.Buffer{}
	return &BufferedLogger{
		TestLogger: NewTestLoggerWithOutput(buffer),
		buffer:     buffer,
	}
}

// GetBuffer returns the underlying buffer.
func (l *BufferedLogger) GetBuffer() *bytes.Buffer {
	return l.buffer
}

// GetOutput returns the buffer contents as a string.
func (l *BufferedLogger) GetOutput() string {
	return l.buffer.String()
}

// ClearBuffer clears the buffer contents.
func (l *BufferedLogger) ClearBuffer() {
	l.buffer.Reset()
}

// ==========================
// File Logger
// ==========================

// FileLogger is a logger that writes to a file.
type FileLogger struct {
	*TestLogger
	file *os.File
}

// NewFileLogger creates a new file logger.
func NewFileLogger(filepath string) (*FileLogger, error) {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &FileLogger{
		TestLogger: NewTestLoggerWithOutput(file),
		file:       file,
	}, nil
}

// Close closes the log file.
func (l *FileLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// ==========================
// Multi Logger
// ==========================

// MultiLogger logs to multiple outputs simultaneously.
type MultiLogger struct {
	*TestLogger
	loggers []*TestLogger
}

// NewMultiLogger creates a new multi-logger.
func NewMultiLogger(loggers ...*TestLogger) *MultiLogger {
	return &MultiLogger{
		TestLogger: NewTestLogger(),
		loggers:    loggers,
	}
}

// AddLogger adds a logger to the multi-logger.
func (l *MultiLogger) AddLogger(logger *TestLogger) {
	l.loggers = append(l.loggers, logger)
}

// Debug logs a debug message to all loggers.
func (l *MultiLogger) Debug(message string, context ...map[string]interface{}) {
	l.TestLogger.Debug(message, context...)
	for _, logger := range l.loggers {
		logger.Debug(message, context...)
	}
}

// Info logs an info message to all loggers.
func (l *MultiLogger) Info(message string, context ...map[string]interface{}) {
	l.TestLogger.Info(message, context...)
	for _, logger := range l.loggers {
		logger.Info(message, context...)
	}
}

// Warn logs a warning message to all loggers.
func (l *MultiLogger) Warn(message string, context ...map[string]interface{}) {
	l.TestLogger.Warn(message, context...)
	for _, logger := range l.loggers {
		logger.Warn(message, context...)
	}
}

// Error logs an error message to all loggers.
func (l *MultiLogger) Error(message string, context ...map[string]interface{}) {
	l.TestLogger.Error(message, context...)
	for _, logger := range l.loggers {
		logger.Error(message, context...)
	}
}

// ==========================
// Logger Helpers
// ==========================

// LogStepExecution logs the execution of a step.
func LogStepExecution(logger *TestLogger, sagaID, stepName string, attempt int) {
	logger.Infof("Executing step %s for saga %s (attempt %d)", stepName, sagaID, attempt)
}

// LogStepSuccess logs the successful completion of a step.
func LogStepSuccess(logger *TestLogger, sagaID, stepName string, duration time.Duration) {
	logger.Infof("Step %s for saga %s completed successfully in %v", stepName, sagaID, duration)
}

// LogStepFailure logs the failure of a step.
func LogStepFailure(logger *TestLogger, sagaID, stepName string, err error) {
	logger.Errorf("Step %s for saga %s failed: %v", stepName, sagaID, err)
}

// LogCompensation logs the compensation of a step.
func LogCompensation(logger *TestLogger, sagaID, stepName string) {
	logger.Warnf("Compensating step %s for saga %s", stepName, sagaID)
}

// LogSagaStarted logs the start of a saga.
func LogSagaStarted(logger *TestLogger, sagaID, sagaName string) {
	logger.Infof("Saga %s (%s) started", sagaID, sagaName)
}

// LogSagaCompleted logs the completion of a saga.
func LogSagaCompleted(logger *TestLogger, sagaID string, duration time.Duration) {
	logger.Infof("Saga %s completed successfully in %v", sagaID, duration)
}

// LogSagaFailed logs the failure of a saga.
func LogSagaFailed(logger *TestLogger, sagaID string, err error) {
	logger.Errorf("Saga %s failed: %v", sagaID, err)
}
