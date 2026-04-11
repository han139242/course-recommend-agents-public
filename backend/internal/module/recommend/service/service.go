package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

// Service 提供规则型推荐能力。
type Service struct {
	graphService *graphservice.Service
	redisRepo    *learningrepo.RedisRepository
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

// New 创建推荐服务。
func New(graphService *graphservice.Service, redisRepo *learningrepo.RedisRepository) *Service {
	return &Service{graphService: graphService, redisRepo: redisRepo}
}

// GetPath 查询推荐路径。
func (s *Service) GetPath(ctx context.Context, query recommenddto.PathQuery) (*recommenddto.PathResponse, error) {
	studentID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}
	target := strings.TrimSpace(query.Target)
	cacheTarget := normalizeRecommendCacheTarget(target)
	limit := limitValue(query.Limit)
	if s.redisRepo != nil && s.redisRepo.Enabled() {
		if cached, err := s.redisRepo.GetRecommendPath(ctx, studentID, cacheTarget, query.Scene, limit); err == nil && cached != "" {
			var response recommenddto.PathResponse
			if jsonErr := json.Unmarshal([]byte(cached), &response); jsonErr == nil && isUsableRecommendPath(&response) {
				return &response, nil
			}
		}
	}
	path := s.buildGraphDrivenPath(ctx, target, query.Scene, limit)
	if len(path) == 0 {
		path = fallbackRecommendPath(query.Scene, limit)
	}
	response := &recommenddto.PathResponse{Path: path}
	if s.redisRepo != nil && s.redisRepo.Enabled() {
		if payload, err := json.Marshal(response); err == nil {
			_ = s.redisRepo.CacheRecommendPath(ctx, studentID, cacheTarget, query.Scene, limit, string(payload))
		}
	}
	return response, nil
}

func (s *Service) buildGraphDrivenPath(ctx context.Context, target string, scene string, limit int) []recommenddto.PathItemView {
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
			seenRoutes[routeID] = struct{}{}
			path = append(path, recommenddto.PathItemView{
				Stage:       stageFromScene(scene),
				Title:       title,
				Detail:      detailForNode(title),
				Route:       fmt.Sprintf("/knowledge/%s", routeID),
				ReasonCode:  "knowledge_graph",
				ReasonLabel: "图谱关联",
				Score:       scoreForIndex(len(path)),
			})
			if len(path) >= limit {
				return path
			}
		}
	}
	return path
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
	return "补弱推荐"
}

func detailForNode(nodeName string) string {
	if strings.TrimSpace(nodeName) == "" {
		return "建议先从这个知识点开始，逐步衔接到后续内容。"
	}
	return fmt.Sprintf("建议先理解 %s，再继续推进到后续更高难度内容。", nodeName)
}

func scoreForIndex(index int) float64 {
	base := 0.95 - float64(index)*0.08
	if base < 0.5 {
		return 0.5
	}
	return base
}

func limitValue(limit int) int {
	if limit <= 0 {
		return 5
	}
	return limit
}

func normalizeRecommendCacheTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return "__auto_v3__"
	}
	return target
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
		{title: "梯度下降", route: "/knowledge/梯度下降", detail: "先补齐优化基础和梯度概念，后续学习会更顺。"},
		{title: "矩阵", route: "/knowledge/矩阵", detail: "矩阵与线性代数是神经网络计算的底层基础。"},
		{title: "卷积神经网络", route: "/knowledge/卷积神经网络", detail: "适合把数学基础和深度学习实践串成一条主线。"},
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
			ReasonCode:  "recommended_start",
			ReasonLabel: "推荐起点",
			Score:       scoreForIndex(idx),
		})
	}
	return path
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
