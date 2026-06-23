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

// RetryJobRequest starts a new generation job from an existing job context.
// ResourceTypes and Difficulty may override the inferred values.
type RetryJobRequest struct {
	ResourceTypes []string `json:"resourceTypes" binding:"omitempty,max=10"`
	Difficulty    string   `json:"difficulty" binding:"omitempty,oneof=easier harder normal"`
	Target        string   `json:"target" binding:"omitempty,max=500"`
	Images        []string `json:"images" binding:"omitempty,max=4"`
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

// MediaStatusResponse exposes the normalized state of an image/video resource.
type MediaStatusResponse struct {
	ResourceID     int64  `json:"resourceId"`
	Kind           string `json:"kind"`
	Status         string `json:"status"`
	Ready          bool   `json:"ready"`
	URL            string `json:"url,omitempty"`
	B64JSON        string `json:"b64Json,omitempty"`
	MimeType       string `json:"mimeType,omitempty"`
	Prompt         string `json:"prompt,omitempty"`
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model,omitempty"`
	ProviderTaskID string `json:"providerTaskId,omitempty"`
	Hint           string `json:"hint"`
	CheckedAt      string `json:"checkedAt"`
}

// QualityMetricView describes one quality dimension in the resource scoring
// report.
type QualityMetricView struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

// ResourceQualityReportView is a lightweight automatic resource-quality
// assessment for demo comparison and review.
type ResourceQualityReportView struct {
	ResourceID   int64               `json:"resourceId"`
	Title        string              `json:"title"`
	TypeLabel    string              `json:"typeLabel"`
	Score        int                 `json:"score"`
	Level        string              `json:"level"`
	Metrics      []QualityMetricView `json:"metrics"`
	Strengths    []string            `json:"strengths"`
	Improvements []string            `json:"improvements"`
	GeneratedAt  string              `json:"generatedAt"`
}

// CompareResourcesRequest asks the system to compare several generated
// resources and recommend the best one for the current learning goal.
type CompareResourcesRequest struct {
	ResourceIDs []int64 `json:"resourceIds" binding:"required,min=2,max=6"`
}

// ResourceComparisonItemView is one row in the comparison result.
type ResourceComparisonItemView struct {
	Resource GeneratedResourceView     `json:"resource"`
	Quality  ResourceQualityReportView `json:"quality"`
	Reason   string                    `json:"reason"`
}

// CompareResourcesResponse returns a ranked comparison across resources.
type CompareResourcesResponse struct {
	WinnerID    int64                        `json:"winnerId"`
	Summary     string                       `json:"summary"`
	Items       []ResourceComparisonItemView `json:"items"`
	GeneratedAt string                       `json:"generatedAt"`
}

// JobReviewResourceView is one resource row in the task-level review summary.
type JobReviewResourceView struct {
	Resource GeneratedResourceView     `json:"resource"`
	Safety   SafetyReportView          `json:"safety"`
	Quality  ResourceQualityReportView `json:"quality"`
	Media    *MediaStatusResponse      `json:"media,omitempty"`
}

// JobReviewSummaryResponse aggregates safety, quality and media readiness for
// an entire generation job.
type JobReviewSummaryResponse struct {
	JobID             int64                   `json:"jobId"`
	Summary           string                  `json:"summary"`
	ResourceCount     int                     `json:"resourceCount"`
	AverageQuality    int                     `json:"averageQuality"`
	AverageSafety     int                     `json:"averageSafety"`
	MediaReadyCount   int                     `json:"mediaReadyCount"`
	MediaPendingCount int                     `json:"mediaPendingCount"`
	Risks             []string                `json:"risks"`
	Suggestions       []string                `json:"suggestions"`
	Resources         []JobReviewResourceView `json:"resources"`
	GeneratedAt       string                  `json:"generatedAt"`
}

// EvidencePackageResourceView packages one resource with its review evidence.
type EvidencePackageResourceView struct {
	Resource GeneratedResourceView     `json:"resource"`
	Safety   SafetyReportView          `json:"safety"`
	Quality  ResourceQualityReportView `json:"quality"`
	Media    *MediaStatusResponse      `json:"media,omitempty"`
}

// JobEvidencePackageResponse returns a JSON submission/evidence bundle for
// competition materials.
type JobEvidencePackageResponse struct {
	JobID       int64                         `json:"jobId"`
	FileName    string                        `json:"fileName"`
	Job         JobView                       `json:"job"`
	Steps       []AgentStepView               `json:"steps"`
	Resources   []EvidencePackageResourceView `json:"resources"`
	Checklist   []string                      `json:"checklist"`
	GeneratedAt string                        `json:"generatedAt"`
}

// StudyChecklistItemView is one actionable learner step derived from a
// generated resource package.
type StudyChecklistItemView struct {
	Index      int    `json:"index"`
	ResourceID int64  `json:"resourceId,omitempty"`
	Title      string `json:"title"`
	Action     string `json:"action"`
	Expected   string `json:"expected"`
	Status     string `json:"status"`
}

// StudyChecklistResponse gives students an ordered resource-use plan.
type StudyChecklistResponse struct {
	JobID       int64                    `json:"jobId"`
	Summary     string                   `json:"summary"`
	Items       []StudyChecklistItemView `json:"items"`
	GeneratedAt string                   `json:"generatedAt"`
}

// ResourceVariantRequest asks the system to generate one improved variant from
// a specific resource.
type ResourceVariantRequest struct {
	Mode       string `json:"mode" binding:"omitempty,oneof=simplify deepen example quiz visual"`
	Target     string `json:"target" binding:"omitempty,max=500"`
	Difficulty string `json:"difficulty" binding:"omitempty,oneof=easier harder normal"`
}

// ResourceProvenanceResponse exposes the generation trace for one resource.
type ResourceProvenanceResponse struct {
	Resource    GeneratedResourceView `json:"resource"`
	Job         JobView               `json:"job"`
	Steps       []AgentStepView       `json:"steps"`
	Events      []LearningEventView   `json:"events"`
	Summary     string                `json:"summary"`
	GeneratedAt string                `json:"generatedAt"`
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

// EffectMetricView is one headline metric on the learning-effect dashboard.
type EffectMetricView struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value"`
	Hint  string `json:"hint,omitempty"`
}

// EffectSummaryResponse aggregates resource usage and feedback into a learning
// effect-assessment dashboard for the A3 workbench.
type EffectSummaryResponse struct {
	Metrics      []EffectMetricView  `json:"metrics"`
	Difficulty   map[string]int      `json:"difficulty"`
	Highlights   []string            `json:"highlights"`
	RecentEvents []LearningEventView `json:"recentEvents,omitempty"`
	GeneratedAt  string              `json:"generatedAt"`
}

// LearningReportResponse returns a shareable Markdown 学情/学习报告 aggregating
// the learner profile, mastery-relevant resources, feedback and effect metrics.
type LearningReportResponse struct {
	FileName    string `json:"fileName"`
	Content     string `json:"content"`
	GeneratedAt string `json:"generatedAt"`
}

// GenerateQuizRequest asks for an interactive single-choice quiz on a knowledge
// point so the learner can self-test and feed the mastery loop.
type GenerateQuizRequest struct {
	KnowledgeID   string `json:"knowledgeId" binding:"omitempty,max=128"`
	KnowledgeName string `json:"knowledgeName" binding:"required,max=255"`
	CourseTitle   string `json:"courseTitle" binding:"omitempty,max=255"`
	Count         int    `json:"count" binding:"omitempty,min=1,max=10"`
}

// QuizQuestionView is one structured single-choice question.
type QuizQuestionView struct {
	Stem        string   `json:"stem"`
	Options     []string `json:"options"`
	Answer      int      `json:"answer"`
	Explanation string   `json:"explanation"`
}

// GenerateQuizResponse returns the generated quiz and whether it came from the
// large model or the local template fallback.
type GenerateQuizResponse struct {
	KnowledgeID   string             `json:"knowledgeId"`
	KnowledgeName string             `json:"knowledgeName"`
	CourseTitle   string             `json:"courseTitle"`
	Questions     []QuizQuestionView `json:"questions"`
	Source        string             `json:"source"`
}

// GenerateMediaRequest asks a real media-generation model to create an image or
// video grounded in the current learning context.
type GenerateMediaRequest struct {
	KnowledgeID   string `json:"knowledgeId" binding:"omitempty,max=128"`
	KnowledgeName string `json:"knowledgeName" binding:"required,max=255"`
	CourseTitle   string `json:"courseTitle" binding:"omitempty,max=255"`
	Target        string `json:"target" binding:"omitempty,max=500"`
	Prompt        string `json:"prompt" binding:"omitempty,max=1000"`
	Size          string `json:"size" binding:"omitempty,max=32"`
	DurationSec   int    `json:"durationSec" binding:"omitempty,min=0,max=30"`
}

// MediaGenerationResponse returns the persisted resource plus normalized media
// fields. URL and B64JSON are both optional because some providers return an
// async task id first.
type MediaGenerationResponse struct {
	Kind           string                `json:"kind"`
	Resource       GeneratedResourceView `json:"resource"`
	URL            string                `json:"url,omitempty"`
	B64JSON        string                `json:"b64Json,omitempty"`
	MimeType       string                `json:"mimeType,omitempty"`
	Prompt         string                `json:"prompt"`
	Provider       string                `json:"provider"`
	Model          string                `json:"model"`
	Status         string                `json:"status"`
	ProviderTaskID string                `json:"providerTaskId,omitempty"`
	GeneratedAt    string                `json:"generatedAt"`
}

// ListResourcesResponse returns current student's resource library.
type ListResourcesResponse struct {
	List  []GeneratedResourceView `json:"list"`
	Total int                     `json:"total"`
}
