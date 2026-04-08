package dto

// StatItem 定义画像统计项。
type StatItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// RadarAxis 定义雷达图轴。
type RadarAxis struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// InsightItem 定义成长洞察项。
type InsightItem struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Note  string  `json:"note"`
}

// SummaryCard 定义成长摘要卡片。
type SummaryCard struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// HeatmapRow 定义热力图行。
type HeatmapRow struct {
	Label  string    `json:"label"`
	Values []float64 `json:"values"`
}

// ProfileView 定义用户画像头部视图。
type ProfileView struct {
	Name          string     `json:"name"`
	Role          string     `json:"role"`
	ClassID       *int64     `json:"classId"`
	ClassName     *string    `json:"className"`
	Target        string     `json:"target"`
	Summary       string     `json:"summary"`
	Stats         []StatItem `json:"stats"`
	Goals         []string   `json:"goals"`
	RecentRecords []string   `json:"recentRecords"`
}

// ProfileResponse 定义能力画像响应。
type ProfileResponse struct {
	Profile            ProfileView    `json:"profile"`
	RadarAxes          []RadarAxis    `json:"radarAxes"`
	GrowthInsights     []InsightItem  `json:"growthInsights"`
	GrowthSummaryCards []SummaryCard  `json:"growthSummaryCards"`
	HeatmapDomains     []string       `json:"heatmapDomains"`
	HeatmapRows        []HeatmapRow   `json:"heatmapRows"`
}
