package logger

import (
	"context"
	"strings"

	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Field 是对外暴露的统一日志字段类型。
type Field = zap.Field

// Logger 对外提供统一日志接口，并隐藏底层日志实现。
type Logger struct {
	base *zap.Logger
}

// Entry 表示绑定了上下文的日志入口。
type Entry struct {
	ctx    context.Context
	logger *zap.Logger
}

// New 创建结构化日志实例。
func New(level string) *Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.MessageKey = "message"
	encoderConfig.LevelKey = "level"
	encoderConfig.CallerKey = "caller"
	encoderConfig.StacktraceKey = "stacktrace"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(parseLevel(level)),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	base, err := config.Build()
	if err != nil {
		panic("初始化 zap 日志失败: " + err.Error())
	}

	return &Logger{base: base}
}

// WithCtx 绑定上下文，并自动注入 trace_id 与 user_id。
func (l *Logger) WithCtx(ctx context.Context) *Entry {
	if ctx == nil {
		ctx = context.Background()
	}

	fields := make([]Field, 0, 2)
	if traceID := ctxmeta.TraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	if userID := ctxmeta.UserID(ctx); userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}

	return &Entry{
		ctx:    ctx,
		logger: l.base.With(fields...),
	}
}

// Sync 刷新日志缓冲。
func (l *Logger) Sync() error {
	if l == nil || l.base == nil {
		return nil
	}
	return l.base.Sync()
}

// Debug 输出调试日志。
func (e *Entry) Debug(msg string, fields ...Field) {
	e.logger.Debug(msg, fields...)
}

// Info 输出信息日志。
func (e *Entry) Info(msg string, fields ...Field) {
	e.logger.Info(msg, fields...)
}

// Warn 输出告警日志。
func (e *Entry) Warn(msg string, fields ...Field) {
	e.logger.Warn(msg, fields...)
}

// Error 输出错误日志。
func (e *Entry) Error(msg string, fields ...Field) {
	e.logger.Error(msg, fields...)
}

// String 构造字符串字段。
func String(key string, value string) Field {
	return zap.String(key, value)
}

// Int 构造整数字段。
func Int(key string, value int) Field {
	return zap.Int(key, value)
}

// Any 构造任意类型字段。
func Any(key string, value any) Field {
	return zap.Any(key, value)
}

// ErrorField 构造错误字段。
func ErrorField(key string, err error) Field {
	if err == nil {
		return zap.String(key, "")
	}
	return zap.NamedError(key, err)
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
