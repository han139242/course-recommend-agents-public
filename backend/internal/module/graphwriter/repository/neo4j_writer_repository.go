package repository

import (
	"context"
	"fmt"
	"time"

	graphwriterentity "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/entity"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jWriterRepository 提供用户能力图谱写入能力。
type Neo4jWriterRepository struct {
	driver neo4j.DriverWithContext
}

// NewNeo4jWriterRepository 创建写入仓储。
func NewNeo4jWriterRepository(driver neo4j.DriverWithContext) *Neo4jWriterRepository {
	return &Neo4jWriterRepository{driver: driver}
}

// Enabled 返回当前是否具备 Neo4j 写入能力。
func (r *Neo4jWriterRepository) Enabled() bool {
	return r != nil && r.driver != nil
}

// UpsertUserCompetencies 批量写入用户能力关系。
func (r *Neo4jWriterRepository) UpsertUserCompetencies(ctx context.Context, events []graphwriterentity.ProfileUpdateEvent, topologyVersion string) error {
	if !r.Enabled() || len(events) == 0 {
		return nil
	}

	payload := make([]map[string]any, 0, len(events))
	for _, event := range events {
		relations := make([]map[string]any, 0, len(event.Relations))
		for _, relation := range event.Relations {
			relations = append(relations, map[string]any{
				"knowledge_id":    relation.KnowledgeID,
				"mastery_score":   relation.MasteryScore,
				"answered_count":  relation.AnsweredCount,
				"correct_count":   relation.CorrectCount,
			})
		}
		payload = append(payload, map[string]any{
			"student_id":       event.StudentID,
			"student_name":     event.Identity.Name,
			"target":           event.Identity.Target,
			"updated_at":       event.UpdatedAt.UTC().Format(time.RFC3339),
			"trace_id":         event.TraceID,
			"source":           event.Source,
			"topology_version": chooseTopologyVersion(event.TopologyVersion, topologyVersion),
			"relations":        relations,
		})
	}

	query := `
UNWIND $events AS event
MERGE (u:User {user_id: toString(event.student_id)})
SET u.name = CASE WHEN trim(coalesce(event.student_name, '')) = '' THEN u.name ELSE event.student_name END,
    u.target = CASE WHEN trim(coalesce(event.target, '')) = '' THEN u.target ELSE event.target END,
    u.last_profile_sync_at = datetime(event.updated_at),
    u.last_profile_trace_id = event.trace_id,
    u.last_profile_source = event.source,
    u.topology_version = event.topology_version
WITH u, event
UNWIND event.relations AS relation
MERGE (c:KnowledgeNode {knowledge_id: relation.knowledge_id})
MERGE (u)-[r:HAS_COMPETENCY]->(c)
SET r.mastery_score = relation.mastery_score,
    r.answered_count = relation.answered_count,
    r.correct_count = relation.correct_count,
    r.updated_at = datetime(event.updated_at),
    r.trace_id = event.trace_id,
    r.topology_version = event.topology_version
`

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, runErr := tx.Run(ctx, query, map[string]any{"events": payload})
		return nil, runErr
	})
	if err != nil {
		return fmt.Errorf("批量写入用户能力图谱失败: %w", err)
	}
	return nil
}

func chooseTopologyVersion(eventVersion string, fallback string) string {
	if eventVersion != "" {
		return eventVersion
	}
	if fallback != "" {
		return fallback
	}
	return "unknown"
}
