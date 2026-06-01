package recommend

import (
	"github.com/gin-gonic/gin"
	frontierrepo "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/repository"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	recommendhandler "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/handler"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 聚合推荐模块依赖。
type Module struct {
	Handler *recommendhandler.Handler
	Service *recommendservice.Service
}

// New 创建推荐模块。
func New(graphService *graphservice.Service, redisClient *redis.Client, db *gorm.DB, frontierRepo *frontierrepo.Repository) *Module {
	redisRepo := learningrepo.NewRedisRepository(redisClient)
	stateRepo := learningrepo.NewStateMySQLRepository(db)
	scorer := recommendservice.NewScorer(recommendservice.DefaultScorerConfig(), graphService, frontierRepo)
	service := recommendservice.New(graphService, redisRepo, stateRepo, scorer)
	return &Module{Handler: recommendhandler.New(service), Service: service}
}

// RegisterRoutes 注册推荐路由。
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/path", m.Handler.GetUnifiedPath)
	group.GET("/unified-path", m.Handler.GetUnifiedPath)
	group.GET("/legacy-path", m.Handler.GetPath)
}
