package repository

import (
	"context"

	learningmodel "github.com/han139242/course-recommend-agents/backend/internal/module/learning/model"
)

// StateRepository 定义学习状态落盘与读取接口。
type StateRepository interface {
	ListLearningProgress(ctx context.Context, studentID int64) ([]learningmodel.LearningProgress, error)
	UpsertLearningProgress(ctx context.Context, progress []learningmodel.LearningProgress) error
}
