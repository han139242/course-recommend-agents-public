package bootstrap

import (
	"github.com/han139242/course-recommend-agents/backend/internal/ai"
	profilemodule "github.com/han139242/course-recommend-agents/backend/internal/module/profile"
	"gorm.io/gorm"
)

// NewProfileModule initializes conversational learner profile extraction.
func NewProfileModule(aiClient *ai.Client, db *gorm.DB) *profilemodule.Module {
	return profilemodule.New(db, aiClient)
}
