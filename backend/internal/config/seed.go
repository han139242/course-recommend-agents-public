package config

import "fmt"

// SeedConfig 定义 seed 命令所需的最小配置。
type SeedConfig struct {
	MySQL MySQLConfig
}

// LoadSeedConfig 加载 seed 命令所需的最小配置。
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
