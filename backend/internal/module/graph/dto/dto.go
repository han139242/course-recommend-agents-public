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

// SeedQuery defines seed-node graph queries for the homepage graph view.
type SeedQuery struct {
	Count int `form:"count" binding:"omitempty,min=1,max=30"`
}

// NeighborQuery defines on-demand neighbor expansion queries.
type NeighborQuery struct {
	NodeID    string `form:"nodeId" binding:"required,max=128"`
	Depth     int    `form:"depth" binding:"omitempty,min=1,max=2"`
	Direction string `form:"direction" binding:"omitempty,oneof=outgoing incoming both"`
	Limit     int    `form:"limit" binding:"omitempty,min=1,max=50"`
}

// PrerequisiteChainQuery defines multi-hop prerequisite chain queries.
type PrerequisiteChainQuery struct {
	Target string `form:"target" binding:"required,max=128"`
	Depth  int    `form:"depth" binding:"omitempty,min=1,max=4"`
}

// GraphPathQuery defines shortest-path graph reasoning queries.
type GraphPathQuery struct {
	From     string `form:"from" binding:"required,max=128"`
	To       string `form:"to" binding:"required,max=128"`
	MaxDepth int    `form:"maxDepth" binding:"omitempty,min=1,max=6"`
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

// CatalogNodeView represents lightweight graph metadata used by search and navigation.
type CatalogNodeView struct {
	ID               string `json:"id"`
	RouteKnowledgeID string `json:"routeKnowledgeId"`
	Name             string `json:"name"`
	CanonicalName    string `json:"canonicalName"`
	ChapterTitle     string `json:"chapterTitle"`
	EntityType       string `json:"entityType"`
	SourcePath       string `json:"sourcePath"`
	CuratedSource    string `json:"curatedSource"`
}

// PrerequisiteIndexEdgeView represents a lightweight prerequisite edge.
type PrerequisiteIndexEdgeView struct {
	SourceRouteKnowledgeID string `json:"sourceRouteKnowledgeId"`
	TargetRouteKnowledgeID string `json:"targetRouteKnowledgeId"`
}

// CatalogResponse is returned by the lightweight graph catalog endpoint.
type CatalogResponse struct {
	Nodes             []CatalogNodeView           `json:"nodes"`
	PrerequisiteEdges []PrerequisiteIndexEdgeView `json:"prerequisiteEdges"`
}

// PrerequisiteChainResponse represents a prerequisite chain with path order preserved.
type PrerequisiteChainResponse struct {
	Target         string          `json:"target"`
	Nodes          []GraphNodeView `json:"nodes"`
	Edges          []GraphEdgeView `json:"edges"`
	OrderedNodeIDs []string        `json:"orderedNodeIds"`
}

// GraphPathResponse represents a shortest path between two knowledge nodes.
type GraphPathResponse struct {
	Found          bool            `json:"found"`
	Nodes          []GraphNodeView `json:"nodes"`
	Edges          []GraphEdgeView `json:"edges"`
	OrderedNodeIDs []string        `json:"orderedNodeIds"`
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
