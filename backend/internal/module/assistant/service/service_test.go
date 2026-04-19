package service

import (
	"net/http"
	"testing"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
)

func TestNewAssistantServiceUsesContextDrivenHTTPTimeout(t *testing.T) {
	t.Parallel()

	svc := New(config.AIConfig{}, nil, nil, nil, nil, nil, nil, nil, nil)
	if svc.httpClient == nil {
		t.Fatal("expected http client to be initialized")
	}
	if svc.httpClient.Timeout != 0 {
		t.Fatalf("expected zero global timeout, got %s", svc.httpClient.Timeout)
	}
}

func TestProviderRetriesSingleProvider(t *testing.T) {
	t.Parallel()

	svc := &Service{
		aiCfg: config.AIConfig{
			APIKey:    "deepseek-key",
			BaseURL:   "https://api.deepseek.com",
			ChatModel: "deepseek-chat",
		},
		httpClient: &http.Client{},
	}

	if got := svc.providerRetries(); got != 1 {
		t.Fatalf("expected one retry for single provider, got %d", got)
	}
}

func TestChatProviderTimeoutDeepSeekSingleProvider(t *testing.T) {
	t.Parallel()

	svc := &Service{
		aiCfg: config.AIConfig{
			APIKey:    "deepseek-key",
			BaseURL:   "https://api.deepseek.com",
			ChatModel: "deepseek-chat",
		},
		httpClient: &http.Client{},
	}

	timeout := svc.chatProviderTimeout(config.AIProviderConfig{
		APIKey:    "deepseek-key",
		BaseURL:   "https://api.deepseek.com",
		ChatModel: "deepseek-chat",
	})
	if timeout.Seconds() < 20 {
		t.Fatalf("expected relaxed deepseek timeout, got %s", timeout)
	}
}

func TestShouldPreferStructuredAssistantResponse(t *testing.T) {
	t.Parallel()

	if !shouldPreferStructuredAssistantResponse(agentIntentPlanner) {
		t.Fatal("expected planner intent to prefer structured assistant response")
	}
	if !shouldPreferStructuredAssistantResponse(agentIntentAssessor) {
		t.Fatal("expected assessor intent to prefer structured assistant response")
	}
	if shouldPreferStructuredAssistantResponse(agentIntentExplorer) {
		t.Fatal("expected explorer intent to keep regular assistant flow")
	}
}
