package middleware

import "github.com/gin-gonic/gin"

// RealIP 预留客户端真实 IP 解析入口。
//
// 当前阶段直接透传，请求链路依赖 Gin 默认的 ClientIP 行为；
// 后续如接入网关或反向代理，可在此处集中补充可信代理解析策略。
func RealIP() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
	}
}
