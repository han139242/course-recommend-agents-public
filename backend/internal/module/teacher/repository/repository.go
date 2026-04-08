package repository

import (
	"context"

	"github.com/han139242/course-recommend-agents/backend/internal/module/teacher/entity"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

// TeacherRepository 定义教师端主数据仓储接口。
// 所有读取方法都必须显式带上教师作用域参数，避免仓储层遗漏租户边界导致越权读取。
type TeacherRepository interface {
	CreateClassroom(ctx context.Context, teacherID int64, name string, grade string) (*entity.Classroom, error)
	ListClassrooms(ctx context.Context, teacherID int64, pagination entity.Pagination) ([]entity.Classroom, int64, error)
	GetClassroomByID(ctx context.Context, teacherID int64, classID int64) (*entity.Classroom, error)
	BatchCreateStudents(ctx context.Context, teacherID int64, classID int64, items []entity.CreateStudentInput) (int, error)
	ListStudents(ctx context.Context, teacherID int64, classID int64, pagination entity.Pagination) ([]entity.Student, int64, error)
	GetStudentByID(ctx context.Context, teacherID int64, studentID int64) (*entity.Student, error)
	IsActiveTeacher(ctx context.Context, userID int64) (bool, error)
}

// CurrentUserID 读取当前登录用户 ID。
func CurrentUserID(ctx context.Context) (int64, bool) {
	return ctxmeta.CurrentUserID(ctx)
}
