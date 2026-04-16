package bootstrap

import (
	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	assistantmodule "github.com/han139242/course-recommend-agents/backend/internal/module/assistant"
	authmodule "github.com/han139242/course-recommend-agents/backend/internal/module/auth"
	competencymodule "github.com/han139242/course-recommend-agents/backend/internal/module/competency"
	dashboardmodule "github.com/han139242/course-recommend-agents/backend/internal/module/dashboard"
	frontiermodule "github.com/han139242/course-recommend-agents/backend/internal/module/frontier"
	graphmodule "github.com/han139242/course-recommend-agents/backend/internal/module/graph"
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
	learningmodule "github.com/han139242/course-recommend-agents/backend/internal/module/learning"
	recommendmodule "github.com/han139242/course-recommend-agents/backend/internal/module/recommend"
	teachermodule "github.com/han139242/course-recommend-agents/backend/internal/module/teacher"
	"github.com/han139242/course-recommend-agents/backend/internal/router"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// NewRouter 初始化路由。
// bootstrap 层通过该函数屏蔽具体路由实现，保持入口装配代码只关注依赖拼装。
func NewRouter(logger *applogger.Logger, cfg config.Config, mysqlDB *gorm.DB, redisClient *redis.Client, neo4jDriver neo4j.DriverWithContext, authModule *authmodule.Module, teacherModule *teachermodule.Module, learningModule *learningmodule.Module, graphModule *graphmodule.Module, graphWriter *graphwriterservice.Service, competencyModule *competencymodule.Module, recommendModule *recommendmodule.Module, dashboardModule *dashboardmodule.Module, frontierModule *frontiermodule.Module, assistantModule *assistantmodule.Module) *gin.Engine {
	return router.New(logger, cfg, mysqlDB, redisClient, neo4jDriver, authModule, teacherModule, learningModule, graphModule, graphWriter, competencyModule, recommendModule, dashboardModule, frontierModule, assistantModule)
}
