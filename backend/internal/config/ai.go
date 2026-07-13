package config

// AIProviderConfig defines a single OpenAI-compatible upstream provider.
type AIProviderConfig struct {
	APIKey              string
	BaseURL             string
	ChatModel           string
	VisionModel         string
	ImageModel          string
	ImageEndpoint       string
	ImageStatusEndpoint string
	VideoModel          string
	VideoEndpoint       string
	VideoStatusEndpoint string
}

// Enabled reports whether the provider has enough configuration to serve requests.
func (c AIProviderConfig) Enabled() bool {
	return c.APIKey != "" && c.BaseURL != "" && c.ChatModel != ""
}

// ImageEnabled reports whether the provider can serve text-to-image requests.
func (c AIProviderConfig) ImageEnabled() bool {
	return c.APIKey != "" && c.BaseURL != "" && c.ImageModel != ""
}

// VideoEnabled reports whether the provider can serve text-to-video requests.
func (c AIProviderConfig) VideoEnabled() bool {
	return c.APIKey != "" && c.BaseURL != "" && c.VideoModel != ""
}

// AIConfig holds the preferred upstream plus an optional fallback provider.
type AIConfig struct {
	APIKey                      string
	BaseURL                     string
	ChatModel                   string
	VisionModel                 string
	ImageModel                  string
	ImageEndpoint               string
	ImageStatusEndpoint         string
	VideoModel                  string
	VideoEndpoint               string
	VideoStatusEndpoint         string
	FallbackAPIKey              string
	FallbackBaseURL             string
	FallbackChatModel           string
	FallbackVisionModel         string
	FallbackImageModel          string
	FallbackImageEndpoint       string
	FallbackImageStatusEndpoint string
	FallbackVideoModel          string
	FallbackVideoEndpoint       string
	FallbackVideoStatusEndpoint string
}

// PrimaryProvider returns the configured preferred upstream.
func (c AIConfig) PrimaryProvider() AIProviderConfig {
	return AIProviderConfig{
		APIKey:              c.APIKey,
		BaseURL:             c.BaseURL,
		ChatModel:           c.ChatModel,
		VisionModel:         c.VisionModel,
		ImageModel:          c.ImageModel,
		ImageEndpoint:       c.ImageEndpoint,
		ImageStatusEndpoint: c.ImageStatusEndpoint,
		VideoModel:          c.VideoModel,
		VideoEndpoint:       c.VideoEndpoint,
		VideoStatusEndpoint: c.VideoStatusEndpoint,
	}
}

// FallbackProvider returns the optional secondary upstream.
func (c AIConfig) FallbackProvider() AIProviderConfig {
	return AIProviderConfig{
		APIKey:              c.FallbackAPIKey,
		BaseURL:             c.FallbackBaseURL,
		ChatModel:           c.FallbackChatModel,
		VisionModel:         c.FallbackVisionModel,
		ImageModel:          c.FallbackImageModel,
		ImageEndpoint:       c.FallbackImageEndpoint,
		ImageStatusEndpoint: c.FallbackImageStatusEndpoint,
		VideoModel:          c.FallbackVideoModel,
		VideoEndpoint:       c.FallbackVideoEndpoint,
		VideoStatusEndpoint: c.FallbackVideoStatusEndpoint,
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

// MediaProviders returns providers that can serve image or video generation,
// keeping the same preferred/fallback ordering as chat completions.
func (c AIConfig) MediaProviders() []AIProviderConfig {
	providers := make([]AIProviderConfig, 0, 2)
	if primary := c.PrimaryProvider(); primary.ImageEnabled() || primary.VideoEnabled() {
		providers = append(providers, primary)
	}
	if fallback := c.FallbackProvider(); fallback.ImageEnabled() || fallback.VideoEnabled() {
		providers = append(providers, fallback)
	}
	return providers
}

// Enabled reports whether at least one upstream is configured.
func (c AIConfig) Enabled() bool {
	return len(c.Providers()) > 0
}
