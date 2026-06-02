package service

import (
	"context"
	"strings"
	"time"

	frontierdto "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/dto"
	frontierrepo "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
)

// LiveSource 定义在线前沿资料来源。
type LiveSource interface {
	GetBriefing(ctx context.Context, knowledgeID string, knowledgeName string) (*frontierdto.BriefingResponse, error)
	ListLibrary(ctx context.Context, limit int) (*frontierdto.LibraryResponse, error)
}

// Service 提供前沿扩展业务能力。
type Service struct {
	repo       *frontierrepo.Repository
	liveSource LiveSource
}

// New 创建前沿扩展服务。
func New(repo *frontierrepo.Repository) *Service {
	return NewWithLiveSource(repo, NewLiveSource())
}

// NewWithLiveSource 创建可注入在线来源的前沿扩展服务。
func NewWithLiveSource(repo *frontierrepo.Repository, liveSource LiveSource) *Service {
	return &Service{repo: repo, liveSource: liveSource}
}

// GetBriefing 查询知识点前沿扩展摘要。
func (s *Service) GetBriefing(ctx context.Context, query frontierdto.BriefingQuery) (*frontierdto.BriefingResponse, error) {
	knowledgeID := strings.TrimSpace(query.KnowledgeID)
	knowledge := strings.TrimSpace(query.Knowledge)
	if knowledgeID == "" && knowledge == "" {
		return nil, apperr.ErrBadRequest
	}

	if s.repo == nil && s.liveSource == nil {
		return nil, apperr.ErrDependencyUnavailable
	}

	var response *frontierdto.BriefingResponse
	if s.repo != nil {
		repoResponse, err := s.repo.GetBriefing(ctx, knowledgeID, knowledge)
		if err != nil {
			return nil, apperr.ErrDependencyUnavailable.WithCause(err)
		}
		response = repoResponse
	}

	if !briefingReady(response) && s.liveSource != nil {
		liveResponse, err := s.liveSource.GetBriefing(ctx, knowledgeID, knowledge)
		if err == nil && liveResponse != nil {
			response = liveResponse
		}
	}

	response = sanitizeBriefingResponse(response, knowledgeID, knowledge)
	s.normalizeStatus(response)
	return response, nil
}

// ListLibrary 查询前沿资料中心列表。
func (s *Service) ListLibrary(ctx context.Context, query frontierdto.LibraryQuery) (*frontierdto.LibraryResponse, error) {
	if s.repo == nil && s.liveSource == nil {
		return nil, apperr.ErrDependencyUnavailable
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 30
	}

	var response *frontierdto.LibraryResponse
	if s.repo != nil {
		repoResponse, err := s.repo.ListLibrary(ctx, limit)
		if err != nil {
			return nil, apperr.ErrDependencyUnavailable.WithCause(err)
		}
		response = repoResponse
	}

	if !libraryReady(response) && s.liveSource != nil {
		liveResponse, err := s.liveSource.ListLibrary(ctx, limit)
		if err == nil && liveResponse != nil {
			response = liveResponse
		}
	}

	if response == nil {
		return &frontierdto.LibraryResponse{Status: "not_ready", Items: []frontierdto.LibraryItemView{}}, nil
	}
	response = sanitizeLibraryResponse(response)
	s.normalizeLibraryStatus(response)
	return response, nil
}

func briefingReady(response *frontierdto.BriefingResponse) bool {
	if response == nil {
		return false
	}
	if normalizeStatusValue(response.Status) != "ready" {
		return false
	}
	if len(response.Papers) > 0 {
		return trustedPapers(response.Papers)
	}
	return strings.TrimSpace(response.DigestSummary) != ""
}

func libraryReady(response *frontierdto.LibraryResponse) bool {
	if response == nil {
		return false
	}
	return normalizeStatusValue(response.Status) == "ready" && trustedLibraryItems(response.Items)
}

func normalizeStatusValue(status string) string {
	next := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(status, "-", "_")))
	if next == "" {
		return "not_ready"
	}
	return next
}

func trustedLibraryItems(items []frontierdto.LibraryItemView) bool {
	for _, item := range items {
		if trustedFrontierSource(item.Source, item.URL) {
			return true
		}
	}
	return false
}

func filterTrustedLibraryItems(items []frontierdto.LibraryItemView) []frontierdto.LibraryItemView {
	result := make([]frontierdto.LibraryItemView, 0, len(items))
	for _, item := range items {
		if trustedFrontierSource(item.Source, item.URL) {
			result = append(result, item)
		}
	}
	return result
}

func filterTrustedPapers(items []frontierdto.PaperView) []frontierdto.PaperView {
	result := make([]frontierdto.PaperView, 0, len(items))
	for _, item := range items {
		if trustedFrontierSource(item.Venue, item.URL) {
			result = append(result, item)
		}
	}
	return result
}

func trustedPapers(items []frontierdto.PaperView) bool {
	for _, item := range items {
		if trustedFrontierSource(item.Venue, item.URL) {
			return true
		}
	}
	return false
}

func sanitizeLibraryResponse(response *frontierdto.LibraryResponse) *frontierdto.LibraryResponse {
	if response == nil {
		return nil
	}
	next := *response
	next.Items = filterTrustedLibraryItems(response.Items)
	next.Total = len(next.Items)
	if len(next.Items) == 0 && normalizeStatusValue(next.Status) == "ready" {
		next.Status = "not_ready"
		next.Message = "当前图谱中的前沿资料仍是旧示例数据，已隐藏；系统会改用 arXiv、OpenAlex 和 GitHub 的真实公开来源。"
	}
	return &next
}

func sanitizeBriefingResponse(response *frontierdto.BriefingResponse, knowledgeID string, knowledge string) *frontierdto.BriefingResponse {
	if response == nil {
		return nil
	}
	next := *response
	next.Papers = filterTrustedPapers(response.Papers)
	if len(response.Papers) > 0 && len(next.Papers) == 0 && normalizeStatusValue(next.Status) == "ready" {
		next.Status = "not_ready"
		next.KnowledgeID = firstNonEmptyString(next.KnowledgeID, knowledgeID)
		next.Knowledge = firstNonEmptyString(next.Knowledge, knowledge)
		next.Message = "当前知识点的前沿资料仍是旧示例链接，已隐藏；请等待真实来源同步完成。"
		next.DigestSummary = ""
		next.Topics = []frontierdto.TopicView{}
	}
	return &next
}

func trustedFrontierSource(source string, rawURL string) bool {
	normalizedURL := strings.ToLower(strings.TrimSpace(rawURL))
	if normalizedURL == "" || strings.Contains(normalizedURL, "example.org") || strings.Contains(normalizedURL, "example.com") {
		return false
	}

	normalizedSource := strings.ToLower(strings.TrimSpace(source))
	if normalizedSource == "arxiv" || normalizedSource == "github" || normalizedSource == "openalex" {
		return true
	}

	trustedDomains := []string{
		"arxiv.org",
		"github.com",
		"openalex.org",
		"openreview.net",
		"doi.org",
		"aclanthology.org",
		"proceedings.mlr.press",
		"neurips.cc",
		"ieee.org",
		"dl.acm.org",
		"nature.com",
		"science.org",
		"cell.com",
	}
	for _, domain := range trustedDomains {
		if strings.Contains(normalizedURL, domain) {
			return true
		}
	}
	return false
}

func (s *Service) normalizeLibraryStatus(response *frontierdto.LibraryResponse) {
	if response == nil {
		return
	}
	status := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(response.Status, "-", "_")))
	if status == "" {
		status = "not_ready"
	}
	response.Status = status
	if response.Items == nil {
		response.Items = []frontierdto.LibraryItemView{}
	}
}

func (s *Service) normalizeStatus(response *frontierdto.BriefingResponse) {
	if response == nil {
		return
	}
	status := normalizeStatusValue(response.Status)
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

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
