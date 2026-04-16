package service

import (
	"context"
	"strings"
	"time"

	learningmodel "github.com/han139242/course-recommend-agents/backend/internal/module/learning/model"
)

type recentAnswerRow struct {
	QuestionCode   string    `gorm:"column:question_code"`
	KnowledgeID    string    `gorm:"column:knowledge_id"`
	Stem           string    `gorm:"column:stem"`
	SelectedOption string    `gorm:"column:selected_option"`
	IsCorrect      bool      `gorm:"column:is_correct"`
	DurationMs     *int      `gorm:"column:duration_ms"`
	AnsweredAt     time.Time `gorm:"column:answered_at"`
}

type recentAnswerView struct {
	QuestionCode   string `json:"questionCode"`
	KnowledgeID    string `json:"knowledgeId"`
	Stem           string `json:"stem"`
	SelectedOption string `json:"selectedOption"`
	IsCorrect      bool   `json:"isCorrect"`
	DurationMs     *int   `json:"durationMs"`
	AnsweredAt     string `json:"answeredAt"`
}

func (s *Service) queryRecentAnswers(ctx context.Context, studentID int64, knowledgeID string, limit int, onlyWrong bool) ([]recentAnswerView, error) {
	sql := `
SELECT sa.question_code, sa.knowledge_id, q.stem, sa.selected_option, sa.is_correct, sa.duration_ms, sa.answered_at
FROM student_answers sa
LEFT JOIN questions q ON q.question_code = sa.question_code
WHERE sa.student_id = ?
`
	args := []any{studentID}
	if strings.TrimSpace(knowledgeID) != "" {
		sql += " AND sa.knowledge_id = ?"
		args = append(args, strings.TrimSpace(knowledgeID))
	}
	if onlyWrong {
		sql += " AND sa.is_correct = ?"
		args = append(args, false)
	}
	sql += " ORDER BY sa.answered_at DESC LIMIT ?"
	args = append(args, limit)

	var rows []recentAnswerRow
	if err := s.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]recentAnswerView, 0, len(rows))
	for _, row := range rows {
		result = append(result, recentAnswerView{
			QuestionCode:   row.QuestionCode,
			KnowledgeID:    row.KnowledgeID,
			Stem:           row.Stem,
			SelectedOption: row.SelectedOption,
			IsCorrect:      row.IsCorrect,
			DurationMs:     row.DurationMs,
			AnsweredAt:     row.AnsweredAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

type learningProgressView struct {
	KnowledgeID    string  `json:"knowledgeId"`
	CanonicalName  string  `json:"canonicalName"`
	MasteryScore   float64 `json:"masteryScore"`
	CorrectRate    float64 `json:"correctRate"`
	AnsweredCount  int     `json:"answeredCount"`
	LastAnsweredAt string  `json:"lastAnsweredAt"`
	Status         string  `json:"status"`
}

func (s *Service) queryKnowledgeRanking(ctx context.Context, studentID int64, limit int, weakest bool) ([]learningProgressView, error) {
	query := s.db.WithContext(ctx).
		Model(&learningmodel.LearningProgress{}).
		Where("student_id = ?", studentID)

	order := "mastery_score DESC, answered_count DESC, updated_at DESC"
	if weakest {
		order = "mastery_score ASC, answered_count DESC, updated_at DESC"
	}

	var rows []learningmodel.LearningProgress
	if err := query.Order(order).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]learningProgressView, 0, len(rows))
	for _, row := range rows {
		lastAnsweredAt := ""
		if row.LastAnsweredAt != nil {
			lastAnsweredAt = row.LastAnsweredAt.Format(time.RFC3339)
		}
		result = append(result, learningProgressView{
			KnowledgeID:    row.KnowledgeID,
			CanonicalName:  row.CanonicalName,
			MasteryScore:   row.MasteryScore,
			CorrectRate:    row.CorrectRate,
			AnsweredCount:  row.AnsweredCount,
			LastAnsweredAt: lastAnsweredAt,
			Status:         row.Status,
		})
	}
	return result, nil
}

type answerStatsRow struct {
	TotalAnswers      int64      `gorm:"column:total_answers"`
	CorrectAnswers    int64      `gorm:"column:correct_answers"`
	WrongAnswers      int64      `gorm:"column:wrong_answers"`
	AverageDurationMs float64    `gorm:"column:average_duration_ms"`
	LastAnsweredAt    *time.Time `gorm:"column:last_answered_at"`
}

func (s *Service) queryAnswerStats(ctx context.Context, studentID int64, knowledgeID string) (map[string]any, error) {
	sql := `
SELECT
  COUNT(*) AS total_answers,
  SUM(CASE WHEN is_correct THEN 1 ELSE 0 END) AS correct_answers,
  SUM(CASE WHEN is_correct THEN 0 ELSE 1 END) AS wrong_answers,
  COALESCE(AVG(duration_ms), 0) AS average_duration_ms,
  MAX(answered_at) AS last_answered_at
FROM student_answers
WHERE student_id = ?
`
	args := []any{studentID}
	if strings.TrimSpace(knowledgeID) != "" {
		sql += " AND knowledge_id = ?"
		args = append(args, strings.TrimSpace(knowledgeID))
	}

	var row answerStatsRow
	if err := s.db.WithContext(ctx).Raw(sql, args...).Scan(&row).Error; err != nil {
		return nil, err
	}

	lastAnsweredAt := ""
	if row.LastAnsweredAt != nil {
		lastAnsweredAt = row.LastAnsweredAt.Format(time.RFC3339)
	}

	correctRate := 0.0
	if row.TotalAnswers > 0 {
		correctRate = float64(row.CorrectAnswers) / float64(row.TotalAnswers)
	}

	return map[string]any{
		"knowledgeId":       strings.TrimSpace(knowledgeID),
		"totalAnswers":      row.TotalAnswers,
		"correctAnswers":    row.CorrectAnswers,
		"wrongAnswers":      row.WrongAnswers,
		"correctRate":       correctRate,
		"averageDurationMs": row.AverageDurationMs,
		"lastAnsweredAt":    lastAnsweredAt,
	}, nil
}
