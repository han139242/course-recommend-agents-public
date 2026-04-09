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
	service := New(frontierrepo.New(graphrepo.NewNeo4jRepository(nil)))
	_, err := service.GetBriefing(context.Background(), frontierdto.BriefingQuery{})
	if err == nil {
		t.Fatal("expected bad request error")
	}
}

func TestGetBriefingReturnsNotReadyWhenGraphUnavailable(t *testing.T) {
	service := New(frontierrepo.New(graphrepo.NewNeo4jRepository(nil)))
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
