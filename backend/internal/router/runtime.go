package router

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	graphmodule "github.com/han139242/course-recommend-agents/backend/internal/module/graph"
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type readinessCheck struct {
	Required bool   `json:"required"`
	Ready    bool   `json:"ready"`
	Message  string `json:"message"`
}

func registerRuntimeRoutes(engine *gin.Engine, cfg config.Config, mysqlDB *gorm.DB, redisClient *redis.Client, neo4jDriver neo4j.DriverWithContext, graphWriter *graphwriterservice.Service, graphModule *graphmodule.Module) {
	engine.GET("/readyz", func(ctx *gin.Context) {
		checks := gin.H{}
		ready := true

		mysqlCheck := readinessCheck{Required: true, Ready: mysqlReady(mysqlDB), Message: "mysql"}
		checks["mysql"] = mysqlCheck
		ready = ready && mysqlCheck.Ready

		redisRequired := cfg.Redis.Addr != ""
		redisCheck := readinessCheck{Required: redisRequired, Ready: redisReady(ctx, redisClient), Message: "redis"}
		checks["redis"] = redisCheck
		if redisCheck.Required {
			ready = ready && redisCheck.Ready
		}

		neo4jRequired := cfg.Neo4j.Password != ""
		neo4jCheck := readinessCheck{Required: neo4jRequired, Ready: neo4jReady(ctx, neo4jDriver), Message: "neo4j"}
		checks["neo4j"] = neo4jCheck
		if neo4jCheck.Required {
			ready = ready && neo4jCheck.Ready
		}

		graphWriterCheck := readinessCheck{Required: cfg.GraphWriter.Enabled, Ready: true, Message: "graphwriter"}
		if graphWriter != nil && cfg.GraphWriter.Enabled {
			runtime := graphWriter.RuntimeStatus()
			checks["graphWriter"] = runtime
			graphWriterCheck.Ready = runtime.Started && !runtime.Stopped && runtime.WorkerActive
			if graphWriterCheck.Required {
				ready = ready && graphWriterCheck.Ready
			}
		} else {
			checks["graphWriter"] = graphWriterCheck
		}

		graphCheck := readinessCheck{Required: neo4jRequired, Ready: graphModule != nil && graphModule.Repo != nil && graphModule.Repo.Enabled(), Message: "graph"}
		checks["graph"] = graphCheck
		if graphCheck.Required {
			ready = ready && graphCheck.Ready
		}

		status := "ready"
		httpStatus := http.StatusOK
		if !ready {
			status = "not_ready"
			httpStatus = http.StatusServiceUnavailable
		}
		ctx.JSON(httpStatus, gin.H{
			"code":    0,
			"message": "ok",
			"data": gin.H{
				"status":       status,
				"buildVersion": cfg.Observability.BuildVersion,
				"checks":       checks,
			},
		})
	})
}

func mysqlReady(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	sqlDB, err := db.DB()
	if err != nil {
		return false
	}
	return sqlDB.Ping() == nil
}

func redisReady(ctx *gin.Context, client *redis.Client) bool {
	if client == nil {
		return false
	}
	return client.Ping(ctx.Request.Context()).Err() == nil
}

func neo4jReady(ctx *gin.Context, driver neo4j.DriverWithContext) bool {
	if driver == nil {
		return false
	}
	return driver.VerifyConnectivity(ctx.Request.Context()) == nil
}

func sqlDBOf(db *gorm.DB) *sql.DB {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil
	}
	return sqlDB
}
