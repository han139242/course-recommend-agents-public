package bootstrap

import (
	"context"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewMySQL 初始化 MySQL 连接；未配置时跳过连接。
func NewMySQL(cfg config.Config, logger *applogger.Logger) (*gorm.DB, error) {
	if cfg.MySQL.DSN == "" {
		logger.WithCtx(context.Background()).Warn("MySQL 未配置，跳过连接初始化")
		return nil, nil
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
