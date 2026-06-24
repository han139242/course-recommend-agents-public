package repository

import (
	"context"
	"errors"
	"strings"
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

func (r *Repository) CreatePushItem(ctx context.Context, item *entity.ResourcePushItem) error {
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *Repository) CreateLearningEvent(ctx context.Context, event *entity.LearningEvent) error {
	now := time.Now()
	if event.OccurredAt.IsZero() {
		event.OccurredAt = now
	}
	event.CreatedAt = now
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *Repository) UpdatePushStatusForResource(ctx context.Context, studentID int64, resourceID int64, status string) error {
	if resourceID <= 0 || status == "" {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&entity.ResourcePushItem{}).
		Where("student_id = ? AND resource_id = ?", studentID, resourceID).
		Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
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

func (r *Repository) ListResources(ctx context.Context, studentID int64, knowledgeID string, resourceType string, keyword string, status string, sortOrder string, limit int) ([]entity.GeneratedResource, error) {
	if limit <= 0 {
		limit = 30
	}
	query := r.db.WithContext(ctx).Model(&entity.GeneratedResource{}).Select("generated_resources.*").Where("generated_resources.student_id = ?", studentID)
	if knowledgeID != "" {
		query = query.Where("generated_resources.knowledge_id = ?", knowledgeID)
	}
	if resourceType != "" {
		query = query.Where("generated_resources.resource_type = ?", resourceType)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"generated_resources.title LIKE ? OR generated_resources.knowledge_name LIKE ? OR generated_resources.course_title LIKE ? OR generated_resources.content LIKE ?",
			like,
			like,
			like,
			like,
		)
	}
	if status != "" {
		query = query.Joins("JOIN resource_push_items ON resource_push_items.resource_id = generated_resources.id AND resource_push_items.student_id = generated_resources.student_id").
			Where("resource_push_items.status = ?", status)
	}
	order := "generated_resources.created_at DESC, generated_resources.id DESC"
	switch strings.ToLower(strings.TrimSpace(sortOrder)) {
	case "oldest":
		order = "generated_resources.created_at ASC, generated_resources.id ASC"
	case "title":
		order = "generated_resources.title ASC, generated_resources.id DESC"
	}
	var resources []entity.GeneratedResource
	if err := query.Order(order).Limit(limit).Find(&resources).Error; err != nil {
		return nil, err
	}
	return resources, nil
}

func (r *Repository) ListPushStatuses(ctx context.Context, studentID int64, resourceIDs []int64) (map[int64]string, error) {
	statuses := map[int64]string{}
	if len(resourceIDs) == 0 {
		return statuses, nil
	}
	var items []entity.ResourcePushItem
	if err := r.db.WithContext(ctx).
		Where("student_id = ? AND resource_id IN ?", studentID, resourceIDs).
		Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ResourceID > 0 && item.Status != "" {
			statuses[item.ResourceID] = item.Status
		}
	}
	return statuses, nil
}

func (r *Repository) ListPushItems(ctx context.Context, studentID int64, limit int) ([]entity.ResourcePushItem, []entity.GeneratedResource, error) {
	if limit <= 0 {
		limit = 20
	}
	var items []entity.ResourcePushItem
	if err := r.db.WithContext(ctx).
		Where("student_id = ?", studentID).
		Order("status ASC, priority DESC, updated_at DESC, id DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, nil, err
	}
	if len(items) == 0 {
		return items, nil, nil
	}

	resourceIDs := make([]int64, 0, len(items))
	seen := map[int64]struct{}{}
	for _, item := range items {
		if item.ResourceID <= 0 {
			continue
		}
		if _, ok := seen[item.ResourceID]; ok {
			continue
		}
		seen[item.ResourceID] = struct{}{}
		resourceIDs = append(resourceIDs, item.ResourceID)
	}
	var resources []entity.GeneratedResource
	if len(resourceIDs) > 0 {
		if err := r.db.WithContext(ctx).
			Where("student_id = ? AND id IN ?", studentID, resourceIDs).
			Find(&resources).Error; err != nil {
			return nil, nil, err
		}
	}
	return items, resources, nil
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

func (r *Repository) ListLearningEvents(ctx context.Context, studentID int64, limit int) ([]entity.LearningEvent, error) {
	if limit <= 0 {
		limit = 30
	}
	var events []entity.LearningEvent
	if err := r.db.WithContext(ctx).
		Where("student_id = ?", studentID).
		Order("occurred_at DESC, id DESC").
		Limit(limit).
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
