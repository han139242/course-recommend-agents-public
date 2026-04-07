package bootstrap

import (
	"context"
	"fmt"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewMySQL 初始化 MySQL 连接。
func NewMySQL(cfg config.Config, logger *applogger.Logger) (*gorm.DB, error) {
	if cfg.MySQL.DSN == "" {
		logger.WithCtx(context.Background()).Error("MySQL 未配置，服务无法启动")
		return nil, fmt.Errorf("MYSQL_DSN 不能为空")
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
