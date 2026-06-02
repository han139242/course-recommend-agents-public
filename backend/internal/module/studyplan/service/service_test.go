package service

import (
	"context"
	"errors"
	"testing"
	"time"

	learningmodel "github.com/han139242/course-recommend-agents/backend/internal/module/learning/model"
	studyplandto "github.com/han139242/course-recommend-agents/backend/internal/module/studyplan/dto"
	studyplanentity "github.com/han139242/course-recommend-agents/backend/internal/module/studyplan/entity"
	studyplanrepo "github.com/han139242/course-recommend-agents/backend/internal/module/studyplan/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

type fakeRepository struct {
	profile *studyplanentity.GoalProfile
	plan    *studyplanentity.StudyPlan
}

func (r *fakeRepository) GetProfile(ctx context.Context, studentID int64) (*studyplanentity.GoalProfile, error) {
	if r.profile == nil {
		return nil, studyplanrepo.ErrNotFound
	}
	return r.profile, nil
}

func (r *fakeRepository) UpsertProfile(ctx context.Context, profile *studyplanentity.GoalProfile) (*studyplanentity.GoalProfile, error) {
	profile.ID = 1
	profile.UpdatedAt = time.Now()
	r.profile = profile
	return profile, nil
}

func (r *fakeRepository) GetCurrentPlan(ctx context.Context, studentID int64) (*studyplanentity.StudyPlan, error) {
	if r.plan == nil {
		return nil, studyplanrepo.ErrNotFound
	}
	return r.plan, nil
}

func (r *fakeRepository) UpsertPlan(ctx context.Context, plan *studyplanentity.StudyPlan) (*studyplanentity.StudyPlan, error) {
	plan.ID = 7
	r.plan = plan
	return plan, nil
}

type fakeStateRepo struct {
	rows []learningmodel.LearningProgress
}

func (r fakeStateRepo) ListLearningProgress(ctx context.Context, studentID int64) ([]learningmodel.LearningProgress, error) {
	return r.rows, nil
}

func (r fakeStateRepo) UpsertLearningProgress(ctx context.Context, progress []learningmodel.LearningProgress) error {
	return nil
}

func TestUpsertProfileNormalizesAndPersistsGoalProfile(t *testing.T) {
	repo := &fakeRepository{}
	service := New(repo, nil, nil)
	ctx := ctxmeta.WithUserID(context.Background(), "12")

	response, err := service.UpsertProfile(ctx, studyplandto.GoalProfileRequest{
		GradeLevel:         "Sophomore",
		GoalType:           "project_building",
		GoalText:           " 想做智能体项目 ",
		TargetDirection:    "智能体",
		LearnedTopics:      []string{"Python", "Python", "线性代数"},
		FoundationLevel:    "beginner",
		WeeklyHours:        8,
		LearningPreference: "project_first",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !response.HasCompletedProfile || response.Profile == nil {
		t.Fatal("expected completed profile")
	}
	if response.Profile.GradeLevel != "sophomore" {
		t.Fatalf("expected normalized grade, got %q", response.Profile.GradeLevel)
	}
	if len(response.Profile.LearnedTopics) != 2 {
		t.Fatalf("expected duplicate topics removed, got %#v", response.Profile.LearnedTopics)
	}
}

func TestGeneratePlanRequiresProfile(t *testing.T) {
	service := New(&fakeRepository{}, nil, nil)
	ctx := ctxmeta.WithUserID(context.Background(), "12")

	_, err := service.GeneratePlan(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, err) {
		t.Fatal("expected non-nil error")
	}
}

func TestGeneratePlanBuildsThreeStagesAndNextSteps(t *testing.T) {
	repo := &fakeRepository{profile: &studyplanentity.GoalProfile{
		ID:                 1,
		StudentID:          12,
		GradeLevel:         "sophomore",
		GoalType:           "project_building",
		GoalText:           "想做智能体项目",
		TargetDirection:    "智能体",
		LearnedTopicsJSON:  `["Python"]`,
		FoundationLevel:    "beginner",
		WeeklyHours:        6,
		LearningPreference: "project_first",
	}}
	service := New(repo, nil, fakeStateRepo{rows: []learningmodel.LearningProgress{
		{StudentID: 12, KnowledgeID: "linear-algebra", CanonicalName: "线性代数", MasteryScore: 35, CorrectRate: 0.4, AnsweredCount: 3},
	}})
	ctx := ctxmeta.WithUserID(context.Background(), "12")

	response, err := service.GeneratePlan(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Plan == nil {
		t.Fatal("expected generated plan")
	}
	if len(response.Plan.Stages) != 3 {
		t.Fatalf("expected three stages, got %d", len(response.Plan.Stages))
	}
	if len(response.Plan.NextSteps) == 0 || response.Plan.NextSteps[0].KnowledgeID != "linear-algebra" {
		t.Fatalf("expected weak knowledge as first next step, got %#v", response.Plan.NextSteps)
	}
}
