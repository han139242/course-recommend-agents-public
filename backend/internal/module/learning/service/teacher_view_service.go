package service

import (
	"context"
	"errors"

	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
)

// GetStudentKnowledgeStatus 查询教师视角的学生知识状态。
func (s *StateService) GetStudentKnowledgeStatus(ctx context.Context, studentID int64, knowledgeID string, page int, pageSize int) (map[string]any, error) {
	response, _, err := s.GetLearningProgress(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if knowledgeID != "" {
		for _, item := range response.List {
			if item.KnowledgeID == knowledgeID {
				return map[string]any{
					"knowledge": map[string]any{"id": item.KnowledgeID, "canonicalName": item.CanonicalName},
					"masteryScore": item.MasteryScore,
					"status": item.Status,
					"correctRate": item.CorrectRate,
					"answeredCount": item.AnsweredCount,
					"lastAnsweredAt": item.LastAnsweredAt,
				}, nil
			}
		}
		return nil, apperr.ErrNotFound.WithCause(errors.New("knowledge status not found"))
	}
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start > len(response.List) {
		start = len(response.List)
	}
	end := start + pageSize
	if end > len(response.List) {
		end = len(response.List)
	}
	list := response.List[start:end]
	return map[string]any{"list": list, "total": len(response.List), "page": page, "pageSize": pageSize}, nil
}

var _ = learningdto.LearningProgressResponse{}
