package entity

import "time"

// Classroom 表示班级领域实体。
type Classroom struct {
	ID           int64
	Name         string
	Grade        string
	TeacherID    int64
	StudentCount int64
	Status       string
	CreatedAt    time.Time
}

// Student 表示学生领域实体。
type Student struct {
	ID        int64
	Name      string
	Email     string
	Role      string
	Title     *string
	StudentNo string
	ClassID   int64
	ClassName string
	Target    string
	Status    string
	CreatedAt time.Time
}

// CreateStudentInput 表示创建学生输入。
type CreateStudentInput struct {
	Name         string
	Email        string
	PasswordHash string
	StudentNo    string
	Target       string
	ClassID      int64
	CreatedBy    int64
}

// Pagination 定义分页参数。
// 该结构仅在仓储层使用，调用方需先完成默认值与上限归一化，避免仓储层承担策略判断。
type Pagination struct {
	Page     int
	PageSize int
}

// Offset 返回分页偏移量。
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}
