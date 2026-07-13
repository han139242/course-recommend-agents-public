package repository

import "errors"

var (
	// ErrUserNotFound 表示未找到目标用户。
	//
	// 仓储层统一返回该错误，便于服务层将“账号不存在”和“密码错误”映射到相同的外部语义，
	// 避免通过错误差异泄露账户是否存在。
	ErrUserNotFound = errors.New("user not found")
	// ErrUserAlreadyExists 表示邮箱已被注册。
	ErrUserAlreadyExists = errors.New("user already exists")
)
