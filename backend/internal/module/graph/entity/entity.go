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
	CuratedSource    string
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
	Summary       string
	Prerequisites []string
	Related       []string
	Scenarios     []string
	ParentTopics  []string
}

// FrontierTopic 表示知识点关联的前沿主题。
type FrontierTopic struct {
	TopicID string
	Name    string
	Reason  string
}

// FrontierPaper 表示知识点关联的代表论文。
type FrontierPaper struct {
	PaperID string
	Title   string
	Venue   string
	Year    int
	URL     string
}

// FrontierKnowledgeRef 表示前沿资料关联到的课程知识点。
type FrontierKnowledgeRef struct {
	KnowledgeID string
	Name        string
	Course      string
	Reason      string
}

// FrontierLibraryItem 表示前沿资料中心中的一条论文或开源线索。
type FrontierLibraryItem struct {
	PaperID       string
	Title         string
	Abstract      string
	Venue         string
	Year          int
	URL           string
	Source        string
	UpdatedAt     string
	Topics        []FrontierTopic
	KnowledgeRefs []FrontierKnowledgeRef
}

// FrontierLibrary 表示前沿资料中心聚合结果。
type FrontierLibrary struct {
	GeneratedAt string
	Total       int
	Items       []FrontierLibraryItem
}

// FrontierBriefing 表示知识点的前沿扩展摘要。
type FrontierBriefing struct {
	KnowledgeID   string
	Status        string
	Knowledge     string
	Message       string
	DigestSummary string
	Topics        []FrontierTopic
	Papers        []FrontierPaper
	GeneratedAt   string
}

// CompetencyMapping 表示知识点映射到能力维度的权重。
type CompetencyMapping struct {
	KnowledgeID    string
	CompetencyID   string
	CompetencyName string
	Category       string
	Weight         float64
}
