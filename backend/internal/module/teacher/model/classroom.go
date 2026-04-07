package model

import "time"

// Classroom 表示 classrooms 表的存储模型。
type Classroom struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string    `gorm:"column:name;type:varchar(128);not null"`
	Grade     *string   `gorm:"column:grade;type:varchar(64)"`
	TeacherID int64     `gorm:"column:teacher_id;not null;index:idx_classrooms_teacher_id"`
	Status    string    `gorm:"column:status;type:varchar(16);not null;default:'active';index:idx_classrooms_status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName 指定 classrooms 表名。
func (Classroom) TableName() string {
	return "classrooms"
}
