package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

const relationPriorityCase = `
CASE type(rel)
  WHEN 'PREREQUISITE_OF' THEN 0
  WHEN 'SUBTOPIC_OF' THEN 1
  WHEN 'PART_OF' THEN 2
  WHEN 'RELATED_TO' THEN 3
  WHEN 'USES' THEN 4
  WHEN 'APPLIES_TO' THEN 5
  ELSE 6
END
`

// NewNeo4jRepository 创建图谱仓储。
func NewNeo4jRepository(driver neo4j.DriverWithContext) *Neo4jRepository {
	return &Neo4jRepository{driver: driver}
}

// Enabled 返回当前是否具备 Neo4j 查询能力。
func (r *Neo4jRepository) Enabled() bool {
	return r != nil && r.driver != nil
}

// LoadSubgraph 查询一跳子图。
func (r *Neo4jRepository) LoadSubgraph(ctx context.Context, root string, limit int) ([]graphentity.Node, []graphentity.Edge, error) {
	if !r.Enabled() {
		return nil, nil, neo4jUsageError("neo4j 未启用")
	}
	resolvedRoot := strings.TrimSpace(root)
	if limit <= 0 {
		limit = 12
	}
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)
	if resolvedRoot == "" {
		defaultRoot, resolveErr := r.resolveDefaultRoot(ctx, session)
		if resolveErr != nil {
			return nil, nil, fmt.Errorf("查询默认图谱根节点失败: %w", resolveErr)
		}
		if defaultRoot == "" {
			return []graphentity.Node{}, []graphentity.Edge{}, nil
		}
		resolvedRoot = defaultRoot
	}
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
MATCH (root)
WHERE ($root <> '' AND (coalesce(root.knowledge_id, root.node_id, root.id, root.name) = $root OR root.name = $root))
OPTIONAL MATCH (root)-[rel]-(neighbor)
WITH root, rel, neighbor
LIMIT $limit
RETURN collect(DISTINCT root) AS roots,
       collect(DISTINCT neighbor) AS neighbors,
       collect(DISTINCT {relation_id: toString(id(rel)), relation_type: type(rel), source_id: coalesce(startNode(rel).knowledge_id, startNode(rel).node_id, startNode(rel).id, startNode(rel).name), target_id: coalesce(endNode(rel).knowledge_id, endNode(rel).node_id, endNode(rel).id, endNode(rel).name), source_name: coalesce(startNode(rel).name, startNode(rel).canonicalName, ''), target_name: coalesce(endNode(rel).name, endNode(rel).canonicalName, '')}) AS edges
`
		records, runErr := tx.Run(ctx, query, map[string]any{"root": resolvedRoot, "limit": limit})
		if runErr != nil {
			return nil, runErr
		}
		if !records.Next(ctx) {
			return struct {
				Nodes []graphentity.Node
				Edges []graphentity.Edge
			}{Nodes: []graphentity.Node{}, Edges: []graphentity.Edge{}}, nil
		}
		record := records.Record()
		rootNodes, _ := record.Get("roots")
		neighborNodes, _ := record.Get("neighbors")
		edgesValue, _ := record.Get("edges")
		nodes := append(parseGraphNodes(rootNodes), parseGraphNodes(neighborNodes)...)
		nodes = dedupeNodes(nodes)
		edges := parseGraphEdges(edgesValue)
		return struct {
			Nodes []graphentity.Node
			Edges []graphentity.Edge
		}{Nodes: nodes, Edges: edges}, records.Err()
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询图谱子图失败: %w", err)
	}
	payload, ok := result.(struct {
		Nodes []graphentity.Node
		Edges []graphentity.Edge
	})
	if !ok {
		return []graphentity.Node{}, []graphentity.Edge{}, nil
	}
	return payload.Nodes, payload.Edges, nil
}

// LoadFullGraph loads a bounded knowledge graph snapshot for the homepage graph view.
func (r *Neo4jRepository) LoadFullGraph(ctx context.Context, limit int) ([]graphentity.Node, []graphentity.Edge, error) {
	if !r.Enabled() {
		return nil, nil, neo4jUsageError("neo4j 未启用")
	}
	if limit <= 0 {
		limit = 1000
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
MATCH (node:KnowledgeNode)
WITH node
ORDER BY coalesce(node.name, node.canonicalName, node.knowledge_id, node.node_id, node.id)
LIMIT $limit
WITH collect(node) AS nodes
UNWIND nodes AS node
OPTIONAL MATCH (node)-[rel]-(neighbor:KnowledgeNode)
WHERE neighbor IN nodes
RETURN nodes AS nodes,
       collect(DISTINCT {
         relation_id: toString(id(rel)),
         relation_type: type(rel),
         source_id: coalesce(startNode(rel).knowledge_id, startNode(rel).node_id, startNode(rel).id, startNode(rel).name),
         target_id: coalesce(endNode(rel).knowledge_id, endNode(rel).node_id, endNode(rel).id, endNode(rel).name),
         source_name: coalesce(startNode(rel).name, startNode(rel).canonicalName, ''),
         target_name: coalesce(endNode(rel).name, endNode(rel).canonicalName, '')
       }) AS edges
`
		records, runErr := tx.Run(ctx, query, map[string]any{"limit": limit})
		if runErr != nil {
			return nil, runErr
		}
		if !records.Next(ctx) {
			return struct {
				Nodes []graphentity.Node
				Edges []graphentity.Edge
			}{Nodes: []graphentity.Node{}, Edges: []graphentity.Edge{}}, nil
		}

		record := records.Record()
		nodesValue, _ := record.Get("nodes")
		edgesValue, _ := record.Get("edges")

		return struct {
			Nodes []graphentity.Node
			Edges []graphentity.Edge
		}{
			Nodes: dedupeNodes(parseGraphNodes(nodesValue)),
			Edges: parseGraphEdges(edgesValue),
		}, records.Err()
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询完整知识图谱失败: %w", err)
	}

	payload, ok := result.(struct {
		Nodes []graphentity.Node
		Edges []graphentity.Edge
	})
	if !ok {
		return []graphentity.Node{}, []graphentity.Edge{}, nil
	}

	return payload.Nodes, payload.Edges, nil
}

// LoadCatalog loads lightweight graph metadata and prerequisite edges for navigation/search.
func (r *Neo4jRepository) LoadCatalog(ctx context.Context) ([]graphentity.Node, []graphentity.Edge, error) {
	if !r.Enabled() {
		return nil, nil, neo4jUsageError("neo4j 未启用")
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
MATCH (node:KnowledgeNode)
WITH collect(node) AS nodes
OPTIONAL MATCH (source:KnowledgeNode)-[rel:PREREQUISITE_OF]->(target:KnowledgeNode)
RETURN nodes AS nodes,
       collect(DISTINCT {
         relation_id: toString(id(rel)),
         relation_type: type(rel),
         source_id: coalesce(source.knowledge_id, source.node_id, source.id, source.name),
         target_id: coalesce(target.knowledge_id, target.node_id, target.id, target.name),
         source_name: coalesce(source.name, source.canonicalName, ''),
         target_name: coalesce(target.name, target.canonicalName, '')
       }) AS prerequisite_edges
`
		records, runErr := tx.Run(ctx, query, map[string]any{})
		if runErr != nil {
			return nil, runErr
		}
		if !records.Next(ctx) {
			return struct {
				Nodes []graphentity.Node
				Edges []graphentity.Edge
			}{Nodes: []graphentity.Node{}, Edges: []graphentity.Edge{}}, nil
		}

		record := records.Record()
		nodesValue, _ := record.Get("nodes")
		edgesValue, _ := record.Get("prerequisite_edges")

		return struct {
			Nodes []graphentity.Node
			Edges []graphentity.Edge
		}{
			Nodes: dedupeNodes(parseGraphNodes(nodesValue)),
			Edges: parseGraphEdges(edgesValue),
		}, records.Err()
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询图谱目录失败: %w", err)
	}

	payload, ok := result.(struct {
		Nodes []graphentity.Node
		Edges []graphentity.Edge
	})
	if !ok {
		return []graphentity.Node{}, []graphentity.Edge{}, nil
	}

	return payload.Nodes, payload.Edges, nil
}

// LoadSeeds loads a bounded graph of important seed nodes for the homepage.
func (r *Neo4jRepository) LoadSeeds(ctx context.Context, count int) ([]graphentity.Node, []graphentity.Edge, error) {
	if !r.Enabled() {
		return nil, nil, neo4jUsageError("neo4j 未启用")
	}
	if count <= 0 {
		count = 14
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
MATCH (node:KnowledgeNode)
WITH node, COUNT { (node)--() } AS degree
ORDER BY degree DESC, coalesce(node.name, node.canonicalName, node.knowledge_id, node.node_id, node.id)
LIMIT $count
WITH collect(node) AS seeds
UNWIND seeds AS seed
OPTIONAL MATCH (seed)-[rel]-(neighbor:KnowledgeNode)
WHERE neighbor IN seeds
RETURN seeds AS nodes,
       collect(DISTINCT {
         relation_id: toString(id(rel)),
         relation_type: type(rel),
         source_id: coalesce(startNode(rel).knowledge_id, startNode(rel).node_id, startNode(rel).id, startNode(rel).name),
         target_id: coalesce(endNode(rel).knowledge_id, endNode(rel).node_id, endNode(rel).id, endNode(rel).name),
         source_name: coalesce(startNode(rel).name, startNode(rel).canonicalName, ''),
         target_name: coalesce(endNode(rel).name, endNode(rel).canonicalName, '')
       }) AS edges
`
		records, runErr := tx.Run(ctx, query, map[string]any{"count": count})
		if runErr != nil {
			return nil, runErr
		}
		if !records.Next(ctx) {
			return struct {
				Nodes []graphentity.Node
				Edges []graphentity.Edge
			}{Nodes: []graphentity.Node{}, Edges: []graphentity.Edge{}}, nil
		}

		record := records.Record()
		nodesValue, _ := record.Get("nodes")
		edgesValue, _ := record.Get("edges")

		return struct {
			Nodes []graphentity.Node
			Edges []graphentity.Edge
		}{
			Nodes: dedupeNodes(parseGraphNodes(nodesValue)),
			Edges: parseGraphEdges(edgesValue),
		}, records.Err()
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询图谱种子节点失败: %w", err)
	}

	payload, ok := result.(struct {
		Nodes []graphentity.Node
		Edges []graphentity.Edge
	})
	if !ok {
		return []graphentity.Node{}, []graphentity.Edge{}, nil
	}

	return payload.Nodes, payload.Edges, nil
}

// LoadNeighbors loads on-demand graph neighbors for a target node.
func (r *Neo4jRepository) LoadNeighbors(ctx context.Context, nodeID string, depth int, direction string, limit int) ([]graphentity.Node, []graphentity.Edge, error) {
	if !r.Enabled() {
		return nil, nil, neo4jUsageError("neo4j 未启用")
	}

	resolvedNodeID := strings.TrimSpace(nodeID)
	if resolvedNodeID == "" {
		return []graphentity.Node{}, []graphentity.Edge{}, nil
	}
	if depth <= 0 {
		depth = 1
	}
	if depth > 2 {
		depth = 2
	}
	if limit <= 0 {
		limit = 20
	}
	if strings.TrimSpace(direction) == "" {
		direction = "both"
	}

	pattern := "-[rel]-(neighbor:KnowledgeNode)"
	pathPattern := "(target)-[*1..2]-(neighbor:KnowledgeNode)"
	switch direction {
	case "outgoing":
		pattern = "-[rel]->(neighbor:KnowledgeNode)"
		pathPattern = "(target)-[*1..2]->(neighbor:KnowledgeNode)"
	case "incoming":
		pattern = "<-[rel]-(neighbor:KnowledgeNode)"
		pathPattern = "(target)<-[*1..2]-(neighbor:KnowledgeNode)"
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := ""
		params := map[string]any{"nodeID": resolvedNodeID, "limit": limit}

		if depth <= 1 {
			query = fmt.Sprintf(`
MATCH (target:KnowledgeNode)
WHERE coalesce(target.knowledge_id, target.node_id, target.id, target.name) = $nodeID
OPTIONAL MATCH (target)%s
WITH target, rel, neighbor
ORDER BY %s, coalesce(neighbor.name, neighbor.canonicalName, neighbor.knowledge_id, neighbor.node_id, neighbor.id)
LIMIT $limit
RETURN collect(DISTINCT target) AS targets,
       collect(DISTINCT neighbor) AS neighbors,
       collect(DISTINCT {
         relation_id: toString(id(rel)),
         relation_type: type(rel),
         source_id: coalesce(startNode(rel).knowledge_id, startNode(rel).node_id, startNode(rel).id, startNode(rel).name),
         target_id: coalesce(endNode(rel).knowledge_id, endNode(rel).node_id, endNode(rel).id, endNode(rel).name),
         source_name: coalesce(startNode(rel).name, startNode(rel).canonicalName, ''),
         target_name: coalesce(endNode(rel).name, endNode(rel).canonicalName, '')
       }) AS edges
`, pattern, relationPriorityCase)
		} else {
			query = fmt.Sprintf(`
MATCH (target:KnowledgeNode)
WHERE coalesce(target.knowledge_id, target.node_id, target.id, target.name) = $nodeID
CALL {
  WITH target
  MATCH path=%s
  WITH path, neighbor,
       [rel IN relationships(path) |
         CASE type(rel)
           WHEN 'PREREQUISITE_OF' THEN 0
           WHEN 'SUBTOPIC_OF' THEN 1
           WHEN 'PART_OF' THEN 2
           WHEN 'RELATED_TO' THEN 3
           WHEN 'USES' THEN 4
           WHEN 'APPLIES_TO' THEN 5
           ELSE 6
         END
       ] AS priorities
  ORDER BY reduce(best = 99, priority IN priorities |
           CASE
             WHEN priority < best THEN priority
             ELSE best
           END
         ),
         length(path),
         coalesce(neighbor.name, neighbor.canonicalName, neighbor.knowledge_id, neighbor.node_id, neighbor.id)
  LIMIT $limit
  RETURN collect(DISTINCT path) AS paths
}
WITH target, paths
UNWIND CASE WHEN size(paths) = 0 THEN [null] ELSE paths END AS path
WITH target,
     CASE WHEN path IS NULL THEN [target] ELSE nodes(path) END AS path_nodes,
     CASE WHEN path IS NULL THEN [] ELSE relationships(path) END AS path_rels
UNWIND path_nodes AS node
WITH collect(DISTINCT node) AS raw_nodes, path_rels
UNWIND CASE WHEN size(path_rels) = 0 THEN [null] ELSE path_rels END AS rel
RETURN raw_nodes AS raw_nodes,
       collect(DISTINCT CASE
         WHEN rel IS NULL THEN null
         ELSE {
           relation_id: toString(id(rel)),
           relation_type: type(rel),
           source_id: coalesce(startNode(rel).knowledge_id, startNode(rel).node_id, startNode(rel).id, startNode(rel).name),
           target_id: coalesce(endNode(rel).knowledge_id, endNode(rel).node_id, endNode(rel).id, endNode(rel).name),
           source_name: coalesce(startNode(rel).name, startNode(rel).canonicalName, ''),
           target_name: coalesce(endNode(rel).name, endNode(rel).canonicalName, '')
         }
       END) AS edges
`, pathPattern)
		}

		records, runErr := tx.Run(ctx, query, params)
		if runErr != nil {
			return nil, runErr
		}
		if !records.Next(ctx) {
			return struct {
				Nodes []graphentity.Node
				Edges []graphentity.Edge
			}{Nodes: []graphentity.Node{}, Edges: []graphentity.Edge{}}, nil
		}

		record := records.Record()
		if depth <= 1 {
			targetValue, _ := record.Get("targets")
			neighborsValue, _ := record.Get("neighbors")
			edgesValue, _ := record.Get("edges")
			nodes := append(parseGraphNodes(targetValue), parseGraphNodes(neighborsValue)...)
			return struct {
				Nodes []graphentity.Node
				Edges []graphentity.Edge
			}{
				Nodes: dedupeNodes(nodes),
				Edges: parseGraphEdges(edgesValue),
			}, records.Err()
		}

		rawNodesValue, _ := record.Get("raw_nodes")
		edgesValue, _ := record.Get("edges")
		return struct {
			Nodes []graphentity.Node
			Edges []graphentity.Edge
		}{
			Nodes: dedupeNodes(parseGraphNodes(rawNodesValue)),
			Edges: parseGraphEdges(edgesValue),
		}, records.Err()
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询图谱邻居失败: %w", err)
	}

	payload, ok := result.(struct {
		Nodes []graphentity.Node
		Edges []graphentity.Edge
	})
	if !ok {
		return []graphentity.Node{}, []graphentity.Edge{}, nil
	}

	return payload.Nodes, payload.Edges, nil
}

// LoadPrerequisiteChain loads a deterministic prerequisite chain for a target knowledge node.
func (r *Neo4jRepository) LoadPrerequisiteChain(ctx context.Context, targetID string, depth int) ([]graphentity.Node, []graphentity.Edge, []string, error) {
	if !r.Enabled() {
		return nil, nil, nil, neo4jUsageError("neo4j 未启用")
	}

	resolvedTargetID := strings.TrimSpace(targetID)
	if resolvedTargetID == "" {
		return []graphentity.Node{}, []graphentity.Edge{}, []string{}, nil
	}
	if depth <= 0 {
		depth = 3
	}
	if depth > 4 {
		depth = 4
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := fmt.Sprintf(`
MATCH (target:KnowledgeNode)
WHERE coalesce(target.knowledge_id, target.node_id, target.id, target.name) = $targetID
   OR target.name = $targetID
CALL {
  WITH target
  OPTIONAL MATCH path = (ancestor:KnowledgeNode)-[:PREREQUISITE_OF*1..%d]->(target)
  WITH path
  ORDER BY length(path) DESC,
           [node IN nodes(path) | coalesce(node.name, node.canonicalName, node.knowledge_id, node.node_id, node.id)]
  LIMIT 1
  RETURN path
}
RETURN CASE
         WHEN path IS NULL THEN [target]
         ELSE nodes(path)
       END AS ordered_nodes,
       CASE
         WHEN path IS NULL THEN []
         ELSE [rel IN relationships(path) | {
           relation_id: toString(id(rel)),
           relation_type: type(rel),
           source_id: coalesce(startNode(rel).knowledge_id, startNode(rel).node_id, startNode(rel).id, startNode(rel).name),
           target_id: coalesce(endNode(rel).knowledge_id, endNode(rel).node_id, endNode(rel).id, endNode(rel).name),
           source_name: coalesce(startNode(rel).name, startNode(rel).canonicalName, ''),
           target_name: coalesce(endNode(rel).name, endNode(rel).canonicalName, '')
         }]
       END AS ordered_rels
`, depth)

		records, runErr := tx.Run(ctx, query, map[string]any{"targetID": resolvedTargetID})
		if runErr != nil {
			return nil, runErr
		}
		if !records.Next(ctx) {
			return struct {
				Nodes          []graphentity.Node
				Edges          []graphentity.Edge
				OrderedNodeIDs []string
			}{
				Nodes:          []graphentity.Node{},
				Edges:          []graphentity.Edge{},
				OrderedNodeIDs: []string{},
			}, nil
		}

		record := records.Record()
		orderedNodesValue, _ := record.Get("ordered_nodes")
		orderedRelsValue, _ := record.Get("ordered_rels")
		orderedNodes := parseGraphNodes(orderedNodesValue)
		return struct {
			Nodes          []graphentity.Node
			Edges          []graphentity.Edge
			OrderedNodeIDs []string
		}{
			Nodes:          dedupeNodes(orderedNodes),
			Edges:          parseGraphEdges(orderedRelsValue),
			OrderedNodeIDs: extractOrderedNodeIDs(orderedNodes),
		}, records.Err()
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("查询先修链失败: %w", err)
	}

	payload, ok := result.(struct {
		Nodes          []graphentity.Node
		Edges          []graphentity.Edge
		OrderedNodeIDs []string
	})
	if !ok {
		return []graphentity.Node{}, []graphentity.Edge{}, []string{}, nil
	}

	return payload.Nodes, payload.Edges, payload.OrderedNodeIDs, nil
}

// FindShortestPath loads a shortest path between two knowledge nodes.
func (r *Neo4jRepository) FindShortestPath(ctx context.Context, fromID string, toID string, maxDepth int) ([]graphentity.Node, []graphentity.Edge, []string, bool, error) {
	if !r.Enabled() {
		return nil, nil, nil, false, neo4jUsageError("neo4j 未启用")
	}

	resolvedFrom := strings.TrimSpace(fromID)
	resolvedTo := strings.TrimSpace(toID)
	if resolvedFrom == "" || resolvedTo == "" {
		return []graphentity.Node{}, []graphentity.Edge{}, []string{}, false, nil
	}
	if maxDepth <= 0 {
		maxDepth = 5
	}
	if maxDepth > 6 {
		maxDepth = 6
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := fmt.Sprintf(`
MATCH (source:KnowledgeNode)
WHERE coalesce(source.knowledge_id, source.node_id, source.id, source.name) = $fromID
   OR source.name = $fromID
MATCH (target:KnowledgeNode)
WHERE coalesce(target.knowledge_id, target.node_id, target.id, target.name) = $toID
   OR target.name = $toID
CALL {
  WITH source, target
  OPTIONAL MATCH path = shortestPath((source)-[*..%d]-(target))
  RETURN path
}
RETURN CASE
         WHEN path IS NULL THEN []
         ELSE nodes(path)
       END AS ordered_nodes,
       CASE
         WHEN path IS NULL THEN []
         ELSE [rel IN relationships(path) | {
           relation_id: toString(id(rel)),
           relation_type: type(rel),
           source_id: coalesce(startNode(rel).knowledge_id, startNode(rel).node_id, startNode(rel).id, startNode(rel).name),
           target_id: coalesce(endNode(rel).knowledge_id, endNode(rel).node_id, endNode(rel).id, endNode(rel).name),
           source_name: coalesce(startNode(rel).name, startNode(rel).canonicalName, ''),
           target_name: coalesce(endNode(rel).name, endNode(rel).canonicalName, '')
         }]
       END AS ordered_rels
`, maxDepth)

		records, runErr := tx.Run(ctx, query, map[string]any{
			"fromID": resolvedFrom,
			"toID":   resolvedTo,
		})
		if runErr != nil {
			return nil, runErr
		}
		if !records.Next(ctx) {
			return struct {
				Nodes          []graphentity.Node
				Edges          []graphentity.Edge
				OrderedNodeIDs []string
				Found          bool
			}{
				Nodes:          []graphentity.Node{},
				Edges:          []graphentity.Edge{},
				OrderedNodeIDs: []string{},
				Found:          false,
			}, nil
		}

		record := records.Record()
		orderedNodesValue, _ := record.Get("ordered_nodes")
		orderedRelsValue, _ := record.Get("ordered_rels")
		orderedNodes := parseGraphNodes(orderedNodesValue)
		return struct {
			Nodes          []graphentity.Node
			Edges          []graphentity.Edge
			OrderedNodeIDs []string
			Found          bool
		}{
			Nodes:          dedupeNodes(orderedNodes),
			Edges:          parseGraphEdges(orderedRelsValue),
			OrderedNodeIDs: extractOrderedNodeIDs(orderedNodes),
			Found:          len(orderedNodes) > 0,
		}, records.Err()
	})
	if err != nil {
		return nil, nil, nil, false, fmt.Errorf("查询图谱最短路径失败: %w", err)
	}

	payload, ok := result.(struct {
		Nodes          []graphentity.Node
		Edges          []graphentity.Edge
		OrderedNodeIDs []string
		Found          bool
	})
	if !ok {
		return []graphentity.Node{}, []graphentity.Edge{}, []string{}, false, nil
	}

	return payload.Nodes, payload.Edges, payload.OrderedNodeIDs, payload.Found, nil
}

func (r *Neo4jRepository) resolveDefaultRoot(ctx context.Context, session neo4j.SessionWithContext) (string, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
MATCH (root:KnowledgeNode)
RETURN coalesce(root.knowledge_id, root.node_id, root.id, root.name) AS root_id
ORDER BY coalesce(root.name, root.knowledge_id, root.id)
LIMIT 1
`
		records, runErr := tx.Run(ctx, query, map[string]any{})
		if runErr != nil {
			return nil, runErr
		}
		if !records.Next(ctx) {
			return "", nil
		}
		rootID, _ := records.Record().Get("root_id")
		return stringOrDefault(rootID, ""), records.Err()
	})
	if err != nil {
		return "", err
	}

	rootID, _ := result.(string)
	return rootID, nil
}

// LoadKnowledgeDetail 查询知识点详情。
func (r *Neo4jRepository) LoadKnowledgeDetail(ctx context.Context, knowledgeID string) (*graphentity.KnowledgeDetail, error) {
	if !r.Enabled() {
		return nil, neo4jUsageError("neo4j 未启用")
	}
	resolvedID := strings.TrimSpace(knowledgeID)
	if resolvedID == "" {
		return nil, nil
	}
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
MATCH (k)
WHERE coalesce(k.knowledge_id, k.node_id, k.id, k.name) = $knowledge_id OR k.name = $knowledge_id
OPTIONAL MATCH (pre)-[:PREREQUISITE_OF]->(k)
OPTIONAL MATCH (k)-[:RELATED_TO]-(related)
OPTIONAL MATCH (k)-[:SUBTOPIC_OF]->(parent)
RETURN k,
       collect(DISTINCT coalesce(pre.name, pre.canonicalName, pre.knowledge_id, pre.id)) AS prerequisites,
       collect(DISTINCT coalesce(related.name, related.canonicalName, related.knowledge_id, related.id)) AS related,
       collect(DISTINCT coalesce(parent.name, parent.canonicalName, parent.knowledge_id, parent.id)) AS parent_topics
LIMIT 1
`
		records, runErr := tx.Run(ctx, query, map[string]any{"knowledge_id": resolvedID})
		if runErr != nil {
			return nil, runErr
		}
		if !records.Next(ctx) {
			return nil, nil
		}
		record := records.Record()
		nodeValue, _ := record.Get("k")
		prerequisitesValue, _ := record.Get("prerequisites")
		relatedValue, _ := record.Get("related")
		parentTopicsValue, _ := record.Get("parent_topics")
		node := parseSingleNode(nodeValue)
		if node.ID == "" {
			return nil, nil
		}
		detail := &graphentity.KnowledgeDetail{Node: node, Summary: node.Description, Prerequisites: parseStringList(prerequisitesValue), Related: parseStringList(relatedValue), Scenarios: []string{}, ParentTopics: parseStringList(parentTopicsValue)}
		return detail, records.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("查询知识点详情失败: %w", err)
	}
	if result == nil {
		return nil, nil
	}
	detail, _ := result.(*graphentity.KnowledgeDetail)
	return detail, nil
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
		topicID := stringOrDefault(mapped["topic_id"], "")
		if topicID == "" {
			continue
		}
		result = append(result, graphentity.FrontierTopic{TopicID: topicID, Name: stringOrDefault(mapped["name"], ""), Reason: stringOrDefault(mapped["reason"], "")})
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
		paperID := stringOrDefault(mapped["paper_id"], "")
		if paperID == "" {
			continue
		}
		result = append(result, graphentity.FrontierPaper{PaperID: paperID, Title: stringOrDefault(mapped["title"], ""), Venue: stringOrDefault(mapped["venue"], ""), Year: intOrDefault(mapped["year"]), URL: stringOrDefault(mapped["url"], "")})
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

// LoadCompetencyMappings 读取知识点到能力维度的映射矩阵。
func (r *Neo4jRepository) LoadCompetencyMappings(ctx context.Context) (map[string][]graphentity.CompetencyMapping, error) {
	if !r.Enabled() {
		return map[string][]graphentity.CompetencyMapping{}, nil
	}
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
MATCH (kp)-[rel:MAPS_TO]->(comp:Competency)
WHERE kp.knowledge_id IS NOT NULL OR kp.node_id IS NOT NULL OR kp.id IS NOT NULL
RETURN coalesce(kp.knowledge_id, kp.node_id, kp.id) AS knowledge_id,
       comp.id AS competency_id,
       comp.name AS competency_name,
       comp.category AS category,
       coalesce(rel.weight, 0) AS weight
`
		records, runErr := tx.Run(ctx, query, map[string]any{})
		if runErr != nil {
			return nil, runErr
		}
		mappings := make(map[string][]graphentity.CompetencyMapping)
		for records.Next(ctx) {
			record := records.Record()
			knowledgeID, _ := record.Get("knowledge_id")
			competencyID, _ := record.Get("competency_id")
			competencyName, _ := record.Get("competency_name")
			category, _ := record.Get("category")
			weight, _ := record.Get("weight")
			key := stringOrDefault(knowledgeID, "")
			if key == "" {
				continue
			}
			mappings[key] = append(mappings[key], graphentity.CompetencyMapping{KnowledgeID: key, CompetencyID: stringOrDefault(competencyID, ""), CompetencyName: stringOrDefault(competencyName, ""), Category: stringOrDefault(category, ""), Weight: floatOrDefault(weight)})
		}
		return mappings, records.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("读取能力映射矩阵失败: %w", err)
	}
	mapped, ok := result.(map[string][]graphentity.CompetencyMapping)
	if !ok {
		return map[string][]graphentity.CompetencyMapping{}, nil
	}
	return mapped, nil
}

func floatOrDefault(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int64:
		return float64(typed)
	case int:
		return float64(typed)
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

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
MATCH (k:KnowledgeNode)
WHERE ($knowledge_id <> '' AND coalesce(k.knowledge_id, k.node_id, k.id) = $knowledge_id)
   OR ($knowledge_name <> '' AND k.name = $knowledge_name)
OPTIONAL MATCH (d:FrontierDigest)-[:CURATED_FOR]->(k)
OPTIONAL MATCH (topic:ResearchTopic)-[:EXTENDS_KNOWLEDGE]->(k)
OPTIONAL MATCH (d)-[:SUMMARIZES_PAPER]->(paper:Paper)
RETURN coalesce(k.knowledge_id, k.node_id, k.id, k.name) AS knowledge_id,
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

func parseGraphNodes(value any) []graphentity.Node {
	items, ok := value.([]any)
	if !ok {
		return []graphentity.Node{}
	}
	result := make([]graphentity.Node, 0, len(items))
	for _, item := range items {
		node := parseSingleNode(item)
		if node.ID != "" {
			result = append(result, node)
		}
	}
	return result
}

func parseSingleNode(value any) graphentity.Node {
	node, ok := value.(neo4j.Node)
	if !ok {
		return graphentity.Node{}
	}
	properties := node.Props
	id := stringOrDefault(properties["knowledge_id"], stringOrDefault(properties["node_id"], stringOrDefault(properties["id"], stringOrDefault(properties["name"], ""))))
	name := stringOrDefault(properties["name"], id)
	canonical := stringOrDefault(properties["canonicalName"], stringOrDefault(properties["canonical_name"], name))
	entityType := stringOrDefault(properties["entityType"], stringOrDefault(properties["entity_type"], "concept"))
	description := stringOrDefault(properties["description"], "")
	chapterTitle := stringOrDefault(properties["chapterTitle"], stringOrDefault(properties["chapter_title"], ""))
	sourcePath := stringOrDefault(properties["sourcePath"], stringOrDefault(properties["source_path"], ""))
	curatedSource := stringOrDefault(properties["curatedSource"], stringOrDefault(properties["curated_source"], ""))
	return graphentity.Node{
		ID:               id,
		RouteKnowledgeID: id,
		Name:             name,
		CanonicalName:    canonical,
		EntityType:       entityType,
		Description:      description,
		ChapterTitle:     chapterTitle,
		SourcePath:       sourcePath,
		CuratedSource:    curatedSource,
	}
}

func parseGraphEdges(value any) []graphentity.Edge {
	items, ok := value.([]any)
	if !ok {
		return []graphentity.Edge{}
	}
	result := make([]graphentity.Edge, 0, len(items))
	for _, item := range items {
		mapped, ok := item.(map[string]any)
		if !ok {
			continue
		}
		relationID := stringOrDefault(mapped["relation_id"], "")
		if relationID == "" {
			continue
		}
		result = append(result, graphentity.Edge{RelationID: relationID, RelationType: stringOrDefault(mapped["relation_type"], ""), SourceID: stringOrDefault(mapped["source_id"], ""), TargetID: stringOrDefault(mapped["target_id"], ""), SourceName: stringOrDefault(mapped["source_name"], ""), TargetName: stringOrDefault(mapped["target_name"], "")})
	}
	return result
}

func parseStringList(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text := stringOrDefault(item, "")
		if text != "" {
			result = append(result, text)
		}
	}
	sort.Strings(result)
	return result
}

func extractOrderedNodeIDs(nodes []graphentity.Node) []string {
	result := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if strings.TrimSpace(node.RouteKnowledgeID) != "" {
			result = append(result, strings.TrimSpace(node.RouteKnowledgeID))
			continue
		}
		if strings.TrimSpace(node.ID) != "" {
			result = append(result, strings.TrimSpace(node.ID))
		}
	}
	return result
}

func dedupeNodes(nodes []graphentity.Node) []graphentity.Node {
	seen := make(map[string]struct{}, len(nodes))
	result := make([]graphentity.Node, 0, len(nodes))
	for _, node := range nodes {
		if node.ID == "" {
			continue
		}
		if _, exists := seen[node.ID]; exists {
			continue
		}
		seen[node.ID] = struct{}{}
		result = append(result, node)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

func neo4jUsageError(message string) error {
	return errors.New(message)
}
