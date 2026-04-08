package dto

// PaginationQuery 定义通用分页参数。
type PaginationQuery struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// Normalize 补齐默认分页参数。
func (q PaginationQuery) Normalize() PaginationQuery {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	return q
}

// ImportQuestionsRequest 定义批量导题请求。
type ImportQuestionsRequest struct {
	Items []ImportQuestionItem `json:"items" binding:"required,min=1,max=100"`
}

// ImportQuestionItem 定义单题导入项。
type ImportQuestionItem struct {
	QuestionID    string              `json:"questionId" binding:"omitempty,max=64"`
	Type          string              `json:"type" binding:"required,max=32"`
	KnowledgeID   string              `json:"knowledgeId" binding:"required,max=64"`
	Stem          string              `json:"stem" binding:"required"`
	Options       []QuestionOptionDTO `json:"options" binding:"required,min=2,max=8"`
	CorrectOption string              `json:"correctOption" binding:"required,max=8"`
	Difficulty    float64             `json:"difficulty" binding:"required"`
	Explanation   string              `json:"explanation" binding:"omitempty"`
}

// QuestionOptionDTO 定义题目选项。
type QuestionOptionDTO struct {
	Key  string `json:"key" binding:"required,max=8"`
	Text string `json:"text" binding:"required"`
}

// ImportQuestionsResponse 定义批量导题响应。
type ImportQuestionsResponse struct {
	CreatedCount int `json:"createdCount"`
}

// ListQuestionsQuery 定义教师题目列表查询参数。
type ListQuestionsQuery struct {
	KnowledgeID string `form:"knowledgeId" binding:"omitempty,max=64"`
	Type        string `form:"type" binding:"omitempty,max=32"`
	Keyword     string `form:"keyword" binding:"omitempty,max=255"`
	PaginationQuery
}

// StudentQuestionsQuery 定义学生拉题查询参数。
type StudentQuestionsQuery struct {
	KnowledgeID string `form:"knowledgeId" binding:"omitempty,max=64"`
	PaginationQuery
}

// TeacherQuestionView 定义教师端题目视图。
type TeacherQuestionView struct {
	QuestionID    string              `json:"questionId"`
	Type          string              `json:"type"`
	KnowledgeID   string              `json:"knowledgeId"`
	Stem          string              `json:"stem"`
	Options       []QuestionOptionDTO `json:"options"`
	CorrectOption string              `json:"correctOption"`
	Difficulty    float64             `json:"difficulty"`
	Explanation   string              `json:"explanation"`
}

// StudentQuestionView 定义学生端题目视图。
type StudentQuestionView struct {
	QuestionID  string              `json:"questionId"`
	Type        string              `json:"type"`
	KnowledgeID string              `json:"knowledgeId"`
	Stem        string              `json:"stem"`
	Options     []QuestionOptionDTO `json:"options"`
	Difficulty  float64             `json:"difficulty"`
}

// QuestionDetailView 定义单题详情。
type QuestionDetailView struct {
	QuestionID    string              `json:"questionId"`
	Type          string              `json:"type"`
	KnowledgeID   string              `json:"knowledgeId"`
	Stem          string              `json:"stem"`
	Options       []QuestionOptionDTO `json:"options"`
	CorrectOption string              `json:"correctOption"`
	Difficulty    float64             `json:"difficulty"`
	Explanation   string              `json:"explanation"`
	CreatedBy     int64               `json:"createdBy"`
	CreatedAt     string              `json:"createdAt"`
}

// TeacherListQuestionsResponse 定义教师端题目列表响应。
type TeacherListQuestionsResponse struct {
	List     []TeacherQuestionView `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

// StudentListQuestionsResponse 定义学生端题目列表响应。
type StudentListQuestionsResponse struct {
	List     []StudentQuestionView `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}
