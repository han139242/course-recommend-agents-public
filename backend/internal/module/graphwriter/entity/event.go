package entity

import "time"

// CompetencyRelation 表示用户与单个能力节点之间的能力分数快照。
type CompetencyRelation struct {
	CompetencyID   string
	CompetencyName string
	Category       string
	Score          float64
}

// ProfileIdentity 表示图谱写入所需的最小用户身份信息。
type ProfileIdentity struct {
	Name   string
	Target string
}

// ProfileUpdateEvent 表示一次待异步写入图谱的用户画像更新事件。
type ProfileUpdateEvent struct {
	StudentID       int64
	Identity        ProfileIdentity
	UpdatedAt       time.Time
	TraceID         string
	Source          string
	Relations       []CompetencyRelation
	TopologyVersion string
}
