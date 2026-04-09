package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	graphwriterrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/repository"
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
)

func TestReadyzReturnsRuntimeGraphWriterStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	cfg := config.Config{App: config.AppConfig{Env: "development"}, Observability: config.ObservabilityConfig{BuildVersion: "dev"}, GraphWriter: config.GraphWriterConfig{Enabled: true}}
	graphWriter := graphwriterservice.New(applogger.New("debug"), graphwriterrepo.NewNeo4jWriterRepository(nil), nil, config.GraphWriterConfig{Enabled: true})
	graphWriter.Start(context.TODO())
	registerRuntimeRoutes(engine, cfg, graphWriter)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if body["code"].(float64) != 0 {
		t.Fatalf("unexpected response body: %v", body)
	}
}
