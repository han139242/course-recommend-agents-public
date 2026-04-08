package dto

// LearningProgressItem 定义知识点掌握状态项。
type LearningProgressItem struct {
	KnowledgeID    string  `json:"knowledgeId"`
	CanonicalName  string  `json:"canonicalName"`
	MasteryScore   float64 `json:"masteryScore"`
	Status         string  `json:"status"`
	CorrectRate    float64 `json:"correctRate"`
	AnsweredCount  int     `json:"answeredCount"`
	LastAnsweredAt string  `json:"lastAnsweredAt"`
}

// LearningProgressResponse 定义学习进度响应。
type LearningProgressResponse struct {
	List []LearningProgressItem `json:"list"`
}
