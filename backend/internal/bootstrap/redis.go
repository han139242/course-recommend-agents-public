package bootstrap

import (
	"context"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// NewRedis 初始化 Redis 客户端。
func NewRedis(ctx context.Context, cfg config.Config, logger *applogger.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		logger.WithCtx(ctx).Warn("Redis 连接失败，当前以降级模式启动", applogger.ErrorField("error", err))
		return nil, nil
	}

	return client, nil
}
