package evaluation

import (
	"github.com/gin-gonic/gin"
	evhandler "github.com/han139242/course-recommend-agents/backend/internal/module/evaluation/handler"
	evsvc "github.com/han139242/course-recommend-agents/backend/internal/module/evaluation/service"
)

// Module 聚合评测模块依赖。
type Module struct {
	Handler *evhandler.Handler
	Service *evsvc.Service
}

// New 创建评测模块。
func New(service *evsvc.Service) *Module {
	return &Module{
		Handler: evhandler.New(service),
		Service: service,
	}
}

// RegisterRoutes 注册评测路由。
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/run", m.Handler.RunBenchmark)
}
