package repository

import (
	"context"
	"testing"

	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
)

func TestGetBriefingReturnsNotReadyWhenGraphDisabled(t *testing.T) {
	repo := New(graphrepo.NewNeo4jRepository(nil))
	resp, err := repo.GetBriefing(context.Background(), "gradient-descent", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "not_ready" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
}

func TestListLibraryReturnsNotReadyWhenGraphDisabled(t *testing.T) {
	repo := New(graphrepo.NewNeo4jRepository(nil))
	resp, err := repo.ListLibrary(context.Background(), 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "not_ready" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected no items, got %d", len(resp.Items))
	}
}
