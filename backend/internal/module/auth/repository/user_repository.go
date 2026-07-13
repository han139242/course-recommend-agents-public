package repository

import (
	"context"

	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
)

// UserRepository 定义认证模块依赖的用户仓储接口。
//
// 认证只关心按唯一标识读取用户，不直接暴露写操作，
// 这样可以把账户维护职责留在更合适的模块中，避免认证服务承担额外状态变更责任。
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByID(ctx context.Context, id int64) (*entity.User, error)
	CreateStudent(ctx context.Context, user *entity.User) error
}
