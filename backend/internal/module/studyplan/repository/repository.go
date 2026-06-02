package repository

import (
	"context"
	"errors"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/module/studyplan/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("study plan resource not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Enabled() bool {
	return r != nil && r.db != nil
}

func (r *Repository) GetProfile(ctx context.Context, studentID int64) (*entity.GoalProfile, error) {
	var item entity.GoalProfile
	err := r.db.WithContext(ctx).Where("student_id = ?", studentID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) UpsertProfile(ctx context.Context, profile *entity.GoalProfile) (*entity.GoalProfile, error) {
	now := time.Now()
	profile.UpdatedAt = now
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "student_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"grade_level",
			"major",
			"goal_type",
			"goal_text",
			"target_direction",
			"learned_topics_json",
			"foundation_level",
			"weekly_hours",
			"learning_preference",
			"updated_at",
		}),
	}).Create(profile).Error
	if err != nil {
		return nil, err
	}
	return r.GetProfile(ctx, profile.StudentID)
}

func (r *Repository) GetCurrentPlan(ctx context.Context, studentID int64) (*entity.StudyPlan, error) {
	var item entity.StudyPlan
	err := r.db.WithContext(ctx).Where("student_id = ?", studentID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) UpsertPlan(ctx context.Context, plan *entity.StudyPlan) (*entity.StudyPlan, error) {
	now := time.Now()
	plan.UpdatedAt = now
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "student_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"profile_id",
			"summary",
			"stages_json",
			"next_steps_json",
			"generated_at",
			"updated_at",
		}),
	}).Create(plan).Error
	if err != nil {
		return nil, err
	}
	return r.GetCurrentPlan(ctx, plan.StudentID)
}
