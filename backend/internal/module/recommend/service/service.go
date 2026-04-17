package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

// Service provides personalized recommendation abilities.
type Service struct {
	graphService *graphservice.Service
	redisRepo    *learningrepo.RedisRepository
	stateRepo    learningrepo.StateRepository
}

type studentKnowledgeState struct {
	KnowledgeID    string
	CanonicalName  string
	MasteryScore   float64
	CorrectRate    float64
	AnsweredCount  int
	LastAnsweredAt string
	Status         string
}

type scoredCandidate struct {
	Node          graphdto.GraphNodeView
	KnowledgeID   string
	Title         string
	ReasonCode    string
	ReasonLabel   string
	SourceCourse  string
	TargetCourse  string
	ReasonDetail  string
	MasteryScore  float64
	CorrectRate   float64
	AnsweredCount int
	Score         float64
}

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

// New creates a recommend service.
func New(graphService *graphservice.Service, redisRepo *learningrepo.RedisRepository, stateRepo learningrepo.StateRepository) *Service {
	return &Service{graphService: graphService, redisRepo: redisRepo, stateRepo: stateRepo}
}

// GetPath returns personalized recommendation path items.
func (s *Service) GetPath(ctx context.Context, query recommenddto.PathQuery) (*recommenddto.PathResponse, error) {
	studentID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}

	target := strings.TrimSpace(query.Target)
	knowledgeID := strings.TrimSpace(query.KnowledgeID)
	limit := limitValue(query.Limit)
	cacheTarget := normalizeRecommendCacheTarget(target, knowledgeID)

	if s.shouldUseCache(query) && s.redisRepo != nil && s.redisRepo.Enabled() {
		if cached, err := s.redisRepo.GetRecommendPath(ctx, studentID, cacheTarget, query.Scene, limit); err == nil && cached != "" {
			var response recommenddto.PathResponse
			if jsonErr := json.Unmarshal([]byte(cached), &response); jsonErr == nil && isUsableRecommendPath(&response) {
				return &response, nil
			}
		}
	}

	progressList, stateLookup := s.loadStudentStates(ctx, studentID)

	path := s.buildPersonalizedPath(ctx, query, progressList, stateLookup, limit)
	if len(path) == 0 {
		root := knowledgeID
		if root == "" {
			root = target
		}
		path = s.buildGraphDrivenPath(ctx, root, query.Scene, limit, stateLookup)
	}
	if len(path) == 0 {
		path = fallbackRecommendPath(query.Scene, limit)
	}

	response := &recommenddto.PathResponse{
		Path:           path,
		Insights:       s.DetectLearningBottlenecks(ctx, query, progressList, stateLookup, path),
		LearnerProfile: inferLearnerProfile(progressList),
	}
	if s.shouldUseCache(query) && s.redisRepo != nil && s.redisRepo.Enabled() {
		if payload, err := json.Marshal(response); err == nil {
			_ = s.redisRepo.CacheRecommendPath(ctx, studentID, cacheTarget, query.Scene, limit, string(payload))
		}
	}
	return response, nil
}

func (s *Service) shouldUseCache(query recommenddto.PathQuery) bool {
	if strings.TrimSpace(query.Scene) == "detail" {
		return false
	}
	return strings.TrimSpace(query.KnowledgeID) == ""
}

func (s *Service) loadStudentStates(ctx context.Context, studentID int64) ([]studentKnowledgeState, map[string]studentKnowledgeState) {
	if s.redisRepo != nil && s.redisRepo.Enabled() {
		if states, err := s.redisRepo.GetMasteryStates(ctx, studentID); err == nil && len(states) > 0 {
			result := make([]studentKnowledgeState, 0, len(states))
			lookup := make(map[string]studentKnowledgeState, len(states)*2)
			for knowledgeID, state := range states {
				correctRate := 0.0
				if state.AnsweredCount > 0 {
					correctRate = float64(state.CorrectCount) / float64(state.AnsweredCount)
				}
				lastAnsweredAt := ""
				if state.LastAnsweredAt != nil {
					lastAnsweredAt = state.LastAnsweredAt.Format(time.RFC3339)
				}
				item := studentKnowledgeState{
					KnowledgeID:    strings.TrimSpace(knowledgeID),
					CanonicalName:  strings.TrimSpace(knowledgeID),
					MasteryScore:   state.MasteryScore,
					CorrectRate:    correctRate,
					AnsweredCount:  state.AnsweredCount,
					LastAnsweredAt: lastAnsweredAt,
					Status:         progressStatus(state.MasteryScore),
				}
				result = append(result, item)
				registerStudentState(lookup, item)
			}
			return result, lookup
		}
	}

	if s.stateRepo == nil {
		return []studentKnowledgeState{}, map[string]studentKnowledgeState{}
	}

	rows, err := s.stateRepo.ListLearningProgress(ctx, studentID)
	if err != nil {
		return []studentKnowledgeState{}, map[string]studentKnowledgeState{}
	}

	result := make([]studentKnowledgeState, 0, len(rows))
	lookup := make(map[string]studentKnowledgeState, len(rows)*2)
	for _, row := range rows {
		lastAnsweredAt := ""
		if row.LastAnsweredAt != nil {
			lastAnsweredAt = row.LastAnsweredAt.Format(time.RFC3339)
		}
		item := studentKnowledgeState{
			KnowledgeID:    strings.TrimSpace(row.KnowledgeID),
			CanonicalName:  strings.TrimSpace(row.CanonicalName),
			MasteryScore:   row.MasteryScore,
			CorrectRate:    row.CorrectRate,
			AnsweredCount:  row.AnsweredCount,
			LastAnsweredAt: lastAnsweredAt,
			Status:         strings.TrimSpace(row.Status),
		}
		result = append(result, item)
		registerStudentState(lookup, item)
	}
	return result, lookup
}

func registerStudentState(lookup map[string]studentKnowledgeState, item studentKnowledgeState) {
	if lookup == nil {
		return
	}
	keys := []string{
		normalizeStateKey(item.KnowledgeID),
		normalizeStateKey(item.CanonicalName),
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		lookup[key] = item
	}
}

func normalizeStateKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func progressStatus(score float64) string {
	if score >= 80 {
		return "solid"
	}
	if score > 0 {
		return "progressing"
	}
	return "not_started"
}

func (s *Service) buildPersonalizedPath(ctx context.Context, query recommenddto.PathQuery, progressList []studentKnowledgeState, stateLookup map[string]studentKnowledgeState, limit int) []recommenddto.PathItemView {
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

func (s *Service) buildWeaknessPath(progressList []studentKnowledgeState, scene string, limit int) []recommenddto.PathItemView {
	if len(progressList) == 0 || limit <= 0 {
		return nil
	}

	sorted := append([]studentKnowledgeState(nil), progressList...)
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

func weaknessScore(item studentKnowledgeState) float64 {
	score := 0.0
	switch {
	case item.MasteryScore < 20:
		score += 62
	case item.MasteryScore < 40:
		score += 52
	case item.MasteryScore < 60:
		score += 38
	case item.MasteryScore < 80:
		score += 14
	default:
		score -= 18
	}
	if item.AnsweredCount == 0 {
		score += 8
	} else if item.CorrectRate < 0.5 {
		score += 18
	} else if item.CorrectRate < 0.7 {
		score += 10
	}
	if item.Status == "not_started" {
		score += 6
	}
	return score
}

func weaknessDetail(item studentKnowledgeState) string {
	name := strings.TrimSpace(item.CanonicalName)
	if name == "" {
		name = strings.TrimSpace(item.KnowledgeID)
	}
	if item.AnsweredCount == 0 {
		return fmt.Sprintf("%s 当前还没有稳定学习记录，建议先从基础定义和最小例子入手。", name)
	}
	return fmt.Sprintf("%s 当前掌握度 %.0f%%，正确率 %.0f%%，适合作为优先补强点。", name, item.MasteryScore, item.CorrectRate*100)
}

func (s *Service) buildKnowledgeContextPath(ctx context.Context, knowledgeID string, scene string, limit int, stateLookup map[string]studentKnowledgeState) []recommenddto.PathItemView {
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
		if !ok {
			continue
		}
		if sameKnowledge(node, knowledgeID) {
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

func lookupStateForNode(lookup map[string]studentKnowledgeState, node graphdto.GraphNodeView) (studentKnowledgeState, bool) {
	keys := []string{node.RouteKnowledgeID, node.ID, node.Name, node.CanonicalName}
	for _, key := range keys {
		if item, ok := lookup[normalizeStateKey(key)]; ok {
			return item, true
		}
	}
	return studentKnowledgeState{}, false
}

func buildScoredCandidate(rootTitle string, rootState studentKnowledgeState, edge graphdto.GraphEdgeView, node graphdto.GraphNodeView, lookup map[string]studentKnowledgeState) scoredCandidate {
	state, hasState := lookupStateForNode(lookup, node)
	relationBoost, reasonCode, reasonLabel := relationScore(rootState, edge, node)
	stateBoost := masteryNeedScore(state, hasState)
	score := normalizedRecommendationScore(relationBoost + stateBoost)

	title := strings.TrimSpace(node.Name)
	if title == "" {
		title = strings.TrimSpace(node.CanonicalName)
	}
	if title == "" {
		title = strings.TrimSpace(node.RouteKnowledgeID)
	}

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
		TargetCourse:  strings.TrimSpace(node.ChapterTitle),
		ReasonDetail:  contextualDetail(rootTitle, reasonLabel, state, hasState, title),
		MasteryScore:  state.MasteryScore,
		CorrectRate:   state.CorrectRate,
		AnsweredCount: state.AnsweredCount,
		Score:         score,
	}
}

func buildFallbackCandidate(rootTitle string, node graphdto.GraphNodeView, lookup map[string]studentKnowledgeState) scoredCandidate {
	state, hasState := lookupStateForNode(lookup, node)
	title := strings.TrimSpace(node.Name)
	if title == "" {
		title = strings.TrimSpace(node.CanonicalName)
	}
	if title == "" {
		title = strings.TrimSpace(node.RouteKnowledgeID)
	}
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
		TargetCourse:  strings.TrimSpace(node.ChapterTitle),
		ReasonDetail:  contextualDetail(rootTitle, "图谱关联", state, hasState, title),
		MasteryScore:  state.MasteryScore,
		CorrectRate:   state.CorrectRate,
		AnsweredCount: state.AnsweredCount,
		Score:         normalizedRecommendationScore(18 + masteryNeedScore(state, hasState)),
	}
}

func relationScore(rootState studentKnowledgeState, edge graphdto.GraphEdgeView, node graphdto.GraphNodeView) (float64, string, string) {
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

func masteryNeedScore(state studentKnowledgeState, hasState bool) float64 {
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

func contextualDetail(rootTitle string, reasonLabel string, state studentKnowledgeState, hasState bool, title string) string {
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

func (s *Service) buildGraphDrivenPath(ctx context.Context, target string, scene string, limit int, stateLookup map[string]studentKnowledgeState) []recommenddto.PathItemView {
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
			title := strings.TrimSpace(node.Name)
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
				TargetCourse:  strings.TrimSpace(node.ChapterTitle),
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

func (s *Service) buildCrossCourseTransferPath(ctx context.Context, progressList []studentKnowledgeState, stateLookup map[string]studentKnowledgeState, limit int) []recommenddto.PathItemView {
	if s.graphService == nil || limit <= 0 || len(progressList) == 0 {
		return nil
	}

	strongStates := append([]studentKnowledgeState(nil), progressList...)
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

		sourceCourse := strings.TrimSpace(rootNode.ChapterTitle)
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

			targetCourse := strings.TrimSpace(neighborNode.ChapterTitle)
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
				Stage:         stageFromScene("cross_course_transfer"),
				Title:         targetTitle,
				Detail:        fmt.Sprintf("你在“%s”课程的「%s」掌握较好，可以把这部分能力迁移到“%s”课程的「%s」。", sourceCourse, sourceTitle, targetCourse, targetTitle),
				Route:         fmt.Sprintf("/knowledge/%s", knowledgeID),
				KnowledgeID:   knowledgeID,
				ReasonCode:    "cross_course_transfer",
				ReasonLabel:   "跨课程迁移",
				SourceCourse:  sourceCourse,
				TargetCourse:  targetCourse,
				MasteryScore:  targetState.MasteryScore,
				CorrectRate:   targetState.CorrectRate,
				AnsweredCount: targetState.AnsweredCount,
				Score:         normalizedRecommendationScore(rawScore),
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

func (s *Service) buildGapBridgePath(ctx context.Context, knowledgeID string, scene string, limit int, stateLookup map[string]studentKnowledgeState) []recommenddto.PathItemView {
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
	targetCourse := strings.TrimSpace(targetNode.ChapterTitle)
	sourceCourse := targetCourse
	sourceTitle := ""
	if lastMasteredIndex >= 0 {
		sourceNode := orderedNodes[lastMasteredIndex]
		sourceCourse = firstNonEmptyCourse(strings.TrimSpace(sourceNode.ChapterTitle), targetCourse)
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
			TargetCourse:  firstNonEmptyCourse(strings.TrimSpace(node.ChapterTitle), targetCourse),
			MasteryScore:  state.MasteryScore,
			CorrectRate:   state.CorrectRate,
			AnsweredCount: state.AnsweredCount,
			Score:         normalizedRecommendationScore(58 - float64(index-startIndex)*6 + masteryNeedScore(state, hasState)),
		})
	}

	return path
}

func (s *Service) buildReviewPath(progressList []studentKnowledgeState, scene string, limit int) []recommenddto.PathItemView {
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

// DetectLearningBottlenecks summarizes the strongest current learning risks.
func (s *Service) DetectLearningBottlenecks(ctx context.Context, query recommenddto.PathQuery, progressList []studentKnowledgeState, stateLookup map[string]studentKnowledgeState, recommendedPath []recommenddto.PathItemView) []recommenddto.LearningInsightView {
	insights := make([]recommenddto.LearningInsightView, 0, 3)
	seen := make(map[string]struct{}, 4)
	appendInsight := func(item recommenddto.LearningInsightView) {
		key := strings.TrimSpace(item.ReasonCode)
		if key == "" {
			key = strings.TrimSpace(item.Title)
		}
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		insights = append(insights, item)
	}

	if knowledgeID := strings.TrimSpace(query.KnowledgeID); knowledgeID != "" {
		if gapPath := s.buildGapBridgePath(ctx, knowledgeID, query.Scene, 1, stateLookup); len(gapPath) > 0 {
			item := gapPath[0]
			appendInsight(recommenddto.LearningInsightView{
				Type:        "proactive_hint",
				ReasonCode:  "gap_bridge",
				Severity:    "high",
				Title:       "当前目标前有基础断层",
				Detail:      item.Detail,
				KnowledgeID: item.KnowledgeID,
				Route:       item.Route,
			})
		}
	}

	if weakness := topWeaknessCandidate(progressList); weakness != nil {
		appendInsight(recommenddto.LearningInsightView{
			Type:        "proactive_hint",
			ReasonCode:  "weakness",
			Severity:    weaknessSeverity(*weakness),
			Title:       "有一个知识点在反复卡住你",
			Detail:      weaknessDetail(*weakness),
			KnowledgeID: strings.TrimSpace(weakness.KnowledgeID),
			Route:       fmt.Sprintf("/knowledge/%s", strings.TrimSpace(weakness.KnowledgeID)),
		})
	}

	if reviewPath := s.buildReviewPath(progressList, query.Scene, 1); len(reviewPath) > 0 {
		item := reviewPath[0]
		appendInsight(recommenddto.LearningInsightView{
			Type:        "proactive_hint",
			ReasonCode:  "forgetting_review",
			Severity:    "medium",
			Title:       "有一个知识点到了回温窗口",
			Detail:      item.Detail,
			KnowledgeID: item.KnowledgeID,
			Route:       item.Route,
		})
	}

	if transfer := topReasonPathItem(recommendedPath, "cross_course_transfer"); transfer != nil {
		appendInsight(recommenddto.LearningInsightView{
			Type:        "proactive_hint",
			ReasonCode:  "cross_course_transfer",
			Severity:    "low",
			Title:       "你可以利用已有强项跨课程迁移",
			Detail:      transfer.Detail,
			KnowledgeID: transfer.KnowledgeID,
			Route:       transfer.Route,
		})
	}

	if len(insights) > 3 {
		return insights[:3]
	}
	return insights
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

func detailForNode(nodeName string, state studentKnowledgeState, hasState bool) string {
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
	return firstNonEmptyText(strings.TrimSpace(node.Name), strings.TrimSpace(node.CanonicalName), strings.TrimSpace(node.RouteKnowledgeID), strings.TrimSpace(node.ID))
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
	return firstNonEmptyText(values...)
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

func topWeaknessCandidate(progressList []studentKnowledgeState) *studentKnowledgeState {
	if len(progressList) == 0 {
		return nil
	}
	candidates := append([]studentKnowledgeState(nil), progressList...)
	sort.SliceStable(candidates, func(i, j int) bool {
		left := weaknessScore(candidates[i])
		right := weaknessScore(candidates[j])
		if math.Abs(left-right) < 0.001 {
			return candidates[i].CanonicalName < candidates[j].CanonicalName
		}
		return left > right
	})
	for _, item := range candidates {
		if item.MasteryScore >= 55 && item.CorrectRate >= 0.65 {
			continue
		}
		if item.AnsweredCount == 0 && item.MasteryScore >= 40 {
			continue
		}
		copyItem := item
		return &copyItem
	}
	return nil
}

func weaknessSeverity(item studentKnowledgeState) string {
	if item.MasteryScore < 30 || (item.AnsweredCount >= 3 && item.CorrectRate < 0.45) {
		return "high"
	}
	if item.MasteryScore < 55 || item.CorrectRate < 0.6 {
		return "medium"
	}
	return "low"
}

func topReasonPathItem(items []recommenddto.PathItemView, reasonCode string) *recommenddto.PathItemView {
	for _, item := range items {
		if strings.TrimSpace(item.ReasonCode) != strings.TrimSpace(reasonCode) {
			continue
		}
		copyItem := item
		return &copyItem
	}
	return nil
}

func inferLearnerProfile(progressList []studentKnowledgeState) *recommenddto.LearnerProfileView {
	activeNodes := 0
	totalAnswered := 0
	totalMastery := 0.0
	strongNodes := 0
	weakNodes := 0

	for _, item := range progressList {
		if item.AnsweredCount > 0 {
			activeNodes++
			totalAnswered += item.AnsweredCount
		}
		totalMastery += item.MasteryScore
		if item.MasteryScore >= 80 {
			strongNodes++
		}
		if item.AnsweredCount > 0 && item.MasteryScore < 40 {
			weakNodes++
		}
	}

	if len(progressList) == 0 || activeNodes == 0 {
		return &recommenddto.LearnerProfileView{
			Style:   "exploratory",
			Label:   "探索型",
			Summary: "当前系统里的学习记录还不多，适合先从兴趣点切入，再逐步把主线补齐。",
		}
	}

	avgDepth := float64(totalAnswered) / float64(activeNodes)
	avgMastery := totalMastery / float64(len(progressList))
	weakRatio := float64(weakNodes) / float64(activeNodes)
	strongRatio := float64(strongNodes) / float64(max(len(progressList), 1))

	switch {
	case weakRatio >= 0.45 || avgMastery < 48:
		return &recommenddto.LearnerProfileView{
			Style:   "struggling",
			Label:   "困难型",
			Summary: "你最近更适合被拆成更小步地学习，先稳住基础和正确率，再往前推进。",
		}
	case avgDepth >= 9 && activeNodes <= 5:
		return &recommenddto.LearnerProfileView{
			Style:   "focused",
			Label:   "专注型",
			Summary: "你倾向于深挖少数主题，适合直接给出重点、路径和少量关键对比。",
		}
	case activeNodes >= 10 && avgDepth <= 4.5:
		return &recommenddto.LearnerProfileView{
			Style:   "exploratory",
			Label:   "探索型",
			Summary: "你喜欢从多个知识点并行摸索，适合多给跨课程连接和有趣的入口。",
		}
	case activeNodes >= 8 && strongRatio >= 0.28:
		return &recommenddto.LearnerProfileView{
			Style:   "systematic",
			Label:   "系统型",
			Summary: "你更适合按知识树和先修顺序稳步推进，结构化学习会让你更快进入状态。",
		}
	default:
		return &recommenddto.LearnerProfileView{
			Style:   "focused",
			Label:   "专注型",
			Summary: "你当前更适合围绕一个目标连续推进，少切换主题会更高效。",
		}
	}
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
