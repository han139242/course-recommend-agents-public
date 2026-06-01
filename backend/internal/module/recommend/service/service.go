package service

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"

	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
)

// Service provides personalized recommendation abilities.
type Service struct {
	graphService *graphservice.Service
	redisRepo    *learningrepo.RedisRepository
	stateRepo    learningrepo.StateRepository
	Scorer       *Scorer
}

type StudentKnowledgeState struct {
	KnowledgeID    string
	CanonicalName  string
	MasteryScore   float64
	CorrectRate    float64
	AnsweredCount  int
	LastAnsweredAt string
	Status         string
}

type scoredCandidate struct {
	Node          graphdto.GraphNodeView
	KnowledgeID   string
	Title         string
	ReasonCode    string
	ReasonLabel   string
	SourceCourse  string
	TargetCourse  string
	ReasonDetail  string
	MasteryScore  float64
	CorrectRate   float64
	AnsweredCount int
	Score         float64
}

// New creates a recommend service.
func New(graphService *graphservice.Service, redisRepo *learningrepo.RedisRepository, stateRepo learningrepo.StateRepository, scorer *Scorer) *Service {
	return &Service{graphService: graphService, redisRepo: redisRepo, stateRepo: stateRepo, Scorer: scorer}
}

// GetPath returns personalized recommendation path items.
func (s *Service) GetPath(ctx context.Context, query recommenddto.PathQuery) (*recommenddto.PathResponse, error) {
	studentID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}

	target := strings.TrimSpace(query.Target)
	knowledgeID := strings.TrimSpace(query.KnowledgeID)
	limit := limitValue(query.Limit)
	cacheTarget := normalizeRecommendCacheTarget(target, knowledgeID)

	if s.shouldUseCache(query) && s.redisRepo != nil && s.redisRepo.Enabled() {
		if cached, err := s.redisRepo.GetRecommendPath(ctx, studentID, cacheTarget, query.Scene, limit); err == nil && cached != "" {
			var response recommenddto.PathResponse
			if jsonErr := json.Unmarshal([]byte(cached), &response); jsonErr == nil && isUsableRecommendPath(&response) {
				return &response, nil
			}
		}
	}

	progressList, stateLookup := s.loadStudentStates(ctx, studentID)

	path := s.buildPersonalizedPath(ctx, query, progressList, stateLookup, limit)
	if len(path) == 0 {
		root := knowledgeID
		if root == "" {
			root = target
		}
		path = s.buildGraphDrivenPath(ctx, root, query.Scene, limit, stateLookup)
	}
	if len(path) == 0 {
		path = fallbackRecommendPath(query.Scene, limit)
	}

	response := &recommenddto.PathResponse{
		Path:           path,
		Insights:       s.DetectLearningBottlenecks(ctx, query, progressList, stateLookup, path),
		LearnerProfile: inferLearnerProfile(progressList),
	}
	if s.shouldUseCache(query) && s.redisRepo != nil && s.redisRepo.Enabled() {
		if payload, err := json.Marshal(response); err == nil {
			_ = s.redisRepo.CacheRecommendPath(ctx, studentID, cacheTarget, query.Scene, limit, string(payload))
		}
	}
	return response, nil
}

// GetUnifiedPath uses KT-KGPath unified scorer for recommendations.
func (s *Service) GetUnifiedPath(ctx context.Context, query recommenddto.PathQuery) (*recommenddto.PathResponse, error) {
	studentID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}

	target := strings.TrimSpace(query.Target)
	knowledgeID := strings.TrimSpace(query.KnowledgeID)
	limit := limitValue(query.Limit)

	progressList, stateLookup := s.loadStudentStates(ctx, studentID)

	candidates := s.collectCandidates(ctx, knowledgeID, target, progressList, stateLookup, limit*4)
	if len(candidates) == 0 {
		return &recommenddto.PathResponse{Path: fallbackRecommendPath(query.Scene, limit)}, nil
	}

	type scoredItem struct {
		item    recommenddto.PathItemView
		factors ScoringFactors
	}
	scored := make([]scoredItem, 0, len(candidates))
	for _, c := range candidates {
		input := ScoreInput{
			Candidate:       c.state,
			KnowledgeID:     c.knowledgeID,
			KnowledgeName:   c.title,
			StateLookup:     stateLookup,
			TargetKnowledge: target,
			SourceCourse:    c.sourceCourse,
			TargetCourse:    c.targetCourse,
			RelationType:    c.relationType,
			SourceMastery:   c.sourceMastery,
		}
		totalScore, factors := s.Scorer.ScoreCandidate(ctx, input)
		c.item.Score = totalScore
		c.item.ScoringBreakdown = scoringBreakdownToView(totalScore, factors)
		scored = append(scored, scoredItem{item: c.item, factors: factors})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if math.Abs(scored[i].item.Score-scored[j].item.Score) < 0.001 {
			return scored[i].item.Title < scored[j].item.Title
		}
		return scored[i].item.Score > scored[j].item.Score
	})

	path := make([]recommenddto.PathItemView, 0, limit)
	seen := make(map[string]struct{}, limit)
	for _, si := range scored {
		key := strings.TrimSpace(si.item.KnowledgeID)
		if key == "" {
			key = strings.TrimSpace(si.item.Route)
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if si.item.MasteryScore >= 85 && si.item.CorrectRate >= 0.85 && si.item.AnsweredCount > 0 {
			continue
		}
		path = append(path, si.item)
		if len(path) >= limit {
			break
		}
	}

	return &recommenddto.PathResponse{
		Path:           path,
		Insights:       s.DetectLearningBottlenecks(ctx, query, progressList, stateLookup, path),
		LearnerProfile: inferLearnerProfile(progressList),
	}, nil
}

func (s *Service) shouldUseCache(query recommenddto.PathQuery) bool {
	if strings.TrimSpace(query.Scene) == "detail" {
		return false
	}
	return strings.TrimSpace(query.KnowledgeID) == ""
}

func (s *Service) loadStudentStates(ctx context.Context, studentID int64) ([]StudentKnowledgeState, map[string]StudentKnowledgeState) {
	if s.redisRepo != nil && s.redisRepo.Enabled() {
		if states, err := s.redisRepo.GetMasteryStates(ctx, studentID); err == nil && len(states) > 0 {
			result := make([]StudentKnowledgeState, 0, len(states))
			lookup := make(map[string]StudentKnowledgeState, len(states)*2)
			for knowledgeID, state := range states {
				correctRate := 0.0
				if state.AnsweredCount > 0 {
					correctRate = float64(state.CorrectCount) / float64(state.AnsweredCount)
				}
				lastAnsweredAt := ""
				if state.LastAnsweredAt != nil {
					lastAnsweredAt = state.LastAnsweredAt.Format(time.RFC3339)
				}
				item := StudentKnowledgeState{
					KnowledgeID:    strings.TrimSpace(knowledgeID),
					CanonicalName:  strings.TrimSpace(knowledgeID),
					MasteryScore:   state.MasteryScore,
					CorrectRate:    correctRate,
					AnsweredCount:  state.AnsweredCount,
					LastAnsweredAt: lastAnsweredAt,
					Status:         progressStatus(state.MasteryScore),
				}
				result = append(result, item)
				registerStudentState(lookup, item)
			}
			return result, lookup
		}
	}

	if s.stateRepo == nil {
		return []StudentKnowledgeState{}, map[string]StudentKnowledgeState{}
	}

	rows, err := s.stateRepo.ListLearningProgress(ctx, studentID)
	if err != nil {
		return []StudentKnowledgeState{}, map[string]StudentKnowledgeState{}
	}

	result := make([]StudentKnowledgeState, 0, len(rows))
	lookup := make(map[string]StudentKnowledgeState, len(rows)*2)
	for _, row := range rows {
		lastAnsweredAt := ""
		if row.LastAnsweredAt != nil {
			lastAnsweredAt = row.LastAnsweredAt.Format(time.RFC3339)
		}
		item := StudentKnowledgeState{
			KnowledgeID:    strings.TrimSpace(row.KnowledgeID),
			CanonicalName:  strings.TrimSpace(row.CanonicalName),
			MasteryScore:   row.MasteryScore,
			CorrectRate:    row.CorrectRate,
			AnsweredCount:  row.AnsweredCount,
			LastAnsweredAt: lastAnsweredAt,
			Status:         strings.TrimSpace(row.Status),
		}
		result = append(result, item)
		registerStudentState(lookup, item)
	}
	return result, lookup
}

func registerStudentState(lookup map[string]StudentKnowledgeState, item StudentKnowledgeState) {
	if lookup == nil {
		return
	}
	keys := []string{
		normalizeStateKey(item.KnowledgeID),
		normalizeStateKey(item.CanonicalName),
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		lookup[key] = item
	}
}

func normalizeStateKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func progressStatus(score float64) string {
	if score >= 80 {
		return "solid"
	}
	if score > 0 {
		return "progressing"
	}
	return "not_started"
}
