package studyplan

import (
	"github.com/gin-gonic/gin"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	studyplanhandler "github.com/han139242/course-recommend-agents/backend/internal/module/studyplan/handler"
	studyplanrepo "github.com/han139242/course-recommend-agents/backend/internal/module/studyplan/repository"
	studyplanservice "github.com/han139242/course-recommend-agents/backend/internal/module/studyplan/service"
	"gorm.io/gorm"
)

type Module struct {
	Handler *studyplanhandler.Handler
	Service *studyplanservice.Service
}

func New(db *gorm.DB, recommendService *recommendservice.Service, stateRepo learningrepo.StateRepository) *Module {
	repo := studyplanrepo.New(db)
	service := studyplanservice.New(repo, recommendService, stateRepo)
	return &Module{Handler: studyplanhandler.New(service), Service: service}
}

func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/profile", m.Handler.GetProfile)
	group.PUT("/profile", m.Handler.UpsertProfile)
	group.GET("/current", m.Handler.GetCurrentPlan)
	group.POST("/generate", m.Handler.GeneratePlan)
}
