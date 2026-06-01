package service

import (
	"context"
	"testing"

	authentity "github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
	competencyservice "github.com/han139242/course-recommend-agents/backend/internal/module/competency/service"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

func TestGetSummaryReturnsAggregatedFields(t *testing.T) {
	ctx := ctxmeta.WithUserID(context.Background(), "1")
	competencySvc := competencyservice.New(learningrepo.NewRedisRepository(nil), competencyservice.NewMatrixManager(nil, nil, nil, ""), nil)
	graphSvc := graphservice.New(graphrepo.NewNeo4jRepository(nil))
	recommendSvc := recommendservice.New(graphSvc, learningrepo.NewRedisRepository(nil), nil, nil)
	service := New(competencySvc, recommendSvc)
	resp, err := service.GetSummary(ctx, &authentity.User{ID: 1, Name: "学生A", Role: "student", Target: "算法工程师"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.QuickStats) == 0 || len(resp.RecommendedPath) == 0 {
		t.Fatalf("unexpected summary response: %+v", resp)
	}
}
