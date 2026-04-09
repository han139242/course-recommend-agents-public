package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	competencydto "github.com/han139242/course-recommend-agents/backend/internal/module/competency/dto"
	competencyentity "github.com/han139242/course-recommend-agents/backend/internal/module/competency/entity"
	graphwriterentity "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/entity"
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

// Service 提供能力画像计算能力。
type Service struct {
	redisRepo     *learningrepo.RedisRepository
	matrixManager *MatrixManager
	graphWriter   *graphwriterservice.Service
}

// New 创建能力画像服务。
func New(redisRepo *learningrepo.RedisRepository, matrixManager *MatrixManager, graphWriter *graphwriterservice.Service) *Service {
	return &Service{redisRepo: redisRepo, matrixManager: matrixManager, graphWriter: graphWriter}
}

// GetProfile 查询当前学生能力画像。
func (s *Service) GetProfile(ctx context.Context, user *competencyentity.UserProfile) (*competencydto.ProfileResponse, error) {
	if user == nil {
		return nil, apperr.ErrUnauthorized
	}
	studentID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		studentID = user.ID
	}
	cachePayload, cacheErr := s.redisRepo.GetCompetencyProfile(ctx, studentID)
	if cacheErr == nil && cachePayload != "" {
		var cached competencydto.ProfileResponse
		if err := json.Unmarshal([]byte(cachePayload), &cached); err == nil {
			return &cached, nil
		}
	}
	response, err := s.BuildProfile(ctx, studentID, user)
	if err != nil {
		return nil, err
	}
	if payload, marshalErr := json.Marshal(response); marshalErr == nil {
		_ = s.redisRepo.CacheCompetencyProfile(ctx, studentID, string(payload))
	}
	return response, nil
}

// BuildProfile 根据当前 mastery 状态构建能力画像，并触发快照同步。
func (s *Service) BuildProfile(ctx context.Context, studentID int64, user *competencyentity.UserProfile) (*competencydto.ProfileResponse, error) {
	if user == nil {
		return nil, apperr.ErrUnauthorized
	}
	states, err := s.redisRepo.GetMasteryStates(ctx, studentID)
	if err != nil {
		states = map[string]learningentity.MasteryState{}
	}
	mastery := make(map[string]float64, len(states))
	heatmapRows := make([]competencydto.HeatmapRow, 0, len(states))
	knowledgeIDs := make([]string, 0, len(states))
	for knowledgeID := range states {
		knowledgeIDs = append(knowledgeIDs, knowledgeID)
	}
	sort.Strings(knowledgeIDs)
	for _, knowledgeID := range knowledgeIDs {
		state := states[knowledgeID]
		mastery[knowledgeID] = state.MasteryScore
		heatmapRows = append(heatmapRows, competencydto.HeatmapRow{Label: knowledgeID, Values: []float64{state.MasteryScore}})
	}
	scores := s.matrixManager.ComputeScores(mastery)
	axes := toRadarAxes(scores)
	response := &competencydto.ProfileResponse{
		Profile: competencydto.ProfileView{
			Name:      user.Name,
			Role:      user.Role,
			ClassID:   user.ClassID,
			ClassName: user.ClassName,
			Target:    user.Target,
			Summary:   "当前画像基于知识点掌握度与能力矩阵映射实时聚合生成。",
			Stats: []competencydto.StatItem{{Label: "学习中知识点", Value: fmt.Sprintf("%d", len(states))}, {Label: "能力维度数", Value: fmt.Sprintf("%d", len(scores))}},
			Goals: []string{"持续补齐当前薄弱知识点，逐步提升能力画像稳定性。"},
			RecentRecords: []string{"最近答题记录已纳入画像实时计算。"},
		},
		RadarAxes:          axes,
		GrowthInsights:     []competencydto.InsightItem{{Label: "能力成长", Value: averageAxis(axes), Note: "画像分数由掌握度向量与能力矩阵映射计算得出。"}},
		GrowthSummaryCards: []competencydto.SummaryCard{{Label: "已形成优势", Value: topAxisLabel(axes)}},
		HeatmapDomains:     []string{"知识点掌握度"},
		HeatmapRows:        heatmapRows,
	}
	s.emitProfileUpdate(ctx, studentID, *user, scores)
	return response, nil
}

func averageAxis(items []competencydto.RadarAxis) float64 {
	if len(items) == 0 {
		return 0
	}
	total := 0.0
	for _, item := range items {
		total += item.Value
	}
	return total / float64(len(items))
}

func topAxisLabel(items []competencydto.RadarAxis) string {
	if len(items) == 0 {
		return "暂无"
	}
	best := items[0]
	for _, item := range items[1:] {
		if item.Value > best.Value {
			best = item
		}
	}
	return best.Name
}

func toRadarAxes(scores []competencyentity.CompetencyScore) []competencydto.RadarAxis {
	result := make([]competencydto.RadarAxis, 0, len(scores))
	for _, score := range scores {
		result = append(result, competencydto.RadarAxis{Name: score.CompetencyName, Value: score.Score})
	}
	return result
}

func (s *Service) emitProfileUpdate(ctx context.Context, studentID int64, user competencyentity.UserProfile, scores []competencyentity.CompetencyScore) {
	if s.graphWriter == nil || !s.graphWriter.Enabled() || len(scores) == 0 {
		return
	}
	relations := make([]graphwriterentity.CompetencyRelation, 0, len(scores))
	for _, score := range scores {
		relations = append(relations, graphwriterentity.CompetencyRelation{CompetencyID: score.CompetencyID, CompetencyName: score.CompetencyName, Category: score.Category, Score: score.Score})
	}
	s.graphWriter.Enqueue(ctx, graphwriterentity.ProfileUpdateEvent{StudentID: studentID, Identity: graphwriterentity.ProfileIdentity{Name: user.Name, Target: user.Target}, UpdatedAt: time.Now().UTC(), TraceID: ctxmeta.TraceID(ctx), Source: "competency.profile", Relations: relations})
}
