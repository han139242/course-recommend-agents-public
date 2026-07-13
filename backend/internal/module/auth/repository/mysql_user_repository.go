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
//
// 该实现负责把认证所需的用户与班级信息一次性查出，避免上层服务为补全展示字段再触发额外查询。
type MySQLUserRepository struct {
	db *gorm.DB
}

// NewMySQLUserRepository 创建 MySQL 用户仓储。
//
// 仓储直接持有数据库句柄，具体事务与连接生命周期由上层基础设施统一管理。
func NewMySQLUserRepository(db *gorm.DB) *MySQLUserRepository {
	return &MySQLUserRepository{db: db}
}

// CreateStudent persists a public self-registered student account.
func (r *MySQLUserRepository) CreateStudent(ctx context.Context, user *entity.User) error {
	record := usermodel.User{
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         "student",
		Status:       UserStatusActive,
		Target:       "",
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return err
	}
	user.ID = record.ID
	user.Role = record.Role
	user.Status = record.Status
	user.CreatedAt = record.CreatedAt
	return nil
}

type userWithClass struct {
	usermodel.User
	ClassName *string `gorm:"column:class_name"`
}

// GetByEmail 根据邮箱查找用户。
//
// 认证使用邮箱作为登录主标识，因此查询未命中时统一转成领域内的未找到错误，
// 便于服务层按安全策略继续做错误收敛。
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
//
// 该查询主要服务于“当前用户信息”场景，需要以数据库最新状态为准，避免仅依赖令牌中的历史快照。
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
	// 认证场景读取频次高于写入频次，提前在基础查询中补齐关联字段，
	// 可以把查询约束集中在一处，避免不同读取方法遗漏相同的联表条件。
	return r.db.WithContext(ctx).
		Table("users").
		Select("users.*, classrooms.name AS class_name").
		Joins("LEFT JOIN classrooms ON classrooms.id = users.class_id")
}

func toEntity(record *userWithClass) *entity.User {
	// 仓储内部模型与领域对象分离，避免上层直接依赖 ORM 结构导致边界混乱。
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
