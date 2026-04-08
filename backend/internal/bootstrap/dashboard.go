package bootstrap

import (
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	competencyservice "github.com/han139242/course-recommend-agents/backend/internal/module/competency/service"
	dashboardmodule "github.com/han139242/course-recommend-agents/backend/internal/module/dashboard"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
)

// NewDashboardModule 初始化仪表盘模块。
func NewDashboardModule(competencyService *competencyservice.Service, recommendService *recommendservice.Service, authService *authservice.AuthService) *dashboardmodule.Module {
	return dashboardmodule.New(competencyService, recommendService, authService)
}
