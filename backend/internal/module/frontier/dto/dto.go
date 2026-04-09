package dto

// BriefingQuery 定义前沿扩展查询参数。
type BriefingQuery struct {
	KnowledgeID string `form:"knowledgeId" binding:"omitempty,max=128"`
	Knowledge   string `form:"knowledge" binding:"omitempty,max=128"`
}

// TopicView 定义前沿主题视图。
type TopicView struct {
	TopicID string `json:"topicId"`
	Name    string `json:"name"`
	Reason  string `json:"reason"`
}

// PaperView 定义代表论文视图。
type PaperView struct {
	PaperID string `json:"paperId"`
	Title   string `json:"title"`
	Venue   string `json:"venue"`
	Year    int    `json:"year"`
	URL     string `json:"url"`
}

// BriefingResponse 定义前沿扩展响应。
type BriefingResponse struct {
	KnowledgeID    string      `json:"knowledgeId"`
	Status         string      `json:"status"`
	Knowledge      string      `json:"knowledge"`
	Message        string      `json:"message"`
	DigestSummary  string      `json:"digestSummary"`
	Topics         []TopicView `json:"topics"`
	Papers         []PaperView `json:"papers"`
	GeneratedAt    string      `json:"generatedAt"`
}
