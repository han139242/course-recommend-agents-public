package service

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
	assistantdto "github.com/han139242/course-recommend-agents/backend/internal/module/assistant/dto"
	authdto "github.com/han139242/course-recommend-agents/backend/internal/module/auth/dto"
	authentity "github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	competencyentity "github.com/han139242/course-recommend-agents/backend/internal/module/competency/entity"
	competencyservice "github.com/han139242/course-recommend-agents/backend/internal/module/competency/service"
	dashboardservice "github.com/han139242/course-recommend-agents/backend/internal/module/dashboard/service"
	frontierservice "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/service"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	learningservice "github.com/han139242/course-recommend-agents/backend/internal/module/learning/service"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
	"gorm.io/gorm"
)

const maxToolRounds = 6

// Service implements an authenticated project-specific assistant with
// controlled tool calling over the existing learning data stack.
type Service struct {
	aiCfg             config.AIConfig
	httpClient        *http.Client
	authService       *authservice.AuthService
	learningService   *learningservice.Service
	competencyService *competencyservice.Service
	recommendService  *recommendservice.Service
	dashboardService  *dashboardservice.Service
	graphService      *graphservice.Service
	frontierService   *frontierservice.Service
	db                *gorm.DB
}

type aiChatResponse struct {
	Choices []struct {
		Message aiAssistantMessage `json:"message"`
	} `json:"choices"`
}

type aiAssistantMessage struct {
	Role      string       `json:"role"`
	Content   any          `json:"content"`
	ToolCalls []aiToolCall `json:"tool_calls"`
}

type aiToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function aiToolFunction `json:"function"`
}

type aiToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// New creates an assistant service.
func New(aiCfg config.AIConfig, authService *authservice.AuthService, learningService *learningservice.Service, competencyService *competencyservice.Service, recommendService *recommendservice.Service, dashboardService *dashboardservice.Service, graphService *graphservice.Service, frontierService *frontierservice.Service, db *gorm.DB) *Service {
	return &Service{
		aiCfg:             aiCfg,
		httpClient:        &http.Client{Timeout: 2 * time.Minute},
		authService:       authService,
		learningService:   learningService,
		competencyService: competencyService,
		recommendService:  recommendService,
		dashboardService:  dashboardService,
		graphService:      graphService,
		frontierService:   frontierService,
		db:                db,
	}
}

// Chat runs the tool-calling agent loop and returns the final answer content.
func (s *Service) Chat(ctx context.Context, request assistantdto.ChatRequest) (string, error) {
	if !s.aiCfg.Enabled() {
		return "", apperr.ErrDependencyUnavailable
	}
	if s.authService == nil {
		return "", apperr.ErrDependencyUnavailable
	}
	if len(request.Messages) == 0 {
		return "", apperr.ErrBadRequest
	}

	userID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		return "", apperr.ErrUnauthorized
	}

	currentUser, err := s.authService.GetCurrentUser(ctx, userID)
	if err != nil {
		return "", err
	}

	messages := s.buildConversation(currentUser, request.Messages)
	for round := 0; round < maxToolRounds; round++ {
		reply, err := s.requestModel(ctx, messages, request.MaxTokens)
		if err != nil {
			return "", err
		}

		if len(reply.ToolCalls) == 0 {
			content := strings.TrimSpace(extractAssistantContent(reply.Content))
			if content == "" {
				return "", apperr.ErrDependencyUnavailable.WithCause(errors.New("assistant returned empty content"))
			}
			return content, nil
		}

		messages = append(messages, map[string]any{
			"role":       "assistant",
			"content":    extractAssistantContent(reply.Content),
			"tool_calls": reply.ToolCalls,
		})

		for _, toolCall := range reply.ToolCalls {
			toolResult := s.executeTool(ctx, currentUser, toolCall.Function.Name, toolCall.Function.Arguments)
			messages = append(messages, map[string]any{
				"role":         "tool",
				"tool_call_id": toolCall.ID,
				"content":      toolResult,
			})
		}
	}

	return "", apperr.ErrDependencyUnavailable.WithCause(errors.New("assistant exceeded tool call limit"))
}

func (s *Service) buildConversation(currentUser *authdto.AuthUserView, original []map[string]any) []map[string]any {
	messages := make([]map[string]any, 0, len(original)+1)
	messages = append(messages, map[string]any{
		"role":    "system",
		"content": s.buildSystemPrompt(currentUser),
	})
	messages = append(messages, normalizeUserMessages(original)...)
	return messages
}

func (s *Service) buildSystemPrompt(currentUser *authdto.AuthUserView) string {
	lines := []string{
		"你是 course-recommend-agents 项目内的学习强 Agent，服务对象是当前已登录用户。",
		"你不是泛泛聊天助手，你必须优先结合系统内实时数据、数据库查询结果、知识图谱和推荐结果回答问题。",
		fmt.Sprintf("当前时间：%s。", time.Now().Format(time.RFC3339)),
	}

	if currentUser != nil {
		lines = append(lines,
			fmt.Sprintf("当前用户：%s。", strings.TrimSpace(currentUser.Name)),
			fmt.Sprintf("当前角色：%s。", strings.TrimSpace(currentUser.Role)),
		)
		if strings.TrimSpace(currentUser.Target) != "" {
			lines = append(lines, fmt.Sprintf("用户目标方向：%s。", strings.TrimSpace(currentUser.Target)))
		}
	}

	lines = append(lines,
		"当用户询问“我现在情况如何、最近表现、薄弱点、推荐原因、下一步学什么、某知识点和我的关系、最近错题、数据库里有什么记录”等问题时，必须先调用相关工具，不能凭空猜测。",
		"如果用户问数据来源，要明确写出来自 MySQL 学习记录、Redis 实时状态、Neo4j 知识图谱，还是推荐/画像服务。",
		"如果工具没有查到结果，要如实说明当前系统没有查到对应数据。",
		"回答使用中文，优先输出：结论、依据、下一步建议。",
		"不要编造学生状态，不要伪造知识点和掌握度。",
	)

	return strings.Join(lines, "\n")
}

func normalizeUserMessages(original []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(original))
	for _, message := range original {
		role, _ := message["role"].(string)
		role = strings.TrimSpace(role)
		switch role {
		case "system", "user", "assistant":
		default:
			continue
		}

		content, exists := message["content"]
		if !exists {
			continue
		}
		result = append(result, map[string]any{
			"role":    role,
			"content": content,
		})
	}
	return result
}

func (s *Service) requestModel(ctx context.Context, messages []map[string]any, maxTokens int) (*aiAssistantMessage, error) {
	payload := map[string]any{
		"model":       s.selectModel(messages),
		"messages":    messages,
		"tools":       buildAgentTools(),
		"tool_choice": "auto",
		"stream":      false,
	}
	if maxTokens > 0 {
		payload["max_tokens"] = maxTokens
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, apperr.ErrInternal.WithCause(err)
	}

	baseURL := strings.TrimRight(s.aiCfg.BaseURL, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+s.aiCfg.APIKey)

	response, err := s.doModelRequest(request, 2)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		raw, _ := io.ReadAll(response.Body)
		message := strings.TrimSpace(string(raw))
		if message == "" {
			message = fmt.Sprintf("upstream assistant model returned HTTP %d", response.StatusCode)
		}
		return nil, apperr.ErrDependencyUnavailable.WithCause(errors.New(message))
	}

	var parsed aiChatResponse
	if err := json.NewDecoder(response.Body).Decode(&parsed); err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	if len(parsed.Choices) == 0 {
		return nil, apperr.ErrDependencyUnavailable.WithCause(errors.New("assistant model returned no choices"))
	}
	return &parsed.Choices[0].Message, nil
}

func (s *Service) doModelRequest(original *http.Request, retries int) (*http.Response, error) {
	var lastErr error
	backoff := 400 * time.Millisecond

	for attempt := 0; attempt <= retries; attempt++ {
		request, err := cloneRequestWithBody(original)
		if err != nil {
			return nil, err
		}

		response, err := s.httpClient.Do(request)
		if err == nil {
			if !shouldRetryStatus(response.StatusCode) || attempt == retries {
				return response, nil
			}
			body, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			lastErr = errors.New(strings.TrimSpace(string(body)))
			if lastErr.Error() == "" {
				lastErr = fmt.Errorf("upstream returned HTTP %d", response.StatusCode)
			}
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

func (s *Service) selectModel(messages []map[string]any) string {
	if messagesIncludeImage(messages) && strings.TrimSpace(s.aiCfg.VisionModel) != "" {
		return s.aiCfg.VisionModel
	}
	return s.aiCfg.ChatModel
}

func extractAssistantContent(content any) string {
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
			if strings.TrimSpace(stringValue(mapped["type"])) != "text" {
				continue
			}
			text := stringValue(mapped["text"])
			if strings.TrimSpace(text) == "" {
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

func shouldRetryStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout ||
		statusCode >= http.StatusInternalServerError
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

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func currentUserEntity(user *authdto.AuthUserView) *authentity.User {
	if user == nil {
		return nil
	}
	return &authentity.User{
		ID:        user.ID,
		Name:      user.Name,
		Role:      user.Role,
		ClassID:   user.ClassID,
		ClassName: user.ClassName,
		Target:    user.Target,
	}
}

func currentUserProfile(user *authdto.AuthUserView) *competencyentity.UserProfile {
	if user == nil {
		return nil
	}
	return &competencyentity.UserProfile{
		ID:        user.ID,
		Name:      user.Name,
		Role:      user.Role,
		ClassID:   user.ClassID,
		ClassName: user.ClassName,
		Target:    user.Target,
	}
}
