package model

import "time"

// QuestionOption 表示题目选项模型。
type QuestionOption struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	QuestionID int64     `gorm:"column:question_id;not null;uniqueIndex:uk_question_options_question_option,priority:1;index:idx_question_options_question_id"`
	OptionKey  string    `gorm:"column:option_key;type:varchar(8);not null;uniqueIndex:uk_question_options_question_option,priority:2"`
	OptionText string    `gorm:"column:option_text;type:text;not null"`
	SortOrder  int       `gorm:"column:sort_order;not null;default:0"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

// TableName 指定 question_options 表名。
// 选项表通过固定表名承接唯一索引约束，避免自动复数化规则变化带来风险。
func (QuestionOption) TableName() string {
	return "question_options"
}
