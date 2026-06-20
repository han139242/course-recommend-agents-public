package resourcegen

import (
	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/ai"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	profileservice "github.com/han139242/course-recommend-agents/backend/internal/module/profile/service"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	resourcegenhandler "github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen/handler"
	resourcegenrepo "github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen/repository"
	resourcegenservice "github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen/service"
	"gorm.io/gorm"
)

type Module struct {
	Handler *resourcegenhandler.Handler
	Service *resourcegenservice.Service
}

func New(db *gorm.DB, aiClient *ai.Client, profileService *profileservice.Service, recommendService *recommendservice.Service, graphService *graphservice.Service) *Module {
	repo := resourcegenrepo.New(db)
	service := resourcegenservice.New(repo, aiClient, profileService, recommendService, graphService)
	return &Module{Handler: resourcegenhandler.New(service), Service: service}
}

func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/generate", m.Handler.Generate)
	group.GET("/jobs/:jobId/export", m.Handler.ExportJob)
	group.GET("/jobs/:jobId/presentation-outline", m.Handler.GetPresentationOutline)
	group.GET("/jobs/:jobId", m.Handler.GetJob)
	group.GET("/recommendations", m.Handler.GetRecommendations)
	group.GET("/feed", m.Handler.GetFeed)
	group.POST("/events", m.Handler.RecordLearningEvent)
	group.GET("/:resourceId/safety-report", m.Handler.GetSafetyReport)
	group.POST("/:resourceId/feedback", m.Handler.SubmitFeedback)
	group.GET("", m.Handler.ListResources)
	group.GET("/:resourceId", m.Handler.GetResource)
}
