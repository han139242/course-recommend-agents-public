package repository

import (
	"errors"

	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository 定义认证模块依赖的用户仓储接口。
type UserRepository interface {
	GetByEmail(email string) (*entity.User, error)
	GetByID(id int64) (*entity.User, error)
}

// MemoryUserRepository 提供当前阶段的内存用户实现，后续可替换为 MySQL。
type MemoryUserRepository struct {
	users []*entity.User
}

// NewMemoryUserRepository 创建内存用户仓储。
func NewMemoryUserRepository() *MemoryUserRepository {
	classID := int64(101)
	teacherPassword := mustHashPassword("123456")
	studentPassword := mustHashPassword("123456")
	return &MemoryUserRepository{
		users: []*entity.User{
			{
				ID:           1,
				Name:         "张老师",
				Email:        "teacher@example.com",
				Password:     teacherPassword,
				Role:         "teacher",
				Status:       "active",
				Title:        "副教授",
				Target:       "",
				CreatedAtISO: "2026-04-04T00:00:00Z",
			},
			{
				ID:           2,
				Name:         "李同学",
				Email:        "student@example.com",
				Password:     studentPassword,
				Role:         "student",
				Status:       "active",
				StudentNo:    "20260001",
				ClassID:      &classID,
				ClassName:    "2026 春人工智能 1 班",
				Target:       "计算机视觉",
				CreatedAtISO: "2026-04-04T00:00:00Z",
			},
		},
	}
}

func mustHashPassword(password string) string {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic("生成演示用户密码哈希失败")
	}
	return string(hashedPassword)
}

// GetByEmail 根据邮箱查找用户。
func (r *MemoryUserRepository) GetByEmail(email string) (*entity.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			copied := *user
			return &copied, nil
		}
	}
	return nil, errors.New("user not found")
}

// GetByID 根据用户 ID 查找用户。
func (r *MemoryUserRepository) GetByID(id int64) (*entity.User, error) {
	for _, user := range r.users {
		if user.ID == id {
			copied := *user
			return &copied, nil
		}
	}
	return nil, errors.New("user not found")
}
