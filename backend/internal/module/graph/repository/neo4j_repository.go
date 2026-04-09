package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	graphentity "github.com/han139242/course-recommend-agents/backend/internal/module/graph/entity"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ErrFrontierBriefingNotReady 表示前沿扩展内容尚未就绪。
var ErrFrontierBriefingNotReady = errors.New("前沿扩展内容尚未就绪")

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

func stringOrDefault(value any, fallback string) string {
	if value == nil {
		return fallback
	}
	if typed, ok := value.(string); ok && strings.TrimSpace(typed) != "" {
		return strings.TrimSpace(typed)
	}
	return fallback
}

func formatGraphTime(value any) string {
	switch typed := value.(type) {
	case time.Time:
		return typed.UTC().Format(time.RFC3339)
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func parseFrontierTopics(value any) []graphentity.FrontierTopic {
	items, ok := value.([]any)
	if !ok {
		return []graphentity.FrontierTopic{}
	}
	result := make([]graphentity.FrontierTopic, 0, len(items))
	for _, item := range items {
		mapped, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, graphentity.FrontierTopic{TopicID: stringOrDefault(mapped["topic_id"], ""), Name: stringOrDefault(mapped["name"], ""), Reason: stringOrDefault(mapped["reason"], "")})
	}
	return result
}

func parseFrontierPapers(value any) []graphentity.FrontierPaper {
	items, ok := value.([]any)
	if !ok {
		return []graphentity.FrontierPaper{}
	}
	result := make([]graphentity.FrontierPaper, 0, len(items))
	for _, item := range items {
		mapped, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, graphentity.FrontierPaper{PaperID: stringOrDefault(mapped["paper_id"], ""), Title: stringOrDefault(mapped["title"], ""), Venue: stringOrDefault(mapped["venue"], ""), Year: intOrDefault(mapped["year"]), URL: stringOrDefault(mapped["url"], "")})
	}
	return result
}

func intOrDefault(value any) int {
	switch typed := value.(type) {
	case int64:
		return int(typed)
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

// LoadFrontierBriefing 查询知识点的前沿扩展摘要。
func (r *Neo4jRepository) LoadFrontierBriefing(ctx context.Context, knowledgeID string, knowledgeName string) (*graphentity.FrontierBriefing, error) {
	resolvedID := strings.TrimSpace(knowledgeID)
	resolvedName := strings.TrimSpace(knowledgeName)
	if resolvedID == "" && resolvedName == "" {
		return nil, ErrFrontierBriefingNotReady
	}
	if !r.Enabled() {
		return nil, ErrFrontierBriefingNotReady
	}
	if resolvedID == "" {
		resolvedID = resolvedName
	}
	if resolvedName == "" {
		resolvedName = resolvedID
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
MATCH (k:KnowledgeNode)
WHERE ($knowledge_id <> '' AND k.knowledge_id = $knowledge_id) OR ($knowledge_id = '' AND k.name = $knowledge_name)
OPTIONAL MATCH (d:FrontierDigest)-[:CURATED_FOR]->(k)
OPTIONAL MATCH (topic:ResearchTopic)-[:EXTENDS_KNOWLEDGE]->(k)
OPTIONAL MATCH (d)-[:SUMMARIZES_PAPER]->(paper:Paper)
RETURN k.knowledge_id AS knowledge_id,
       coalesce(k.name, $knowledge_name, $knowledge_id) AS knowledge_name,
       d.summary AS digest_summary,
       d.status AS digest_status,
       d.generated_at AS generated_at,
       collect(DISTINCT topic{.topic_id, .name, reason: coalesce(topic.summary, '')}) AS topics,
       collect(DISTINCT paper{.paper_id, .title, .venue, .year, .url}) AS papers
LIMIT 1
`
		records, runErr := tx.Run(ctx, query, map[string]any{"knowledge_id": resolvedID, "knowledge_name": resolvedName})
		if runErr != nil {
			return nil, runErr
		}
		if !records.Next(ctx) {
			return nil, ErrFrontierBriefingNotReady
		}
		record := records.Record()
		knowledgeValue, _ := record.Get("knowledge_id")
		knowledgeNameValue, _ := record.Get("knowledge_name")
		summaryValue, _ := record.Get("digest_summary")
		statusValue, _ := record.Get("digest_status")
		generatedAtValue, _ := record.Get("generated_at")
		topicsValue, _ := record.Get("topics")
		papersValue, _ := record.Get("papers")
		briefing := &graphentity.FrontierBriefing{
			KnowledgeID:   stringOrDefault(knowledgeValue, resolvedID),
			Knowledge:     stringOrDefault(knowledgeNameValue, resolvedName),
			DigestSummary: stringOrDefault(summaryValue, ""),
			Status:        stringOrDefault(statusValue, "not_ready"),
			GeneratedAt:   formatGraphTime(generatedAtValue),
			Topics:        parseFrontierTopics(topicsValue),
			Papers:        parseFrontierPapers(papersValue),
		}
		return briefing, records.Err()
	})
	if err != nil {
		if errors.Is(err, ErrFrontierBriefingNotReady) {
			return nil, err
		}
		return nil, fmt.Errorf("查询前沿扩展摘要失败: %w", err)
	}
	briefing, ok := result.(*graphentity.FrontierBriefing)
	if !ok || briefing == nil {
		return nil, ErrFrontierBriefingNotReady
	}
	if strings.TrimSpace(briefing.DigestSummary) == "" {
		return nil, ErrFrontierBriefingNotReady
	}
	if strings.TrimSpace(briefing.Status) == "" {
		briefing.Status = "ready"
	}
	briefing.Message = "已从图谱载入前沿扩展摘要。"
	return briefing, nil
}
