package model

import "time"

// Question 表示题目主表模型。
type Question struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	QuestionCode  string    `gorm:"column:question_code;type:varchar(64);not null;uniqueIndex:uk_questions_question_code"`
	Type          string    `gorm:"column:type;type:varchar(32);not null"`
	KnowledgeID   string    `gorm:"column:knowledge_id;type:varchar(64);not null;index:idx_questions_knowledge_id"`
	Stem          string    `gorm:"column:stem;type:text;not null"`
	CorrectOption string    `gorm:"column:correct_option;type:varchar(8);not null"`
	Difficulty    float64   `gorm:"column:difficulty;type:decimal(4,2);not null"`
	Explanation   *string   `gorm:"column:explanation;type:text"`
	Status        string    `gorm:"column:status;type:varchar(16);not null;default:'active';index:idx_questions_status"`
	CreatedBy     int64     `gorm:"column:created_by;not null;index:idx_questions_created_by"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Question) TableName() string {
	return "questions"
}
