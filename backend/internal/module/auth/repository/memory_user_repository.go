package repository

import (
	"context"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
)

// MemoryUserRepository 提供当前阶段的内存用户实现，后续可替换为 MySQL。
//
// 该实现主要用于早期联调或脱离数据库的测试场景，保证认证流程在基础设施未就绪时仍可验证。
type MemoryUserRepository struct {
	users []*entity.User
}

// NewMemoryUserRepository 创建内存用户仓储。
//
// 这里预置教师和学生两类样例账户，目的是覆盖不同角色在认证后的基础行为差异。
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
//
// 返回副本而不是内部切片中的原始对象，避免调用方误修改内存仓储中的种子数据。
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
//
// 与按邮箱查询保持同样的副本返回语义，确保不同读取路径下的数据隔离规则一致。
func (r *MemoryUserRepository) GetByID(_ context.Context, id int64) (*entity.User, error) {
	for _, user := range r.users {
		if user.ID == id {
			copied := *user
			return &copied, nil
		}
	}
	return nil, ErrUserNotFound
}
