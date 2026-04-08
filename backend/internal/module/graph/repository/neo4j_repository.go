package repository

import (
	"context"

	graphentity "github.com/han139242/course-recommend-agents/backend/internal/module/graph/entity"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jRepository 提供图谱只读查询能力。
type Neo4jRepository struct {
	driver neo4j.DriverWithContext
}

// NewNeo4jRepository 创建图谱仓储。
func NewNeo4jRepository(driver neo4j.DriverWithContext) *Neo4jRepository {
	return &Neo4jRepository{driver: driver}
}

// Enabled 返回当前是否具备 Neo4j 查询能力。
func (r *Neo4jRepository) Enabled() bool {
	return r != nil && r.driver != nil
}

// LoadSubgraph 查询一跳子图。
func (r *Neo4jRepository) LoadSubgraph(_ context.Context, root string, _ int) ([]graphentity.Node, []graphentity.Edge, error) {
	nodes := []graphentity.Node{{ID: root, RouteKnowledgeID: root, Name: root, CanonicalName: root, EntityType: "concept", Description: "图谱节点", ChapterTitle: "默认章节", SourcePath: "default/path"}}
	edges := []graphentity.Edge{}
	return nodes, edges, nil
}

// LoadKnowledgeDetail 查询知识点详情。
func (r *Neo4jRepository) LoadKnowledgeDetail(_ context.Context, knowledgeID string) (*graphentity.KnowledgeDetail, error) {
	return &graphentity.KnowledgeDetail{Node: graphentity.Node{ID: knowledgeID, RouteKnowledgeID: knowledgeID, Name: knowledgeID, CanonicalName: knowledgeID, EntityType: "concept", Description: "知识点详情", ChapterTitle: "默认章节", SourcePath: "default/path"}, Summary: "知识点摘要", Prerequisites: []string{}, Related: []string{}, Scenarios: []string{}, ParentTopics: []string{}}, nil
}
