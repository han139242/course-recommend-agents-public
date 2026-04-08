package recommend

import (
	"github.com/gin-gonic/gin"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	recommendhandler "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/handler"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	"github.com/redis/go-redis/v9"
)

// Module 聚合推荐模块依赖。
type Module struct {
	Handler *recommendhandler.Handler
	Service *recommendservice.Service
}

// New 创建推荐模块。
func New(graphService *graphservice.Service, redisClient *redis.Client) *Module {
	redisRepo := learningrepo.NewRedisRepository(redisClient)
	service := recommendservice.New(graphService, redisRepo)
	return &Module{Handler: recommendhandler.New(service), Service: service}
}

// RegisterRoutes 注册推荐路由。
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/path", m.Handler.GetPath)
}
