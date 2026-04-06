package middleware

import "github.com/gin-gonic/gin"

// RealIP 保留客户端真实 IP 解析入口，当前阶段直接透传。
func RealIP() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
	}
}
