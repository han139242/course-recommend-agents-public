package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrPasswordMismatch 表示密码不匹配。
	ErrPasswordMismatch = errors.New("password mismatch")
)

// Service 负责密码哈希与校验。
// 该服务屏蔽底层加密实现，便于业务层只依赖稳定接口。
type Service struct{}

// NewService 创建密码服务。
func NewService() *Service {
	return &Service{}
}

// Verify 校验明文密码与哈希密码是否匹配。
// 对 bcrypt 的不匹配错误做显式归一化，方便上层统一分支处理。
func (s *Service) Verify(password string, hashedPassword string) error {
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
func (s *Service) Hash(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}
