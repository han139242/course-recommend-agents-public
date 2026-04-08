package model

import "time"

// StudentAnswer 表示学生答题记录模型。
type StudentAnswer struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement"`
	StudentID      int64     `gorm:"column:student_id;not null;index:idx_student_answers_student_answered,priority:1;index:idx_student_answers_student_knowledge,priority:1"`
	QuestionCode   string    `gorm:"column:question_code;type:varchar(64);not null;index:idx_student_answers_question_answered,priority:1"`
	KnowledgeID    string    `gorm:"column:knowledge_id;type:varchar(64);not null;index:idx_student_answers_student_knowledge,priority:2"`
	SelectedOption string    `gorm:"column:selected_option;type:varchar(8);not null"`
	IsCorrect      bool      `gorm:"column:is_correct;not null"`
	DurationMs     *int      `gorm:"column:duration_ms"`
	AnsweredAt     time.Time `gorm:"column:answered_at;not null;index:idx_student_answers_student_answered,priority:2;index:idx_student_answers_question_answered,priority:2"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (StudentAnswer) TableName() string {
	return "student_answers"
}
