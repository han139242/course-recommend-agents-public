package model

import "time"

// LearningProgress 表示学生知识点掌握度落盘模型。
type LearningProgress struct {
	ID             int64      `gorm:"column:id;primaryKey;autoIncrement"`
	StudentID      int64      `gorm:"column:student_id;not null;uniqueIndex:uk_learning_progress_student_knowledge,priority:1"`
	KnowledgeID    string     `gorm:"column:knowledge_id;type:varchar(64);not null;uniqueIndex:uk_learning_progress_student_knowledge,priority:2"`
	CanonicalName  string     `gorm:"column:canonical_name;type:varchar(255);not null"`
	MasteryScore   float64    `gorm:"column:mastery_score;type:decimal(5,2);not null;default:0"`
	Status         string     `gorm:"column:status;type:varchar(16);not null;default:'not_started'"`
	CorrectRate    float64    `gorm:"column:correct_rate;type:decimal(5,4);not null;default:0"`
	AnsweredCount  int        `gorm:"column:answered_count;not null;default:0"`
	LastAnsweredAt *time.Time `gorm:"column:last_answered_at"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName 指定 learning_progress 表名。
// 学习进度表依赖唯一键维护每个学生在每个知识点上的单行聚合记录。
func (LearningProgress) TableName() string {
	return "learning_progress"
}
