package competency

import (
	"github.com/gin-gonic/gin"
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	competencyhandler "github.com/han139242/course-recommend-agents/backend/internal/module/competency/handler"
	competencyservice "github.com/han139242/course-recommend-agents/backend/internal/module/competency/service"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	"github.com/redis/go-redis/v9"
)

// Module 聚合能力画像模块依赖。
type Module struct {
	Handler *competencyhandler.Handler
	Service *competencyservice.Service
}

// New 创建能力画像模块。
func New(redisClient *redis.Client, authService *authservice.AuthService) *Module {
	redisRepo := learningrepo.NewRedisRepository(redisClient)
	service := competencyservice.New(redisRepo)
	return &Module{Handler: competencyhandler.New(service, authService), Service: service}
}

// RegisterRoutes 注册能力画像路由。
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/profile", m.Handler.GetProfile)
}
