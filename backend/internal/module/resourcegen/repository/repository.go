package repository

import (
	"context"
	"errors"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("generated resource not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateJob(ctx context.Context, job *entity.GenerationJob) error {
	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *Repository) UpdateJob(ctx context.Context, job *entity.GenerationJob) error {
	job.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(job).Error
}

func (r *Repository) CreateStep(ctx context.Context, step *entity.AgentStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}

// UpdateStep persists status, summary and timestamp changes for one agent step.
func (r *Repository) UpdateStep(ctx context.Context, step *entity.AgentStep) error {
	return r.db.WithContext(ctx).Save(step).Error
}

func (r *Repository) CreateResources(ctx context.Context, resources []entity.GeneratedResource) error {
	if len(resources) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&resources).Error
}

// CreateResource inserts a single generated resource and populates its ID, used
// by the background pipeline to persist resources incrementally.
func (r *Repository) CreateResource(ctx context.Context, resource *entity.GeneratedResource) error {
	return r.db.WithContext(ctx).Create(resource).Error
}

func (r *Repository) GetJob(ctx context.Context, studentID int64, jobID int64) (*entity.GenerationJob, []entity.AgentStep, []entity.GeneratedResource, error) {
	var job entity.GenerationJob
	err := r.db.WithContext(ctx).Where("id = ? AND student_id = ?", jobID, studentID).First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, nil, err
	}

	var steps []entity.AgentStep
	if err := r.db.WithContext(ctx).Where("job_id = ? AND student_id = ?", jobID, studentID).Order("id ASC").Find(&steps).Error; err != nil {
		return nil, nil, nil, err
	}

	var resources []entity.GeneratedResource
	if err := r.db.WithContext(ctx).Where("job_id = ? AND student_id = ?", jobID, studentID).Order("id ASC").Find(&resources).Error; err != nil {
		return nil, nil, nil, err
	}
	return &job, steps, resources, nil
}

func (r *Repository) ListResources(ctx context.Context, studentID int64, knowledgeID string, resourceType string, limit int) ([]entity.GeneratedResource, error) {
	if limit <= 0 {
		limit = 30
	}
	query := r.db.WithContext(ctx).Where("student_id = ?", studentID)
	if knowledgeID != "" {
		query = query.Where("knowledge_id = ?", knowledgeID)
	}
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	var resources []entity.GeneratedResource
	if err := query.Order("created_at DESC, id DESC").Limit(limit).Find(&resources).Error; err != nil {
		return nil, err
	}
	return resources, nil
}

func (r *Repository) GetResource(ctx context.Context, studentID int64, resourceID int64) (*entity.GeneratedResource, error) {
	var resource entity.GeneratedResource
	err := r.db.WithContext(ctx).Where("id = ? AND student_id = ?", resourceID, studentID).First(&resource).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

func (r *Repository) UpsertFeedback(ctx context.Context, feedback *entity.ResourceFeedback) (*entity.ResourceFeedback, error) {
	now := time.Now()
	feedback.UpdatedAt = now
	if feedback.CreatedAt.IsZero() {
		feedback.CreatedAt = now
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "resource_id"}, {Name: "student_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"job_id",
			"rating",
			"difficulty",
			"helpful",
			"completed",
			"comment",
			"next_action",
			"updated_at",
		}),
	}).Create(feedback).Error; err != nil {
		return nil, err
	}

	var stored entity.ResourceFeedback
	if err := r.db.WithContext(ctx).Where("resource_id = ? AND student_id = ?", feedback.ResourceID, feedback.StudentID).First(&stored).Error; err != nil {
		return nil, err
	}
	return &stored, nil
}

func (r *Repository) ListFeedback(ctx context.Context, studentID int64, limit int) ([]entity.ResourceFeedback, error) {
	if limit <= 0 {
		limit = 30
	}
	var feedback []entity.ResourceFeedback
	if err := r.db.WithContext(ctx).Where("student_id = ?", studentID).Order("updated_at DESC, id DESC").Limit(limit).Find(&feedback).Error; err != nil {
		return nil, err
	}
	return feedback, nil
}
