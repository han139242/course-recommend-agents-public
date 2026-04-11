package config

// FrontendConfig 定义前端静态资源托管配置。
// DistDir 指向构建后的前端目录，存在时由 Go 服务直接承接页面访问。
type FrontendConfig struct {
	DistDir string
}
