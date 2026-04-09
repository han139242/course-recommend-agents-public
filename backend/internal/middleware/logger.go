package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
)

// Logger 输出单条聚合访问日志。
//
// 该中间件在请求结束后统一记录路径、状态码与耗时，
// 便于排查链路问题，同时避免在处理过程中产生重复访问日志。
func Logger(logger *applogger.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()
		ctx.Next()

		latency := time.Since(startedAt)
		path := ctx.FullPath()
		if path == "" {
			path = ctx.Request.URL.Path
		}
		fields := []applogger.Field{
			applogger.String("method", ctx.Request.Method),
			applogger.String("path", path),
			applogger.String("raw_path", ctx.Request.URL.Path),
			applogger.String("client_ip", ctx.ClientIP()),
			applogger.Int("status", ctx.Writer.Status()),
			applogger.Duration("latency", latency),
			applogger.Int("body_size", ctx.Writer.Size()),
			applogger.String("user_agent", ctx.Request.UserAgent()),
		}
		if latency >= 500*time.Millisecond {
			fields = append(fields, applogger.Bool("slow_request", true))
			logger.WithCtx(ctx.Request.Context()).Warn("检测到慢请求", fields...)
			return
		}
		logger.WithCtx(ctx.Request.Context()).Info("请求完成", fields...)
	}
}
