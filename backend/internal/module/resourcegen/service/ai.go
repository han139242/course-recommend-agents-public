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
// context for a resource type, including an optional difficulty adjustment.
func resourceUserPrompt(resourceType string, genCtx generationContext) string {
	prompt := "请基于以下学习上下文，为学生生成「" + resourceTypeLabel(resourceType) + "」。\n\n" +
		resourceContextBlock(genCtx) +
		"\n\n要求：内容个性化、结构清晰、可直接用于学习。"
	if instruction := difficultyInstruction(genCtx.Difficulty); instruction != "" {
		prompt += "\n" + instruction
	}
	return prompt
}

// difficultyInstruction returns an extra writer instruction when the learner
// asked to regenerate a resource at an easier or harder level.
func difficultyInstruction(difficulty string) string {
	switch difficulty {
	case "easier":
		return "本次请降低难度：使用更基础直白的语言，增加分步骤解释、最小示例和先修铺垫，减少综合性挑战。"
	case "harder":
		return "本次请提高难度：在已掌握基础的前提下，增加综合应用、对比分析和挑战性问题，并适当延伸到真实场景。"
	default:
		return ""
	}
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
	if digest := strings.TrimSpace(genCtx.ImageDigest); digest != "" {
		fmt.Fprintf(&builder, "图片识别内容（来自学生上传图片，由 VisionReaderAgent 提取）：%s\n", digest)
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

	system := "你是多智能体资源生成系统中的 SafetyReviewerAgent，负责审查生成内容的事实性、防幻觉、来源可追溯、敏感合规和教学适配性。" +
		"只输出 JSON，结构为 {\"status\":\"passed|attention_required|blocked\",\"score\":0-100," +
		"\"checks\":[{\"key\":\"\",\"name\":\"\",\"status\":\"passed|warning|blocked\",\"conclusion\":\"\"}]," +
		"\"risks\":[\"\"],\"suggestions\":[\"\"]}。检查项至少覆盖事实性与防幻觉、来源可追溯、敏感与违规过滤、格式完整性、教学适配性。" +
		"出现伪造来源、明显错误事实、危险代码或无法验证的具体论文/链接时必须 blocked；缺少来源时至少 warning。使用简体中文。"
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

	status := normalizeSafetyStatus(parsed.Status, score)
	for _, check := range parsed.Checks {
		if strings.EqualFold(check.Status, "blocked") {
			status = "blocked"
		}
	}
	return resourcegendto.SafetyReportView{
		ResourceID:   resource.ID,
		Title:        resource.Title,
		Status:       status,
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

// aiQuiz generates a structured single-choice quiz with the large model. It
// returns false when the model is unavailable or the output cannot be parsed so
// the caller can fall back to a local template quiz.
func (s *Service) aiQuiz(reqCtx context.Context, knowledgeName string, courseTitle string, count int) ([]resourcegendto.QuizQuestionView, bool) {
	if s.aiClient == nil || !s.aiClient.Enabled() {
		return nil, false
	}

	system := "你是多智能体资源生成系统中的 QuestionAgent，负责为知识点出结构化单选练习题。" +
		"只输出 JSON，结构为 {\"questions\":[{\"stem\":\"\",\"options\":[\"\",\"\",\"\",\"\"],\"answer\":0,\"explanation\":\"\"}]}。" +
		"每题恰好 4 个选项，answer 为正确选项下标（从 0 开始），explanation 说明为什么正确并提示易错点。" + antiHallucinationRules
	user := fmt.Sprintf("请基于课程《%s》的知识点「%s」出 %d 道单选练习题，难度循序渐进。", firstNonEmpty(courseTitle, "相关课程"), knowledgeName, count)

	raw, err := s.aiClient.Complete(reqCtx, ai.Request{
		System:      system,
		User:        user,
		MaxTokens:   1400,
		Temperature: 0.5,
		JSON:        true,
		Timeout:     40 * time.Second,
	})
	if err != nil {
		return nil, false
	}

	var parsed struct {
		Questions []resourcegendto.QuizQuestionView `json:"questions"`
	}
	if err := json.Unmarshal([]byte(stripCodeFence(raw)), &parsed); err != nil {
		return nil, false
	}
	questions := sanitizeQuizQuestions(parsed.Questions)
	if len(questions) == 0 {
		return nil, false
	}
	return questions, true
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

// aiImageDigest runs the VisionReaderAgent: it reads the learner-uploaded images
// with the vision model and returns a concise, grounded text digest used as
// generation context. It returns an empty string when vision is unavailable,
// there are no images, or the call fails.
func (s *Service) aiImageDigest(reqCtx context.Context, genCtx generationContext) string {
	if s.aiClient == nil || !s.aiClient.VisionEnabled() || len(genCtx.Images) == 0 {
		return ""
	}
	system := "你是多智能体资源生成系统中的 VisionReaderAgent，负责阅读学生上传的图片（教材页、题目截图或手写笔记）。" +
		antiHallucinationRules +
		"请用简体中文输出结构化要点：1) 图片涉及的知识点或主题；2) 关键概念、公式或题目；3) 可用于生成个性化学习资源的要点。只描述图片中真实存在的内容，不要编造。"
	content, err := s.aiClient.Complete(reqCtx, ai.Request{
		System: system,
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: "请阅读以下图片，并结合学习上下文提取要点。\n\n" + resourceContextBlock(genCtx),
				Images:  genCtx.Images,
			},
		},
		MaxTokens:   700,
		Temperature: 0.3,
		Timeout:     45 * time.Second,
	})
	if err != nil {
		return ""
	}
	return clampRunes(strings.TrimSpace(content), 1500)
}

// aiRecommendations asks the planning agent to propose the next resource packages
// based on the learner profile and accumulated feedback (ratings, difficulty,
// completion and comments), so recommendations adapt dynamically. It returns
// ok=false when the model is unavailable or its output cannot be used, letting
// the caller fall back to the heuristic recommendations.
func (s *Service) aiRecommendations(reqCtx context.Context, profileSummary string, dims map[string]string, resources []resourcegenentity.GeneratedResource, feedback []resourcegenentity.ResourceFeedback, summary resourcegendto.FeedbackSummaryView) ([]resourcegendto.ResourceRecommendationView, bool) {
	if s.aiClient == nil || !s.aiClient.Enabled() {
		return nil, false
	}

	knowledgeName, knowledgeID, courseTitle := latestResourceTarget(resources)
	target := firstNonEmpty(dims["learning_goal"], profileSummary, "围绕当前知识点完成理解、练习和迁移")

	promptContext := map[string]any{
		"profileSummary":   profileSummary,
		"learningGoal":     dims["learning_goal"],
		"knowledgeFound":   dims["knowledge_foundation"],
		"resourcePref":     dims["resource_preference"],
		"errorPref":        dims["error_preference"],
		"currentKnowledge": knowledgeName,
		"feedbackSummary": map[string]any{
			"total":          summary.Total,
			"averageRating":  summary.AverageRating,
			"helpfulCount":   summary.HelpfulCount,
			"completedCount": summary.CompletedCount,
			"difficulty":     summary.Difficulty,
		},
		"recentComments":  recentFeedbackComments(feedback, 5),
		"recentResources": recentResourceLabels(resources, 6),
		"allowedTypes":    []string{"explanation_doc", "mind_map", "practice_questions", "coding_case", "reading_material", "video_script"},
	}
	payload, err := json.Marshal(promptContext)
	if err != nil {
		return nil, false
	}

	system := "你是多智能体资源生成系统中的学习规划智能体，根据学生画像和最近的资源学习反馈，动态推荐下一步应生成的资源包。" +
		"只输出 JSON，结构为 {\"recommendations\":[{\"title\":\"\",\"knowledgeName\":\"\",\"resourceTypes\":[\"explanation_doc\"],\"reason\":\"\",\"actionText\":\"\",\"priority\":0}]}。" +
		"resourceTypes 只能取 allowedTypes 中的值；priority 为 1-100 的整数，越高越优先；最多 4 条；reason 要结合反馈数据说明为什么这样推荐；全部使用简体中文。"
	raw, err := s.aiClient.Complete(reqCtx, ai.Request{
		System:      system,
		User:        "学习上下文与反馈数据如下：\n" + string(payload),
		MaxTokens:   900,
		Temperature: 0.5,
		JSON:        true,
		Timeout:     30 * time.Second,
	})
	if err != nil {
		return nil, false
	}

	var parsed struct {
		Recommendations []struct {
			Title         string   `json:"title"`
			KnowledgeName string   `json:"knowledgeName"`
			ResourceTypes []string `json:"resourceTypes"`
			Reason        string   `json:"reason"`
			ActionText    string   `json:"actionText"`
			Priority      float64  `json:"priority"`
		} `json:"recommendations"`
	}
	if err := json.Unmarshal([]byte(stripCodeFence(raw)), &parsed); err != nil {
		return nil, false
	}
	if len(parsed.Recommendations) == 0 {
		return nil, false
	}

	result := make([]resourcegendto.ResourceRecommendationView, 0, len(parsed.Recommendations))
	for index, item := range parsed.Recommendations {
		types := normalizeResourceTypes(item.ResourceTypes)
		priority := int(item.Priority + 0.5)
		if priority < 1 {
			priority = 60
		}
		if priority > 100 {
			priority = 100
		}
		result = append(result, resourcegendto.ResourceRecommendationView{
			ID:            fmt.Sprintf("ai-rec-%d", index+1),
			Title:         firstNonEmpty(strings.TrimSpace(item.Title), "下一步资源包"),
			KnowledgeID:   knowledgeID,
			KnowledgeName: firstNonEmpty(strings.TrimSpace(item.KnowledgeName), knowledgeName),
			CourseTitle:   courseTitle,
			Target:        target,
			ResourceTypes: types,
			Reason:        firstNonEmpty(strings.TrimSpace(item.Reason), "根据最近反馈动态推荐。"),
			ActionText:    firstNonEmpty(strings.TrimSpace(item.ActionText), "生成资源包"),
			Priority:      priority,
		})
		if len(result) >= 4 {
			break
		}
	}
	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

// recentFeedbackComments returns up to limit non-empty learner comments.
func recentFeedbackComments(feedback []resourcegenentity.ResourceFeedback, limit int) []string {
	comments := make([]string, 0, limit)
	for _, item := range feedback {
		comment := strings.TrimSpace(item.Comment)
		if comment == "" {
			continue
		}
		comments = append(comments, comment)
		if len(comments) >= limit {
			break
		}
	}
	return comments
}

// recentResourceLabels returns up to limit recent "knowledge·type" labels to give
// the planner a view of what was already generated.
func recentResourceLabels(resources []resourcegenentity.GeneratedResource, limit int) []string {
	labels := make([]string, 0, limit)
	for _, resource := range resources {
		labels = append(labels, firstNonEmpty(resource.KnowledgeName, "知识点")+"·"+firstNonEmpty(resource.TypeLabel, resource.ResourceType))
		if len(labels) >= limit {
			break
		}
	}
	return labels
}
