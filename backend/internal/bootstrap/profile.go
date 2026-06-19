package bootstrap

import (
	profilemodule "github.com/han139242/course-recommend-agents/backend/internal/module/profile"
	"gorm.io/gorm"
)

// NewProfileModule initializes conversational learner profile extraction.
func NewProfileModule(db *gorm.DB) *profilemodule.Module {
	return profilemodule.New(db)
}
