package entity

// Node 表示图节点。
type Node struct {
	ID               string
	RouteKnowledgeID string
	Name             string
	CanonicalName    string
	EntityType       string
	Description      string
	ChapterTitle     string
	SourcePath       string
}

// Edge 表示图边。
type Edge struct {
	RelationID   string
	RelationType string
	SourceID     string
	TargetID     string
	SourceName   string
	TargetName   string
}

// KnowledgeDetail 表示知识点详情。
type KnowledgeDetail struct {
	Node
	Summary         string
	Prerequisites   []string
	Related         []string
	Scenarios       []string
	ParentTopics    []string
}
