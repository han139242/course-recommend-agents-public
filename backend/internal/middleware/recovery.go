package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Recovery 捕获 panic 并返回统一错误响应。
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
