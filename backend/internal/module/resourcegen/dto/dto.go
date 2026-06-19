package dto

// GenerateResourcesRequest creates a personalized resource package.
type GenerateResourcesRequest struct {
	KnowledgeID   string   `json:"knowledgeId" binding:"omitempty,max=128"`
	KnowledgeName string   `json:"knowledgeName" binding:"omitempty,max=255"`
	CourseTitle   string   `json:"courseTitle" binding:"omitempty,max=255"`
	Target        string   `json:"target" binding:"omitempty,max=500"`
	ResourceTypes []string `json:"resourceTypes" binding:"omitempty,max=10"`
}

// ResourceQuery filters generated resources.
type ResourceQuery struct {
	KnowledgeID string `form:"knowledgeId" binding:"omitempty,max=128"`
	Type        string `form:"type" binding:"omitempty,max=64"`
	Limit       int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

// JobView describes one generation job.
type JobView struct {
	ID            int64                   `json:"id"`
	StudentID     int64                   `json:"studentId"`
	KnowledgeID   string                  `json:"knowledgeId"`
	KnowledgeName string                  `json:"knowledgeName"`
	CourseTitle   string                  `json:"courseTitle"`
	Target        string                  `json:"target"`
	Status        string                  `json:"status"`
	Progress      int                     `json:"progress"`
	CurrentAgent  string                  `json:"currentAgent"`
	ErrorMessage  string                  `json:"errorMessage,omitempty"`
	CreatedAt     string                  `json:"createdAt"`
	UpdatedAt     string                  `json:"updatedAt"`
	Steps         []AgentStepView         `json:"steps,omitempty"`
	Resources     []GeneratedResourceView `json:"resources,omitempty"`
}

// AgentStepView records a visible multi-agent step.
type AgentStepView struct {
	ID        int64  `json:"id"`
	AgentKey  string `json:"agentKey"`
	AgentName string `json:"agentName"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	Summary   string `json:"summary"`
	StartedAt string `json:"startedAt"`
	EndedAt   string `json:"endedAt,omitempty"`
}

// GeneratedResourceView describes a generated learning resource.
type GeneratedResourceView struct {
	ID            int64  `json:"id"`
	JobID         int64  `json:"jobId"`
	ResourceType  string `json:"resourceType"`
	TypeLabel     string `json:"typeLabel"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	Format        string `json:"format"`
	KnowledgeID   string `json:"knowledgeId"`
	KnowledgeName string `json:"knowledgeName"`
	CourseTitle   string `json:"courseTitle"`
	Reason        string `json:"reason"`
	Status        string `json:"status"`
	CreatedAt     string `json:"createdAt"`
}

// SubmitFeedbackRequest records learner feedback for one generated resource.
type SubmitFeedbackRequest struct {
	Rating     int    `json:"rating" binding:"required,min=1,max=5"`
	Difficulty string `json:"difficulty" binding:"omitempty,max=32"`
	Helpful    bool   `json:"helpful"`
	Completed  bool   `json:"completed"`
	Comment    string `json:"comment" binding:"omitempty,max=500"`
}

// FeedbackView describes one resource feedback record.
type FeedbackView struct {
	ID         int64  `json:"id"`
	ResourceID int64  `json:"resourceId"`
	JobID      int64  `json:"jobId"`
	Rating     int    `json:"rating"`
	Difficulty string `json:"difficulty"`
	Helpful    bool   `json:"helpful"`
	Completed  bool   `json:"completed"`
	Comment    string `json:"comment"`
	NextAction string `json:"nextAction"`
	UpdatedAt  string `json:"updatedAt"`
}

// FeedbackSummaryView aggregates recent feedback for learning-effect assessment.
type FeedbackSummaryView struct {
	Total          int            `json:"total"`
	AverageRating  float64        `json:"averageRating"`
	HelpfulCount   int            `json:"helpfulCount"`
	CompletedCount int            `json:"completedCount"`
	Difficulty     map[string]int `json:"difficulty"`
	Insight        string         `json:"insight"`
}

// ResourceRecommendationView suggests the next resource package to generate.
type ResourceRecommendationView struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	KnowledgeID   string   `json:"knowledgeId"`
	KnowledgeName string   `json:"knowledgeName"`
	CourseTitle   string   `json:"courseTitle"`
	Target        string   `json:"target"`
	ResourceTypes []string `json:"resourceTypes"`
	Reason        string   `json:"reason"`
	ActionText    string   `json:"actionText"`
	Priority      int      `json:"priority"`
}

// ResourceRecommendationsResponse returns feedback-aware recommendations.
type ResourceRecommendationsResponse struct {
	Summary         string                       `json:"summary"`
	FeedbackSummary FeedbackSummaryView          `json:"feedbackSummary"`
	List            []ResourceRecommendationView `json:"list"`
}

// SafetyCheckView describes one content-safety or hallucination check.
type SafetyCheckView struct {
	Key        string `json:"key"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// SafetyReportView returns a visible safety and anti-hallucination report.
type SafetyReportView struct {
	ResourceID   int64             `json:"resourceId"`
	Title        string            `json:"title"`
	Status       string            `json:"status"`
	Score        int               `json:"score"`
	Checks       []SafetyCheckView `json:"checks"`
	Risks        []string          `json:"risks"`
	Suggestions  []string          `json:"suggestions"`
	ReviewedAt   string            `json:"reviewedAt"`
	ReviewerName string            `json:"reviewerName"`
}

// ExportJobResponse returns an exportable Markdown package.
type ExportJobResponse struct {
	JobID       int64  `json:"jobId"`
	FileName    string `json:"fileName"`
	Content     string `json:"content"`
	GeneratedAt string `json:"generatedAt"`
}

// GenerateResourcesResponse returns the job and generated resources.
type GenerateResourcesResponse struct {
	Job       JobView                 `json:"job"`
	Resources []GeneratedResourceView `json:"resources"`
}

// ListResourcesResponse returns current student's resource library.
type ListResourcesResponse struct {
	List  []GeneratedResourceView `json:"list"`
	Total int                     `json:"total"`
}
