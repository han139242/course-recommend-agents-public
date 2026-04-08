package bootstrap

import (
	"github.com/gin-gonic/gin"
	authmodule "github.com/han139242/course-recommend-agents/backend/internal/module/auth"
	competencymodule "github.com/han139242/course-recommend-agents/backend/internal/module/competency"
	dashboardmodule "github.com/han139242/course-recommend-agents/backend/internal/module/dashboard"
	graphmodule "github.com/han139242/course-recommend-agents/backend/internal/module/graph"
	learningmodule "github.com/han139242/course-recommend-agents/backend/internal/module/learning"
	recommendmodule "github.com/han139242/course-recommend-agents/backend/internal/module/recommend"
	teachermodule "github.com/han139242/course-recommend-agents/backend/internal/module/teacher"
	"github.com/han139242/course-recommend-agents/backend/internal/router"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
)

// NewRouter 初始化路由。
// bootstrap 层通过该函数屏蔽具体路由实现，保持入口装配代码只关注依赖拼装。
func NewRouter(logger *applogger.Logger, authModule *authmodule.Module, teacherModule *teachermodule.Module, learningModule *learningmodule.Module, graphModule *graphmodule.Module, competencyModule *competencymodule.Module, recommendModule *recommendmodule.Module, dashboardModule *dashboardmodule.Module) *gin.Engine {
	return router.New(logger, authModule, teacherModule, learningModule, graphModule, competencyModule, recommendModule, dashboardModule)
}
