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

// RecordPracticeRequest 记录一次知识点级练习作答（不依赖题库题目），
// 供 AI 生成的随堂练习把答题结果接入真实掌握度链路。
type RecordPracticeRequest struct {
	KnowledgeID string `json:"knowledgeId" binding:"required,max=64"`
	Correct     bool   `json:"correct"`
}

// RecordPracticeResponse 返回练习作答后的掌握度更新结果。
type RecordPracticeResponse struct {
	Accepted     bool    `json:"accepted"`
	KnowledgeID  string  `json:"knowledgeId"`
	MasteryScore float64 `json:"masteryScore"`
	Degraded     bool    `json:"degraded"`
}

// SubmitQuizRequest 掌握度挑战批量提交。
// 包含完整题目数据，后端统一评分并记录掌握度，前端不可篡改对错。
type SubmitQuizRequest struct {
	KnowledgeID string           `json:"knowledgeId" binding:"required,max=64"`
	Answers     []QuizAnswerItem `json:"answers" binding:"required,min=1,max=20"`
	Questions   []QuizQuestion   `json:"questions" binding:"required,min=1,max=20"`
	AnsweredAt  string           `json:"answeredAt" binding:"required"`
}

// QuizQuestion 题目定义（前端传入，后端用于评分）。
type QuizQuestion struct {
	QuestionID    string   `json:"questionId" binding:"required,max=64"`
	Question      string   `json:"question" binding:"required"`
	Options       []string `json:"options" binding:"required,min=2,max=8"`
	CorrectOption int      `json:"correctOption"`
	Explanation   string   `json:"explanation"`
}

// QuizAnswerItem 单道答题。
type QuizAnswerItem struct {
	QuestionID     string `json:"questionId" binding:"required,max=64"`
	SelectedOption int    `json:"selectedOption"`
}

// SubmitQuizResponse 批量提交响应。
type SubmitQuizResponse struct {
	Results      []QuizResultItem `json:"results"`
	TotalCorrect int              `json:"totalCorrect"`
	Total        int              `json:"total"`
	MasteryDelta float64          `json:"masteryDelta"`
}

// QuizResultItem 单题评分结果。
type QuizResultItem struct {
	QuestionID  string `json:"questionId"`
	Correct     bool   `json:"correct"`
	CorrectIdx  int    `json:"correctIdx"`
	CorrectText string `json:"correctText"`
	Explanation string `json:"explanation"`
}
