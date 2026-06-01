package bootstrap

import (
	frontierrepo "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/repository"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	recommendmodule "github.com/han139242/course-recommend-agents/backend/internal/module/recommend"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// NewRecommendModule initializes the recommend module with KT-KGPath scorer.
func NewRecommendModule(driver neo4j.DriverWithContext, redisClient *redis.Client, db *gorm.DB) *recommendmodule.Module {
	graphRepository := graphrepo.NewNeo4jRepository(driver)
	graphService := graphservice.New(graphRepository)
	frontierRepo := frontierrepo.New(graphRepository)
	return recommendmodule.New(graphService, redisClient, db, frontierRepo)
}
