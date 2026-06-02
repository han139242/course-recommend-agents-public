package service

import (
	"strings"

	"github.com/han139242/course-recommend-agents/backend/internal/coursemeta"
	assistantdto "github.com/han139242/course-recommend-agents/backend/internal/module/assistant/dto"
)

func inferClientActions(messages []map[string]any) []assistantdto.ClientAction {
	latestUser := strings.TrimSpace(latestUserMessage(messages))
	if latestUser == "" {
		return nil
	}

	actions := make([]assistantdto.ClientAction, 0, 4)
	normalized := normalizeActionText(latestUser)
	primaryTarget := ""

	if action := inferGoalProfileProposal(normalized); action != nil {
		actions = append(actions, *action)
	}

	if target := inferPrerequisiteChainTarget(normalized); target != "" {
		primaryTarget = firstResolvedTarget(primaryTarget, target)
		actions = append(actions, assistantdto.ClientAction{
			Type:   "highlight_prerequisite_chain",
			Target: target,
			Flash:  true,
		})
	}

	if containsAny(normalized, "打开", "查看", "看一下", "看下", "看看", "看一眼") {
		if target := resolveCompoundActionTarget(extractActionTarget(normalized, "打开", "查看", "看一下", "看下", "看看", "看一眼"), primaryTarget); target != "" && containsAny(normalized, "详情页", "详情", "详情面板") {
			primaryTarget = firstResolvedTarget(primaryTarget, target)
			actions = append(actions, assistantdto.ClientAction{
				Type:   "open_knowledge_detail",
				Target: target,
			})
		}
	}

	if target := extractActionTarget(normalized, "展开", "扩展"); target != "" {
		primaryTarget = firstResolvedTarget(primaryTarget, target)
		actions = append(actions, assistantdto.ClientAction{
			Type:   "expand_knowledge",
			Target: target,
			Flash:  true,
		})
	}

	if containsAny(
		normalized,
		"适配视图",
		"适配当前图谱",
		"适配图谱",
		"fit view",
		"完整视图",
		"缩放到全图",
		"缩放图谱",
		"缩放视图",
		"看全图",
		"看完整图",
		"居中全图",
	) {
		actions = append(actions, assistantdto.ClientAction{Type: "fit_graph_view"})
	}

	if containsAny(normalized, "聚焦", "定位", "选中") {
		if target := resolveCompoundActionTarget(extractActionTarget(normalized, "聚焦", "定位", "选中"), primaryTarget); target != "" {
			actions = append(actions, assistantdto.ClientAction{
				Type:   "focus_knowledge",
				Target: target,
				Flash:  true,
			})
		}
	}

	if courseTitle := inferCourseFilterTarget(normalized); courseTitle != "" {
		actions = append(actions, assistantdto.ClientAction{
			Type:        "filter_course",
			CourseTitle: courseTitle,
		})
	}

	if containsAny(normalized, "重置场景", "重置图谱", "恢复全局", "恢复默认视图", "清空高亮", "重置页面") {
		actions = append(actions, assistantdto.ClientAction{Type: "reset_graph_scene"})
	}

	return dedupeClientActions(actions)
}

func inferGoalProfileProposal(message string) *assistantdto.ClientAction {
	if !looksLikeGoalProfileMessage(message) {
		return nil
	}

	payload := map[string]any{
		"gradeLevel":         inferGradeLevel(message),
		"major":              inferMajor(message),
		"goalType":           inferGoalType(message),
		"goalText":           trimGoalText(message),
		"targetDirection":    inferTargetDirection(message),
		"learnedTopics":      inferLearnedTopics(message),
		"foundationLevel":    inferFoundationLevel(message),
		"weeklyHours":        6,
		"learningPreference": inferLearningPreference(message),
	}

	return &assistantdto.ClientAction{
		Type:    "propose_goal_profile",
		Payload: payload,
	}
}

func looksLikeGoalProfileMessage(message string) bool {
	if containsAny(message, "学习目标", "制定目标", "学习计划", "规划学习", "想学", "想做", "想要做", "目标是", "我想完成") {
		return true
	}
	return containsAny(message, "大一", "大二", "大三", "大四", "研究生") &&
		containsAny(message, "学过", "学了", "会一点", "基础", "想")
}

func inferGradeLevel(message string) string {
	switch {
	case containsAny(message, "大一", "一年级"):
		return "freshman"
	case containsAny(message, "大二", "二年级"):
		return "sophomore"
	case containsAny(message, "大三", "三年级"):
		return "junior"
	case containsAny(message, "大四", "四年级"):
		return "senior"
	case containsAny(message, "研究生", "研一", "研二", "研三"):
		return "postgraduate"
	default:
		return "other"
	}
}

func inferGoalType(message string) string {
	switch {
	case containsAny(message, "项目", "作品", "系统", "应用", "智能体", "agent", "Agent"):
		return "project_building"
	case containsAny(message, "就业", "岗位", "求职", "算法工程师", "开发", "实习"):
		return "career_preparation"
	default:
		return "course_learning"
	}
}

func inferMajor(message string) string {
	candidates := []string{"人工智能", "软件工程", "计算机科学", "计算机", "数据科学", "自动化", "电子信息"}
	for _, candidate := range candidates {
		if strings.Contains(message, candidate) {
			return candidate
		}
	}
	return ""
}

func inferTargetDirection(message string) string {
	candidates := []string{
		"智能体", "大模型应用", "RAG", "算法工程师", "机器学习", "深度学习", "计算机视觉", "自然语言处理", "知识图谱", "强化学习",
	}
	for _, candidate := range candidates {
		if strings.Contains(message, candidate) {
			return candidate
		}
	}
	return ""
}

func inferLearnedTopics(message string) []string {
	candidates := []string{
		"Python", "高等数学", "线性代数", "概率论", "机器学习", "深度学习", "神经网络", "数据结构", "知识图谱", "自然语言处理",
	}
	result := make([]string, 0, 6)
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(message), strings.ToLower(candidate)) {
			result = append(result, candidate)
		}
	}
	return result
}

func inferFoundationLevel(message string) string {
	switch {
	case containsAny(message, "基础很好", "比较熟", "掌握不错", "学得还可以"):
		return "advanced"
	case containsAny(message, "一般", "一点", "学过一点", "不太稳", "薄弱"):
		return "beginner"
	default:
		return "intermediate"
	}
}

func inferLearningPreference(message string) string {
	switch {
	case containsAny(message, "项目", "实践", "做出来", "作品"):
		return "project_first"
	case containsAny(message, "考试", "考研", "期末"):
		return "exam_first"
	case containsAny(message, "理论", "原理", "公式"):
		return "theory_first"
	default:
		return "balanced"
	}
}

func trimGoalText(message string) string {
	text := strings.TrimSpace(message)
	runes := []rune(text)
	if len(runes) > 220 {
		return string(runes[:220])
	}
	return text
}

func shouldHandleActionsDirectly(messages []map[string]any, actions []assistantdto.ClientAction) bool {
	if len(actions) == 0 {
		return false
	}

	latestUser := strings.TrimSpace(latestUserMessage(messages))
	if latestUser == "" {
		return false
	}

	return !containsAny(latestUser,
		"解释",
		"讲解",
		"介绍",
		"分析",
		"总结",
		"是什么",
		"为什么",
		"如何",
		"怎么",
		"推荐",
		"建议",
		"对比",
		"区别",
		"前沿",
		"briefing",
	)
}

func latestUserMessage(messages []map[string]any) string {
	for index := len(messages) - 1; index >= 0; index-- {
		if strings.TrimSpace(stringValue(messages[index]["role"])) != "user" {
			continue
		}
		text := strings.TrimSpace(extractAssistantContent(messages[index]["content"]))
		if text != "" {
			return text
		}
	}
	return ""
}

func normalizeActionText(input string) string {
	replacer := strings.NewReplacer(
		"\r", " ",
		"\n", " ",
		"\t", " ",
		"“", "\"",
		"”", "\"",
		"‘", "'",
		"’", "'",
		"：", ":",
	)
	return strings.TrimSpace(replacer.Replace(input))
}

func extractActionTarget(message string, keywords ...string) string {
	return cleanActionTarget(extractRawActionTarget(message, keywords...))
}

func extractRawActionTarget(message string, keywords ...string) string {
	bestIndex := -1
	candidate := ""

	for _, keyword := range keywords {
		index := strings.Index(message, keyword)
		if index < 0 {
			continue
		}
		if bestIndex >= 0 && index >= bestIndex {
			continue
		}
		bestIndex = index
		candidate = message[index+len(keyword):]
	}

	return candidate
}

func cleanActionTarget(candidate string) string {
	text := strings.TrimSpace(candidate)
	if text == "" {
		return ""
	}

	leadingTokens := []string{"到", "把", "一下", "一个", "给", "请", "帮我", "给我"}
	for {
		trimmed := text
		for _, token := range leadingTokens {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, token))
		}
		if trimmed == text {
			break
		}
		text = trimmed
	}

	stopTokens := []string{
		"，", "。", "！", "？", ",", ".", "!", "?", ";", "；", "\n",
		" 并", " 然后", " 再", " 而且", " 并且", " 并把", " 并执行", " 并帮我", " 并操作", " 并打开",
		"并聚焦", "并定位", "并选中", "并打开", "并查看", "并高亮", "并展开", "并扩展", "并适配", "并重置",
		"然后聚焦", "然后定位", "然后选中", "然后打开", "然后查看", "然后高亮", "然后展开", "然后扩展",
		"再聚焦", "再定位", "再选中", "再打开", "再查看", "再高亮", "再展开", "再扩展",
	}
	cutIndex := len(text)
	for _, token := range stopTokens {
		if index := strings.Index(text, token); index >= 0 && index < cutIndex {
			cutIndex = index
		}
	}
	text = strings.TrimSpace(text[:cutIndex])

	trailingTokens := []string{
		"的关联节点",
		"关联节点",
		"相关节点",
		"节点",
		"的前置知识链",
		"前置知识链",
		"的前置链",
		"前置链",
		"的先修链",
		"先修链",
		"的详情页",
		"详情页",
		"的详情",
		"详情",
		"知识图谱",
		"图谱",
		"图里",
	}
	for _, token := range trailingTokens {
		text = strings.TrimSpace(strings.TrimSuffix(text, token))
	}

	text = strings.Trim(text, "\"'“”‘’()（）[]【】")
	return strings.TrimSpace(text)
}

func cleanCourseTitle(candidate string) string {
	raw := strings.TrimSpace(strings.Trim(candidate, "\"'“”‘’()（）[]【】"))
	if normalized := coursemeta.NormalizeCourseTitleWithFallback(raw, ""); normalized != "" {
		return normalized
	}
	text := cleanActionTarget(candidate)
	if normalized := coursemeta.NormalizeCourseTitleWithFallback(text, ""); normalized != "" {
		return normalized
	}
	text = strings.TrimSpace(strings.TrimSuffix(text, "课程知识图谱"))
	text = strings.TrimSpace(strings.TrimSuffix(text, "课程图谱"))
	text = strings.TrimSpace(strings.TrimSuffix(text, "知识图谱"))
	text = strings.TrimSpace(strings.TrimSuffix(text, "图谱"))
	return coursemeta.NormalizeCourseTitleWithFallback(text, "")
}

func inferPrerequisiteChainTarget(message string) string {
	if !containsAny(message, "前置链", "前置知识链", "先修链", "前置路径", "前置关系", "先修关系") {
		return ""
	}

	target := extractActionTarget(message, "高亮", "看一下", "看下", "看看", "看一眼", "查看", "帮我看", "带我看")
	if target == "" {
		target = extractActionTarget(message, "聚焦", "定位")
	}
	return cleanActionTarget(target)
}

func inferCourseFilterTarget(message string) string {
	if courseTitle := cleanCourseTitle(extractRawActionTarget(message, "按课程筛选", "筛选课程", "课程筛选", "筛选")); courseTitle != "" {
		return courseTitle
	}

	if !containsAny(message, "图谱", "课程", "课程视图") {
		return ""
	}

	return cleanCourseTitle(extractRawActionTarget(
		message,
		"图谱切换到",
		"图谱切到",
		"切换到课程",
		"切到课程",
		"课程切换到",
		"课程切到",
		"切换到",
		"切到",
	))
}

func containsAny(input string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func resolveCompoundActionTarget(target string, fallbacks ...string) string {
	if !isImplicitViewportTarget(target) {
		return target
	}

	for _, fallback := range fallbacks {
		if !isImplicitViewportTarget(fallback) {
			return strings.TrimSpace(fallback)
		}
	}
	return ""
}

func firstResolvedTarget(values ...string) string {
	for _, value := range values {
		if isImplicitViewportTarget(value) {
			continue
		}
		return strings.TrimSpace(value)
	}
	return ""
}

func isImplicitViewportTarget(target string) bool {
	normalized := strings.TrimSpace(target)
	switch normalized {
	case "", "图里", "图上", "页面", "详情页", "当前图谱", "当前页面", "这里", "那里", "它":
		return true
	}
	return false
}

func buildActionFallbackReply(actions []assistantdto.ClientAction, modelContent string) string {
	if len(actions) == 0 {
		return strings.TrimSpace(modelContent)
	}

	lines := make([]string, 0, len(actions)+1)
	for _, action := range actions {
		switch action.Type {
		case "propose_goal_profile":
			lines = append(lines, "我已经根据你的描述整理出一份学习目标画像，请确认后生成学习计划。")
		case "focus_knowledge":
			lines = append(lines, "已为你尝试聚焦到“"+action.Target+"”。")
		case "expand_knowledge":
			lines = append(lines, "已为你尝试展开“"+action.Target+"”的关联节点。")
		case "highlight_prerequisite_chain":
			lines = append(lines, "已为你尝试高亮“"+action.Target+"”的前置链。")
		case "open_knowledge_detail":
			lines = append(lines, "已为你尝试打开“"+action.Target+"”的详情页。")
		case "filter_course":
			lines = append(lines, "已为你尝试按课程筛选“"+action.CourseTitle+"”。")
		case "fit_graph_view":
			lines = append(lines, "已为你尝试适配当前图谱视图。")
		case "reset_graph_scene":
			lines = append(lines, "已为你尝试重置当前图谱场景。")
		}
	}

	content := strings.TrimSpace(modelContent)
	if content == "" || shouldSuppressModelContentForAction(content) {
		return strings.Join(lines, "\n")
	}

	lines = append(lines, content)
	return strings.Join(lines, "\n\n")
}

func shouldSuppressModelContentForAction(content string) bool {
	return containsAny(content,
		"未登录",
		"当前可用功能",
		"我可以帮您",
		"我可以帮你",
		"告诉我您",
		"告诉我你",
	)
}
