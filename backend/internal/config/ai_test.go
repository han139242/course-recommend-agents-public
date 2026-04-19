package config

import "testing"

func TestResolveAIProviderOrder(t *testing.T) {
	t.Parallel()

	openAIProvider := AIProviderConfig{
		APIKey:    "openai-key",
		BaseURL:   "https://api.openai.com",
		ChatModel: "gpt-4.1-mini",
	}
	deepseekProvider := AIProviderConfig{
		APIKey:    "deepseek-key",
		BaseURL:   "https://api.deepseek.com",
		ChatModel: "deepseek-chat",
	}

	testCases := []struct {
		name          string
		priority      string
		openAI        AIProviderConfig
		deepseek      AIProviderConfig
		wantPrimary   string
		wantSecondary string
	}{
		{
			name:          "defaults to openai then deepseek",
			priority:      "",
			openAI:        openAIProvider,
			deepseek:      deepseekProvider,
			wantPrimary:   "https://api.openai.com",
			wantSecondary: "https://api.deepseek.com",
		},
		{
			name:          "respects explicit deepseek priority",
			priority:      "deepseek,openai",
			openAI:        openAIProvider,
			deepseek:      deepseekProvider,
			wantPrimary:   "https://api.deepseek.com",
			wantSecondary: "https://api.openai.com",
		},
		{
			name:          "skips disabled provider but keeps remaining",
			priority:      "deepseek,openai",
			openAI:        openAIProvider,
			deepseek:      AIProviderConfig{},
			wantPrimary:   "https://api.openai.com",
			wantSecondary: "",
		},
		{
			name:          "ignores unknown aliases",
			priority:      "unknown,open_ai",
			openAI:        openAIProvider,
			deepseek:      deepseekProvider,
			wantPrimary:   "https://api.openai.com",
			wantSecondary: "",
		},
		{
			name:          "single explicit provider disables fallback",
			priority:      "openai",
			openAI:        openAIProvider,
			deepseek:      deepseekProvider,
			wantPrimary:   "https://api.openai.com",
			wantSecondary: "",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			primary, secondary := resolveAIProviderOrder(testCase.priority, testCase.openAI, testCase.deepseek)

			if primary.BaseURL != testCase.wantPrimary {
				t.Fatalf("expected primary provider %q, got %q", testCase.wantPrimary, primary.BaseURL)
			}
			if secondary.BaseURL != testCase.wantSecondary {
				t.Fatalf("expected secondary provider %q, got %q", testCase.wantSecondary, secondary.BaseURL)
			}
		})
	}
}
