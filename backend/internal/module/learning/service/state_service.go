package service

import (
	"context"
	"time"

	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	learningmodel "github.com/han139242/course-recommend-agents/backend/internal/module/learning/model"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
)

// StateService 提供学习状态缓存与快照能力。
type StateService struct {
	redisRepo *learningrepo.RedisRepository
	mysqlRepo learningrepo.StateRepository
}

// NewStateService 创建学习状态服务。
func NewStateService(redisRepo *learningrepo.RedisRepository, mysqlRepo learningrepo.StateRepository) *StateService {
	return &StateService{redisRepo: redisRepo, mysqlRepo: mysqlRepo}
}

// SnapshotMastery 增量更新掌握度并按需要回写快照。
func (s *StateService) SnapshotMastery(ctx context.Context, studentID int64, knowledgeID string, _ float64, answeredAt time.Time, correct bool) (bool, error) {
	if s.redisRepo != nil && s.redisRepo.Enabled() {
		if _, err := s.redisRepo.RecordAnswer(ctx, studentID, knowledgeID, correct, answeredAt); err == nil {
			return false, nil
		}
	}
	// Redis 不可用时只返回降级标记，不伪造学习进度快照，避免把本次增量错误写成完整状态。
	return true, nil
}

// GetLearningProgress 查询学习进度，优先读 Redis，失败时回源 MySQL。
func (s *StateService) GetLearningProgress(ctx context.Context, studentID int64) (*learningdto.LearningProgressResponse, bool, error) {
	redisFailed := false
	if s.redisRepo != nil && s.redisRepo.Enabled() {
		states, err := s.redisRepo.GetMasteryStates(ctx, studentID)
		if err == nil {
			if len(states) > 0 {
				items := make([]learningdto.LearningProgressItem, 0, len(states))
				for knowledgeID, state := range states {
					lastAnsweredAt := ""
					if state.LastAnsweredAt != nil {
						lastAnsweredAt = state.LastAnsweredAt.Format(time.RFC3339)
					}
					correctRate := 0.0
					if state.AnsweredCount > 0 {
						correctRate = float64(state.CorrectCount) / float64(state.AnsweredCount)
					}
					items = append(items, learningdto.LearningProgressItem{KnowledgeID: knowledgeID, CanonicalName: knowledgeID, MasteryScore: state.MasteryScore, Status: progressStatus(state.MasteryScore), CorrectRate: correctRate, AnsweredCount: state.AnsweredCount, LastAnsweredAt: lastAnsweredAt})
				}
				return &learningdto.LearningProgressResponse{List: items}, false, nil
			}
		} else {
			redisFailed = true
		}
	}
	if s.mysqlRepo == nil {
		return &learningdto.LearningProgressResponse{List: []learningdto.LearningProgressItem{}}, redisFailed, nil
	}
	items, err := s.mysqlRepo.ListLearningProgress(ctx, studentID)
	if err != nil {
		return nil, redisFailed, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	response := &learningdto.LearningProgressResponse{List: toLearningProgressItems(items)}
	return response, redisFailed, nil
}

// FlushStudentProgress 从 Redis 拉取学生掌握度向量并回写 MySQL 快照。
func (s *StateService) FlushStudentProgress(ctx context.Context, studentID int64) error {
	if s.redisRepo == nil || !s.redisRepo.Enabled() || s.mysqlRepo == nil {
		return nil
	}
	states, err := s.redisRepo.GetMasteryStates(ctx, studentID)
	if err != nil {
		return err
	}
	progress := make([]learningmodel.LearningProgress, 0, len(states))
	for knowledgeID, state := range states {
		correctRate := 0.0
		if state.AnsweredCount > 0 {
			correctRate = float64(state.CorrectCount) / float64(state.AnsweredCount)
		}
		progress = append(progress, learningmodel.LearningProgress{StudentID: studentID, KnowledgeID: knowledgeID, CanonicalName: knowledgeID, MasteryScore: state.MasteryScore, Status: progressStatus(state.MasteryScore), CorrectRate: correctRate, AnsweredCount: state.AnsweredCount, LastAnsweredAt: state.LastAnsweredAt})
	}
	return s.mysqlRepo.UpsertLearningProgress(ctx, progress)
}

func toLearningProgressItems(items []learningmodel.LearningProgress) []learningdto.LearningProgressItem {
	result := make([]learningdto.LearningProgressItem, 0, len(items))
	for _, item := range items {
		lastAnsweredAt := ""
		if item.LastAnsweredAt != nil {
			lastAnsweredAt = item.LastAnsweredAt.Format(time.RFC3339)
		}
		result = append(result, learningdto.LearningProgressItem{KnowledgeID: item.KnowledgeID, CanonicalName: item.CanonicalName, MasteryScore: item.MasteryScore, Status: item.Status, CorrectRate: item.CorrectRate, AnsweredCount: item.AnsweredCount, LastAnsweredAt: lastAnsweredAt})
	}
	return result
}

func progressStatus(score float64) string {
	if score >= 80 {
		return "solid"
	}
	if score > 0 {
		return "progressing"
	}
	return learningentity.LearningStatusNotStarted
}

func maxMastery(value float64, floor float64) float64 {
	if value < floor {
		return floor
	}
	return value
}
