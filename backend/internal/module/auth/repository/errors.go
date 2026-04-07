package repository

import "errors"

var (
	// ErrUserNotFound 表示用户不存在。
	ErrUserNotFound = errors.New("user not found")
)
