package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/pkg/constant"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

// RequestID 为每个请求生成 request_id 与 trace_id。
//
// trace_id 写入 context 供日志链路透传，request_id 保留在 Gin 上下文中，
// 便于响应扩展或与网关层约定对接。
func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := newID()
		traceID := newID()

		requestCtx := ctx.Request.Context()
		requestCtx = ctxmeta.WithTraceID(requestCtx, traceID)
		ctx.Request = ctx.Request.WithContext(requestCtx)

		ctx.Set(constant.ContextRequestID, requestID)
		ctx.Next()
	}
}

// newID 生成十六进制随机标识；随机源失败时返回固定降级值。
func newID() string {
	buffer := make([]byte, 16)
	_, err := rand.Read(buffer)
	if err != nil {
		return "generated-request-id"
	}
	return hex.EncodeToString(buffer)
}
