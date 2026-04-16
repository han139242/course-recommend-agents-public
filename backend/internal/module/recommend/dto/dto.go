package dto

// PathQuery 定义推荐路径查询参数。
type PathQuery struct {
	Target      string `form:"target" binding:"omitempty,max=128"`
	KnowledgeID string `form:"knowledgeId" binding:"omitempty,max=128"`
	Limit       int    `form:"limit" binding:"omitempty,min=1,max=20"`
	Scene       string `form:"scene" binding:"omitempty,max=64"`
}

// PathItemView 定义推荐路径项。
type PathItemView struct {
	Stage         string  `json:"stage"`
	Title         string  `json:"title"`
	Detail        string  `json:"detail"`
	Route         string  `json:"route"`
	KnowledgeID   string  `json:"knowledgeId"`
	ReasonCode    string  `json:"reasonCode"`
	ReasonLabel   string  `json:"reasonLabel"`
	MasteryScore  float64 `json:"masteryScore"`
	CorrectRate   float64 `json:"correctRate"`
	AnsweredCount int     `json:"answeredCount"`
	Score         float64 `json:"score"`
}

// PathResponse 定义推荐路径响应。
type PathResponse struct {
	Path []PathItemView `json:"path"`
}
