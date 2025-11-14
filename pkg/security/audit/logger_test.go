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

package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewAuditLogger(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		config      *AuditLoggerConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "with nil config - stdout",
			config: &AuditLoggerConfig{
				Enabled:     true,
				OutputType:  OutputTypeStdout,
				MinSeverity: SeverityInfo,
			},
			wantErr: false,
		},
		{
			name: "with file config",
			config: &AuditLoggerConfig{
				Enabled:     true,
				OutputType:  OutputTypeFile,
				FilePath:    filepath.Join(tmpDir, "audit.log"),
				MinSeverity: SeverityInfo,
			},
			wantErr: false,
		},
		{
			name: "with disabled config",
			config: &AuditLoggerConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "with stdout output",
			config: &AuditLoggerConfig{
				Enabled:    true,
				OutputType: OutputTypeStdout,
			},
			wantErr: false,
		},
		{
			name: "with custom writer",
			config: &AuditLoggerConfig{
				Enabled:      true,
				OutputType:   OutputTypeCustom,
				CustomWriter: &bytes.Buffer{},
			},
			wantErr: false,
		},
		{
			name: "with invalid output type",
			config: &AuditLoggerConfig{
				Enabled:    true,
				OutputType: "invalid",
			},
			wantErr:     true,
			errContains: "invalid output_type",
		},
		{
			name: "with invalid min severity",
			config: &AuditLoggerConfig{
				Enabled:     true,
				OutputType:  OutputTypeStdout,
				MinSeverity: "invalid",
			},
			wantErr:     true,
			errContains: "invalid min_severity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewAuditLogger(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewAuditLogger() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewAuditLogger() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("NewAuditLogger() unexpected error = %v", err)
				return
			}
			if logger == nil {
				t.Errorf("NewAuditLogger() returned nil logger")
				return
			}
			defer logger.Close()
		})
	}
}

func TestAuditLogger_Log(t *testing.T) {
	tests := []struct {
		name       string
		config     *AuditLoggerConfig
		event      *AuditEvent
		wantLogged bool
	}{
		{
			name: "log info event",
			config: &AuditLoggerConfig{
				Enabled:      true,
				OutputType:   OutputTypeCustom,
				CustomWriter: &bytes.Buffer{},
				MinSeverity:  SeverityInfo,
				Async:        false,
			},
			event: NewAuditEvent(EventTypeAuthSuccess, SeverityInfo).
				WithActor(ActorInfo{
					UserID:   "user123",
					Username: "testuser",
				}).
				WithResult(ResultInfo{
					Status: ResultSuccess,
				}),
			wantLogged: true,
		},
		{
			name: "filter low severity event",
			config: &AuditLoggerConfig{
				Enabled:      true,
				OutputType:   OutputTypeCustom,
				CustomWriter: &bytes.Buffer{},
				MinSeverity:  SeverityHigh,
				Async:        false,
			},
			event: NewAuditEvent(EventTypeAuthSuccess, SeverityInfo).
				WithActor(ActorInfo{
					UserID: "user123",
				}),
			wantLogged: false,
		},
		{
			name: "filter by event type",
			config: &AuditLoggerConfig{
				Enabled:          true,
				OutputType:       OutputTypeCustom,
				CustomWriter:     &bytes.Buffer{},
				MinSeverity:      SeverityInfo,
				EventTypeFilters: []string{EventTypeAuthSuccess},
				Async:            false,
			},
			event: NewAuditEvent(EventTypeAuthFailure, SeverityMedium).
				WithActor(ActorInfo{
					UserID: "user123",
				}),
			wantLogged: false,
		},
		{
			name: "log high severity event",
			config: &AuditLoggerConfig{
				Enabled:      true,
				OutputType:   OutputTypeCustom,
				CustomWriter: &bytes.Buffer{},
				MinSeverity:  SeverityMedium,
				Async:        false,
			},
			event: NewAuditEvent(EventTypeAccessDenied, SeverityHigh).
				WithActor(ActorInfo{
					UserID: "user123",
				}).
				WithTarget(TargetInfo{
					ResourceType: "api",
					ResourceID:   "resource123",
				}),
			wantLogged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewAuditLogger(tt.config)
			if err != nil {
				t.Fatalf("NewAuditLogger() error = %v", err)
			}
			defer logger.Close()

			err = logger.Log(tt.event)
			if err != nil {
				t.Errorf("Log() error = %v", err)
			}

			// 同步日志立即刷新
			if !tt.config.Async {
				logger.zapLogger.Sync()
			}
		})
	}
}

func TestAuditLogger_LogWithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	// 创建带上下文的请求
	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "req-123")
	ctx = context.WithValue(ctx, "trace_id", "trace-456")
	ctx = context.WithValue(ctx, "span_id", "span-789")

	event := NewAuditEvent(EventTypeAuthSuccess, SeverityInfo).
		WithActor(ActorInfo{
			UserID: "user123",
		})

	err = logger.LogWithContext(ctx, event)
	if err != nil {
		t.Errorf("LogWithContext() error = %v", err)
	}

	// 验证上下文信息被提取
	if event.Context.RequestID != "req-123" {
		t.Errorf("RequestID = %v, want %v", event.Context.RequestID, "req-123")
	}
	if event.Context.TraceID != "trace-456" {
		t.Errorf("TraceID = %v, want %v", event.Context.TraceID, "trace-456")
	}
	if event.Context.SpanID != "span-789" {
		t.Errorf("SpanID = %v, want %v", event.Context.SpanID, "span-789")
	}
}

func TestAuditLogger_LogAuthAttempt(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	actor := ActorInfo{
		UserID:    "user123",
		Username:  "testuser",
		IPAddress: "192.168.1.1",
	}

	err = logger.LogAuthAttempt(actor, "jwt", ResultSuccess)
	if err != nil {
		t.Errorf("LogAuthAttempt() error = %v", err)
	}

	logger.zapLogger.Sync()
}

func TestAuditLogger_LogAuthSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	actor := ActorInfo{
		UserID:   "user123",
		Username: "testuser",
	}

	err = logger.LogAuthSuccess(actor, "jwt")
	if err != nil {
		t.Errorf("LogAuthSuccess() error = %v", err)
	}
}

func TestAuditLogger_LogAuthFailure(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	actor := ActorInfo{
		UserID:    "user123",
		Username:  "testuser",
		IPAddress: "192.168.1.1",
	}

	err = logger.LogAuthFailure(actor, "jwt", "invalid token")
	if err != nil {
		t.Errorf("LogAuthFailure() error = %v", err)
	}
}

func TestAuditLogger_LogPolicyEvaluation(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	actor := ActorInfo{
		UserID: "user123",
		Roles:  []string{"admin"},
	}

	target := TargetInfo{
		ResourceType: "api",
		ResourceID:   "resource123",
	}

	err = logger.LogPolicyEvaluation(actor, "rbac", ResultSuccess, target)
	if err != nil {
		t.Errorf("LogPolicyEvaluation() error = %v", err)
	}
}

func TestAuditLogger_LogAccessDenied(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	actor := ActorInfo{
		UserID: "user123",
		Roles:  []string{"user"},
	}

	target := TargetInfo{
		ResourceType: "api",
		ResourceID:   "admin-resource",
	}

	err = logger.LogAccessDenied(actor, target, "insufficient permissions")
	if err != nil {
		t.Errorf("LogAccessDenied() error = %v", err)
	}
}

func TestAuditLogger_LogAccessGranted(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	actor := ActorInfo{
		UserID: "user123",
		Roles:  []string{"admin"},
	}

	target := TargetInfo{
		ResourceType: "api",
		ResourceID:   "resource123",
	}

	err = logger.LogAccessGranted(actor, target)
	if err != nil {
		t.Errorf("LogAccessGranted() error = %v", err)
	}
}

func TestAuditLogger_LogSensitiveOperation(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	actor := ActorInfo{
		UserID: "admin123",
		Roles:  []string{"admin"},
	}

	action := ActionInfo{
		Operation:   "export_data",
		Description: "Export user data",
	}

	target := TargetInfo{
		ResourceType: "user_data",
		ResourceID:   "batch-001",
	}

	result := ResultInfo{
		Status: ResultSuccess,
	}

	err = logger.LogSensitiveOperation(actor, action, target, result)
	if err != nil {
		t.Errorf("LogSensitiveOperation() error = %v", err)
	}
}

func TestAuditLogger_LogDataAccess(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	actor := ActorInfo{
		UserID: "user123",
	}

	target := TargetInfo{
		ResourceType: "document",
		ResourceID:   "doc123",
	}

	err = logger.LogDataAccess(actor, target)
	if err != nil {
		t.Errorf("LogDataAccess() error = %v", err)
	}
}

func TestAuditLogger_LogDataModification(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	actor := ActorInfo{
		UserID: "user123",
	}

	target := TargetInfo{
		ResourceType: "document",
		ResourceID:   "doc123",
	}

	err = logger.LogDataModification(actor, target, "update")
	if err != nil {
		t.Errorf("LogDataModification() error = %v", err)
	}
}

func TestAuditLogger_LogDataDeletion(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	actor := ActorInfo{
		UserID: "admin123",
		Roles:  []string{"admin"},
	}

	target := TargetInfo{
		ResourceType: "document",
		ResourceID:   "doc123",
	}

	err = logger.LogDataDeletion(actor, target)
	if err != nil {
		t.Errorf("LogDataDeletion() error = %v", err)
	}
}

func TestAuditLogger_LogSecurityViolation(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	actor := ActorInfo{
		UserID:    "user123",
		IPAddress: "192.168.1.1",
	}

	details := map[string]interface{}{
		"attempt_count": 5,
		"time_window":   "5m",
	}

	err = logger.LogSecurityViolation(actor, "brute force detected", details)
	if err != nil {
		t.Errorf("LogSecurityViolation() error = %v", err)
	}
}

func TestAuditLogger_AsyncMode(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        true,
		BufferSize:   100,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	// 记录多个事件
	for i := 0; i < 10; i++ {
		event := NewAuditEvent(EventTypeAuthSuccess, SeverityInfo).
			WithActor(ActorInfo{
				UserID: "user123",
			})
		err = logger.Log(event)
		if err != nil {
			t.Errorf("Log() error = %v", err)
		}
	}

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)
}

func TestAuditLogger_FileOutput(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "audit.log")

	config := &AuditLoggerConfig{
		Enabled:     true,
		OutputType:  OutputTypeFile,
		FilePath:    logFile,
		MinSeverity: SeverityInfo,
		Async:       false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}

	event := NewAuditEvent(EventTypeAuthSuccess, SeverityInfo).
		WithActor(ActorInfo{
			UserID:   "user123",
			Username: "testuser",
		})

	err = logger.Log(event)
	if err != nil {
		t.Errorf("Log() error = %v", err)
	}

	// 关闭并刷新
	err = logger.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file was not created: %v", err)
	}
}

func TestAuditLogger_Disabled(t *testing.T) {
	config := &AuditLoggerConfig{
		Enabled: false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	event := NewAuditEvent(EventTypeAuthSuccess, SeverityInfo)
	err = logger.Log(event)
	if err != nil {
		t.Errorf("Log() error = %v", err)
	}
}

func TestAuditLogger_Close(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        true,
		BufferSize:   100,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}

	// 记录事件
	event := NewAuditEvent(EventTypeAuthSuccess, SeverityInfo)
	err = logger.Log(event)
	if err != nil {
		t.Errorf("Log() error = %v", err)
	}

	// 关闭
	err = logger.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 再次关闭应该不会出错
	err = logger.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

func TestAuditLogger_SeverityFiltering(t *testing.T) {
	tests := []struct {
		name          string
		minSeverity   string
		eventSeverity string
		wantLogged    bool
	}{
		{
			name:          "info event with info min",
			minSeverity:   SeverityInfo,
			eventSeverity: SeverityInfo,
			wantLogged:    true,
		},
		{
			name:          "low event with info min",
			minSeverity:   SeverityInfo,
			eventSeverity: SeverityLow,
			wantLogged:    true,
		},
		{
			name:          "info event with high min",
			minSeverity:   SeverityHigh,
			eventSeverity: SeverityInfo,
			wantLogged:    false,
		},
		{
			name:          "critical event with medium min",
			minSeverity:   SeverityMedium,
			eventSeverity: SeverityCritical,
			wantLogged:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			config := &AuditLoggerConfig{
				Enabled:      true,
				OutputType:   OutputTypeCustom,
				CustomWriter: buf,
				MinSeverity:  tt.minSeverity,
				Async:        false,
			}

			logger, err := NewAuditLogger(config)
			if err != nil {
				t.Fatalf("NewAuditLogger() error = %v", err)
			}
			defer logger.Close()

			event := NewAuditEvent(EventTypeAuthSuccess, tt.eventSeverity)
			err = logger.Log(event)
			if err != nil {
				t.Errorf("Log() error = %v", err)
			}
		})
	}
}

func TestAuditLogger_EventTypeFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:          true,
		OutputType:       OutputTypeCustom,
		CustomWriter:     buf,
		MinSeverity:      SeverityInfo,
		EventTypeFilters: []string{EventTypeAuthSuccess, EventTypeAuthFailure},
		Async:            false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	// 应该被记录
	event1 := NewAuditEvent(EventTypeAuthSuccess, SeverityInfo)
	err = logger.Log(event1)
	if err != nil {
		t.Errorf("Log() error = %v", err)
	}

	// 应该被过滤
	event2 := NewAuditEvent(EventTypePolicyEvaluation, SeverityInfo)
	err = logger.Log(event2)
	if err != nil {
		t.Errorf("Log() error = %v", err)
	}
}

func TestAuditLogger_WithZapLogger(t *testing.T) {
	// 创建自定义 Zap Logger
	buf := &bytes.Buffer{}
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:    "timestamp",
		MessageKey: "message",
		EncodeTime: zapcore.ISO8601TimeEncoder,
	})
	core := zapcore.NewCore(encoder, zapcore.AddSync(buf), zapcore.InfoLevel)
	zapLogger := zap.New(core)

	config := &AuditLoggerConfig{
		Enabled:     true,
		OutputType:  OutputTypeCustom,
		ZapLogger:   zapLogger,
		MinSeverity: SeverityInfo,
		Async:       false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	event := NewAuditEvent(EventTypeAuthSuccess, SeverityInfo)
	err = logger.Log(event)
	if err != nil {
		t.Errorf("Log() error = %v", err)
	}
}

func TestAuditEvent_EventChecks(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		checkFunc func(*AuditEvent) bool
		want      bool
	}{
		{
			name:      "auth event check - success",
			eventType: EventTypeAuthSuccess,
			checkFunc: (*AuditEvent).IsAuthenticationEvent,
			want:      true,
		},
		{
			name:      "auth event check - failure",
			eventType: EventTypePolicyEvaluation,
			checkFunc: (*AuditEvent).IsAuthenticationEvent,
			want:      false,
		},
		{
			name:      "authz event check - success",
			eventType: EventTypePolicyEvaluation,
			checkFunc: (*AuditEvent).IsAuthorizationEvent,
			want:      true,
		},
		{
			name:      "sensitive op check - success",
			eventType: EventTypeSensitiveOperation,
			checkFunc: (*AuditEvent).IsSensitiveOperation,
			want:      true,
		},
		{
			name:      "security event check - success",
			eventType: EventTypeSecurityViolation,
			checkFunc: (*AuditEvent).IsSecurityEvent,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewAuditEvent(tt.eventType, SeverityInfo)
			if got := tt.checkFunc(event); got != tt.want {
				t.Errorf("check function returned %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuditEvent_IsCritical(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		want     bool
	}{
		{
			name:     "info is not critical",
			severity: SeverityInfo,
			want:     false,
		},
		{
			name:     "low is not critical",
			severity: SeverityLow,
			want:     false,
		},
		{
			name:     "medium is not critical",
			severity: SeverityMedium,
			want:     false,
		},
		{
			name:     "high is critical",
			severity: SeverityHigh,
			want:     true,
		},
		{
			name:     "critical is critical",
			severity: SeverityCritical,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewAuditEvent(EventTypeAuthSuccess, tt.severity)
			if got := event.IsCritical(); got != tt.want {
				t.Errorf("IsCritical() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuditEvent_FluentAPI(t *testing.T) {
	event := NewAuditEvent(EventTypeAuthSuccess, SeverityInfo).
		WithActor(ActorInfo{
			UserID:   "user123",
			Username: "testuser",
		}).
		WithAction(ActionInfo{
			Operation: "login",
			Method:    "jwt",
		}).
		WithTarget(TargetInfo{
			ResourceType: "api",
			ResourceID:   "auth-service",
		}).
		WithResult(ResultInfo{
			Status: ResultSuccess,
		}).
		WithContext(ContextInfo{
			RequestID: "req-123",
			TraceID:   "trace-456",
		}).
		AddMetadata("custom_field", "custom_value")

	// 验证所有字段都被正确设置
	if event.Actor.UserID != "user123" {
		t.Errorf("Actor.UserID = %v, want %v", event.Actor.UserID, "user123")
	}
	if event.Action.Operation != "login" {
		t.Errorf("Action.Operation = %v, want %v", event.Action.Operation, "login")
	}
	if event.Target.ResourceType != "api" {
		t.Errorf("Target.ResourceType = %v, want %v", event.Target.ResourceType, "api")
	}
	if event.Result.Status != ResultSuccess {
		t.Errorf("Result.Status = %v, want %v", event.Result.Status, ResultSuccess)
	}
	if event.Context.RequestID != "req-123" {
		t.Errorf("Context.RequestID = %v, want %v", event.Context.RequestID, "req-123")
	}
	if event.Metadata["custom_field"] != "custom_value" {
		t.Errorf("Metadata[custom_field] = %v, want %v", event.Metadata["custom_field"], "custom_value")
	}
}

func TestAuditEvent_JSON(t *testing.T) {
	event := NewAuditEvent(EventTypeAuthSuccess, SeverityInfo).
		WithActor(ActorInfo{
			UserID:   "user123",
			Username: "testuser",
		}).
		WithAction(ActionInfo{
			Operation: "login",
		}).
		WithResult(ResultInfo{
			Status: ResultSuccess,
		})

	// 序列化为 JSON
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// 反序列化
	var decoded AuditEvent
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// 验证关键字段
	if decoded.EventType != EventTypeAuthSuccess {
		t.Errorf("EventType = %v, want %v", decoded.EventType, EventTypeAuthSuccess)
	}
	if decoded.Actor.UserID != "user123" {
		t.Errorf("Actor.UserID = %v, want %v", decoded.Actor.UserID, "user123")
	}
}

func BenchmarkAuditLogger_Log(b *testing.B) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		b.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	event := NewAuditEvent(EventTypeAuthSuccess, SeverityInfo).
		WithActor(ActorInfo{
			UserID: "user123",
		})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Log(event)
	}
}

func BenchmarkAuditLogger_LogAsync(b *testing.B) {
	buf := &bytes.Buffer{}
	config := &AuditLoggerConfig{
		Enabled:      true,
		OutputType:   OutputTypeCustom,
		CustomWriter: buf,
		MinSeverity:  SeverityInfo,
		Async:        true,
		BufferSize:   10000,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		b.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	event := NewAuditEvent(EventTypeAuthSuccess, SeverityInfo).
		WithActor(ActorInfo{
			UserID: "user123",
		})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Log(event)
	}
}
