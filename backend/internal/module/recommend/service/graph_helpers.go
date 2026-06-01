package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/han139242/course-recommend-agents/backend/internal/coursemeta"
	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
)

func edgeTouchesRoot(edge graphdto.GraphEdgeView, rootNode graphdto.GraphNodeView, knowledgeID string) bool {
	rootKeys := []string{
		strings.TrimSpace(knowledgeID),
		strings.TrimSpace(rootNode.ID),
		strings.TrimSpace(rootNode.RouteKnowledgeID),
		strings.TrimSpace(rootNode.Name),
		strings.TrimSpace(rootNode.CanonicalName),
	}
	for _, key := range rootKeys {
		if key == "" {
			continue
		}
		if sameKnowledgeID(edge.SourceID, key) || sameKnowledgeID(edge.TargetID, key) || sameKnowledgeID(edge.SourceName, key) || sameKnowledgeID(edge.TargetName, key) {
			return true
		}
	}
	return false
}

func sameKnowledge(node graphdto.GraphNodeView, knowledgeID string) bool {
	keys := []string{node.ID, node.RouteKnowledgeID, node.Name, node.CanonicalName}
	for _, key := range keys {
		if sameKnowledgeID(key, knowledgeID) {
			return true
		}
	}
	return false
}

func sameKnowledgeID(left string, right string) bool {
	return normalizeStateKey(left) != "" && normalizeStateKey(left) == normalizeStateKey(right)
}

func lookupStateForNode(lookup map[string]StudentKnowledgeState, node graphdto.GraphNodeView) (StudentKnowledgeState, bool) {
	keys := []string{node.RouteKnowledgeID, node.ID, node.Name, node.CanonicalName}
	for _, key := range keys {
		if item, ok := lookup[normalizeStateKey(key)]; ok {
			return item, true
		}
	}
	return StudentKnowledgeState{}, false
}

func buildScoredCandidate(rootTitle string, rootState StudentKnowledgeState, edge graphdto.GraphEdgeView, node graphdto.GraphNodeView, lookup map[string]StudentKnowledgeState) scoredCandidate {
	state, hasState := lookupStateForNode(lookup, node)
	relationBoost, reasonCode, reasonLabel := relationScore(rootState, edge, node)
	stateBoost := masteryNeedScore(state, hasState)
	score := normalizedRecommendationScore(relationBoost + stateBoost)

	title := displayNodeTitle(node)

	knowledgeID := strings.TrimSpace(node.RouteKnowledgeID)
	if knowledgeID == "" {
		knowledgeID = strings.TrimSpace(node.ID)
	}

	return scoredCandidate{
		Node:          node,
		KnowledgeID:   knowledgeID,
		Title:         title,
		ReasonCode:    reasonCode,
		ReasonLabel:   reasonLabel,
		TargetCourse:  graphNodeCourseTitle(node, coursemeta.DefaultCourseTitle),
		ReasonDetail:  contextualDetail(rootTitle, reasonLabel, state, hasState, title),
		MasteryScore:  state.MasteryScore,
		CorrectRate:   state.CorrectRate,
		AnsweredCount: state.AnsweredCount,
		Score:         score,
	}
}

func buildFallbackCandidate(rootTitle string, node graphdto.GraphNodeView, lookup map[string]StudentKnowledgeState) scoredCandidate {
	state, hasState := lookupStateForNode(lookup, node)
	title := displayNodeTitle(node)
	knowledgeID := strings.TrimSpace(node.RouteKnowledgeID)
	if knowledgeID == "" {
		knowledgeID = strings.TrimSpace(node.ID)
	}
	return scoredCandidate{
		Node:          node,
		KnowledgeID:   knowledgeID,
		Title:         title,
		ReasonCode:    "knowledge_graph",
		ReasonLabel:   "图谱关联",
		TargetCourse:  graphNodeCourseTitle(node, coursemeta.DefaultCourseTitle),
		ReasonDetail:  contextualDetail(rootTitle, "图谱关联", state, hasState, title),
		MasteryScore:  state.MasteryScore,
		CorrectRate:   state.CorrectRate,
		AnsweredCount: state.AnsweredCount,
		Score:         normalizedRecommendationScore(18 + masteryNeedScore(state, hasState)),
	}
}

func relationScore(rootState StudentKnowledgeState, edge graphdto.GraphEdgeView, node graphdto.GraphNodeView) (float64, string, string) {
	relationType := strings.TrimSpace(edge.RelationType)
	rootMastery := rootState.MasteryScore
	switch relationType {
	case "PREREQUISITE_OF":
		if sameKnowledgeID(edge.TargetID, node.RouteKnowledgeID) || sameKnowledgeID(edge.TargetID, node.ID) {
			if rootMastery >= 60 {
				return 28, "next_step", "后续衔接"
			}
			return 10, "next_step", "后续衔接"
		}
		if rootMastery < 60 {
			return 52, "prerequisite", "先修补齐"
		}
		return 30, "prerequisite", "先修补齐"
	case "RELATED_TO":
		return 24, "related", "关联补强"
	case "SUBTOPIC_OF":
		if rootMastery >= 60 {
			return 20, "extension", "主题扩展"
		}
		return 14, "extension", "主题扩展"
	default:
		return 16, "knowledge_graph", "图谱关联"
	}
}

func masteryNeedScore(state StudentKnowledgeState, hasState bool) float64 {
	if !hasState {
		return 22
	}
	score := 0.0
	switch {
	case state.MasteryScore < 20:
		score += 40
	case state.MasteryScore < 45:
		score += 28
	case state.MasteryScore < 60:
		score += 18
	case state.MasteryScore < 80:
		score += 8
	default:
		score -= 22
	}
	if state.AnsweredCount > 0 {
		if state.CorrectRate < 0.5 {
			score += 16
		} else if state.CorrectRate < 0.7 {
			score += 9
		}
	} else {
		score += 6
	}
	return score
}

func contextualDetail(rootTitle string, reasonLabel string, state StudentKnowledgeState, hasState bool, title string) string {
	if !hasState {
		return fmt.Sprintf("%s 与当前知识点「%s」关系紧密，系统里还没有稳定学习记录，适合尽快纳入下一步。", title, rootTitle)
	}
	if strings.Contains(reasonLabel, "先修") {
		return fmt.Sprintf("%s 是「%s」的关键基础，你当前掌握度 %.0f%%，建议先补齐再继续推进。", title, rootTitle, state.MasteryScore)
	}
	if strings.Contains(reasonLabel, "后续") {
		return fmt.Sprintf("%s 可以作为「%s」之后的自然衔接点，你当前掌握度 %.0f%%。", title, rootTitle, state.MasteryScore)
	}
	return fmt.Sprintf("%s 与「%s」关联较强，你当前掌握度 %.0f%%，适合作为当前阶段的补强或扩展内容。", title, rootTitle, state.MasteryScore)
}

func normalizedRecommendationScore(raw float64) float64 {
	score := 0.45 + raw/100.0
	if score > 0.98 {
		return 0.98
	}
	if score < 0.35 {
		return 0.35
	}
	return score
}

func prioritizeRecommendNodes(root string, nodes []graphdto.GraphNodeView) []graphdto.GraphNodeView {
	prioritized := append([]graphdto.GraphNodeView(nil), nodes...)
	normalizedRoot := strings.TrimSpace(root)
	sort.SliceStable(prioritized, func(i, j int) bool {
		left := recommendationNodeRank(normalizedRoot, prioritized[i])
		right := recommendationNodeRank(normalizedRoot, prioritized[j])
		if left != right {
			return left < right
		}
		return strings.TrimSpace(prioritized[i].Name) < strings.TrimSpace(prioritized[j].Name)
	})
	return prioritized
}

func recommendationNodeRank(root string, node graphdto.GraphNodeView) int {
	if root != "" {
		if strings.TrimSpace(node.Name) == root || strings.TrimSpace(node.CanonicalName) == root || strings.TrimSpace(node.RouteKnowledgeID) == root || strings.TrimSpace(node.ID) == root {
			return 0
		}
	}

	switch strings.TrimSpace(node.EntityType) {
	case "algorithm":
		return 1
	case "concept":
		return 2
	case "method":
		return 3
	case "application":
		return 4
	case "topic":
		return 5
	default:
		return 6
	}
}

func stageFromScene(scene string) string {
	if scene == "after_wrong_answer" {
		return "错题补强"
	}
	if scene == "detail" {
		return "详情页推荐"
	}
	if scene == "cross_course_transfer" {
		return "跨课程迁移"
	}
	return "补弱推荐"
}

func detailForNode(nodeName string, state StudentKnowledgeState, hasState bool) string {
	if strings.TrimSpace(nodeName) == "" {
		return "建议先从这个知识点开始，逐步衔接到后续内容。"
	}
	if !hasState {
		return fmt.Sprintf("建议优先理解 %s，当前系统里还没有你的稳定学习记录。", nodeName)
	}
	return fmt.Sprintf("%s 当前掌握度 %.0f%%，适合在这一阶段继续补强。", nodeName, state.MasteryScore)
}

func scoreForIndex(index int) float64 {
	base := 0.95 - float64(index)*0.08
	if base < 0.5 {
		return 0.5
	}
	return base
}

func transferRelationWeight(relationType string) float64 {
	switch strings.TrimSpace(relationType) {
	case "PREREQUISITE_OF":
		return 1.05
	case "SUBTOPIC_OF":
		return 0.92
	case "RELATED_TO":
		return 1.15
	case "USES", "APPLIES_TO":
		return 1.08
	default:
		return 0.88
	}
}

func buildNodeLookup(nodes []graphdto.GraphNodeView) map[string]graphdto.GraphNodeView {
	lookup := make(map[string]graphdto.GraphNodeView, len(nodes)*4)
	for _, node := range nodes {
		for _, key := range []string{node.RouteKnowledgeID, node.ID, node.Name, node.CanonicalName} {
			normalized := normalizeStateKey(key)
			if normalized == "" {
				continue
			}
			lookup[normalized] = node
		}
	}
	return lookup
}

func findGraphNode(lookup map[string]graphdto.GraphNodeView, keys ...string) (graphdto.GraphNodeView, bool) {
	for _, key := range keys {
		if node, ok := lookup[normalizeStateKey(key)]; ok {
			return node, true
		}
	}
	return graphdto.GraphNodeView{}, false
}

func orderedGraphNodes(orderedNodeIDs []string, lookup map[string]graphdto.GraphNodeView) []graphdto.GraphNodeView {
	result := make([]graphdto.GraphNodeView, 0, len(orderedNodeIDs))
	for _, orderedNodeID := range orderedNodeIDs {
		if node, ok := lookup[normalizeStateKey(orderedNodeID)]; ok {
			result = append(result, node)
		}
	}
	return result
}

func displayNodeTitle(node graphdto.GraphNodeView) string {
	name := strings.TrimSpace(node.Name)
	canonicalName := strings.TrimSpace(node.CanonicalName)
	routeID := strings.TrimSpace(node.RouteKnowledgeID)

	if canonicalName != "" && (looksLikeMachineKnowledgeID(name) || sameKnowledgeID(name, routeID)) {
		return canonicalName
	}

	return firstNonEmptyText(name, canonicalName, routeID, strings.TrimSpace(node.ID))
}

func looksLikeMachineKnowledgeID(value string) bool {
	normalized := strings.TrimSpace(value)
	if len(normalized) < 16 {
		return false
	}

	for _, char := range normalized {
		if (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F') {
			continue
		}
		return false
	}

	return true
}

func graphNodeCourseTitle(node graphdto.GraphNodeView, fallback string) string {
	return coursemeta.ResolveNodeCourseTitleWithFallback(
		node.CuratedSource,
		node.SourcePath,
		"",
		"",
		"",
		node.ChapterTitle,
		fallback,
	)
}
