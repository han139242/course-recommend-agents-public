package service

import (
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
func (s *AuthService) Login(email string, password string) (*dto.LoginResponse, error) {
	user, err := s.repo.GetByEmail(strings.TrimSpace(email))
	if err != nil {
		return nil, apperr.ErrUnauthorized.WithCause(err)
	}

	if err := s.passwordService.Verify(password, user.Password); err != nil {
		return nil, apperr.ErrUnauthorized.WithCause(errors.New("password not match"))
	}

	if user.Status != "active" {
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
func (s *AuthService) GetCurrentUser(userID int64) (*dto.AuthUserView, error) {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, apperr.ErrNotFound.WithCause(err)
	}

	if user.Status != "active" {
		return nil, apperr.ErrForbidden.WithCause(errors.New("user disabled"))
	}

	userView := buildUserView(user)
	return &userView, nil
}

func buildUserView(user *entity.User) dto.AuthUserView {
	var studentNo *string
	if user.StudentNo != "" {
		studentNo = &user.StudentNo
	}

	var className *string
	if user.ClassName != "" {
		className = &user.ClassName
	}

	return dto.AuthUserView{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		Title:     user.Title,
		StudentNo: studentNo,
		ClassID:   user.ClassID,
		ClassName: className,
		Target:    user.Target,
		Status:    user.Status,
		CreatedAt: user.CreatedAtISO,
	}
}
