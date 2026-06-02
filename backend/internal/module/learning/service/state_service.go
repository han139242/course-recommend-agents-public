package service

import (
	"context"
	"math"
	"time"

	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
	learningmodel "github.com/han139242/course-recommend-agents/backend/internal/module/learning/model"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
)

// StateService 提供学习状态缓存与快照能力。
type StateService struct {
	redisRepo            *learningrepo.RedisRepository
	mysqlRepo            learningrepo.StateRepository
	prerequisiteProvider PrerequisiteProvider
}

// PrerequisiteProvider 提供课程先修索引，供掌握度关系约束使用。
type PrerequisiteProvider interface {
	GetCatalog(ctx context.Context) (*graphdto.CatalogResponse, error)
}

// NewStateService 创建学习状态服务。
func NewStateService(redisRepo *learningrepo.RedisRepository, mysqlRepo learningrepo.StateRepository, providers ...PrerequisiteProvider) *StateService {
	var provider PrerequisiteProvider
	if len(providers) > 0 {
		provider = providers[0]
	}
	return &StateService{redisRepo: redisRepo, mysqlRepo: mysqlRepo, prerequisiteProvider: provider}
}

// SetPrerequisiteProvider 注入课程先修关系来源，用于掌握度的关系约束。
func (s *StateService) SetPrerequisiteProvider(provider PrerequisiteProvider) {
	if s == nil {
		return
	}
	s.prerequisiteProvider = provider
}

// SnapshotMastery 增量更新掌握度并按需要回写快照。
func (s *StateService) SnapshotMastery(ctx context.Context, studentID int64, knowledgeID string, _ float64, answeredAt time.Time, correct bool) (learningentity.MasteryState, bool, error) {
	if s.redisRepo != nil && s.redisRepo.Enabled() {
		state, err := s.redisRepo.RecordAnswer(ctx, studentID, knowledgeID, correct, answeredAt)
		if err == nil {
			return state, false, nil
		}
	}
	// Redis 不可用时只返回降级标记，不伪造学习进度快照，避免把本次增量错误写成完整状态。
	return learningentity.MasteryState{KnowledgeID: knowledgeID}, true, nil
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
				return &learningdto.LearningProgressResponse{List: s.applyPrerequisiteAdjustments(ctx, items)}, false, nil
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
	response := &learningdto.LearningProgressResponse{List: s.applyPrerequisiteAdjustments(ctx, toLearningProgressItems(items))}
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
	progress = s.applyPrerequisiteAdjustmentsToModels(ctx, progress)
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

func (s *StateService) applyPrerequisiteAdjustments(ctx context.Context, items []learningdto.LearningProgressItem) []learningdto.LearningProgressItem {
	if len(items) == 0 || s == nil || s.prerequisiteProvider == nil {
		return items
	}
	catalog, err := s.prerequisiteProvider.GetCatalog(ctx)
	if err != nil || catalog == nil || len(catalog.PrerequisiteEdges) == 0 {
		return items
	}
	byID := make(map[string]int, len(items))
	for index, item := range items {
		if item.KnowledgeID != "" {
			byID[item.KnowledgeID] = index
		}
	}
	parentScores := make(map[string][]float64)
	for _, edge := range catalog.PrerequisiteEdges {
		sourceID := edge.SourceRouteKnowledgeID
		targetID := edge.TargetRouteKnowledgeID
		if targetID == "" {
			continue
		}
		if _, ok := byID[targetID]; !ok {
			continue
		}
		score := 0.0
		if sourceIndex, ok := byID[sourceID]; ok {
			score = items[sourceIndex].MasteryScore
		}
		parentScores[targetID] = append(parentScores[targetID], score)
	}
	if len(parentScores) == 0 {
		return items
	}
	next := make([]learningdto.LearningProgressItem, len(items))
	copy(next, items)
	for knowledgeID, scores := range parentScores {
		index, ok := byID[knowledgeID]
		if !ok {
			continue
		}
		adjusted := relationAdjustedMastery(next[index].MasteryScore, scores)
		if adjusted < next[index].MasteryScore {
			next[index].MasteryScore = adjusted
			next[index].Status = progressStatus(adjusted)
		}
	}
	return next
}

func (s *StateService) applyPrerequisiteAdjustmentsToModels(ctx context.Context, items []learningmodel.LearningProgress) []learningmodel.LearningProgress {
	if len(items) == 0 {
		return items
	}
	dtoItems := make([]learningdto.LearningProgressItem, 0, len(items))
	for _, item := range items {
		dtoItems = append(dtoItems, learningdto.LearningProgressItem{
			KnowledgeID:   item.KnowledgeID,
			CanonicalName: item.CanonicalName,
			MasteryScore:  item.MasteryScore,
			Status:        item.Status,
			CorrectRate:   item.CorrectRate,
			AnsweredCount: item.AnsweredCount,
		})
	}
	adjustedItems := s.applyPrerequisiteAdjustments(ctx, dtoItems)
	adjustedByID := make(map[string]learningdto.LearningProgressItem, len(adjustedItems))
	for _, item := range adjustedItems {
		adjustedByID[item.KnowledgeID] = item
	}
	next := make([]learningmodel.LearningProgress, len(items))
	copy(next, items)
	for index, item := range next {
		if adjusted, ok := adjustedByID[item.KnowledgeID]; ok {
			next[index].MasteryScore = adjusted.MasteryScore
			next[index].Status = adjusted.Status
		}
	}
	return next
}

func relationAdjustedMastery(base float64, prerequisiteScores []float64) float64 {
	if len(prerequisiteScores) == 0 {
		return clampMastery(base)
	}
	total := 0.0
	minScore := 100.0
	for _, score := range prerequisiteScores {
		normalized := clampMastery(score)
		total += normalized
		if normalized < minScore {
			minScore = normalized
		}
	}
	average := total / float64(len(prerequisiteScores))
	capScore := 100.0
	switch {
	case minScore < 30:
		capScore = 55
	case average < 50:
		capScore = 68
	case average < 70:
		capScore = 82
	}
	return roundMastery(math.Min(clampMastery(base), capScore))
}

func clampMastery(value float64) float64 {
	return math.Max(0, math.Min(100, value))
}

func roundMastery(value float64) float64 {
	return math.Round(value*100) / 100
}

// GetMasteryStates 读取当前学生掌握度状态，供异步图谱写入使用。
func (s *StateService) GetMasteryStates(ctx context.Context, studentID int64) (map[string]learningentity.MasteryState, error) {
	if s.redisRepo == nil || !s.redisRepo.Enabled() {
		return map[string]learningentity.MasteryState{}, nil
	}
	states, err := s.redisRepo.GetMasteryStates(ctx, studentID)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	return states, nil
}

func maxMastery(value float64, floor float64) float64 {
	if value < floor {
		return floor
	}
	return value
}
