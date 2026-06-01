package entity

// BenchmarkTask 定义一次评测任务的输入参数。
type BenchmarkTask struct {
	Name           string              `json:"name"`
	StudentHistory []AnswerRecord      `json:"studentHistory"`
	CandidatePool  []CandidateNode     `json:"candidatePool"`
	GroundTruth    []string            `json:"groundTruth"`
	Prerequisites  []PrerequisiteEdge  `json:"prerequisites,omitempty"`
	Difficulties   map[string]float64  `json:"difficulties,omitempty"`
}

// AnswerRecord 学生答题记录。
type AnswerRecord struct {
	KnowledgeID    string  `json:"knowledgeId"`
	CanonicalName  string  `json:"canonicalName"`
	MasteryScore   float64 `json:"masteryScore"`
	CorrectRate    float64 `json:"correctRate"`
	AnsweredCount  int     `json:"answeredCount"`
	LastAnsweredAt string  `json:"lastAnsweredAt,omitempty"`
	Status         string  `json:"status"`
}

// CandidateNode 候选知识节点。
type CandidateNode struct {
	KnowledgeID   string `json:"knowledgeId"`
	CanonicalName string `json:"name"`
	CourseTitle   string `json:"courseTitle"`
}

// PrerequisiteEdge 先修边。
type PrerequisiteEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// BenchmarkResult 单个 baseline 的评测聚合结果。
type BenchmarkResult struct {
	Baseline              string  `json:"baseline"`
	HitAt1                float64 `json:"hitAt1"`
	HitAt3                float64 `json:"hitAt3"`
	HitAt5                float64 `json:"hitAt5"`
	NDCGAt5               float64 `json:"ndcgAt5"`
	NDCGAt10              float64 `json:"ndcgAt10"`
	MRR                   float64 `json:"mrr"`
	PrerequisiteViolation float64 `json:"prerequisiteViolationRate"`
	DifficultySmoothness  float64 `json:"difficultySmoothness"`
	AveragePathLength     float64 `json:"averagePathLength"`
	CoverageRate          float64 `json:"coverageRate"`
}
