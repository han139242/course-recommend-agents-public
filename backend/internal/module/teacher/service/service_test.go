package service

import (
	"context"
	"errors"
	"testing"
	"time"

	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
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

type stateReaderStub struct {
	getStudentKnowledgeStatus func(ctx context.Context, studentID int64, knowledgeID string, page int, pageSize int) (map[string]any, error)
	getLearningProgress       func(ctx context.Context, studentID int64) (*learningdto.LearningProgressResponse, bool, error)
}

func (s *stateReaderStub) GetStudentKnowledgeStatus(ctx context.Context, studentID int64, knowledgeID string, page int, pageSize int) (map[string]any, error) {
	return s.getStudentKnowledgeStatus(ctx, studentID, knowledgeID, page, pageSize)
}

func (s *stateReaderStub) GetLearningProgress(ctx context.Context, studentID int64) (*learningdto.LearningProgressResponse, bool, error) {
	return s.getLearningProgress(ctx, studentID)
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
	service := New(repo, sharedpassword.NewService(), nil)
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
	service := New(repo, sharedpassword.NewService(), nil)
	_, err := service.CreateClassroom(newTeacherContext(1), teacherdto.CreateClassroomRequest{Name: "AI 一班"})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 409 {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

// TestBatchCreateStudentsRejectDuplicateInRequest 验证请求内重复数据会在进入仓储前被拒绝。
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
	service := New(repo, sharedpassword.NewService(), nil)
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
	service := New(repo, sharedpassword.NewService(), nil)
	resp, err := service.GetStudentDetail(newTeacherContext(1), 201)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != 201 || resp.ClassName != "AI 一班" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// TestRequireTeacherForbiddenWhenRoleChanged 验证服务层会以数据库实时角色为准，而不是仅信任 token 快照。
func TestRequireTeacherForbiddenWhenRoleChanged(t *testing.T) {
	repo := &teacherRepoStub{
		isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return false, nil },
		createClassroom: func(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error) { return nil, nil },
		listClassrooms:  func(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error) { return nil, 0, nil },
		batchStudents:   func(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error) { return 0, nil },
		listStudents:    func(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error) { return nil, 0, nil },
		getStudentByID:  func(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error) { return nil, nil },
	}
	service := New(repo, sharedpassword.NewService(), nil)
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
	service := New(repo, sharedpassword.NewService(), nil)
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
	service := New(repo, sharedpassword.NewService(), nil)
	_, err := service.GetStudentDetail(newTeacherContext(1), 999)
	if err == nil || apperr.AsAppError(err).HTTPStatus != 404 {
		t.Fatalf("expected not found error, got %v", err)
	}
	if !errors.Is(apperr.AsAppError(err).Cause, teacherrepo.ErrStudentNotFound) {
		t.Fatalf("expected student semantic cause, got %v", apperr.AsAppError(err).Cause)
	}
}

func TestGetStudentKnowledgeStatusRejectsUnknownKnowledgeID(t *testing.T) {
	repo := &teacherRepoStub{
		isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return true, nil },
		createClassroom: func(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error) { return nil, nil },
		listClassrooms:  func(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error) { return nil, 0, nil },
		batchStudents:   func(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error) { return 0, nil },
		listStudents:    func(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error) { return nil, 0, nil },
		getStudentByID: func(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error) {
			return &teacherentity.Student{ID: studentID, Name: "学生A", ClassID: 101, ClassName: "AI 一班"}, nil
		},
	}
	stateReader := &stateReaderStub{getStudentKnowledgeStatus: func(ctx context.Context, studentID int64, knowledgeID string, page int, pageSize int) (map[string]any, error) {
		return nil, apperr.ErrNotFound
	}}
	service := New(repo, sharedpassword.NewService(), stateReader)
	_, err := service.GetStudentKnowledgeStatus(newTeacherContext(1), 201, "kp_missing", teacherdto.PaginationQuery{})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 404 {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestGetStudentKnowledgeStatusUsesRealPagination(t *testing.T) {
	repo := &teacherRepoStub{
		isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return true, nil },
		createClassroom: func(ctx context.Context, teacherID int64, name string, grade string) (*teacherentity.Classroom, error) { return nil, nil },
		listClassrooms:  func(ctx context.Context, teacherID int64, pagination teacherentity.Pagination) ([]teacherentity.Classroom, int64, error) { return nil, 0, nil },
		batchStudents:   func(ctx context.Context, teacherID int64, classID int64, items []teacherentity.CreateStudentInput) (int, error) { return 0, nil },
		listStudents:    func(ctx context.Context, teacherID int64, classID int64, pagination teacherentity.Pagination) ([]teacherentity.Student, int64, error) { return nil, 0, nil },
		getStudentByID: func(ctx context.Context, teacherID int64, studentID int64) (*teacherentity.Student, error) {
			return &teacherentity.Student{ID: studentID, Name: "学生A", ClassID: 101, ClassName: "AI 一班"}, nil
		},
	}
	stateReader := &stateReaderStub{getStudentKnowledgeStatus: func(ctx context.Context, studentID int64, knowledgeID string, page int, pageSize int) (map[string]any, error) {
		return map[string]any{"list": []map[string]any{{"knowledgeId": "kp_1"}}, "total": 10, "page": page, "pageSize": pageSize}, nil
	}}
	service := New(repo, sharedpassword.NewService(), stateReader)
	resp, err := service.GetStudentKnowledgeStatus(newTeacherContext(1), 201, "", teacherdto.PaginationQuery{Page: 2, PageSize: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp["page"].(int) != 2 || resp["pageSize"].(int) != 5 {
		t.Fatalf("unexpected pagination payload: %+v", resp)
	}
}
