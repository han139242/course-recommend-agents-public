package bootstrap

import (
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	competencymodule "github.com/han139242/course-recommend-agents/backend/internal/module/competency"
	"github.com/redis/go-redis/v9"
)

// NewCompetencyModule 初始化能力画像模块。
func NewCompetencyModule(redisClient *redis.Client, authService *authservice.AuthService) *competencymodule.Module {
	return competencymodule.New(redisClient, authService)
}
