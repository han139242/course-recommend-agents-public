package bootstrap

import (
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	authmodule "github.com/han139242/course-recommend-agents/backend/internal/module/auth"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
)

// NewAuthModule 初始化认证模块。
func NewAuthModule(cfg config.Config, logger *applogger.Logger) *authmodule.Module {
	return authmodule.New(cfg.Auth, logger)
}
