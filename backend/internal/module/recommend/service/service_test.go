package service

import (
	"context"
	"testing"
	"time"

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
	service := New(graphSvc, learningrepo.NewRedisRepository(nil), nil)
	resp, err := service.GetPath(ctx, recommenddto.PathQuery{Target: "kp_001", Limit: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Path) == 0 {
		t.Fatal("expected non-empty path")
	}
}

func TestInferLearnerProfileReturnsStrugglingStyle(t *testing.T) {
	profile := inferLearnerProfile([]studentKnowledgeState{
		{KnowledgeID: "a", MasteryScore: 22, CorrectRate: 0.32, AnsweredCount: 6},
		{KnowledgeID: "b", MasteryScore: 35, CorrectRate: 0.41, AnsweredCount: 4},
		{KnowledgeID: "c", MasteryScore: 48, CorrectRate: 0.57, AnsweredCount: 2},
	})

	if profile == nil {
		t.Fatal("expected learner profile")
	}
	if profile.Style != "struggling" {
		t.Fatalf("expected struggling style, got %q", profile.Style)
	}
}

func TestDetectLearningBottlenecksIncludesWeaknessAndTransfer(t *testing.T) {
	service := &Service{}
	progressList := []studentKnowledgeState{
		{
			KnowledgeID:    "weak-node",
			CanonicalName:  "Weak Node",
			MasteryScore:   18,
			CorrectRate:    0.33,
			AnsweredCount:  5,
			LastAnsweredAt: time.Now().AddDate(0, 0, -10).Format(time.RFC3339),
		},
		{
			KnowledgeID:    "review-node",
			CanonicalName:  "Review Node",
			MasteryScore:   78,
			CorrectRate:    0.82,
			AnsweredCount:  4,
			LastAnsweredAt: time.Now().AddDate(0, 0, -20).Format(time.RFC3339),
		},
	}

	insights := service.DetectLearningBottlenecks(context.Background(), recommenddto.PathQuery{}, progressList, map[string]studentKnowledgeState{}, []recommenddto.PathItemView{
		{
			KnowledgeID:  "transfer-node",
			Route:        "/knowledge/transfer-node",
			ReasonCode:   "cross_course_transfer",
			ReasonLabel:  "跨课程迁移",
			SourceCourse: "线性代数",
			TargetCourse: "深度学习",
			Detail:       "你可以把矩阵理解迁移到权重初始化。",
		},
	})

	if len(insights) == 0 {
		t.Fatal("expected proactive insights")
	}

	foundWeakness := false
	foundTransfer := false
	for _, item := range insights {
		if item.ReasonCode == "weakness" {
			foundWeakness = true
		}
		if item.ReasonCode == "cross_course_transfer" {
			foundTransfer = true
		}
	}

	if !foundWeakness {
		t.Fatal("expected weakness insight")
	}
	if !foundTransfer {
		t.Fatal("expected transfer insight")
	}
}
