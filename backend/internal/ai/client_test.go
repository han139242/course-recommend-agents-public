package ai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
)

func TestGenerateAndPollRealMediaProvider(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("missing provider authorization header")
		}
		switch r.URL.Path {
		case "/v1/videos/generations":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"task-42","status":"queued"}`)
		case "/v1/videos/tasks/task-42":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"task-42","status":"completed","output":{"url":"https://cdn.example/demo.mp4","mime_type":"video/mp4"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(config.AIConfig{
		APIKey:              "test-key",
		BaseURL:             server.URL,
		ChatModel:           "test-chat",
		VideoModel:          "test-video",
		VideoEndpoint:       server.URL + "/v1/videos/generations",
		VideoStatusEndpoint: server.URL + "/v1/videos/tasks/{id}",
	})
	created, err := client.GenerateVideo(context.Background(), MediaRequest{Prompt: "教学动画"})
	if err != nil || created.ProviderTaskID != "task-42" {
		t.Fatalf("unexpected generation result: %#v, err=%v", created, err)
	}
	completed, err := client.GetMediaStatus(context.Background(), "video", created.ProviderTaskID)
	if err != nil || completed.URL != "https://cdn.example/demo.mp4" || completed.Status != "completed" {
		t.Fatalf("unexpected status result: %#v, err=%v", completed, err)
	}
}
