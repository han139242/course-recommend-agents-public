package config

import "time"

// AuthConfig 定义认证配置。
type AuthConfig struct {
	JWTSecret string
	TokenTTL  time.Duration
}
