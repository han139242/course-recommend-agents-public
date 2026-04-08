package dto

// SubmitAnswerRequest 定义答题提交请求。
type SubmitAnswerRequest struct {
	QuestionID     string `json:"questionId" binding:"required,max=64"`
	KnowledgeID    string `json:"knowledgeId" binding:"required,max=64"`
	SelectedOption string `json:"selectedOption" binding:"required,max=8"`
	DurationMs     *int   `json:"durationMs" binding:"omitempty,min=0"`
	AnsweredAt     string `json:"answeredAt" binding:"required"`
}

// SubmitAnswerResponse 定义答题提交响应。
type SubmitAnswerResponse struct {
	Accepted    bool   `json:"accepted"`
	QuestionID  string `json:"questionId"`
	KnowledgeID string `json:"knowledgeId"`
	Degraded    bool   `json:"degraded"`
}
