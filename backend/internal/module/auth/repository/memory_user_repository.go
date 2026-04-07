package repository

import (
	"context"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
)

// MemoryUserRepository 提供当前阶段的内存用户实现，后续可替换为 MySQL。
type MemoryUserRepository struct {
	users []*entity.User
}

// NewMemoryUserRepository 创建内存用户仓储。
func NewMemoryUserRepository(teacherPasswordHash string, studentPasswordHash string) *MemoryUserRepository {
	classID := int64(1)
	title := "副教授"
	studentNo := "20260001"
	className := "2026 春人工智能 1 班"
	createdAt := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)

	return &MemoryUserRepository{
		users: []*entity.User{
			{
				ID:           1,
				Name:         "张老师",
				Email:        "teacher@example.com",
				PasswordHash: teacherPasswordHash,
				Role:         "teacher",
				Status:       UserStatusActive,
				Title:        &title,
				Target:       "",
				CreatedAt:    createdAt,
			},
			{
				ID:           2,
				Name:         "李同学",
				Email:        "student@example.com",
				PasswordHash: studentPasswordHash,
				Role:         "student",
				Status:       UserStatusActive,
				StudentNo:    &studentNo,
				ClassID:      &classID,
				ClassName:    &className,
				Target:       "计算机视觉",
				CreatedAt:    createdAt,
			},
		},
	}
}

// GetByEmail 根据邮箱查找用户。
func (r *MemoryUserRepository) GetByEmail(_ context.Context, email string) (*entity.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			copied := *user
			return &copied, nil
		}
	}
	return nil, ErrUserNotFound
}

// GetByID 根据用户 ID 查找用户。
func (r *MemoryUserRepository) GetByID(_ context.Context, id int64) (*entity.User, error) {
	for _, user := range r.users {
		if user.ID == id {
			copied := *user
			return &copied, nil
		}
	}
	return nil, ErrUserNotFound
}
