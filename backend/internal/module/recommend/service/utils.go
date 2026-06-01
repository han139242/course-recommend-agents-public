package service

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/coursemeta"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
)

var defaultRecommendationRoots = []string{
	"梯度下降",
	"偏导数",
	"导数",
	"矩阵",
	"最优化问题",
	"卷积神经网络",
	"检索增强生成",
	"知识图谱",
	"",
}

func firstNonEmptyText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmptyCourse(values ...string) string {
	for _, value := range values {
		if normalized := coursemeta.NormalizeCourseTitleWithFallback(value, ""); normalized != "" {
			return normalized
		}
	}
	return ""
}

func forgettingFactor(lastAnsweredAt string) float64 {
	days := daysSinceLastAnswer(lastAnsweredAt)
	if days <= 0 {
		return 1
	}
	return math.Exp(-days / 7)
}

func effectiveMastery(mastery float64, lastAnsweredAt string) float64 {
	return mastery * forgettingFactor(lastAnsweredAt)
}

func daysSinceLastAnswer(lastAnsweredAt string) float64 {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(lastAnsweredAt))
	if err != nil {
		return 0
	}
	days := time.Since(parsed).Hours() / 24
	if days < 0 {
		return 0
	}
	return days
}

func limitValue(limit int) int {
	if limit <= 0 {
		return 5
	}
	return limit
}

func normalizeRecommendCacheTarget(target string, knowledgeID string) string {
	target = strings.TrimSpace(target)
	knowledgeID = strings.TrimSpace(knowledgeID)
	if target == "" && knowledgeID == "" {
		return "__auto_v5__"
	}
	return fmt.Sprintf("v5::%s::%s", knowledgeID, target)
}

func recommendRoots(target string) []string {
	roots := make([]string, 0, len(defaultRecommendationRoots)+1)
	seen := map[string]struct{}{}
	appendRoot := func(root string) {
		normalized := strings.TrimSpace(root)
		if _, exists := seen[normalized]; exists {
			return
		}
		seen[normalized] = struct{}{}
		roots = append(roots, normalized)
	}

	if strings.TrimSpace(target) != "" {
		appendRoot(target)
	}
	for _, root := range defaultRecommendationRoots {
		appendRoot(root)
	}
	return roots
}

func fallbackRecommendPath(scene string, limit int) []recommenddto.PathItemView {
	defaults := []struct {
		title  string
		route  string
		detail string
	}{
		{title: "梯度下降", route: "/knowledge/gradient-descent", detail: "先补齐优化基础和梯度概念，后续学习会更顺。"},
		{title: "矩阵运算与线性代数", route: "/knowledge/matrix-ops", detail: "矩阵与线性代数是神经网络计算的底层基础。"},
		{title: "卷积神经网络 CNN", route: "/knowledge/cnn", detail: "适合把数学基础和深度学习实践串成一条主线。"},
	}

	path := make([]recommenddto.PathItemView, 0, min(limit, len(defaults)))
	for idx, item := range defaults {
		if len(path) >= limit {
			break
		}
		path = append(path, recommenddto.PathItemView{
			Stage:       stageFromScene(scene),
			Title:       item.title,
			Detail:      item.detail,
			Route:       item.route,
			KnowledgeID: strings.TrimPrefix(item.route, "/knowledge/"),
			ReasonCode:  "recommended_start",
			ReasonLabel: "推荐起点",
			Score:       scoreForIndex(idx),
		})
	}
	return path
}

func mergePathItems(left []recommenddto.PathItemView, right []recommenddto.PathItemView, limit int) []recommenddto.PathItemView {
	result := make([]recommenddto.PathItemView, 0, limit)
	seen := make(map[string]struct{}, limit)
	appendItem := func(item recommenddto.PathItemView) {
		key := strings.TrimSpace(item.Route)
		if key == "" {
			key = strings.TrimSpace(item.KnowledgeID)
		}
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}

	for _, item := range left {
		appendItem(item)
		if len(result) >= limit {
			return result
		}
	}
	for _, item := range right {
		appendItem(item)
		if len(result) >= limit {
			return result
		}
	}
	return result
}

func isUsableRecommendPath(response *recommenddto.PathResponse) bool {
	if response == nil || len(response.Path) == 0 {
		return false
	}
	for _, item := range response.Path {
		if strings.TrimSpace(item.Title) == "" || strings.Contains(item.Route, "default-target") {
			return false
		}
	}
	return true
}
