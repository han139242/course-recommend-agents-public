package config

// RedisConfig 定义 Redis 连接配置。
// Redis 用于缓存与加速场景，配置缺失时可由上层决定是否降级。
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}
