package service

import (
	"context"

	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
)

// StateReader 定义学习状态读取能力，供其他模块按接口依赖。
type StateReader interface {
	GetStudentKnowledgeStatus(ctx context.Context, studentID int64, knowledgeID string, page int, pageSize int) (map[string]any, error)
	GetLearningProgress(ctx context.Context, studentID int64) (*learningdto.LearningProgressResponse, bool, error)
}
