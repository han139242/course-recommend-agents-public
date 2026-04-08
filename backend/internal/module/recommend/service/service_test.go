package service

import (
	"context"
	"testing"

	"github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

func TestGetPathReturnsFallbackRecommendation(t *testing.T) {
	ctx := ctxmeta.WithUserID(context.Background(), "2")
	graphRepo := repository.NewNeo4jRepository(nil)
	graphSvc := graphservice.New(graphRepo)
	service := New(graphSvc, learningrepo.NewRedisRepository(nil))
	resp, err := service.GetPath(ctx, recommenddto.PathQuery{Target: "kp_001", Limit: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Path) == 0 {
		t.Fatal("expected non-empty path")
	}
}
