package bootstrap

import (
	"context"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// NewRedis initializes Redis as an optional dependency.
func NewRedis(ctx context.Context, cfg config.Config, logger *applogger.Logger) (*redis.Client, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		logger.WithCtx(ctx).Warn("Redis unavailable, starting in degraded mode", applogger.ErrorField("error", err))
		return nil, nil
	}

	return client, nil
}
