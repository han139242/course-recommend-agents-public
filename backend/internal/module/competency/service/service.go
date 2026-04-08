package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	competencydto "github.com/han139242/course-recommend-agents/backend/internal/module/competency/dto"
	competencyentity "github.com/han139242/course-recommend-agents/backend/internal/module/competency/entity"
	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/constant"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

// Service 提供能力画像计算能力。
type Service struct {
	redisRepo *learningrepo.RedisRepository
}

// New 创建能力画像服务。
func New(redisRepo *learningrepo.RedisRepository) *Service {
	return &Service{redisRepo: redisRepo}
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
	states, err := s.redisRepo.GetMasteryStates(ctx, studentID)
	if err != nil {
		states = map[string]learningentity.MasteryState{}
	}
	axes := make([]competencydto.RadarAxis, 0, len(states))
	heatmapRows := make([]competencydto.HeatmapRow, 0, len(states))
	knowledgeIDs := make([]string, 0, len(states))
	for knowledgeID := range states {
		knowledgeIDs = append(knowledgeIDs, knowledgeID)
	}
	sort.Strings(knowledgeIDs)
	for _, knowledgeID := range knowledgeIDs {
		state := states[knowledgeID]
		axisScore := state.MasteryScore / constant.RedisCompetencyScale
		axes = append(axes, competencydto.RadarAxis{Name: domainLabel(knowledgeID), Value: axisScore})
		heatmapRows = append(heatmapRows, competencydto.HeatmapRow{Label: knowledgeID, Values: []float64{state.MasteryScore}})
	}
	response := &competencydto.ProfileResponse{
		Profile: competencydto.ProfileView{
			Name:      user.Name,
			Role:      user.Role,
			ClassID:   user.ClassID,
			ClassName: user.ClassName,
			Target:    user.Target,
			Summary:   "当前画像基于知识点掌握度与答题状态实时聚合生成。",
			Stats: []competencydto.StatItem{{Label: "学习中知识点", Value: fmt.Sprintf("%d", len(states))}},
			Goals: []string{"持续补齐当前薄弱知识点，逐步提升能力画像稳定性。"},
			RecentRecords: []string{"最近答题记录已纳入画像实时计算。"},
		},
		RadarAxes:          axes,
		GrowthInsights:     []competencydto.InsightItem{{Label: "能力成长", Value: averageAxis(axes), Note: "画像分数由掌握度向量与能力维度映射计算得出。"}},
		GrowthSummaryCards: []competencydto.SummaryCard{{Label: "已形成优势", Value: topAxisLabel(axes)}},
		HeatmapDomains:     []string{"综合能力"},
		HeatmapRows:        heatmapRows,
	}
	if payload, marshalErr := json.Marshal(response); marshalErr == nil {
		_ = s.redisRepo.CacheCompetencyProfile(ctx, studentID, string(payload))
	}
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

func domainLabel(knowledgeID string) string {
	return knowledgeID
}
