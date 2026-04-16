package bootstrap

import (
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	assistantmodule "github.com/han139242/course-recommend-agents/backend/internal/module/assistant"
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	competencyservice "github.com/han139242/course-recommend-agents/backend/internal/module/competency/service"
	dashboardservice "github.com/han139242/course-recommend-agents/backend/internal/module/dashboard/service"
	frontierservice "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/service"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	learningservice "github.com/han139242/course-recommend-agents/backend/internal/module/learning/service"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	"gorm.io/gorm"
)

// NewAssistantModule creates the authenticated project assistant module.
func NewAssistantModule(cfg config.Config, authService *authservice.AuthService, learningService *learningservice.Service, competencyService *competencyservice.Service, recommendService *recommendservice.Service, dashboardService *dashboardservice.Service, graphService *graphservice.Service, frontierService *frontierservice.Service, db *gorm.DB) *assistantmodule.Module {
	return assistantmodule.New(cfg.AI, authService, learningService, competencyService, recommendService, dashboardService, graphService, frontierService, db)
}
