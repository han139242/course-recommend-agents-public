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
