package service

import sharedpassword "github.com/han139242/course-recommend-agents/backend/pkg/password"

// ErrPasswordMismatch 表示密码不匹配。
var ErrPasswordMismatch = sharedpassword.ErrPasswordMismatch

// PasswordService 负责密码哈希与校验。
type PasswordService = sharedpassword.Service

// NewPasswordService 创建密码服务。
func NewPasswordService() *PasswordService {
	return sharedpassword.NewService()
}
