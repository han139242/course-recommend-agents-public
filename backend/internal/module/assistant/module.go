package assistant

import (
	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	assistanthandler "github.com/han139242/course-recommend-agents/backend/internal/module/assistant/handler"
	assistantservice "github.com/han139242/course-recommend-agents/backend/internal/module/assistant/service"
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	competencyservice "github.com/han139242/course-recommend-agents/backend/internal/module/competency/service"
	dashboardservice "github.com/han139242/course-recommend-agents/backend/internal/module/dashboard/service"
	frontierservice "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/service"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	learningservice "github.com/han139242/course-recommend-agents/backend/internal/module/learning/service"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	"gorm.io/gorm"
)

// Module groups assistant dependencies.
type Module struct {
	Handler *assistanthandler.Handler
	Service *assistantservice.Service
}

// New creates the assistant module.
func New(aiCfg config.AIConfig, authService *authservice.AuthService, learningService *learningservice.Service, competencyService *competencyservice.Service, recommendService *recommendservice.Service, dashboardService *dashboardservice.Service, graphService *graphservice.Service, frontierService *frontierservice.Service, db *gorm.DB) *Module {
	service := assistantservice.New(aiCfg, authService, learningService, competencyService, recommendService, dashboardService, graphService, frontierService, db)
	return &Module{Handler: assistanthandler.New(service), Service: service}
}

// RegisterRoutes registers assistant routes.
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/chat", m.Handler.Chat)
	group.POST("/knowledge-card", m.Handler.GenerateKnowledgeCard)
}

// RegisterPublicRoutes registers unauthenticated assistant routes.
func (m *Module) RegisterPublicRoutes(group *gin.RouterGroup) {
	group.POST("/public-chat", m.Handler.PublicChat)
}
