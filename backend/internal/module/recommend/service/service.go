package service

import (
	"context"
	"encoding/json"
	"fmt"

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
	target := query.Target
	if target == "" {
		target = "default-target"
	}
	if s.redisRepo != nil && s.redisRepo.Enabled() {
		if cached, err := s.redisRepo.GetRecommendPath(ctx, studentID, target, query.Scene, limitValue(query.Limit)); err == nil && cached != "" {
			var response recommenddto.PathResponse
			if jsonErr := json.Unmarshal([]byte(cached), &response); jsonErr == nil {
				return &response, nil
			}
		}
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 5
	}
	path := make([]recommenddto.PathItemView, 0, limit)
	subgraph, graphErr := s.graphService.GetSubgraph(ctx, graphdto.SubgraphQuery{Root: target, Limit: limit})
	if graphErr == nil && subgraph != nil {
		for _, node := range subgraph.Nodes {
			if len(path) >= limit {
				break
			}
			path = append(path, recommenddto.PathItemView{Stage: stageFromScene(query.Scene), Title: node.Name, Detail: detailForNode(node.RouteKnowledgeID), Route: fmt.Sprintf("/knowledge/%s", node.RouteKnowledgeID), ReasonCode: "missing_prerequisite", ReasonLabel: "先修缺口", Score: scoreForIndex(len(path))})
		}
	}
	if len(path) == 0 {
		path = append(path, recommenddto.PathItemView{Stage: stageFromScene(query.Scene), Title: target, Detail: "建议优先补齐当前目标相关的先修知识点。", Route: fmt.Sprintf("/knowledge/%s", target), ReasonCode: "missing_prerequisite", ReasonLabel: "先修缺口", Score: 0.87})
	}
	response := &recommenddto.PathResponse{Path: path}
	if s.redisRepo != nil && s.redisRepo.Enabled() {
		if payload, err := json.Marshal(response); err == nil {
			_ = s.redisRepo.CacheRecommendPath(ctx, studentID, target, query.Scene, limitValue(limit), string(payload))
		}
	}
	return response, nil
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

func detailForNode(knowledgeID string) string {
	return fmt.Sprintf("建议先学习 %s，再进入后续更高难度内容。", knowledgeID)
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
