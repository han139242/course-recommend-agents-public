package service

import (
	"context"

	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
)

// Service 提供图谱只读查询能力。
type Service struct {
	repo *graphrepo.Neo4jRepository
}

// New 创建图谱服务。
func New(repo *graphrepo.Neo4jRepository) *Service {
	return &Service{repo: repo}
}

// GetSubgraph 查询子图。
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
	response := &graphdto.SubgraphResponse{Nodes: make([]graphdto.GraphNodeView, 0, len(nodes)), Edges: make([]graphdto.GraphEdgeView, 0, len(edges))}
	for _, node := range nodes {
		response.Nodes = append(response.Nodes, graphdto.GraphNodeView{ID: node.ID, RouteKnowledgeID: node.RouteKnowledgeID, Name: node.Name, CanonicalName: node.CanonicalName, EntityType: node.EntityType, Description: node.Description, ChapterTitle: node.ChapterTitle, SourcePath: node.SourcePath})
	}
	for _, edge := range edges {
		response.Edges = append(response.Edges, graphdto.GraphEdgeView{RelationID: edge.RelationID, RelationType: edge.RelationType, SourceID: edge.SourceID, TargetID: edge.TargetID, SourceName: edge.SourceName, TargetName: edge.TargetName})
	}
	return response, nil
}

// GetKnowledgeDetail 查询知识点详情。
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
	return &graphdto.KnowledgeDetailView{ID: detail.ID, RouteKnowledgeID: detail.RouteKnowledgeID, Name: detail.Name, CanonicalName: detail.CanonicalName, EntityType: detail.EntityType, Description: detail.Description, ChapterTitle: detail.ChapterTitle, SourcePath: detail.SourcePath, Summary: detail.Summary, Prerequisites: detail.Prerequisites, Related: detail.Related, Scenarios: detail.Scenarios, ParentTopics: detail.ParentTopics, MasteryScore: 0, Status: "not_started", MasteryText: "当前暂无学习记录。", RecommendReason: "建议先从当前知识点的先修内容开始。"}, nil
}
