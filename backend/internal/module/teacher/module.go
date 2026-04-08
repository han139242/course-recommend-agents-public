package teacher

import (
	"github.com/gin-gonic/gin"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	learningservice "github.com/han139242/course-recommend-agents/backend/internal/module/learning/service"
	teacherhandler "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/handler"
	teacherrepo "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/repository"
	teacherservice "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/service"
	sharedpassword "github.com/han139242/course-recommend-agents/backend/pkg/password"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 聚合教师端模块依赖。
type Module struct {
	Handler *teacherhandler.Handler
}

// New 创建教师端模块。
// 教师模块通过共享密码组件生成学生初始密码，避免依赖认证模块内部实现。
func New(db *gorm.DB, redisClient *redis.Client) *Module {
	repo := teacherrepo.NewMySQLRepository(db)
	passwordService := sharedpassword.NewService()
	stateReader := learningservice.NewStateService(learningrepo.NewRedisRepository(redisClient), learningrepo.NewStateMySQLRepository(db))
	service := teacherservice.New(repo, passwordService, stateReader)
	return &Module{Handler: teacherhandler.New(service)}
}

// RegisterRoutes 注册教师端路由。
// 该分组默认运行在教师鉴权中间件之后，因此这里只暴露教师可见的班级与学生管理接口，
// 不在路由层重复声明角色判断，统一交由 service 层做实时权限校验。
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/classes", m.Handler.CreateClassroom)
	group.GET("/classes", m.Handler.ListClassrooms)
	group.POST("/classes/:classId/students", m.Handler.BatchCreateStudents)
	group.GET("/classes/:classId/students", m.Handler.ListStudents)
	group.GET("/students/:studentId", m.Handler.GetStudentDetail)
	group.GET("/students/:studentId/knowledge-status", m.Handler.GetStudentKnowledgeStatus)
}
