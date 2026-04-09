package service

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	authdto "github.com/han139242/course-recommend-agents/backend/internal/module/auth/dto"
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	graphwriterrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/repository"
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

type stubLearningRepo struct {
	submitResult     *learningentity.SubmitAnswerResult
	submitErr        error
	activeStudent    bool
	activeStudentErr error
	activeTeacher    bool
	activeTeacherErr error
	listStudentItems []learningentity.Question
	listStudentTotal int64
	listStudentErr   error
}

func (s *stubLearningRepo) BatchCreateQuestions(_ context.Context, _ []learningentity.ImportQuestionInput) (int, error) {
	return 0, nil
}

func (s *stubLearningRepo) ListQuestions(_ context.Context, _ learningentity.QuestionFilter) ([]learningentity.Question, int64, error) {
	return nil, 0, nil
}

func (s *stubLearningRepo) GetQuestionByQuestionID(_ context.Context, _ string) (*learningentity.Question, error) {
	return nil, nil
}

func (s *stubLearningRepo) ListStudentQuestions(_ context.Context, _ learningentity.StudentQuestionFilter) ([]learningentity.Question, int64, error) {
	return s.listStudentItems, s.listStudentTotal, s.listStudentErr
}

func (s *stubLearningRepo) SubmitAnswer(_ context.Context, _ learningentity.SubmitAnswerInput) (*learningentity.SubmitAnswerResult, error) {
	if s.submitErr != nil {
		return nil, s.submitErr
	}
	return s.submitResult, nil
}

func (s *stubLearningRepo) IsActiveStudent(_ context.Context, _ int64) (bool, error) {
	return s.activeStudent, s.activeStudentErr
}

func (s *stubLearningRepo) IsActiveTeacher(_ context.Context, _ int64) (bool, error) {
	return s.activeTeacher, s.activeTeacherErr
}

type stubStateService struct {
	snapshotState    learningentity.MasteryState
	snapshotDegraded bool
	snapshotErr      error
	progressResponse *learningdto.LearningProgressResponse
	progressDegraded bool
	progressErr      error
}

func (s *stubStateService) SnapshotMastery(_ context.Context, _ int64, _ string, _ float64, _ time.Time, _ bool) (learningentity.MasteryState, bool, error) {
	return s.snapshotState, s.snapshotDegraded, s.snapshotErr
}

func (s *stubStateService) GetLearningProgress(_ context.Context, _ int64) (*learningdto.LearningProgressResponse, bool, error) {
	return s.progressResponse, s.progressDegraded, s.progressErr
}

type stubAuthService struct {
	user *authdto.AuthUserView
	err  error
}

func (s *stubAuthService) GetCurrentUser(_ context.Context, _ int64) (*authdto.AuthUserView, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

func TestSubmitAnswerDoesNotPolluteDegradedWithGraphWriterFailure(t *testing.T) {
	repo := &stubLearningRepo{
		submitResult:  &learningentity.SubmitAnswerResult{Accepted: true, QuestionID: "q_001", KnowledgeID: "kp_001", Correct: true},
		activeStudent: true,
	}
	graphWriter := graphwriterservice.New(applogger.New("debug"), graphwriterrepo.NewNeo4jWriterRepository(nil), nil, config.GraphWriterConfig{Enabled: false})
	service := &Service{
		repo:         repo,
		redis:        redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}),
		graphWriter:  graphWriter,
		stateService: &stubStateService{snapshotState: learningentity.MasteryState{KnowledgeID: "kp_001", MasteryScore: 100, AnsweredCount: 1, CorrectCount: 1}},
		authService:  &stubAuthService{user: &authdto.AuthUserView{Name: "学生A", Target: "算法工程师"}},
	}
	resp, err := service.SubmitAnswer(withUserID(2), learningdto.SubmitAnswerRequest{QuestionID: "q_001", KnowledgeID: "kp_001", SelectedOption: "A", AnsweredAt: time.Now().UTC().Format(time.RFC3339)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Degraded {
		t.Fatal("expected degraded to reflect only main answer chain")
	}
}

func TestEnqueueGraphSyncReturnsFalseWhenGraphWriterDisabled(t *testing.T) {
	graphWriter := graphwriterservice.New(applogger.New("debug"), graphwriterrepo.NewNeo4jWriterRepository(nil), nil, config.GraphWriterConfig{Enabled: false})
	service := &Service{graphWriter: graphWriter, authService: &stubAuthService{user: &authdto.AuthUserView{Name: "学生A", Target: "算法工程师"}}}
	accepted := service.enqueueGraphSync(context.Background(), 2, time.Now().UTC(), learningentity.MasteryState{KnowledgeID: "kp_001"})
	if accepted {
		t.Fatal("expected graph sync enqueue to be rejected when graph writer is disabled")
	}
}

func TestEnqueueGraphSyncRequiresUserIdentity(t *testing.T) {
	graphWriter := graphwriterservice.New(applogger.New("debug"), graphwriterrepo.NewNeo4jWriterRepository(nil), nil, config.GraphWriterConfig{Enabled: true})
	service := &Service{graphWriter: graphWriter}
	accepted := service.enqueueGraphSync(context.Background(), 2, time.Now().UTC(), learningentity.MasteryState{KnowledgeID: "kp_001", MasteryScore: 80})
	if accepted {
		t.Fatal("expected enqueue to fail without user identity")
	}
}

func TestSubmitAnswerMapsQuestionNotFound(t *testing.T) {
	repo := &stubLearningRepo{submitErr: learningrepo.ErrQuestionNotFound, activeStudent: true}
	service := &Service{repo: repo, stateService: &stubStateService{}}
	_, err := service.SubmitAnswer(withUserID(2), learningdto.SubmitAnswerRequest{QuestionID: "q_404", KnowledgeID: "kp_001", SelectedOption: "A", AnsweredAt: time.Now().UTC().Format(time.RFC3339)})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListStudentQuestionsDoesNotLeakCorrectAnswer(t *testing.T) {
	repo := &stubLearningRepo{
		activeStudent: true,
		listStudentItems: []learningentity.Question{{
			QuestionID:    "q_001",
			Type:          learningentity.QuestionTypeSingleChoice,
			KnowledgeID:   "kp_001",
			Stem:          "题目",
			Options:       []learningentity.QuestionOption{{Key: "A", Text: "选项A"}, {Key: "B", Text: "选项B"}},
			CorrectOption: "A",
			Difficulty:    0.4,
		}},
		listStudentTotal: 1,
	}
	service := &Service{repo: repo, stateService: &stubStateService{}}
	resp, err := service.ListStudentQuestions(withUserID(2), learningdto.StudentQuestionsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("unexpected list length: %d", len(resp.List))
	}
	if len(resp.List[0].Options) != 2 {
		t.Fatalf("unexpected options: %+v", resp.List[0])
	}
}

func TestRequireStudentReturnsDependencyErrorWhenRepositoryFails(t *testing.T) {
	repo := &stubLearningRepo{activeStudentErr: errors.New("db down")}
	service := &Service{repo: repo, stateService: &stubStateService{}}
	_, err := service.requireStudent(withUserID(2))
	if err == nil {
		t.Fatal("expected dependency unavailable error")
	}
}

func TestNormalizeImportQuestionItemsRejectsDuplicateQuestionID(t *testing.T) {
	_, err := normalizeImportQuestionItems([]learningdto.ImportQuestionItem{{QuestionID: "q_001", Type: learningentity.QuestionTypeSingleChoice, KnowledgeID: "kp_001", Stem: "题目1", Options: []learningdto.QuestionOptionDTO{{Key: "A", Text: "1"}, {Key: "B", Text: "2"}}, CorrectOption: "A", Difficulty: 0.5}, {QuestionID: "q_001", Type: learningentity.QuestionTypeSingleChoice, KnowledgeID: "kp_002", Stem: "题目2", Options: []learningdto.QuestionOptionDTO{{Key: "A", Text: "1"}, {Key: "B", Text: "2"}}, CorrectOption: "A", Difficulty: 0.6}}, 1)
	if err == nil {
		t.Fatal("expected duplicate questionId conflict")
	}
}

func withUserID(userID int64) context.Context {
	return ctxmeta.WithUserID(context.Background(), strconv.FormatInt(userID, 10))
}
