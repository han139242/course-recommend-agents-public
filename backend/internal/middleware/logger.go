package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
)

// Logger 输出单条聚合访问日志。
func Logger(logger *applogger.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()
		ctx.Next()

		logger.WithCtx(ctx.Request.Context()).Info(
			"请求完成",
			applogger.String("method", ctx.Request.Method),
			applogger.String("path", ctx.FullPath()),
			applogger.String("raw_path", ctx.Request.URL.Path),
			applogger.String("client_ip", ctx.ClientIP()),
			applogger.Int("status", ctx.Writer.Status()),
			applogger.String("latency", time.Since(startedAt).String()),
		)
	}
}
