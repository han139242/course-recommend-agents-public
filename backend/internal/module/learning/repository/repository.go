package repository

import (
	"context"

	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
)

// Repository 定义学习模块仓储接口。
type Repository interface {
	BatchCreateQuestions(ctx context.Context, items []learningentity.ImportQuestionInput) (int, error)
	ListQuestions(ctx context.Context, filter learningentity.QuestionFilter) ([]learningentity.Question, int64, error)
	GetQuestionByQuestionID(ctx context.Context, questionID string) (*learningentity.Question, error)
	ListStudentQuestions(ctx context.Context, filter learningentity.StudentQuestionFilter) ([]learningentity.Question, int64, error)
	SubmitAnswer(ctx context.Context, input learningentity.SubmitAnswerInput) (*learningentity.SubmitAnswerResult, error)
	IsActiveStudent(ctx context.Context, userID int64) (bool, error)
	IsActiveTeacher(ctx context.Context, userID int64) (bool, error)
}
