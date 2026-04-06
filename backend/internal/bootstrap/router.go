package bootstrap

import (
	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/router"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
)

// NewRouter 初始化路由。
func NewRouter(logger *applogger.Logger) *gin.Engine {
	return router.New(logger)
}
