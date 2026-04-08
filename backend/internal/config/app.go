package config

// AppConfig 定义应用基础配置。
// 该配置描述应用身份与运行环境，用于日志、监控和环境分流判断。
type AppConfig struct {
	Name string
	Env  string
}
