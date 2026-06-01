package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/han139242/course-recommend-agents/backend/internal/coursemeta"
	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
)

func (s *Service) buildPersonalizedPath(ctx context.Context, query recommenddto.PathQuery, progressList []StudentKnowledgeState, stateLookup map[string]StudentKnowledgeState, limit int) []recommenddto.PathItemView {
	result := make([]recommenddto.PathItemView, 0, limit)
	sectionLimit := min(max(limit, 1), 4)
	knowledgeID := strings.TrimSpace(query.KnowledgeID)

	if knowledgeID != "" {
		result = mergePathItems(result, s.buildGapBridgePath(ctx, knowledgeID, query.Scene, sectionLimit, stateLookup), limit)
		if len(result) >= limit {
			return result
		}
		result = mergePathItems(result, s.buildKnowledgeContextPath(ctx, knowledgeID, query.Scene, sectionLimit, stateLookup), limit)
		if len(result) >= limit {
			return result
		}
	}

	result = mergePathItems(result, s.buildCrossCourseTransferPath(ctx, progressList, stateLookup, sectionLimit), limit)
	if len(result) >= limit {
		return result
	}
	result = mergePathItems(result, s.buildReviewPath(progressList, query.Scene, sectionLimit), limit)
	if len(result) >= limit {
		return result
	}
	result = mergePathItems(result, s.buildWeaknessPath(progressList, query.Scene, limit), limit)
	if len(result) >= limit {
		return result
	}

	root := strings.TrimSpace(query.Target)
	graphPath := s.buildGraphDrivenPath(ctx, root, query.Scene, max(limit*2, 8), stateLookup)
	return mergePathItems(result, graphPath, limit)
}

func (s *Service) buildWeaknessPath(progressList []StudentKnowledgeState, scene string, limit int) []recommenddto.PathItemView {
	if len(progressList) == 0 || limit <= 0 {
		return nil
	}

	sorted := append([]StudentKnowledgeState(nil), progressList...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := weaknessScore(sorted[i])
		right := weaknessScore(sorted[j])
		if math.Abs(left-right) < 0.001 {
			return sorted[i].CanonicalName < sorted[j].CanonicalName
		}
		return left > right
	})

	path := make([]recommenddto.PathItemView, 0, limit)
	for _, item := range sorted {
		knowledgeID := strings.TrimSpace(item.KnowledgeID)
		if knowledgeID == "" {
			continue
		}
		if item.MasteryScore >= 85 && item.CorrectRate >= 0.8 {
			continue
		}
		title := strings.TrimSpace(item.CanonicalName)
		if title == "" {
			title = knowledgeID
		}
		path = append(path, recommenddto.PathItemView{
			Stage:         stageFromScene(scene),
			Title:         title,
			Detail:        weaknessDetail(item),
			Route:         fmt.Sprintf("/knowledge/%s", knowledgeID),
			KnowledgeID:   knowledgeID,
			ReasonCode:    "weakness",
			ReasonLabel:   "薄弱补强",
			MasteryScore:  item.MasteryScore,
			CorrectRate:   item.CorrectRate,
			AnsweredCount: item.AnsweredCount,
			Score:         normalizedRecommendationScore(weaknessScore(item)),
		})
		if len(path) >= limit {
			break
		}
	}
	return path
}

func (s *Service) buildKnowledgeContextPath(ctx context.Context, knowledgeID string, scene string, limit int, stateLookup map[string]StudentKnowledgeState) []recommenddto.PathItemView {
	if s.graphService == nil || limit <= 0 {
		return nil
	}

	subgraph, err := s.graphService.GetSubgraph(ctx, graphdto.SubgraphQuery{Root: knowledgeID, Limit: max(limit*4, 12)})
	if err != nil || subgraph == nil || len(subgraph.Nodes) == 0 {
		return nil
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

	rootState, _ := lookupStateForNode(stateLookup, rootNode)
	scored := make(map[string]scoredCandidate)
	for _, edge := range subgraph.Edges {
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

		candidate := buildScoredCandidate(rootTitle, rootState, edge, node, stateLookup)
		if candidate.KnowledgeID == "" || candidate.Title == "" {
			continue
		}
		if existing, exists := scored[candidate.KnowledgeID]; !exists || candidate.Score > existing.Score {
			scored[candidate.KnowledgeID] = candidate
		}
	}

	if len(scored) == 0 {
		for _, node := range prioritizeRecommendNodes(knowledgeID, subgraph.Nodes) {
			if sameKnowledge(node, knowledgeID) {
				continue
			}
			candidate := buildFallbackCandidate(rootTitle, node, stateLookup)
			if candidate.KnowledgeID == "" || candidate.Title == "" {
				continue
			}
			scored[candidate.KnowledgeID] = candidate
		}
	}

	items := make([]scoredCandidate, 0, len(scored))
	for _, item := range scored {
		if item.MasteryScore >= 88 && item.CorrectRate >= 0.85 && item.AnsweredCount > 0 {
			continue
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if math.Abs(items[i].Score-items[j].Score) < 0.001 {
			return items[i].Title < items[j].Title
		}
		return items[i].Score > items[j].Score
	})

	path := make([]recommenddto.PathItemView, 0, min(limit, len(items)))
	for _, item := range items {
		path = append(path, recommenddto.PathItemView{
			Stage:         stageFromScene(scene),
			Title:         item.Title,
			Detail:        item.ReasonDetail,
			Route:         fmt.Sprintf("/knowledge/%s", item.KnowledgeID),
			KnowledgeID:   item.KnowledgeID,
			ReasonCode:    item.ReasonCode,
			ReasonLabel:   item.ReasonLabel,
			SourceCourse:  item.SourceCourse,
			TargetCourse:  item.TargetCourse,
			MasteryScore:  item.MasteryScore,
			CorrectRate:   item.CorrectRate,
			AnsweredCount: item.AnsweredCount,
			Score:         item.Score,
		})
		if len(path) >= limit {
			break
		}
	}
	return path
}

func (s *Service) buildGraphDrivenPath(ctx context.Context, target string, scene string, limit int, stateLookup map[string]StudentKnowledgeState) []recommenddto.PathItemView {
	if s.graphService == nil || limit <= 0 {
		return nil
	}

	path := make([]recommenddto.PathItemView, 0, limit)
	seenRoutes := make(map[string]struct{}, limit)
	for _, root := range recommendRoots(target) {
		subgraph, err := s.graphService.GetSubgraph(ctx, graphdto.SubgraphQuery{Root: root, Limit: limit})
		if err != nil || subgraph == nil || len(subgraph.Nodes) == 0 {
			continue
		}
		for _, node := range prioritizeRecommendNodes(root, subgraph.Nodes) {
			routeID := strings.TrimSpace(node.RouteKnowledgeID)
			title := displayNodeTitle(node)
			if routeID == "" || title == "" {
				continue
			}
			if _, exists := seenRoutes[routeID]; exists {
				continue
			}
			state, hasState := lookupStateForNode(stateLookup, node)
			if hasState && state.MasteryScore >= 88 && state.CorrectRate >= 0.85 && state.AnsweredCount > 0 {
				continue
			}
			seenRoutes[routeID] = struct{}{}
			path = append(path, recommenddto.PathItemView{
				Stage:         stageFromScene(scene),
				Title:         title,
				Detail:        detailForNode(title, state, hasState),
				Route:         fmt.Sprintf("/knowledge/%s", routeID),
				KnowledgeID:   routeID,
				ReasonCode:    "knowledge_graph",
				ReasonLabel:   "图谱关联",
				TargetCourse:  graphNodeCourseTitle(node, coursemeta.DefaultCourseTitle),
				MasteryScore:  state.MasteryScore,
				CorrectRate:   state.CorrectRate,
				AnsweredCount: state.AnsweredCount,
				Score:         scoreForIndex(len(path)),
			})
			if len(path) >= limit {
				return path
			}
		}
	}
	return path
}

func (s *Service) buildCrossCourseTransferPath(ctx context.Context, progressList []StudentKnowledgeState, stateLookup map[string]StudentKnowledgeState, limit int) []recommenddto.PathItemView {
	if s.graphService == nil || limit <= 0 || len(progressList) == 0 {
		return nil
	}

	strongStates := append([]StudentKnowledgeState(nil), progressList...)
	sort.SliceStable(strongStates, func(i, j int) bool {
		if math.Abs(strongStates[i].MasteryScore-strongStates[j].MasteryScore) < 0.001 {
			return strongStates[i].CorrectRate > strongStates[j].CorrectRate
		}
		return strongStates[i].MasteryScore > strongStates[j].MasteryScore
	})

	scored := make(map[string]recommenddto.PathItemView)
	sourceBudget := 0
	for _, sourceState := range strongStates {
		if sourceBudget >= 6 {
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
		sourceTitle := displayNodeTitle(rootNode)

		for _, edge := range subgraph.Edges {
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

			targetState, hasTargetState := lookupStateForNode(stateLookup, neighborNode)
			if hasTargetState && targetState.MasteryScore >= 85 && targetState.CorrectRate >= 0.8 && targetState.AnsweredCount > 0 {
				continue
			}

			knowledgeID := strings.TrimSpace(neighborNode.RouteKnowledgeID)
			if knowledgeID == "" {
				knowledgeID = strings.TrimSpace(neighborNode.ID)
			}
			if knowledgeID == "" {
				continue
			}

			targetTitle := displayNodeTitle(neighborNode)
			rawScore := (sourceState.MasteryScore * transferRelationWeight(edge.RelationType) * (1 - min(targetState.MasteryScore, 100)/100)) / 2.1
			if !hasTargetState {
				rawScore += 10
			}

			item := recommenddto.PathItemView{
				Stage:       stageFromScene("cross_course_transfer"),
				Title:       targetTitle,
				Detail:      fmt.Sprintf("你在「%s」课程的「%s」掌握较好，可以把这部分能力迁移到「%s」课程的「%s」。", sourceCourse, sourceTitle, targetCourse, targetTitle),
				Route:       fmt.Sprintf("/knowledge/%s", knowledgeID),
				KnowledgeID: knowledgeID,
				ReasonCode:  "cross_course_transfer",
				ReasonLabel: "跨课程迁移",
				SourceCourse: sourceCourse,
				TargetCourse: targetCourse,
				MasteryScore: targetState.MasteryScore,
				CorrectRate:  targetState.CorrectRate,
				AnsweredCount: targetState.AnsweredCount,
				Score:        normalizedRecommendationScore(rawScore),
			}

			if existing, exists := scored[knowledgeID]; !exists || item.Score > existing.Score {
				scored[knowledgeID] = item
			}
		}
	}

	items := make([]recommenddto.PathItemView, 0, len(scored))
	for _, item := range scored {
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if math.Abs(items[i].Score-items[j].Score) < 0.001 {
			if items[i].TargetCourse == items[j].TargetCourse {
				return items[i].Title < items[j].Title
			}
			return items[i].TargetCourse < items[j].TargetCourse
		}
		return items[i].Score > items[j].Score
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func (s *Service) buildGapBridgePath(ctx context.Context, knowledgeID string, scene string, limit int, stateLookup map[string]StudentKnowledgeState) []recommenddto.PathItemView {
	if s.graphService == nil || limit <= 0 || strings.TrimSpace(knowledgeID) == "" {
		return nil
	}

	chain, err := s.graphService.GetPrerequisiteChain(ctx, graphdto.PrerequisiteChainQuery{
		Target: strings.TrimSpace(knowledgeID),
		Depth:  3,
	})
	if err != nil || chain == nil || len(chain.Nodes) == 0 {
		return nil
	}

	nodeByKey := buildNodeLookup(chain.Nodes)
	orderedNodes := orderedGraphNodes(chain.OrderedNodeIDs, nodeByKey)
	if len(orderedNodes) == 0 {
		return nil
	}

	lastMasteredIndex := -1
	for index, node := range orderedNodes {
		state, hasState := lookupStateForNode(stateLookup, node)
		if hasState && state.MasteryScore >= 60 {
			lastMasteredIndex = index
		}
	}

	startIndex := lastMasteredIndex + 1
	if startIndex >= len(orderedNodes) {
		return nil
	}

	targetNode := orderedNodes[len(orderedNodes)-1]
	targetTitle := displayNodeTitle(targetNode)
	targetCourse := graphNodeCourseTitle(targetNode, coursemeta.DefaultCourseTitle)
	sourceCourse := targetCourse
	sourceTitle := ""
	if lastMasteredIndex >= 0 {
		sourceNode := orderedNodes[lastMasteredIndex]
		sourceCourse = firstNonEmptyCourse(graphNodeCourseTitle(sourceNode, ""), targetCourse)
		sourceTitle = displayNodeTitle(sourceNode)
	}

	path := make([]recommenddto.PathItemView, 0, min(limit, len(orderedNodes)-startIndex))
	for index := startIndex; index < len(orderedNodes) && len(path) < limit; index++ {
		node := orderedNodes[index]
		state, hasState := lookupStateForNode(stateLookup, node)
		if index == len(orderedNodes)-1 && hasState && state.MasteryScore >= 80 && state.CorrectRate >= 0.8 && state.AnsweredCount > 0 {
			continue
		}

		title := displayNodeTitle(node)
		routeID := strings.TrimSpace(node.RouteKnowledgeID)
		if routeID == "" {
			routeID = strings.TrimSpace(node.ID)
		}
		if routeID == "" {
			continue
		}

		detail := fmt.Sprintf("为了更稳地走到目标「%s」，建议先补上桥接知识「%s」。", targetTitle, title)
		if sourceTitle != "" && index == startIndex {
			detail = fmt.Sprintf("你已经具备「%s」，从这里到目标「%s」之间，最先需要补上的桥接点是「%s」。", sourceTitle, targetTitle, title)
		}

		path = append(path, recommenddto.PathItemView{
			Stage:         stageFromScene(scene),
			Title:         title,
			Detail:        detail,
			Route:         fmt.Sprintf("/knowledge/%s", routeID),
			KnowledgeID:   routeID,
			ReasonCode:    "gap_bridge",
			ReasonLabel:   "缺口桥接",
			SourceCourse:  sourceCourse,
			TargetCourse:  firstNonEmptyCourse(graphNodeCourseTitle(node, ""), targetCourse),
			MasteryScore:  state.MasteryScore,
			CorrectRate:   state.CorrectRate,
			AnsweredCount: state.AnsweredCount,
			Score:         normalizedRecommendationScore(58 - float64(index-startIndex)*6 + masteryNeedScore(state, hasState)),
		})
	}
	return path
}

func (s *Service) buildReviewPath(progressList []StudentKnowledgeState, scene string, limit int) []recommenddto.PathItemView {
	if limit <= 0 || len(progressList) == 0 {
		return nil
	}

	type reviewCandidate struct {
		item             recommenddto.PathItemView
		effectiveMastery float64
	}

	candidates := make([]reviewCandidate, 0, len(progressList))
	for _, state := range progressList {
		if state.MasteryScore < 60 || strings.TrimSpace(state.LastAnsweredAt) == "" {
			continue
		}
		effective := effectiveMastery(state.MasteryScore, state.LastAnsweredAt)
		if effective >= 50 {
			continue
		}

		knowledgeID := strings.TrimSpace(state.KnowledgeID)
		title := firstNonEmptyText(strings.TrimSpace(state.CanonicalName), knowledgeID)
		days := daysSinceLastAnswer(state.LastAnsweredAt)
		candidates = append(candidates, reviewCandidate{
			item: recommenddto.PathItemView{
				Stage:         stageFromScene(scene),
				Title:         title,
				Detail:        fmt.Sprintf("你上次练到「%s」已经是 %.0f 天前了，当前有效掌握度约 %.0f%%，适合安排一次复习回温。", title, days, effective),
				Route:         fmt.Sprintf("/knowledge/%s", knowledgeID),
				KnowledgeID:   knowledgeID,
				ReasonCode:    "forgetting_review",
				ReasonLabel:   "遗忘复习",
				MasteryScore:  state.MasteryScore,
				CorrectRate:   state.CorrectRate,
				AnsweredCount: state.AnsweredCount,
				Score:         normalizedRecommendationScore(62 - effective + (state.MasteryScore - effective)),
			},
			effectiveMastery: effective,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if math.Abs(candidates[i].effectiveMastery-candidates[j].effectiveMastery) < 0.001 {
			return candidates[i].item.Title < candidates[j].item.Title
		}
		return candidates[i].effectiveMastery < candidates[j].effectiveMastery
	})

	path := make([]recommenddto.PathItemView, 0, min(limit, len(candidates)))
	for _, candidate := range candidates {
		path = append(path, candidate.item)
		if len(path) >= limit {
			break
		}
	}
	return path
}
