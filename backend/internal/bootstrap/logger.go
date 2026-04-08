package bootstrap

import (
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
)

// NewLogger 初始化日志组件。
// 日志属于最早可用的基础设施，便于后续装配阶段输出启动诊断信息。
func NewLogger(cfg config.Config) *applogger.Logger {
	return applogger.New(cfg.Log.Level)
}
