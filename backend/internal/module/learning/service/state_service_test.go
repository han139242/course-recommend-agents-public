package service

import (
	"context"
	"testing"

	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
)

type stubPrerequisiteProvider struct {
	catalog *graphdto.CatalogResponse
	err     error
}

func (s stubPrerequisiteProvider) GetCatalog(context.Context) (*graphdto.CatalogResponse, error) {
	return s.catalog, s.err
}

func TestRelationAdjustedMasteryCapsWhenPrerequisiteWeak(t *testing.T) {
	score := relationAdjustedMastery(96, []float64{20})
	if score != 55 {
		t.Fatalf("expected weak prerequisite cap, got %.2f", score)
	}
}

func TestApplyPrerequisiteAdjustmentsUsesGraphRelations(t *testing.T) {
	service := NewStateService(nil, nil, stubPrerequisiteProvider{
		catalog: &graphdto.CatalogResponse{
			PrerequisiteEdges: []graphdto.PrerequisiteIndexEdgeView{
				{SourceRouteKnowledgeID: "matrix", TargetRouteKnowledgeID: "gradient"},
			},
		},
	})

	items := []learningdto.LearningProgressItem{
		{KnowledgeID: "matrix", MasteryScore: 25, Status: "progressing", AnsweredCount: 4},
		{KnowledgeID: "gradient", MasteryScore: 92, Status: "solid", AnsweredCount: 10},
	}

	adjusted := service.applyPrerequisiteAdjustments(context.Background(), items)
	if adjusted[1].MasteryScore != 55 {
		t.Fatalf("expected downstream mastery to be capped by prerequisite, got %.2f", adjusted[1].MasteryScore)
	}
	if adjusted[1].Status == "solid" {
		t.Fatalf("expected downstream status to be downgraded, got %s", adjusted[1].Status)
	}
	if items[1].MasteryScore != 92 {
		t.Fatalf("expected original slice to remain unchanged")
	}
}
