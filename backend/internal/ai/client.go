// Package ai provides a small, reusable OpenAI-compatible chat-completions
// client used by content-generation features (resource generation, learner
// profiling, safety review) that need real large-model output. It mirrors the
// proven request/retry/fallback behaviour of the assistant module but exposes a
// minimal, dependency-free surface so multiple modules can share one client.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
)

const defaultTimeout = 45 * time.Second

// Client talks to one or more OpenAI-compatible upstreams (e.g. DeepSeek).
type Client struct {
	cfg        config.AIConfig
	httpClient *http.Client
}

// New builds an AI client from the resolved AI configuration. The returned
// client is safe for concurrent use and can be shared across modules.
func New(cfg config.AIConfig) *Client {
	return &Client{cfg: cfg, httpClient: &http.Client{}}
}

// Enabled reports whether at least one upstream provider is configured. Callers
// use this to decide between a real model call and a local template fallback.
func (c *Client) Enabled() bool {
	return c != nil && c.cfg.Enabled()
}

// VisionEnabled reports whether at least one configured provider exposes a
// vision-capable model, so callers can decide whether to send image inputs.
func (c *Client) VisionEnabled() bool {
	if !c.Enabled() {
		return false
	}
	for _, provider := range c.cfg.Providers() {
		if strings.TrimSpace(provider.VisionModel) != "" {
			return true
		}
	}
	return false
}

// Message is one chat message. Images carries optional vision inputs (data URLs
// or remote URLs) that are appended to the textual content.
type Message struct {
	Role    string
	Content string
	Images  []string
}

// Request configures a single completion call. When Messages is empty the
// System/User fields are used to build a two-message conversation.
type Request struct {
	System      string
	User        string
	Messages    []Message
	MaxTokens   int
	Temperature float64
	JSON        bool
	Timeout     time.Duration
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content any `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete runs a chat completion against the configured providers in order and
// returns the assistant text. It returns the first non-empty success; if every
// provider fails it returns the last error so callers can fall back locally.
func (c *Client) Complete(ctx context.Context, req Request) (string, error) {
	if !c.Enabled() {
		return "", errors.New("ai client is not configured")
	}

	messages, hasImage := buildMessages(req)
	var lastErr error
	for _, provider := range c.cfg.Providers() {
		content, err := c.completeWithProvider(ctx, provider, messages, hasImage, req)
		if err == nil && strings.TrimSpace(content) != "" {
			return content, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("ai providers returned empty content")
	}
	return "", lastErr
}

// completeWithProvider issues one request to a single upstream provider.
func (c *Client) completeWithProvider(ctx context.Context, provider config.AIProviderConfig, messages []map[string]any, hasImage bool, req Request) (string, error) {
	payload := map[string]any{
		"model":    selectModel(provider, hasImage),
		"messages": messages,
		"stream":   false,
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	temperature := req.Temperature
	if temperature <= 0 {
		temperature = 0.6
	}
	payload["temperature"] = temperature
	if req.JSON {
		payload["response_format"] = map[string]any{"type": "json_object"}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	baseURL := strings.TrimRight(provider.BaseURL, "/")
	httpReq, err := http.NewRequestWithContext(requestCtx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)

	response, err := c.doWithRetry(httpReq, 1)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		raw, _ := io.ReadAll(response.Body)
		message := strings.TrimSpace(string(raw))
		if message == "" {
			message = fmt.Sprintf("upstream model returned HTTP %d", response.StatusCode)
		}
		return "", errors.New(message)
	}

	var parsed chatResponse
	if err := json.NewDecoder(response.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("model returned no choices")
	}
	return extractText(parsed.Choices[0].Message.Content), nil
}

// doWithRetry performs the request with a bounded retry on transient statuses.
func (c *Client) doWithRetry(original *http.Request, retries int) (*http.Response, error) {
	var lastErr error
	backoff := 400 * time.Millisecond

	for attempt := 0; attempt <= retries; attempt++ {
		clone, err := cloneRequestWithBody(original)
		if err != nil {
			return nil, err
		}

		response, err := c.httpClient.Do(clone)
		if err == nil {
			if !shouldRetryStatus(response.StatusCode) || attempt == retries {
				return response, nil
			}
			_ = response.Body.Close()
			lastErr = fmt.Errorf("upstream returned HTTP %d", response.StatusCode)
		} else {
			lastErr = err
		}

		if attempt == retries {
			break
		}
		time.Sleep(backoff)
		backoff *= 2
	}

	return nil, lastErr
}

// buildMessages converts the request into the OpenAI message array and reports
// whether any vision (image) content is present.
func buildMessages(req Request) ([]map[string]any, bool) {
	source := req.Messages
	if len(source) == 0 {
		if strings.TrimSpace(req.System) != "" {
			source = append(source, Message{Role: "system", Content: req.System})
		}
		if strings.TrimSpace(req.User) != "" {
			source = append(source, Message{Role: "user", Content: req.User})
		}
	} else if strings.TrimSpace(req.System) != "" {
		source = append([]Message{{Role: "system", Content: req.System}}, source...)
	}

	hasImage := false
	messages := make([]map[string]any, 0, len(source))
	for _, message := range source {
		role := strings.TrimSpace(message.Role)
		if role == "" {
			role = "user"
		}
		if len(message.Images) == 0 {
			messages = append(messages, map[string]any{"role": role, "content": message.Content})
			continue
		}

		parts := make([]map[string]any, 0, len(message.Images)+1)
		if strings.TrimSpace(message.Content) != "" {
			parts = append(parts, map[string]any{"type": "text", "text": message.Content})
		}
		for _, image := range message.Images {
			if strings.TrimSpace(image) == "" {
				continue
			}
			hasImage = true
			parts = append(parts, map[string]any{
				"type":      "image_url",
				"image_url": map[string]any{"url": image},
			})
		}
		messages = append(messages, map[string]any{"role": role, "content": parts})
	}
	return messages, hasImage
}

// selectModel prefers the vision model when image input is present.
func selectModel(provider config.AIProviderConfig, hasImage bool) string {
	if hasImage && strings.TrimSpace(provider.VisionModel) != "" {
		return provider.VisionModel
	}
	return provider.ChatModel
}

// extractText flattens a model message content (string or content-part array)
// into plain text.
func extractText(content any) string {
	switch typed := content.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		var builder strings.Builder
		for _, item := range typed {
			mapped, ok := item.(map[string]any)
			if !ok {
				continue
			}
			text, _ := mapped["text"].(string)
			builder.WriteString(text)
		}
		return strings.TrimSpace(builder.String())
	default:
		return ""
	}
}

// shouldRetryStatus reports whether an HTTP status warrants a retry.
func shouldRetryStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout ||
		statusCode >= http.StatusInternalServerError
}

// cloneRequestWithBody clones a request including a fresh body reader so it can
// be retried safely.
func cloneRequestWithBody(original *http.Request) (*http.Request, error) {
	var body io.ReadCloser
	var err error
	if original.GetBody != nil {
		body, err = original.GetBody()
		if err != nil {
			return nil, err
		}
	}

	cloned := original.Clone(original.Context())
	cloned.Body = body
	return cloned, nil
}
