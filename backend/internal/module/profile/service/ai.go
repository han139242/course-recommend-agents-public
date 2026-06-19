package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/ai"
	profiledto "github.com/han139242/course-recommend-agents/backend/internal/module/profile/dto"
)

// requiredProfileDimensionNames lists the six fixed learner-profile dimensions
// required by the A3 spec, with their Chinese display names.
var requiredProfileDimensionNames = []struct {
	Key  string
	Name string
}{
	{"knowledge_foundation", "知识基础"},
	{"learning_goal", "学习目标"},
	{"learning_progress", "学习进度"},
	{"cognitive_style", "认知风格"},
	{"error_preference", "易错点偏好"},
	{"resource_preference", "资源偏好"},
}

// inferProfile produces the learner-profile dimensions and summary, preferring
// the large model and falling back to the local keyword inference when the model
// is unavailable or returns an incomplete result.
func (s *Service) inferProfile(ctx context.Context, message string) ([]profiledto.ProfileDimensionView, string) {
	if dims, summary := s.aiInferProfile(ctx, message); len(dims) >= len(requiredProfileDimensionNames) {
		if strings.TrimSpace(summary) == "" {
			summary = buildProfileSummary(dims)
		}
		return dims, summary
	}
	dims := inferProfileDimensions(message)
	return dims, buildProfileSummary(dims)
}

// aiInferProfile asks the model to extract the six structured profile dimensions
// from a free-text self-description. It returns nil when the model is
// unavailable or its output is missing any required dimension.
func (s *Service) aiInferProfile(ctx context.Context, message string) ([]profiledto.ProfileDimensionView, string) {
	if s.aiClient == nil || !s.aiClient.Enabled() {
		return nil, ""
	}

	system := "你是对话式学习画像分析智能体。根据学生自述，输出结构化学习画像，只输出 JSON：" +
		"{\"summary\":\"\",\"dimensions\":[{\"key\":\"\",\"name\":\"\",\"value\":\"\",\"confidence\":0-1,\"evidence\":\"\"}]}。" +
		"dimensions 必须且仅包含 6 个维度，key 依次为 knowledge_foundation、learning_goal、learning_progress、cognitive_style、error_preference、resource_preference；" +
		"name 依次用中文：知识基础、学习目标、学习进度、认知风格、易错点偏好、资源偏好；" +
		"evidence 必须引用学生自述中的依据，不要编造；confidence 取 0 到 1；全程使用简体中文。"
	user := "学生自述：" + strings.TrimSpace(message)

	raw, err := s.aiClient.Complete(ctx, ai.Request{
		System:      system,
		User:        user,
		MaxTokens:   900,
		Temperature: 0.3,
		JSON:        true,
		Timeout:     30 * time.Second,
	})
	if err != nil {
		return nil, ""
	}

	var parsed struct {
		Summary    string                            `json:"summary"`
		Dimensions []profiledto.ProfileDimensionView `json:"dimensions"`
	}
	if err := json.Unmarshal([]byte(stripJSONFence(raw)), &parsed); err != nil {
		return nil, ""
	}

	dims := normalizeAIDimensions(parsed.Dimensions)
	if dims == nil {
		return nil, ""
	}
	return dims, strings.TrimSpace(parsed.Summary)
}

// normalizeAIDimensions validates that every required dimension is present with a
// non-empty value, fills missing display names, and clamps confidence. It
// returns nil when any required dimension is missing.
func normalizeAIDimensions(input []profiledto.ProfileDimensionView) []profiledto.ProfileDimensionView {
	byKey := make(map[string]profiledto.ProfileDimensionView, len(input))
	for _, item := range input {
		key := strings.TrimSpace(item.Key)
		if key == "" || strings.TrimSpace(item.Value) == "" {
			continue
		}
		byKey[key] = item
	}

	result := make([]profiledto.ProfileDimensionView, 0, len(requiredProfileDimensionNames))
	for _, required := range requiredProfileDimensionNames {
		item, ok := byKey[required.Key]
		if !ok {
			return nil
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = required.Name
		}
		result = append(result, profiledto.ProfileDimensionView{
			Key:        required.Key,
			Name:       name,
			Value:      strings.TrimSpace(item.Value),
			Confidence: clampConfidence(item.Confidence),
			Evidence:   strings.TrimSpace(item.Evidence),
		})
	}
	return result
}

// stripJSONFence removes a single wrapping Markdown code fence around JSON output.
func stripJSONFence(content string) string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
		trimmed = trimmed[idx+1:]
	} else {
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
	}
	trimmed = strings.TrimSpace(trimmed)
	trimmed = strings.TrimSuffix(trimmed, "```")
	return strings.TrimSpace(trimmed)
}
