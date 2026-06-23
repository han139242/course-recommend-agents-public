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
	group.POST("/quiz", m.Handler.GenerateQuiz)
	group.POST("/media/image", m.Handler.GenerateImage)
	group.POST("/media/video", m.Handler.GenerateVideo)
	group.GET("/jobs/:jobId/export", m.Handler.ExportJob)
	group.GET("/jobs/:jobId/review-summary", m.Handler.GetJobReviewSummary)
	group.GET("/jobs/:jobId/evidence-package", m.Handler.GetJobEvidencePackage)
	group.GET("/jobs/:jobId/study-checklist", m.Handler.GetStudyChecklist)
	group.GET("/jobs/:jobId/presentation-outline", m.Handler.GetPresentationOutline)
	group.GET("/jobs/:jobId/demo-script", m.Handler.GetDemoScript)
	group.GET("/jobs/:jobId/stream", m.Handler.StreamJob)
	group.POST("/jobs/:jobId/retry", m.Handler.RetryJob)
	group.GET("/jobs/:jobId", m.Handler.GetJob)
	group.POST("/compare", m.Handler.CompareResources)
	group.GET("/recommendations", m.Handler.GetRecommendations)
	group.GET("/effect-summary", m.Handler.GetEffectSummary)
	group.GET("/learning-report", m.Handler.GetLearningReport)
	group.GET("/feed", m.Handler.GetFeed)
	group.POST("/events", m.Handler.RecordLearningEvent)
	group.GET("/:resourceId/media-status", m.Handler.GetMediaStatus)
	group.GET("/:resourceId/quality-report", m.Handler.GetQualityReport)
	group.GET("/:resourceId/provenance", m.Handler.GetProvenance)
	group.GET("/:resourceId/safety-report", m.Handler.GetSafetyReport)
	group.POST("/:resourceId/variant", m.Handler.GenerateVariant)
	group.POST("/:resourceId/feedback", m.Handler.SubmitFeedback)
	group.GET("", m.Handler.ListResources)
	group.GET("/:resourceId", m.Handler.GetResource)
}
