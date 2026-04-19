package service

import (
	"fmt"
	"strings"

	"github.com/han139242/course-recommend-agents/backend/internal/coursemeta"
	assistantdto "github.com/han139242/course-recommend-agents/backend/internal/module/assistant/dto"
)

type localChatContext struct {
	KnowledgeName  string
	CourseTitle    string
	Summary        string
	Prerequisites  []string
	MasterySummary string
}

func (s *Service) buildLocalFallbackChatResponse(messages []map[string]any, actions []assistantdto.ClientAction, profile assistantAgentProfile) *assistantdto.ChatResponse {
	dedupedActions := dedupeClientActions(actions)
	content := strings.TrimSpace(buildLocalFallbackContent(messages, dedupedActions))
	if content == "" {
		content = "在线 AI 当前暂不可用，我先保留图谱控制能力。你可以继续让我聚焦、展开、高亮前置链，或者打开详情页。"
	}
	if len(dedupedActions) > 0 {
		content = buildActionFallbackReply(dedupedActions, content)
	}

	return &assistantdto.ChatResponse{
		Content:    content,
		Actions:    dedupedActions,
		Agent:      string(profile.Intent),
		AgentLabel: profile.Label,
	}
}

func buildLocalFallbackContent(messages []map[string]any, actions []assistantdto.ClientAction) string {
	question := strings.TrimSpace(latestUserMessage(messages))
	context := extractLocalChatContext(messages)
	knowledgeName := firstNonEmptyString(
		context.KnowledgeName,
		firstActionTarget(actions),
		inferKnowledgeNameFromQuestion(question),
	)

	parts := []string{
		"在线 AI 当前连接不稳定，我先基于当前页面上下文给你本地回答。",
	}

	switch {
	case knowledgeName != "" && context.Summary != "":
		parts = append(parts, fmt.Sprintf("当前聚焦“%s”：%s", knowledgeName, context.Summary))
	case knowledgeName != "":
		parts = append(parts, fmt.Sprintf("当前聚焦“%s”。如果你愿意，我可以继续围绕它做展开、聚焦、高亮前置链或打开详情。", knowledgeName))
	default:
		parts = append(parts, "现在是全局图谱视角。你不需要先点击节点，也可以直接问我学习路线、课程关系、先学什么，或者让我按名字帮你定位某个知识点。")
	}

	if context.CourseTitle != "" {
		parts = append(parts, fmt.Sprintf("它当前位于“%s”这条课程线里。", context.CourseTitle))
	}

	if hasAnyKeyword(question, "前置", "先学", "基础") && len(context.Prerequisites) > 0 && knowledgeName != "" {
		parts = append(parts, fmt.Sprintf("建议先补这些前置知识：%s，再回来看“%s”的位置和作用。", strings.Join(context.Prerequisites, "、"), knowledgeName))
	} else if hasAnyKeyword(question, "下一步", "后续", "怎么学", "路线", "路径") && knowledgeName != "" {
		if len(context.Prerequisites) > 0 {
			parts = append(parts, fmt.Sprintf("下一步建议：先梳理 %s，再展开“%s”的一跳邻居，看它会通向哪些主题。", strings.Join(context.Prerequisites, "、"), knowledgeName))
		} else {
			parts = append(parts, fmt.Sprintf("下一步建议：先看“%s”的定义和典型应用，再展开它的邻居节点补齐上下文。", knowledgeName))
		}
	} else if hasAnyKeyword(question, "下一步", "后续", "怎么学", "路线", "路径") {
		parts = append(parts, "下一步建议：先从你现在最熟悉的一门课程切入，展开一个核心节点的一跳邻居，再沿前置链和后续去向逐步收窄范围，这样会比直接盯全图更清晰。")
	} else if hasAnyKeyword(question, "概览", "总览", "图谱", "结构") {
		parts = append(parts, "全局使用方式：先看课程簇和桥接关系，再挑一个核心节点展开，这样最容易把跨课程联系和学习顺序看明白。")
	} else if hasAnyKeyword(question, "对比", "区别") && knowledgeName != "" {
		parts = append(parts, fmt.Sprintf("更适合的比较方式是：把“%s”和它的前置知识、相关知识放在一起，对比定义、作用和图谱位置。", knowledgeName))
	} else if len(actions) == 0 && knowledgeName != "" {
		if len(context.Prerequisites) > 0 {
			parts = append(parts, fmt.Sprintf("你可以接着让我高亮“%s”的前置链，或者直接展开它的关联节点。", knowledgeName))
		} else {
			parts = append(parts, fmt.Sprintf("你可以接着让我展开“%s”的关联节点，或者打开它的详情页继续看。", knowledgeName))
		}
	} else if len(actions) == 0 {
		parts = append(parts, "你现在可以直接问我“推荐一条学习路线”“帮我概览当前图谱”，或者直接说出一个知识点名称让我帮你定位。")
	}

	return strings.Join(parts, "\n\n")
}

func extractLocalChatContext(messages []map[string]any) localChatContext {
	var context localChatContext
	for index := len(messages) - 1; index >= 0; index-- {
		content := strings.TrimSpace(extractAssistantContent(messages[index]["content"]))
		if content == "" {
			continue
		}
		parseLocalChatContextBlock(&context, content)
	}
	return context
}

func parseLocalChatContextBlock(context *localChatContext, content string) {
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "当前知识点："):
			context.KnowledgeName = firstNonEmptyString(context.KnowledgeName, strings.TrimSpace(strings.TrimPrefix(line, "当前知识点：")))
		case strings.HasPrefix(line, "课程："):
			context.CourseTitle = firstNonEmptyString(
				context.CourseTitle,
				coursemeta.NormalizeCourseTitleWithFallback(strings.TrimSpace(strings.TrimPrefix(line, "课程：")), ""),
			)
		case strings.HasPrefix(line, "简介："):
			context.Summary = firstNonEmptyString(context.Summary, strings.TrimSpace(strings.TrimPrefix(line, "简介：")))
		case strings.HasPrefix(line, "前置知识："):
			if len(context.Prerequisites) == 0 {
				context.Prerequisites = splitKnowledgeList(strings.TrimSpace(strings.TrimPrefix(line, "前置知识：")))
			}
		case strings.HasPrefix(line, "掌握度："):
			context.MasterySummary = firstNonEmptyString(context.MasterySummary, strings.TrimSpace(strings.TrimPrefix(line, "掌握度：")))
		case strings.Contains(line, "我们现在聚焦在“"):
			context.KnowledgeName = firstNonEmptyString(context.KnowledgeName, extractQuotedText(line))
		}
	}
}

func splitKnowledgeList(input string) []string {
	normalized := strings.NewReplacer(",", "、", "，", "、", ";", "、", "；", "、").Replace(strings.TrimSpace(input))
	if normalized == "" || normalized == "暂无" {
		return nil
	}

	parts := strings.Split(normalized, "、")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" || item == "暂无" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func inferKnowledgeNameFromQuestion(question string) string {
	normalized := normalizeActionText(question)
	prefixes := []string{
		"帮我解释", "解释", "讲解", "介绍", "帮我理解", "理解",
		"帮我对比", "对比", "帮我说明", "说明", "告诉我",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(normalized, prefix) {
			name := cleanActionTarget(strings.TrimSpace(strings.TrimPrefix(normalized, prefix)))
			if !isImplicitViewportTarget(name) {
				return name
			}
		}
	}
	return extractQuotedText(normalized)
}

func firstActionTarget(actions []assistantdto.ClientAction) string {
	for _, action := range actions {
		if !isImplicitViewportTarget(action.Target) {
			return strings.TrimSpace(action.Target)
		}
	}
	return ""
}

func extractQuotedText(input string) string {
	pairs := [][2]string{
		{"“", "”"},
		{"\"", "\""},
		{"'", "'"},
	}
	for _, pair := range pairs {
		start := strings.Index(input, pair[0])
		if start < 0 {
			continue
		}
		end := strings.Index(input[start+len(pair[0]):], pair[1])
		if end < 0 {
			continue
		}
		value := strings.TrimSpace(input[start+len(pair[0]) : start+len(pair[0])+end])
		if value != "" {
			return value
		}
	}
	return ""
}

func hasAnyKeyword(input string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func mergeClientActions(groups ...[]assistantdto.ClientAction) []assistantdto.ClientAction {
	merged := make([]assistantdto.ClientAction, 0, 8)
	for _, group := range groups {
		merged = append(merged, group...)
	}
	return dedupeClientActions(merged)
}
