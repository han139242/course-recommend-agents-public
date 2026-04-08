package service

import (
	"context"
	"errors"
	"strings"
	"time"

	teacherdto "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/dto"
	teacherentity "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/entity"
	teacherrepo "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	sharedpassword "github.com/han139242/course-recommend-agents/backend/pkg/password"
)

// TeacherService 提供教师端主数据业务能力。
type TeacherService struct {
	repo            teacherrepo.TeacherRepository
	passwordService *sharedpassword.Service
}

// New 创建教师端服务。
func New(repo teacherrepo.TeacherRepository, passwordService *sharedpassword.Service) *TeacherService {
	return &TeacherService{repo: repo, passwordService: passwordService}
}

// CreateClassroom 创建班级。
// 班级名称唯一性按“同一教师作用域”约束，跨教师允许重名；真正的唯一性由仓储层依赖数据库索引保障。
func (s *TeacherService) CreateClassroom(ctx context.Context, request teacherdto.CreateClassroomRequest) (*teacherdto.CreateClassroomResponse, error) {
	teacherID, err := s.requireTeacher(ctx)
	if err != nil {
		return nil, err
	}

	classroom, err := s.repo.CreateClassroom(ctx, teacherID, request.Name, request.Grade)
	if err != nil {
		if errors.Is(err, teacherrepo.ErrDuplicateClassroomName) {
			return nil, apperr.ErrConflict.WithCause(err)
		}
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return &teacherdto.CreateClassroomResponse{Classroom: toClassroomView(*classroom)}, nil
}

// ListClassrooms 查询班级列表。
// 分页参数会先归一化后再下沉到仓储层，确保接口返回的 page/pageSize 与实际 SQL 偏移量保持一致。
func (s *TeacherService) ListClassrooms(ctx context.Context, query teacherdto.PaginationQuery) (*teacherdto.ListClassroomsResponse, error) {
	teacherID, err := s.requireTeacher(ctx)
	if err != nil {
		return nil, err
	}

	normalized := query.Normalize()
	items, total, err := s.repo.ListClassrooms(ctx, teacherID, teacherentity.Pagination{Page: normalized.Page, PageSize: normalized.PageSize})
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	list := make([]teacherdto.ClassroomView, 0, len(items))
	for _, item := range items {
		list = append(list, toClassroomView(item))
	}

	return &teacherdto.ListClassroomsResponse{List: list, Total: total, Page: normalized.Page, PageSize: normalized.PageSize}, nil
}

// BatchCreateStudents 批量创建学生。
// 该流程会先在请求内做邮箱与学号去重，再交给仓储层开启事务批量写入，避免部分成功导致班级名册不一致。
func (s *TeacherService) BatchCreateStudents(ctx context.Context, classID int64, request teacherdto.BatchCreateStudentsRequest) (*teacherdto.BatchCreateStudentsResponse, error) {
	teacherID, err := s.requireTeacher(ctx)
	if err != nil {
		return nil, err
	}

	items, err := s.buildCreateStudentInputs(request.Items, classID, teacherID)
	if err != nil {
		return nil, err
	}

	createdCount, err := s.repo.BatchCreateStudents(ctx, teacherID, classID, items)
	if err != nil {
		switch {
		case errors.Is(err, teacherrepo.ErrClassroomNotFound):
			return nil, apperr.ErrNotFound.WithCause(err)
		case errors.Is(err, teacherrepo.ErrDuplicateEmail), errors.Is(err, teacherrepo.ErrDuplicateStudentNo), errors.Is(err, teacherrepo.ErrDuplicateClassroomName):
			return nil, apperr.ErrConflict.WithCause(err)
		default:
			return nil, apperr.ErrDependencyUnavailable.WithCause(err)
		}
	}

	return &teacherdto.BatchCreateStudentsResponse{ClassID: classID, CreatedCount: createdCount}, nil
}

// ListStudents 查询班级学生列表。
func (s *TeacherService) ListStudents(ctx context.Context, classID int64, query teacherdto.PaginationQuery) (*teacherdto.ListStudentsResponse, error) {
	teacherID, err := s.requireTeacher(ctx)
	if err != nil {
		return nil, err
	}

	normalized := query.Normalize()
	items, total, err := s.repo.ListStudents(ctx, teacherID, classID, teacherentity.Pagination{Page: normalized.Page, PageSize: normalized.PageSize})
	if err != nil {
		if errors.Is(err, teacherrepo.ErrClassroomNotFound) {
			return nil, apperr.ErrNotFound.WithCause(err)
		}
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	list := make([]teacherdto.StudentView, 0, len(items))
	for _, item := range items {
		list = append(list, toStudentView(item))
	}

	return &teacherdto.ListStudentsResponse{List: list, Total: total, Page: normalized.Page, PageSize: normalized.PageSize}, nil
}

// GetStudentDetail 查询单个学生档案。
func (s *TeacherService) GetStudentDetail(ctx context.Context, studentID int64) (*teacherdto.StudentDetailView, error) {
	teacherID, err := s.requireTeacher(ctx)
	if err != nil {
		return nil, err
	}

	student, err := s.repo.GetStudentByID(ctx, teacherID, studentID)
	if err != nil {
		if errors.Is(err, teacherrepo.ErrStudentNotFound) {
			return nil, apperr.ErrNotFound.WithCause(err)
		}
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	view := toStudentDetailView(*student)
	return &view, nil
}

// requireTeacher 统一收口教师端接口的身份校验。
// 中间件写入的 role 是 token 签发时的快照，这里会按 userID 回查数据库中的当前角色与状态，避免旧 token 延迟生效。
func (s *TeacherService) requireTeacher(ctx context.Context) (int64, error) {
	userID, ok := teacherrepo.CurrentUserID(ctx)
	if !ok {
		return 0, apperr.ErrUnauthorized
	}
	allowed, err := s.repo.IsActiveTeacher(ctx, userID)
	if err != nil {
		return 0, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	if !allowed {
		return 0, apperr.ErrForbidden
	}
	return userID, nil
}

// buildCreateStudentInputs 归一化批量建学生输入并生成密码哈希。
// 这里不保留明文密码，所有记录在进入仓储层前都必须转换为哈希值，降低日志和链路泄露风险。
func (s *TeacherService) buildCreateStudentInputs(items []teacherdto.CreateStudentItem, classID int64, teacherID int64) ([]teacherentity.CreateStudentInput, error) {
	normalizedItems, err := normalizeCreateStudentItems(items)
	if err != nil {
		return nil, err
	}

	inputs := make([]teacherentity.CreateStudentInput, 0, len(normalizedItems))
	for _, item := range normalizedItems {
		hash, hashErr := s.passwordService.Hash(item.Password)
		if hashErr != nil {
			return nil, apperr.ErrInternal.WithCause(hashErr)
		}
		inputs = append(inputs, teacherentity.CreateStudentInput{
			Name:         item.Name,
			Email:        item.Email,
			PasswordHash: hash,
			StudentNo:    item.StudentNo,
			Target:       item.Target,
			ClassID:      classID,
			CreatedBy:    teacherID,
		})
	}
	return inputs, nil
}

// normalizeCreateStudentItems 规范化批量创建学生请求。
// 去重规则大小写不敏感地比较邮箱，并对学号按裁剪后的字符串比较，确保一次请求内不会因为脏数据导致事务回滚。
func normalizeCreateStudentItems(items []teacherdto.CreateStudentItem) ([]teacherdto.CreateStudentItem, error) {
	normalized := make([]teacherdto.CreateStudentItem, 0, len(items))
	emailSeen := make(map[string]struct{}, len(items))
	studentNoSeen := make(map[string]struct{}, len(items))
	for _, item := range items {
		normalizedItem := teacherdto.CreateStudentItem{
			Name:      strings.TrimSpace(item.Name),
			StudentNo: strings.TrimSpace(item.StudentNo),
			Email:     strings.ToLower(strings.TrimSpace(item.Email)),
			Password:  item.Password,
			Target:    strings.TrimSpace(item.Target),
		}

		if _, exists := emailSeen[normalizedItem.Email]; exists {
			return nil, apperr.ErrConflict.WithCause(errors.New("批量请求内邮箱重复"))
		}
		emailSeen[normalizedItem.Email] = struct{}{}

		if _, exists := studentNoSeen[normalizedItem.StudentNo]; exists {
			return nil, apperr.ErrConflict.WithCause(errors.New("批量请求内学号重复"))
		}
		studentNoSeen[normalizedItem.StudentNo] = struct{}{}
		normalized = append(normalized, normalizedItem)
	}
	return normalized, nil
}

func toClassroomView(item teacherentity.Classroom) teacherdto.ClassroomView {
	return teacherdto.ClassroomView{
		ID:           item.ID,
		Name:         item.Name,
		Grade:        item.Grade,
		StudentCount: item.StudentCount,
		Status:       item.Status,
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
	}
}

func toStudentView(item teacherentity.Student) teacherdto.StudentView {
	return teacherdto.StudentView{
		ID:        item.ID,
		Name:      item.Name,
		StudentNo: item.StudentNo,
		Email:     item.Email,
		Target:    item.Target,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
	}
}

func toStudentDetailView(item teacherentity.Student) teacherdto.StudentDetailView {
	return teacherdto.StudentDetailView{
		ID:        item.ID,
		Name:      item.Name,
		Email:     item.Email,
		Role:      item.Role,
		Title:     item.Title,
		StudentNo: item.StudentNo,
		ClassID:   item.ClassID,
		ClassName: item.ClassName,
		Target:    item.Target,
		Status:    item.Status,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
	}
}
