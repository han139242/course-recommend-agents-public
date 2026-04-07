package repository

import (
	"context"

	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
)

// UserRepository 定义认证模块依赖的用户仓储接口。
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByID(ctx context.Context, id int64) (*entity.User, error)
}
