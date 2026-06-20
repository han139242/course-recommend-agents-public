package dto

// GenerateResourcesRequest creates a personalized resource package.
type GenerateResourcesRequest struct {
	KnowledgeID   string   `json:"knowledgeId" binding:"omitempty,max=128"`
	KnowledgeName string   `json:"knowledgeName" binding:"omitempty,max=255"`
	CourseTitle   string   `json:"courseTitle" binding:"omitempty,max=255"`
	Target        string   `json:"target" binding:"omitempty,max=500"`
	ResourceTypes []string `json:"resourceTypes" binding:"omitempty,max=10"`
	// Difficulty optionally regenerates resources at an easier or harder level.
	Difficulty string `json:"difficulty" binding:"omitempty,oneof=easier harder normal"`
	// Images carries optional learner-uploaded images (data URLs or remote URLs)
	// that the VisionReaderAgent reads to ground generation.
	Images []string `json:"images" binding:"omitempty,max=4"`
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
	ID            int64                  `json:"id"`
	JobID         int64                  `json:"jobId"`
	ResourceType  string                 `json:"resourceType"`
	TypeLabel     string                 `json:"typeLabel"`
	Title         string                 `json:"title"`
	Content       string                 `json:"content"`
	Format        string                 `json:"format"`
	KnowledgeID   string                 `json:"knowledgeId"`
	KnowledgeName string                 `json:"knowledgeName"`
	CourseTitle   string                 `json:"courseTitle"`
	Reason        string                 `json:"reason"`
	Status        string                 `json:"status"`
	CreatedAt     string                 `json:"createdAt"`
	Evidence      []ResourceEvidenceView `json:"evidence,omitempty"`
}

// ResourceEvidenceView explains which context grounded a generated resource.
type ResourceEvidenceView struct {
	Label string `json:"label"`
	Value string `json:"value"`
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

// RecordLearningEventRequest records one resource or learning behavior signal.
type RecordLearningEventRequest struct {
	EventType     string         `json:"eventType" binding:"required,max=64"`
	KnowledgeID   string         `json:"knowledgeId" binding:"omitempty,max=128"`
	KnowledgeName string         `json:"knowledgeName" binding:"omitempty,max=255"`
	ResourceID    int64          `json:"resourceId" binding:"omitempty,min=0"`
	ResourceType  string         `json:"resourceType" binding:"omitempty,max=64"`
	Payload       map[string]any `json:"payload"`
}

// LearningEventView exposes behavior signals for profile timeline and demos.
type LearningEventView struct {
	ID            int64          `json:"id"`
	EventType     string         `json:"eventType"`
	KnowledgeID   string         `json:"knowledgeId"`
	KnowledgeName string         `json:"knowledgeName"`
	ResourceID    int64          `json:"resourceId,omitempty"`
	ResourceType  string         `json:"resourceType,omitempty"`
	Payload       map[string]any `json:"payload,omitempty"`
	OccurredAt    string         `json:"occurredAt"`
}

// ResourcePushItemView is one personalized resource feed item.
type ResourcePushItemView struct {
	ID            int64                 `json:"id"`
	KnowledgeID   string                `json:"knowledgeId"`
	KnowledgeName string                `json:"knowledgeName"`
	CourseTitle   string                `json:"courseTitle"`
	Stage         string                `json:"stage"`
	Reason        string                `json:"reason"`
	Priority      int                   `json:"priority"`
	Status        string                `json:"status"`
	ActionText    string                `json:"actionText"`
	CreatedAt     string                `json:"createdAt"`
	Resource      GeneratedResourceView `json:"resource"`
}

// ResourceFeedResponse returns the personalized pushed-resource stream.
type ResourceFeedResponse struct {
	Summary      string                 `json:"summary"`
	Items        []ResourcePushItemView `json:"items"`
	RecentEvents []LearningEventView    `json:"recentEvents,omitempty"`
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

// PresentationSlideView describes one generated presentation slide.
type PresentationSlideView struct {
	Index        int      `json:"index"`
	Title        string   `json:"title"`
	Bullets      []string `json:"bullets"`
	SpeakerNotes string   `json:"speakerNotes"`
	VisualHint   string   `json:"visualHint"`
}

// PresentationOutlineResponse returns a PPT-ready outline for a resource job.
type PresentationOutlineResponse struct {
	JobID       int64                   `json:"jobId"`
	Title       string                  `json:"title"`
	Subtitle    string                  `json:"subtitle"`
	Slides      []PresentationSlideView `json:"slides"`
	GeneratedAt string                  `json:"generatedAt"`
}

// DemoScriptSceneView describes one timed scene for the competition demo video.
type DemoScriptSceneView struct {
	Index       int      `json:"index"`
	TimeRange   string   `json:"timeRange"`
	Title       string   `json:"title"`
	Actions     []string `json:"actions"`
	Narration   string   `json:"narration"`
	ProofPoints []string `json:"proofPoints"`
}

// DemoScriptResponse returns a video-ready 7-minute demo script.
type DemoScriptResponse struct {
	JobID       int64                 `json:"jobId"`
	Title       string                `json:"title"`
	Duration    string                `json:"duration"`
	Scenes      []DemoScriptSceneView `json:"scenes"`
	Checklist   []string              `json:"checklist"`
	GeneratedAt string                `json:"generatedAt"`
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
