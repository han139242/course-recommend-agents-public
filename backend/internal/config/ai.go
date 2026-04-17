package config

// AIProviderConfig defines a single OpenAI-compatible upstream provider.
type AIProviderConfig struct {
	APIKey      string
	BaseURL     string
	ChatModel   string
	VisionModel string
}

// Enabled reports whether the provider has enough configuration to serve requests.
func (c AIProviderConfig) Enabled() bool {
	return c.APIKey != "" && c.BaseURL != "" && c.ChatModel != ""
}

// AIConfig holds the preferred upstream plus an optional fallback provider.
type AIConfig struct {
	APIKey              string
	BaseURL             string
	ChatModel           string
	VisionModel         string
	FallbackAPIKey      string
	FallbackBaseURL     string
	FallbackChatModel   string
	FallbackVisionModel string
}

// PrimaryProvider returns the configured preferred upstream.
func (c AIConfig) PrimaryProvider() AIProviderConfig {
	return AIProviderConfig{
		APIKey:      c.APIKey,
		BaseURL:     c.BaseURL,
		ChatModel:   c.ChatModel,
		VisionModel: c.VisionModel,
	}
}

// FallbackProvider returns the optional secondary upstream.
func (c AIConfig) FallbackProvider() AIProviderConfig {
	return AIProviderConfig{
		APIKey:      c.FallbackAPIKey,
		BaseURL:     c.FallbackBaseURL,
		ChatModel:   c.FallbackChatModel,
		VisionModel: c.FallbackVisionModel,
	}
}

// Providers returns the ordered list of configured upstreams.
func (c AIConfig) Providers() []AIProviderConfig {
	providers := make([]AIProviderConfig, 0, 2)
	if primary := c.PrimaryProvider(); primary.Enabled() {
		providers = append(providers, primary)
	}
	if fallback := c.FallbackProvider(); fallback.Enabled() {
		providers = append(providers, fallback)
	}
	return providers
}

// Enabled reports whether at least one upstream is configured.
func (c AIConfig) Enabled() bool {
	return len(c.Providers()) > 0
}
