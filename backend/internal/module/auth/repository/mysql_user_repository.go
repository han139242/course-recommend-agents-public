package repository

import (
	"context"
	"errors"

	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
	classroommodel "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/model"
	usermodel "github.com/han139242/course-recommend-agents/backend/internal/module/user/model"
	"gorm.io/gorm"
)

// MySQLUserRepository 提供基于 MySQL 的用户仓储实现。
type MySQLUserRepository struct {
	db *gorm.DB
}

// NewMySQLUserRepository 创建 MySQL 用户仓储。
func NewMySQLUserRepository(db *gorm.DB) *MySQLUserRepository {
	return &MySQLUserRepository{db: db}
}

type userWithClass struct {
	usermodel.User
	ClassName *string `gorm:"column:class_name"`
}

// GetByEmail 根据邮箱查找用户。
func (r *MySQLUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	var record userWithClass
	if err := r.baseQuery(ctx).Where("users.email = ?", email).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toEntity(&record), nil
}

// GetByID 根据用户 ID 查找用户。
func (r *MySQLUserRepository) GetByID(ctx context.Context, id int64) (*entity.User, error) {
	var record userWithClass
	if err := r.baseQuery(ctx).Where("users.id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toEntity(&record), nil
}

func (r *MySQLUserRepository) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("users").
		Select("users.*, classrooms.name AS class_name").
		Joins("LEFT JOIN classrooms ON classrooms.id = users.class_id")
}

func toEntity(record *userWithClass) *entity.User {
	return &entity.User{
		ID:           record.ID,
		Name:         record.Name,
		Email:        record.Email,
		PasswordHash: record.PasswordHash,
		Role:         record.Role,
		Status:       record.Status,
		Title:        record.Title,
		StudentNo:    record.StudentNo,
		ClassID:      record.ClassID,
		ClassName:    record.ClassName,
		Target:       record.Target,
		CreatedAt:    record.CreatedAt,
	}
}

var _ = classroommodel.Classroom{}
