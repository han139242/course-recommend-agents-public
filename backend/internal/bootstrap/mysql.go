package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewMySQL initializes the required MySQL connection.
func NewMySQL(cfg config.Config, logger *applogger.Logger) (*gorm.DB, error) {
	if cfg.MySQL.DSN == "" {
		logger.WithCtx(context.Background()).Error("MySQL is not configured; backend cannot start")
		return nil, fmt.Errorf("MYSQL_DSN cannot be empty")
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, err
	}

	return db, nil
}
