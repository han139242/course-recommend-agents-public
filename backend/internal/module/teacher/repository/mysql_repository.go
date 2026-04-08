package repository

import (
	"context"
	"errors"
	"strings"

	teacherentity "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/entity"
	teachermodel "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/model"
	usermodel "github.com/han139242/course-recommend-agents/backend/internal/module/user/model"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// MySQLRepository 提供教师端基于 MySQL 的仓储实现。
type MySQLRepository struct {
	db *gorm.DB
}

// NewMySQLRepository 创建教师端 MySQL 仓储。
func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

type classroomRecord struct {
	teachermodel.Classroom
	GradeValue   *string `gorm:"column:grade"`
	StudentCount int64   `gorm:"column:student_count"`
}

type studentRecord struct {
	usermodel.User
	ClassName *string `gorm:"column:class_name"`
}

// CreateClassroom 创建班级。
func (r *MySQLRepository) CreateClassroom(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error) {
	classroom := teachermodel.Classroom{
		Name:      strings.TrimSpace(name),
		TeacherID: teacherID,
		Status:    StatusActive,
	}
	if grade != "" {
		gradeValue := strings.TrimSpace(grade)
		classroom.Grade = &gradeValue
	}

	if err := r.db.WithContext(ctx).Create(&classroom).Error; err != nil {
		return nil, translateWriteError(err)
	}

	return &teacherentity.Classroom{
		ID:           classroom.ID,
		Name:         classroom.Name,
		Grade:        stringPointerValue(classroom.Grade),
		TeacherID:    classroom.TeacherID,
		StudentCount: 0,
		Status:       classroom.Status,
		CreatedAt:    classroom.CreatedAt,
	}, nil
}

// ListClassrooms 查询班级列表。
func (r *MySQLRepository) ListClassrooms(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error) {
	var total int64
	baseQuery := r.classroomsQuery(ctx, teacherID)
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []classroomRecord
	if err := baseQuery.
		Order("classrooms.created_at DESC, classrooms.id DESC").
		Offset(pagination.Offset()).
		Limit(pagination.PageSize).
		Scan(&records).Error; err != nil {
		return nil, 0, err
	}

	items := make([]teacherentity.Classroom, 0, len(records))
	for _, record := range records {
		items = append(items, teacherentity.Classroom{
			ID:           record.ID,
			Name:         record.Name,
			Grade:        stringPointerValue(record.GradeValue),
			TeacherID:    record.TeacherID,
			StudentCount: record.StudentCount,
			Status:       record.Status,
			CreatedAt:    record.CreatedAt,
		})
	}
	return items, total, nil
}

// GetClassroomByID 查询单个班级。
func (r *MySQLRepository) GetClassroomByID(ctx context.Context, teacherID int64, classID int64) (*teacherentity.Classroom, error) {
	var record classroomRecord
	if err := r.classroomsQuery(ctx, teacherID).Where("classrooms.id = ?", classID).Take(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrClassroomNotFound
		}
		return nil, err
	}

	return &teacherentity.Classroom{
		ID:           record.ID,
		Name:         record.Name,
		Grade:        stringPointerValue(record.GradeValue),
		TeacherID:    record.TeacherID,
		StudentCount: record.StudentCount,
		Status:       record.Status,
		CreatedAt:    record.CreatedAt,
	}, nil
}

// BatchCreateStudents 批量创建学生。
func (r *MySQLRepository) BatchCreateStudents(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error) {
	createdCount := 0
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var classroom teachermodel.Classroom
		if err := tx.Where("id = ? AND teacher_id = ? AND status = ?", classID, teacherID, StatusActive).Take(&classroom).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassroomNotFound
			}
			return err
		}

		users := make([]usermodel.User, 0, len(items))
		for _, item := range items {
			studentNo := strings.TrimSpace(item.StudentNo)
			users = append(users, usermodel.User{
				Name:         strings.TrimSpace(item.Name),
				Email:        strings.ToLower(strings.TrimSpace(item.Email)),
				PasswordHash: item.PasswordHash,
				Role:         RoleStudent,
				StudentNo:    &studentNo,
				Target:       strings.TrimSpace(item.Target),
				ClassID:      &classID,
				Status:       StatusActive,
				CreatedBy:    &item.CreatedBy,
			})
		}

		if err := tx.CreateInBatches(users, 100).Error; err != nil {
			return translateWriteError(err)
		}
		createdCount = len(users)
		return nil
	}); err != nil {
		return 0, err
	}
	return createdCount, nil
}

// ListStudents 查询班级学生列表。
func (r *MySQLRepository) ListStudents(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error) {
	var classroom teachermodel.Classroom
	if err := r.db.WithContext(ctx).Where("id = ? AND teacher_id = ? AND status = ?", classID, teacherID, StatusActive).Take(&classroom).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, ErrClassroomNotFound
		}
		return nil, 0, err
	}

	baseQuery := r.studentsQuery(ctx, classID)

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []studentRecord
	if err := baseQuery.
		Order("users.created_at DESC, users.id DESC").
		Offset(pagination.Offset()).
		Limit(pagination.PageSize).
		Scan(&records).Error; err != nil {
		return nil, 0, err
	}

	items := make([]teacherentity.Student, 0, len(records))
	for _, record := range records {
		items = append(items, toStudentEntity(record))
	}
	return items, total, nil
}

// GetStudentByID 查询单个学生档案。
func (r *MySQLRepository) GetStudentByID(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error) {
	var record studentRecord
	if err := r.db.WithContext(ctx).
		Table("users").
		Select("users.*, classrooms.name AS class_name").
		Joins("INNER JOIN classrooms ON classrooms.id = users.class_id").
		Where("users.id = ? AND users.role = ? AND users.status = ?", studentID, RoleStudent, StatusActive).
		Where("classrooms.teacher_id = ? AND classrooms.status = ?", teacherID, StatusActive).
		Take(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStudentNotFound
		}
		return nil, err
	}
	student := toStudentEntity(record)
	return &student, nil
}

func (r *MySQLRepository) classroomsQuery(ctx context.Context, teacherID int64) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("classrooms").
		Select("classrooms.*, classrooms.grade, COUNT(users.id) AS student_count").
		Joins("LEFT JOIN users ON users.class_id = classrooms.id AND users.role = ? AND users.status = ?", RoleStudent, StatusActive).
		Where("classrooms.teacher_id = ? AND classrooms.status = ?", teacherID, StatusActive).
		Group("classrooms.id")
}

func (r *MySQLRepository) studentsQuery(ctx context.Context, classID int64) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("users").
		Select("users.*, classrooms.name AS class_name").
		Joins("INNER JOIN classrooms ON classrooms.id = users.class_id").
		Where("users.class_id = ? AND users.role = ? AND users.status = ?", classID, RoleStudent, StatusActive)
}

// IsActiveTeacher 校验用户当前是否仍为启用中的教师。
func (r *MySQLRepository) IsActiveTeacher(ctx context.Context, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&usermodel.User{}).
		Where("id = ? AND role = ? AND status = ?", userID, RoleTeacher, StatusActive).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func translateWriteError(err error) error {
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		message := strings.ToLower(mysqlErr.Message)
		switch {
		case strings.Contains(message, "uk_users_email"):
			return ErrDuplicateEmail
		case strings.Contains(message, "uk_users_student_no"):
			return ErrDuplicateStudentNo
		case strings.Contains(message, "uk_class") || strings.Contains(message, "classroom"):
			return ErrDuplicateClassroomName
		default:
			return err
		}
	}
	return err
}

func toStudentEntity(record studentRecord) teacherentity.Student {
	return teacherentity.Student{
		ID:        record.ID,
		Name:      record.Name,
		Email:     record.Email,
		Role:      record.Role,
		Title:     record.Title,
		StudentNo: stringPointerValue(record.StudentNo),
		ClassID:   int64PointerValue(record.ClassID),
		ClassName: stringPointerValue(record.ClassName),
		Target:    record.Target,
		Status:    record.Status,
		CreatedAt: record.CreatedAt,
	}
}

func int64PointerValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
