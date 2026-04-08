package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/dto"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
)

// AuthService 提供认证相关业务能力。
//
// 该服务负责把账号查询、密码校验和令牌签发串成一个完整登录流程，
// 对外屏蔽底层存储与鉴权细节，便于在不影响接口层的前提下调整实现。
type AuthService struct {
	repo            repository.UserRepository
	jwtService      *JWTService
	passwordService *PasswordService
}

// NewAuthService 创建认证服务。
//
// 构造函数显式接收依赖，便于在测试中替换仓储或令牌实现，避免业务逻辑与基础设施强耦合。
func NewAuthService(repo repository.UserRepository, jwtService *JWTService, passwordService *PasswordService) *AuthService {
	return &AuthService{repo: repo, jwtService: jwtService, passwordService: passwordService}
}

// Login 执行登录逻辑。
//
// 登录前会先规范化邮箱输入，确保不同大小写或首尾空格不会影响账号匹配结果。
// 当账号不存在或密码错误时，对外统一返回未授权，降低账户枚举风险。
// 只有处于启用状态的用户才允许签发新令牌，避免停用账号继续建立新会话。
func (s *AuthService) Login(ctx context.Context, email string, password string) (*dto.LoginResponse, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	user, err := s.repo.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, apperr.ErrUnauthorized.WithCause(err)
		}
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	if err := s.passwordService.Verify(password, user.PasswordHash); err != nil {
		if errors.Is(err, ErrPasswordMismatch) {
			return nil, apperr.ErrUnauthorized.WithCause(err)
		}
		return nil, apperr.ErrInternal.WithCause(err)
	}

	if user.Status != repository.UserStatusActive {
		return nil, apperr.ErrForbidden.WithCause(errors.New("user disabled"))
	}

	token, expiredAt, err := s.jwtService.Sign(user)
	if err != nil {
		return nil, apperr.ErrInternal.WithCause(err)
	}

	return &dto.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(expiredAt).Seconds()),
		User:        buildUserView(user),
	}, nil
}

// GetCurrentUser 根据用户 ID 获取当前登录用户。
//
// 该方法依赖上游鉴权链路提供可信的用户 ID，服务层只负责按最新数据回查，
// 从而避免客户端长期持有的令牌载荷与数据库中的用户状态产生偏差。
func (s *AuthService) GetCurrentUser(ctx context.Context, userID int64) (*dto.AuthUserView, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, apperr.ErrNotFound.WithCause(err)
		}
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	if user.Status != repository.UserStatusActive {
		return nil, apperr.ErrForbidden.WithCause(errors.New("user disabled"))
	}

	userView := buildUserView(user)
	return &userView, nil
}

func buildUserView(user *entity.User) dto.AuthUserView {
	// 统一由服务层完成领域对象到响应对象的映射，避免处理器层散落字段拼装逻辑。
	return dto.AuthUserView{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		Title:     stringValueOrEmpty(user.Title),
		StudentNo: user.StudentNo,
		ClassID:   user.ClassID,
		ClassName: user.ClassName,
		Target:    user.Target,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}
}

func stringValueOrEmpty(value *string) string {
	// 响应层约定 title 始终返回字符串，避免前端在可空字段上做额外判空分支。
	if value == nil {
		return ""
	}
	return *value
}
