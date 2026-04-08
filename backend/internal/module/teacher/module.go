package teacher

import (
	"github.com/gin-gonic/gin"
	teacherhandler "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/handler"
	teacherrepo "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/repository"
	teacherservice "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/service"
	sharedpassword "github.com/han139242/course-recommend-agents/backend/pkg/password"
	"gorm.io/gorm"
)

// Module 聚合教师端模块依赖。
type Module struct {
	Handler *teacherhandler.Handler
}

// New 创建教师端模块。
// 教师模块通过共享密码组件生成学生初始密码，避免依赖认证模块内部实现。
func New(db *gorm.DB) *Module {
	repo := teacherrepo.NewMySQLRepository(db)
	passwordService := sharedpassword.NewService()
	service := teacherservice.New(repo, passwordService)
	return &Module{Handler: teacherhandler.New(service)}
}

// RegisterRoutes 注册教师端路由。
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/classes", m.Handler.CreateClassroom)
	group.GET("/classes", m.Handler.ListClassrooms)
	group.POST("/classes/:classId/students", m.Handler.BatchCreateStudents)
	group.GET("/classes/:classId/students", m.Handler.ListStudents)
	group.GET("/students/:studentId", m.Handler.GetStudentDetail)
}
