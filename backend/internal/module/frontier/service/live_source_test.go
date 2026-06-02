package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	frontierdto "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/dto"
)

func TestLiveSourceListLibraryFetchesRealSourcesAndCachesWeekly(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/arxiv":
			w.Header().Set("Content-Type", "application/atom+xml")
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2605.31593v1</id>
    <title>Stateful Online Monitoring Catches Distributed Agent Attacks</title>
    <updated>2026-05-29T17:57:00Z</updated>
    <published>2026-05-29T17:57:00Z</published>
    <summary>Language models and agents require online safety monitoring.</summary>
    <category term="cs.AI" />
    <author><name>Davis Brown</name></author>
  </entry>
</feed>`))
		case "/github":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
  "items": [{
    "id": 42,
    "full_name": "open/course-rag-agent",
    "html_url": "https://github.com/open/course-rag-agent",
    "description": "RAG agent workflow for LLM applications",
    "updated_at": "2026-06-01T10:00:00Z",
    "pushed_at": "2026-06-01T10:00:00Z",
    "language": "Go",
    "stargazers_count": 1200,
    "topics": ["llm", "rag", "agent"],
    "owner": {"login": "open"}
  }]
}`))
		case "/openalex":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	source := newLiveSource(sourceConfig{
		arxivEndpoint:    server.URL + "/arxiv",
		gitHubEndpoint:   server.URL + "/github",
		openAlexEndpoint: server.URL + "/openalex",
		now:              func() time.Time { return time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC) },
	})

	first, err := source.ListLibrary(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.Status != "ready" {
		t.Fatalf("unexpected status: %s", first.Status)
	}
	if first.RefreshInterval != "7d" || first.NextRefreshAt == "" {
		t.Fatalf("expected weekly refresh metadata: %+v", first)
	}
	if len(first.Items) != 2 {
		t.Fatalf("expected two items, got %d", len(first.Items))
	}
	if !strings.Contains(first.Items[0].Title+" "+first.Items[1].Title, "course-rag-agent") {
		t.Fatalf("expected GitHub item in response: %+v", first.Items)
	}

	second, err := source.ListLibrary(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected cache error: %v", err)
	}
	if len(second.Items) != 2 {
		t.Fatalf("expected cached items, got %d", len(second.Items))
	}
	if requests != 3 {
		t.Fatalf("expected one request per source, got %d", requests)
	}
}

func TestLiveSourceGetBriefingFiltersByKnowledge(t *testing.T) {
	source := newLiveSource(sourceConfig{
		cacheTTL: time.Hour,
		now:      func() time.Time { return time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC) },
	})
	source.cache = &libraryCache{
		expiresAt: time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC),
		response:  &frontierReadyFixture,
	}

	resp, err := source.GetBriefing(context.Background(), "rag", "检索增强生成 RAG")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "ready" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
	if len(resp.Papers) != 1 {
		t.Fatalf("expected one matching paper, got %d", len(resp.Papers))
	}
	if !strings.Contains(resp.Papers[0].Title, "RAG") {
		t.Fatalf("unexpected paper: %+v", resp.Papers[0])
	}
}

var frontierReadyFixture = frontierLibraryFixture()

func frontierLibraryFixture() frontierdto.LibraryResponse {
	return frontierdto.LibraryResponse{
		Status:      "ready",
		GeneratedAt: "2026-06-01T12:00:00Z",
		Items: []frontierdto.LibraryItemView{
			{
				PaperID:  "arxiv:rag",
				Title:    "RAG Agents for Knowledge Graph Reasoning",
				Abstract: "Retrieval augmented generation agents use knowledge graphs.",
				Venue:    "arXiv",
				Year:     2026,
				URL:      "https://arxiv.org/abs/2605.00001",
				Topics: []frontierdto.TopicView{
					{TopicID: "rag", Name: "检索增强生成", Reason: "fixture"},
				},
				KnowledgeRefs: []frontierdto.KnowledgeRefView{
					{KnowledgeID: "rag", Name: "检索增强生成 RAG", Course: "大模型", Reason: "fixture"},
				},
			},
			{
				PaperID:  "arxiv:vision",
				Title:    "Video Diffusion Transformer",
				Abstract: "Multimodal video generation.",
				Venue:    "arXiv",
				Year:     2026,
				URL:      "https://arxiv.org/abs/2605.00002",
				Topics: []frontierdto.TopicView{
					{TopicID: "multimodal", Name: "多模态学习", Reason: "fixture"},
				},
			},
		},
	}
}
