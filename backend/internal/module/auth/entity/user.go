package entity

import "time"

// User 表示当前阶段认证模块使用的用户领域实体。
//
// 该实体聚焦认证与会话建立所必需的信息，避免直接暴露底层持久化模型给服务层和处理器层。
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
