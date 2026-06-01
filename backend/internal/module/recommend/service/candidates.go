package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
)

// candidateEntry bundles a candidate knowledge point with its context for unified scoring.
type candidateEntry struct {
	state         StudentKnowledgeState
	knowledgeID   string
	title         string
	sourceCourse  string
	targetCourse  string
	relationType  string
	sourceMastery float64
	item          recommenddto.PathItemView
}

// collectCandidates gathers raw candidates from all available sources (weakness,
// review, gap-bridge, knowledge-context, cross-course transfer, graph-driven)
// for the unified KT-KGPath scorer.
func (s *Service) collectCandidates(ctx context.Context, knowledgeID string, target string, progressList []StudentKnowledgeState, stateLookup map[string]StudentKnowledgeState, limit int) []candidateEntry {
	seen := make(map[string]bool, limit)
	result := make([]candidateEntry, 0, limit)

	addItem := func(item recommenddto.PathItemView, state StudentKnowledgeState, sourceCourse, targetCourse, relationType string, sourceMastery float64) {
		kid := strings.TrimSpace(item.KnowledgeID)
		if kid == "" {
			kid = strings.TrimSpace(item.Route)
		}
		if kid == "" {
			return
		}
		key := normalizeStateKey(kid)
		if seen[key] {
			return
		}
		seen[key] = true
		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = kid
		}
		result = append(result, candidateEntry{
			state:         state,
			knowledgeID:   kid,
			title:         title,
			sourceCourse:  sourceCourse,
			targetCourse:  targetCourse,
			relationType:  relationType,
			sourceMastery: sourceMastery,
			item:          item,
		})
	}

	// 1) Weakness candidates
	for _, state := range progressList {
		kid := strings.TrimSpace(state.KnowledgeID)
		if kid == "" || seen[normalizeStateKey(kid)] {
			continue
		}
		if state.MasteryScore >= 85 && state.CorrectRate >= 0.8 {
			continue
		}
		title := strings.TrimSpace(state.CanonicalName)
		if title == "" {
			title = kid
		}
		item := recommenddto.PathItemView{
			Stage:         stageFromScene(""),
			Title:         title,
			Detail:        weaknessDetail(state),
			Route:         fmt.Sprintf("/knowledge/%s", kid),
			KnowledgeID:   kid,
			ReasonCode:    "weakness",
			ReasonLabel:   "薄弱补强",
			MasteryScore:  state.MasteryScore,
			CorrectRate:   state.CorrectRate,
			AnsweredCount: state.AnsweredCount,
		}
		addItem(item, state, "", "", "", 0)
		if len(result) >= limit {
			return result
		}
	}

	// 2) Review / forgetting candidates
	for _, state := range progressList {
		kid := strings.TrimSpace(state.KnowledgeID)
		if kid == "" || state.MasteryScore < 60 || strings.TrimSpace(state.LastAnsweredAt) == "" {
			continue
		}
		effective := effectiveMastery(state.MasteryScore, state.LastAnsweredAt)
		if effective >= 50 {
			continue
		}
		title := firstNonEmptyText(strings.TrimSpace(state.CanonicalName), kid)
		days := daysSinceLastAnswer(state.LastAnsweredAt)
		item := recommenddto.PathItemView{
			Stage:         stageFromScene(""),
			Title:         title,
			Detail:        fmt.Sprintf("你上次练到「%s」已经是 %.0f 天前了，当前有效掌握度约 %.0f%%，适合安排一次复习回温。", title, days, effective),
			Route:         fmt.Sprintf("/knowledge/%s", kid),
			KnowledgeID:   kid,
			ReasonCode:    "forgetting_review",
			ReasonLabel:   "遗忘复习",
			MasteryScore:  state.MasteryScore,
			CorrectRate:   state.CorrectRate,
			AnsweredCount: state.AnsweredCount,
		}
		addItem(item, state, "", "", "", 0)
		if len(result) >= limit {
			return result
		}
	}

	// 3) Graph-based: knowledge context + gap bridge (when knowledgeID is set)
	if s.graphService != nil && strings.TrimSpace(knowledgeID) != "" {
		s.collectGraphContextCandidates(ctx, knowledgeID, stateLookup, seen, &result, limit)
		s.collectGapBridgeCandidates(ctx, knowledgeID, stateLookup, seen, &result, limit)
	}

	// 4) Cross-course transfer candidates
	if s.graphService != nil && len(progressList) > 0 {
		s.collectCrossCourseCandidates(ctx, progressList, stateLookup, seen, &result, limit)
	}

	// 5) Graph-driven fallback
	if s.graphService != nil {
		for _, root := range recommendRoots(target) {
			if len(result) >= limit {
				break
			}
			subgraph, err := s.graphService.GetSubgraph(ctx, graphdto.SubgraphQuery{Root: root, Limit: limit})
			if err != nil || subgraph == nil || len(subgraph.Nodes) == 0 {
				continue
			}
			for _, node := range prioritizeRecommendNodes(root, subgraph.Nodes) {
				kid := strings.TrimSpace(node.RouteKnowledgeID)
				title := displayNodeTitle(node)
				if kid == "" || title == "" {
					continue
				}
				key := normalizeStateKey(kid)
				if seen[key] {
					continue
				}
				state, hasState := lookupStateForNode(stateLookup, node)
				if hasState && state.MasteryScore >= 88 && state.CorrectRate >= 0.85 && state.AnsweredCount > 0 {
					continue
				}
				seen[key] = true
				item := recommenddto.PathItemView{
					Stage:         stageFromScene(""),
					Title:         title,
					Detail:        detailForNode(title, state, hasState),
					Route:         fmt.Sprintf("/knowledge/%s", kid),
					KnowledgeID:   kid,
					ReasonCode:    "knowledge_graph",
					ReasonLabel:   "图谱关联",
					TargetCourse:  graphNodeCourseTitle(node, ""),
					MasteryScore:  state.MasteryScore,
					CorrectRate:   state.CorrectRate,
					AnsweredCount: state.AnsweredCount,
				}
				result = append(result, candidateEntry{
					state:         state,
					knowledgeID:   kid,
					title:         title,
					sourceCourse:  "",
					targetCourse:  item.TargetCourse,
					relationType:  "RELATED_TO",
					sourceMastery: 0,
					item:          item,
				})
				if len(result) >= limit {
					break
				}
			}
		}
	}

	return result
}

func (s *Service) collectGraphContextCandidates(ctx context.Context, knowledgeID string, stateLookup map[string]StudentKnowledgeState, seen map[string]bool, result *[]candidateEntry, limit int) {
	subgraph, err := s.graphService.GetSubgraph(ctx, graphdto.SubgraphQuery{Root: knowledgeID, Limit: max(limit*2, 12)})
	if err != nil || subgraph == nil || len(subgraph.Nodes) == 0 {
		return
	}

	nodeByID := make(map[string]graphdto.GraphNodeView, len(subgraph.Nodes))
	rootNode := graphdto.GraphNodeView{}
	for _, node := range subgraph.Nodes {
		nodeByID[node.ID] = node
		if sameKnowledge(node, knowledgeID) {
			rootNode = node
		}
	}
	rootTitle := strings.TrimSpace(rootNode.Name)
	if rootTitle == "" {
		rootTitle = strings.TrimSpace(knowledgeID)
	}

	for _, edge := range subgraph.Edges {
		if len(*result) >= limit {
			return
		}
		if !edgeTouchesRoot(edge, rootNode, knowledgeID) {
			continue
		}
		neighborID := strings.TrimSpace(edge.SourceID)
		if sameKnowledgeID(neighborID, knowledgeID) {
			neighborID = strings.TrimSpace(edge.TargetID)
		}
		node, ok := nodeByID[neighborID]
		if !ok || sameKnowledge(node, knowledgeID) {
			continue
		}
		kid := strings.TrimSpace(node.RouteKnowledgeID)
		if kid == "" {
			kid = strings.TrimSpace(node.ID)
		}
		if kid == "" {
			continue
		}
		key := normalizeStateKey(kid)
		if seen[key] {
			continue
		}
		state, _ := lookupStateForNode(stateLookup, node)
		title := displayNodeTitle(node)
		if title == "" {
			title = kid
		}
		seen[key] = true
		item := recommenddto.PathItemView{
			Stage:         stageFromScene(""),
			Title:         title,
			Detail:        contextualDetail(rootTitle, "图谱关联", state, true, title),
			Route:         fmt.Sprintf("/knowledge/%s", kid),
			KnowledgeID:   kid,
			ReasonCode:    "knowledge_graph",
			ReasonLabel:   "图谱关联",
			TargetCourse:  graphNodeCourseTitle(node, ""),
			MasteryScore:  state.MasteryScore,
			CorrectRate:   state.CorrectRate,
			AnsweredCount: state.AnsweredCount,
		}
		*result = append(*result, candidateEntry{
			state:         state,
			knowledgeID:   kid,
			title:         title,
			sourceCourse:  "",
			targetCourse:  item.TargetCourse,
			relationType:  strings.TrimSpace(edge.RelationType),
			sourceMastery: 0,
			item:          item,
		})
	}
}

func (s *Service) collectGapBridgeCandidates(ctx context.Context, knowledgeID string, stateLookup map[string]StudentKnowledgeState, seen map[string]bool, result *[]candidateEntry, limit int) {
	chain, err := s.graphService.GetPrerequisiteChain(ctx, graphdto.PrerequisiteChainQuery{
		Target: strings.TrimSpace(knowledgeID),
		Depth:  3,
	})
	if err != nil || chain == nil || len(chain.Nodes) == 0 {
		return
	}

	nodeByKey := buildNodeLookup(chain.Nodes)
	orderedNodes := orderedGraphNodes(chain.OrderedNodeIDs, nodeByKey)
	if len(orderedNodes) == 0 {
		return
	}

	lastMasteredIndex := -1
	for idx, node := range orderedNodes {
		state, hasState := lookupStateForNode(stateLookup, node)
		if hasState && state.MasteryScore >= 60 {
			lastMasteredIndex = idx
		}
	}

	startIndex := lastMasteredIndex + 1
	if startIndex >= len(orderedNodes) {
		return
	}

	targetNode := orderedNodes[len(orderedNodes)-1]
	targetTitle := displayNodeTitle(targetNode)

	for index := startIndex; index < len(orderedNodes) && len(*result) < limit; index++ {
		node := orderedNodes[index]
		state, hasState := lookupStateForNode(stateLookup, node)
		if index == len(orderedNodes)-1 && hasState && state.MasteryScore >= 80 && state.CorrectRate >= 0.8 && state.AnsweredCount > 0 {
			continue
		}
		title := displayNodeTitle(node)
		kid := strings.TrimSpace(node.RouteKnowledgeID)
		if kid == "" {
			kid = strings.TrimSpace(node.ID)
		}
		if kid == "" {
			continue
		}
		key := normalizeStateKey(kid)
		if seen[key] {
			continue
		}
		seen[key] = true
		item := recommenddto.PathItemView{
			Stage:        stageFromScene(""),
			Title:        title,
			Detail:       fmt.Sprintf("为了更稳地走到目标「%s」，建议先补上桥接知识「%s」。", targetTitle, title),
			Route:        fmt.Sprintf("/knowledge/%s", kid),
			KnowledgeID:  kid,
			ReasonCode:   "gap_bridge",
			ReasonLabel:  "缺口桥接",
			TargetCourse: graphNodeCourseTitle(node, ""),
			MasteryScore: state.MasteryScore,
			CorrectRate:  state.CorrectRate,
		}
		*result = append(*result, candidateEntry{
			state:         state,
			knowledgeID:   kid,
			title:         title,
			sourceCourse:  "",
			targetCourse:  item.TargetCourse,
			relationType:  "PREREQUISITE_OF",
			sourceMastery: 0,
			item:          item,
		})
	}
}

func (s *Service) collectCrossCourseCandidates(ctx context.Context, progressList []StudentKnowledgeState, stateLookup map[string]StudentKnowledgeState, seen map[string]bool, result *[]candidateEntry, limit int) {
	strongStates := append([]StudentKnowledgeState(nil), progressList...)
	sort.SliceStable(strongStates, func(i, j int) bool {
		if strongStates[i].MasteryScore == strongStates[j].MasteryScore {
			return strongStates[i].CorrectRate > strongStates[j].CorrectRate
		}
		return strongStates[i].MasteryScore > strongStates[j].MasteryScore
	})

	sourceBudget := 0
	for _, sourceState := range strongStates {
		if sourceBudget >= 6 || len(*result) >= limit {
			break
		}
		if sourceState.MasteryScore < 60 || sourceState.AnsweredCount <= 0 {
			continue
		}
		sourceBudget++

		subgraph, err := s.graphService.GetSubgraph(ctx, graphdto.SubgraphQuery{
			Root:  strings.TrimSpace(sourceState.KnowledgeID),
			Limit: 10,
		})
		if err != nil || subgraph == nil || len(subgraph.Nodes) == 0 {
			continue
		}

		nodeByKey := buildNodeLookup(subgraph.Nodes)
		rootNode, ok := findGraphNode(nodeByKey, sourceState.KnowledgeID, sourceState.CanonicalName)
		if !ok {
			continue
		}

		sourceCourse := graphNodeCourseTitle(rootNode, "")
		if sourceCourse == "" {
			continue
		}

		for _, edge := range subgraph.Edges {
			if len(*result) >= limit {
				return
			}
			if !edgeTouchesRoot(edge, rootNode, sourceState.KnowledgeID) {
				continue
			}
			neighborID := strings.TrimSpace(edge.SourceID)
			if sameKnowledgeID(neighborID, sourceState.KnowledgeID) {
				neighborID = strings.TrimSpace(edge.TargetID)
			}
			neighborNode, ok := findGraphNode(nodeByKey, neighborID, edge.SourceName, edge.TargetName)
			if !ok || sameKnowledge(neighborNode, sourceState.KnowledgeID) {
				continue
			}
			targetCourse := graphNodeCourseTitle(neighborNode, "")
			if targetCourse == "" || normalizeStateKey(targetCourse) == normalizeStateKey(sourceCourse) {
				continue
			}
			kid := strings.TrimSpace(neighborNode.RouteKnowledgeID)
			if kid == "" {
				kid = strings.TrimSpace(neighborNode.ID)
			}
			if kid == "" {
				continue
			}
			key := normalizeStateKey(kid)
			if seen[key] {
				continue
			}
			targetState, _ := lookupStateForNode(stateLookup, neighborNode)
			title := displayNodeTitle(neighborNode)
			if title == "" {
				title = kid
			}
			seen[key] = true
			item := recommenddto.PathItemView{
				Stage:         stageFromScene("cross_course_transfer"),
				Title:         title,
				Detail:        fmt.Sprintf("你在「%s」课程的「%s」掌握较好，可以把这部分能力迁移到「%s」课程的「%s」。", sourceCourse, displayNodeTitle(rootNode), targetCourse, title),
				Route:         fmt.Sprintf("/knowledge/%s", kid),
				KnowledgeID:   kid,
				ReasonCode:    "cross_course_transfer",
				ReasonLabel:   "跨课程迁移",
				SourceCourse:  sourceCourse,
				TargetCourse:  targetCourse,
				MasteryScore:  targetState.MasteryScore,
				CorrectRate:   targetState.CorrectRate,
				AnsweredCount: targetState.AnsweredCount,
			}
			*result = append(*result, candidateEntry{
				state:         targetState,
				knowledgeID:   kid,
				title:         title,
				sourceCourse:  sourceCourse,
				targetCourse:  targetCourse,
				relationType:  strings.TrimSpace(edge.RelationType),
				sourceMastery: sourceState.MasteryScore,
				item:          item,
			})
		}
	}
}

func scoringBreakdownToView(overall float64, factors ScoringFactors) *recommenddto.ScoringBreakdownView {
	return &recommenddto.ScoringBreakdownView{
		Weakness:              clamp(factors.Weakness, 0, 1),
		PrerequisiteReadiness: clamp(factors.PrerequisiteReadiness, 0, 1),
		TargetRelevance:       clamp(factors.TargetRelevance, 0, 1),
		ForgettingRisk:        clamp(factors.ForgettingRisk, 0, 1),
		CrossCourseTransfer:   clamp(factors.CrossCourseTransfer, 0, 1),
		FrontierRelevance:     clamp(factors.FrontierRelevance, 0, 1),
		MasteredPenalty:       clamp(factors.MasteredPenalty, 0, 1),
		Overall:               clamp(overall, 0, 1),
	}
}
