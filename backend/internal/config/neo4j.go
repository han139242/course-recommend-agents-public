package config

// Neo4jConfig 定义 Neo4j 连接配置。
// Neo4j 当前为可选依赖，因此配置允许存在默认值与部分缺省场景。
type Neo4jConfig struct {
	URI      string
	Username string
	Password string
}
