package dto

import competencydto "github.com/han139242/course-recommend-agents/backend/internal/module/competency/dto"
import recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"

// KnowledgeCardView 定义首页知识卡片。
type KnowledgeCardView struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	Progress float64 `json:"progress"`
	Title    string  `json:"title"`
	Domain   string  `json:"domain"`
	Tag      string  `json:"tag"`
	Route    string  `json:"route"`
}

// SummaryResponse 定义仪表盘摘要响应。
type SummaryResponse struct {
	QuickStats      []competencydto.StatItem    `json:"quickStats"`
	Stats           []competencydto.StatItem    `json:"stats"`
	KnowledgeCards  []KnowledgeCardView         `json:"knowledgeCards"`
	RecommendedPath []recommenddto.PathItemView `json:"recommendedPath"`
}
