package bootstrap

import (
	teachermodule "github.com/han139242/course-recommend-agents/backend/internal/module/teacher"
	"gorm.io/gorm"
)

// NewTeacherModule 初始化教师端模块。
// 教师端模块当前只依赖 MySQL，因此保持轻量装配，避免过早引入额外抽象。
func NewTeacherModule(db *gorm.DB) *teachermodule.Module {
	return teachermodule.New(db)
}
