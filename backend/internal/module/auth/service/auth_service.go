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
type AuthService struct {
	repo            repository.UserRepository
	jwtService      *JWTService
	passwordService *PasswordService
}

// NewAuthService 创建认证服务。
func NewAuthService(repo repository.UserRepository, jwtService *JWTService, passwordService *PasswordService) *AuthService {
	return &AuthService{repo: repo, jwtService: jwtService, passwordService: passwordService}
}

// Login 执行登录逻辑。
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
	if value == nil {
		return ""
	}
	return *value
}
