package frontier

import (
	"github.com/gin-gonic/gin"
	frontierhandler "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/handler"
	frontierrepo "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/repository"
	frontierservice "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/service"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
)

// Module 聚合前沿扩展模块依赖。
type Module struct {
	Handler *frontierhandler.Handler
	Service *frontierservice.Service
}

// New 创建前沿扩展模块。
func New(graphRepository *graphrepo.Neo4jRepository) *Module {
	repo := frontierrepo.New(graphRepository)
	service := frontierservice.New(repo)
	return &Module{Handler: frontierhandler.New(service), Service: service}
}

// RegisterRoutes 注册前沿扩展路由。
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/briefing", m.Handler.GetBriefing)
}
