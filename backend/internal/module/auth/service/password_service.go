package service

import sharedpassword "github.com/han139242/course-recommend-agents/backend/pkg/password"

// ErrPasswordMismatch 表示密码不匹配。
var ErrPasswordMismatch = sharedpassword.ErrPasswordMismatch

// PasswordService 负责密码哈希与校验。
//
// 这里直接复用共享密码服务，避免 auth 模块自行维护一套散列策略，
// 从而保证注册、登录、重置密码等链路在算法和错误语义上保持一致。
type PasswordService = sharedpassword.Service

// NewPasswordService 创建密码服务。
//
// 认证模块通过统一入口获取密码能力，便于未来替换哈希参数或实现时不影响上层调用。
func NewPasswordService() *PasswordService {
	return sharedpassword.NewService()
}
