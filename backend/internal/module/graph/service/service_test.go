package service

import (
	"context"
	"testing"

	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
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
