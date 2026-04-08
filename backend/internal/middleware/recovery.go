package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Recovery 捕获 panic 并返回统一错误响应。
//
// 该中间件的职责是兜底保护请求生命周期，防止单次 panic 直接打断进程；
// 对外仅暴露统一错误，避免把内部实现细节泄露给客户端。
func Recovery(logger *applogger.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.WithCtx(ctx.Request.Context()).Error(
					"请求发生 panic",
					applogger.Any("panic", fmt.Sprint(recovered)),
				)
				result.Error(ctx, apperr.ErrInternal.HTTPStatus, apperr.ErrInternal.Code, apperr.ErrInternal.Message)
				ctx.Abort()
			}
		}()

		ctx.Next()
	}
}
