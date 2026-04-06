package result

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

// Response 定义统一响应结构。
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	TraceID   string      `json:"trace_id"`
	Timestamp string      `json:"timestamp"`
}

// Success 返回统一成功响应。
func Success(ctx *gin.Context, data interface{}) {
	ctx.JSON(200, Response{
		Code:      0,
		Message:   "ok",
		Data:      data,
		TraceID:   ctxmeta.TraceID(ctx.Request.Context()),
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// Error 返回统一失败响应。
func Error(ctx *gin.Context, httpStatus int, code int, message string) {
	ctx.JSON(httpStatus, Response{
		Code:      code,
		Message:   message,
		Data:      nil,
		TraceID:   ctxmeta.TraceID(ctx.Request.Context()),
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
