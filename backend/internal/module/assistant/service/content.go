package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	assistantdto "github.com/han139242/course-recommend-agents/backend/internal/module/assistant/dto"
	frontierdto "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/dto"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

// GenerateKnowledgeCard builds a personalized knowledge card using retrieved
// project context rather than only front-end prompt stitching.
func (s *Service) GenerateKnowledgeCard(ctx context.Context, request assistantdto.KnowledgeCardRequest) (*assistantdto.KnowledgeCardResponse, error) {
	if !s.aiCfg.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	userID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}

	currentUser, err := s.authService.GetCurrentUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	knowledgeID := strings.TrimSpace(request.KnowledgeID)
	knowledgeDetail := map[string]any{
		"knowledgeId":   knowledgeID,
		"name":          strings.TrimSpace(request.Name),
		"domain":        strings.TrimSpace(request.Domain),
		"levelLabel":    strings.TrimSpace(request.LevelLabel),
		"description":   strings.TrimSpace(request.Description),
		"prerequisites": request.Prerequisites,
		"related":       request.Related,
		"scenarios":     request.Scenarios,
	}
	sources := []string{"request_context"}

	if s.graphService != nil {
		if detail, detailErr := s.graphService.GetKnowledgeDetail(ctx, knowledgeID); detailErr == nil && detail != nil {
			knowledgeDetail = map[string]any{
				"knowledgeId":   detail.RouteKnowledgeID,
				"name":          firstNonEmpty(detail.Name, request.Name, knowledgeID),
				"domain":        firstNonEmpty(firstTopic(detail.ParentTopics), request.Domain, detail.ChapterTitle, detail.EntityType),
				"levelLabel":    firstNonEmpty(request.LevelLabel, "核心"),
				"description":   firstNonEmpty(detail.Description, detail.Summary, request.Description),
				"prerequisites": preferStringList(detail.Prerequisites, request.Prerequisites),
				"related":       preferStringList(detail.Related, request.Related),
				"scenarios":     preferStringList(detail.Scenarios, request.Scenarios),
			}
			sources = appendUniqueString(sources, "knowledge_graph")
		}
	}

	studentState, _ := s.lookupKnowledgeProgress(ctx, knowledgeID)
	if len(studentState) > 0 {
		sources = appendUniqueString(sources, "learning_progress")
	}

	progressSummary := map[string]any{}
	if s.learningService != nil {
		if progress, _, progressErr := s.learningService.GetLearningProgress(ctx); progressErr == nil {
			progressSummary = map[string]any{
				"weakKnowledge":   pickWeakKnowledge(progress.List, 3),
				"strongKnowledge": pickStrongKnowledge(progress.List, 3),
				"overview":        summarizeProgress(progress.List),
			}
		}
	}

	recommendedPath := []recommenddto.PathItemView{}
	if s.recommendService != nil {
		if pathResp, pathErr := s.recommendService.GetPath(ctx, recommenddto.PathQuery{
			Scene:       "detail",
			KnowledgeID: knowledgeID,
			Limit:       3,
		}); pathErr == nil && pathResp != nil {
			recommendedPath = pathResp.Path
			if len(recommendedPath) > 0 {
				sources = appendUniqueString(sources, "recommend_service")
			}
		}
	}

	frontierSummary := &frontierdto.BriefingResponse{}
	if s.frontierService != nil {
		if frontierResp, frontierErr := s.frontierService.GetBriefing(ctx, frontierdto.BriefingQuery{KnowledgeID: knowledgeID, Knowledge: strings.TrimSpace(request.Name)}); frontierErr == nil && frontierResp != nil {
			frontierSummary = frontierResp
			if strings.TrimSpace(frontierResp.DigestSummary) != "" {
				sources = appendUniqueString(sources, "frontier_service")
			}
		}
	}

	systemPrompt := strings.Join([]string{
		"你是课程知识卡生成助手。",
		"你要根据项目内检索到的真实上下文，为当前学生生成个性化学习卡片。",
		"必须使用中文 Markdown 输出，并严格保留这些二级标题：",
		"## 概念解释",
		"## 关键要点",
		"## 与先修知识的关系",
		"## 应用举例",
		"## 你的下一步",
		"要求：",
		"1. 内容要贴合当前学生掌握状态，不能写成通用教材段落。",
		"2. 如果项目上下文没有数据，明确说“当前系统暂无记录”，不要编造。",
		"3. 总体简洁，但要比摘要更有针对性，建议控制在 260 到 420 字左右。",
		"4. “你的下一步”要优先结合推荐路径和学生当前弱项来写。",
	}, "\n")

	userPayload := map[string]any{
		"student": map[string]any{
			"id":     currentUser.ID,
			"name":   currentUser.Name,
			"role":   currentUser.Role,
			"target": currentUser.Target,
		},
		"knowledge":       knowledgeDetail,
		"studentState":    studentState,
		"progressSummary": progressSummary,
		"recommendedPath": recommendedPath,
		"frontierBriefing": map[string]any{
			"status":        frontierSummary.Status,
			"digestSummary": frontierSummary.DigestSummary,
			"topics":        frontierSummary.Topics,
		},
	}

	payloadJSON, err := json.Marshal(userPayload)
	if err != nil {
		return nil, apperr.ErrInternal.WithCause(err)
	}

	content, err := s.requestPlainModel(ctx, []map[string]any{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": string(payloadJSON)},
	}, 520)
	if err != nil {
		return nil, err
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, apperr.ErrDependencyUnavailable.WithCause(errors.New("knowledge card model returned empty content"))
	}

	return &assistantdto.KnowledgeCardResponse{
		Content: content,
		Sources: sources,
	}, nil
}

func (s *Service) requestPlainModel(ctx context.Context, messages []map[string]any, maxTokens int) (string, error) {
	var lastErr error
	for _, provider := range s.aiCfg.Providers() {
		content, err := s.requestPlainModelWithProvider(ctx, provider, messages, maxTokens)
		if err == nil {
			return content, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", apperr.ErrDependencyUnavailable.WithCause(errors.New("knowledge card provider is not configured"))
}

func (s *Service) requestPlainModelWithProvider(ctx context.Context, provider config.AIProviderConfig, messages []map[string]any, maxTokens int) (string, error) {
	payload := map[string]any{
		"model":    provider.ChatModel,
		"messages": messages,
		"stream":   false,
	}
	if maxTokens > 0 {
		payload["max_tokens"] = maxTokens
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", apperr.ErrInternal.WithCause(err)
	}

	baseURL := strings.TrimRight(provider.BaseURL, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", apperr.ErrDependencyUnavailable.WithCause(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+provider.APIKey)

	response, err := s.doModelRequest(request, s.providerRetries())
	if err != nil {
		return "", apperr.ErrDependencyUnavailable.WithCause(err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return "", apperr.ErrDependencyUnavailable.WithCause(fmt.Errorf("knowledge card upstream returned HTTP %d", response.StatusCode))
	}

	var parsed aiChatResponse
	if err := json.NewDecoder(response.Body).Decode(&parsed); err != nil {
		return "", apperr.ErrDependencyUnavailable.WithCause(err)
	}
	if len(parsed.Choices) == 0 {
		return "", apperr.ErrDependencyUnavailable.WithCause(errors.New("knowledge card model returned no choices"))
	}

	return extractAssistantContent(parsed.Choices[0].Message.Content), nil
}

func firstTopic(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.TrimSpace(items[0])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func preferStringList(primary []string, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func appendUniqueString(items []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return items
	}
	for _, item := range items {
		if strings.TrimSpace(item) == value {
			return items
		}
	}
	return append(items, value)
}
