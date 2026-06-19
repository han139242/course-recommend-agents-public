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
