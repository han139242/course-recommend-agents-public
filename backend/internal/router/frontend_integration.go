package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

type aiProxyRequest struct {
	Messages  []map[string]any `json:"messages"`
	Stream    bool             `json:"stream"`
	MaxTokens int              `json:"maxTokens"`
}

type aiContentResponse struct {
	Choices []struct {
		Message struct {
			Content any `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func registerFrontendIntegrationRoutes(engine *gin.Engine, cfg config.Config) {
	engine.POST("/api/chat", func(ctx *gin.Context) {
		proxyAIStream(ctx, cfg)
	})
	engine.POST("/api/generate-content", func(ctx *gin.Context) {
		proxyAIContent(ctx, cfg)
	})

	distDir := resolveFrontendDistDir(cfg.Frontend.DistDir)
	indexPath := filepath.Join(distDir, "index.html")
	if distDir == "" || !fileExists(indexPath) {
		return
	}

	engine.NoRoute(func(ctx *gin.Context) {
		if strings.HasPrefix(ctx.Request.URL.Path, "/api") || strings.HasPrefix(ctx.Request.URL.Path, "/debug/pprof") || ctx.Request.URL.Path == "/healthz" || ctx.Request.URL.Path == "/readyz" {
			result.Error(ctx, http.StatusNotFound, http.StatusNotFound, "not found")
			return
		}
		if ctx.Request.Method != http.MethodGet && ctx.Request.Method != http.MethodHead {
			result.Error(ctx, http.StatusNotFound, http.StatusNotFound, "not found")
			return
		}

		requestPath := strings.TrimPrefix(filepath.Clean(ctx.Request.URL.Path), string(filepath.Separator))
		if requestPath != "." && requestPath != "" {
			targetPath := filepath.Join(distDir, requestPath)
			if isPathWithinBase(distDir, targetPath) && fileExists(targetPath) {
				ctx.File(targetPath)
				return
			}
		}

		ctx.File(indexPath)
	})
}

func proxyAIStream(ctx *gin.Context, cfg config.Config) {
	requestBody, ok := bindAIProxyRequest(ctx, cfg)
	if !ok {
		return
	}

	upstream, err := newAIProxyHTTPRequest(ctx.Request.Context(), cfg.AI, requestBody, true)
	if err != nil {
		ctx.String(http.StatusBadGateway, err.Error())
		return
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	response, err := doAIProxyRequest(client, upstream, 2)
	if err != nil {
		ctx.String(http.StatusBadGateway, err.Error())
		return
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		ctx.Data(response.StatusCode, "text/plain; charset=utf-8", body)
		return
	}

	ctx.Status(response.StatusCode)
	ctx.Header("Content-Type", "text/event-stream; charset=utf-8")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	flusher, _ := ctx.Writer.(http.Flusher)
	buffer := make([]byte, 32*1024)
	for {
		n, readErr := response.Body.Read(buffer)
		if n > 0 {
			_, _ = ctx.Writer.Write(buffer[:n])
			if flusher != nil {
				flusher.Flush()
			}
		}
		if errors.Is(readErr, io.EOF) {
			return
		}
		if readErr != nil {
			return
		}
	}
}

func proxyAIContent(ctx *gin.Context, cfg config.Config) {
	requestBody, ok := bindAIProxyRequest(ctx, cfg)
	if !ok {
		return
	}

	upstream, err := newAIProxyHTTPRequest(ctx.Request.Context(), cfg.AI, requestBody, false)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	client := &http.Client{Timeout: 2 * time.Minute}
	response, err := doAIProxyRequest(client, upstream, 2)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		ctx.Data(response.StatusCode, "application/json; charset=utf-8", body)
		return
	}

	var payload aiContentResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"content": extractAIContent(payload),
	})
}

func bindAIProxyRequest(ctx *gin.Context, cfg config.Config) (aiProxyRequest, bool) {
	if !cfg.AI.Enabled() {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "AI proxy is not configured. Please set OPENAI_API_KEY or DEEPSEEK_API_KEY and related env vars.",
		})
		return aiProxyRequest{}, false
	}

	var requestBody aiProxyRequest
	if err := ctx.ShouldBindJSON(&requestBody); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid AI request body"})
		return aiProxyRequest{}, false
	}
	if len(requestBody.Messages) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "messages are required"})
		return aiProxyRequest{}, false
	}

	return requestBody, true
}

func newAIProxyHTTPRequest(ctx context.Context, aiCfg config.AIConfig, requestBody aiProxyRequest, stream bool) (*http.Request, error) {
	payload := map[string]any{
		"model":    selectAIModel(aiCfg, requestBody.Messages),
		"messages": requestBody.Messages,
		"stream":   stream,
	}
	if requestBody.MaxTokens > 0 {
		payload["max_tokens"] = requestBody.MaxTokens
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(aiCfg.BaseURL, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+aiCfg.APIKey)
	return request, nil
}

func doAIProxyRequest(client *http.Client, original *http.Request, retries int) (*http.Response, error) {
	var lastErr error
	backoff := 400 * time.Millisecond

	for attempt := 0; attempt <= retries; attempt++ {
		request, err := cloneRequestWithBody(original)
		if err != nil {
			return nil, err
		}

		response, err := client.Do(request)
		if err == nil {
			if !shouldRetryAIStatus(response.StatusCode) || attempt == retries {
				return response, nil
			}
		}

		if err != nil {
			lastErr = err
		} else {
			body, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			lastErr = fmt.Errorf("%s", strings.TrimSpace(string(body)))
			if lastErr.Error() == "" {
				lastErr = fmt.Errorf("upstream AI service returned HTTP %d", response.StatusCode)
			}
		}

		if attempt == retries {
			return nil, lastErr
		}

		time.Sleep(backoff)
		backoff *= 2
	}

	return nil, lastErr
}

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

func shouldRetryAIStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout ||
		statusCode >= http.StatusInternalServerError
}

func selectAIModel(aiCfg config.AIConfig, messages []map[string]any) string {
	if messagesIncludeImage(messages) && aiCfg.VisionModel != "" {
		return aiCfg.VisionModel
	}
	return aiCfg.ChatModel
}

func messagesIncludeImage(messages []map[string]any) bool {
	for _, message := range messages {
		parts, ok := message["content"].([]any)
		if !ok {
			continue
		}
		for _, part := range parts {
			mapped, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if mapped["type"] != "image_url" {
				continue
			}
			imageValue, ok := mapped["image_url"].(map[string]any)
			if !ok {
				continue
			}
			if strings.TrimSpace(stringValue(imageValue["url"])) != "" {
				return true
			}
		}
	}
	return false
}

func extractAIContent(payload aiContentResponse) string {
	if len(payload.Choices) == 0 {
		return ""
	}

	content := payload.Choices[0].Message.Content
	switch typed := content.(type) {
	case string:
		return typed
	case []any:
		var builder strings.Builder
		for _, item := range typed {
			mapped, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if mapped["type"] != "text" {
				continue
			}
			text := stringValue(mapped["text"])
			if text == "" {
				continue
			}
			if builder.Len() > 0 {
				builder.WriteByte('\n')
			}
			builder.WriteString(text)
		}
		return builder.String()
	default:
		return ""
	}
}

func stringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func resolveFrontendDistDir(distDir string) string {
	cleaned := strings.TrimSpace(distDir)
	if cleaned == "" {
		return ""
	}

	if filepath.IsAbs(cleaned) {
		if dirExists(cleaned) {
			return cleaned
		}
		return ""
	}

	for _, candidate := range frontendDistCandidates(cleaned) {
		if dirExists(candidate) {
			return candidate
		}
	}

	return ""
}

func frontendDistCandidates(distDir string) []string {
	candidates := make([]string, 0, 6)
	seen := make(map[string]struct{})

	addCandidate := func(base string, suffix string) {
		if strings.TrimSpace(base) == "" {
			return
		}

		candidate := filepath.Clean(filepath.Join(base, suffix))
		if _, exists := seen[candidate]; exists {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}

	if workingDir, err := os.Getwd(); err == nil {
		addCandidate(workingDir, distDir)
		addCandidate(workingDir, filepath.Join("backend", distDir))
		addCandidate(filepath.Dir(workingDir), distDir)
	}

	if executablePath, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executablePath)
		addCandidate(executableDir, distDir)
		addCandidate(executableDir, filepath.Join("backend", distDir))
		addCandidate(filepath.Dir(executableDir), distDir)
	}

	return candidates
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isPathWithinBase(base string, target string) bool {
	relative, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return relative == "." || (!strings.HasPrefix(relative, "..") && relative != "")
}
