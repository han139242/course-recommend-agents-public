package service

import (
	"strings"

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

	if target := extractActionTarget(normalized, "高亮"); target != "" && containsAny(normalized, "前置链", "前置知识链", "先修链", "前置路径") {
		primaryTarget = firstResolvedTarget(primaryTarget, target)
		actions = append(actions, assistantdto.ClientAction{
			Type:   "highlight_prerequisite_chain",
			Target: target,
			Flash:  true,
		})
	}

	if containsAny(normalized, "打开", "查看") {
		if target := resolveCompoundActionTarget(extractActionTarget(normalized, "打开", "查看"), primaryTarget); target != "" && containsAny(normalized, "详情页", "详情") {
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

	if containsAny(normalized, "聚焦", "定位", "选中") {
		if target := resolveCompoundActionTarget(extractActionTarget(normalized, "聚焦", "定位", "选中"), primaryTarget); target != "" {
			actions = append(actions, assistantdto.ClientAction{
				Type:   "focus_knowledge",
				Target: target,
				Flash:  true,
			})
		}
	}

	if containsAny(normalized, "按课程筛选", "筛选课程", "课程筛选") {
		if courseTitle := cleanCourseTitle(extractActionTarget(normalized, "按课程筛选", "筛选课程", "课程筛选", "筛选")); courseTitle != "" {
			actions = append(actions, assistantdto.ClientAction{
				Type:        "filter_course",
				CourseTitle: courseTitle,
			})
		}
	}

	if containsAny(normalized, "适配视图", "适配当前图谱", "适配图谱", "fit view", "完整视图", "缩放到全图", "居中全图") {
		actions = append(actions, assistantdto.ClientAction{Type: "fit_graph_view"})
	}

	if containsAny(normalized, "重置场景", "重置图谱", "恢复全局", "恢复默认视图", "清空高亮", "重置页面") {
		actions = append(actions, assistantdto.ClientAction{Type: "reset_graph_scene"})
	}

	return dedupeClientActions(actions)
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

	return cleanActionTarget(candidate)
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
	text := cleanActionTarget(candidate)
	text = strings.TrimSpace(strings.TrimSuffix(text, "课程知识图谱"))
	text = strings.TrimSpace(strings.TrimSuffix(text, "课程图谱"))
	text = strings.TrimSpace(strings.TrimSuffix(text, "知识图谱"))
	text = strings.TrimSpace(strings.TrimSuffix(text, "图谱"))
	return strings.TrimSpace(text)
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
