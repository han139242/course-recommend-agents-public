package repository

import (
	"context"
	"errors"
	"strings"

	frontierdto "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/dto"
	graphentity "github.com/han139242/course-recommend-agents/backend/internal/module/graph/entity"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
)

// Repository 提供前沿扩展只读查询能力。
type Repository struct {
	graphRepo *graphrepo.Neo4jRepository
}

// New 创建前沿查询仓储。
func New(graphRepo *graphrepo.Neo4jRepository) *Repository {
	return &Repository{graphRepo: graphRepo}
}

// GetBriefing 查询知识点的前沿扩展信息。
func (r *Repository) GetBriefing(ctx context.Context, knowledgeID string, knowledgeName string) (*frontierdto.BriefingResponse, error) {
	if r.graphRepo == nil || !r.graphRepo.Enabled() {
		return notReadyResponse(knowledgeID, knowledgeName), nil
	}
	briefing, err := r.graphRepo.LoadFrontierBriefing(ctx, knowledgeID, knowledgeName)
	if err != nil {
		if errors.Is(err, graphrepo.ErrFrontierBriefingNotReady) {
			return notReadyResponse(knowledgeID, knowledgeName), nil
		}
		return nil, err
	}
	if briefing == nil {
		return notReadyResponse(knowledgeID, knowledgeName), nil
	}
	return &frontierdto.BriefingResponse{
		KnowledgeID:   briefing.KnowledgeID,
		Status:        briefing.Status,
		Knowledge:     briefing.Knowledge,
		Message:       briefing.Message,
		DigestSummary: briefing.DigestSummary,
		Topics:        toTopicViews(briefing.Topics),
		Papers:        toPaperViews(briefing.Papers),
		GeneratedAt:   briefing.GeneratedAt,
	}, nil
}

// ListLibrary 查询前沿资料中心聚合列表。
func (r *Repository) ListLibrary(ctx context.Context, limit int) (*frontierdto.LibraryResponse, error) {
	if r.graphRepo == nil || !r.graphRepo.Enabled() {
		return notReadyLibraryResponse(), nil
	}
	library, err := r.graphRepo.LoadFrontierLibrary(ctx, limit)
	if err != nil {
		if errors.Is(err, graphrepo.ErrFrontierBriefingNotReady) {
			return notReadyLibraryResponse(), nil
		}
		return nil, err
	}
	if library == nil {
		return notReadyLibraryResponse(), nil
	}
	return &frontierdto.LibraryResponse{
		Status:      "ready",
		Message:     "已从图谱载入前沿资料。",
		GeneratedAt: library.GeneratedAt,
		Total:       library.Total,
		Items:       toLibraryItemViews(library.Items),
	}, nil
}

func notReadyResponse(knowledgeID string, knowledgeName string) *frontierdto.BriefingResponse {
	return &frontierdto.BriefingResponse{
		KnowledgeID:   normalizeKnowledgeID(knowledgeID, knowledgeName),
		Status:        "not_ready",
		Knowledge:     normalizeKnowledgeName(knowledgeID, knowledgeName),
		Message:       "当前知识点的前沿内容尚未就绪，请稍后重试。",
		DigestSummary: "",
		Topics:        []frontierdto.TopicView{},
		Papers:        []frontierdto.PaperView{},
		GeneratedAt:   "",
	}
}

func notReadyLibraryResponse() *frontierdto.LibraryResponse {
	return &frontierdto.LibraryResponse{
		Status:      "not_ready",
		Message:     "当前还没有可展示的前沿资料，请先运行前沿同步链路。",
		GeneratedAt: "",
		Total:       0,
		Items:       []frontierdto.LibraryItemView{},
	}
}

func normalizeKnowledgeID(knowledgeID string, knowledgeName string) string {
	if strings.TrimSpace(knowledgeID) != "" {
		return strings.TrimSpace(knowledgeID)
	}
	return strings.TrimSpace(knowledgeName)
}

func normalizeKnowledgeName(knowledgeID string, knowledgeName string) string {
	if strings.TrimSpace(knowledgeName) != "" {
		return strings.TrimSpace(knowledgeName)
	}
	return strings.TrimSpace(knowledgeID)
}

func toTopicViews(items []graphentity.FrontierTopic) []frontierdto.TopicView {
	result := make([]frontierdto.TopicView, 0, len(items))
	for _, item := range items {
		result = append(result, frontierdto.TopicView{TopicID: item.TopicID, Name: item.Name, Reason: item.Reason})
	}
	return result
}

func toPaperViews(items []graphentity.FrontierPaper) []frontierdto.PaperView {
	result := make([]frontierdto.PaperView, 0, len(items))
	for _, item := range items {
		result = append(result, frontierdto.PaperView{PaperID: item.PaperID, Title: item.Title, Venue: item.Venue, Year: item.Year, URL: item.URL})
	}
	return result
}

func toKnowledgeRefViews(items []graphentity.FrontierKnowledgeRef) []frontierdto.KnowledgeRefView {
	result := make([]frontierdto.KnowledgeRefView, 0, len(items))
	for _, item := range items {
		result = append(result, frontierdto.KnowledgeRefView{KnowledgeID: item.KnowledgeID, Name: item.Name, Course: item.Course, Reason: item.Reason})
	}
	return result
}

func toLibraryItemViews(items []graphentity.FrontierLibraryItem) []frontierdto.LibraryItemView {
	result := make([]frontierdto.LibraryItemView, 0, len(items))
	for _, item := range items {
		result = append(result, frontierdto.LibraryItemView{
			PaperID:       item.PaperID,
			Title:         item.Title,
			Abstract:      item.Abstract,
			Venue:         item.Venue,
			Year:          item.Year,
			URL:           item.URL,
			Source:        item.Source,
			UpdatedAt:     item.UpdatedAt,
			Topics:        toTopicViews(item.Topics),
			KnowledgeRefs: toKnowledgeRefViews(item.KnowledgeRefs),
		})
	}
	return result
}
