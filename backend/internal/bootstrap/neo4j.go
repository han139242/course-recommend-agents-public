package bootstrap

import (
	"context"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// NewNeo4j 初始化 Neo4j Driver；认证信息缺失时跳过。
func NewNeo4j(ctx context.Context, cfg config.Config, logger *applogger.Logger) (neo4j.DriverWithContext, error) {
	if cfg.Neo4j.Password == "" {
		logger.WithCtx(ctx).Warn("Neo4j 密码未配置，跳过连接初始化")
		return nil, nil
	}

	driver, err := neo4j.NewDriverWithContext(cfg.Neo4j.URI, neo4j.BasicAuth(cfg.Neo4j.Username, cfg.Neo4j.Password, ""))
	if err != nil {
		return nil, err
	}

	if err := driver.VerifyConnectivity(ctx); err != nil {
		logger.WithCtx(ctx).Warn("Neo4j 连接失败，当前以降级模式启动", applogger.ErrorField("error", err))
		_ = driver.Close(ctx)
		return nil, nil
	}

	return driver, nil
}
