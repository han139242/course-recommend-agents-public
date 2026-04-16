package dto

// SubgraphQuery defines one-hop graph queries around a root node.
type SubgraphQuery struct {
	Root  string `form:"root" binding:"omitempty,max=128"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=50"`
}

// FullGraphQuery defines full graph queries for the homepage graph view.
type FullGraphQuery struct {
	Limit int `form:"limit" binding:"omitempty,min=1,max=2000"`
}

// GraphNodeView represents a knowledge graph node.
type GraphNodeView struct {
	ID               string `json:"id"`
	RouteKnowledgeID string `json:"routeKnowledgeId"`
	Name             string `json:"name"`
	CanonicalName    string `json:"canonicalName"`
	EntityType       string `json:"entityType"`
	Description      string `json:"description"`
	ChapterTitle     string `json:"chapterTitle"`
	SourcePath       string `json:"sourcePath"`
	CuratedSource    string `json:"curatedSource"`
}

// GraphEdgeView represents a relationship between two graph nodes.
type GraphEdgeView struct {
	RelationID   string `json:"relationId"`
	RelationType string `json:"relationType"`
	SourceID     string `json:"sourceId"`
	TargetID     string `json:"targetId"`
	SourceName   string `json:"sourceName"`
	TargetName   string `json:"targetName"`
}

// SubgraphResponse is returned by graph query endpoints.
type SubgraphResponse struct {
	Nodes []GraphNodeView `json:"nodes"`
	Edges []GraphEdgeView `json:"edges"`
}

// KnowledgeDetailView represents detail information for a knowledge node.
type KnowledgeDetailView struct {
	ID               string   `json:"id"`
	RouteKnowledgeID string   `json:"routeKnowledgeId"`
	Name             string   `json:"name"`
	CanonicalName    string   `json:"canonicalName"`
	EntityType       string   `json:"entityType"`
	Description      string   `json:"description"`
	ChapterTitle     string   `json:"chapterTitle"`
	SourcePath       string   `json:"sourcePath"`
	CuratedSource    string   `json:"curatedSource"`
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
