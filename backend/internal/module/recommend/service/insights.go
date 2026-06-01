package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
)

// DetectLearningBottlenecks summarizes the strongest current learning risks.
func (s *Service) DetectLearningBottlenecks(ctx context.Context, query recommenddto.PathQuery, progressList []StudentKnowledgeState, stateLookup map[string]StudentKnowledgeState, recommendedPath []recommenddto.PathItemView) []recommenddto.LearningInsightView {
	insights := make([]recommenddto.LearningInsightView, 0, 3)
	seen := make(map[string]struct{}, 4)
	appendInsight := func(item recommenddto.LearningInsightView) {
		key := strings.TrimSpace(item.ReasonCode)
		if key == "" {
			key = strings.TrimSpace(item.Title)
		}
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		insights = append(insights, item)
	}

	if knowledgeID := strings.TrimSpace(query.KnowledgeID); knowledgeID != "" {
		if gapPath := s.buildGapBridgePath(ctx, knowledgeID, query.Scene, 1, stateLookup); len(gapPath) > 0 {
			item := gapPath[0]
			appendInsight(recommenddto.LearningInsightView{
				Type:        "proactive_hint",
				ReasonCode:  "gap_bridge",
				Severity:    "high",
				Title:       "当前目标前有基础断层",
				Detail:      item.Detail,
				KnowledgeID: item.KnowledgeID,
				Route:       item.Route,
			})
		}
	}

	if weakness := topWeaknessCandidate(progressList); weakness != nil {
		appendInsight(recommenddto.LearningInsightView{
			Type:        "proactive_hint",
			ReasonCode:  "weakness",
			Severity:    weaknessSeverity(*weakness),
			Title:       "有一个知识点在反复卡住你",
			Detail:      weaknessDetail(*weakness),
			KnowledgeID: strings.TrimSpace(weakness.KnowledgeID),
			Route:       fmt.Sprintf("/knowledge/%s", strings.TrimSpace(weakness.KnowledgeID)),
		})
	}

	if reviewPath := s.buildReviewPath(progressList, query.Scene, 1); len(reviewPath) > 0 {
		item := reviewPath[0]
		appendInsight(recommenddto.LearningInsightView{
			Type:        "proactive_hint",
			ReasonCode:  "forgetting_review",
			Severity:    "medium",
			Title:       "有一个知识点到了回温窗口",
			Detail:      item.Detail,
			KnowledgeID: item.KnowledgeID,
			Route:       item.Route,
		})
	}

	if transfer := topReasonPathItem(recommendedPath, "cross_course_transfer"); transfer != nil {
		appendInsight(recommenddto.LearningInsightView{
			Type:        "proactive_hint",
			ReasonCode:  "cross_course_transfer",
			Severity:    "low",
			Title:       "你可以利用已有强项跨课程迁移",
			Detail:      transfer.Detail,
			KnowledgeID: transfer.KnowledgeID,
			Route:       transfer.Route,
		})
	}

	if len(insights) > 3 {
		return insights[:3]
	}
	return insights
}

func weaknessScore(item StudentKnowledgeState) float64 {
	score := 0.0
	switch {
	case item.MasteryScore < 20:
		score += 62
	case item.MasteryScore < 40:
		score += 52
	case item.MasteryScore < 60:
		score += 38
	case item.MasteryScore < 80:
		score += 14
	default:
		score -= 18
	}
	if item.AnsweredCount == 0 {
		score += 8
	} else if item.CorrectRate < 0.5 {
		score += 18
	} else if item.CorrectRate < 0.7 {
		score += 10
	}
	if item.Status == "not_started" {
		score += 6
	}
	return score
}

func weaknessDetail(item StudentKnowledgeState) string {
	name := strings.TrimSpace(item.CanonicalName)
	if name == "" {
		name = strings.TrimSpace(item.KnowledgeID)
	}
	if item.AnsweredCount == 0 {
		return fmt.Sprintf("%s 当前还没有稳定学习记录，建议先从基础定义和最小例子入手。", name)
	}
	return fmt.Sprintf("%s 当前掌握度 %.0f%%，正确率 %.0f%%，适合作为优先补强点。", name, item.MasteryScore, item.CorrectRate*100)
}

func topWeaknessCandidate(progressList []StudentKnowledgeState) *StudentKnowledgeState {
	if len(progressList) == 0 {
		return nil
	}
	candidates := append([]StudentKnowledgeState(nil), progressList...)
	sort.SliceStable(candidates, func(i, j int) bool {
		left := weaknessScore(candidates[i])
		right := weaknessScore(candidates[j])
		if math.Abs(left-right) < 0.001 {
			return candidates[i].CanonicalName < candidates[j].CanonicalName
		}
		return left > right
	})
	for _, item := range candidates {
		if item.MasteryScore >= 55 && item.CorrectRate >= 0.65 {
			continue
		}
		if item.AnsweredCount == 0 && item.MasteryScore >= 40 {
			continue
		}
		copyItem := item
		return &copyItem
	}
	return nil
}

func weaknessSeverity(item StudentKnowledgeState) string {
	if item.MasteryScore < 30 || (item.AnsweredCount >= 3 && item.CorrectRate < 0.45) {
		return "high"
	}
	if item.MasteryScore < 55 || item.CorrectRate < 0.6 {
		return "medium"
	}
	return "low"
}

func topReasonPathItem(items []recommenddto.PathItemView, reasonCode string) *recommenddto.PathItemView {
	for _, item := range items {
		if strings.TrimSpace(item.ReasonCode) != strings.TrimSpace(reasonCode) {
			continue
		}
		copyItem := item
		return &copyItem
	}
	return nil
}

func inferLearnerProfile(progressList []StudentKnowledgeState) *recommenddto.LearnerProfileView {
	activeNodes := 0
	totalAnswered := 0
	totalMastery := 0.0
	strongNodes := 0
	weakNodes := 0

	for _, item := range progressList {
		if item.AnsweredCount > 0 {
			activeNodes++
			totalAnswered += item.AnsweredCount
		}
		totalMastery += item.MasteryScore
		if item.MasteryScore >= 80 {
			strongNodes++
		}
		if item.AnsweredCount > 0 && item.MasteryScore < 40 {
			weakNodes++
		}
	}

	if len(progressList) == 0 || activeNodes == 0 {
		return &recommenddto.LearnerProfileView{
			Style:   "exploratory",
			Label:   "探索型",
			Summary: "当前系统里的学习记录还不多，适合先从兴趣点切入，再逐步把主线补齐。",
		}
	}

	avgDepth := float64(totalAnswered) / float64(activeNodes)
	avgMastery := totalMastery / float64(len(progressList))
	weakRatio := float64(weakNodes) / float64(activeNodes)
	strongRatio := float64(strongNodes) / float64(max(len(progressList), 1))

	switch {
	case weakRatio >= 0.45 || avgMastery < 48:
		return &recommenddto.LearnerProfileView{
			Style:   "struggling",
			Label:   "困难型",
			Summary: "你最近更适合被拆成更小步地学习，先稳住基础和正确率，再往前推进。",
		}
	case avgDepth >= 9 && activeNodes <= 5:
		return &recommenddto.LearnerProfileView{
			Style:   "focused",
			Label:   "专注型",
			Summary: "你倾向于深挖少数主题，适合直接给出重点、路径和少量关键对比。",
		}
	case activeNodes >= 10 && avgDepth <= 4.5:
		return &recommenddto.LearnerProfileView{
			Style:   "exploratory",
			Label:   "探索型",
			Summary: "你喜欢从多个知识点并行摸索，适合多给跨课程连接和有趣的入口。",
		}
	case activeNodes >= 8 && strongRatio >= 0.28:
		return &recommenddto.LearnerProfileView{
			Style:   "systematic",
			Label:   "系统型",
			Summary: "你更适合按知识树和先修顺序稳步推进，结构化学习会让你更快进入状态。",
		}
	default:
		return &recommenddto.LearnerProfileView{
			Style:   "focused",
			Label:   "专注型",
			Summary: "你当前更适合围绕一个目标连续推进，少切换主题会更高效。",
		}
	}
}
