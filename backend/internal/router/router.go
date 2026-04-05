package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// New 创建 Gin 路由实例。
func New() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())

	engine.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	engine.GET("/readyz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	})

	api := engine.Group("/api")
	api.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	return engine
}
