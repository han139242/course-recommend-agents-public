package graph

import (
	"github.com/gin-gonic/gin"
	graphhandler "github.com/han139242/course-recommend-agents/backend/internal/module/graph/handler"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Module 聚合图谱模块依赖。
type Module struct {
	Handler *graphhandler.Handler
	Service *graphservice.Service
	Repo    *graphrepo.Neo4jRepository
	Driver  neo4j.DriverWithContext
}

// New 创建图谱模块。
func New(driver neo4j.DriverWithContext) *Module {
	repo := graphrepo.NewNeo4jRepository(driver)
	service := graphservice.New(repo)
	return &Module{Handler: graphhandler.New(service), Service: service, Repo: repo, Driver: driver}
}

// RegisterRoutes 注册图谱路由。
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/subgraph", m.Handler.GetSubgraph)
}

// RegisterKnowledgeRoutes 注册知识点详情路由。
func (m *Module) RegisterKnowledgeRoutes(group *gin.RouterGroup) {
	group.GET(":knowledgeId", m.Handler.GetKnowledgeDetail)
}
