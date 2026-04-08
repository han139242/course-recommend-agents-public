package dashboard

import (
	"github.com/gin-gonic/gin"
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	competencyservice "github.com/han139242/course-recommend-agents/backend/internal/module/competency/service"
	dashboardhandler "github.com/han139242/course-recommend-agents/backend/internal/module/dashboard/handler"
	dashboardservice "github.com/han139242/course-recommend-agents/backend/internal/module/dashboard/service"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
)

// Module 聚合仪表盘模块依赖。
type Module struct {
	Handler *dashboardhandler.Handler
	Service *dashboardservice.Service
}

// New 创建仪表盘模块。
func New(competencyService *competencyservice.Service, recommendService *recommendservice.Service, authService *authservice.AuthService) *Module {
	service := dashboardservice.New(competencyService, recommendService)
	return &Module{Handler: dashboardhandler.New(service, authService), Service: service}
}

// RegisterRoutes 注册仪表盘路由。
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/summary", m.Handler.GetSummary)
}
