package entity

import "time"

// CompetencyWeight 表示单个知识点映射到某个能力维度的权重。
type CompetencyWeight struct {
	CompetencyID   string
	CompetencyName string
	Category       string
	Weight         float64
}

// MatrixSnapshot 表示当前启用的能力矩阵快照。
type MatrixSnapshot struct {
	Version   string
	UpdatedAt time.Time
	Mappings  map[string][]CompetencyWeight
}

// CompetencyScore 表示用户在某个能力维度上的聚合分数。
type CompetencyScore struct {
	CompetencyID   string
	CompetencyName string
	Category       string
	Score          float64
}
