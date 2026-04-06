package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/handler"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/repository"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
)

// Module 聚合认证模块运行期依赖。
type Module struct {
	Handler    *handler.Handler
	JWTService *service.JWTService
}

// New 创建认证模块。
func New(cfg config.AuthConfig, _ *applogger.Logger) *Module {
	repo := repository.NewMemoryUserRepository()
	jwtService := service.NewJWTService(cfg)
	passwordService := service.NewPasswordService()
	authService := service.NewAuthService(repo, jwtService, passwordService)
	return &Module{
		Handler:    handler.New(authService),
		JWTService: jwtService,
	}
}

// RegisterPublicRoutes 注册公开认证路由。
func (m *Module) RegisterPublicRoutes(group *gin.RouterGroup) {
	group.POST("/login", m.Handler.Login)
}

// RegisterAuthRoutes 注册需要登录的认证路由。
func (m *Module) RegisterAuthRoutes(group *gin.RouterGroup) {
	group.GET("/me", m.Handler.Me)
}
