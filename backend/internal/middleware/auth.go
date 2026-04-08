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

// Auth 校验 Bearer Token 并向请求上下文注入身份信息。
//
// 这里注入的是 token 中的身份快照，不会回源确认用户最新状态；
// 对权限敏感的接口仍应在业务层基于 userID 做二次校验。
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

// formatUserID 将用户 ID 转为上下文中统一使用的字符串表示。
func formatUserID(userID int64) string {
	return strconv.FormatInt(userID, 10)
}
