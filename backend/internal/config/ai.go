package config

// AIConfig 定义兼容 OpenAI Chat Completions 协议的 AI 代理配置。
// 当前用于把前端的对话与内容生成请求统一转发到上游模型服务。
type AIConfig struct {
	APIKey      string
	BaseURL     string
	ChatModel   string
	VisionModel string
}

// Enabled 返回当前是否启用 AI 代理能力。
func (c AIConfig) Enabled() bool {
	return c.APIKey != "" && c.BaseURL != "" && c.ChatModel != ""
}
