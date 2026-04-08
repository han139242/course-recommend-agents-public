package service

import (
	"context"
	"errors"
	"strings"
	"time"

	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
	"github.com/redis/go-redis/v9"
)

// Service 提供题库与答题业务能力。
type Service struct {
	repo  learningrepo.Repository
	redis *redis.Client
}

// New 创建学习模块服务。
func New(repo learningrepo.Repository, redisClient *redis.Client) *Service {
	return &Service{repo: repo, redis: redisClient}
}

// ImportQuestions 教师批量导题。
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
	response := &learningdto.SubmitAnswerResponse{Accepted: result.Accepted, QuestionID: result.QuestionID, KnowledgeID: result.KnowledgeID, Degraded: s.redis == nil || result.Degraded}
	return response, nil
}

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
