package service

import (
	"context"
	"strings"

	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	graphentity "github.com/han139242/course-recommend-agents/backend/internal/module/graph/entity"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
)

// Service provides read-only graph query capabilities.
type Service struct {
	repo *graphrepo.Neo4jRepository
}

// New creates a graph service.
func New(repo *graphrepo.Neo4jRepository) *Service {
	return &Service{repo: repo}
}

// GetSubgraph loads a one-hop graph around a root node.
func (s *Service) GetSubgraph(ctx context.Context, query graphdto.SubgraphQuery) (*graphdto.SubgraphResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 12
	}

	nodes, edges, err := s.repo.LoadSubgraph(ctx, query.Root, limit)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildGraphResponse(nodes, edges), nil
}

// GetFullGraph loads a bounded graph snapshot for the homepage graph view.
func (s *Service) GetFullGraph(ctx context.Context, query graphdto.FullGraphQuery) (*graphdto.SubgraphResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 1000
	}

	nodes, edges, err := s.repo.LoadFullGraph(ctx, limit)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildGraphResponse(nodes, edges), nil
}

// GetCatalog loads lightweight graph metadata for search, navigation, and prerequisite lookup.
func (s *Service) GetCatalog(ctx context.Context) (*graphdto.CatalogResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	nodes, edges, err := s.repo.LoadCatalog(ctx)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildCatalogResponse(nodes, edges), nil
}

// GetSeeds loads seed nodes for the homepage graph canvas.
func (s *Service) GetSeeds(ctx context.Context, query graphdto.SeedQuery) (*graphdto.SubgraphResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	count := query.Count
	if count <= 0 {
		count = 14
	}

	nodes, edges, err := s.repo.LoadSeeds(ctx, count)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildGraphResponse(nodes, edges), nil
}

// GetNeighbors loads on-demand graph neighbors for a target node.
func (s *Service) GetNeighbors(ctx context.Context, query graphdto.NeighborQuery) (*graphdto.SubgraphResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	depth := query.Depth
	if depth <= 0 {
		depth = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	direction := query.Direction
	if direction == "" {
		direction = "both"
	}

	nodes, edges, err := s.repo.LoadNeighbors(ctx, query.NodeID, depth, direction, limit)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildGraphResponse(nodes, edges), nil
}

// GetPrerequisiteChain loads a bounded prerequisite chain for a target knowledge node.
func (s *Service) GetPrerequisiteChain(ctx context.Context, query graphdto.PrerequisiteChainQuery) (*graphdto.PrerequisiteChainResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	depth := query.Depth
	if depth <= 0 {
		depth = 3
	}

	nodes, edges, orderedNodeIDs, err := s.repo.LoadPrerequisiteChain(ctx, query.Target, depth)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildPrerequisiteChainResponse(strings.TrimSpace(query.Target), nodes, edges, orderedNodeIDs), nil
}

// FindPath loads a shortest graph path between two knowledge nodes.
func (s *Service) FindPath(ctx context.Context, query graphdto.GraphPathQuery) (*graphdto.GraphPathResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	maxDepth := query.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 5
	}

	nodes, edges, orderedNodeIDs, found, err := s.repo.FindShortestPath(ctx, query.From, query.To, maxDepth)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildGraphPathResponse(nodes, edges, orderedNodeIDs, found), nil
}

// GetKnowledgeDetail loads detail information for a knowledge node.
func (s *Service) GetKnowledgeDetail(ctx context.Context, knowledgeID string) (*graphdto.KnowledgeDetailView, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	detail, err := s.repo.LoadKnowledgeDetail(ctx, knowledgeID)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	if detail == nil {
		return nil, apperr.ErrNotFound
	}

	return &graphdto.KnowledgeDetailView{
		ID:               detail.ID,
		RouteKnowledgeID: detail.RouteKnowledgeID,
		Name:             detail.Name,
		CanonicalName:    detail.CanonicalName,
		EntityType:       detail.EntityType,
		Description:      detail.Description,
		ChapterTitle:     detail.ChapterTitle,
		SourcePath:       detail.SourcePath,
		CuratedSource:    detail.CuratedSource,
		Summary:          detail.Summary,
		Prerequisites:    detail.Prerequisites,
		Related:          detail.Related,
		Scenarios:        detail.Scenarios,
		ParentTopics:     detail.ParentTopics,
		MasteryScore:     0,
		Status:           "not_started",
		MasteryText:      "当前暂无学习记录。",
		RecommendReason:  "建议先从当前知识点的先修内容开始。",
	}, nil
}

func buildCatalogResponse(nodes []graphentity.Node, edges []graphentity.Edge) *graphdto.CatalogResponse {
	response := &graphdto.CatalogResponse{
		Nodes:             make([]graphdto.CatalogNodeView, 0, len(nodes)),
		PrerequisiteEdges: make([]graphdto.PrerequisiteIndexEdgeView, 0, len(edges)),
	}

	for _, node := range nodes {
		response.Nodes = append(response.Nodes, graphdto.CatalogNodeView{
			ID:               node.ID,
			RouteKnowledgeID: node.RouteKnowledgeID,
			Name:             node.Name,
			CanonicalName:    node.CanonicalName,
			ChapterTitle:     node.ChapterTitle,
			EntityType:       node.EntityType,
			SourcePath:       node.SourcePath,
			CuratedSource:    node.CuratedSource,
		})
	}

	for _, edge := range edges {
		if edge.SourceID == "" || edge.TargetID == "" {
			continue
		}
		response.PrerequisiteEdges = append(response.PrerequisiteEdges, graphdto.PrerequisiteIndexEdgeView{
			SourceRouteKnowledgeID: edge.SourceID,
			TargetRouteKnowledgeID: edge.TargetID,
		})
	}

	return response
}

func buildGraphResponse(nodes []graphentity.Node, edges []graphentity.Edge) *graphdto.SubgraphResponse {
	response := &graphdto.SubgraphResponse{
		Nodes: make([]graphdto.GraphNodeView, 0, len(nodes)),
		Edges: make([]graphdto.GraphEdgeView, 0, len(edges)),
	}

	for _, node := range nodes {
		response.Nodes = append(response.Nodes, graphdto.GraphNodeView{
			ID:               node.ID,
			RouteKnowledgeID: node.RouteKnowledgeID,
			Name:             node.Name,
			CanonicalName:    node.CanonicalName,
			EntityType:       node.EntityType,
			Description:      node.Description,
			ChapterTitle:     node.ChapterTitle,
			SourcePath:       node.SourcePath,
			CuratedSource:    node.CuratedSource,
		})
	}

	for _, edge := range edges {
		response.Edges = append(response.Edges, graphdto.GraphEdgeView{
			RelationID:   edge.RelationID,
			RelationType: edge.RelationType,
			SourceID:     edge.SourceID,
			TargetID:     edge.TargetID,
			SourceName:   edge.SourceName,
			TargetName:   edge.TargetName,
		})
	}

	return response
}

func buildPrerequisiteChainResponse(target string, nodes []graphentity.Node, edges []graphentity.Edge, orderedNodeIDs []string) *graphdto.PrerequisiteChainResponse {
	graph := buildGraphResponse(nodes, edges)
	return &graphdto.PrerequisiteChainResponse{
		Target:         target,
		Nodes:          graph.Nodes,
		Edges:          graph.Edges,
		OrderedNodeIDs: normalizeOrderedNodeIDs(orderedNodeIDs),
	}
}

func buildGraphPathResponse(nodes []graphentity.Node, edges []graphentity.Edge, orderedNodeIDs []string, found bool) *graphdto.GraphPathResponse {
	graph := buildGraphResponse(nodes, edges)
	return &graphdto.GraphPathResponse{
		Found:          found,
		Nodes:          graph.Nodes,
		Edges:          graph.Edges,
		OrderedNodeIDs: normalizeOrderedNodeIDs(orderedNodeIDs),
	}
}

func normalizeOrderedNodeIDs(items []string) []string {
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
