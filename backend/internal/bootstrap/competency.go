package bootstrap

import (
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	competencymodule "github.com/han139242/course-recommend-agents/backend/internal/module/competency"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// NewCompetencyModule 初始化能力画像模块。
func NewCompetencyModule(logger *applogger.Logger, redisClient *redis.Client, authService *authservice.AuthService, graphRepository *graphrepo.Neo4jRepository, graphWriter *graphwriterservice.Service, topologyChannel string) *competencymodule.Module {
	return competencymodule.New(logger, redisClient, authService, graphRepository, graphWriter, topologyChannel)
}
