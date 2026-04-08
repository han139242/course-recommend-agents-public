package config

// MySQLConfig 定义 MySQL 连接配置。
// DSN 由部署环境提供，不在代码内拼接以降低配置分散风险。
type MySQLConfig struct {
	DSN string
}
