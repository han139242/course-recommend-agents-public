package config

// LogConfig 定义日志配置。
// 当前仅暴露日志级别，后续如需扩展输出目标可继续在此聚合。
type LogConfig struct {
	Level string
}
