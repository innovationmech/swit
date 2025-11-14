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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/syslog"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// OutputType 定义输出类型
type OutputType string

const (
	OutputTypeFile   OutputType = "file"
	OutputTypeSyslog OutputType = "syslog"
	OutputTypeStdout OutputType = "stdout"
	OutputTypeCustom OutputType = "custom"
)

// SyslogProtocol 定义 Syslog 协议类型
type SyslogProtocol string

const (
	SyslogProtocolUDP  SyslogProtocol = "udp"
	SyslogProtocolTCP  SyslogProtocol = "tcp"
	SyslogProtocolUnix SyslogProtocol = "unix"
)

// AuditLoggerConfig 审计日志配置
type AuditLoggerConfig struct {
	// Enabled 是否启用审计日志
	Enabled bool `yaml:"enabled" json:"enabled"`

	// OutputType 输出类型：file, syslog, stdout, custom
	OutputType OutputType `yaml:"output_type" json:"output_type"`

	// 文件输出配置
	FilePath      string `yaml:"file_path" json:"file_path"`
	MaxSize       int    `yaml:"max_size" json:"max_size"`               // 最大文件大小（MB）
	MaxBackups    int    `yaml:"max_backups" json:"max_backups"`         // 最大备份文件数
	MaxAge        int    `yaml:"max_age" json:"max_age"`                 // 最大保留天数
	Compress      bool   `yaml:"compress" json:"compress"`               // 是否压缩旧日志
	LocalTime     bool   `yaml:"local_time" json:"local_time"`           // 是否使用本地时间
	RotateOnStart bool   `yaml:"rotate_on_start" json:"rotate_on_start"` // 启动时是否轮转

	// Syslog 配置
	SyslogNetwork  string          `yaml:"syslog_network" json:"syslog_network"`   // tcp, udp, unix
	SyslogAddress  string          `yaml:"syslog_address" json:"syslog_address"`   // 地址
	SyslogTag      string          `yaml:"syslog_tag" json:"syslog_tag"`           // 标签
	SyslogPriority syslog.Priority `yaml:"syslog_priority" json:"syslog_priority"` // 优先级

	// 日志级别过滤
	MinSeverity string `yaml:"min_severity" json:"min_severity"` // info, low, medium, high, critical

	// 事件类型过滤
	EventTypeFilters []string `yaml:"event_type_filters" json:"event_type_filters"` // 只记录指定类型的事件

	// 缓冲配置
	BufferSize int  `yaml:"buffer_size" json:"buffer_size"` // 异步缓冲大小
	Async      bool `yaml:"async" json:"async"`             // 是否异步写入

	// 格式配置
	PrettyPrint bool `yaml:"pretty_print" json:"pretty_print"` // 是否格式化 JSON

	// 自定义 Writer（用于测试或特殊场景）
	CustomWriter io.Writer `yaml:"-" json:"-"`

	// 自定义 Zap Logger
	ZapLogger *zap.Logger `yaml:"-" json:"-"`
}

// DefaultAuditLoggerConfig 返回默认配置
func DefaultAuditLoggerConfig() *AuditLoggerConfig {
	return &AuditLoggerConfig{
		Enabled:        true,
		OutputType:     OutputTypeFile,
		FilePath:       "/var/log/swit/audit.log",
		MaxSize:        100, // 100MB
		MaxBackups:     10,  // 保留10个备份
		MaxAge:         30,  // 保留30天
		Compress:       true,
		LocalTime:      true,
		RotateOnStart:  false,
		MinSeverity:    SeverityInfo,
		BufferSize:     1000,
		Async:          true,
		PrettyPrint:    false,
		SyslogTag:      "swit-audit",
		SyslogPriority: syslog.LOG_INFO | syslog.LOG_USER,
	}
}

// Validate 验证配置
func (c *AuditLoggerConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	// 设置默认值
	if c.MinSeverity == "" {
		c.MinSeverity = SeverityInfo
	}

	switch c.OutputType {
	case OutputTypeFile:
		if c.FilePath == "" {
			return fmt.Errorf("file_path is required for file output")
		}
		// 确保目录存在
		dir := filepath.Dir(c.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}
	case OutputTypeSyslog:
		if c.SyslogAddress == "" {
			return fmt.Errorf("syslog_address is required for syslog output")
		}
	case OutputTypeStdout:
		// 无需验证
	case OutputTypeCustom:
		if c.CustomWriter == nil && c.ZapLogger == nil {
			return fmt.Errorf("custom_writer or zap_logger is required for custom output")
		}
	default:
		return fmt.Errorf("invalid output_type: %s", c.OutputType)
	}

	// 验证严重性级别
	validSeverities := map[string]bool{
		SeverityInfo:     true,
		SeverityLow:      true,
		SeverityMedium:   true,
		SeverityHigh:     true,
		SeverityCritical: true,
	}
	if !validSeverities[c.MinSeverity] {
		return fmt.Errorf("invalid min_severity: %s", c.MinSeverity)
	}

	return nil
}

// AuditLogger 审计日志记录器
type AuditLogger struct {
	config       *AuditLoggerConfig
	zapLogger    *zap.Logger
	eventChan    chan *AuditEvent
	done         chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	closed       bool
	severityMap  map[string]int
	customWriter io.Writer
}

// NewAuditLogger 创建新的审计日志记录器
func NewAuditLogger(config *AuditLoggerConfig) (*AuditLogger, error) {
	if config == nil {
		config = DefaultAuditLoggerConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid audit logger config: %w", err)
	}

	if !config.Enabled {
		return &AuditLogger{
			config: config,
			closed: true,
		}, nil
	}

	logger := &AuditLogger{
		config: config,
		done:   make(chan struct{}),
		severityMap: map[string]int{
			SeverityInfo:     0,
			SeverityLow:      1,
			SeverityMedium:   2,
			SeverityHigh:     3,
			SeverityCritical: 4,
		},
	}

	// 初始化 Zap Logger
	if err := logger.initLogger(); err != nil {
		return nil, err
	}

	// 启动异步处理
	if config.Async {
		logger.eventChan = make(chan *AuditEvent, config.BufferSize)
		logger.wg.Add(1)
		go logger.processEvents()
	}

	return logger, nil
}

// initLogger 初始化 Zap Logger
func (l *AuditLogger) initLogger() error {
	// 如果提供了自定义 Zap Logger，直接使用
	if l.config.ZapLogger != nil {
		l.zapLogger = l.config.ZapLogger
		return nil
	}

	var zapConfig zap.Config

	// 基础配置
	zapConfig = zap.Config{
		Level:       zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{},
		ErrorOutputPaths: []string{"stderr"},
	}

	// 配置输出路径
	switch l.config.OutputType {
	case OutputTypeFile:
		zapConfig.OutputPaths = []string{l.config.FilePath}
	case OutputTypeStdout:
		zapConfig.OutputPaths = []string{"stdout"}
		if l.config.PrettyPrint {
			zapConfig.Encoding = "console"
		}
	case OutputTypeSyslog:
		// Syslog 需要特殊处理
		return l.initSyslogLogger()
	case OutputTypeCustom:
		// 自定义 Writer 需要特殊处理
		return l.initCustomLogger()
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return fmt.Errorf("failed to build zap logger: %w", err)
	}

	l.zapLogger = logger
	return nil
}

// initSyslogLogger 初始化 Syslog Logger
func (l *AuditLogger) initSyslogLogger() error {
	writer, err := syslog.Dial(
		l.config.SyslogNetwork,
		l.config.SyslogAddress,
		l.config.SyslogPriority,
		l.config.SyslogTag,
	)
	if err != nil {
		return fmt.Errorf("failed to connect to syslog: %w", err)
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			MessageKey:     "message",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
		}),
		zapcore.AddSync(writer),
		zapcore.InfoLevel,
	)

	l.zapLogger = zap.New(core)
	return nil
}

// initCustomLogger 初始化自定义 Logger
func (l *AuditLogger) initCustomLogger() error {
	if l.config.CustomWriter == nil {
		return fmt.Errorf("custom writer is required")
	}

	l.customWriter = l.config.CustomWriter

	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		MessageKey:     "message",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	})

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(l.config.CustomWriter),
		zapcore.InfoLevel,
	)

	l.zapLogger = zap.New(core)
	return nil
}

// processEvents 异步处理事件
func (l *AuditLogger) processEvents() {
	defer l.wg.Done()

	for {
		select {
		case event := <-l.eventChan:
			l.writeEvent(event)
		case <-l.done:
			// 处理剩余事件
			for len(l.eventChan) > 0 {
				event := <-l.eventChan
				l.writeEvent(event)
			}
			return
		}
	}
}

// writeEvent 写入事件
func (l *AuditLogger) writeEvent(event *AuditEvent) {
	if !l.shouldLogEvent(event) {
		return
	}

	// 将事件序列化为 JSON
	eventJSON, err := l.serializeEvent(event)
	if err != nil {
		l.zapLogger.Error("failed to serialize audit event",
			zap.String("event_id", event.EventID),
			zap.Error(err))
		return
	}

	// 记录到 Zap
	l.zapLogger.Info("audit_event",
		zap.String("event_json", string(eventJSON)))
}

// serializeEvent 序列化事件
func (l *AuditLogger) serializeEvent(event *AuditEvent) ([]byte, error) {
	if l.config.PrettyPrint {
		return json.MarshalIndent(event, "", "  ")
	}
	return json.Marshal(event)
}

// shouldLogEvent 判断是否应该记录事件
func (l *AuditLogger) shouldLogEvent(event *AuditEvent) bool {
	// 检查严重性级别
	if !l.checkSeverity(event.Severity) {
		return false
	}

	// 检查事件类型过滤
	if len(l.config.EventTypeFilters) > 0 {
		found := false
		for _, filter := range l.config.EventTypeFilters {
			if filter == event.EventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// checkSeverity 检查严重性级别
func (l *AuditLogger) checkSeverity(severity string) bool {
	minLevel, ok := l.severityMap[l.config.MinSeverity]
	if !ok {
		minLevel = 0
	}

	eventLevel, ok := l.severityMap[severity]
	if !ok {
		eventLevel = 0
	}

	return eventLevel >= minLevel
}

// Log 记录审计事件
func (l *AuditLogger) Log(event *AuditEvent) error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.closed || !l.config.Enabled {
		return nil
	}

	if l.config.Async {
		select {
		case l.eventChan <- event:
			return nil
		default:
			// 缓冲区满，直接写入
			l.writeEvent(event)
			return nil
		}
	}

	// 同步写入
	l.writeEvent(event)
	return nil
}

// LogWithContext 使用上下文记录审计事件
func (l *AuditLogger) LogWithContext(ctx context.Context, event *AuditEvent) error {
	// 从上下文中提取信息
	if requestID := extractFromContext(ctx, "request_id"); requestID != "" {
		event.Context.RequestID = requestID
	}
	if traceID := extractFromContext(ctx, "trace_id"); traceID != "" {
		event.Context.TraceID = traceID
	}
	if spanID := extractFromContext(ctx, "span_id"); spanID != "" {
		event.Context.SpanID = spanID
	}

	return l.Log(event)
}

// LogAuthAttempt 记录认证尝试
func (l *AuditLogger) LogAuthAttempt(actor ActorInfo, method, result string) error {
	event := NewAuditEvent(EventTypeAuthAttempt, SeverityInfo).
		WithActor(actor).
		WithAction(ActionInfo{
			Operation: "authenticate",
			Method:    method,
		}).
		WithResult(ResultInfo{
			Status: result,
		})

	return l.Log(event)
}

// LogAuthSuccess 记录认证成功
func (l *AuditLogger) LogAuthSuccess(actor ActorInfo, method string) error {
	event := NewAuditEvent(EventTypeAuthSuccess, SeverityInfo).
		WithActor(actor).
		WithAction(ActionInfo{
			Operation: "authenticate",
			Method:    method,
		}).
		WithResult(ResultInfo{
			Status: ResultSuccess,
		})

	return l.Log(event)
}

// LogAuthFailure 记录认证失败
func (l *AuditLogger) LogAuthFailure(actor ActorInfo, method, reason string) error {
	event := NewAuditEvent(EventTypeAuthFailure, SeverityMedium).
		WithActor(actor).
		WithAction(ActionInfo{
			Operation: "authenticate",
			Method:    method,
		}).
		WithResult(ResultInfo{
			Status:  ResultFailure,
			Message: reason,
		})

	return l.Log(event)
}

// LogPolicyEvaluation 记录策略评估
func (l *AuditLogger) LogPolicyEvaluation(actor ActorInfo, policy, decision string, target TargetInfo) error {
	severity := SeverityInfo
	if decision == ResultDenied {
		severity = SeverityMedium
	}

	event := NewAuditEvent(EventTypePolicyEvaluation, severity).
		WithActor(actor).
		WithAction(ActionInfo{
			Operation:   "evaluate_policy",
			Description: policy,
		}).
		WithTarget(target).
		WithResult(ResultInfo{
			Status: decision,
		})

	return l.Log(event)
}

// LogAccessDenied 记录访问拒绝
func (l *AuditLogger) LogAccessDenied(actor ActorInfo, target TargetInfo, reason string) error {
	event := NewAuditEvent(EventTypeAccessDenied, SeverityHigh).
		WithActor(actor).
		WithAction(ActionInfo{
			Operation: "access",
		}).
		WithTarget(target).
		WithResult(ResultInfo{
			Status:  ResultDenied,
			Message: reason,
		})

	return l.Log(event)
}

// LogAccessGranted 记录访问授予
func (l *AuditLogger) LogAccessGranted(actor ActorInfo, target TargetInfo) error {
	event := NewAuditEvent(EventTypeAccessGranted, SeverityInfo).
		WithActor(actor).
		WithAction(ActionInfo{
			Operation: "access",
		}).
		WithTarget(target).
		WithResult(ResultInfo{
			Status: ResultSuccess,
		})

	return l.Log(event)
}

// LogSensitiveOperation 记录敏感操作
func (l *AuditLogger) LogSensitiveOperation(actor ActorInfo, action ActionInfo, target TargetInfo, result ResultInfo) error {
	event := NewAuditEvent(EventTypeSensitiveOperation, SeverityHigh).
		WithActor(actor).
		WithAction(action).
		WithTarget(target).
		WithResult(result)

	return l.Log(event)
}

// LogDataAccess 记录数据访问
func (l *AuditLogger) LogDataAccess(actor ActorInfo, target TargetInfo) error {
	event := NewAuditEvent(EventTypeDataAccess, SeverityInfo).
		WithActor(actor).
		WithAction(ActionInfo{
			Operation: "read",
		}).
		WithTarget(target).
		WithResult(ResultInfo{
			Status: ResultSuccess,
		})

	return l.Log(event)
}

// LogDataModification 记录数据修改
func (l *AuditLogger) LogDataModification(actor ActorInfo, target TargetInfo, operation string) error {
	event := NewAuditEvent(EventTypeDataModification, SeverityMedium).
		WithActor(actor).
		WithAction(ActionInfo{
			Operation: operation,
		}).
		WithTarget(target).
		WithResult(ResultInfo{
			Status: ResultSuccess,
		})

	return l.Log(event)
}

// LogDataDeletion 记录数据删除
func (l *AuditLogger) LogDataDeletion(actor ActorInfo, target TargetInfo) error {
	event := NewAuditEvent(EventTypeDataDeletion, SeverityHigh).
		WithActor(actor).
		WithAction(ActionInfo{
			Operation: "delete",
		}).
		WithTarget(target).
		WithResult(ResultInfo{
			Status: ResultSuccess,
		})

	return l.Log(event)
}

// LogSecurityViolation 记录安全违规
func (l *AuditLogger) LogSecurityViolation(actor ActorInfo, violation string, details map[string]interface{}) error {
	event := NewAuditEvent(EventTypeSecurityViolation, SeverityCritical).
		WithActor(actor).
		WithAction(ActionInfo{
			Operation:   "security_violation",
			Description: violation,
			Parameters:  details,
		}).
		WithResult(ResultInfo{
			Status: ResultDenied,
		})

	return l.Log(event)
}

// Close 关闭审计日志记录器
func (l *AuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true

	if l.config.Async {
		close(l.done)
		l.wg.Wait()
		close(l.eventChan)
	}

	if l.zapLogger != nil {
		return l.zapLogger.Sync()
	}

	return nil
}

// extractFromContext 从上下文中提取字符串值
func extractFromContext(ctx context.Context, key string) string {
	if v := ctx.Value(key); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
