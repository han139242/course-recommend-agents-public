package entity

import "time"

// User 表示当前阶段认证模块使用的用户领域实体。
type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	Role         string
	Status       string
	Title        *string
	StudentNo    *string
	ClassID      *int64
	ClassName    *string
	Target       string
	CreatedAt    time.Time
}
