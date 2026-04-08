package config

import "time"

// AuthConfig 定义认证配置。
// 认证配置属于安全边界的一部分，缺失关键字段时应阻止服务启动。
type AuthConfig struct {
	JWTSecret string
	TokenTTL  time.Duration
}
