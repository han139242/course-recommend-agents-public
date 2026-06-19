package bootstrap

import (
	"github.com/han139242/course-recommend-agents/backend/internal/ai"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	profileservice "github.com/han139242/course-recommend-agents/backend/internal/module/profile/service"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	resourcegenmodule "github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen"
	"gorm.io/gorm"
)

// NewResourceGenModule initializes personalized multi-agent resource generation.
func NewResourceGenModule(aiClient *ai.Client, db *gorm.DB, profileService *profileservice.Service, recommendService *recommendservice.Service, graphService *graphservice.Service) *resourcegenmodule.Module {
	return resourcegenmodule.New(db, aiClient, profileService, recommendService, graphService)
}
