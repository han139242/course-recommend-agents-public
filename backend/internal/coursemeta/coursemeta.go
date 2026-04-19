package coursemeta

import (
	"regexp"
	"strings"
)

const (
	DefaultCourseTitle = "其他"
	AllCourseTitle     = "全部"
)

var sourcePattern = regexp.MustCompile(`(?i)(course_[a-z_]+)`)

var courseSourceTitleMap = map[string]string{
	"course_ai_engineering":   "AI工程",
	"course_ai_ethics":        "AI伦理",
	"course_ai_intro":         "人工智能导论",
	"course_calculus":         "高等数学",
	"course_cv":               "计算机视觉",
	"course_data_structure":   "数据结构",
	"course_deep_learning":    "深度学习",
	"course_knowledge_graph":  "知识图谱",
	"course_linear_algebra":   "线性代数",
	"course_llm":              "大模型",
	"course_machine_learning": "机器学习",
	"course_nlp":              "自然语言处理",
	"course_probability":      "概率与统计",
	"course_python":           "Python编程",
	"course_rl":               "强化学习",
}

var courseTitleSet = func() map[string]struct{} {
	result := make(map[string]struct{}, len(courseSourceTitleMap)+2)
	for _, title := range courseSourceTitleMap {
		result[title] = struct{}{}
	}
	result[DefaultCourseTitle] = struct{}{}
	result[AllCourseTitle] = struct{}{}
	return result
}()

var legacyCourseTitleAliases = map[string]string{
	"人工智能基础知识骨架":    "人工智能导论",
	"人工智能课程知识图谱":    "知识图谱",
	"人工智能在游戏设计中的应用": "人工智能导论",
	"智能体与多智能体系统":    "人工智能导论",
	"智能计算及其应用":      "人工智能导论",
	"绪论":            "人工智能导论",
	"人工神经网络及其应用":    "深度学习",
	"专家系统与机器学习":     "人工智能导论",
	"搜索求解策略":        "人工智能导论",
	"知识表示":          "人工智能导论",
	"不确定性推理方法":      "人工智能导论",
	"确定性推理方法":       "人工智能导论",
	"自然语言处理及其应用":    "自然语言处理",
	"线性代数基础":        "线性代数",
	"概率统计":          "概率与统计",
	"Python程序设计":    "Python编程",
}

var courseTitleKeywordRules = []struct {
	pattern *regexp.Regexp
	title   string
}{
	{pattern: regexp.MustCompile(`(?i)自然语言处理|nlp`), title: "自然语言处理"},
	{pattern: regexp.MustCompile(`(?i)知识图谱|knowledge[_\s-]*graph`), title: "知识图谱"},
	{pattern: regexp.MustCompile(`(?i)强化学习|reinforcement[_\s-]*learning|\brl\b`), title: "强化学习"},
	{pattern: regexp.MustCompile(`(?i)计算机视觉|computer[_\s-]*vision|\bcv\b|image|video`), title: "计算机视觉"},
	{pattern: regexp.MustCompile(`(?i)深度学习|deep[_\s-]*learning|neural[_\s-]*network`), title: "深度学习"},
	{pattern: regexp.MustCompile(`(?i)ai[_\s-]*engineering|mlops|serving|deployment|compression|edge[_\s-]*ai|inference`), title: "AI工程"},
	{pattern: regexp.MustCompile(`(?i)机器学习|machine[_\s-]*learning|automl`), title: "机器学习"},
	{pattern: regexp.MustCompile(`(?i)线性代数|linear[_\s-]*algebra`), title: "线性代数"},
	{pattern: regexp.MustCompile(`(?i)概率.*统计|统计.*概率|概率统计|probability|statistics`), title: "概率与统计"},
	{pattern: regexp.MustCompile(`(?i)高等数学|微积分|数学分析|calculus`), title: "高等数学"},
	{pattern: regexp.MustCompile(`(?i)python`), title: "Python编程"},
	{pattern: regexp.MustCompile(`(?i)数据结构|data[_\s-]*structure`), title: "数据结构"},
	{pattern: regexp.MustCompile(`(?i)ai[_\s-]*ethics|可信ai|负责任ai|算法治理|ethics|fairness|privacy|safety|xai|governance`), title: "AI伦理"},
	{pattern: regexp.MustCompile(`(?i)大模型|\bllm\b|transformer|rag|prompt|language[_\s-]*model`), title: "大模型"},
	{
		pattern: regexp.MustCompile(`(?i)人工智能导论|人工智能基础|知识表示|ontology|reasoning|search|planning|agent|multi[_\s-]*agent|expert[_\s-]*system|game[_\s-]*ai|intelligent[_\s-]*computing|artificial[_\s-]*intelligence`),
		title:   "人工智能导论",
	},
}

func normalizeText(value string) string {
	return strings.TrimSpace(value)
}

func normalizeCourseSource(value string) string {
	normalized := normalizeText(value)
	if normalized == "" {
		return ""
	}
	if matches := sourcePattern.FindStringSubmatch(normalized); len(matches) > 1 {
		return strings.ToLower(matches[1])
	}
	return strings.ToLower(normalized)
}

func matchKeywordRule(value string) string {
	normalized := normalizeText(value)
	if normalized == "" {
		return ""
	}
	for _, rule := range courseTitleKeywordRules {
		if rule.pattern.MatchString(normalized) {
			return rule.title
		}
	}
	return ""
}

// GetCourseTitleFromSource resolves a course title from a stable source key.
func GetCourseTitleFromSource(value string) string {
	return courseSourceTitleMap[normalizeCourseSource(value)]
}

func resolveCourseHint(value string) string {
	normalized := normalizeText(value)
	if normalized == "" {
		return ""
	}
	if title := GetCourseTitleFromSource(normalized); title != "" {
		return title
	}
	if _, ok := courseTitleSet[normalized]; ok {
		return normalized
	}
	if title := legacyCourseTitleAliases[normalized]; title != "" {
		return title
	}
	return matchKeywordRule(normalized)
}

// NormalizeCourseTitleWithFallback normalizes a course label while keeping a caller-provided fallback.
func NormalizeCourseTitleWithFallback(value string, fallback string) string {
	normalized := normalizeText(value)
	if normalized == "" {
		return fallback
	}
	if hint := resolveCourseHint(normalized); hint != "" {
		return hint
	}
	return normalized
}

// NormalizeCourseTitle normalizes a course label with the default fallback.
func NormalizeCourseTitle(value string) string {
	return NormalizeCourseTitleWithFallback(value, DefaultCourseTitle)
}

// ResolveNodeCourseTitleWithFallback resolves a normalized course title from node metadata.
func ResolveNodeCourseTitleWithFallback(curatedSource, sourcePath, source, courseSource, courseTitle, chapterTitle, fallback string) string {
	for _, candidate := range []string{curatedSource, sourcePath, source, courseSource} {
		if title := resolveCourseHint(candidate); title != "" {
			return title
		}
	}
	for _, candidate := range []string{courseTitle, chapterTitle} {
		if title := resolveCourseHint(candidate); title != "" {
			return title
		}
	}
	normalized := normalizeText(courseTitle)
	if normalized == "" {
		normalized = normalizeText(chapterTitle)
	}
	if normalized != "" {
		return normalized
	}
	return fallback
}

// ResolveNodeCourseTitle resolves a normalized course title from node metadata with the default fallback.
func ResolveNodeCourseTitle(curatedSource, sourcePath, source, courseSource, courseTitle, chapterTitle string) string {
	return ResolveNodeCourseTitleWithFallback(curatedSource, sourcePath, source, courseSource, courseTitle, chapterTitle, DefaultCourseTitle)
}
