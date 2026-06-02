package service

import (
	"context"
	"testing"

	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	graphentity "github.com/han139242/course-recommend-agents/backend/internal/module/graph/entity"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
)

func TestGetSubgraphReturnsDependencyUnavailableWhenNeo4jDisabled(t *testing.T) {
	repo := graphrepo.NewNeo4jRepository(nil)
	service := New(repo)
	_, err := service.GetSubgraph(context.Background(), graphdto.SubgraphQuery{Root: "kp_001", Limit: 5})
	if err == nil {
		t.Fatal("expected dependency unavailable when neo4j is disabled")
	}
}

func TestGetKnowledgeDetailReturnsDependencyUnavailableWhenNeo4jDisabled(t *testing.T) {
	repo := graphrepo.NewNeo4jRepository(nil)
	service := New(repo)
	_, err := service.GetKnowledgeDetail(context.Background(), "kp_001")
	if err == nil {
		t.Fatal("expected dependency unavailable when neo4j is disabled")
	}
}

func TestGetCatalogReturnsDependencyUnavailableWhenNeo4jDisabled(t *testing.T) {
	repo := graphrepo.NewNeo4jRepository(nil)
	service := New(repo)
	_, err := service.GetCatalog(context.Background())
	if err == nil {
		t.Fatal("expected dependency unavailable when neo4j is disabled")
	}
}

func TestGetSeedsReturnsDependencyUnavailableWhenNeo4jDisabled(t *testing.T) {
	repo := graphrepo.NewNeo4jRepository(nil)
	service := New(repo)
	_, err := service.GetSeeds(context.Background(), graphdto.SeedQuery{Count: 14})
	if err == nil {
		t.Fatal("expected dependency unavailable when neo4j is disabled")
	}
}

func TestGetNeighborsReturnsDependencyUnavailableWhenNeo4jDisabled(t *testing.T) {
	repo := graphrepo.NewNeo4jRepository(nil)
	service := New(repo)
	_, err := service.GetNeighbors(context.Background(), graphdto.NeighborQuery{NodeID: "kp_001"})
	if err == nil {
		t.Fatal("expected dependency unavailable when neo4j is disabled")
	}
}

func TestGetPrerequisiteChainReturnsDependencyUnavailableWhenNeo4jDisabled(t *testing.T) {
	repo := graphrepo.NewNeo4jRepository(nil)
	service := New(repo)
	_, err := service.GetPrerequisiteChain(context.Background(), graphdto.PrerequisiteChainQuery{Target: "kp_001"})
	if err == nil {
		t.Fatal("expected dependency unavailable when neo4j is disabled")
	}
}

func TestFindPathReturnsDependencyUnavailableWhenNeo4jDisabled(t *testing.T) {
	repo := graphrepo.NewNeo4jRepository(nil)
	service := New(repo)
	_, err := service.FindPath(context.Background(), graphdto.GraphPathQuery{From: "kp_001", To: "kp_002"})
	if err == nil {
		t.Fatal("expected dependency unavailable when neo4j is disabled")
	}
}

func TestBuildGraphResponseCanonicalizesAIONodesForStudentView(t *testing.T) {
	response := buildGraphResponse(
		[]graphentity.Node{
			{ID: "AIO:LargeLanguageModel", Name: "AIO:Large Language Model", CanonicalName: "AIO:Large Language Model", ChapterTitle: "AIO Ontology ML/DL 核心概念"},
			{ID: "course-transformer", Name: "Transformer", CanonicalName: "Transformer", ChapterTitle: "大模型"},
		},
		[]graphentity.Edge{
			{RelationID: "r1", RelationType: "RELATED_TO", SourceID: "AIO:LargeLanguageModel", TargetID: "course-transformer"},
		},
	)

	if len(response.Nodes) != 2 {
		t.Fatalf("unexpected nodes: %+v", response.Nodes)
	}
	llm := response.Nodes[0]
	if response.Nodes[1].RouteKnowledgeID == "llm-app" {
		llm = response.Nodes[1]
	}
	if llm.RouteKnowledgeID != "llm-app" || llm.CanonicalName != "大语言模型" || llm.ChapterTitle != "大模型" {
		t.Fatalf("expected AIO LLM node to be localized, got %+v", llm)
	}
	if response.Edges[0].SourceID == "AIO:LargeLanguageModel" && response.Edges[0].SourceName != "大语言模型" {
		t.Fatalf("expected edge label to use localized name, got %+v", response.Edges[0])
	}
}

func TestBuildGraphResponseDropsUnmappedAIOIsolates(t *testing.T) {
	response := buildGraphResponse(
		[]graphentity.Node{
			{ID: "AIO:VerySpecificOntologyTerm", Name: "AIO:Very Specific Ontology Term", ChapterTitle: "AIO Ontology ML/DL 核心概念"},
			{ID: "kp_001", Name: "梯度下降", CanonicalName: "梯度下降", ChapterTitle: "机器学习"},
		},
		nil,
	)

	if len(response.Nodes) != 1 {
		t.Fatalf("expected only course node, got %+v", response.Nodes)
	}
	if response.Nodes[0].ID != "kp_001" {
		t.Fatalf("unexpected remaining node: %+v", response.Nodes[0])
	}
}

func TestBuildGraphResponseLocalizesPlainEnglishCourseNodes(t *testing.T) {
	response := buildGraphResponse(
		[]graphentity.Node{
			{ID: "deep-rl", RouteKnowledgeID: "deep-rl", Name: "Deep RL", CanonicalName: "Deep RL", ChapterTitle: "强化学习"},
			{ID: "series", RouteKnowledgeID: "series", Name: "Series", CanonicalName: "Series", ChapterTitle: "Python编程"},
		},
		nil,
	)

	names := map[string]string{}
	for _, node := range response.Nodes {
		names[node.ID] = node.CanonicalName
	}
	if names["deep-rl"] != "深度强化学习（Deep RL）" {
		t.Fatalf("expected Deep RL to be localized, got %+v", response.Nodes)
	}
	if names["series"] != "序列对象（Series）" {
		t.Fatalf("expected Series to be localized, got %+v", response.Nodes)
	}
}

func TestBuildGraphResponseMergesGeneratedDuplicateNames(t *testing.T) {
	response := buildGraphResponse(
		[]graphentity.Node{
			{ID: "aaaaaaaaaaaaaaaa", RouteKnowledgeID: "aaaaaaaaaaaaaaaa", Name: "Deep RL", CanonicalName: "Deep RL", ChapterTitle: "强化学习"},
			{ID: "bbbbbbbbbbbbbbbb", RouteKnowledgeID: "bbbbbbbbbbbbbbbb", Name: "Deep RL", CanonicalName: "Deep RL", ChapterTitle: "强化学习"},
		},
		[]graphentity.Edge{
			{RelationID: "r1", RelationType: "RELATED_TO", SourceID: "aaaaaaaaaaaaaaaa", TargetID: "bbbbbbbbbbbbbbbb"},
		},
	)

	if len(response.Nodes) != 1 {
		t.Fatalf("expected duplicate generated nodes to merge, got %+v", response.Nodes)
	}
	if response.Nodes[0].CanonicalName != "深度强化学习（Deep RL）" {
		t.Fatalf("expected merged node to use localized name, got %+v", response.Nodes[0])
	}
	if len(response.Edges) != 0 {
		t.Fatalf("expected self edge to be removed after merge, got %+v", response.Edges)
	}
}
