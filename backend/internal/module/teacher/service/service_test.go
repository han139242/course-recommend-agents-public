package service

import (
	"context"
	"errors"
	"testing"
	"time"

	teacherdto "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/dto"
	teacherentity "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/entity"
	teacherrepo "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
	sharedpassword "github.com/han139242/course-recommend-agents/backend/pkg/password"
)

type teacherRepoStub struct {
	isActiveTeacher func(ctx context.Context, userID int64) (bool, error)
	createClassroom func(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error)
	listClassrooms  func(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error)
	batchStudents   func(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error)
	listStudents    func(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error)
	getStudentByID  func(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error)
}

func (s *teacherRepoStub) CreateClassroom(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error) {
	return s.createClassroom(ctx, teacherID, name, grade)
}

func (s *teacherRepoStub) ListClassrooms(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error) {
	return s.listClassrooms(ctx, teacherID, pagination)
}

func (s *teacherRepoStub) GetClassroomByID(ctx context.Context, teacherID int64, classID int64) (*teacherentity.Classroom, error) {
	return nil, nil
}

func (s *teacherRepoStub) BatchCreateStudents(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error) {
	return s.batchStudents(ctx, teacherID, classID, items)
}

func (s *teacherRepoStub) ListStudents(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error) {
	return s.listStudents(ctx, teacherID, classID, pagination)
}

func (s *teacherRepoStub) GetStudentByID(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error) {
	return s.getStudentByID(ctx, teacherID, studentID)
}

func (s *teacherRepoStub) IsActiveTeacher(ctx context.Context, userID int64) (bool, error) {
	return s.isActiveTeacher(ctx, userID)
}

func newTeacherContext(userID int64) context.Context {
	ctx := context.Background()
	ctx = ctxmeta.WithUserID(ctx, "1")
	if userID != 1 {
		ctx = ctxmeta.WithUserID(ctx, "2")
	}
	return ctx
}

func TestCreateClassroomSuccess(t *testing.T) {
	repo := &teacherRepoStub{
		isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) {
			return userID == 1, nil
		},
		createClassroom: func(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error) {
			if teacherID != 1 {
				t.Fatalf("unexpected teacherID: %d", teacherID)
			}
			if name != "AI 一班" {
				t.Fatalf("unexpected name: %s", name)
			}
			return &teacherentity.Classroom{ID: 10, Name: name, Grade: grade, StudentCount: 0, Status: teacherrepo.StatusActive, CreatedAt: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}, nil
		},
		listClassrooms: func(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error) { return nil, 0, nil },
		batchStudents:  func(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error) { return 0, nil },
		listStudents:   func(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error) { return nil, 0, nil },
		getStudentByID: func(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error) { return nil, nil },
	}
	service := New(repo, sharedpassword.NewService())
	resp, err := service.CreateClassroom(newTeacherContext(1), teacherdto.CreateClassroomRequest{Name: "AI 一班", Grade: "2026 春"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Classroom.ID != 10 {
		t.Fatalf("unexpected classroom id: %d", resp.Classroom.ID)
	}
}

func TestCreateClassroomDuplicate(t *testing.T) {
	repo := &teacherRepoStub{
		isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return true, nil },
		createClassroom: func(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error) {
			return nil, teacherrepo.ErrDuplicateClassroomName
		},
		listClassrooms: func(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error) { return nil, 0, nil },
		batchStudents:  func(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error) { return 0, nil },
		listStudents:   func(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error) { return nil, 0, nil },
		getStudentByID: func(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error) { return nil, nil },
	}
	service := New(repo, sharedpassword.NewService())
	_, err := service.CreateClassroom(newTeacherContext(1), teacherdto.CreateClassroomRequest{Name: "AI 一班"})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 409 {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestBatchCreateStudentsRejectDuplicateInRequest(t *testing.T) {
	repo := &teacherRepoStub{
		isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return true, nil },
		createClassroom: func(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error) { return nil, nil },
		listClassrooms:  func(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error) { return nil, 0, nil },
		batchStudents: func(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error) {
			t.Fatal("batch create should not be called when request has duplicates")
			return 0, nil
		},
		listStudents:   func(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error) { return nil, 0, nil },
		getStudentByID: func(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error) { return nil, nil },
	}
	service := New(repo, sharedpassword.NewService())
	_, err := service.BatchCreateStudents(newTeacherContext(1), 101, teacherdto.BatchCreateStudentsRequest{Items: []teacherdto.CreateStudentItem{{Name: "A", StudentNo: "1001", Email: "same@example.com", Password: "123456"}, {Name: "B", StudentNo: "1002", Email: "same@example.com", Password: "123456"}}})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 409 {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestGetStudentDetailSuccess(t *testing.T) {
	repo := &teacherRepoStub{
		isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return true, nil },
		createClassroom: func(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error) { return nil, nil },
		listClassrooms:  func(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error) { return nil, 0, nil },
		batchStudents:   func(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error) { return 0, nil },
		listStudents:    func(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error) { return nil, 0, nil },
		getStudentByID: func(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error) {
			title := ""
			return &teacherentity.Student{ID: studentID, Name: "学生A", Email: "a@example.com", Role: teacherrepo.RoleStudent, Title: &title, StudentNo: "20260001", ClassID: 101, ClassName: "AI 一班", Target: "算法工程师", Status: teacherrepo.StatusActive, CreatedAt: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}, nil
		},
	}
	service := New(repo, sharedpassword.NewService())
	resp, err := service.GetStudentDetail(newTeacherContext(1), 201)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != 201 || resp.ClassName != "AI 一班" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRequireTeacherForbiddenWhenRoleChanged(t *testing.T) {
	repo := &teacherRepoStub{
		isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return false, nil },
		createClassroom: func(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error) { return nil, nil },
		listClassrooms:  func(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error) { return nil, 0, nil },
		batchStudents:   func(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error) { return 0, nil },
		listStudents:    func(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error) { return nil, 0, nil },
		getStudentByID:  func(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error) { return nil, nil },
	}
	service := New(repo, sharedpassword.NewService())
	_, err := service.ListClassrooms(newTeacherContext(1), teacherdto.PaginationQuery{})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 403 {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestRequireTeacherDependencyError(t *testing.T) {
	repo := &teacherRepoStub{
		isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return false, errors.New("db down") },
		createClassroom: func(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error) { return nil, nil },
		listClassrooms:  func(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error) { return nil, 0, nil },
		batchStudents:   func(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error) { return 0, nil },
		listStudents:    func(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error) { return nil, 0, nil },
		getStudentByID:  func(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error) { return nil, nil },
	}
	service := New(repo, sharedpassword.NewService())
	_, err := service.ListClassrooms(newTeacherContext(1), teacherdto.PaginationQuery{})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 503 {
		t.Fatalf("expected dependency unavailable error, got %v", err)
	}
}

func TestGetStudentDetailNotFoundUsesStudentSemantic(t *testing.T) {
	repo := &teacherRepoStub{
		isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return true, nil },
		createClassroom: func(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error) { return nil, nil },
		listClassrooms:  func(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error) { return nil, 0, nil },
		batchStudents:   func(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error) { return 0, nil },
		listStudents:    func(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error) { return nil, 0, nil },
		getStudentByID:  func(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error) { return nil, teacherrepo.ErrStudentNotFound },
	}
	service := New(repo, sharedpassword.NewService())
	_, err := service.GetStudentDetail(newTeacherContext(1), 999)
	if err == nil || apperr.AsAppError(err).HTTPStatus != 404 {
		t.Fatalf("expected not found error, got %v", err)
	}
	if !errors.Is(apperr.AsAppError(err).Cause, teacherrepo.ErrStudentNotFound) {
		t.Fatalf("expected student semantic cause, got %v", apperr.AsAppError(err).Cause)
	}
}
