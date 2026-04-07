package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrPasswordMismatch 表示密码不匹配。
	ErrPasswordMismatch = errors.New("password mismatch")
)

// PasswordService 负责密码哈希与校验。
type PasswordService struct{}

// NewPasswordService 创建密码服务。
func NewPasswordService() *PasswordService {
	return &PasswordService{}
}

// Verify 校验明文密码与哈希密码是否匹配。
func (s *PasswordService) Verify(password string, hashedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err == nil {
		return nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrPasswordMismatch
	}
	return err
}

// Hash 生成密码哈希。
func (s *PasswordService) Hash(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}
