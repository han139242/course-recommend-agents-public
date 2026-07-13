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

// Module 聚合认证模块对外暴露的运行期组件。
//
// 该结构体用于把路由层和鉴权基础能力一起注册到应用中，
// 这样上层只需要依赖模块本身，而不需要了解仓储、密码校验、JWT 等内部装配细节。
type Module struct {
	Handler     *handler.Handler
	JWTService  *service.JWTService
	AuthService *service.AuthService
}

// New 创建认证模块。
//
// 这里要求数据库连接已经可用，因为当前认证数据来源于 MySQL。
// 若在启动阶段放行空连接，问题会延迟到首次登录请求时才暴露，排查成本更高。
func New(appCfg config.AppConfig, cfg config.AuthConfig, db *gorm.DB, _ *applogger.Logger) (*Module, error) {
	if db == nil {
		return nil, fmt.Errorf("认证模块初始化失败：MySQL 未连接")
	}

	jwtService := service.NewJWTService(cfg, appCfg)
	passwordService := service.NewPasswordService()
	userRepo := repository.NewMySQLUserRepository(db)
	authService := service.NewAuthService(userRepo, jwtService, passwordService)
	return &Module{
		Handler:     handler.New(authService),
		JWTService:  jwtService,
		AuthService: authService,
	}, nil
}

// RegisterPublicRoutes 注册公开认证路由。
//
// 这里仅挂载无需预先携带凭证的入口，避免把登录接口错误地放入鉴权链路后形成死循环。
func (m *Module) RegisterPublicRoutes(group *gin.RouterGroup) {
	group.POST("/login", m.Handler.Login)
	group.POST("/register", m.Handler.Register)
}

// RegisterAuthRoutes 注册需要登录的认证路由。
//
// 该分组中的接口默认依赖外层鉴权中间件提前解析用户身份，因此这里不再重复校验令牌。
func (m *Module) RegisterAuthRoutes(group *gin.RouterGroup) {
	group.GET("/me", m.Handler.Me)
}
