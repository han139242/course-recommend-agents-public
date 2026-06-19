package profile

import (
	"github.com/gin-gonic/gin"
	profilehandler "github.com/han139242/course-recommend-agents/backend/internal/module/profile/handler"
	profilerepo "github.com/han139242/course-recommend-agents/backend/internal/module/profile/repository"
	profileservice "github.com/han139242/course-recommend-agents/backend/internal/module/profile/service"
	"gorm.io/gorm"
)

type Module struct {
	Handler *profilehandler.Handler
	Service *profileservice.Service
}

func New(db *gorm.DB) *Module {
	repo := profilerepo.New(db)
	service := profileservice.New(repo)
	return &Module{Handler: profilehandler.New(service), Service: service}
}

func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/extract", m.Handler.ExtractProfile)
	group.GET("/current", m.Handler.GetCurrent)
	group.PUT("/current", m.Handler.UpsertProfile)
	group.POST("/refresh", m.Handler.RefreshProfile)
}
