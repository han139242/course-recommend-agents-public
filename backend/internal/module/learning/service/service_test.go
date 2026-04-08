package service

import (
	"context"
	"errors"
	"testing"
	"time"

	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

type learningRepoStub struct {
	batchCreateQuestions func(ctx context.Context, items []learningentity.ImportQuestionInput) (int, error)
	listQuestions        func(ctx context.Context, filter learningentity.QuestionFilter) ([]learningentity.Question, int64, error)
	getQuestion          func(ctx context.Context, questionID string) (*learningentity.Question, error)
	listStudentQuestions func(ctx context.Context, filter learningentity.StudentQuestionFilter) ([]learningentity.Question, int64, error)
	submitAnswer         func(ctx context.Context, input learningentity.SubmitAnswerInput) (*learningentity.SubmitAnswerResult, error)
	isActiveStudent      func(ctx context.Context, userID int64) (bool, error)
	isActiveTeacher      func(ctx context.Context, userID int64) (bool, error)
}

func (s *learningRepoStub) BatchCreateQuestions(ctx context.Context, items []learningentity.ImportQuestionInput) (int, error) {
	return s.batchCreateQuestions(ctx, items)
}
func (s *learningRepoStub) ListQuestions(ctx context.Context, filter learningentity.QuestionFilter) ([]learningentity.Question, int64, error) {
	return s.listQuestions(ctx, filter)
}
func (s *learningRepoStub) GetQuestionByQuestionID(ctx context.Context, questionID string) (*learningentity.Question, error) {
	return s.getQuestion(ctx, questionID)
}
func (s *learningRepoStub) ListStudentQuestions(ctx context.Context, filter learningentity.StudentQuestionFilter) ([]learningentity.Question, int64, error) {
	return s.listStudentQuestions(ctx, filter)
}
func (s *learningRepoStub) SubmitAnswer(ctx context.Context, input learningentity.SubmitAnswerInput) (*learningentity.SubmitAnswerResult, error) {
	return s.submitAnswer(ctx, input)
}
func (s *learningRepoStub) IsActiveStudent(ctx context.Context, userID int64) (bool, error) {
	return s.isActiveStudent(ctx, userID)
}
func (s *learningRepoStub) IsActiveTeacher(ctx context.Context, userID int64) (bool, error) {
	return s.isActiveTeacher(ctx, userID)
}

func teacherContext() context.Context {
	ctx := context.Background()
	ctx = ctxmeta.WithUserID(ctx, "1")
	ctx = ctxmeta.WithRole(ctx, "teacher")
	return ctx
}

func studentContext() context.Context {
	ctx := context.Background()
	ctx = ctxmeta.WithUserID(ctx, "2")
	ctx = ctxmeta.WithRole(ctx, "student")
	return ctx
}

func TestImportQuestionsRejectsInvalidType(t *testing.T) {
	repo := &learningRepoStub{isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return true, nil }}
	service := New(repo, nil)
	_, err := service.ImportQuestions(teacherContext(), learningdto.ImportQuestionsRequest{Items: []learningdto.ImportQuestionItem{{Type: "multiple_choice", KnowledgeID: "kp_001", Stem: "题干", Options: []learningdto.QuestionOptionDTO{{Key: "A", Text: "1"}, {Key: "B", Text: "2"}}, CorrectOption: "A", Difficulty: 0.5}}})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 400 {
		t.Fatalf("expected bad request, got %v", err)
	}
}

func TestImportQuestionsRejectsDuplicateQuestionIDInRequest(t *testing.T) {
	repo := &learningRepoStub{isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return true, nil }}
	service := New(repo, nil)
	_, err := service.ImportQuestions(teacherContext(), learningdto.ImportQuestionsRequest{Items: []learningdto.ImportQuestionItem{{QuestionID: "q_1", Type: learningentity.QuestionTypeSingleChoice, KnowledgeID: "kp_001", Stem: "题干1", Options: []learningdto.QuestionOptionDTO{{Key: "A", Text: "1"}, {Key: "B", Text: "2"}}, CorrectOption: "A", Difficulty: 0.5}, {QuestionID: "q_1", Type: learningentity.QuestionTypeSingleChoice, KnowledgeID: "kp_002", Stem: "题干2", Options: []learningdto.QuestionOptionDTO{{Key: "A", Text: "1"}, {Key: "B", Text: "2"}}, CorrectOption: "A", Difficulty: 0.5}}})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 409 {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestImportQuestionsSuccess(t *testing.T) {
	repo := &learningRepoStub{isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return true, nil }, batchCreateQuestions: func(ctx context.Context, items []learningentity.ImportQuestionInput) (int, error) {
		if len(items) != 1 || items[0].CreatedBy != 1 {
			t.Fatalf("unexpected items: %+v", items)
		}
		if items[0].KnowledgeID != "kp_001" || items[0].Stem != "题干" {
			t.Fatalf("unexpected normalized payload: %+v", items[0])
		}
		return 1, nil
	}}
	service := New(repo, nil)
	resp, err := service.ImportQuestions(teacherContext(), learningdto.ImportQuestionsRequest{Items: []learningdto.ImportQuestionItem{{QuestionID: "q_1", Type: learningentity.QuestionTypeSingleChoice, KnowledgeID: "kp_001", Stem: "题干", Options: []learningdto.QuestionOptionDTO{{Key: "A", Text: "1"}, {Key: "B", Text: "2"}}, CorrectOption: "A", Difficulty: 0.5}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CreatedCount != 1 {
		t.Fatalf("unexpected created count: %d", resp.CreatedCount)
	}
}

func TestListStudentQuestionsRequiresActiveStudent(t *testing.T) {
	repo := &learningRepoStub{isActiveStudent: func(ctx context.Context, userID int64) (bool, error) { return false, nil }}
	service := New(repo, nil)
	_, err := service.ListStudentQuestions(studentContext(), learningdto.StudentQuestionsQuery{})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 403 {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestSubmitAnswerReturnsDegradedWhenRedisUnavailable(t *testing.T) {
	repo := &learningRepoStub{
		isActiveStudent: func(ctx context.Context, userID int64) (bool, error) { return true, nil },
		submitAnswer: func(ctx context.Context, input learningentity.SubmitAnswerInput) (*learningentity.SubmitAnswerResult, error) {
			if input.StudentID != 2 {
				t.Fatalf("unexpected student id: %d", input.StudentID)
			}
			return &learningentity.SubmitAnswerResult{Accepted: true, QuestionID: input.QuestionID, KnowledgeID: input.KnowledgeID, Degraded: false}, nil
		},
	}
	service := New(repo, nil)
	durationMs := 1000
	resp, err := service.SubmitAnswer(studentContext(), learningdto.SubmitAnswerRequest{QuestionID: "q_1", KnowledgeID: "kp_1", SelectedOption: "A", DurationMs: &durationMs, AnsweredAt: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Degraded {
		t.Fatal("expected degraded=true when redis is unavailable")
	}
}

func TestSubmitAnswerMapsQuestionNotFound(t *testing.T) {
	repo := &learningRepoStub{
		isActiveStudent: func(ctx context.Context, userID int64) (bool, error) { return true, nil },
		submitAnswer:    func(ctx context.Context, input learningentity.SubmitAnswerInput) (*learningentity.SubmitAnswerResult, error) { return nil, learningrepo.ErrQuestionNotFound },
	}
	service := New(repo, nil)
	_, err := service.SubmitAnswer(studentContext(), learningdto.SubmitAnswerRequest{QuestionID: "q_1", KnowledgeID: "kp_1", SelectedOption: "A", DurationMs: nil, AnsweredAt: time.Now().Format(time.RFC3339)})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 404 {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestRequireStudentDependencyError(t *testing.T) {
	repo := &learningRepoStub{isActiveStudent: func(ctx context.Context, userID int64) (bool, error) { return false, errors.New("db down") }}
	service := New(repo, nil)
	_, err := service.ListStudentQuestions(studentContext(), learningdto.StudentQuestionsQuery{})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 503 {
		t.Fatalf("expected dependency unavailable, got %v", err)
	}
}

func TestImportQuestionsRejectsBlankFieldsAfterTrim(t *testing.T) {
	repo := &learningRepoStub{isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return true, nil }}
	service := New(repo, nil)
	_, err := service.ImportQuestions(teacherContext(), learningdto.ImportQuestionsRequest{Items: []learningdto.ImportQuestionItem{{Type: learningentity.QuestionTypeSingleChoice, KnowledgeID: "   ", Stem: "   ", Options: []learningdto.QuestionOptionDTO{{Key: "A", Text: "1"}, {Key: "B", Text: "2"}}, CorrectOption: "A", Difficulty: 0.5}}})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 400 {
		t.Fatalf("expected bad request, got %v", err)
	}
}

func TestListStudentQuestionsDoesNotLeakCorrectAnswer(t *testing.T) {
	repo := &learningRepoStub{
		isActiveStudent: func(ctx context.Context, userID int64) (bool, error) { return true, nil },
		listStudentQuestions: func(ctx context.Context, filter learningentity.StudentQuestionFilter) ([]learningentity.Question, int64, error) {
			return []learningentity.Question{{QuestionID: "q_1", Type: learningentity.QuestionTypeSingleChoice, KnowledgeID: "kp_1", Stem: "题干", CorrectOption: "A", Difficulty: 0.5, Explanation: "解析", Options: []learningentity.QuestionOption{{Key: "A", Text: "1"}, {Key: "B", Text: "2"}}}}, 1, nil
		},
	}
	service := New(repo, nil)
	resp, err := service.ListStudentQuestions(studentContext(), learningdto.StudentQuestionsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("unexpected list length: %d", len(resp.List))
	}
	if resp.List[0].QuestionID != "q_1" || resp.List[0].Difficulty != 0.5 {
		t.Fatalf("unexpected question payload: %+v", resp.List[0])
	}
}

func TestTeacherListQuestionsRequiresRealtimeTeacherCheck(t *testing.T) {
	repo := &learningRepoStub{isActiveTeacher: func(ctx context.Context, userID int64) (bool, error) { return false, nil }}
	service := New(repo, nil)
	_, err := service.ListQuestions(teacherContext(), learningdto.ListQuestionsQuery{})
	if err == nil || apperr.AsAppError(err).HTTPStatus != 403 {
		t.Fatalf("expected forbidden, got %v", err)
	}
}
