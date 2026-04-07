package bootstrap

import (
	"fmt"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	authmodule "github.com/han139242/course-recommend-agents/backend/internal/module/auth"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"gorm.io/gorm"
)

// NewAuthModule 初始化认证模块。
func NewAuthModule(cfg config.Config, db *gorm.DB, logger *applogger.Logger) (*authmodule.Module, error) {
	module, err := authmodule.New(cfg.Auth, db, logger)
	if err != nil {
		return nil, fmt.Errorf("初始化认证模块失败: %w", err)
	}
	return module, nil
}
