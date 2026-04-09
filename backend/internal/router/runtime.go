package router

import (
	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

func registerRuntimeRoutes(engine *gin.Engine, cfg config.Config, graphWriter *graphwriterservice.Service) {
	engine.GET("/readyz", func(ctx *gin.Context) {
		status := "ready"
		checks := gin.H{}
		if graphWriter != nil && cfg.GraphWriter.Enabled {
			runtime := graphWriter.RuntimeStatus()
			checks["graphWriter"] = runtime
			if !runtime.Started || runtime.Stopped || !runtime.WorkerActive {
				status = "degraded"
			}
		} else {
			checks["graphWriter"] = gin.H{"enabled": false}
		}
		result.Success(ctx, gin.H{
			"status":       status,
			"buildVersion": cfg.Observability.BuildVersion,
			"checks":       checks,
		})
	})
}
