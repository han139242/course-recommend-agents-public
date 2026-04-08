package repository

import (
	"context"
	"errors"
	"time"

	learningmodel "github.com/han139242/course-recommend-agents/backend/internal/module/learning/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StateMySQLRepository 提供学习状态快照的 MySQL 访问能力。
type StateMySQLRepository struct {
	db *gorm.DB
}

// NewStateMySQLRepository 创建学习状态 MySQL 仓储。
func NewStateMySQLRepository(db *gorm.DB) *StateMySQLRepository {
	return &StateMySQLRepository{db: db}
}

// ListLearningProgress 查询学生当前学习进度快照。
func (r *StateMySQLRepository) ListLearningProgress(ctx context.Context, studentID int64) ([]learningmodel.LearningProgress, error) {
	var items []learningmodel.LearningProgress
	err := r.db.WithContext(ctx).
		Where("student_id = ?", studentID).
		Order("updated_at DESC, knowledge_id ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// UpsertLearningProgress 批量写入学习进度快照。
func (r *StateMySQLRepository) UpsertLearningProgress(ctx context.Context, progress []learningmodel.LearningProgress) error {
	if len(progress) == 0 {
		return nil
	}
	for index := range progress {
		progress[index].UpdatedAt = time.Now()
		if progress[index].CreatedAt.IsZero() {
			progress[index].CreatedAt = progress[index].UpdatedAt
		}
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "student_id"}, {Name: "knowledge_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"canonical_name", "mastery_score", "status", "correct_rate", "answered_count", "last_answered_at", "updated_at"}),
	}).Create(&progress).Error
}

var _ StateRepository = (*StateMySQLRepository)(nil)
var _ = errors.New
