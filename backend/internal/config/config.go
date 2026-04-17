package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	AI            AIConfig
	Frontend      FrontendConfig
	Log           LogConfig
	GraphWriter   GraphWriterConfig
	Observability ObservabilityConfig
}

// Load 从环境变量加载配置，并补齐开发环境默认值。
//
// 对可选基础设施使用偏宽松的默认策略，对核心安全配置保持强校验，
// 以便本地开发可快速启动，同时避免生产环境在关键项缺失时静默运行。
func Load() (Config, error) {
	loadDotEnv()

	appEnv := getEnv("APP_ENV", "development")

	cfg := Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "course-recommend-agents-backend"),
			Env:  appEnv,
		},
		HTTP: HTTPConfig{
			Host:        getEnv("HTTP_HOST", "0.0.0.0"),
			Port:        getEnv("HTTP_PORT", "8080"),
			ReadTimeout: getDurationEnv("HTTP_READ_TIMEOUT", 5*time.Second),
			// AI 对话与学习内容生成会走流式输出，默认 10s 会导致响应被服务端提前截断。
			WriteTimeout:    getDurationEnv("HTTP_WRITE_TIMEOUT", 2*time.Minute),
			ShutdownTimeout: getDurationEnv("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		MySQL: MySQLConfig{
			DSN: getMySQLDSN(),
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
			JWTSecret: getEnv("JWT_SECRET", defaultJWTSecret(appEnv)),
			TokenTTL:  getDurationEnv("JWT_TOKEN_TTL", 24*time.Hour),
		},
		AI: buildAIConfig(),
		Frontend: FrontendConfig{
			DistDir: getEnv("FRONTEND_DIST_DIR", "web/dist"),
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
			BuildVersion:       getEnv("BUILD_VERSION", "dev"),
			BuildCommit:        getEnv("BUILD_COMMIT", "local"),
			BuildTime:          getEnv("BUILD_TIME", ""),
			EnablePprof:        getBoolEnv("ENABLE_PPROF", false),
			EnableMetrics:      getBoolEnv("ENABLE_METRICS", false),
			EnableTracing:      getBoolEnv("ENABLE_TRACING", false),
			SlowQueryThreshold: getEnv("SLOW_QUERY_THRESHOLD", "500ms"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	if cfg.AI.VisionModel == "" {
		cfg.AI.VisionModel = cfg.AI.ChatModel
	}

	return cfg, nil
}

func loadDotEnv() {
	candidates := []string{
		".env",
		filepath.Join("..", ".env"),
		filepath.Join("..", "..", ".env"),
	}

	for _, candidate := range candidates {
		raw, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}

		lines := strings.Split(string(raw), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "export ") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
			}

			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}

			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if key == "" || os.Getenv(key) != "" {
				continue
			}
			if len(value) >= 2 {
				if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
					value = value[1 : len(value)-1]
				}
			}

			_ = os.Setenv(key, value)
		}
		return
	}
}

func getMySQLDSN() string {
	if dsn := strings.TrimSpace(os.Getenv("MYSQL_DSN")); dsn != "" {
		return dsn
	}

	host := strings.TrimSpace(os.Getenv("MYSQL_HOST"))
	if host == "" {
		return ""
	}

	port := getEnv("MYSQL_PORT", "3306")
	user := getEnv("MYSQL_USER", "root")
	password := os.Getenv("MYSQL_PASSWORD")
	database := getEnv("MYSQL_DATABASE", "course_recommend")
	params := getEnv("MYSQL_PARAMS", "parseTime=true&charset=utf8mb4&loc=Local")

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", user, password, host, port, database, params)
}

func defaultJWTSecret(appEnv string) string {
	if strings.EqualFold(appEnv, "production") {
		return ""
	}
	return "course-recommend-dev-secret"
}

func buildAIConfig() AIConfig {
	openAIKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	deepseekKey := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))

	openAIProvider := AIProviderConfig{
		APIKey:      openAIKey,
		BaseURL:     strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
		ChatModel:   strings.TrimSpace(os.Getenv("OPENAI_MODEL")),
		VisionModel: strings.TrimSpace(os.Getenv("OPENAI_VISION_MODEL")),
	}
	if openAIProvider.BaseURL == "" && openAIProvider.APIKey != "" {
		openAIProvider.BaseURL = "https://api.openai.com"
	}
	if openAIProvider.ChatModel == "" && openAIProvider.APIKey != "" {
		openAIProvider.ChatModel = "gpt-4.1-mini"
	}

	deepseekProvider := AIProviderConfig{
		APIKey:      deepseekKey,
		BaseURL:     strings.TrimSpace(os.Getenv("DEEPSEEK_BASE_URL")),
		ChatModel:   strings.TrimSpace(os.Getenv("DEEPSEEK_MODEL")),
		VisionModel: strings.TrimSpace(os.Getenv("DEEPSEEK_VISION_MODEL")),
	}
	if deepseekProvider.BaseURL == "" && deepseekProvider.APIKey != "" {
		deepseekProvider.BaseURL = "https://api.deepseek.com"
	}
	if deepseekProvider.ChatModel == "" && deepseekProvider.APIKey != "" {
		deepseekProvider.ChatModel = "deepseek-chat"
	}

	primary := AIProviderConfig{}
	fallback := AIProviderConfig{}
	switch {
	case openAIProvider.Enabled():
		primary = openAIProvider
		if deepseekProvider.Enabled() {
			fallback = deepseekProvider
		}
	case deepseekProvider.Enabled():
		primary = deepseekProvider
	}

	return AIConfig{
		APIKey:              primary.APIKey,
		BaseURL:             primary.BaseURL,
		ChatModel:           primary.ChatModel,
		VisionModel:         primary.VisionModel,
		FallbackAPIKey:      fallback.APIKey,
		FallbackBaseURL:     fallback.BaseURL,
		FallbackChatModel:   fallback.ChatModel,
		FallbackVisionModel: fallback.VisionModel,
	}
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
