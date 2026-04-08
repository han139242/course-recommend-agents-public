package entity

import "time"

// LearningProgress 表示知识点掌握度视图。
type LearningProgress struct {
	KnowledgeID   string
	CanonicalName string
	MasteryScore  float64
	Status        string
	CorrectRate   float64
	AnsweredCount int
	LastAnsweredAt *time.Time
}
