package config

import "time"

// HTTPConfig 定义 HTTP 服务配置。
// 这些参数决定监听地址、请求超时与优雅停机窗口。
type HTTPConfig struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}
