package ctxmeta

import "context"

type contextKey string

const (
	traceIDKey   contextKey = "trace_id"
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
)

// WithTraceID 向上下文写入 trace_id。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceID 从上下文读取 trace_id。
func TraceID(ctx context.Context) string {
	value, _ := ctx.Value(traceIDKey).(string)
	return value
}

// WithRequestID 向上下文写入 request_id。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestID 从上下文读取 request_id。
func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

// WithUserID 向上下文写入 user_id。
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserID 从上下文读取 user_id。
func UserID(ctx context.Context) string {
	value, _ := ctx.Value(userIDKey).(string)
	return value
}
