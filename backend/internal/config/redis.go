package config

// RedisConfig 定义 Redis 连接配置。
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}
