package service

import (
	"context"
	"testing"
	"time"

	frontierdto "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/dto"
	frontierrepo "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/repository"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
)

func TestGetBriefingRequiresKnowledgeSelector(t *testing.T) {
	service := NewWithLiveSource(frontierrepo.New(graphrepo.NewNeo4jRepository(nil)), nil)
	_, err := service.GetBriefing(context.Background(), frontierdto.BriefingQuery{})
	if err == nil {
		t.Fatal("expected bad request error")
	}
}

func TestGetBriefingReturnsNotReadyWhenGraphUnavailable(t *testing.T) {
	service := NewWithLiveSource(frontierrepo.New(graphrepo.NewNeo4jRepository(nil)), nil)
	resp, err := service.GetBriefing(context.Background(), frontierdto.BriefingQuery{KnowledgeID: "gradient-descent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "not_ready" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
	if resp.KnowledgeID == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestListLibraryReturnsNotReadyWhenGraphUnavailable(t *testing.T) {
	service := NewWithLiveSource(frontierrepo.New(graphrepo.NewNeo4jRepository(nil)), nil)
	resp, err := service.ListLibrary(context.Background(), frontierdto.LibraryQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "not_ready" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
	if resp.Items == nil {
		t.Fatal("expected empty items slice")
	}
}

func TestNormalizeStatusMarksStale(t *testing.T) {
	service := New(nil)
	resp := &frontierdto.BriefingResponse{Status: "ready", GeneratedAt: time.Now().Add(-31 * 24 * time.Hour).UTC().Format(time.RFC3339)}
	service.normalizeStatus(resp)
	if resp.Status != "stale" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
}

func TestNormalizeStatusKeepsFailedMessage(t *testing.T) {
	service := New(nil)
	resp := &frontierdto.BriefingResponse{Status: "failed"}
	service.normalizeStatus(resp)
	if resp.Message == "" {
		t.Fatal("expected failed status message")
	}
}

func TestNormalizeStatusNormalizesCaseAndSeparator(t *testing.T) {
	service := New(nil)
	resp := &frontierdto.BriefingResponse{Status: "Not-Ready"}
	service.normalizeStatus(resp)
	if resp.Status != "not_ready" {
		t.Fatalf("unexpected normalized status: %s", resp.Status)
	}
}

func TestLibraryReadyRejectsExampleLinks(t *testing.T) {
	resp := &frontierdto.LibraryResponse{
		Status: "ready",
		Items: []frontierdto.LibraryItemView{
			{Title: "Seed Paper", URL: "https://example.org/papers/demo", Source: "frontier_seed"},
		},
	}
	if libraryReady(resp) {
		t.Fatal("expected example.org seed data to be treated as not ready")
	}
}

func TestLibraryReadyRejectsExampleLinksEvenWhenSourceLooksTrusted(t *testing.T) {
	resp := &frontierdto.LibraryResponse{
		Status: "ready",
		Items: []frontierdto.LibraryItemView{
			{Title: "Seed Paper", URL: "https://example.org/papers/demo", Source: "arxiv"},
		},
	}
	if libraryReady(resp) {
		t.Fatal("expected example.org URL to be rejected before trusting source metadata")
	}
}

func TestSanitizeLibraryResponseDropsExampleLinks(t *testing.T) {
	resp := sanitizeLibraryResponse(&frontierdto.LibraryResponse{
		Status: "ready",
		Items: []frontierdto.LibraryItemView{
			{Title: "Seed Paper", URL: "https://example.org/papers/demo", Source: "frontier_seed_metadata"},
			{Title: "Real Paper", URL: "https://arxiv.org/abs/2605.12345", Source: "arxiv"},
		},
	})

	if resp.Status != "ready" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
	if len(resp.Items) != 1 || resp.Items[0].Title != "Real Paper" {
		t.Fatalf("expected only trusted item, got %+v", resp.Items)
	}
	if resp.Total != 1 {
		t.Fatalf("unexpected total: %d", resp.Total)
	}
}

func TestSanitizeLibraryResponseMarksOnlyExampleLinksNotReady(t *testing.T) {
	resp := sanitizeLibraryResponse(&frontierdto.LibraryResponse{
		Status: "ready",
		Items: []frontierdto.LibraryItemView{
			{Title: "Seed Paper", URL: "https://example.org/papers/demo", Source: "frontier_seed_metadata"},
		},
	})

	if resp.Status != "not_ready" {
		t.Fatalf("expected not_ready after filtering examples, got %s", resp.Status)
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected no items, got %+v", resp.Items)
	}
}

func TestSanitizeBriefingResponseDropsExamplePapers(t *testing.T) {
	resp := sanitizeBriefingResponse(&frontierdto.BriefingResponse{
		Status: "ready",
		Papers: []frontierdto.PaperView{
			{Title: "Seed Paper", URL: "https://example.org/papers/demo", Venue: "CVPR"},
		},
		DigestSummary: "seed",
	}, "kp", "知识点")

	if resp.Status != "not_ready" {
		t.Fatalf("expected not_ready after filtering example papers, got %s", resp.Status)
	}
	if len(resp.Papers) != 0 || resp.DigestSummary != "" {
		t.Fatalf("unexpected sanitized briefing: %+v", resp)
	}
}
