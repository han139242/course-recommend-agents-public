package router

import (
	"net/http/pprof"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	"github.com/han139242/course-recommend-agents/backend/internal/middleware"
	authmodule "github.com/han139242/course-recommend-agents/backend/internal/module/auth"
	competencymodule "github.com/han139242/course-recommend-agents/backend/internal/module/competency"
	dashboardmodule "github.com/han139242/course-recommend-agents/backend/internal/module/dashboard"
	frontiermodule "github.com/han139242/course-recommend-agents/backend/internal/module/frontier"
	graphmodule "github.com/han139242/course-recommend-agents/backend/internal/module/graph"
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
	learningmodule "github.com/han139242/course-recommend-agents/backend/internal/module/learning"
	recommendmodule "github.com/han139242/course-recommend-agents/backend/internal/module/recommend"
	teachermodule "github.com/han139242/course-recommend-agents/backend/internal/module/teacher"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// New 创建 Gin 路由实例。
//
// 路由层只负责统一挂载中间件、健康检查与模块路由，
// 不在此处处理业务编排，避免入口层承担过多职责。
func New(logger *applogger.Logger, cfg config.Config, mysqlDB *gorm.DB, redisClient *redis.Client, neo4jDriver neo4j.DriverWithContext, authModule *authmodule.Module, teacherModule *teachermodule.Module, learningModule *learningmodule.Module, graphModule *graphmodule.Module, graphWriter *graphwriterservice.Service, competencyModule *competencymodule.Module, recommendModule *recommendmodule.Module, dashboardModule *dashboardmodule.Module, frontierModule *frontiermodule.Module) *gin.Engine {
	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.RealIP())
	engine.Use(middleware.Logger(logger))
	engine.Use(middleware.Recovery(logger))

	engine.GET("/healthz", func(ctx *gin.Context) {
		result.Success(ctx, gin.H{
			"status":       "ok",
			"buildVersion": cfg.Observability.BuildVersion,
			"buildCommit":  cfg.Observability.BuildCommit,
			"buildTime":    cfg.Observability.BuildTime,
		})
	})

	registerRuntimeRoutes(engine, cfg, mysqlDB, redisClient, neo4jDriver, graphWriter, graphModule)

	api := engine.Group("/api")

	// publicGroup 只注册无需登录即可访问的接口，便于探活和登录前流程复用。
	publicGroup := api.Group("")
	publicGroup.GET("/ping", func(ctx *gin.Context) {
		result.Success(ctx, gin.H{
			"message": "pong",
		})
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
	competencyGroup := authGroup.Group("/competency")
	competencyModule.RegisterRoutes(competencyGroup)
	recommendGroup := authGroup.Group("/recommend")
	recommendModule.RegisterRoutes(recommendGroup)
	dashboardGroup := authGroup.Group("/dashboard")
	dashboardModule.RegisterRoutes(dashboardGroup)
	frontierGroup := api.Group("/frontier")
	frontierModule.RegisterRoutes(frontierGroup)
	graphGroup := api.Group("/graph")
	graphModule.RegisterRoutes(graphGroup)
	knowledgeGroup := api.Group("/knowledge")
	graphModule.RegisterKnowledgeRoutes(knowledgeGroup)
	if cfg.Observability.EnablePprof && cfg.App.Env != "production" {
		pprofGroup := engine.Group("/debug/pprof")
		pprofGroup.GET("/", gin.WrapF(pprof.Index))
		pprofGroup.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		pprofGroup.GET("/profile", gin.WrapF(pprof.Profile))
		pprofGroup.GET("/symbol", gin.WrapF(pprof.Symbol))
		pprofGroup.GET("/trace", gin.WrapF(pprof.Trace))
	}

	registerFrontendIntegrationRoutes(engine, cfg)

	return engine
}
