package service

import "golang.org/x/crypto/bcrypt"

// PasswordService 负责密码哈希与校验。
type PasswordService struct{}

// NewPasswordService 创建密码服务。
func NewPasswordService() *PasswordService {
	return &PasswordService{}
}

// Verify 校验明文密码与哈希密码是否匹配。
func (s *PasswordService) Verify(password string, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
