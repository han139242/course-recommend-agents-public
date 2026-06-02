package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	studyplandto "github.com/han139242/course-recommend-agents/backend/internal/module/studyplan/dto"
	studyplanentity "github.com/han139242/course-recommend-agents/backend/internal/module/studyplan/entity"
	studyplanrepo "github.com/han139242/course-recommend-agents/backend/internal/module/studyplan/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

type Service struct {
	repo             repository
	recommendService *recommendservice.Service
	stateRepo        learningrepo.StateRepository
}

type repository interface {
	GetProfile(ctx context.Context, studentID int64) (*studyplanentity.GoalProfile, error)
	UpsertProfile(ctx context.Context, profile *studyplanentity.GoalProfile) (*studyplanentity.GoalProfile, error)
	GetCurrentPlan(ctx context.Context, studentID int64) (*studyplanentity.StudyPlan, error)
	UpsertPlan(ctx context.Context, plan *studyplanentity.StudyPlan) (*studyplanentity.StudyPlan, error)
}

func New(repo repository, recommendService *recommendservice.Service, stateRepo learningrepo.StateRepository) *Service {
	return &Service{repo: repo, recommendService: recommendService, stateRepo: stateRepo}
}

func (s *Service) GetProfile(ctx context.Context) (*studyplandto.ProfileResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.repo.GetProfile(ctx, studentID)
	if errors.Is(err, studyplanrepo.ErrNotFound) {
		return &studyplandto.ProfileResponse{Profile: nil, HasCompletedProfile: false}, nil
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	view := profileToView(profile)
	return &studyplandto.ProfileResponse{Profile: view, HasCompletedProfile: true}, nil
}

func (s *Service) UpsertProfile(ctx context.Context, request studyplandto.GoalProfileRequest) (*studyplandto.ProfileResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeProfileRequest(request)
	if err != nil {
		return nil, err
	}
	topicsJSON, err := json.Marshal(normalized.LearnedTopics)
	if err != nil {
		return nil, apperr.ErrBadRequest.WithCause(err)
	}
	profile, err := s.repo.UpsertProfile(ctx, &studyplanentity.GoalProfile{
		StudentID:          studentID,
		GradeLevel:         normalized.GradeLevel,
		Major:              normalized.Major,
		GoalType:           normalized.GoalType,
		GoalText:           normalized.GoalText,
		TargetDirection:    normalized.TargetDirection,
		LearnedTopicsJSON:  string(topicsJSON),
		FoundationLevel:    normalized.FoundationLevel,
		WeeklyHours:        normalized.WeeklyHours,
		LearningPreference: normalized.LearningPreference,
	})
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	view := profileToView(profile)
	return &studyplandto.ProfileResponse{Profile: view, HasCompletedProfile: true}, nil
}

func (s *Service) GetCurrentPlan(ctx context.Context) (*studyplandto.CurrentPlanResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	profile, profileErr := s.repo.GetProfile(ctx, studentID)
	if profileErr != nil && !errors.Is(profileErr, studyplanrepo.ErrNotFound) {
		return nil, apperr.ErrDependencyUnavailable.WithCause(profileErr)
	}
	plan, planErr := s.repo.GetCurrentPlan(ctx, studentID)
	if planErr != nil && !errors.Is(planErr, studyplanrepo.ErrNotFound) {
		return nil, apperr.ErrDependencyUnavailable.WithCause(planErr)
	}
	return &studyplandto.CurrentPlanResponse{
		Profile: profileToView(profile),
		Plan:    planToView(plan),
	}, nil
}

func (s *Service) GeneratePlan(ctx context.Context) (*studyplandto.CurrentPlanResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.repo.GetProfile(ctx, studentID)
	if errors.Is(err, studyplanrepo.ErrNotFound) {
		return nil, apperr.ErrBadRequest.WithCause(err)
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	target := buildRecommendTarget(profile)
	pathResponse := &recommenddto.PathResponse{}
	if s.recommendService != nil {
		if generated, genErr := s.recommendService.GetUnifiedPath(ctx, recommenddto.PathQuery{
			Target: target,
			Scene:  "goal_plan",
			Limit:  9,
		}); genErr == nil && generated != nil {
			pathResponse = generated
		}
	}
	path := pathResponse.Path
	if len(path) == 0 && s.recommendService != nil {
		if fallback, fallbackErr := s.recommendService.GetPath(ctx, recommenddto.PathQuery{Target: target, Scene: "goal_plan", Limit: 9}); fallbackErr == nil && fallback != nil {
			path = fallback.Path
		}
	}

	stages := buildStages(profile, path, s.loadWeakProgress(ctx, studentID))
	nextSteps := firstStageItems(stages, 3)
	summary := buildSummary(profile, nextSteps)
	stagesJSON, _ := json.Marshal(stages)
	nextStepsJSON, _ := json.Marshal(nextSteps)

	plan, err := s.repo.UpsertPlan(ctx, &studyplanentity.StudyPlan{
		StudentID:     studentID,
		ProfileID:     profile.ID,
		Summary:       summary,
		StagesJSON:    string(stagesJSON),
		NextStepsJSON: string(nextStepsJSON),
		GeneratedAt:   time.Now(),
	})
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	return &studyplandto.CurrentPlanResponse{Profile: profileToView(profile), Plan: planToView(plan)}, nil
}

func currentStudentID(ctx context.Context) (int64, error) {
	studentID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		return 0, apperr.ErrUnauthorized
	}
	return studentID, nil
}

func normalizeProfileRequest(request studyplandto.GoalProfileRequest) (studyplandto.GoalProfileRequest, error) {
	request.GradeLevel = normalizeEnum(request.GradeLevel)
	request.GoalType = normalizeEnum(request.GoalType)
	request.FoundationLevel = normalizeEnum(request.FoundationLevel)
	request.LearningPreference = normalizeEnum(request.LearningPreference)
	request.Major = strings.TrimSpace(request.Major)
	request.GoalText = strings.TrimSpace(request.GoalText)
	request.TargetDirection = strings.TrimSpace(request.TargetDirection)
	if request.WeeklyHours <= 0 {
		request.WeeklyHours = 6
	}
	if request.GoalText == "" || !allowedGradeLevels[request.GradeLevel] || !allowedGoalTypes[request.GoalType] || !allowedFoundationLevels[request.FoundationLevel] || !allowedPreferences[request.LearningPreference] {
		return request, apperr.ErrBadRequest
	}
	topics := make([]string, 0, len(request.LearnedTopics))
	seen := map[string]struct{}{}
	for _, item := range request.LearnedTopics {
		topic := strings.TrimSpace(item)
		if topic == "" {
			continue
		}
		if len([]rune(topic)) > 64 {
			topic = string([]rune(topic)[:64])
		}
		key := strings.ToLower(topic)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		topics = append(topics, topic)
	}
	request.LearnedTopics = topics
	return request, nil
}

func normalizeEnum(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

var allowedGradeLevels = map[string]bool{
	"freshman": true, "sophomore": true, "junior": true, "senior": true, "postgraduate": true, "other": true,
}

var allowedGoalTypes = map[string]bool{
	"course_learning": true, "career_preparation": true, "project_building": true,
}

var allowedFoundationLevels = map[string]bool{
	"beginner": true, "intermediate": true, "advanced": true,
}

var allowedPreferences = map[string]bool{
	"theory_first": true, "project_first": true, "exam_first": true, "balanced": true,
}

func buildRecommendTarget(profile *studyplanentity.GoalProfile) string {
	parts := []string{profile.TargetDirection, profile.GoalText, profile.Major}
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			return strings.TrimSpace(part)
		}
	}
	return "人工智能学习路径"
}

func (s *Service) loadWeakProgress(ctx context.Context, studentID int64) []recommenddto.PathItemView {
	if s.stateRepo == nil {
		return nil
	}
	rows, err := s.stateRepo.ListLearningProgress(ctx, studentID)
	if err != nil {
		return nil
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].MasteryScore == rows[j].MasteryScore {
			return rows[i].CanonicalName < rows[j].CanonicalName
		}
		return rows[i].MasteryScore < rows[j].MasteryScore
	})
	items := make([]recommenddto.PathItemView, 0, 3)
	for _, row := range rows {
		if len(items) >= 3 {
			break
		}
		if row.MasteryScore >= 60 || strings.TrimSpace(row.KnowledgeID) == "" {
			continue
		}
		items = append(items, recommenddto.PathItemView{
			Stage:         "foundation",
			Title:         firstNonEmpty(row.CanonicalName, row.KnowledgeID),
			Detail:        "当前掌握度偏低，适合作为目标路径前的基础补齐点。",
			Route:         fmt.Sprintf("/knowledge/%s", row.KnowledgeID),
			KnowledgeID:   row.KnowledgeID,
			ReasonCode:    "weakness",
			ReasonLabel:   "基础补齐",
			MasteryScore:  row.MasteryScore,
			CorrectRate:   row.CorrectRate,
			AnsweredCount: row.AnsweredCount,
			Score:         0.88,
		})
	}
	return items
}

func buildStages(profile *studyplanentity.GoalProfile, path []recommenddto.PathItemView, weakItems []recommenddto.PathItemView) []studyplandto.StudyPlanStageView {
	foundationItems := append([]recommenddto.PathItemView{}, weakItems...)
	coreItems := []recommenddto.PathItemView{}
	applicationItems := []recommenddto.PathItemView{}

	for _, item := range path {
		switch item.ReasonCode {
		case "gap_bridge", "weakness", "forgetting_review":
			foundationItems = appendUniquePathItem(foundationItems, item)
		case "cross_course_transfer":
			applicationItems = appendUniquePathItem(applicationItems, item)
		default:
			coreItems = appendUniquePathItem(coreItems, item)
		}
	}
	if len(coreItems) == 0 {
		coreItems = takePathItems(path, 0, 3)
	}
	if len(applicationItems) == 0 {
		applicationItems = takePathItems(path, 3, 6)
	}
	if len(foundationItems) == 0 {
		foundationItems = takePathItems(path, 0, 2)
	}

	return []studyplandto.StudyPlanStageView{
		{
			Key:     "foundation",
			Title:   "基础补齐",
			Summary: "先把目标路径前最容易卡住的数学、编程或课程基础补稳。",
			Items:   limitPathItems(foundationItems, 3),
		},
		{
			Key:     "core",
			Title:   "核心学习",
			Summary: fmt.Sprintf("围绕“%s”建立主线理解，优先学习与目标最近的知识点。", firstNonEmpty(profile.TargetDirection, profile.GoalText)),
			Items:   limitPathItems(coreItems, 3),
		},
		{
			Key:     "application",
			Title:   "项目应用",
			Summary: "把已学知识迁移到项目、就业或课程成果中，形成可展示的学习闭环。",
			Items:   limitPathItems(applicationItems, 3),
		},
	}
}

func firstStageItems(stages []studyplandto.StudyPlanStageView, limit int) []recommenddto.PathItemView {
	result := []recommenddto.PathItemView{}
	for _, stage := range stages {
		for _, item := range stage.Items {
			result = appendUniquePathItem(result, item)
			if len(result) >= limit {
				return result
			}
		}
	}
	return result
}

func buildSummary(profile *studyplanentity.GoalProfile, nextSteps []recommenddto.PathItemView) string {
	target := firstNonEmpty(profile.TargetDirection, profile.GoalText, "当前学习目标")
	if len(nextSteps) == 0 {
		return fmt.Sprintf("已根据“%s”生成目标画像，下一步建议先从基础梳理和课程图谱探索开始。", target)
	}
	return fmt.Sprintf("已根据“%s”和每周约 %d 小时学习时间生成三阶段路线，当前优先从“%s”开始。", target, profile.WeeklyHours, nextSteps[0].Title)
}

func profileToView(profile *studyplanentity.GoalProfile) *studyplandto.GoalProfileView {
	if profile == nil {
		return nil
	}
	return &studyplandto.GoalProfileView{
		ID:                 profile.ID,
		GradeLevel:         profile.GradeLevel,
		Major:              profile.Major,
		GoalType:           profile.GoalType,
		GoalText:           profile.GoalText,
		TargetDirection:    profile.TargetDirection,
		LearnedTopics:      decodeStringList(profile.LearnedTopicsJSON),
		FoundationLevel:    profile.FoundationLevel,
		WeeklyHours:        profile.WeeklyHours,
		LearningPreference: profile.LearningPreference,
		UpdatedAt:          profile.UpdatedAt.Format(time.RFC3339),
	}
}

func planToView(plan *studyplanentity.StudyPlan) *studyplandto.StudyPlanView {
	if plan == nil {
		return nil
	}
	stages := []studyplandto.StudyPlanStageView{}
	nextSteps := []recommenddto.PathItemView{}
	_ = json.Unmarshal([]byte(plan.StagesJSON), &stages)
	_ = json.Unmarshal([]byte(plan.NextStepsJSON), &nextSteps)
	return &studyplandto.StudyPlanView{
		ID:          plan.ID,
		Summary:     plan.Summary,
		Stages:      stages,
		NextSteps:   nextSteps,
		GeneratedAt: plan.GeneratedAt.Format(time.RFC3339),
	}
}

func decodeStringList(raw string) []string {
	items := []string{}
	if strings.TrimSpace(raw) == "" {
		return items
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}
	return items
}

func appendUniquePathItem(items []recommenddto.PathItemView, item recommenddto.PathItemView) []recommenddto.PathItemView {
	key := strings.TrimSpace(item.KnowledgeID)
	if key == "" {
		key = strings.TrimSpace(item.Route)
	}
	if key == "" {
		return items
	}
	for _, existing := range items {
		existingKey := strings.TrimSpace(existing.KnowledgeID)
		if existingKey == "" {
			existingKey = strings.TrimSpace(existing.Route)
		}
		if existingKey == key {
			return items
		}
	}
	return append(items, item)
}

func takePathItems(items []recommenddto.PathItemView, start int, end int) []recommenddto.PathItemView {
	if start >= len(items) {
		return []recommenddto.PathItemView{}
	}
	if end > len(items) {
		end = len(items)
	}
	return append([]recommenddto.PathItemView{}, items[start:end]...)
}

func limitPathItems(items []recommenddto.PathItemView, limit int) []recommenddto.PathItemView {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
