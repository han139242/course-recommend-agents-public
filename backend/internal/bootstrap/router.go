package bootstrap

import (
	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/router"
)

// NewRouter 初始化路由。
func NewRouter() *gin.Engine {
	return router.New()
}
