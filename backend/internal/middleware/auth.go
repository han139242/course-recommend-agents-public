package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Auth 校验 Bearer Token 并注入用户身份。
func Auth(jwtService *service.JWTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorization := strings.TrimSpace(ctx.GetHeader("Authorization"))
		if authorization == "" {
			result.Error(ctx, apperr.ErrUnauthorized.HTTPStatus, apperr.ErrUnauthorized.Code, apperr.ErrUnauthorized.Message)
			ctx.Abort()
			return
		}

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authorization, bearerPrefix) {
			result.Error(ctx, apperr.ErrUnauthorized.HTTPStatus, apperr.ErrUnauthorized.Code, apperr.ErrUnauthorized.Message)
			ctx.Abort()
			return
		}

		rawToken := strings.TrimSpace(strings.TrimPrefix(authorization, bearerPrefix))
		claims, err := jwtService.Parse(rawToken)
		if err != nil {
			result.Error(ctx, apperr.ErrUnauthorized.HTTPStatus, apperr.ErrUnauthorized.Code, apperr.ErrUnauthorized.Message)
			ctx.Abort()
			return
		}

		requestCtx := ctx.Request.Context()
		requestCtx = ctxmeta.WithUserID(requestCtx, formatUserID(claims.UserID))
		requestCtx = ctxmeta.WithRole(requestCtx, claims.Role)
		ctx.Request = ctx.Request.WithContext(requestCtx)
		ctx.Next()
	}
}

func formatUserID(userID int64) string {
	return strconv.FormatInt(userID, 10)
}
