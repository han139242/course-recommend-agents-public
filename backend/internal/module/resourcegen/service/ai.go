package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/ai"
	resourcegendto "github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen/dto"
	resourcegenentity "github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen/entity"
)

// antiHallucinationRules is appended to every resource-writer system prompt so
// generated content stays grounded in the supplied context and meets the A3
// anti-hallucination requirement.
const antiHallucinationRules = "严格遵守：只围绕给定知识点和上下文生成内容；不要编造不存在的事实、数据、人名、论文或链接；不确定的内容要明确标注“需进一步核实”；全程使用简体中文；语气和难度贴合学生画像。"

// aiResourceContent generates one resource's body with the large model. It
// returns an empty string when the model is unavailable, errors, or produces
// content that fails format/length validation, so callers can fall back to the
// local template.
func (s *Service) aiResourceContent(reqCtx context.Context, resourceType string, genCtx generationContext) string {
	if s.aiClient == nil || !s.aiClient.Enabled() {
		return ""
	}
	content, err := s.aiClient.Complete(reqCtx, ai.Request{
		System:      resourceSystemPrompt(resourceType),
		User:        resourceUserPrompt(resourceType, genCtx),
		MaxTokens:   1600,
		Temperature: 0.6,
		Timeout:     45 * time.Second,
	})
	if err != nil {
		return ""
	}
	return sanitizeResourceContent(resourceType, content)
}

// resourceSystemPrompt returns the role-specific writer-agent instruction for a
// resource type, including format constraints and anti-hallucination rules.
func resourceSystemPrompt(resourceType string) string {
	switch resourceType {
	case "mind_map":
		return "你是多智能体资源生成系统中的 MindMapAgent。" + antiHallucinationRules +
			"只输出 Mermaid mindmap 语法，必须以 mindmap 开头，用层级缩进表示核心概念、先修基础、应用场景和个性化目标；不要输出任何解释文字，也不要使用代码围栏。"
	case "practice_questions":
		return "你是多智能体资源生成系统中的 QuestionAgent。" + antiHallucinationRules +
			"输出 Markdown 练习题，包含单选题、判断题和简答题；每道题给出参考答案、解析、易错提醒和关联知识点。"
	case "coding_case":
		return "你是多智能体资源生成系统中的 CaseAgent。" + antiHallucinationRules +
			"输出 Markdown 代码实操案例，包含实验目标、实验步骤、核心代码（使用 ```python 代码块）、运行说明和思考题。"
	case "reading_material":
		return "你是多智能体资源生成系统中的 ReadingAgent。" + antiHallucinationRules +
			"输出 Markdown 拓展阅读材料，包含阅读路径、推荐方向（只给方向性建议，不要杜撰具体不存在的书名或链接）、阅读理由、关键词和预期收获。"
	case "video_script":
		return "你是多智能体资源生成系统中的 MultimodalAgent。" + antiHallucinationRules +
			"输出 Markdown 短视频/动画讲解脚本，包含分镜、旁白、画面建议、字幕和素材提示；第一阶段只产出脚本设计，不生成真实视频文件。"
	default:
		return "你是多智能体资源生成系统中的 ContentWriterAgent。" + antiHallucinationRules +
			"输出 Markdown 课程讲解文档，包含学习目标、核心概念、先修知识、示例、易错点和小结。"
	}
}

// resourceUserPrompt builds the user message carrying the personalized learning
// context for a resource type.
func resourceUserPrompt(resourceType string, genCtx generationContext) string {
	return "请基于以下学习上下文，为学生生成「" + resourceTypeLabel(resourceType) + "」。\n\n" +
		resourceContextBlock(genCtx) +
		"\n\n要求：内容个性化、结构清晰、可直接用于学习。"
}

// resourceContextBlock assembles the grounding context (knowledge point, course,
// goal, profile dimensions, knowledge detail, prerequisites, recommended path)
// reused across all writer-agent prompts.
func resourceContextBlock(genCtx generationContext) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "知识点：%s\n", firstNonEmpty(genCtx.KnowledgeName, "核心知识点"))
	if genCtx.CourseTitle != "" {
		fmt.Fprintf(&builder, "课程：%s\n", genCtx.CourseTitle)
	}
	if genCtx.Target != "" {
		fmt.Fprintf(&builder, "学习目标：%s\n", genCtx.Target)
	}

	dims := []struct {
		key   string
		label string
	}{
		{"knowledge_foundation", "知识基础"},
		{"learning_goal", "学习目标偏好"},
		{"learning_progress", "学习进度"},
		{"cognitive_style", "认知风格"},
		{"error_preference", "易错点偏好"},
		{"resource_preference", "资源偏好"},
	}
	for _, dim := range dims {
		if value := strings.TrimSpace(genCtx.ProfileDims[dim.key]); value != "" {
			fmt.Fprintf(&builder, "%s：%s\n", dim.label, value)
		}
	}

	if detail := detailText(genCtx.KnowledgeDetail); detail != "" {
		fmt.Fprintf(&builder, "知识点说明：%s\n", detail)
	}
	if prerequisites := prerequisiteSummary(genCtx); prerequisites != "" {
		fmt.Fprintf(&builder, "先修基础：%s\n", prerequisites)
	}
	if path := pathSummary(genCtx.Path); path != "" {
		fmt.Fprintf(&builder, "推荐学习路径：%s\n", path)
	}
	return strings.TrimSpace(builder.String())
}

// sanitizeResourceContent normalizes model output per resource type and rejects
// content that does not satisfy the expected format (returning an empty string
// to trigger the template fallback).
func sanitizeResourceContent(resourceType string, content string) string {
	trimmed := stripCodeFence(content)
	if resourceType == "mind_map" {
		if !strings.Contains(trimmed, "mindmap") {
			return ""
		}
	}
	if len([]rune(trimmed)) < 24 {
		return ""
	}
	return trimmed
}

// stripCodeFence removes a single wrapping Markdown code fence (``` or ```lang)
// when the whole content is fenced, leaving inner content intact.
func stripCodeFence(content string) string {
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

// aiSafetyReport asks the SafetyReviewerAgent (large model) to audit a resource
// and return a structured report. It returns ok=false when the model is
// unavailable or its output cannot be parsed, so callers fall back to the local
// heuristic review.
func (s *Service) aiSafetyReport(reqCtx context.Context, resource resourcegenentity.GeneratedResource) (resourcegendto.SafetyReportView, bool) {
	if s.aiClient == nil || !s.aiClient.Enabled() {
		return resourcegendto.SafetyReportView{}, false
	}

	system := "你是多智能体资源生成系统中的 SafetyReviewerAgent，负责审查生成内容的事实性、防幻觉、敏感合规和教学适配性。" +
		"只输出 JSON，结构为 {\"status\":\"passed|attention_required|blocked\",\"score\":0-100," +
		"\"checks\":[{\"key\":\"\",\"name\":\"\",\"status\":\"passed|warning|blocked\",\"conclusion\":\"\"}]," +
		"\"risks\":[\"\"],\"suggestions\":[\"\"]}。检查项至少覆盖事实性与防幻觉、敏感与违规过滤、格式完整性、教学适配性。使用简体中文。"
	user := "请审查以下学习资源内容：\n\n类型：" + resource.TypeLabel + "\n标题：" + resource.Title +
		"\n\n正文：\n" + clampRunes(resource.Content, 3000)

	raw, err := s.aiClient.Complete(reqCtx, ai.Request{
		System:      system,
		User:        user,
		MaxTokens:   800,
		Temperature: 0.2,
		JSON:        true,
		Timeout:     30 * time.Second,
	})
	if err != nil {
		return resourcegendto.SafetyReportView{}, false
	}

	var parsed struct {
		Status      string                           `json:"status"`
		Score       float64                          `json:"score"`
		Checks      []resourcegendto.SafetyCheckView `json:"checks"`
		Risks       []string                         `json:"risks"`
		Suggestions []string                         `json:"suggestions"`
	}
	if err := json.Unmarshal([]byte(stripCodeFence(raw)), &parsed); err != nil {
		return resourcegendto.SafetyReportView{}, false
	}
	if len(parsed.Checks) == 0 {
		return resourcegendto.SafetyReportView{}, false
	}

	score := int(parsed.Score + 0.5)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return resourcegendto.SafetyReportView{
		ResourceID:   resource.ID,
		Title:        resource.Title,
		Status:       normalizeSafetyStatus(parsed.Status, score),
		Score:        score,
		Checks:       parsed.Checks,
		Risks:        parsed.Risks,
		Suggestions:  ensureSafetySuggestions(parsed.Suggestions),
		ReviewedAt:   time.Now().Format(time.RFC3339),
		ReviewerName: "SafetyReviewerAgent",
	}, true
}

// normalizeSafetyStatus maps a model status to the allowed set and enforces a
// blocked verdict when the score is too low.
func normalizeSafetyStatus(status string, score int) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed", "pass", "通过":
		if score < 60 {
			return "blocked"
		}
		return "passed"
	case "blocked", "block", "阻断":
		return "blocked"
	default:
		if score < 60 {
			return "blocked"
		}
		return "attention_required"
	}
}

// ensureSafetySuggestions guarantees a minimal set of review suggestions.
func ensureSafetySuggestions(suggestions []string) []string {
	cleaned := make([]string, 0, len(suggestions)+2)
	for _, item := range suggestions {
		if strings.TrimSpace(item) != "" {
			cleaned = append(cleaned, strings.TrimSpace(item))
		}
	}
	if len(cleaned) == 0 {
		cleaned = append(cleaned,
			"展示或提交前建议由团队成员进行一次人工事实核对。",
			"若引用外部教材、论文或开源项目，应在资源中补充来源与协议说明。",
		)
	}
	return cleaned
}

// clampRunes truncates a string to at most limit runes, appending an ellipsis.
func clampRunes(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "…"
}

// resourceWriterSummary describes the writer-agent step, reflecting whether the
// large model or the local templates produced the resources.
func (s *Service) resourceWriterSummary(genCtx generationContext) string {
	labels := make([]string, 0, len(genCtx.ResourceTypes))
	for _, resourceType := range genCtx.ResourceTypes {
		labels = append(labels, resourceTypeLabel(resourceType))
	}
	joined := strings.Join(labels, "、")
	if joined == "" {
		joined = "讲解文档、思维导图、练习题、拓展阅读和实操案例"
	}
	if s.aiClient != nil && s.aiClient.Enabled() {
		return "大模型多智能体并行生成：" + joined + "。"
	}
	return "多智能体协同生成：" + joined + "。"
}

// safetyReviewerSummary describes the safety-review step, reflecting whether the
// large-model reviewer or the local heuristic is in effect.
func (s *Service) safetyReviewerSummary() string {
	if s.aiClient != nil && s.aiClient.Enabled() {
		return "由大模型 SafetyReviewerAgent 审查事实性、防幻觉、敏感合规和教学适配，并标记为可展示资源。"
	}
	return "检查资源格式、学习依据和内容安全，标记为可展示资源。"
}
