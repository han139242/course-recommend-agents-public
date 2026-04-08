package entity

import "time"

const (
	QuestionTypeSingleChoice = "single_choice"
	QuestionStatusActive     = "active"
	QuestionStatusDisabled   = "disabled"
	UserRoleStudent          = "student"
	LearningStatusNotStarted = "not_started"
)

// QuestionOption 表示题目选项。
type QuestionOption struct {
	Key  string
	Text string
}

// Question 表示题目领域实体。
type Question struct {
	ID            int64
	QuestionID    string
	Type          string
	KnowledgeID   string
	Stem          string
	Options       []QuestionOption
	CorrectOption string
	Difficulty    float64
	Explanation   string
	CreatedBy     int64
	Status        string
	CreatedAt     time.Time
}

// ImportQuestionInput 表示导题输入。
type ImportQuestionInput struct {
	QuestionID    string
	Type          string
	KnowledgeID   string
	Stem          string
	Options       []QuestionOption
	CorrectOption string
	Difficulty    float64
	Explanation   string
	CreatedBy     int64
}

// QuestionFilter 表示题目查询条件。
type QuestionFilter struct {
	KnowledgeID string
	Type        string
	Keyword     string
	Pagination  Pagination
}

// StudentQuestionFilter 表示学生端拉题查询条件。
type StudentQuestionFilter struct {
	StudentID    int64
	KnowledgeID  string
	Pagination   Pagination
}

// SubmitAnswerInput 表示答题提交输入。
type SubmitAnswerInput struct {
	StudentID       int64
	QuestionID      string
	KnowledgeID     string
	SelectedOption  string
	DurationMs      *int
	AnsweredAt      time.Time
}

// SubmitAnswerResult 表示答题处理结果。
type SubmitAnswerResult struct {
	Accepted    bool
	QuestionID  string
	KnowledgeID string
	Degraded    bool
}

// Pagination 定义分页参数。
type Pagination struct {
	Page     int
	PageSize int
}

// Offset 返回分页偏移量。
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}
