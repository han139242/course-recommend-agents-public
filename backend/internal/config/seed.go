package config

import "fmt"

// SeedConfig 定义 seed 命令所需的最小配置。
// 该命令只依赖数据库连接，因此保持独立配置，避免复用完整服务启动校验。
type SeedConfig struct {
	MySQL MySQLConfig
}

// LoadSeedConfig 加载 seed 命令所需的最小配置。
// 对 seed 场景不做多余默认填充，缺少数据库连接时直接失败，防止误写入错误环境。
func LoadSeedConfig() (SeedConfig, error) {
	cfg := SeedConfig{
		MySQL: MySQLConfig{
			DSN: getEnv("MYSQL_DSN", ""),
		},
	}

	if cfg.MySQL.DSN == "" {
		return SeedConfig{}, fmt.Errorf("MYSQL_DSN 不能为空")
	}

	return cfg, nil
}
