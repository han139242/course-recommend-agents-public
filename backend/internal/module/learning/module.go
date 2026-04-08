package learning

import (
	"github.com/gin-gonic/gin"
	learninghandler "github.com/han139242/course-recommend-agents/backend/internal/module/learning/handler"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	learningservice "github.com/han139242/course-recommend-agents/backend/internal/module/learning/service"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 聚合学习模块依赖。
type Module struct {
	Handler *learninghandler.Handler
}

// New 创建学习模块。
func New(db *gorm.DB, redisClient *redis.Client) *Module {
	repo := learningrepo.NewMySQLRepository(db)
	service := learningservice.New(repo, redisClient)
	return &Module{Handler: learninghandler.New(service)}
}

// RegisterTeacherRoutes 注册教师题库路由。
// 该分组只暴露题库维护能力，具体的教师身份与启用状态仍由 service 层做数据库实时校验。
func (m *Module) RegisterTeacherRoutes(group *gin.RouterGroup) {
	group.POST("/questions/batch", m.Handler.ImportQuestions)
	group.GET("/questions", m.Handler.ListQuestions)
	group.GET("/questions/:questionId", m.Handler.GetQuestionDetail)
}

// RegisterStudentRoutes 注册学生学习路由。
// 学生侧接口只暴露拉题与答题提交，不返回题目答案等教师可见信息，避免路由层误共享视图模型。
func (m *Module) RegisterStudentRoutes(group *gin.RouterGroup) {
	group.GET("/questions", m.Handler.ListStudentQuestions)
	group.POST("/answers", m.Handler.SubmitAnswer)
}
