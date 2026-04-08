package router

import (
	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/middleware"
	authmodule "github.com/han139242/course-recommend-agents/backend/internal/module/auth"
	learningmodule "github.com/han139242/course-recommend-agents/backend/internal/module/learning"
	teachermodule "github.com/han139242/course-recommend-agents/backend/internal/module/teacher"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// New 创建 Gin 路由实例。
//
// 路由层只负责统一挂载中间件、健康检查与模块路由，
// 不在此处处理业务编排，避免入口层承担过多职责。
func New(logger *applogger.Logger, authModule *authmodule.Module, teacherModule *teachermodule.Module, learningModule *learningmodule.Module) *gin.Engine {
	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.RealIP())
	engine.Use(middleware.Logger(logger))
	engine.Use(middleware.Recovery(logger))

	engine.GET("/healthz", func(ctx *gin.Context) {
		result.Success(ctx, gin.H{
			"status": "ok",
		})
	})

	engine.GET("/readyz", func(ctx *gin.Context) {
		result.Success(ctx, gin.H{
			"status": "ready",
		})
	})

	api := engine.Group("/api")

	// publicGroup 只注册无需登录即可访问的接口，便于探活和登录前流程复用。
	publicGroup := api.Group("")
	publicGroup.GET("/ping", func(ctx *gin.Context) {
		result.Success(ctx, gin.H{
			"message": "pong",
		})
	})
	publicGroup.GET("/panic", func(ctx *gin.Context) {
		panic("模拟 panic")
	})
	publicAuthGroup := publicGroup.Group("/auth")
	authModule.RegisterPublicRoutes(publicAuthGroup)

	// authGroup 统一挂鉴权中间件，确保后续受保护路由共享同一身份注入约束。
	authGroup := api.Group("")
	authGroup.Use(middleware.Auth(authModule.JWTService))
	protectedAuthGroup := authGroup.Group("/auth")
	authModule.RegisterAuthRoutes(protectedAuthGroup)
	teacherGroup := authGroup.Group("/teacher")
	teacherModule.RegisterRoutes(teacherGroup)
	learningModule.RegisterTeacherRoutes(teacherGroup)
	learningGroup := authGroup.Group("/learning")
	learningModule.RegisterStudentRoutes(learningGroup)

	return engine
}
