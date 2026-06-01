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

// KnowledgeRefView 定义前沿资料关联知识点视图。
type KnowledgeRefView struct {
	KnowledgeID string `json:"knowledgeId"`
	Name        string `json:"name"`
	Course      string `json:"course"`
	Reason      string `json:"reason"`
}

// LibraryItemView 定义前沿资料中心条目视图。
type LibraryItemView struct {
	PaperID       string             `json:"paperId"`
	Title         string             `json:"title"`
	Abstract      string             `json:"abstract"`
	Venue         string             `json:"venue"`
	Year          int                `json:"year"`
	URL           string             `json:"url"`
	Source        string             `json:"source"`
	UpdatedAt     string             `json:"updatedAt"`
	Topics        []TopicView        `json:"topics"`
	KnowledgeRefs []KnowledgeRefView `json:"knowledgeRefs"`
}

// LibraryQuery 定义前沿资料中心查询参数。
type LibraryQuery struct {
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

// LibraryResponse 定义前沿资料中心响应。
type LibraryResponse struct {
	Status      string            `json:"status"`
	Message     string            `json:"message"`
	GeneratedAt string            `json:"generatedAt"`
	Total       int               `json:"total"`
	Items       []LibraryItemView `json:"items"`
}

// BriefingResponse 定义前沿扩展响应。
type BriefingResponse struct {
	KnowledgeID   string      `json:"knowledgeId"`
	Status        string      `json:"status"`
	Knowledge     string      `json:"knowledge"`
	Message       string      `json:"message"`
	DigestSummary string      `json:"digestSummary"`
	Topics        []TopicView `json:"topics"`
	Papers        []PaperView `json:"papers"`
	GeneratedAt   string      `json:"generatedAt"`
}
