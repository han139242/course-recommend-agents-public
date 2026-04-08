package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
	learningmodel "github.com/han139242/course-recommend-agents/backend/internal/module/learning/model"
	usermodel "github.com/han139242/course-recommend-agents/backend/internal/module/user/model"
	"gorm.io/gorm"
)

const (
	userRoleStudent = "student"
)

// MySQLRepository 提供学习模块的 MySQL 仓储实现。
type MySQLRepository struct {
	db *gorm.DB
}

// NewMySQLRepository 创建学习模块 MySQL 仓储。
func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

// BatchCreateQuestions 批量创建题目。
func (r *MySQLRepository) BatchCreateQuestions(ctx context.Context, items []learningentity.ImportQuestionInput) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	createdCount := 0
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for index, item := range items {
			questionCode := item.QuestionID
			if questionCode == "" {
				questionCode = fmt.Sprintf("q_auto_%d_%d", time.Now().UnixNano(), index+1)
			}
			explanation := nullableString(strings.TrimSpace(item.Explanation))
			question := learningmodel.Question{
				QuestionCode:  questionCode,
				Type:          item.Type,
				KnowledgeID:   item.KnowledgeID,
				Stem:          item.Stem,
				CorrectOption: item.CorrectOption,
				Difficulty:    item.Difficulty,
				Explanation:   explanation,
				Status:        learningentity.QuestionStatusActive,
				CreatedBy:     item.CreatedBy,
			}
			if err := tx.Create(&question).Error; err != nil {
				return translateWriteError(err)
			}
			options := make([]learningmodel.QuestionOption, 0, len(item.Options))
			for order, option := range item.Options {
				options = append(options, learningmodel.QuestionOption{
					QuestionID: question.ID,
					OptionKey:  option.Key,
					OptionText: option.Text,
					SortOrder:  order + 1,
				})
			}
			if err := tx.Create(&options).Error; err != nil {
				return err
			}
			createdCount++
		}
		return nil
	}); err != nil {
		return 0, err
	}
	return createdCount, nil
}

// ListQuestions 查询教师端题目列表。
func (r *MySQLRepository) ListQuestions(ctx context.Context, filter learningentity.QuestionFilter) ([]learningentity.Question, int64, error) {
	baseQuery := r.questionQuery(ctx)
	if filter.KnowledgeID != "" {
		baseQuery = baseQuery.Where("questions.knowledge_id = ?", filter.KnowledgeID)
	}
	if filter.Type != "" {
		baseQuery = baseQuery.Where("questions.type = ?", filter.Type)
	}
	if filter.Keyword != "" {
		baseQuery = baseQuery.Where("questions.stem LIKE ?", "%"+filter.Keyword+"%")
	}

	var total int64
	if err := baseQuery.Distinct("questions.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var questions []learningmodel.Question
	if err := baseQuery.
		Order("questions.created_at DESC, questions.id DESC").
		Offset(filter.Pagination.Offset()).
		Limit(filter.Pagination.PageSize).
		Find(&questions).Error; err != nil {
		return nil, 0, err
	}
	items, err := r.attachOptions(ctx, questions)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetQuestionByQuestionID 查询单题详情。
func (r *MySQLRepository) GetQuestionByQuestionID(ctx context.Context, questionID string) (*learningentity.Question, error) {
	var question learningmodel.Question
	if err := r.db.WithContext(ctx).
		Where("question_code = ? AND status = ?", questionID, learningentity.QuestionStatusActive).
		Take(&question).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}
	items, err := r.attachOptions(ctx, []learningmodel.Question{question})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrQuestionNotFound
	}
	return &items[0], nil
}

// ListStudentQuestions 查询学生可见题目列表。
func (r *MySQLRepository) ListStudentQuestions(ctx context.Context, filter learningentity.StudentQuestionFilter) ([]learningentity.Question, int64, error) {
	baseQuery := r.questionQuery(ctx)
	if filter.KnowledgeID != "" {
		baseQuery = baseQuery.Where("questions.knowledge_id = ?", filter.KnowledgeID)
	}
	var total int64
	if err := baseQuery.Distinct("questions.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var questions []learningmodel.Question
	if err := baseQuery.
		Order("questions.created_at DESC, questions.id DESC").
		Offset(filter.Pagination.Offset()).
		Limit(filter.Pagination.PageSize).
		Find(&questions).Error; err != nil {
		return nil, 0, err
	}
	items, err := r.attachOptions(ctx, questions)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// SubmitAnswer 提交学生答题结果。
func (r *MySQLRepository) SubmitAnswer(ctx context.Context, input learningentity.SubmitAnswerInput) (*learningentity.SubmitAnswerResult, error) {
	question, err := r.GetQuestionByQuestionID(ctx, input.QuestionID)
	if err != nil {
		return nil, err
	}
	if question.KnowledgeID != input.KnowledgeID {
		return nil, ErrInvalidQuestion
	}
	if !containsOption(question.Options, input.SelectedOption) {
		return nil, ErrInvalidQuestion
	}
	isCorrect := question.CorrectOption == input.SelectedOption
	result := &learningentity.SubmitAnswerResult{Accepted: true, QuestionID: input.QuestionID, KnowledgeID: input.KnowledgeID, Degraded: false}
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		answer := learningmodel.StudentAnswer{
			StudentID:      input.StudentID,
			QuestionCode:   input.QuestionID,
			KnowledgeID:    input.KnowledgeID,
			SelectedOption: input.SelectedOption,
			IsCorrect:      isCorrect,
			DurationMs:     input.DurationMs,
			AnsweredAt:     input.AnsweredAt,
		}
		if err := tx.Create(&answer).Error; err != nil {
			return err
		}

		progress, err := getOrCreateLearningProgress(ctx, tx, input.StudentID, input.KnowledgeID, input.AnsweredAt)
		if err != nil {
			return err
		}
		progress.AnsweredCount++
		if isCorrect {
			progress.CorrectRate = ((progress.CorrectRate * float64(progress.AnsweredCount-1)) + 1) / float64(progress.AnsweredCount)
		} else {
			progress.CorrectRate = (progress.CorrectRate * float64(progress.AnsweredCount-1)) / float64(progress.AnsweredCount)
		}
		progress.MasteryScore = progress.CorrectRate * 100
		progress.LastAnsweredAt = &input.AnsweredAt
		progress.UpdatedAt = time.Now()
		if progress.AnsweredCount == 0 {
			progress.Status = learningentity.LearningStatusNotStarted
		} else {
			progress.Status = learningProgressStatus(progress.MasteryScore)
		}
		if err := tx.Save(progress).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// IsActiveStudent 校验用户当前是否仍为启用中的学生。
func (r *MySQLRepository) IsActiveStudent(ctx context.Context, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&usermodel.User{}).
		Where("id = ? AND role = ? AND status = ? AND class_id IS NOT NULL", userID, userRoleStudent, learningentity.QuestionStatusActive).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsActiveTeacher 校验用户当前是否仍为启用中的教师。
func (r *MySQLRepository) IsActiveTeacher(ctx context.Context, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&usermodel.User{}).
		Where("id = ? AND role = ? AND status = ?", userID, "teacher", learningentity.QuestionStatusActive).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *MySQLRepository) questionQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Model(&learningmodel.Question{}).
		Where("questions.status = ?", learningentity.QuestionStatusActive)
}

func (r *MySQLRepository) attachOptions(ctx context.Context, questions []learningmodel.Question) ([]learningentity.Question, error) {
	if len(questions) == 0 {
		return []learningentity.Question{}, nil
	}
	ids := make([]int64, 0, len(questions))
	for _, question := range questions {
		ids = append(ids, question.ID)
	}
	var options []learningmodel.QuestionOption
	if err := r.db.WithContext(ctx).
		Where("question_id IN ?", ids).
		Order("sort_order ASC, id ASC").
		Find(&options).Error; err != nil {
		return nil, err
	}
	optionMap := make(map[int64][]learningentity.QuestionOption, len(ids))
	for _, option := range options {
		optionMap[option.QuestionID] = append(optionMap[option.QuestionID], learningentity.QuestionOption{Key: option.OptionKey, Text: option.OptionText})
	}
	items := make([]learningentity.Question, 0, len(questions))
	for _, question := range questions {
		items = append(items, learningentity.Question{
			ID:            question.ID,
			QuestionID:    question.QuestionCode,
			Type:          question.Type,
			KnowledgeID:   question.KnowledgeID,
			Stem:          question.Stem,
			Options:       optionMap[question.ID],
			CorrectOption: question.CorrectOption,
			Difficulty:    question.Difficulty,
			Explanation:   stringValue(question.Explanation),
			CreatedBy:     question.CreatedBy,
			Status:        question.Status,
			CreatedAt:     question.CreatedAt,
		})
	}
	return items, nil
}

func translateWriteError(err error) error {
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		message := strings.ToLower(mysqlErr.Message)
		if strings.Contains(message, "uk_questions_question_code") {
			return ErrDuplicateQuestionCode
		}
	}
	return err
}

func getOrCreateLearningProgress(_ context.Context, tx *gorm.DB, studentID int64, knowledgeID string, answeredAt time.Time) (*learningmodel.LearningProgress, error) {
	var progress learningmodel.LearningProgress
	if err := tx.Where("student_id = ? AND knowledge_id = ?", studentID, knowledgeID).Take(&progress).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		progress = learningmodel.LearningProgress{
			StudentID:      studentID,
			KnowledgeID:    knowledgeID,
			CanonicalName:  knowledgeID,
			MasteryScore:   0,
			Status:         learningentity.LearningStatusNotStarted,
			CorrectRate:    0,
			AnsweredCount:  0,
			LastAnsweredAt: &answeredAt,
		}
		if err := tx.Create(&progress).Error; err != nil {
			return nil, err
		}
	}
	return &progress, nil
}

func learningProgressStatus(masteryScore float64) string {
	if masteryScore >= 80 {
		return "solid"
	}
	if masteryScore > 0 {
		return "progressing"
	}
	return learningentity.LearningStatusNotStarted
}

func containsOption(options []learningentity.QuestionOption, selected string) bool {
	for _, option := range options {
		if option.Key == selected {
			return true
		}
	}
	return false
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
