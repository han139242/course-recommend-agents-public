package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/pkg/constant"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

// RequestID 为每个请求生成 request_id 与 trace_id。
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

func newID() string {
	buffer := make([]byte, 16)
	_, err := rand.Read(buffer)
	if err != nil {
		return "generated-request-id"
	}
	return hex.EncodeToString(buffer)
}
