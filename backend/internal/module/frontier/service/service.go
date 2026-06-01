package service

import (
	"context"
	"strings"
	"time"

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
	s.normalizeStatus(response)
	return response, nil
}

// ListLibrary 查询前沿资料中心列表。
func (s *Service) ListLibrary(ctx context.Context, query frontierdto.LibraryQuery) (*frontierdto.LibraryResponse, error) {
	if s.repo == nil {
		return nil, apperr.ErrDependencyUnavailable
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 30
	}
	response, err := s.repo.ListLibrary(ctx, limit)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	if response == nil {
		return &frontierdto.LibraryResponse{Status: "not_ready", Items: []frontierdto.LibraryItemView{}}, nil
	}
	status := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(response.Status, "-", "_")))
	if status == "" {
		status = "not_ready"
	}
	response.Status = status
	if response.Items == nil {
		response.Items = []frontierdto.LibraryItemView{}
	}
	return response, nil
}

func (s *Service) normalizeStatus(response *frontierdto.BriefingResponse) {
	if response == nil {
		return
	}
	status := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(response.Status, "-", "_")))
	if status == "" {
		status = "not_ready"
	}
	switch status {
	case "ready":
		if response.GeneratedAt != "" {
			if generatedAt, err := time.Parse(time.RFC3339, response.GeneratedAt); err == nil && time.Since(generatedAt) > 30*24*time.Hour {
				status = "stale"
				if strings.TrimSpace(response.Message) == "" {
					response.Message = "当前摘要已过期，建议等待离线链路刷新。"
				}
			}
		}
	case "failed":
		if strings.TrimSpace(response.Message) == "" {
			response.Message = "前沿内容生成失败，请稍后重试。"
		}
	case "stale", "not_ready":
		// 保留原始业务含义
	default:
		status = "not_ready"
	}
	response.Status = status
}
