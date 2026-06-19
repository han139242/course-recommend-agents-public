package repository

import (
	"context"
	"errors"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/module/profile/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("learning profile not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Enabled() bool {
	return r != nil && r.db != nil
}

func (r *Repository) GetCurrent(ctx context.Context, studentID int64) (*entity.LearningProfile, []entity.ProfileDimension, error) {
	var profile entity.LearningProfile
	err := r.db.WithContext(ctx).Where("student_id = ?", studentID).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	var dimensions []entity.ProfileDimension
	if err := r.db.WithContext(ctx).Where("student_id = ?", studentID).Order("id ASC").Find(&dimensions).Error; err != nil {
		return nil, nil, err
	}
	return &profile, dimensions, nil
}

func (r *Repository) UpsertProfile(ctx context.Context, profile *entity.LearningProfile, dimensions []entity.ProfileDimension, versionPayload string) (*entity.LearningProfile, []entity.ProfileDimension, error) {
	now := time.Now()
	profile.UpdatedAt = now
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "student_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"summary",
				"source_text",
				"updated_at",
			}),
		}).Create(profile).Error; err != nil {
			return err
		}

		var stored entity.LearningProfile
		if err := tx.Where("student_id = ?", profile.StudentID).First(&stored).Error; err != nil {
			return err
		}
		profile.ID = stored.ID
		profile.CreatedAt = stored.CreatedAt
		profile.UpdatedAt = stored.UpdatedAt

		for index := range dimensions {
			dimensions[index].ProfileID = stored.ID
			dimensions[index].StudentID = profile.StudentID
			dimensions[index].UpdatedAt = now
			if dimensions[index].CreatedAt.IsZero() {
				dimensions[index].CreatedAt = now
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "student_id"}, {Name: "dimension_key"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"profile_id",
					"dimension_name",
					"dimension_value",
					"confidence",
					"evidence",
					"updated_at",
				}),
			}).Create(&dimensions[index]).Error; err != nil {
				return err
			}
		}

		if versionPayload != "" {
			version := entity.ProfileVersion{
				ProfileID:   stored.ID,
				StudentID:   profile.StudentID,
				Summary:     profile.Summary,
				PayloadJSON: versionPayload,
				CreatedAt:   now,
			}
			if err := tx.Create(&version).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return r.GetCurrent(ctx, profile.StudentID)
}
