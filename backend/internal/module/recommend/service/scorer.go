package service

import (
	"context"
	"math"
	"strings"

	frontierrepo "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/repository"
	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
)

// isInvalidFloat 检查浮点数是否为 NaN 或 Inf。
func isInvalidFloat(v float64) bool {
	return math.IsNaN(v) || math.IsInf(v, 0)
}

// ScorerConfig 统一评分器权重配置，可通过评测框架调优。
type ScorerConfig struct {
	Alpha  float64 // 薄弱度权重
	Beta   float64 // 先修满足度权重
	Gamma  float64 // 目标相关度权重
	Delta  float64 // 遗忘风险权重
	Eta    float64 // 跨课程迁移权重
	Theta  float64 // 前沿关联度权重
	Lambda float64 // 已掌握惩罚权重
}

// DefaultScorerConfig 返回初始权重，后续通过评测数据调优。
// 权重设计依据：薄弱度和先修满足度是核心信号（权重最高），
// 遗忘风险和跨课程迁移为辅助增强，前沿关联度为创新因子先设低权重。
func DefaultScorerConfig() ScorerConfig {
	return ScorerConfig{
		Alpha:  0.30,
		Beta:   0.20,
		Gamma:  0.20,
		Delta:  0.10,
		Eta:    0.10,
		Theta:  0.05,
		Lambda: 0.05,
	}
}

// ScoringFactors 将每个评分维度的贡献值序列化输出，供前端展示评分分解。
type ScoringFactors struct {
	Weakness              float64 `json:"weakness"`
	PrerequisiteReadiness float64 `json:"prerequisiteReadiness"`
	TargetRelevance       float64 `json:"targetRelevance"`
	ForgettingRisk        float64 `json:"forgettingRisk"`
	CrossCourseTransfer   float64 `json:"crossCourseTransfer"`
	FrontierRelevance     float64 `json:"frontierRelevance"`
	MasteredPenalty       float64 `json:"masteredPenalty"`
}

// ScoreInput 封装一次评分所需的全部上下文。
type ScoreInput struct {
	Candidate        StudentKnowledgeState
	KnowledgeID      string
	KnowledgeName    string
	StateLookup      map[string]StudentKnowledgeState
	TargetKnowledge  string
	SourceCourse     string
	TargetCourse     string
	RelationType     string
	SourceMastery    float64
}

// NewScoreInput 创建评分输入，提供给外部模块（如评测框架）使用。
func NewScoreInput(
	knowledgeID, knowledgeName string,
	masteryScore, correctRate float64,
	answeredCount int,
	lastAnsweredAt, status string,
	stateLookup map[string]StudentKnowledgeState,
	targetKnowledge, sourceCourse, targetCourse, relationType string,
	sourceMastery float64,
) ScoreInput {
	return ScoreInput{
		Candidate: StudentKnowledgeState{
			KnowledgeID:    knowledgeID,
			CanonicalName:  knowledgeName,
			MasteryScore:   masteryScore,
			CorrectRate:    correctRate,
			AnsweredCount:  answeredCount,
			LastAnsweredAt: lastAnsweredAt,
			Status:         status,
		},
		KnowledgeID:     knowledgeID,
		KnowledgeName:   knowledgeName,
		StateLookup:     stateLookup,
		TargetKnowledge: targetKnowledge,
		SourceCourse:    sourceCourse,
		TargetCourse:    targetCourse,
		RelationType:    relationType,
		SourceMastery:   sourceMastery,
	}
}

// Scorer 实现 KT-KGPath 统一评分公式。
type Scorer struct {
	config       ScorerConfig
	graphService *graphservice.Service
	frontierRepo *frontierrepo.Repository
}

// NewScorer 创建统一评分器。
func NewScorer(config ScorerConfig, graphService *graphservice.Service, frontierRepo *frontierrepo.Repository) *Scorer {
	return &Scorer{
		config:       config,
		graphService: graphService,
		frontierRepo: frontierRepo,
	}
}

// ScoreCandidate 统一评分入口，返回 [0,1] 归一化分数 + 因子分解。
func (s *Scorer) ScoreCandidate(ctx context.Context, input ScoreInput) (float64, ScoringFactors) {
	factors := ScoringFactors{
		Weakness:             s.weaknessFactor(input.Candidate),
		PrerequisiteReadiness: s.prerequisiteReadinessFactor(ctx, input),
		TargetRelevance:      s.targetRelevanceFactor(ctx, input),
		ForgettingRisk:       s.forgettingRiskFactor(input.Candidate),
		CrossCourseTransfer:  s.crossCourseTransferFactor(input),
		FrontierRelevance:    s.frontierRelevanceFactor(ctx, input),
		MasteredPenalty:      s.masteredPenalty(input.Candidate),
	}

	rawScore := s.config.Alpha*factors.Weakness +
		s.config.Beta*factors.PrerequisiteReadiness +
		s.config.Gamma*factors.TargetRelevance +
		s.config.Delta*factors.ForgettingRisk +
		s.config.Eta*factors.CrossCourseTransfer +
		s.config.Theta*factors.FrontierRelevance -
		s.config.Lambda*factors.MasteredPenalty

	normalized := clamp(rawScore, 0, 1)

	// 防御性检查：确保因子全部为合法值
	if isInvalidFloat(normalized) {
		normalized = 0
	}
	factors.Weakness = clamp(factors.Weakness, 0, 1)
	factors.PrerequisiteReadiness = clamp(factors.PrerequisiteReadiness, 0, 1)
	factors.TargetRelevance = clamp(factors.TargetRelevance, 0, 1)
	factors.ForgettingRisk = clamp(factors.ForgettingRisk, 0, 1)
	factors.CrossCourseTransfer = clamp(factors.CrossCourseTransfer, 0, 1)
	factors.FrontierRelevance = clamp(factors.FrontierRelevance, 0, 1)
	factors.MasteredPenalty = clamp(factors.MasteredPenalty, 0, 1)

	return normalized, factors
}

// weaknessFactor 计算薄弱度因子 [0,1]。
// 基于现有 weaknessScore() 逻辑归一化：原始分数范围约 [-18, 88]。
func (s *Scorer) weaknessFactor(state StudentKnowledgeState) float64 {
	raw := 0.0
	switch {
	case state.MasteryScore < 20:
		raw += 62
	case state.MasteryScore < 40:
		raw += 52
	case state.MasteryScore < 60:
		raw += 38
	case state.MasteryScore < 80:
		raw += 14
	default:
		raw -= 18
	}
	if state.AnsweredCount == 0 {
		raw += 8
	} else if state.CorrectRate < 0.5 {
		raw += 18
	} else if state.CorrectRate < 0.7 {
		raw += 10
	}
	if state.Status == "not_started" {
		raw += 6
	}
	// 归一化：原始范围约 [-18, 88] → [0, 1]
	return clamp((raw+18)/106.0, 0, 1)
}

// prerequisiteReadinessFactor 计算先修满足度因子 [0,1]。
// 查 Neo4j 先修链，检查学生对每个先修知识点的掌握程度。
func (s *Scorer) prerequisiteReadinessFactor(ctx context.Context, input ScoreInput) float64 {
	if s.graphService == nil || strings.TrimSpace(input.KnowledgeID) == "" {
		return 0.5 // 无图谱数据时保持中立
	}

	chain, err := s.graphService.GetPrerequisiteChain(ctx, graphdto.PrerequisiteChainQuery{
		Target: input.KnowledgeID,
		Depth:  3,
	})
	if err != nil || chain == nil || len(chain.OrderedNodeIDs) == 0 {
		return 0.5 // 无先修链时中立
	}

	total := 0.0
	count := 0.0
	for _, nodeID := range chain.OrderedNodeIDs {
		// 跳过自身
		if sameKnowledgeID(strings.TrimSpace(nodeID), input.KnowledgeID) {
			continue
		}
		count++
		state, ok := input.StateLookup[normalizeStateKey(nodeID)]
		if !ok {
			total += 0.0
			continue
		}
		switch {
		case state.MasteryScore >= 80:
			total += 1.0
		case state.MasteryScore >= 60:
			total += 0.6
		case state.MasteryScore >= 30:
			total += 0.2
		default:
			total += 0.0
		}
	}
	if count == 0 {
		return 0.5
	}
	return clamp(total/count, 0, 1)
}

// targetRelevanceFactor 计算目标相关度因子 [0,1]。
// 使用 FindShortestPath 距离归一化，距离越近越相关。
func (s *Scorer) targetRelevanceFactor(ctx context.Context, input ScoreInput) float64 {
	target := strings.TrimSpace(input.TargetKnowledge)
	if target == "" || strings.TrimSpace(input.KnowledgeID) == "" {
		return 0.5 // 无目标时中立
	}
	if sameKnowledgeID(input.KnowledgeID, target) {
		return 1.0
	}
	if s.graphService == nil {
		return 0.5
	}

	path, err := s.graphService.FindPath(ctx, graphdto.GraphPathQuery{
		From:     input.KnowledgeID,
		To:       target,
		MaxDepth: 5,
	})
	if err != nil || path == nil || !path.Found || len(path.OrderedNodeIDs) <= 1 {
		return 0.3 // 不可达时偏低
	}

	depth := float64(len(path.OrderedNodeIDs) - 1)
	// 距离归一化：深度0=1.0，深度5+=0.2
	relevance := 1.0 - depth*0.16
	return clamp(relevance, 0.2, 1.0)
}

// forgettingRiskFactor 计算遗忘风险因子 [0,1]。
// 基于现有 effectiveMastery() 模型：有效掌握度 = mastery * exp(-days/7)。
func (s *Scorer) forgettingRiskFactor(state StudentKnowledgeState) float64 {
	if state.MasteryScore < 60 || strings.TrimSpace(state.LastAnsweredAt) == "" {
		return 0.0 // 未达到基础掌握或从未答题，无遗忘风险
	}
	effective := effectiveMastery(state.MasteryScore, state.LastAnsweredAt)
	risk := 1.0 - effective/100.0
	return clamp(risk, 0, 1)
}

// crossCourseTransferFactor 计算跨课程迁移价值因子 [0,1]。
// 基于现有 transferRelationWeight() 逻辑归一化。
func (s *Scorer) crossCourseTransferFactor(input ScoreInput) float64 {
	srcCourse := strings.TrimSpace(input.SourceCourse)
	tgtCourse := strings.TrimSpace(input.TargetCourse)
	if srcCourse == "" || tgtCourse == "" || srcCourse == tgtCourse {
		return 0.0 // 同课程无迁移价值
	}
	if input.SourceMastery < 60 {
		return 0.0 // 源知识未掌握，无法迁移
	}

	weight := transferRelationWeight(input.RelationType)
	// 归一化到 [0,1]：transfer 权重范围约 [0.88, 1.15]
	normalized := (weight - 0.88) / 0.27
	// 源掌握度越高迁移价值越大
	return clamp(normalized*(input.SourceMastery/100.0), 0, 1)
}

// frontierRelevanceFactor 计算前沿关联度因子 [0,1]。
// 调用 frontierRepo 查询前沿摘要、主题和论文。
// 策略：仅在 briefing 状态为 ready 时计算，其他状态返回 0。
func (s *Scorer) frontierRelevanceFactor(ctx context.Context, input ScoreInput) float64 {
	if s.frontierRepo == nil {
		return 0.0
	}
	knowledgeID := strings.TrimSpace(input.KnowledgeID)
	knowledgeName := strings.TrimSpace(input.KnowledgeName)
	if knowledgeID == "" && knowledgeName == "" {
		return 0.0
	}

	briefing, err := s.frontierRepo.GetBriefing(ctx, knowledgeID, knowledgeName)
	if err != nil || briefing == nil {
		return 0.0
	}

	status := strings.ToLower(strings.TrimSpace(briefing.Status))
	switch status {
	case "ready":
		// 计算主题和论文的综合得分
		topicScore := normalizeCount(len(briefing.Topics), 5)
		paperScore := normalizeCount(len(briefing.Papers), 5)

		// 摘要内容加分
		summaryBonus := 0.0
		if strings.TrimSpace(briefing.DigestSummary) != "" {
			summaryBonus = 0.15
		}

		return clamp((topicScore+paperScore)/2.0+summaryBonus, 0, 1)
	case "stale":
		return 0.3
	case "not_ready":
		return 0.0
	case "failed":
		return 0.0
	default:
		return 0.0
	}
}

// masteredPenalty 计算已掌握惩罚因子 [0,1]。
// 与现有 filter（mastery >= 85 && correctRate >= 0.85 跳过）互补，
// 对接近掌握的知识点施加软惩罚。
func (s *Scorer) masteredPenalty(state StudentKnowledgeState) float64 {
	switch {
	case state.MasteryScore >= 90:
		return 1.0
	case state.MasteryScore >= 80:
		return 0.5
	case state.MasteryScore >= 70:
		return 0.2
	default:
		return 0.0
	}
}

// normalizeCount 将计数归一化到 [0,1]，count 为 0 时返回 0。
func normalizeCount(count int, maxCount int) float64 {
	if count <= 0 {
		return 0.0
	}
	if count >= maxCount {
		return 1.0
	}
	return float64(count) / float64(maxCount)
}

// clamp 将值限制在 [min, max] 区间。
func clamp(value, min, max float64) float64 {
	if math.IsNaN(value) {
		return min
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
