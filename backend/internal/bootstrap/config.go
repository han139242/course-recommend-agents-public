package bootstrap

import "github.com/han139242/course-recommend-agents/backend/internal/config"

// LoadConfig 加载应用配置。
// 该函数保留在 bootstrap 层，便于入口统一从装配流程获取配置对象。
func LoadConfig() (config.Config, error) {
	return config.Load()
}
