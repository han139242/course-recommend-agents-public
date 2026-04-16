package service

import (
	"context"

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
