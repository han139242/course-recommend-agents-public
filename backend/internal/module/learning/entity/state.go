package entity

import "time"

// MasteryState 表示单个知识点的实时学习状态。
type MasteryState struct {
	KnowledgeID    string
	MasteryScore   float64
	CorrectCount   int
	AnsweredCount  int
	LastAnsweredAt *time.Time
}
