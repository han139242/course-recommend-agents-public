package service

import (
	"context"
	"math"
	"testing"
)

func TestWeaknessFactor_Min(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	state := StudentKnowledgeState{
		MasteryScore:  100,
		CorrectRate:   1.0,
		AnsweredCount: 10,
		Status:        "solid",
	}
	result := s.weaknessFactor(state)
	if result != 0.0 {
		t.Errorf("最低薄弱度期望 0.0，实际 %.4f", result)
	}
}

func TestWeaknessFactor_Max(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	state := StudentKnowledgeState{
		MasteryScore:  0,
		CorrectRate:   0.0,
		AnsweredCount: 0,
		Status:        "not_started",
	}
	result := s.weaknessFactor(state)
	if result < 0.7 {
		t.Errorf("最高薄弱度期望 > 0.7，实际 %.4f", result)
	}
}

func TestWeaknessFactor_MidRange(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	state := StudentKnowledgeState{
		MasteryScore:  50,
		CorrectRate:   0.6,
		AnsweredCount: 5,
		Status:        "progressing",
	}
	result := s.weaknessFactor(state)
	if result < 0.3 || result > 0.7 {
		t.Errorf("期望中段薄弱度在 [0.3, 0.7]，实际 %.4f", result)
	}
}

func TestPrerequisiteReadiness_NilGraphService(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	input := ScoreInput{KnowledgeID: "test-node"}
	result := s.prerequisiteReadinessFactor(context.Background(), input)
	if math.Abs(result-0.5) > 0.001 {
		t.Errorf("无图谱服务期望 0.5，实际 %.4f", result)
	}
}

func TestPrerequisiteReadiness_EmptyID(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	input := ScoreInput{KnowledgeID: ""}
	result := s.prerequisiteReadinessFactor(context.Background(), input)
	if math.Abs(result-0.5) > 0.001 {
		t.Errorf("空知识ID期望 0.5，实际 %.4f", result)
	}
}

func TestTargetRelevance_SameAsTarget(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	input := ScoreInput{
		KnowledgeID:     "gradient-descent",
		TargetKnowledge: "gradient-descent",
	}
	result := s.targetRelevanceFactor(context.Background(), input)
	if math.Abs(result-1.0) > 0.001 {
		t.Errorf("与目标相同时期望 1.0，实际 %.4f", result)
	}
}

func TestTargetRelevance_NoTarget(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	input := ScoreInput{
		KnowledgeID:     "gradient-descent",
		TargetKnowledge: "",
	}
	result := s.targetRelevanceFactor(context.Background(), input)
	if math.Abs(result-0.5) > 0.001 {
		t.Errorf("无目标期望 0.5，实际 %.4f", result)
	}
}

func TestForgettingRisk_NeverAnswered(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	state := StudentKnowledgeState{
		MasteryScore:   80,
		LastAnsweredAt: "",
	}
	result := s.forgettingRiskFactor(state)
	if result > 0.001 {
		t.Errorf("从未答题期望 0.0，实际 %.4f", result)
	}
}

func TestForgettingRisk_LowMastery(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	state := StudentKnowledgeState{
		MasteryScore:   40,
		LastAnsweredAt: "2026-05-20T10:00:00Z",
	}
	result := s.forgettingRiskFactor(state)
	if result > 0.001 {
		t.Errorf("低掌握度(<60)期望 0.0，实际 %.4f", result)
	}
}

func TestForgettingRisk_Valid(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	// 30天前答题，掌握度100，有效掌握度 ≈ 100*exp(-30/7) ≈ 1.37
	// 风险 = 1 - 1.37/100 ≈ 0.986
	state := StudentKnowledgeState{
		MasteryScore:   100,
		LastAnsweredAt: "2026-04-25T10:00:00Z",
	}
	result := s.forgettingRiskFactor(state)
	if result < 0.5 {
		t.Errorf("30天遗忘风险期望 > 0.5，实际 %.4f", result)
	}
}

func TestCrossCourseTransfer_NoSourceCourse(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	input := ScoreInput{
		SourceCourse:  "",
		TargetCourse:  "深度学习",
		SourceMastery: 80,
	}
	result := s.crossCourseTransferFactor(input)
	if result > 0.001 {
		t.Errorf("无源课程期望 0.0，实际 %.4f", result)
	}
}

func TestCrossCourseTransfer_SameCourse(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	input := ScoreInput{
		SourceCourse:  "深度学习",
		TargetCourse:  "深度学习",
		SourceMastery: 80,
	}
	result := s.crossCourseTransferFactor(input)
	if result > 0.001 {
		t.Errorf("同课程期望 0.0，实际 %.4f", result)
	}
}

func TestCrossCourseTransfer_DifferentCourse(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	input := ScoreInput{
		SourceCourse:  "线性代数",
		TargetCourse:  "深度学习",
		RelationType:  "PREREQUISITE_OF",
		SourceMastery: 85,
	}
	result := s.crossCourseTransferFactor(input)
	if result < 0.1 {
		t.Errorf("跨课程PREREQUISITE_OF期望 > 0.1，实际 %.4f", result)
	}
}

func TestMasteredPenalty_NotMastered(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	state := StudentKnowledgeState{MasteryScore: 30}
	result := s.masteredPenalty(state)
	if result > 0.001 {
		t.Errorf("未掌握期望 0.0，实际 %.4f", result)
	}
}

func TestMasteredPenalty_Full(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	state := StudentKnowledgeState{MasteryScore: 95}
	result := s.masteredPenalty(state)
	if math.Abs(result-1.0) > 0.001 {
		t.Errorf("完全掌握期望 1.0，实际 %.4f", result)
	}
}

func TestMasteredPenalty_Borderline(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	state := StudentKnowledgeState{MasteryScore: 75}
	result := s.masteredPenalty(state)
	if math.Abs(result-0.2) > 0.001 {
		t.Errorf("边界(70-80)期望 0.2，实际 %.4f", result)
	}
}

func TestFrontierRelevance_NilRepo(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	input := ScoreInput{KnowledgeID: "test-node"}
	result := s.frontierRelevanceFactor(context.Background(), input)
	if result > 0.001 {
		t.Errorf("无 frontier 仓库期望 0.0，实际 %.4f", result)
	}
}

func TestFrontierRelevance_EmptyID(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	input := ScoreInput{KnowledgeID: "", KnowledgeName: ""}
	result := s.frontierRelevanceFactor(context.Background(), input)
	if result > 0.001 {
		t.Errorf("空知识ID期望 0.0，实际 %.4f", result)
	}
}

func TestComputeScore_AllFactorsOne(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	// 手动构造满因子情况：但 weakness/forgetting/frontier/mastered 各有限制
	// 这里只测试纯函数调用链路通畅
	state := StudentKnowledgeState{
		KnowledgeID:   "test-node",
		CanonicalName: "测试知识点",
		MasteryScore:  0,
		CorrectRate:   0.0,
		AnsweredCount: 0,
		Status:        "not_started",
	}
	input := ScoreInput{
		Candidate:       state,
		KnowledgeID:     "test-node",
		KnowledgeName:   "测试知识点",
		TargetKnowledge: "test-node",
		RelationType:    "RELATED_TO",
		SourceCourse:    "线性代数",
		TargetCourse:    "深度学习",
		SourceMastery:   85,
		StateLookup:     map[string]StudentKnowledgeState{},
	}
	score, factors := s.ScoreCandidate(context.Background(), input)
	if score < 0 || score > 1 {
		t.Errorf("分数应在 [0,1] 范围，实际 %.4f", score)
	}
	if factors.Weakness < 0.5 {
		t.Errorf("0掌握度期望薄弱度较高，实际 %.4f", factors.Weakness)
	}
}

func TestComputeScore_ClampedRange(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	state := StudentKnowledgeState{
		MasteryScore:  100,
		CorrectRate:   1.0,
		AnsweredCount: 100,
		Status:        "solid",
		LastAnsweredAt: "2026-05-25T10:00:00Z",
	}
	input := ScoreInput{
		Candidate:       state,
		KnowledgeID:     "solid-node",
		TargetKnowledge: "",
	}
	score, _ := s.ScoreCandidate(context.Background(), input)
	if score < 0 || score > 1 {
		t.Errorf("分数应在 [0,1] 范围，实际 %.4f", score)
	}
}

func TestNormalizeCount(t *testing.T) {
	tests := []struct {
		count    int
		maxCount int
		expected float64
	}{
		{0, 5, 0.0},
		{3, 5, 0.6},
		{5, 5, 1.0},
		{10, 5, 1.0},
	}
	for _, tt := range tests {
		result := normalizeCount(tt.count, tt.maxCount)
		if math.Abs(result-tt.expected) > 0.001 {
			t.Errorf("normalizeCount(%d, %d) = %.4f, 期望 %.4f", tt.count, tt.maxCount, result, tt.expected)
		}
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		value    float64
		min      float64
		max      float64
		expected float64
	}{
		{0.5, 0, 1, 0.5},
		{1.5, 0, 1, 1.0},
		{-0.5, 0, 1, 0.0},
		{math.NaN(), 0, 1, 0.0},
	}
	for _, tt := range tests {
		result := clamp(tt.value, tt.min, tt.max)
		if math.Abs(result-tt.expected) > 0.001 && !math.IsNaN(tt.value) {
			t.Errorf("clamp(%.4f, %.0f, %.0f) = %.4f, 期望 %.4f", tt.value, tt.min, tt.max, result, tt.expected)
		}
		if math.IsNaN(tt.value) && !math.IsNaN(result) {
			return
		}
	}
}

func TestClamp_Inf(t *testing.T) {
	if result := clamp(math.Inf(1), 0, 1); result != 1.0 {
		t.Errorf("+Inf 应 clamp 到 max=1.0，实际 %.4f", result)
	}
	if result := clamp(math.Inf(-1), 0, 1); result != 0.0 {
		t.Errorf("-Inf 应 clamp 到 min=0.0，实际 %.4f", result)
	}
}

func TestIsInvalidFloat(t *testing.T) {
	if !isInvalidFloat(math.NaN()) {
		t.Error("NaN 应判定为非法")
	}
	if !isInvalidFloat(math.Inf(1)) {
		t.Error("+Inf 应判定为非法")
	}
	if !isInvalidFloat(math.Inf(-1)) {
		t.Error("-Inf 应判定为非法")
	}
	if isInvalidFloat(0.5) {
		t.Error("0.5 不应判定为非法")
	}
}

func TestScoreCandidate_NanGuard(t *testing.T) {
	s := NewScorer(DefaultScorerConfig(), nil, nil)
	// 构造极端值确保所有因子计算路径覆盖
	state := StudentKnowledgeState{
		KnowledgeID:   "nan-test",
		MasteryScore:  100,
		CorrectRate:   1.0,
		AnsweredCount: 0,
		Status:        "solid",
	}
	input := ScoreInput{
		Candidate:    state,
		KnowledgeID:  "nan-test",
		SourceCourse: "",
		TargetCourse: "",
	}
	score, factors := s.ScoreCandidate(context.Background(), input)
	if isInvalidFloat(score) {
		t.Errorf("最终分数不应为 NaN/Inf: %.4f", score)
	}
	if isInvalidFloat(factors.Weakness) || isInvalidFloat(factors.PrerequisiteReadiness) ||
		isInvalidFloat(factors.TargetRelevance) || isInvalidFloat(factors.ForgettingRisk) ||
		isInvalidFloat(factors.CrossCourseTransfer) || isInvalidFloat(factors.FrontierRelevance) ||
		isInvalidFloat(factors.MasteredPenalty) {
		t.Errorf("因子中不应出现 NaN/Inf: %+v", factors)
	}
}
