package model

import "time"

// User 表示 users 表的存储模型。
type User struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name         string    `gorm:"column:name;type:varchar(64);not null"`
	Email        string    `gorm:"column:email;type:varchar(128);not null;uniqueIndex:uk_users_email"`
	PasswordHash string    `gorm:"column:password_hash;type:varchar(255);not null"`
	Role         string    `gorm:"column:role;type:varchar(16);not null;index:idx_users_role"`
	Title        *string   `gorm:"column:title;type:varchar(64)"`
	StudentNo    *string   `gorm:"column:student_no;type:varchar(64);uniqueIndex:uk_users_student_no"`
	ClassID      *int64    `gorm:"column:class_id;index:idx_users_class_id"`
	Target       string    `gorm:"column:target;type:varchar(255);default:''"`
	Status       string    `gorm:"column:status;type:varchar(16);not null;default:'active';index:idx_users_status"`
	CreatedBy    *int64    `gorm:"column:created_by"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName 指定 users 表名。
func (User) TableName() string {
	return "users"
}
