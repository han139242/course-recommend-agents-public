package service

import (
	"context"
	"errors"
	"strings"
	"time"

	authdto "github.com/han139242/course-recommend-agents/backend/internal/module/auth/dto"
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	graphwriterentity "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/entity"
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
	"github.com/redis/go-redis/v9"
)

// Service 提供题库与答题业务能力。
type authReader interface {
	GetCurrentUser(ctx context.Context, userID int64) (*authdto.AuthUserView, error)
}

type stateReader interface {
	SnapshotMastery(ctx context.Context, studentID int64, knowledgeID string, delta float64, answeredAt time.Time, correct bool) (learningentity.MasteryState, bool, error)
	GetLearningProgress(ctx context.Context, studentID int64) (*learningdto.LearningProgressResponse, bool, error)
}

type Service struct {
	repo         learningrepo.Repository
	redis        *redis.Client
	authService  authReader
	graphWriter  *graphwriterservice.Service
	stateService stateReader
}

// New 创建学习模块服务。
func New(repo learningrepo.Repository, redisClient *redis.Client, authService *authservice.AuthService, graphWriter *graphwriterservice.Service) *Service {
	redisRepo := learningrepo.NewRedisRepository(redisClient)
	var stateRepo learningrepo.StateRepository
	if mysqlRepo, ok := repo.(*learningrepo.MySQLRepository); ok {
		stateRepo = learningrepo.NewStateMySQLRepository(mysqlRepo.DB())
	}
	return &Service{repo: repo, redis: redisClient, authService: authService, graphWriter: graphWriter, stateService: NewStateService(redisRepo, stateRepo)}
}

// ImportQuestions 教师批量导题。
// 请求会先完成题型、知识点、选项与正确答案的一致性校验，再交给仓储层按事务批量落库，避免题干与选项半写入。
func (s *Service) ImportQuestions(ctx context.Context, request learningdto.ImportQuestionsRequest) (*learningdto.ImportQuestionsResponse, error) {
	teacherID, err := s.requireTeacher(ctx)
	if err != nil {
		return nil, err
	}
	items, err := normalizeImportQuestionItems(request.Items, teacherID)
	if err != nil {
		return nil, err
	}
	createdCount, err := s.repo.BatchCreateQuestions(ctx, items)
	if err != nil {
		if errors.Is(err, learningrepo.ErrDuplicateQuestionCode) {
			return nil, apperr.ErrConflict.WithCause(err)
		}
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	return &learningdto.ImportQuestionsResponse{CreatedCount: createdCount}, nil
}

// ListQuestions 教师查询题目列表。
// 关键字、题型、知识点筛选都会先裁剪空白字符，避免因为前端拼接空格导致查询条件失真。
func (s *Service) ListQuestions(ctx context.Context, query learningdto.ListQuestionsQuery) (*learningdto.TeacherListQuestionsResponse, error) {
	if _, err := s.requireTeacher(ctx); err != nil {
		return nil, err
	}
	normalized := query.PaginationQuery.Normalize()
	items, total, err := s.repo.ListQuestions(ctx, learningentity.QuestionFilter{KnowledgeID: strings.TrimSpace(query.KnowledgeID), Type: strings.TrimSpace(query.Type), Keyword: strings.TrimSpace(query.Keyword), Pagination: learningentity.Pagination{Page: normalized.Page, PageSize: normalized.PageSize}})
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	return &learningdto.TeacherListQuestionsResponse{List: toTeacherQuestionViews(items), Total: total, Page: normalized.Page, PageSize: normalized.PageSize}, nil
}

// GetQuestionDetail 教师查询单题详情。
func (s *Service) GetQuestionDetail(ctx context.Context, questionID string) (*learningdto.QuestionDetailView, error) {
	if _, err := s.requireTeacher(ctx); err != nil {
		return nil, err
	}
	question, err := s.repo.GetQuestionByQuestionID(ctx, strings.TrimSpace(questionID))
	if err != nil {
		if errors.Is(err, learningrepo.ErrQuestionNotFound) {
			return nil, apperr.ErrNotFound.WithCause(err)
		}
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	view := toQuestionDetailView(*question)
	return &view, nil
}

// ListStudentQuestions 学生查询练习题目。
// 学生视图会主动剔除正确答案与解析，仅返回作答所需字段，防止题库数据在练习接口泄露。
func (s *Service) ListStudentQuestions(ctx context.Context, query learningdto.StudentQuestionsQuery) (*learningdto.StudentListQuestionsResponse, error) {
	studentID, err := s.requireStudent(ctx)
	if err != nil {
		return nil, err
	}
	normalized := query.PaginationQuery.Normalize()
	items, total, err := s.repo.ListStudentQuestions(ctx, learningentity.StudentQuestionFilter{StudentID: studentID, KnowledgeID: strings.TrimSpace(query.KnowledgeID), Pagination: learningentity.Pagination{Page: normalized.Page, PageSize: normalized.PageSize}})
	if err != nil {
		switch {
		case errors.Is(err, learningrepo.ErrForbiddenQuestion):
			return nil, apperr.ErrForbidden.WithCause(err)
		default:
			return nil, apperr.ErrDependencyUnavailable.WithCause(err)
		}
	}
	return &learningdto.StudentListQuestionsResponse{List: toStudentQuestionViews(items), Total: total, Page: normalized.Page, PageSize: normalized.PageSize}, nil
}

// SubmitAnswer 学生提交答题结果。
// 服务层负责把时间字符串解析为统一时间格式，并把题目不存在、题目参数不一致、越权访问映射成明确的业务错误。
func (s *Service) SubmitAnswer(ctx context.Context, request learningdto.SubmitAnswerRequest) (*learningdto.SubmitAnswerResponse, error) {
	studentID, err := s.requireStudent(ctx)
	if err != nil {
		return nil, err
	}
	answeredAt, err := time.Parse(time.RFC3339, strings.TrimSpace(request.AnsweredAt))
	if err != nil {
		return nil, apperr.ErrBadRequest.WithCause(err)
	}
	result, err := s.repo.SubmitAnswer(ctx, learningentity.SubmitAnswerInput{StudentID: studentID, QuestionID: strings.TrimSpace(request.QuestionID), KnowledgeID: strings.TrimSpace(request.KnowledgeID), SelectedOption: strings.TrimSpace(request.SelectedOption), DurationMs: request.DurationMs, AnsweredAt: answeredAt})
	if err != nil {
		switch {
		case errors.Is(err, learningrepo.ErrQuestionNotFound):
			return nil, apperr.ErrNotFound.WithCause(err)
		case errors.Is(err, learningrepo.ErrInvalidQuestion):
			return nil, apperr.ErrBadRequest.WithCause(err)
		case errors.Is(err, learningrepo.ErrForbiddenQuestion):
			return nil, apperr.ErrForbidden.WithCause(err)
		default:
			return nil, apperr.ErrDependencyUnavailable.WithCause(err)
		}
	}
	state, degraded, stateErr := s.stateService.SnapshotMastery(ctx, studentID, strings.TrimSpace(request.KnowledgeID), masteryDelta(result), answeredAt, result.Correct)
	if stateErr != nil {
		return nil, stateErr
	}
	s.enqueueGraphSync(ctx, studentID, answeredAt, state)
	response := &learningdto.SubmitAnswerResponse{Accepted: result.Accepted, QuestionID: result.QuestionID, KnowledgeID: result.KnowledgeID, Degraded: degraded || s.redis == nil || result.Degraded}
	return response, nil
}

// GetLearningProgress 查询学生当前学习进度，优先读取 Redis，失败时回源 MySQL。
func (s *Service) GetLearningProgress(ctx context.Context) (*learningdto.LearningProgressResponse, bool, error) {
	studentID, err := s.requireStudent(ctx)
	if err != nil {
		return nil, false, err
	}
	response, degraded, stateErr := s.stateService.GetLearningProgress(ctx, studentID)
	if stateErr != nil {
		return nil, degraded, stateErr
	}
	return response, degraded, nil
}

// requireStudent 校验当前请求是否来自启用中的学生。
// 这里以数据库实时状态为准，避免学生被移出班级或停用后仍凭旧 token 继续答题。
func (s *Service) requireStudent(ctx context.Context) (int64, error) {
	userID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		return 0, apperr.ErrUnauthorized
	}
	allowed, err := s.repo.IsActiveStudent(ctx, userID)
	if err != nil {
		return 0, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	if !allowed {
		return 0, apperr.ErrForbidden
	}
	return userID, nil
}

// requireTeacher 校验当前请求是否来自启用中的教师。
// 与教师模块保持一致，权限判断不直接信任上下文角色快照，而是回查数据库最新状态。
func (s *Service) requireTeacher(ctx context.Context) (int64, error) {
	userID, ok := ctxmeta.CurrentUserID(ctx)
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

// normalizeImportQuestionItems 规范化导题请求。
// 一次批量导入内 questionId 只要出现重复就直接拒绝，保证仓储层事务不会因为局部脏数据反复回滚。
func normalizeImportQuestionItems(items []learningdto.ImportQuestionItem, teacherID int64) ([]learningentity.ImportQuestionInput, error) {
	normalized := make([]learningentity.ImportQuestionInput, 0, len(items))
	questionIDSeen := make(map[string]struct{}, len(items))
	for _, item := range items {
		questionID := strings.TrimSpace(item.QuestionID)
		if questionID != "" {
			if _, exists := questionIDSeen[questionID]; exists {
				return nil, apperr.ErrConflict.WithCause(errors.New("批量请求内 questionId 重复"))
			}
			questionIDSeen[questionID] = struct{}{}
		}
		questionType := strings.TrimSpace(item.Type)
		if questionType != learningentity.QuestionTypeSingleChoice {
			return nil, apperr.ErrBadRequest.WithCause(errors.New("当前只支持 single_choice 题型"))
		}
		knowledgeID := strings.TrimSpace(item.KnowledgeID)
		if knowledgeID == "" {
			return nil, apperr.ErrBadRequest.WithCause(errors.New("knowledgeId 不能为空"))
		}
		stem := strings.TrimSpace(item.Stem)
		if stem == "" {
			return nil, apperr.ErrBadRequest.WithCause(errors.New("stem 不能为空"))
		}
		difficulty := item.Difficulty
		if difficulty < 0 || difficulty > 1 {
			return nil, apperr.ErrBadRequest.WithCause(errors.New("题目难度必须在 0 到 1 之间"))
		}
		options, err := normalizeQuestionOptions(item.Options)
		if err != nil {
			return nil, err
		}
		correctOption := strings.TrimSpace(item.CorrectOption)
		if correctOption == "" {
			return nil, apperr.ErrBadRequest.WithCause(errors.New("correctOption 不能为空"))
		}
		if !hasOptionKey(options, correctOption) {
			return nil, apperr.ErrBadRequest.WithCause(errors.New("正确选项必须存在于题目选项中"))
		}
		normalized = append(normalized, learningentity.ImportQuestionInput{QuestionID: questionID, Type: questionType, KnowledgeID: knowledgeID, Stem: stem, Options: options, CorrectOption: correctOption, Difficulty: difficulty, Explanation: strings.TrimSpace(item.Explanation), CreatedBy: teacherID})
	}
	return normalized, nil
}

// normalizeQuestionOptions 规范化题目选项。
// 选项键必须唯一且内容非空，防止前端重复 key 导致正确答案映射到多个候选项。
func normalizeQuestionOptions(items []learningdto.QuestionOptionDTO) ([]learningentity.QuestionOption, error) {
	optionSeen := make(map[string]struct{}, len(items))
	options := make([]learningentity.QuestionOption, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		text := strings.TrimSpace(item.Text)
		if key == "" {
			return nil, apperr.ErrBadRequest.WithCause(errors.New("选项 key 不能为空"))
		}
		if text == "" {
			return nil, apperr.ErrBadRequest.WithCause(errors.New("选项内容不能为空"))
		}
		if _, exists := optionSeen[key]; exists {
			return nil, apperr.ErrBadRequest.WithCause(errors.New("选项 key 不能重复"))
		}
		optionSeen[key] = struct{}{}
		options = append(options, learningentity.QuestionOption{Key: key, Text: text})
	}
	return options, nil
}

func hasOptionKey(options []learningentity.QuestionOption, key string) bool {
	for _, option := range options {
		if option.Key == key {
			return true
		}
	}
	return false
}

func (s *Service) enqueueGraphSync(ctx context.Context, studentID int64, answeredAt time.Time, state learningentity.MasteryState) bool {
	if s.graphWriter == nil || !s.graphWriter.Enabled() {
		return false
	}
	profile, ok := s.loadGraphProfile(ctx, studentID)
	if !ok {
		return false
	}
	if strings.TrimSpace(state.KnowledgeID) == "" {
		return false
	}
	event := graphwriterentity.ProfileUpdateEvent{
		StudentID: studentID,
		Identity: graphwriterentity.ProfileIdentity{
			Name:   profile.Name,
			Target: profile.Target,
		},
		UpdatedAt: answeredAt,
		TraceID:   ctxmeta.TraceID(ctx),
		Source:    "learning.submit_answer",
		Relations: []graphwriterentity.CompetencyRelation{{
			KnowledgeID:   state.KnowledgeID,
			MasteryScore:  state.MasteryScore,
			AnsweredCount: state.AnsweredCount,
			CorrectCount:  state.CorrectCount,
		}},
	}
	return s.graphWriter.Enqueue(ctx, event)
}

func (s *Service) loadGraphProfile(ctx context.Context, studentID int64) (graphwriterentity.ProfileIdentity, bool) {
	if s.authService == nil {
		return graphwriterentity.ProfileIdentity{}, false
	}
	userView, err := s.authService.GetCurrentUser(ctx, studentID)
	if err != nil {
		return graphwriterentity.ProfileIdentity{}, false
	}
	if strings.TrimSpace(userView.Name) == "" {
		return graphwriterentity.ProfileIdentity{}, false
	}
	return graphwriterentity.ProfileIdentity{Name: userView.Name, Target: userView.Target}, true
}

func toTeacherQuestionViews(items []learningentity.Question) []learningdto.TeacherQuestionView {
	result := make([]learningdto.TeacherQuestionView, 0, len(items))
	for _, item := range items {
		result = append(result, learningdto.TeacherQuestionView{QuestionID: item.QuestionID, Type: item.Type, KnowledgeID: item.KnowledgeID, Stem: item.Stem, Options: toQuestionOptionDTOs(item.Options), CorrectOption: item.CorrectOption, Difficulty: item.Difficulty, Explanation: item.Explanation})
	}
	return result
}

func toStudentQuestionViews(items []learningentity.Question) []learningdto.StudentQuestionView {
	result := make([]learningdto.StudentQuestionView, 0, len(items))
	for _, item := range items {
		result = append(result, learningdto.StudentQuestionView{QuestionID: item.QuestionID, Type: item.Type, KnowledgeID: item.KnowledgeID, Stem: item.Stem, Options: toQuestionOptionDTOs(item.Options), Difficulty: item.Difficulty})
	}
	return result
}

func toQuestionDetailView(item learningentity.Question) learningdto.QuestionDetailView {
	return learningdto.QuestionDetailView{QuestionID: item.QuestionID, Type: item.Type, KnowledgeID: item.KnowledgeID, Stem: item.Stem, Options: toQuestionOptionDTOs(item.Options), CorrectOption: item.CorrectOption, Difficulty: item.Difficulty, Explanation: item.Explanation, CreatedBy: item.CreatedBy, CreatedAt: item.CreatedAt.Format(time.RFC3339)}
}

func toQuestionOptionDTOs(items []learningentity.QuestionOption) []learningdto.QuestionOptionDTO {
	result := make([]learningdto.QuestionOptionDTO, 0, len(items))
	for _, item := range items {
		result = append(result, learningdto.QuestionOptionDTO{Key: item.Key, Text: item.Text})
	}
	return result
}

func masteryDelta(result *learningentity.SubmitAnswerResult) float64 {
	if result == nil {
		return 0
	}
	if result.Accepted {
		return 10
	}
	return 0
}
