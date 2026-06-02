package bootstrap

import (
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	studyplanmodule "github.com/han139242/course-recommend-agents/backend/internal/module/studyplan"
	"gorm.io/gorm"
)

// NewStudyPlanModule initializes goal profile and study plan orchestration.
func NewStudyPlanModule(db *gorm.DB, recommendService *recommendservice.Service) *studyplanmodule.Module {
	return studyplanmodule.New(db, recommendService, learningrepo.NewStateMySQLRepository(db))
}
