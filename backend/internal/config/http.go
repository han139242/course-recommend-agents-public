package config

import "time"

// HTTPConfig 定义 HTTP 服务配置。
type HTTPConfig struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}
