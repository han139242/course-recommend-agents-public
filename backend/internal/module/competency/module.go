package competency

import (
	"github.com/gin-gonic/gin"
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	competencyhandler "github.com/han139242/course-recommend-agents/backend/internal/module/competency/handler"
	competencyservice "github.com/han139242/course-recommend-agents/backend/internal/module/competency/service"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// Module 聚合能力画像模块依赖。
type Module struct {
	Handler *competencyhandler.Handler
	Service *competencyservice.Service
	Matrix  *competencyservice.MatrixManager
}

// New 创建能力画像模块。
func New(logger *applogger.Logger, redisClient *redis.Client, authService *authservice.AuthService, graphRepository *graphrepo.Neo4jRepository, graphWriter *graphwriterservice.Service, topologyChannel string) *Module {
	redisRepo := learningrepo.NewRedisRepository(redisClient)
	matrixManager := competencyservice.NewMatrixManager(logger, graphRepository, redisClient, topologyChannel)
	service := competencyservice.New(redisRepo, matrixManager, graphWriter)
	return &Module{Handler: competencyhandler.New(service, authService), Service: service, Matrix: matrixManager}
}

// RegisterRoutes 注册能力画像路由。
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/profile", m.Handler.GetProfile)
}
