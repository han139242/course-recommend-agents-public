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
