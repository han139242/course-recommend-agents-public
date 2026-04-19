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
	competencydto "github.com/han139242/course-recommend-agents/backend/internal/module/competency/dto"
	competencyentity "github.com/han139242/course-recommend-agents/backend/internal/module/competency/entity"
	competencyservice "github.com/han139242/course-recommend-agents/backend/internal/module/competency/service"
	dashboardservice "github.com/han139242/course-recommend-agents/backend/internal/module/dashboard/service"
	frontierservice "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/service"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
	learningservice "github.com/han139242/course-recommend-agents/backend/internal/module/learning/service"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
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

type agentIntent string

const (
	agentIntentPlanner  agentIntent = "planner"
	agentIntentExplorer agentIntent = "explorer"
	agentIntentAssessor agentIntent = "assessor"
)

type assistantAgentProfile struct {
	Intent         agentIntent
	Name           string
	Label          string
	LearnerProfile *recommenddto.LearnerProfileView
}

// New creates an assistant service.
func New(aiCfg config.AIConfig, authService *authservice.AuthService, learningService *learningservice.Service, competencyService *competencyservice.Service, recommendService *recommendservice.Service, dashboardService *dashboardservice.Service, graphService *graphservice.Service, frontierService *frontierservice.Service, db *gorm.DB) *Service {
	return &Service{
		aiCfg:             aiCfg,
		// Use per-request contexts to control provider latency. A global client
		// timeout causes DeepSeek planning/assessment calls to fail early and
		// incorrectly fall back to local-only replies.
		httpClient:        &http.Client{},
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

// Chat runs the authenticated tool-calling loop and returns the final answer plus queued UI actions.
func (s *Service) Chat(ctx context.Context, request assistantdto.ChatRequest) (*assistantdto.ChatResponse, error) {
	if s.authService == nil {
		return nil, apperr.ErrDependencyUnavailable
	}
	if len(request.Messages) == 0 {
		return nil, apperr.ErrBadRequest
	}

	userID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}

	currentUser, err := s.authService.GetCurrentUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	profile := s.resolveAgentProfile(ctx, request.Messages, false)
	messages := s.buildConversation(currentUser, request.Messages, false, profile)
	return s.runChatLoop(ctx, currentUser, messages, request.MaxTokens, profile, false)
}

// ChatPublic runs a limited public chat loop without authenticated user context.
func (s *Service) ChatPublic(ctx context.Context, request assistantdto.ChatRequest) (*assistantdto.ChatResponse, error) {
	if len(request.Messages) == 0 {
		return nil, apperr.ErrBadRequest
	}

	profile := s.resolveAgentProfile(ctx, request.Messages, true)
	messages := s.buildConversation(nil, request.Messages, true, profile)
	return s.runChatLoop(ctx, nil, messages, request.MaxTokens, profile, true)
}

func (s *Service) runChatLoop(ctx context.Context, currentUser *authdto.AuthUserView, messages []map[string]any, maxTokens int, profile assistantAgentProfile, isPublic bool) (*assistantdto.ChatResponse, error) {
	inferredActions := inferClientActions(messages)
	queuedActions := make([]assistantdto.ClientAction, 0, 4)
	if shouldHandleActionsDirectly(messages, inferredActions) {
		return &assistantdto.ChatResponse{
			Content:    buildActionFallbackReply(inferredActions, ""),
			Actions:    inferredActions,
			Agent:      string(profile.Intent),
			AgentLabel: profile.Label,
		}, nil
	}

	if len(inferredActions) == 0 && shouldPreferPlainAssistantResponse(messages, isPublic) {
		content, err := s.requestPlainModel(ctx, messages, resolvePlainChatMaxTokens(maxTokens))
		if err == nil && strings.TrimSpace(content) != "" {
			return &assistantdto.ChatResponse{
				Content:    strings.TrimSpace(content),
				Agent:      string(profile.Intent),
				AgentLabel: profile.Label,
			}, nil
		}
	}

	if len(inferredActions) == 0 && !isPublic && shouldPreferStructuredAssistantResponse(profile.Intent) {
		if structured := s.buildStructuredModelFallbackResponse(ctx, currentUser, messages, maxTokens, profile, isPublic, nil); structured != nil {
			return structured, nil
		}
	}

	if !s.aiCfg.Enabled() {
		return s.buildLocalFallbackChatResponse(messages, inferredActions, profile), nil
	}

	for round := 0; round < maxToolRounds; round++ {
		reply, err := s.requestModel(ctx, messages, maxTokens, profile, isPublic)
		if err != nil {
			if len(queuedActions) == 0 && len(inferredActions) == 0 && shouldPreferPlainAssistantResponse(messages, isPublic) {
				content, plainErr := s.requestPlainModel(ctx, messages, resolvePlainChatMaxTokens(maxTokens))
				if plainErr == nil && strings.TrimSpace(content) != "" {
					return &assistantdto.ChatResponse{
						Content:    strings.TrimSpace(content),
						Agent:      string(profile.Intent),
						AgentLabel: profile.Label,
					}, nil
				}
			}
			if fallback := s.buildStructuredModelFallbackResponse(ctx, currentUser, messages, maxTokens, profile, isPublic, mergeClientActions(queuedActions, inferredActions)); fallback != nil {
				return fallback, nil
			}
			return s.buildLocalFallbackChatResponse(messages, mergeClientActions(queuedActions, inferredActions), profile), nil
		}

		if len(reply.ToolCalls) == 0 {
			content := strings.TrimSpace(extractAssistantContent(reply.Content))
			if content == "" {
				if fallback := s.buildStructuredModelFallbackResponse(ctx, currentUser, messages, maxTokens, profile, isPublic, mergeClientActions(queuedActions, inferredActions)); fallback != nil {
					return fallback, nil
				}
				return s.buildLocalFallbackChatResponse(messages, mergeClientActions(queuedActions, inferredActions), profile), nil
			}
			if round == 0 && len(queuedActions) == 0 {
				if fallbackActions := inferClientActions(messages); len(fallbackActions) > 0 {
					return &assistantdto.ChatResponse{
						Content:    buildActionFallbackReply(fallbackActions, content),
						Actions:    fallbackActions,
						Agent:      string(profile.Intent),
						AgentLabel: profile.Label,
					}, nil
				}
			}
			return &assistantdto.ChatResponse{
				Content:    content,
				Actions:    dedupeClientActions(queuedActions),
				Agent:      string(profile.Intent),
				AgentLabel: profile.Label,
			}, nil
		}

		messages = append(messages, map[string]any{
			"role":       "assistant",
			"content":    extractAssistantContent(reply.Content),
			"tool_calls": reply.ToolCalls,
		})

		for _, toolCall := range reply.ToolCalls {
			toolResult, actions := s.executeTool(ctx, currentUser, toolCall.Function.Name, toolCall.Function.Arguments)
			if len(actions) > 0 {
				queuedActions = append(queuedActions, actions...)
			}
			messages = append(messages, map[string]any{
				"role":         "tool",
				"tool_call_id": toolCall.ID,
				"content":      toolResult,
			})
		}
	}

	if fallback := s.buildStructuredModelFallbackResponse(ctx, currentUser, messages, maxTokens, profile, isPublic, mergeClientActions(queuedActions, inferredActions)); fallback != nil {
		return fallback, nil
	}
	return s.buildLocalFallbackChatResponse(messages, mergeClientActions(queuedActions, inferredActions), profile), nil
}

func (s *Service) buildConversation(currentUser *authdto.AuthUserView, original []map[string]any, isPublic bool, profile assistantAgentProfile) []map[string]any {
	messages := make([]map[string]any, 0, len(original)+1)
	messages = append(messages, map[string]any{
		"role":    "system",
		"content": s.buildSystemPrompt(currentUser, isPublic, profile),
	})
	messages = append(messages, normalizeUserMessages(original)...)
	return messages
}

func (s *Service) buildSystemPrompt(currentUser *authdto.AuthUserView, isPublic bool, profile assistantAgentProfile) string {
	lines := []string{
		fmt.Sprintf("你是 course-recommend-agents 项目内的 %s。", profile.Label),
		"你不是泛化聊天助手，优先结合系统已有数据、知识图谱和工具结果回答问题。",
		fmt.Sprintf("当前时间：%s。", time.Now().Format(time.RFC3339)),
	}

	if isPublic {
		lines = append(lines,
			"当前会话未登录，只能使用公共知识图谱、公共知识详情、前沿摘要和页面 UI 动作工具。",
			"不要声称你拿到了用户个人学习数据、错题记录、能力画像或推荐路径；这些在未登录状态下不可用。",
		)
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
		"当用户询问当前情况、最近表现、薄弱点、推荐原因、下一步学习什么、某知识点和用户的关系、最近错题或数据库中有什么记录时，必须先调用相关工具，不能凭空猜测。",
		"如果用户问数据来源，要明确写出来自 MySQL 学习记录、Redis 实时状态、Neo4j 知识图谱，还是推荐/前沿服务。",
		"如果工具没有查到结果，要如实说明当前系统没有查到对应数据。",
		"回答使用中文，优先输出：结论、依据、下一步建议。",
		"不要编造学生状态，不要伪造知识点和掌握度。",
		"当用户明确要求你控制图谱或页面操作时，应调用 UI 动作工具，例如聚焦知识点、展开节点、高亮前置链、适配视图、重置场景或打开详情页。",
		"如果只是解释知识或给学习建议，不要为了显得聪明而乱动页面。",
		"在 UI 动作工具被调用后，最终回答要简短说明你已经做了什么，并提醒用户接下来可以看哪里。",
	)

	lines = append(lines, buildAgentPrompt(profile, isPublic)...)
	lines = append(lines, buildLearnerStylePrompt(profile.LearnerProfile)...)

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

func (s *Service) requestModel(ctx context.Context, messages []map[string]any, maxTokens int, profile assistantAgentProfile, isPublic bool) (*aiAssistantMessage, error) {
	var lastErr error
	for _, provider := range s.aiCfg.Providers() {
		reply, err := s.requestModelWithProvider(ctx, provider, messages, maxTokens, profile, isPublic)
		if err == nil {
			return reply, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, apperr.ErrDependencyUnavailable.WithCause(errors.New("assistant provider is not configured"))
}

func (s *Service) requestModelWithProvider(ctx context.Context, provider config.AIProviderConfig, messages []map[string]any, maxTokens int, profile assistantAgentProfile, isPublic bool) (*aiAssistantMessage, error) {
	payload := map[string]any{
		"model":       s.selectModel(provider, messages),
		"messages":    messages,
		"tools":       buildAgentTools(profile.Intent, isPublic),
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

	baseURL := strings.TrimRight(provider.BaseURL, "/")
	requestCtx, cancel := context.WithTimeout(ctx, s.chatProviderTimeout(provider))
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+provider.APIKey)

	response, err := s.doModelRequest(request, s.providerRetries())
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

func (s *Service) providerRetries() int {
	if len(s.aiCfg.Providers()) > 1 {
		return 0
	}
	return 1
}

func (s *Service) chatProviderTimeout(provider config.AIProviderConfig) time.Duration {
	baseURL := strings.ToLower(strings.TrimSpace(provider.BaseURL))

	switch {
	case strings.Contains(baseURL, "api.openai.com"):
		if len(s.aiCfg.Providers()) > 1 {
			return 8 * time.Second
		}
		return 14 * time.Second
	case strings.Contains(baseURL, "api.deepseek.com"):
		if len(s.aiCfg.Providers()) > 1 {
			return 14 * time.Second
		}
		return 28 * time.Second
	case strings.Contains(baseURL, "openclaudecode.cn"):
		if len(s.aiCfg.Providers()) > 1 {
			return 14 * time.Second
		}
		return 24 * time.Second
	case len(s.aiCfg.Providers()) > 1:
		return 12 * time.Second
	default:
		return 20 * time.Second
	}
}

func shouldPreferStructuredAssistantResponse(intent agentIntent) bool {
	switch intent {
	case agentIntentPlanner, agentIntentAssessor:
		return true
	default:
		return false
	}
}

func (s *Service) selectModel(provider config.AIProviderConfig, messages []map[string]any) string {
	if messagesIncludeImage(messages) && strings.TrimSpace(provider.VisionModel) != "" {
		return provider.VisionModel
	}
	return provider.ChatModel
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

func (s *Service) resolveAgentProfile(ctx context.Context, messages []map[string]any, isPublic bool) assistantAgentProfile {
	intent := classifyIntent(lastUserMessage(messages))
	learnerProfile := s.lookupLearnerProfile(ctx, isPublic)
	switch intent {
	case agentIntentPlanner:
		return assistantAgentProfile{
			Intent:         agentIntentPlanner,
			Name:           "Planner",
			Label:          "学习规划助手",
			LearnerProfile: learnerProfile,
		}
	case agentIntentAssessor:
		return assistantAgentProfile{
			Intent:         agentIntentAssessor,
			Name:           "Assessor",
			Label:          "评估诊断助手",
			LearnerProfile: learnerProfile,
		}
	default:
		return assistantAgentProfile{
			Intent:         agentIntentExplorer,
			Name:           "Explorer",
			Label:          "知识探索助手",
			LearnerProfile: learnerProfile,
		}
	}
}

func classifyIntent(message string) agentIntent {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return agentIntentExplorer
	}

	planningKeywords := []string{"下一步", "推荐", "学什么", "路线", "路径", "规划", "计划", "安排", "如何学"}
	assessingKeywords := []string{"进度", "掌握", "薄弱", "复盘", "表现", "情况", "分析", "评估", "诊断", "统计"}
	exploringKeywords := []string{"解释", "是什么", "什么是", "对比", "区别", "原理", "为什么", "怎么理解", "联系"}

	if containsAnyKeyword(normalized, planningKeywords) {
		return agentIntentPlanner
	}
	if containsAnyKeyword(normalized, assessingKeywords) {
		return agentIntentAssessor
	}
	if containsAnyKeyword(normalized, exploringKeywords) {
		return agentIntentExplorer
	}
	return agentIntentExplorer
}

func lastUserMessage(messages []map[string]any) string {
	for index := len(messages) - 1; index >= 0; index-- {
		role, _ := messages[index]["role"].(string)
		if strings.TrimSpace(role) != "user" {
			continue
		}
		return extractAssistantContent(messages[index]["content"])
	}
	return ""
}

func containsAnyKeyword(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(strings.TrimSpace(keyword))) {
			return true
		}
	}
	return false
}

func shouldPreferPlainAssistantResponse(messages []map[string]any, isPublic bool) bool {
	userMessage := strings.TrimSpace(strings.ToLower(lastUserMessage(messages)))
	if userMessage == "" {
		return false
	}

	explanationSignals := []string{
		"解释", "讲解", "介绍", "总结", "概括", "是什么", "两句话", "一句话", "为什么", "怎么理解", "对比", "区别",
	}
	if !containsAnyKeyword(userMessage, explanationSignals) {
		return false
	}

	toolHeavySignals := []string{
		"学习进度", "掌握度", "错题", "推荐路径", "学习路径", "数据库", "前沿", "论文", "路径", "图谱关系", "最短路径",
	}
	if containsAnyKeyword(userMessage, toolHeavySignals) {
		return false
	}

	for _, message := range messages {
		if strings.TrimSpace(stringValue(message["role"])) != "system" {
			continue
		}
		content := strings.TrimSpace(extractAssistantContent(message["content"]))
		if strings.Contains(content, "当前知识点：") || strings.Contains(content, "课程：") {
			return true
		}
	}

	if isPublic && countConversationMessages(messages) <= 2 {
		return true
	}

	return countConversationMessages(messages) <= 1
}

func resolvePlainChatMaxTokens(maxTokens int) int {
	if maxTokens > 0 {
		return maxTokens
	}
	return 320
}

func countConversationMessages(messages []map[string]any) int {
	total := 0
	for _, message := range messages {
		switch strings.TrimSpace(stringValue(message["role"])) {
		case "user", "assistant":
			total++
		}
	}
	return total
}

func buildAgentPrompt(profile assistantAgentProfile, isPublic bool) []string {
	switch profile.Intent {
	case agentIntentPlanner:
		return []string{
			"当前角色侧重：面向未来的学习规划。",
			"优先给出清晰的学习顺序、阶段目标和 2 到 4 步可执行计划。",
			"当推荐路径、能力画像或学习进度可用时，必须基于这些结果制定计划。",
			"如果用户只问一个概念，也可以简短解释，但最后仍要落到下一步行动。",
		}
	case agentIntentAssessor:
		return []string{
			"当前角色侧重：数据驱动的学习评估与诊断。",
			"优先客观呈现学习状态、掌握度、薄弱点、最近表现和风险，不要只给鸡汤式建议。",
			"涉及进度、掌握度、错题、复盘、能力画像时，优先调用数据库、学习进度、能力画像、首页摘要工具。",
			"结论要区分“系统已有证据”和“你的推断”。",
		}
	default:
		lines := []string{
			"当前角色侧重：知识探索与图谱推理。",
			"优先通过知识图谱、知识详情、前沿摘要和图谱路径工具来解释概念之间的关系。",
			"默认采用苏格拉底式教学：如果用户没有明确说“直接告诉我”或“直接讲”，先提出 1 个简短、启发式的问题，再给出解释。",
			"引导问题应尽量借助学生已掌握的前置知识或相邻知识作为类比起点，避免连续抛出多个问题。",
			"当用户明确要求直接讲解、赶时间或只要结论时，立刻切换成直接解释模式。",
		}
		if isPublic {
			lines = append(lines, "当前未登录，不能声称你掌握了该学生的真实学习记录，但仍可用图谱关系做启发式引导。")
		} else {
			lines = append(lines, "如果需要寻找类比起点，可先调用 check_student_prior_knowledge 工具，再决定从哪里开始引导。")
		}
		return lines
	}
}

func (s *Service) buildStructuredModelFallbackResponse(ctx context.Context, currentUser *authdto.AuthUserView, messages []map[string]any, maxTokens int, profile assistantAgentProfile, isPublic bool, actions []assistantdto.ClientAction) *assistantdto.ChatResponse {
	if !s.aiCfg.Enabled() {
		return nil
	}

	payload := s.buildStructuredModelFallbackPayload(ctx, currentUser, messages, profile, isPublic)
	if len(payload) == 0 {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil
	}

	content, err := s.requestPlainModel(ctx, []map[string]any{
		{
			"role":    "system",
			"content": buildStructuredModelFallbackPrompt(profile, isPublic),
		},
		{
			"role":    "user",
			"content": string(body),
		},
	}, resolvePlainChatMaxTokens(maxTokens))
	if err != nil || strings.TrimSpace(content) == "" {
		content = strings.TrimSpace(buildStructuredLocalFallbackReply(payload, profile))
		if content == "" {
			return nil
		}
		return &assistantdto.ChatResponse{
			Content:    content,
			Actions:    dedupeClientActions(actions),
			Agent:      string(profile.Intent),
			AgentLabel: profile.Label,
		}
	}

	return &assistantdto.ChatResponse{
		Content:    strings.TrimSpace(content),
		Actions:    dedupeClientActions(actions),
		Agent:      string(profile.Intent),
		AgentLabel: profile.Label,
	}
}

func (s *Service) buildStructuredModelFallbackPayload(ctx context.Context, currentUser *authdto.AuthUserView, messages []map[string]any, profile assistantAgentProfile, isPublic bool) map[string]any {
	payload := map[string]any{
		"question":    strings.TrimSpace(lastUserMessage(messages)),
		"pageContext": extractLocalChatContext(messages),
		"isPublic":    isPublic,
	}

	if currentUser != nil {
		payload["currentUser"] = compactCurrentUserPayload(s.toolGetCurrentUser(currentUser))
	}

	switch profile.Intent {
	case agentIntentPlanner:
		if currentUser != nil {
			if pathPayload, err := s.toolGetRecommendedPath(ctx, `{"scene":"assistant","limit":3}`); err == nil {
				payload["recommendedPath"] = compactRecommendedPathPayload(pathPayload)
			}
			if progressPayload, err := s.toolGetLearningProgress(ctx); err == nil {
				payload["learningProgress"] = compactLearningProgressPayload(progressPayload)
			}
			if competencyPayload, err := s.toolGetCompetencyProfile(ctx, currentUser); err == nil {
				payload["competencyProfile"] = compactCompetencyPayload(competencyPayload)
			}
		}
	case agentIntentAssessor:
		if currentUser != nil {
			if progressPayload, err := s.toolGetLearningProgress(ctx); err == nil {
				payload["learningProgress"] = compactLearningProgressPayload(progressPayload)
			}
			if competencyPayload, err := s.toolGetCompetencyProfile(ctx, currentUser); err == nil {
				payload["competencyProfile"] = compactCompetencyPayload(competencyPayload)
			}
			if dashboardPayload, err := s.toolGetDashboardSummary(ctx, currentUser); err == nil {
				payload["dashboardSummary"] = compactDashboardPayload(dashboardPayload)
			}
		}
	default:
		if currentUser != nil {
			if progressPayload, err := s.toolGetLearningProgress(ctx); err == nil {
				payload["learningProgress"] = compactLearningProgressPayload(progressPayload)
			}
		}
	}

	if len(payload) == 3 && currentUser == nil {
		return nil
	}
	return payload
}

func buildStructuredModelFallbackPrompt(profile assistantAgentProfile, isPublic bool) string {
	lines := []string{
		fmt.Sprintf("你是 course-recommend-agents 项目内的 %s。", profile.Label),
		"工具调用当前不可用，但系统已经为你整理了一份结构化上下文 JSON。",
		"你必须严格依据 JSON 回答，不能编造系统里没有的数据。",
		"回答使用中文，优先输出：结论、依据、下一步建议。",
		"如果 JSON 里的字段为空或缺失，要明确说当前系统没有查到对应记录。",
	}

	switch profile.Intent {
	case agentIntentPlanner:
		lines = append(lines,
			"当前角色侧重：学习规划。",
			"优先基于 recommendedPath、learningProgress、competencyProfile 给出 2 到 4 步的下一步建议。",
		)
	case agentIntentAssessor:
		lines = append(lines,
			"当前角色侧重：学习诊断。",
			"优先指出已有证据支持的强项、薄弱点和风险，不要只给泛泛建议。",
		)
	default:
		lines = append(lines,
			"当前角色侧重：知识探索。",
			"优先围绕 pageContext 里的当前知识点、前置知识和相邻关系做解释。",
		)
	}

	if isPublic {
		lines = append(lines, "当前未登录，不能声称掌握了学生个人学习记录。")
	}

	return strings.Join(lines, "\n")
}

func compactCurrentUserPayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return nil
	}
	return map[string]any{
		"source":      payload["source"],
		"user":        payload["user"],
		"generatedAt": payload["generatedAt"],
	}
}

func compactRecommendedPathPayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return nil
	}
	return map[string]any{
		"source":      payload["source"],
		"path":        payload["path"],
		"generatedAt": payload["generatedAt"],
	}
}

func compactLearningProgressPayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return nil
	}
	return map[string]any{
		"source":           payload["source"],
		"learningSnapshot": payload["learningSnapshot"],
		"weakKnowledge":    payload["weakKnowledge"],
		"strongKnowledge":  payload["strongKnowledge"],
		"generatedAt":      payload["generatedAt"],
	}
}

func compactCompetencyPayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return nil
	}
	return map[string]any{
		"source":        payload["source"],
		"strongestAxes": payload["strongestAxes"],
		"weakestAxes":   payload["weakestAxes"],
		"generatedAt":   payload["generatedAt"],
	}
}

func compactDashboardPayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return nil
	}
	return map[string]any{
		"source":          payload["source"],
		"summary":         payload["summary"],
		"recommendedPath": payload["recommendedPath"],
		"generatedAt":     payload["generatedAt"],
	}
}

func buildStructuredLocalFallbackReply(payload map[string]any, profile assistantAgentProfile) string {
	switch profile.Intent {
	case agentIntentPlanner:
		return buildPlannerLocalFallbackReply(payload)
	case agentIntentAssessor:
		return buildAssessorLocalFallbackReply(payload)
	default:
		return ""
	}
}

func buildPlannerLocalFallbackReply(payload map[string]any) string {
	pathItems := extractRecommendedPathItems(payload["recommendedPath"])
	weakKnowledge := extractLearningProgressItems(payload["learningProgress"], "weakKnowledge")
	weakAxes := extractRadarAxes(payload["competencyProfile"], "weakestAxes")

	lines := []string{
		"在线 AI 当前连接不稳定，我先基于系统里的推荐结果和学习数据给你规划下一步。",
	}

	if len(pathItems) > 0 {
		first := pathItems[0]
		lines = append(lines, "结论：建议先从「"+strings.TrimSpace(first.Title)+"」开始。")
		if detail := strings.TrimSpace(first.Detail); detail != "" {
			lines = append(lines, "推荐依据："+detail)
		}
		if len(pathItems) > 1 {
			nextTitles := make([]string, 0, len(pathItems)-1)
			for _, item := range pathItems[1:] {
				title := strings.TrimSpace(item.Title)
				if title == "" {
					continue
				}
				nextTitles = append(nextTitles, title)
			}
			if len(nextTitles) > 0 {
				lines = append(lines, "后续衔接："+strings.Join(nextTitles, " -> "))
			}
		}
	}

	if len(weakKnowledge) > 0 {
		lines = append(lines, "当前更需要补的知识点："+strings.Join(learningProgressNames(weakKnowledge, 3), "、"))
	}
	if len(weakAxes) > 0 {
		lines = append(lines, "薄弱能力轴："+strings.Join(radarAxisNames(weakAxes, 3), "、"))
	}

	lines = append(lines, "下一步建议：先完成当前第一推荐节点，再回图谱里展开它的一跳邻居，把前置知识和后续去向一起看清楚。")
	return strings.Join(lines, "\n\n")
}

func buildAssessorLocalFallbackReply(payload map[string]any) string {
	weakKnowledge := extractLearningProgressItems(payload["learningProgress"], "weakKnowledge")
	strongKnowledge := extractLearningProgressItems(payload["learningProgress"], "strongKnowledge")
	weakAxes := extractRadarAxes(payload["competencyProfile"], "weakestAxes")
	strongAxes := extractRadarAxes(payload["competencyProfile"], "strongestAxes")

	lines := []string{
		"在线 AI 当前连接不稳定，我先基于系统里的学习记录和能力画像给你做一次诊断。",
	}

	if len(weakKnowledge) > 0 {
		lines = append(lines, "当前薄弱知识点："+strings.Join(learningProgressNames(weakKnowledge, 3), "、"))
	}
	if len(strongKnowledge) > 0 {
		lines = append(lines, "当前相对稳定的知识点："+strings.Join(learningProgressNames(strongKnowledge, 3), "、"))
	}
	if len(weakAxes) > 0 {
		lines = append(lines, "薄弱能力轴："+strings.Join(radarAxisNames(weakAxes, 3), "、"))
	}
	if len(strongAxes) > 0 {
		lines = append(lines, "优势能力轴："+strings.Join(radarAxisNames(strongAxes, 3), "、"))
	}

	lines = append(lines, "建议：优先围绕薄弱知识点做一次小范围复习，再结合图谱前置链检查是不是卡在基础概念上。")
	return strings.Join(lines, "\n\n")
}

func extractRecommendedPathItems(raw any) []recommenddto.PathItemView {
	mapped, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	items, ok := mapped["path"].([]recommenddto.PathItemView)
	if !ok {
		return nil
	}
	return items
}

func extractLearningProgressItems(raw any, key string) []learningdto.LearningProgressItem {
	mapped, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	items, ok := mapped[key].([]learningdto.LearningProgressItem)
	if !ok {
		return nil
	}
	return items
}

func extractRadarAxes(raw any, key string) []competencydto.RadarAxis {
	mapped, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	items, ok := mapped[key].([]competencydto.RadarAxis)
	if !ok {
		return nil
	}
	return items
}

func learningProgressNames(items []learningdto.LearningProgressItem, limit int) []string {
	if limit <= 0 || len(items) == 0 {
		return nil
	}

	names := make([]string, 0, min(limit, len(items)))
	for _, item := range items {
		name := strings.TrimSpace(item.CanonicalName)
		if name == "" {
			name = strings.TrimSpace(item.KnowledgeID)
		}
		if name == "" {
			continue
		}
		names = append(names, name)
		if len(names) >= limit {
			break
		}
	}
	return names
}

func radarAxisNames(items []competencydto.RadarAxis, limit int) []string {
	if limit <= 0 || len(items) == 0 {
		return nil
	}

	names := make([]string, 0, min(limit, len(items)))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
		if len(names) >= limit {
			break
		}
	}
	return names
}

func (s *Service) lookupLearnerProfile(ctx context.Context, isPublic bool) *recommenddto.LearnerProfileView {
	if isPublic || s.recommendService == nil {
		return nil
	}
	response, err := s.recommendService.GetPath(ctx, recommenddto.PathQuery{
		Scene: "dashboard",
		Limit: 1,
	})
	if err != nil || response == nil || response.LearnerProfile == nil {
		return nil
	}
	return response.LearnerProfile
}

func buildLearnerStylePrompt(profile *recommenddto.LearnerProfileView) []string {
	if profile == nil {
		return nil
	}

	lines := []string{
		fmt.Sprintf("当前推断的学习风格：%s（%s）。", strings.TrimSpace(profile.Label), strings.TrimSpace(profile.Style)),
		fmt.Sprintf("风格摘要：%s", strings.TrimSpace(profile.Summary)),
	}

	switch strings.TrimSpace(profile.Style) {
	case "systematic":
		lines = append(lines,
			"回答时尽量按知识树、先修顺序和分层结构来组织内容。",
			"推荐步骤时优先给出稳定的线性推进路线，而不是太多分叉选项。",
		)
	case "exploratory":
		lines = append(lines,
			"回答时可以多给跨课程联系、类比和有趣入口，帮助用户保持探索感。",
			"推荐时保留 1 到 2 个备选方向，让用户可以选择自己更感兴趣的切入点。",
		)
	case "focused":
		lines = append(lines,
			"回答时尽量直奔主题，减少不必要的铺垫和分支信息。",
			"推荐时优先指出当前最关键的一步，而不是同时给太多任务。",
		)
	case "struggling":
		lines = append(lines,
			"回答时使用鼓励但诚实的语气，把复杂问题拆成更小的步骤。",
			"如果发现理解门槛较高，优先回到基础定义、最小例子和先修补齐，而不是直接推高级扩展。",
		)
	default:
		lines = append(lines, "回答时保持清晰、具体，并尽量贴合当前用户的学习节奏。")
	}

	return lines
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func dedupeClientActions(actions []assistantdto.ClientAction) []assistantdto.ClientAction {
	if len(actions) == 0 {
		return nil
	}

	result := make([]assistantdto.ClientAction, 0, len(actions))
	seen := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		action.Type = strings.TrimSpace(action.Type)
		action.Target = strings.TrimSpace(action.Target)
		action.CourseTitle = strings.TrimSpace(action.CourseTitle)
		if action.Type == "" {
			continue
		}
		key := strings.Join([]string{
			action.Type,
			action.Target,
			action.CourseTitle,
			fmt.Sprintf("%t", action.Flash),
		}, "::")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, action)
	}

	if len(result) == 0 {
		return nil
	}
	return result
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
