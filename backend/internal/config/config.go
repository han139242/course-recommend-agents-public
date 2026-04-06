package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 聚合应用运行所需的全部配置。
type Config struct {
	App   AppConfig
	HTTP  HTTPConfig
	MySQL MySQLConfig
	Redis RedisConfig
	Neo4j Neo4jConfig
	Auth  AuthConfig
	Log   LogConfig
}

// Load 从环境变量加载配置，并补齐开发环境默认值。
func Load() (Config, error) {
	cfg := Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "course-recommend-agents-backend"),
			Env:  getEnv("APP_ENV", "development"),
		},
		HTTP: HTTPConfig{
			Host:            getEnv("HTTP_HOST", "0.0.0.0"),
			Port:            getEnv("HTTP_PORT", "8080"),
			ReadTimeout:     getDurationEnv("HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout:    getDurationEnv("HTTP_WRITE_TIMEOUT", 10*time.Second),
			ShutdownTimeout: getDurationEnv("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		MySQL: MySQLConfig{
			DSN: os.Getenv("MYSQL_DSN"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		Neo4j: Neo4jConfig{
			URI:      getEnv("NEO4J_URI", "neo4j://127.0.0.1:7687"),
			Username: getEnv("NEO4J_USERNAME", "neo4j"),
			Password: os.Getenv("NEO4J_PASSWORD"),
		},
		Auth: AuthConfig{
			JWTSecret: os.Getenv("JWT_SECRET"),
			TokenTTL:  getDurationEnv("JWT_TOKEN_TTL", 24*time.Hour),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate 校验关键配置是否完整。
func (c Config) Validate() error {
	if c.HTTP.Port == "" {
		return fmt.Errorf("HTTP_PORT 不能为空")
	}

	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET 不能为空")
	}

	return nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
