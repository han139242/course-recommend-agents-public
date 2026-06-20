package learning

import (
	"github.com/gin-gonic/gin"
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	competencyservice "github.com/han139242/course-recommend-agents/backend/internal/module/competency/service"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
	learninghandler "github.com/han139242/course-recommend-agents/backend/internal/module/learning/handler"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	learningservice "github.com/han139242/course-recommend-agents/backend/internal/module/learning/service"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 聚合学习模块依赖。
type Module struct {
	Handler *learninghandler.Handler
	Service *learningservice.Service
}

// New 创建学习模块。
func New(db *gorm.DB, redisClient *redis.Client, authService *authservice.AuthService, graphWriter *graphwriterservice.Service, graphService ...*graphservice.Service) *Module {
	repo := learningrepo.NewMySQLRepository(db)
	var service *learningservice.Service
	if len(graphService) > 0 && graphService[0] != nil {
		service = learningservice.New(repo, redisClient, authService, graphWriter, graphService[0])
	} else {
		service = learningservice.New(repo, redisClient, authService, graphWriter)
	}
	return &Module{Handler: learninghandler.New(service), Service: service}
}

// SetCompetencyService 注入能力画像聚合服务。
func (m *Module) SetCompetencyService(service *competencyservice.Service) {
	if m == nil || m.Service == nil {
		return
	}
	m.Service.SetCompetencyService(service)
}

// RegisterTeacherRoutes 注册教师题库路由。
// 该分组只暴露题库维护能力，具体的教师身份与启用状态仍由 service 层做数据库实时校验。
func (m *Module) RegisterTeacherRoutes(group *gin.RouterGroup) {
	group.POST("/questions/batch", m.Handler.ImportQuestions)
	group.GET("/questions", m.Handler.ListQuestions)
	group.GET("/questions/:questionId", m.Handler.GetQuestionDetail)
}

// RegisterStudentRoutes 注册学生学习路由。
func (m *Module) RegisterStudentRoutes(group *gin.RouterGroup) {
	group.GET("/questions", m.Handler.ListStudentQuestions)
	group.POST("/answers", m.Handler.SubmitAnswer)
	group.POST("/practice", m.Handler.RecordPractice)
	group.POST("/quiz-submit", m.Handler.SubmitQuiz)
	group.GET("/progress", m.Handler.GetLearningProgress)
}
