package service

import (
	"context"
	"strings"

	frontierdto "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/dto"
	frontierrepo "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
)

// Service 提供前沿扩展业务能力。
type Service struct {
	repo *frontierrepo.Repository
}

// New 创建前沿扩展服务。
func New(repo *frontierrepo.Repository) *Service {
	return &Service{repo: repo}
}

// GetBriefing 查询知识点前沿扩展摘要。
func (s *Service) GetBriefing(ctx context.Context, query frontierdto.BriefingQuery) (*frontierdto.BriefingResponse, error) {
	knowledgeID := strings.TrimSpace(query.KnowledgeID)
	knowledge := strings.TrimSpace(query.Knowledge)
	if knowledgeID == "" && knowledge == "" {
		return nil, apperr.ErrBadRequest
	}
	if s.repo == nil {
		return nil, apperr.ErrDependencyUnavailable
	}
	response, err := s.repo.GetBriefing(ctx, knowledgeID, knowledge)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	return response, nil
}
