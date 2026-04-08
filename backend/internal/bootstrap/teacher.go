package bootstrap

import (
	teachermodule "github.com/han139242/course-recommend-agents/backend/internal/module/teacher"
	"gorm.io/gorm"
)

// NewTeacherModule 初始化教师端模块。
func NewTeacherModule(db *gorm.DB) *teachermodule.Module {
	return teachermodule.New(db)
}
