package auth

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/handler"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/repository"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"gorm.io/gorm"
)

// Module 聚合认证模块运行期依赖。
type Module struct {
	Handler    *handler.Handler
	JWTService *service.JWTService
}

// New 创建认证模块。
func New(appCfg config.AppConfig, cfg config.AuthConfig, db *gorm.DB, _ *applogger.Logger) (*Module, error) {
	if db == nil {
		return nil, fmt.Errorf("认证模块初始化失败：MySQL 未连接")
	}

	jwtService := service.NewJWTService(cfg, appCfg)
	passwordService := service.NewPasswordService()
	userRepo := repository.NewMySQLUserRepository(db)
	authService := service.NewAuthService(userRepo, jwtService, passwordService)
	return &Module{
		Handler:    handler.New(authService),
		JWTService: jwtService,
	}, nil
}

// RegisterPublicRoutes 注册公开认证路由。
func (m *Module) RegisterPublicRoutes(group *gin.RouterGroup) {
	group.POST("/login", m.Handler.Login)
}

// RegisterAuthRoutes 注册需要登录的认证路由。
func (m *Module) RegisterAuthRoutes(group *gin.RouterGroup) {
	group.GET("/me", m.Handler.Me)
}
