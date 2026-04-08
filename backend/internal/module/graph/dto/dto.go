package dto

// SubgraphQuery 定义图谱子图查询参数。
type SubgraphQuery struct {
	Root  string `form:"root" binding:"omitempty,max=128"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=50"`
}

// GraphNodeView 定义图节点视图。
type GraphNodeView struct {
	ID               string `json:"id"`
	RouteKnowledgeID string `json:"routeKnowledgeId"`
	Name             string `json:"name"`
	CanonicalName    string `json:"canonicalName"`
	EntityType       string `json:"entityType"`
	Description      string `json:"description"`
	ChapterTitle     string `json:"chapterTitle"`
	SourcePath       string `json:"sourcePath"`
}

// GraphEdgeView 定义图边视图。
type GraphEdgeView struct {
	RelationID   string `json:"relationId"`
	RelationType string `json:"relationType"`
	SourceID     string `json:"sourceId"`
	TargetID     string `json:"targetId"`
	SourceName   string `json:"sourceName"`
	TargetName   string `json:"targetName"`
}

// SubgraphResponse 定义子图响应。
type SubgraphResponse struct {
	Nodes []GraphNodeView `json:"nodes"`
	Edges []GraphEdgeView `json:"edges"`
}

// KnowledgeDetailView 定义知识点详情视图。
type KnowledgeDetailView struct {
	ID               string   `json:"id"`
	RouteKnowledgeID string   `json:"routeKnowledgeId"`
	Name             string   `json:"name"`
	CanonicalName    string   `json:"canonicalName"`
	EntityType       string   `json:"entityType"`
	Description      string   `json:"description"`
	ChapterTitle     string   `json:"chapterTitle"`
	SourcePath       string   `json:"sourcePath"`
	Summary          string   `json:"summary"`
	Prerequisites    []string `json:"prerequisites"`
	Related          []string `json:"related"`
	Scenarios        []string `json:"scenarios"`
	ParentTopics     []string `json:"parentTopics"`
	MasteryScore     float64  `json:"masteryScore"`
	Status           string   `json:"status"`
	MasteryText      string   `json:"masteryText"`
	RecommendReason  string   `json:"recommendReason"`
}
