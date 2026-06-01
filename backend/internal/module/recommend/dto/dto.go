package dto

// PathQuery 定义推荐路径查询参数。
type PathQuery struct {
	Target      string `form:"target" binding:"omitempty,max=128"`
	KnowledgeID string `form:"knowledgeId" binding:"omitempty,max=128"`
	Limit       int    `form:"limit" binding:"omitempty,min=1,max=20"`
	Scene       string `form:"scene" binding:"omitempty,max=64"`
}

// PathItemView 定义推荐路径项。
type PathItemView struct {
	Stage            string                `json:"stage"`
	Title            string                `json:"title"`
	Detail           string                `json:"detail"`
	Route            string                `json:"route"`
	KnowledgeID      string                `json:"knowledgeId"`
	ReasonCode       string                `json:"reasonCode"`
	ReasonLabel      string                `json:"reasonLabel"`
	SourceCourse     string                `json:"sourceCourse,omitempty"`
	TargetCourse     string                `json:"targetCourse,omitempty"`
	MasteryScore     float64               `json:"masteryScore"`
	CorrectRate      float64               `json:"correctRate"`
	AnsweredCount    int                   `json:"answeredCount"`
	Score            float64               `json:"score"`
	ScoringBreakdown *ScoringBreakdownView `json:"scoringBreakdown,omitempty"`
}

// ScoringBreakdownView 展示 KT-KGPath 统一评分器的因子分解。
type ScoringBreakdownView struct {
	Weakness              float64 `json:"weakness"`
	PrerequisiteReadiness float64 `json:"prerequisiteReadiness"`
	TargetRelevance       float64 `json:"targetRelevance"`
	ForgettingRisk        float64 `json:"forgettingRisk"`
	CrossCourseTransfer   float64 `json:"crossCourseTransfer"`
	FrontierRelevance     float64 `json:"frontierRelevance"`
	MasteredPenalty       float64 `json:"masteredPenalty"`
	Overall               float64 `json:"overall"`
}

// LearningInsightView defines a proactive learning hint.
type LearningInsightView struct {
	Type        string `json:"type"`
	ReasonCode  string `json:"reasonCode"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	KnowledgeID string `json:"knowledgeId,omitempty"`
	Route       string `json:"route,omitempty"`
}

// LearnerProfileView defines the inferred learning style.
type LearnerProfileView struct {
	Style   string `json:"style"`
	Label   string `json:"label"`
	Summary string `json:"summary"`
}

// PathResponse 定义推荐路径响应。
type PathResponse struct {
	Path           []PathItemView        `json:"path"`
	Insights       []LearningInsightView `json:"insights,omitempty"`
	LearnerProfile *LearnerProfileView   `json:"learnerProfile,omitempty"`
}
