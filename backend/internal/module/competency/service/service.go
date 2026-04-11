package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	competencydto "github.com/han139242/course-recommend-agents/backend/internal/module/competency/dto"
	competencyentity "github.com/han139242/course-recommend-agents/backend/internal/module/competency/entity"
	graphentity "github.com/han139242/course-recommend-agents/backend/internal/module/graph/entity"
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
		if err := json.Unmarshal([]byte(cachePayload), &cached); err == nil && profileCacheUsable(&cached) {
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
	knowledgeIDs := make([]string, 0, len(states))
	for knowledgeID := range states {
		knowledgeIDs = append(knowledgeIDs, knowledgeID)
	}
	sort.Strings(knowledgeIDs)
	for _, knowledgeID := range knowledgeIDs {
		state := states[knowledgeID]
		mastery[knowledgeID] = state.MasteryScore
	}

	resolvedStates := s.resolveKnowledgeStates(ctx, mastery)
	scores := s.matrixManager.ComputeScores(mastery)
	if len(scores) == 0 {
		scores = deriveScoresFromKnowledge(resolvedStates)
	}
	axes := toRadarAxes(scores)
	heatmapDomains, heatmapRows := buildKnowledgeHeatmap(resolvedStates, axes)
	growthInsights := buildGrowthInsights(axes)
	growthSummaryCards := buildGrowthSummaryCards(axes)
	response := &competencydto.ProfileResponse{
		Profile: competencydto.ProfileView{
			Name:      user.Name,
			Role:      user.Role,
			ClassID:   user.ClassID,
			ClassName: user.ClassName,
			Target:    user.Target,
			Summary:   "当前画像优先基于知识图谱能力映射生成；若映射尚未补齐，则退回到知识路径聚合结果。",
			Stats: []competencydto.StatItem{{Label: "学习中知识点", Value: fmt.Sprintf("%d", len(states))}, {Label: "能力维度数", Value: fmt.Sprintf("%d", len(scores))}},
			Goals: []string{"持续补齐当前薄弱知识点，逐步提升能力画像稳定性。"},
			RecentRecords: []string{"最近答题记录已纳入画像实时计算。"},
		},
		RadarAxes:          axes,
		GrowthInsights:     growthInsights,
		GrowthSummaryCards: growthSummaryCards,
		HeatmapDomains:     heatmapDomains,
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
		result = append(result, competencydto.RadarAxis{Name: score.CompetencyName, Value: score.Score * 100})
	}
	return result
}

type resolvedKnowledgeState struct {
	KnowledgeID string
	Label       string
	Domain      string
	Mastery     float64
}

func (s *Service) resolveKnowledgeStates(ctx context.Context, mastery map[string]float64) []resolvedKnowledgeState {
	result := make([]resolvedKnowledgeState, 0, len(mastery))
	knowledgeIDs := make([]string, 0, len(mastery))
	for knowledgeID := range mastery {
		knowledgeIDs = append(knowledgeIDs, knowledgeID)
	}
	sort.Strings(knowledgeIDs)

	for _, knowledgeID := range knowledgeIDs {
		item := resolvedKnowledgeState{
			KnowledgeID: knowledgeID,
			Label:       knowledgeID,
			Domain:      "知识路径",
			Mastery:     mastery[knowledgeID],
		}

		if s.matrixManager != nil && s.matrixManager.graphRepo != nil && s.matrixManager.graphRepo.Enabled() {
			if detail, err := s.matrixManager.graphRepo.LoadKnowledgeDetail(ctx, knowledgeID); err == nil && detail != nil {
				if name := strings.TrimSpace(detail.Name); name != "" {
					item.Label = name
				}
				if domain := deriveKnowledgeDomain(detail); domain != "" {
					item.Domain = domain
				}
			}
		}

		result = append(result, item)
	}

	return result
}

func deriveKnowledgeDomain(detail *graphentity.KnowledgeDetail) string {
	if detail == nil {
		return "知识路径"
	}
	for _, topic := range detail.ParentTopics {
		if trimmed := strings.TrimSpace(topic); trimmed != "" {
			return trimmed
		}
	}
	if chapter := strings.TrimSpace(detail.ChapterTitle); chapter != "" {
		return chapter
	}
	switch strings.TrimSpace(detail.EntityType) {
	case "algorithm":
		return "算法能力"
	case "method":
		return "方法理解"
	case "application":
		return "场景应用"
	case "topic":
		return "主题主线"
	default:
		return "知识路径"
	}
}

func deriveScoresFromKnowledge(items []resolvedKnowledgeState) []competencyentity.CompetencyScore {
	type domainAggregate struct {
		Total float64
		Count float64
	}

	aggregated := make(map[string]domainAggregate)
	for _, item := range items {
		domain := strings.TrimSpace(item.Domain)
		if domain == "" {
			domain = "知识路径"
		}
		current := aggregated[domain]
		current.Total += item.Mastery
		current.Count++
		aggregated[domain] = current
	}

	result := make([]competencyentity.CompetencyScore, 0, len(aggregated))
	for domain, aggregate := range aggregated {
		if aggregate.Count <= 0 {
			continue
		}
		result = append(result, competencyentity.CompetencyScore{
			CompetencyID:   domain,
			CompetencyName: domain,
			Category:       "knowledge_path",
			Score:          (aggregate.Total / aggregate.Count) / 100.0,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].CompetencyName < result[j].CompetencyName
		}
		return result[i].Score > result[j].Score
	})
	return result
}

func buildKnowledgeHeatmap(items []resolvedKnowledgeState, axes []competencydto.RadarAxis) ([]string, []competencydto.HeatmapRow) {
	if len(items) == 0 {
		return []string{"掌握度"}, []competencydto.HeatmapRow{}
	}

	domainOrder := make([]string, 0, len(axes))
	for _, axis := range axes {
		if strings.TrimSpace(axis.Name) != "" {
			domainOrder = append(domainOrder, axis.Name)
		}
	}
	if len(domainOrder) == 0 {
		domainSeen := make(map[string]struct{})
		for _, item := range items {
			domain := strings.TrimSpace(item.Domain)
			if domain == "" {
				domain = "知识路径"
			}
			if _, exists := domainSeen[domain]; exists {
				continue
			}
			domainSeen[domain] = struct{}{}
			domainOrder = append(domainOrder, domain)
		}
	}

	rows := make([]competencydto.HeatmapRow, 0, len(items))
	for _, item := range items {
		values := make([]float64, len(domainOrder))
		for index, domain := range domainOrder {
			if domain == item.Domain {
				values[index] = item.Mastery
			}
		}
		rows = append(rows, competencydto.HeatmapRow{Label: item.Label, Values: values})
	}

	sort.Slice(rows, func(i, j int) bool {
		left := maxHeatValue(rows[i].Values)
		right := maxHeatValue(rows[j].Values)
		if left == right {
			return rows[i].Label < rows[j].Label
		}
		return left < right
	})

	return domainOrder, rows
}

func buildGrowthInsights(items []competencydto.RadarAxis) []competencydto.InsightItem {
	if len(items) == 0 {
		return []competencydto.InsightItem{{Label: "能力成长", Value: 0, Note: "当前还没有足够的学习记录来生成稳定画像。"}}
	}

	sorted := append([]competencydto.RadarAxis{}, items...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Value == sorted[j].Value {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].Value < sorted[j].Value
	})

	limit := 3
	if len(sorted) < limit {
		limit = len(sorted)
	}
	result := make([]competencydto.InsightItem, 0, limit)
	for _, item := range sorted[:limit] {
		result = append(result, competencydto.InsightItem{
			Label: item.Name,
			Value: item.Value,
			Note:  "该能力维度由对应知识路径上的关键节点掌握度聚合得出，可直接作为下一轮补强重点。",
		})
	}
	return result
}

func buildGrowthSummaryCards(items []competencydto.RadarAxis) []competencydto.SummaryCard {
	if len(items) == 0 {
		return []competencydto.SummaryCard{{Label: "已形成优势", Value: "暂无"}}
	}

	sorted := append([]competencydto.RadarAxis{}, items...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Value == sorted[j].Value {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].Value > sorted[j].Value
	})

	shortBoard := sorted[len(sorted)-1].Name
	return []competencydto.SummaryCard{
		{Label: "当前优势", Value: sorted[0].Name},
		{Label: "优先补强", Value: shortBoard},
		{Label: "生成方式", Value: "知识路径聚合"},
	}
}

func maxHeatValue(values []float64) float64 {
	result := 0.0
	for _, value := range values {
		if value > result {
			result = value
		}
	}
	return result
}

func profileCacheUsable(profile *competencydto.ProfileResponse) bool {
	if profile == nil {
		return false
	}
	if len(profile.RadarAxes) > 0 {
		return true
	}
	for _, row := range profile.HeatmapRows {
		if len(row.Values) > 1 {
			return true
		}
	}
	return false
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
