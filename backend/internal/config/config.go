package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 聚合应用运行所需的全部配置。
// 不同子配置按基础设施和运行阶段拆分，便于启动时集中校验。
type Config struct {
	App           AppConfig
	HTTP          HTTPConfig
	MySQL         MySQLConfig
	Redis         RedisConfig
	Neo4j         Neo4jConfig
	Auth          AuthConfig
	Log           LogConfig
	GraphWriter   GraphWriterConfig
	Observability ObservabilityConfig
}

// Load 从环境变量加载配置，并补齐开发环境默认值。
//
// 对可选基础设施使用偏宽松的默认策略，对核心安全配置保持强校验，
// 以便本地开发可快速启动，同时避免生产环境在关键项缺失时静默运行。
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
		GraphWriter: GraphWriterConfig{
			Enabled:         getBoolEnv("GRAPH_WRITER_ENABLED", true),
			QueueSize:       getIntEnv("GRAPH_WRITER_QUEUE_SIZE", 128),
			BatchSize:       getIntEnv("GRAPH_WRITER_BATCH_SIZE", 16),
			FlushInterval:   getDurationEnv("GRAPH_WRITER_FLUSH_INTERVAL", 2*time.Second),
			RetryTimes:      getIntEnv("GRAPH_WRITER_RETRY_TIMES", 3),
			TopologyChannel: getEnv("GRAPH_WRITER_TOPOLOGY_CHANNEL", "topology_updates"),
		},
		Observability: ObservabilityConfig{
			BuildVersion: getEnv("BUILD_VERSION", "dev"),
			BuildCommit:  getEnv("BUILD_COMMIT", "local"),
			BuildTime:    getEnv("BUILD_TIME", ""),
			EnablePprof:  getBoolEnv("ENABLE_PPROF", false),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate 校验关键配置是否完整。
//
// 这里仅校验会直接影响服务可用性或安全边界的必填项，
// 其余可降级依赖交由各自 bootstrap 阶段按需判断。
func (c Config) Validate() error {
	if c.HTTP.Port == "" {
		return fmt.Errorf("HTTP_PORT 不能为空")
	}

	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET 不能为空")
	}

	return nil
}

// getEnv 读取字符串环境变量，缺失时返回回退值。
func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

// getIntEnv 读取整型环境变量，解析失败时回退到默认值。
func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		fmt.Printf("配置项 %s 不是合法整数，已回退默认值\n", key)
		return fallback
	}

	return parsed
}

// getBoolEnv 读取布尔环境变量，格式非法时回退到默认值。
func getBoolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		fmt.Printf("配置项 %s 不是合法布尔值，已回退默认值\n", key)
		return fallback
	}

	return parsed
}

// getDurationEnv 读取时长环境变量，格式非法时回退到默认值。
func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		fmt.Printf("配置项 %s 不是合法时长，已回退默认值\n", key)
		return fallback
	}

	return parsed
}
