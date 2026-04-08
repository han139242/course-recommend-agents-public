package bootstrap

import (
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
	recommendmodule "github.com/han139242/course-recommend-agents/backend/internal/module/recommend"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
)

// NewRecommendModule 初始化推荐模块。
func NewRecommendModule(driver neo4j.DriverWithContext, redisClient *redis.Client) *recommendmodule.Module {
	graphRepository := graphrepo.NewNeo4jRepository(driver)
	graphService := graphservice.New(graphRepository)
	return recommendmodule.New(graphService, redisClient)
}
