package bootstrap

import (
	"context"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// NewNeo4j initializes Neo4j as an optional dependency.
func NewNeo4j(ctx context.Context, cfg config.Config, logger *applogger.Logger) (neo4j.DriverWithContext, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if cfg.Neo4j.Password == "" {
		logger.WithCtx(ctx).Warn("Neo4j password is empty, skipping initialization")
		return nil, nil
	}

	driver, err := neo4j.NewDriverWithContext(cfg.Neo4j.URI, neo4j.BasicAuth(cfg.Neo4j.Username, cfg.Neo4j.Password, ""))
	if err != nil {
		return nil, err
	}

	verifyCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := driver.VerifyConnectivity(verifyCtx); err != nil {
		logger.WithCtx(ctx).Warn("Neo4j unavailable, starting in degraded mode", applogger.ErrorField("error", err))
		_ = driver.Close(context.Background())
		return nil, nil
	}

	return driver, nil
}
