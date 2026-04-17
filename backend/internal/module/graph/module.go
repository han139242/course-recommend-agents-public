package graph

import (
	"github.com/gin-gonic/gin"
	graphhandler "github.com/han139242/course-recommend-agents/backend/internal/module/graph/handler"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Module groups graph dependencies.
type Module struct {
	Handler *graphhandler.Handler
	Service *graphservice.Service
	Repo    *graphrepo.Neo4jRepository
	Driver  neo4j.DriverWithContext
}

// New creates a graph module.
func New(driver neo4j.DriverWithContext) *Module {
	repo := graphrepo.NewNeo4jRepository(driver)
	service := graphservice.New(repo)
	return &Module{
		Handler: graphhandler.New(service),
		Service: service,
		Repo:    repo,
		Driver:  driver,
	}
}

// RegisterRoutes registers graph routes.
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/catalog", m.Handler.GetCatalog)
	group.GET("/seeds", m.Handler.GetSeeds)
	group.GET("/neighbors", m.Handler.GetNeighbors)
	group.GET("/path", m.Handler.GetPath)
	group.GET("/subgraph", m.Handler.GetSubgraph)
	group.GET("/full", m.Handler.GetFullGraph)
}

// RegisterKnowledgeRoutes registers knowledge detail routes.
func (m *Module) RegisterKnowledgeRoutes(group *gin.RouterGroup) {
	group.GET(":knowledgeId", m.Handler.GetKnowledgeDetail)
}
