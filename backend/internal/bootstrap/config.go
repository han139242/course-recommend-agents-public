package bootstrap

import "github.com/han139242/course-recommend-agents/backend/internal/config"

// LoadConfig 加载应用配置。
func LoadConfig() (config.Config, error) {
	return config.Load()
}
