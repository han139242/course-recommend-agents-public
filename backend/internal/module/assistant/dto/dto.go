package dto

// ChatRequest defines the authenticated project assistant request payload.
type ChatRequest struct {
	Messages  []map[string]any `json:"messages" binding:"required,min=1"`
	MaxTokens int              `json:"maxTokens" binding:"omitempty,min=1,max=2048"`
}

// ClientAction represents a UI action that should be executed by the frontend.
type ClientAction struct {
	Type        string         `json:"type"`
	Target      string         `json:"target,omitempty"`
	CourseTitle string         `json:"courseTitle,omitempty"`
	Flash       bool           `json:"flash,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}

// ChatResponse contains the final assistant content plus optional UI actions.
type ChatResponse struct {
	Content    string         `json:"content"`
	Actions    []ClientAction `json:"actions,omitempty"`
	Agent      string         `json:"agent,omitempty"`
	AgentLabel string         `json:"agentLabel,omitempty"`
}

// KnowledgeCardRequest defines the personalized knowledge card generation request.
type KnowledgeCardRequest struct {
	KnowledgeID   string   `json:"knowledgeId" binding:"required,max=128"`
	Name          string   `json:"name" binding:"omitempty,max=255"`
	Domain        string   `json:"domain" binding:"omitempty,max=255"`
	LevelLabel    string   `json:"levelLabel" binding:"omitempty,max=64"`
	Description   string   `json:"description" binding:"omitempty"`
	Mastery       float64  `json:"mastery" binding:"omitempty"`
	Prerequisites []string `json:"prerequisites" binding:"omitempty,max=20"`
	Related       []string `json:"related" binding:"omitempty,max=20"`
	Scenarios     []string `json:"scenarios" binding:"omitempty,max=20"`
}

// KnowledgeCardResponse defines the personalized knowledge card response.
type KnowledgeCardResponse struct {
	Content string   `json:"content"`
	Sources []string `json:"sources"`
}
