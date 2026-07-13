package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/ai"
	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	profiledto "github.com/han139242/course-recommend-agents/backend/internal/module/profile/dto"
	profileservice "github.com/han139242/course-recommend-agents/backend/internal/module/profile/service"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	resourcegendto "github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen/dto"
	resourcegenentity "github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen/entity"
	resourcegenrepo "github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

type Service struct {
	repo             repository
	aiClient         *ai.Client
	profileService   *profileservice.Service
	recommendService *recommendservice.Service
	graphService     *graphservice.Service
}

type repository interface {
	CreateJob(ctx context.Context, job *resourcegenentity.GenerationJob) error
	UpdateJob(ctx context.Context, job *resourcegenentity.GenerationJob) error
	CreateStep(ctx context.Context, step *resourcegenentity.AgentStep) error
	UpdateStep(ctx context.Context, step *resourcegenentity.AgentStep) error
	CreateResources(ctx context.Context, resources []resourcegenentity.GeneratedResource) error
	CreateResource(ctx context.Context, resource *resourcegenentity.GeneratedResource) error
	UpdateResource(ctx context.Context, resource *resourcegenentity.GeneratedResource) error
	CreatePushItem(ctx context.Context, item *resourcegenentity.ResourcePushItem) error
	CreateLearningEvent(ctx context.Context, event *resourcegenentity.LearningEvent) error
	UpdatePushStatusForResource(ctx context.Context, studentID int64, resourceID int64, status string) error
	GetJob(ctx context.Context, studentID int64, jobID int64) (*resourcegenentity.GenerationJob, []resourcegenentity.AgentStep, []resourcegenentity.GeneratedResource, error)
	ListResources(ctx context.Context, studentID int64, knowledgeID string, resourceType string, keyword string, status string, sortOrder string, limit int) ([]resourcegenentity.GeneratedResource, error)
	ListPushStatuses(ctx context.Context, studentID int64, resourceIDs []int64) (map[int64]string, error)
	ListPushItems(ctx context.Context, studentID int64, limit int) ([]resourcegenentity.ResourcePushItem, []resourcegenentity.GeneratedResource, error)
	GetResource(ctx context.Context, studentID int64, resourceID int64) (*resourcegenentity.GeneratedResource, error)
	UpsertFeedback(ctx context.Context, feedback *resourcegenentity.ResourceFeedback) (*resourcegenentity.ResourceFeedback, error)
	ListFeedback(ctx context.Context, studentID int64, limit int) ([]resourcegenentity.ResourceFeedback, error)
	ListLearningEvents(ctx context.Context, studentID int64, limit int) ([]resourcegenentity.LearningEvent, error)
}

type generationContext struct {
	StudentID       int64
	KnowledgeID     string
	KnowledgeName   string
	CourseTitle     string
	Target          string
	Profile         *profiledto.LearningProfileView
	ProfileDims     map[string]string
	KnowledgeDetail *graphdto.KnowledgeDetailView
	Path            []recommenddto.PathItemView
	ResourceTypes   []string
	Difficulty      string
	Images          []string
	ImageDigest     string
}

func New(repo repository, aiClient *ai.Client, profileService *profileservice.Service, recommendService *recommendservice.Service, graphService *graphservice.Service) *Service {
	return &Service{repo: repo, aiClient: aiClient, profileService: profileService, recommendService: recommendService, graphService: graphService}
}

func (s *Service) Generate(ctx context.Context, request resourcegendto.GenerateResourcesRequest) (*resourcegendto.GenerateResourcesResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}

	prelim := buildPreliminaryContext(studentID, request)
	job := &resourcegenentity.GenerationJob{
		StudentID:     studentID,
		KnowledgeID:   prelim.KnowledgeID,
		KnowledgeName: prelim.KnowledgeName,
		CourseTitle:   prelim.CourseTitle,
		Target:        firstNonEmpty(prelim.Target, prelim.KnowledgeName),
		Status:        resourcegenentity.JobStatusRunning,
		Progress:      5,
		CurrentAgent:  "Orchestrator",
	}
	if err := s.repo.CreateJob(ctx, job); err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
		StudentID:     studentID,
		EventType:     "resource_generation_started",
		KnowledgeID:   prelim.KnowledgeID,
		KnowledgeName: prelim.KnowledgeName,
		ResourceType:  strings.Join(prelim.ResourceTypes, ","),
		PayloadJSON:   mustEventPayload(map[string]any{"target": job.Target, "courseTitle": job.CourseTitle}),
	})

	steps := s.buildSteps(prelim)
	for index := range steps {
		steps[index].JobID = job.ID
		steps[index].StudentID = studentID
		if err := s.repo.CreateStep(ctx, &steps[index]); err != nil {
			s.failJob(ctx, job, err.Error())
			return nil, apperr.ErrDependencyUnavailable.WithCause(err)
		}
	}

	// Snapshot the response view before launching the background pipeline so the
	// goroutine's concurrent step mutations cannot race with building the reply.
	jobView := jobToView(job, steps, nil)

	// Run the multi-agent pipeline in the background so the HTTP response returns
	// immediately and the client can poll job progress instead of blocking on a
	// long synchronous model call. The context is detached from the request
	// (which is cancelled once we return) but still carries the student identity
	// used by the profile and recommend services.
	detached := ctxmeta.WithUserID(context.Background(), strconv.FormatInt(studentID, 10))
	detached, cancel := context.WithTimeout(detached, 4*time.Minute)
	go func() {
		defer cancel()
		s.runGeneration(detached, studentID, job.ID, request, steps)
	}()

	return &resourcegendto.GenerateResourcesResponse{Job: jobView, Resources: jobView.Resources}, nil
}

// runGeneration executes the multi-agent pipeline for one job in the background.
// It advances each agent step through running/completed and bumps job progress so
// the client can visualize the run, persisting every generated resource as soon
// as it is ready. On panic it marks the job failed.
func (s *Service) runGeneration(ctx context.Context, studentID int64, jobID int64, request resourcegendto.GenerateResourcesRequest, steps []resourcegenentity.AgentStep) {
	var job *resourcegenentity.GenerationJob
	defer func() {
		if recovered := recover(); recovered != nil && job != nil {
			s.failJob(ctx, job, fmt.Sprintf("资源生成任务异常：%v", recovered))
		}
	}()

	stored, _, _, err := s.repo.GetJob(ctx, studentID, jobID)
	if err != nil {
		return
	}
	job = stored
	if s.stopIfJobCanceled(ctx, studentID, jobID, job) {
		return
	}

	stepByKey := make(map[string]*resourcegenentity.AgentStep, len(steps))
	for index := range steps {
		stepByKey[steps[index].AgentKey] = &steps[index]
	}

	genCtx := s.buildGenerationContext(ctx, studentID, request)
	summaries := s.stepSummaries(genCtx)

	job.KnowledgeID = genCtx.KnowledgeID
	job.KnowledgeName = genCtx.KnowledgeName
	job.CourseTitle = genCtx.CourseTitle
	job.Target = genCtx.Target

	// VisionReaderAgent runs first when images were uploaded, producing a text
	// digest that grounds every downstream writer agent.
	if visionStep := stepByKey["vision_reader"]; visionStep != nil {
		if s.stopIfJobCanceled(ctx, studentID, jobID, job) {
			return
		}
		s.bumpJob(ctx, job, 10, "VisionReaderAgent")
		s.advanceStep(ctx, visionStep, summaries["vision_reader"], resourcegenentity.StepStatusRunning)
		genCtx.ImageDigest = s.aiImageDigest(ctx, genCtx)
		doneSummary := summaries["vision_reader"]
		if strings.TrimSpace(genCtx.ImageDigest) == "" {
			doneSummary = "未能从上传图片中提取到可用要点，本次按文字上下文生成。"
		}
		s.advanceStep(ctx, visionStep, doneSummary, resourcegenentity.StepStatusCompleted)
	}

	if s.stopIfJobCanceled(ctx, studentID, jobID, job) {
		return
	}
	s.bumpJob(ctx, job, 14, "ProfileAnalystAgent")
	s.advanceStep(ctx, stepByKey["profile_analyst"], summaries["profile_analyst"], resourcegenentity.StepStatusRunning)
	s.advanceStep(ctx, stepByKey["profile_analyst"], summaries["profile_analyst"], resourcegenentity.StepStatusCompleted)
	s.bumpJob(ctx, job, 22, "CoursePlannerAgent")

	if s.stopIfJobCanceled(ctx, studentID, jobID, job) {
		return
	}
	s.advanceStep(ctx, stepByKey["course_planner"], summaries["course_planner"], resourcegenentity.StepStatusRunning)
	s.advanceStep(ctx, stepByKey["course_planner"], summaries["course_planner"], resourcegenentity.StepStatusCompleted)
	s.bumpJob(ctx, job, 32, "ResourceWriterAgents")

	if s.stopIfJobCanceled(ctx, studentID, jobID, job) {
		return
	}
	s.advanceStep(ctx, stepByKey["resource_writer"], summaries["resource_writer"], resourcegenentity.StepStatusRunning)
	s.generateResourcesIncrementally(ctx, genCtx, jobID, job)
	if s.stopIfJobCanceled(ctx, studentID, jobID, job) {
		return
	}
	s.advanceStep(ctx, stepByKey["resource_writer"], summaries["resource_writer"], resourcegenentity.StepStatusCompleted)

	s.bumpJob(ctx, job, 92, "SafetyReviewerAgent")
	if s.stopIfJobCanceled(ctx, studentID, jobID, job) {
		return
	}
	s.advanceStep(ctx, stepByKey["safety_reviewer"], summaries["safety_reviewer"], resourcegenentity.StepStatusRunning)
	s.advanceStep(ctx, stepByKey["safety_reviewer"], summaries["safety_reviewer"], resourcegenentity.StepStatusCompleted)

	if s.stopIfJobCanceled(ctx, studentID, jobID, job) {
		return
	}
	job.Status = resourcegenentity.JobStatusCompleted
	job.Progress = 100
	job.CurrentAgent = "SafetyReviewerAgent"
	_ = s.repo.UpdateJob(ctx, job)
}

func (s *Service) GetJob(ctx context.Context, rawJobID string) (*resourcegendto.JobView, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	jobID, err := strconv.ParseInt(strings.TrimSpace(rawJobID), 10, 64)
	if err != nil || jobID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	job, steps, resources, err := s.repo.GetJob(ctx, studentID, jobID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	view := jobToView(job, steps, resources)
	return &view, nil
}

// RetryJob starts a fresh generation run from a previous job. It does not
// mutate the old job, which keeps the audit trail clear for demos and exports.
func (s *Service) RetryJob(ctx context.Context, rawJobID string, request resourcegendto.RetryJobRequest) (*resourcegendto.GenerateResourcesResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	jobID, err := strconv.ParseInt(strings.TrimSpace(rawJobID), 10, 64)
	if err != nil || jobID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	job, _, resources, err := s.repo.GetJob(ctx, studentID, jobID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	resourceTypes := normalizeResourceTypes(request.ResourceTypes)
	if len(request.ResourceTypes) == 0 {
		resourceTypes = inferResourceTypes(resources)
	}
	retryTarget := strings.TrimSpace(request.Target)
	if retryTarget == "" {
		retryTarget = firstNonEmpty(job.Target, job.KnowledgeName)
	}

	response, err := s.Generate(ctx, resourcegendto.GenerateResourcesRequest{
		KnowledgeID:   job.KnowledgeID,
		KnowledgeName: firstNonEmpty(job.KnowledgeName, job.KnowledgeID),
		CourseTitle:   job.CourseTitle,
		Target:        retryTarget,
		ResourceTypes: resourceTypes,
		Difficulty:    request.Difficulty,
		Images:        request.Images,
	})
	if err != nil {
		return nil, err
	}
	if response != nil {
		s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
			StudentID:     studentID,
			EventType:     "resource_retry_started",
			KnowledgeID:   job.KnowledgeID,
			KnowledgeName: job.KnowledgeName,
			PayloadJSON: mustEventPayload(map[string]any{
				"retryOfJobId":  job.ID,
				"newJobId":      response.Job.ID,
				"resourceTypes": resourceTypes,
				"difficulty":    request.Difficulty,
			}),
		})
	}
	return response, nil
}

func (s *Service) CancelJob(ctx context.Context, rawJobID string) (*resourcegendto.JobView, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	job, steps, resources, err := s.getJobParts(ctx, studentID, rawJobID)
	if err != nil {
		return nil, err
	}
	if job.Status == resourcegenentity.JobStatusCompleted || job.Status == resourcegenentity.JobStatusFailed || job.Status == resourcegenentity.JobStatusCanceled {
		view := jobToView(job, steps, resources)
		return &view, nil
	}

	job.Status = resourcegenentity.JobStatusCanceled
	job.CurrentAgent = "已停止"
	job.ErrorMessage = "用户已停止本次资源生成任务。"
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	now := time.Now()
	for index := range steps {
		if steps[index].Status == resourcegenentity.StepStatusCompleted {
			continue
		}
		steps[index].Status = resourcegenentity.StepStatusCanceled
		steps[index].Summary = firstNonEmpty(steps[index].Summary, steps[index].AgentName) + "（已停止）"
		steps[index].EndedAt = &now
		_ = s.repo.UpdateStep(ctx, &steps[index])
	}
	s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
		StudentID:     studentID,
		EventType:     "resource_generation_canceled",
		KnowledgeID:   job.KnowledgeID,
		KnowledgeName: job.KnowledgeName,
		PayloadJSON:   mustEventPayload(map[string]any{"jobId": job.ID, "progress": job.Progress}),
	})
	view := jobToView(job, steps, resources)
	return &view, nil
}

func (s *Service) ListResources(ctx context.Context, query resourcegendto.ResourceQuery) (*resourcegendto.ListResourcesResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	status := normalizeLearningStatus(query.Status)
	resources, err := s.repo.ListResources(
		ctx,
		studentID,
		strings.TrimSpace(query.KnowledgeID),
		strings.TrimSpace(query.Type),
		strings.TrimSpace(query.Keyword),
		status,
		normalizeResourceSort(query.Sort),
		query.Limit,
	)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	statuses, _ := s.repo.ListPushStatuses(ctx, studentID, resourceIDs(resources))
	list := resourcesToViewsWithLearningStatus(resources, statuses)
	return &resourcegendto.ListResourcesResponse{List: list, Total: len(list)}, nil
}

func (s *Service) GetResource(ctx context.Context, rawResourceID string) (*resourcegendto.GeneratedResourceView, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := strconv.ParseInt(strings.TrimSpace(rawResourceID), 10, 64)
	if err != nil || resourceID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	resource, err := s.repo.GetResource(ctx, studentID, resourceID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	view := resourceToView(*resource)
	return &view, nil
}

func (s *Service) SubmitFeedback(ctx context.Context, rawResourceID string, request resourcegendto.SubmitFeedbackRequest) (*resourcegendto.FeedbackView, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := strconv.ParseInt(strings.TrimSpace(rawResourceID), 10, 64)
	if err != nil || resourceID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	if request.Rating < 1 || request.Rating > 5 {
		return nil, apperr.ErrBadRequest
	}
	resource, err := s.repo.GetResource(ctx, studentID, resourceID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	feedback := &resourcegenentity.ResourceFeedback{
		ResourceID: resource.ID,
		JobID:      resource.JobID,
		StudentID:  studentID,
		Rating:     request.Rating,
		Difficulty: normalizeDifficulty(request.Difficulty),
		Helpful:    request.Helpful,
		Completed:  request.Completed,
		Comment:    strings.TrimSpace(request.Comment),
		NextAction: buildNextAction(resource, request),
	}
	stored, err := s.repo.UpsertFeedback(ctx, feedback)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	_ = s.repo.UpdatePushStatusForResource(ctx, studentID, resource.ID, pushStatusFromFeedback(request))
	s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
		StudentID:     studentID,
		EventType:     "resource_feedback_submitted",
		KnowledgeID:   resource.KnowledgeID,
		KnowledgeName: resource.KnowledgeName,
		ResourceID:    resource.ID,
		ResourceType:  resource.ResourceType,
		PayloadJSON: mustEventPayload(map[string]any{
			"rating":     request.Rating,
			"difficulty": normalizeDifficulty(request.Difficulty),
			"helpful":    request.Helpful,
			"completed":  request.Completed,
			"comment":    strings.TrimSpace(request.Comment),
			"nextAction": stored.NextAction,
		}),
	})
	view := feedbackToView(*stored)
	return &view, nil
}

func (s *Service) UpdateLearningStatus(ctx context.Context, rawResourceID string, request resourcegendto.UpdateLearningStatusRequest) (*resourcegendto.ResourceLearningStatusResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := strconv.ParseInt(strings.TrimSpace(rawResourceID), 10, 64)
	if err != nil || resourceID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	status := normalizeLearningStatus(request.Status)
	if status == "" {
		return nil, apperr.ErrBadRequest
	}
	resource, err := s.repo.GetResource(ctx, studentID, resourceID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	if err := s.repo.UpdatePushStatusForResource(ctx, studentID, resource.ID, status); err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	eventType := "resource_status_updated"
	if status == "completed" {
		eventType = "resource_completed"
	}
	s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
		StudentID:     studentID,
		EventType:     eventType,
		KnowledgeID:   resource.KnowledgeID,
		KnowledgeName: resource.KnowledgeName,
		ResourceID:    resource.ID,
		ResourceType:  resource.ResourceType,
		PayloadJSON: mustEventPayload(map[string]any{
			"status": status,
			"note":   strings.TrimSpace(request.Note),
		}),
	})
	return &resourcegendto.ResourceLearningStatusResponse{
		ResourceID:     resource.ID,
		LearningStatus: status,
		ActionText:     pushActionText(resource.ResourceType, status),
		UpdatedAt:      time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Service) GetRecommendations(ctx context.Context) (*resourcegendto.ResourceRecommendationsResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}

	var profile *profiledto.LearningProfileView
	if s.profileService != nil {
		if response, err := s.profileService.GetCurrent(ctx); err == nil && response != nil {
			profile = response.Profile
		}
	}

	resources, err := s.repo.ListResources(ctx, studentID, "", "", "", "", "", 12)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	feedback, err := s.repo.ListFeedback(ctx, studentID, 30)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	summary := buildFeedbackSummary(feedback)
	recommendations := buildRecommendations(profile, resources, summary)

	// Prefer model-driven recommendations that adapt to recent feedback; fall
	// back to the heuristic list when the model is unavailable or unusable.
	profileSummaryText := ""
	if profile != nil {
		profileSummaryText = profile.Summary
	}
	if aiRecs, ok := s.aiRecommendations(ctx, profileSummaryText, dimensionsMap(profile), resources, feedback, summary); ok {
		recommendations = aiRecs
	}

	return &resourcegendto.ResourceRecommendationsResponse{
		Summary:         buildRecommendationSummary(profile, resources, summary),
		FeedbackSummary: summary,
		List:            recommendations,
	}, nil
}

// GetLearningReport 汇总学习画像、资源、反馈与学习效果，生成可分享/可提交的
// Markdown 学情报告，串起“画像 → 资源 → 测评 → 效果”的完整证据链。
func (s *Service) GetLearningReport(ctx context.Context) (*resourcegendto.LearningReportResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}

	var profile *profiledto.LearningProfileView
	if s.profileService != nil {
		if response, profErr := s.profileService.GetCurrent(ctx); profErr == nil && response != nil {
			profile = response.Profile
		}
	}

	resources, err := s.repo.ListResources(ctx, studentID, "", "", "", "", "", 100)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	feedback, err := s.repo.ListFeedback(ctx, studentID, 60)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	summary := buildFeedbackSummary(feedback)

	knowledgeSet := make(map[string]struct{})
	for _, resource := range resources {
		key := firstNonEmpty(strings.TrimSpace(resource.KnowledgeID), strings.TrimSpace(resource.KnowledgeName))
		if key != "" {
			knowledgeSet[key] = struct{}{}
		}
	}

	var builder strings.Builder
	builder.WriteString("# 个人学情学习报告\n\n")
	builder.WriteString(fmt.Sprintf("生成时间：%s\n\n", time.Now().Format("2006-01-02 15:04")))

	builder.WriteString("## 一、学习画像\n\n")
	if profile != nil {
		builder.WriteString(fmt.Sprintf("画像摘要：%s\n\n", firstNonEmpty(profile.Summary, "暂无摘要")))
		if len(profile.Dimensions) > 0 {
			builder.WriteString("| 维度 | 画像值 | 置信度 |\n| --- | --- | --- |\n")
			for _, dim := range profile.Dimensions {
				builder.WriteString(fmt.Sprintf("| %s | %s | %d%% |\n", dim.Name, oneLine(dim.Value), int(dim.Confidence*100+0.5)))
			}
			builder.WriteString("\n")
		}
	} else {
		builder.WriteString("尚未生成学习画像。\n\n")
	}

	builder.WriteString("## 二、学习效果概览\n\n")
	builder.WriteString(fmt.Sprintf("- 已生成资源：%d 份，覆盖知识点：%d 个\n", len(resources), len(knowledgeSet)))
	builder.WriteString(fmt.Sprintf("- 反馈记录：%d 条，完成学习：%d 次\n", summary.Total, summary.CompletedCount))
	if summary.AverageRating > 0 {
		builder.WriteString(fmt.Sprintf("- 平均评分：%.1f / 5\n", summary.AverageRating))
	}
	builder.WriteString(fmt.Sprintf("- 难度分布：偏易 %d / 适中 %d / 偏难 %d\n", summary.Difficulty["easy"], summary.Difficulty["fit"], summary.Difficulty["hard"]))
	builder.WriteString(fmt.Sprintf("- 学习建议：%s\n\n", summary.Insight))

	builder.WriteString("## 三、近期个性化资源\n\n")
	if len(resources) == 0 {
		builder.WriteString("暂无生成资源。\n\n")
	} else {
		limit := len(resources)
		if limit > 10 {
			limit = 10
		}
		for _, resource := range resources[:limit] {
			builder.WriteString(fmt.Sprintf("- [%s] %s（%s）\n", resource.TypeLabel, oneLine(resource.Title), firstNonEmpty(resource.KnowledgeName, "通用")))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("## 四、下一步建议\n\n")
	builder.WriteString("- 优先攻克偏难与低评分对应的知识点，配合分步练习与代码案例。\n")
	builder.WriteString("- 完成随堂练习以更新掌握度，并据此重新规划学习计划。\n")

	content := builder.String()
	return &resourcegendto.LearningReportResponse{
		FileName:    fmt.Sprintf("learning-report-%d.md", studentID),
		Content:     content,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// oneLine 去除换行，便于写入 Markdown 表格单元格。
func oneLine(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\n", " "), "|", "/"))
}

// GetEffectSummary aggregates generated resources, learner feedback and recent
// learning behaviour into a learning effect-assessment dashboard, evidencing the
// "学习效果评估 + 随学随新" closed loop required by the A3 problem.
func (s *Service) GetEffectSummary(ctx context.Context) (*resourcegendto.EffectSummaryResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}

	resources, err := s.repo.ListResources(ctx, studentID, "", "", "", "", "", 100)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	feedback, err := s.repo.ListFeedback(ctx, studentID, 60)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	events, err := s.repo.ListLearningEvents(ctx, studentID, 12)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	summary := buildFeedbackSummary(feedback)

	// 统计覆盖的知识点数量，用于体现学习广度。
	knowledgeSet := make(map[string]struct{})
	for _, resource := range resources {
		key := strings.TrimSpace(resource.KnowledgeID)
		if key == "" {
			key = strings.TrimSpace(resource.KnowledgeName)
		}
		if key != "" {
			knowledgeSet[key] = struct{}{}
		}
	}

	helpfulRate := 0
	completionRate := 0
	if summary.Total > 0 {
		helpfulRate = int(float64(summary.HelpfulCount)/float64(summary.Total)*100 + 0.5)
		completionRate = int(float64(summary.CompletedCount)/float64(summary.Total)*100 + 0.5)
	}

	averageRatingText := "暂无"
	if summary.AverageRating > 0 {
		averageRatingText = strconv.FormatFloat(summary.AverageRating, 'f', 1, 64)
	}

	metrics := []resourcegendto.EffectMetricView{
		{Key: "resources", Label: "生成资源", Value: strconv.Itoa(len(resources)), Hint: "多智能体已生成的资源总数"},
		{Key: "knowledge", Label: "覆盖知识点", Value: strconv.Itoa(len(knowledgeSet)), Hint: "资源关联的不同知识点"},
		{Key: "completed", Label: "完成学习", Value: strconv.Itoa(summary.CompletedCount), Hint: completionHint(completionRate, summary.Total)},
		{Key: "rating", Label: "平均评分", Value: averageRatingText, Hint: "满分 5 分"},
		{Key: "helpful", Label: "有帮助率", Value: strconv.Itoa(helpfulRate) + "%", Hint: strconv.Itoa(summary.HelpfulCount) + " 条反馈认为有帮助"},
	}

	highlights := buildEffectHighlights(summary, len(resources), len(knowledgeSet), events)

	return &resourcegendto.EffectSummaryResponse{
		Metrics:      metrics,
		Difficulty:   summary.Difficulty,
		Highlights:   highlights,
		RecentEvents: learningEventsToViews(events),
		GeneratedAt:  time.Now().Format(time.RFC3339),
	}, nil
}

// GenerateQuiz 生成知识点结构化互动练习。优先用大模型出题，模型不可用时
// 回退到本地模板题，保证离线演示也能完成“答题→效果回写”闭环。
func (s *Service) GenerateQuiz(ctx context.Context, request resourcegendto.GenerateQuizRequest) (*resourcegendto.GenerateQuizResponse, error) {
	if _, err := currentStudentID(ctx); err != nil {
		return nil, err
	}
	knowledgeName := strings.TrimSpace(request.KnowledgeName)
	if knowledgeName == "" {
		return nil, apperr.ErrBadRequest
	}
	courseTitle := strings.TrimSpace(request.CourseTitle)
	count := request.Count
	if count <= 0 || count > 10 {
		count = 3
	}

	source := "template"
	questions, ok := s.aiQuiz(ctx, knowledgeName, courseTitle, count)
	if ok {
		source = "ai"
	} else {
		questions = buildTemplateQuiz(knowledgeName, count)
	}
	if len(questions) > count {
		questions = questions[:count]
	}

	return &resourcegendto.GenerateQuizResponse{
		KnowledgeID:   strings.TrimSpace(request.KnowledgeID),
		KnowledgeName: knowledgeName,
		CourseTitle:   courseTitle,
		Questions:     questions,
		Source:        source,
	}, nil
}

// sanitizeQuizQuestions 清洗模型输出，保证每题至少 2 个选项且答案下标合法。
func sanitizeQuizQuestions(input []resourcegendto.QuizQuestionView) []resourcegendto.QuizQuestionView {
	cleaned := make([]resourcegendto.QuizQuestionView, 0, len(input))
	for _, question := range input {
		stem := strings.TrimSpace(question.Stem)
		options := make([]string, 0, len(question.Options))
		for _, option := range question.Options {
			if trimmed := strings.TrimSpace(option); trimmed != "" {
				options = append(options, trimmed)
			}
		}
		if stem == "" || len(options) < 2 {
			continue
		}
		answer := question.Answer
		if answer < 0 || answer >= len(options) {
			answer = 0
		}
		cleaned = append(cleaned, resourcegendto.QuizQuestionView{
			Stem:        stem,
			Options:     options,
			Answer:      answer,
			Explanation: strings.TrimSpace(question.Explanation),
		})
	}
	return cleaned
}

// buildTemplateQuiz 在模型不可用时生成可演示的本地模板题。
func buildTemplateQuiz(knowledgeName string, count int) []resourcegendto.QuizQuestionView {
	base := []resourcegendto.QuizQuestionView{
		{
			Stem: "关于「" + knowledgeName + "」，下列说法最准确的是？",
			Options: []string{
				knowledgeName + " 是该课程的核心概念之一，需要结合实例理解。",
				knowledgeName + " 与课程内容无关，可以直接跳过。",
				knowledgeName + " 只需死记硬背，无需理解原理。",
				knowledgeName + " 在实际应用中没有价值。",
			},
			Answer:      0,
			Explanation: "正确选项强调结合实例理解核心概念，其余选项是常见学习误区。",
		},
		{
			Stem: "学习「" + knowledgeName + "」时，较合理的学习顺序是？",
			Options: []string{
				"先理解概念与前置知识，再通过案例与练习巩固。",
				"直接背诵结论，不看任何例子。",
				"只做题，不看讲解与原理。",
				"跳跃式学习，不关注前置知识。",
			},
			Answer:      0,
			Explanation: "概念→案例→练习是更稳健的学习路径，能减少易错点。",
		},
		{
			Stem: "检验自己是否掌握「" + knowledgeName + "」的较好方式是？",
			Options: []string{
				"能用自己的话讲清概念，并独立完成相关练习或小项目。",
				"只要看过一遍讲解就算掌握。",
				"能背出定义即可，不需要应用。",
				"做对一道题就说明完全掌握。",
			},
			Answer:      0,
			Explanation: "能复述并迁移应用，才是掌握知识点的可靠标志。",
		},
	}

	if count <= 0 {
		count = len(base)
	}
	questions := make([]resourcegendto.QuizQuestionView, 0, count)
	for i := 0; i < count; i++ {
		questions = append(questions, base[i%len(base)])
	}
	return questions
}

// GenerateImage calls a configured real text-to-image model and stores the
// output as a generated learning resource.
func (s *Service) GenerateImage(ctx context.Context, request resourcegendto.GenerateMediaRequest) (*resourcegendto.MediaGenerationResponse, error) {
	return s.generateMedia(ctx, "image", request)
}

// GenerateVideo calls a configured real text-to-video model and stores the
// output as a generated learning resource.
func (s *Service) GenerateVideo(ctx context.Context, request resourcegendto.GenerateMediaRequest) (*resourcegendto.MediaGenerationResponse, error) {
	return s.generateMedia(ctx, "video", request)
}

func (s *Service) generateMedia(ctx context.Context, kind string, request resourcegendto.GenerateMediaRequest) (*resourcegendto.MediaGenerationResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	if s.aiClient == nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(errors.New("media generation model is not configured"))
	}

	genCtx := s.buildGenerationContext(ctx, studentID, resourcegendto.GenerateResourcesRequest{
		KnowledgeID:   request.KnowledgeID,
		KnowledgeName: request.KnowledgeName,
		CourseTitle:   request.CourseTitle,
		Target:        request.Target,
	})
	prompt := buildMediaPrompt(kind, genCtx, request.Prompt)
	mediaReq := ai.MediaRequest{
		Prompt:      prompt,
		Size:        firstNonEmpty(strings.TrimSpace(request.Size), defaultMediaSize(kind)),
		DurationSec: request.DurationSec,
		Timeout:     2 * time.Minute,
	}
	if kind == "video" && mediaReq.DurationSec <= 0 {
		mediaReq.DurationSec = 6
	}

	var media ai.MediaResponse
	switch kind {
	case "video":
		media, err = s.aiClient.GenerateVideo(ctx, mediaReq)
	default:
		media, err = s.aiClient.GenerateImage(ctx, mediaReq)
		kind = "image"
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	mediaReady := media.URL != "" || media.B64JSON != ""
	jobStatus := resourcegenentity.JobStatusCompleted
	jobProgress := 100
	if !mediaReady && media.ProviderTaskID != "" {
		jobStatus = resourcegenentity.JobStatusRunning
		jobProgress = 70
	}

	job := &resourcegenentity.GenerationJob{
		StudentID:     studentID,
		KnowledgeID:   genCtx.KnowledgeID,
		KnowledgeName: genCtx.KnowledgeName,
		CourseTitle:   genCtx.CourseTitle,
		Target:        genCtx.Target,
		Status:        jobStatus,
		Progress:      jobProgress,
		CurrentAgent:  mediaAgentName(kind),
	}
	if err := s.repo.CreateJob(ctx, job); err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	now := time.Now()
	stepStatus := resourcegenentity.StepStatusCompleted
	var endedAt *time.Time = &now
	if !mediaReady && media.ProviderTaskID != "" {
		stepStatus = resourcegenentity.StepStatusRunning
		endedAt = nil
	}
	step := &resourcegenentity.AgentStep{
		JobID:     job.ID,
		StudentID: studentID,
		AgentKey:  mediaAgentKey(kind),
		AgentName: mediaAgentName(kind),
		Role:      mediaAgentRole(kind),
		Status:    stepStatus,
		Summary:   mediaStepSummary(kind, media),
		StartedAt: now,
		EndedAt:   endedAt,
	}
	if err := s.repo.CreateStep(ctx, step); err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	content := marshalMediaContent(kind, media, prompt)
	resourceStatus := resourcegenentity.ResourceStatusReady
	if !mediaReady {
		resourceStatus = resourcegenentity.ResourceStatusPending
	}
	resource := &resourcegenentity.GeneratedResource{
		JobID:         job.ID,
		StudentID:     studentID,
		ResourceType:  mediaResourceType(kind),
		TypeLabel:     mediaTypeLabel(kind),
		Title:         mediaTypeLabel(kind) + "：" + genCtx.KnowledgeName,
		Content:       content,
		Format:        kind,
		KnowledgeID:   genCtx.KnowledgeID,
		KnowledgeName: genCtx.KnowledgeName,
		CourseTitle:   genCtx.CourseTitle,
		Reason:        mediaResourceReason(kind, media),
		Status:        resourceStatus,
	}
	if err := s.repo.CreateResource(ctx, resource); err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	_ = s.repo.CreatePushItem(ctx, buildPushItem(resource, genCtx))
	s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
		StudentID:     studentID,
		EventType:     mediaEventType(kind),
		KnowledgeID:   resource.KnowledgeID,
		KnowledgeName: resource.KnowledgeName,
		ResourceID:    resource.ID,
		ResourceType:  resource.ResourceType,
		PayloadJSON: mustEventPayload(map[string]any{
			"title":          resource.Title,
			"provider":       media.Provider,
			"model":          media.Model,
			"status":         media.Status,
			"providerTaskId": media.ProviderTaskID,
		}),
	})

	view := resourceToView(*resource)
	return &resourcegendto.MediaGenerationResponse{
		Kind:           kind,
		Resource:       view,
		URL:            media.URL,
		B64JSON:        media.B64JSON,
		MimeType:       media.MimeType,
		Prompt:         prompt,
		Provider:       media.Provider,
		Model:          media.Model,
		Status:         firstNonEmpty(media.Status, "ready"),
		ProviderTaskID: media.ProviderTaskID,
		GeneratedAt:    time.Now().Format(time.RFC3339),
	}, nil
}

// completionHint 描述资源完成比例。
func completionHint(rate int, total int) string {
	if total == 0 {
		return "完成学习后提交反馈即可统计"
	}
	return "占已反馈资源的 " + strconv.Itoa(rate) + "%"
}

// buildEffectHighlights 生成学习效果评估的文字洞察。
func buildEffectHighlights(summary resourcegendto.FeedbackSummaryView, resourceCount int, knowledgeCount int, events []resourcegenentity.LearningEvent) []string {
	highlights := make([]string, 0, 3)
	if resourceCount == 0 {
		return []string{"还没有生成资源，先完成一次资源生成即可看到学习效果评估。"}
	}
	highlights = append(highlights, "已生成 "+strconv.Itoa(resourceCount)+" 份个性化资源，覆盖 "+strconv.Itoa(knowledgeCount)+" 个知识点。")
	if summary.Total > 0 {
		highlights = append(highlights, summary.Insight)
	}
	if len(events) > 0 {
		highlights = append(highlights, "近期记录到 "+strconv.Itoa(len(events))+" 条学习行为信号，画像会据此随学随新。")
	}
	return highlights
}

func (s *Service) GetFeed(ctx context.Context) (*resourcegendto.ResourceFeedResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	items, resources, err := s.repo.ListPushItems(ctx, studentID, 24)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	recentEvents, err := s.repo.ListLearningEvents(ctx, studentID, 12)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	resourceByID := make(map[int64]resourcegenentity.GeneratedResource, len(resources))
	for _, resource := range resources {
		resourceByID[resource.ID] = resource
	}

	views := make([]resourcegendto.ResourcePushItemView, 0, len(items))
	for _, item := range items {
		resource, ok := resourceByID[item.ResourceID]
		if !ok {
			continue
		}
		views = append(views, pushItemToView(item, resource))
	}

	if len(views) == 0 {
		fallback, err := s.repo.ListResources(ctx, studentID, "", "", "", "", "", 8)
		if err != nil {
			return nil, apperr.ErrDependencyUnavailable.WithCause(err)
		}
		for index, resource := range fallback {
			item := resourcegenentity.ResourcePushItem{
				ID:            int64(-index - 1),
				StudentID:     studentID,
				ResourceID:    resource.ID,
				JobID:         resource.JobID,
				KnowledgeID:   resource.KnowledgeID,
				KnowledgeName: resource.KnowledgeName,
				CourseTitle:   resource.CourseTitle,
				Stage:         resourceStage(resource.ResourceType),
				Reason:        buildPushReason(resource.ResourceType, resource.Reason),
				Priority:      resourcePriority(resource.ResourceType),
				Status:        "pending",
				CreatedAt:     resource.CreatedAt,
			}
			views = append(views, pushItemToView(item, resource))
		}
	}

	return &resourcegendto.ResourceFeedResponse{
		Summary:      buildFeedSummary(views, recentEvents),
		Items:        views,
		RecentEvents: learningEventsToViews(recentEvents),
	}, nil
}

func (s *Service) RecordLearningEvent(ctx context.Context, request resourcegendto.RecordLearningEventRequest) (*resourcegendto.LearningEventView, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	eventType := normalizeEventType(request.EventType)
	if eventType == "" {
		return nil, apperr.ErrBadRequest
	}

	knowledgeID := strings.TrimSpace(request.KnowledgeID)
	knowledgeName := strings.TrimSpace(request.KnowledgeName)
	resourceType := strings.TrimSpace(request.ResourceType)
	resourceID := request.ResourceID
	if resourceID > 0 {
		if resource, err := s.repo.GetResource(ctx, studentID, resourceID); err == nil && resource != nil {
			knowledgeID = firstNonEmpty(knowledgeID, resource.KnowledgeID)
			knowledgeName = firstNonEmpty(knowledgeName, resource.KnowledgeName)
			resourceType = firstNonEmpty(resourceType, resource.ResourceType)
		}
	}

	event := resourcegenentity.LearningEvent{
		StudentID:     studentID,
		EventType:     eventType,
		KnowledgeID:   knowledgeID,
		KnowledgeName: knowledgeName,
		ResourceID:    resourceID,
		ResourceType:  resourceType,
		PayloadJSON:   mustEventPayload(request.Payload),
		OccurredAt:    time.Now(),
	}
	if err := s.repo.CreateLearningEvent(ctx, &event); err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	view := learningEventToView(event)
	return &view, nil
}

func (s *Service) GetSafetyReport(ctx context.Context, rawResourceID string) (*resourcegendto.SafetyReportView, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := strconv.ParseInt(strings.TrimSpace(rawResourceID), 10, 64)
	if err != nil || resourceID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	resource, err := s.repo.GetResource(ctx, studentID, resourceID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	if report, ok := s.aiSafetyReport(ctx, *resource); ok {
		return &report, nil
	}
	report := buildSafetyReport(*resource)
	return &report, nil
}

func (s *Service) GetMediaStatus(ctx context.Context, rawResourceID string) (*resourcegendto.MediaStatusResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := strconv.ParseInt(strings.TrimSpace(rawResourceID), 10, 64)
	if err != nil || resourceID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	resource, err := s.repo.GetResource(ctx, studentID, resourceID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	if resource.Format != "image" && resource.Format != "video" && resource.ResourceType != "generated_image" && resource.ResourceType != "generated_video" {
		return nil, apperr.ErrBadRequest.WithCause(errors.New("resource is not a generated media artifact"))
	}
	status := buildMediaStatus(*resource)
	if !status.Ready && status.ProviderTaskID != "" && s.aiClient != nil {
		if latest, pollErr := s.aiClient.GetMediaStatus(ctx, status.Kind, status.ProviderTaskID); pollErr == nil {
			latest.Prompt = status.Prompt
			if latest.Provider == "" {
				latest.Provider = status.Provider
			}
			if latest.Model == "" {
				latest.Model = status.Model
			}
			resource.Content = marshalMediaContent(status.Kind, latest, status.Prompt)
			if latest.URL != "" || latest.B64JSON != "" {
				resource.Status = resourcegenentity.ResourceStatusReady
			} else if mediaStatusFailed(latest.Status) {
				resource.Status = resourcegenentity.ResourceStatusFailed
			}
			_ = s.repo.UpdateResource(ctx, resource)
			s.syncMediaJob(ctx, resource, latest)
			status = buildMediaStatus(*resource)
		}
	}
	s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
		StudentID:     studentID,
		EventType:     "media_status_checked",
		KnowledgeID:   resource.KnowledgeID,
		KnowledgeName: resource.KnowledgeName,
		ResourceID:    resource.ID,
		ResourceType:  resource.ResourceType,
		PayloadJSON: mustEventPayload(map[string]any{
			"status":         status.Status,
			"ready":          status.Ready,
			"providerTaskId": status.ProviderTaskID,
		}),
	})
	return &status, nil
}

func (s *Service) syncMediaJob(ctx context.Context, resource *resourcegenentity.GeneratedResource, media ai.MediaResponse) {
	job, steps, _, err := s.repo.GetJob(ctx, resource.StudentID, resource.JobID)
	if err != nil {
		return
	}
	ready := media.URL != "" || media.B64JSON != ""
	if ready {
		job.Status = resourcegenentity.JobStatusCompleted
		job.Progress = 100
	} else if mediaStatusFailed(media.Status) {
		job.Status = resourcegenentity.JobStatusFailed
		job.ErrorMessage = "媒体提供商返回失败状态：" + media.Status
	} else {
		job.Status = resourcegenentity.JobStatusRunning
		job.Progress = 80
	}
	_ = s.repo.UpdateJob(ctx, job)
	for index := range steps {
		if ready {
			steps[index].Status = resourcegenentity.StepStatusCompleted
			now := time.Now()
			steps[index].EndedAt = &now
			steps[index].Summary = "真实媒体已生成并可在资源详情页预览。"
		} else if mediaStatusFailed(media.Status) {
			steps[index].Status = resourcegenentity.StepStatusCanceled
			steps[index].Summary = "媒体提供商返回失败状态：" + media.Status
		}
		_ = s.repo.UpdateStep(ctx, &steps[index])
	}
}

func mediaStatusFailed(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return status == "failed" || status == "error" || status == "canceled" || status == "cancelled"
}

func (s *Service) GetQualityReport(ctx context.Context, rawResourceID string) (*resourcegendto.ResourceQualityReportView, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := strconv.ParseInt(strings.TrimSpace(rawResourceID), 10, 64)
	if err != nil || resourceID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	resource, err := s.repo.GetResource(ctx, studentID, resourceID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	feedback, _ := s.repo.ListFeedback(ctx, studentID, 80)
	report := buildQualityReport(*resource, latestFeedbackForResource(feedback, resource.ID))
	s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
		StudentID:     studentID,
		EventType:     "resource_quality_checked",
		KnowledgeID:   resource.KnowledgeID,
		KnowledgeName: resource.KnowledgeName,
		ResourceID:    resource.ID,
		ResourceType:  resource.ResourceType,
		PayloadJSON: mustEventPayload(map[string]any{
			"score": report.Score,
			"level": report.Level,
		}),
	})
	return &report, nil
}

func (s *Service) CompareResources(ctx context.Context, request resourcegendto.CompareResourcesRequest) (*resourcegendto.CompareResourcesResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	if len(request.ResourceIDs) < 2 || len(request.ResourceIDs) > 6 {
		return nil, apperr.ErrBadRequest
	}
	feedback, _ := s.repo.ListFeedback(ctx, studentID, 100)
	items := make([]resourcegendto.ResourceComparisonItemView, 0, len(request.ResourceIDs))
	seen := map[int64]struct{}{}
	for _, resourceID := range request.ResourceIDs {
		if resourceID <= 0 {
			return nil, apperr.ErrBadRequest
		}
		if _, exists := seen[resourceID]; exists {
			continue
		}
		seen[resourceID] = struct{}{}
		resource, err := s.repo.GetResource(ctx, studentID, resourceID)
		if errors.Is(err, resourcegenrepo.ErrNotFound) {
			return nil, apperr.ErrNotFound
		}
		if err != nil {
			return nil, apperr.ErrDependencyUnavailable.WithCause(err)
		}
		report := buildQualityReport(*resource, latestFeedbackForResource(feedback, resource.ID))
		items = append(items, resourcegendto.ResourceComparisonItemView{
			Resource: resourceToView(*resource),
			Quality:  report,
			Reason:   comparisonReason(report),
		})
	}
	if len(items) < 2 {
		return nil, apperr.ErrBadRequest
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Quality.Score > items[j].Quality.Score
	})
	winnerID := items[0].Resource.ID
	s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
		StudentID:   studentID,
		EventType:   "resource_compared",
		ResourceID:  winnerID,
		PayloadJSON: mustEventPayload(map[string]any{"winnerId": winnerID, "count": len(items)}),
	})
	return &resourcegendto.CompareResourcesResponse{
		WinnerID:    winnerID,
		Summary:     buildComparisonSummary(items),
		Items:       items,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Service) GetJobReviewSummary(ctx context.Context, rawJobID string) (*resourcegendto.JobReviewSummaryResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	job, _, resources, err := s.getJobParts(ctx, studentID, rawJobID)
	if err != nil {
		return nil, err
	}
	feedback, _ := s.repo.ListFeedback(ctx, studentID, 100)
	review := buildJobReviewSummary(job, resources, feedback)
	s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
		StudentID:     studentID,
		EventType:     "job_review_generated",
		KnowledgeID:   job.KnowledgeID,
		KnowledgeName: job.KnowledgeName,
		PayloadJSON: mustEventPayload(map[string]any{
			"jobId":          job.ID,
			"averageQuality": review.AverageQuality,
			"averageSafety":  review.AverageSafety,
		}),
	})
	return &review, nil
}

func (s *Service) GetJobEvidencePackage(ctx context.Context, rawJobID string) (*resourcegendto.JobEvidencePackageResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	job, steps, resources, err := s.getJobParts(ctx, studentID, rawJobID)
	if err != nil {
		return nil, err
	}
	feedback, _ := s.repo.ListFeedback(ctx, studentID, 100)
	packageView := buildEvidencePackage(job, steps, resources, feedback)
	s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
		StudentID:     studentID,
		EventType:     "evidence_package_exported",
		KnowledgeID:   job.KnowledgeID,
		KnowledgeName: job.KnowledgeName,
		PayloadJSON:   mustEventPayload(map[string]any{"jobId": job.ID, "resourceCount": len(resources)}),
	})
	return &packageView, nil
}

func (s *Service) GetStudyChecklist(ctx context.Context, rawJobID string) (*resourcegendto.StudyChecklistResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	job, _, resources, err := s.getJobParts(ctx, studentID, rawJobID)
	if err != nil {
		return nil, err
	}
	checklist := buildStudyChecklist(job, resources)
	s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
		StudentID:     studentID,
		EventType:     "study_checklist_generated",
		KnowledgeID:   job.KnowledgeID,
		KnowledgeName: job.KnowledgeName,
		PayloadJSON:   mustEventPayload(map[string]any{"jobId": job.ID, "items": len(checklist.Items)}),
	})
	return &checklist, nil
}

func (s *Service) GenerateVariant(ctx context.Context, rawResourceID string, request resourcegendto.ResourceVariantRequest) (*resourcegendto.GenerateResourcesResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := strconv.ParseInt(strings.TrimSpace(rawResourceID), 10, 64)
	if err != nil || resourceID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	resource, err := s.repo.GetResource(ctx, studentID, resourceID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	target := variantTarget(*resource, request)
	modeType := variantResourceType(resource.ResourceType, request.Mode)

	if resource.ResourceType == "generated_image" {
		media, err := s.GenerateImage(ctx, resourcegendto.GenerateMediaRequest{
			KnowledgeID:   resource.KnowledgeID,
			KnowledgeName: firstNonEmpty(resource.KnowledgeName, resource.KnowledgeID),
			CourseTitle:   resource.CourseTitle,
			Target:        target,
			Prompt:        target,
			Size:          "1024x1024",
		})
		if err != nil {
			return nil, err
		}
		return s.responseForGeneratedResource(ctx, studentID, media.Resource.JobID)
	}
	if resource.ResourceType == "generated_video" {
		media, err := s.GenerateVideo(ctx, resourcegendto.GenerateMediaRequest{
			KnowledgeID:   resource.KnowledgeID,
			KnowledgeName: firstNonEmpty(resource.KnowledgeName, resource.KnowledgeID),
			CourseTitle:   resource.CourseTitle,
			Target:        target,
			Prompt:        target,
			Size:          "1280x720",
			DurationSec:   6,
		})
		if err != nil {
			return nil, err
		}
		return s.responseForGeneratedResource(ctx, studentID, media.Resource.JobID)
	}

	response, err := s.Generate(ctx, resourcegendto.GenerateResourcesRequest{
		KnowledgeID:   resource.KnowledgeID,
		KnowledgeName: firstNonEmpty(resource.KnowledgeName, resource.KnowledgeID),
		CourseTitle:   resource.CourseTitle,
		Target:        target,
		ResourceTypes: []string{modeType},
		Difficulty:    request.Difficulty,
	})
	if err != nil {
		return nil, err
	}
	if response != nil {
		s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
			StudentID:     studentID,
			EventType:     "resource_variant_started",
			KnowledgeID:   resource.KnowledgeID,
			KnowledgeName: resource.KnowledgeName,
			ResourceID:    resource.ID,
			ResourceType:  resource.ResourceType,
			PayloadJSON: mustEventPayload(map[string]any{
				"mode":     request.Mode,
				"target":   target,
				"newJobId": response.Job.ID,
			}),
		})
	}
	return response, nil
}

func (s *Service) GetProvenance(ctx context.Context, rawResourceID string) (*resourcegendto.ResourceProvenanceResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := strconv.ParseInt(strings.TrimSpace(rawResourceID), 10, 64)
	if err != nil || resourceID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	resource, err := s.repo.GetResource(ctx, studentID, resourceID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	job, steps, _, err := s.repo.GetJob(ctx, studentID, resource.JobID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	events, _ := s.repo.ListLearningEvents(ctx, studentID, 80)
	view := buildProvenance(resource, job, steps, events)
	s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
		StudentID:     studentID,
		EventType:     "resource_provenance_viewed",
		KnowledgeID:   resource.KnowledgeID,
		KnowledgeName: resource.KnowledgeName,
		ResourceID:    resource.ID,
		ResourceType:  resource.ResourceType,
		PayloadJSON:   mustEventPayload(map[string]any{"jobId": job.ID}),
	})
	return &view, nil
}

func (s *Service) ExportJobMarkdown(ctx context.Context, rawJobID string) (*resourcegendto.ExportJobResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	jobID, err := strconv.ParseInt(strings.TrimSpace(rawJobID), 10, 64)
	if err != nil || jobID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	job, steps, resources, err := s.repo.GetJob(ctx, studentID, jobID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	content := buildJobMarkdown(job, steps, resources)
	return &resourcegendto.ExportJobResponse{
		JobID:       job.ID,
		FileName:    fmt.Sprintf("resource-package-%d.md", job.ID),
		Content:     content,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Service) GetPresentationOutline(ctx context.Context, rawJobID string) (*resourcegendto.PresentationOutlineResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	jobID, err := strconv.ParseInt(strings.TrimSpace(rawJobID), 10, 64)
	if err != nil || jobID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	job, steps, resources, err := s.repo.GetJob(ctx, studentID, jobID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	outline := buildPresentationOutline(job, steps, resources)
	return &outline, nil
}

func (s *Service) GetDemoScript(ctx context.Context, rawJobID string) (*resourcegendto.DemoScriptResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	jobID, err := strconv.ParseInt(strings.TrimSpace(rawJobID), 10, 64)
	if err != nil || jobID <= 0 {
		return nil, apperr.ErrBadRequest
	}
	job, steps, resources, err := s.repo.GetJob(ctx, studentID, jobID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	script := buildDemoScript(job, steps, resources)
	return &script, nil
}

func (s *Service) buildGenerationContext(ctx context.Context, studentID int64, request resourcegendto.GenerateResourcesRequest) generationContext {
	knowledgeID := strings.TrimSpace(request.KnowledgeID)
	knowledgeName := strings.TrimSpace(request.KnowledgeName)
	courseTitle := strings.TrimSpace(request.CourseTitle)
	target := strings.TrimSpace(request.Target)

	var detail *graphdto.KnowledgeDetailView
	if s.graphService != nil && knowledgeID != "" {
		if loaded, err := s.graphService.GetKnowledgeDetail(ctx, knowledgeID); err == nil && loaded != nil {
			detail = loaded
			knowledgeName = firstNonEmpty(knowledgeName, loaded.CanonicalName, loaded.Name, knowledgeID)
			courseTitle = firstNonEmpty(courseTitle, loaded.ChapterTitle)
		}
	}

	var profile *profiledto.LearningProfileView
	if s.profileService != nil {
		if response, err := s.profileService.GetCurrent(ctx); err == nil && response != nil {
			profile = response.Profile
		}
	}
	dims := dimensionsMap(profile)
	target = firstNonEmpty(target, dims["learning_goal"], dims["learning_goal"], knowledgeName, "人工智能课程学习")

	var path []recommenddto.PathItemView
	if s.recommendService != nil {
		if response, err := s.recommendService.GetUnifiedPath(ctx, recommenddto.PathQuery{
			Target:      target,
			KnowledgeID: knowledgeID,
			Scene:       "resource_generation",
			Limit:       5,
		}); err == nil && response != nil {
			path = response.Path
		}
	}
	if knowledgeName == "" && len(path) > 0 {
		knowledgeName = firstNonEmpty(path[0].Title, path[0].KnowledgeID)
		knowledgeID = firstNonEmpty(knowledgeID, path[0].KnowledgeID)
	}
	knowledgeName = firstNonEmpty(knowledgeName, knowledgeID, "核心知识点")

	return generationContext{
		StudentID:       studentID,
		KnowledgeID:     knowledgeID,
		KnowledgeName:   knowledgeName,
		CourseTitle:     firstNonEmpty(courseTitle, "人工智能专业课程"),
		Target:          target,
		Profile:         profile,
		ProfileDims:     dims,
		KnowledgeDetail: detail,
		Path:            path,
		ResourceTypes:   normalizeResourceTypes(request.ResourceTypes),
		Difficulty:      normalizeDifficultyLevel(request.Difficulty),
		Images:          sanitizeImages(request.Images),
	}
}

// buildPreliminaryContext builds a lightweight generation context from the raw
// request alone (no external service calls) so the initial job and agent steps
// can be created and returned to the client immediately.
func buildPreliminaryContext(studentID int64, request resourcegendto.GenerateResourcesRequest) generationContext {
	return generationContext{
		StudentID:     studentID,
		KnowledgeID:   strings.TrimSpace(request.KnowledgeID),
		KnowledgeName: firstNonEmpty(strings.TrimSpace(request.KnowledgeName), strings.TrimSpace(request.KnowledgeID), "核心知识点"),
		CourseTitle:   firstNonEmpty(strings.TrimSpace(request.CourseTitle), "人工智能专业课程"),
		Target:        strings.TrimSpace(request.Target),
		ProfileDims:   map[string]string{},
		ResourceTypes: normalizeResourceTypes(request.ResourceTypes),
		Difficulty:    normalizeDifficultyLevel(request.Difficulty),
		Images:        sanitizeImages(request.Images),
	}
}

// normalizeDifficultyLevel keeps only the supported regeneration levels.
func normalizeDifficultyLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "easier":
		return "easier"
	case "harder":
		return "harder"
	default:
		return ""
	}
}

// sanitizeImages trims, drops empty entries and caps the number of images to a
// safe maximum for a single vision request.
func sanitizeImages(images []string) []string {
	const maxImages = 3
	result := make([]string, 0, len(images))
	for _, image := range images {
		trimmed := strings.TrimSpace(image)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
		if len(result) >= maxImages {
			break
		}
	}
	return result
}

// stepSummaries returns the per-agent step summaries for a generation context.
// It is reused both when steps are first created (preliminary context) and when
// they are advanced with the fully resolved context.
func (s *Service) stepSummaries(genCtx generationContext) map[string]string {
	return map[string]string{
		"vision_reader":   "读取学生上传的图片（教材页/题目/笔记），提取知识点与关键信息作为生成依据。",
		"profile_analyst": "读取学生画像、学习目标和资源偏好，形成个性化资源生成需求：" + profileSummary(genCtx),
		"course_planner":  "结合知识图谱和推荐路径，锁定“" + genCtx.KnowledgeName + "”及其先修/后续学习顺序。",
		"resource_writer": s.resourceWriterSummary(genCtx),
		"safety_reviewer": s.safetyReviewerSummary(),
	}
}

// agentStepSpec is the static definition of one multi-agent step.
type agentStepSpec struct {
	Key  string
	Name string
	Role string
}

// buildSteps creates the ordered multi-agent steps for a job in the pending
// state; runGeneration later advances them to running/completed as work proceeds.
// A VisionReaderAgent step is prepended only when images are supplied and a
// vision-capable model is configured.
func (s *Service) buildSteps(genCtx generationContext) []resourcegenentity.AgentStep {
	summaries := s.stepSummaries(genCtx)
	specs := make([]agentStepSpec, 0, 5)
	if len(genCtx.Images) > 0 && s.aiClient != nil && s.aiClient.VisionEnabled() {
		specs = append(specs, agentStepSpec{"vision_reader", "VisionReaderAgent", "多模态识别智能体"})
	}
	specs = append(specs,
		agentStepSpec{"profile_analyst", "ProfileAnalystAgent", "画像分析智能体"},
		agentStepSpec{"course_planner", "CoursePlannerAgent", "课程规划智能体"},
		agentStepSpec{"resource_writer", "ResourceWriterAgents", "资源生成智能体组"},
		agentStepSpec{"safety_reviewer", "SafetyReviewerAgent", "安全校验智能体"},
	)

	now := time.Now()
	result := make([]resourcegenentity.AgentStep, 0, len(specs))
	for _, spec := range specs {
		result = append(result, resourcegenentity.AgentStep{
			AgentKey:  spec.Key,
			AgentName: spec.Name,
			Role:      spec.Role,
			Status:    resourcegenentity.StepStatusPending,
			Summary:   summaries[spec.Key],
			StartedAt: now,
		})
	}
	return result
}

// advanceStep updates one agent step's status and summary and stamps its
// start/end time, persisting the change. It is a no-op when the step is missing.
func (s *Service) advanceStep(ctx context.Context, step *resourcegenentity.AgentStep, summary string, status string) {
	if step == nil {
		return
	}
	now := time.Now()
	if summary != "" {
		step.Summary = summary
	}
	step.Status = status
	switch status {
	case resourcegenentity.StepStatusRunning:
		step.StartedAt = now
		step.EndedAt = nil
	case resourcegenentity.StepStatusCompleted:
		if step.StartedAt.IsZero() {
			step.StartedAt = now
		}
		step.EndedAt = &now
	}
	_ = s.repo.UpdateStep(ctx, step)
}

// bumpJob updates the job's progress percentage and currently-active agent.
func (s *Service) bumpJob(ctx context.Context, job *resourcegenentity.GenerationJob, progress int, currentAgent string) {
	if s.stopIfJobCanceled(ctx, job.StudentID, job.ID, job) {
		return
	}
	job.Progress = progress
	if currentAgent != "" {
		job.CurrentAgent = currentAgent
	}
	_ = s.repo.UpdateJob(ctx, job)
}

// failJob marks a job as failed with a clamped error message.
func (s *Service) failJob(ctx context.Context, job *resourcegenentity.GenerationJob, message string) {
	if s.stopIfJobCanceled(ctx, job.StudentID, job.ID, job) {
		return
	}
	job.Status = resourcegenentity.JobStatusFailed
	job.ErrorMessage = clampRunes(message, 900)
	_ = s.repo.UpdateJob(ctx, job)
}

func (s *Service) stopIfJobCanceled(ctx context.Context, studentID int64, jobID int64, job *resourcegenentity.GenerationJob) bool {
	if s.isJobCanceled(ctx, studentID, jobID) {
		if job != nil {
			job.Status = resourcegenentity.JobStatusCanceled
			job.CurrentAgent = "已停止"
			job.ErrorMessage = "用户已停止本次资源生成任务。"
		}
		return true
	}
	return false
}

func (s *Service) isJobCanceled(ctx context.Context, studentID int64, jobID int64) bool {
	if studentID <= 0 || jobID <= 0 {
		return false
	}
	job, _, _, err := s.repo.GetJob(ctx, studentID, jobID)
	return err == nil && job != nil && job.Status == resourcegenentity.JobStatusCanceled
}

// generateResourcesIncrementally produces every requested resource concurrently
// (large model with template fallback) and persists each one as soon as it is
// ready, bumping job progress so the client sees resources appear one by one.
// Resource creation and progress updates are serialized by a mutex while the
// slow model calls run in parallel outside the lock.
func (s *Service) generateResourcesIncrementally(ctx context.Context, genCtx generationContext, jobID int64, job *resourcegenentity.GenerationJob) {
	total := len(genCtx.ResourceTypes)
	if total == 0 {
		return
	}

	var mu sync.Mutex
	completed := 0
	var wg sync.WaitGroup
	for _, resourceType := range genCtx.ResourceTypes {
		wg.Add(1)
		go func(resourceType string) {
			defer wg.Done()
			if s.isJobCanceled(ctx, genCtx.StudentID, jobID) {
				return
			}
			label := resourceTypeLabel(resourceType)
			content := s.aiResourceContent(ctx, resourceType, genCtx)
			if strings.TrimSpace(content) == "" {
				content = buildResourceContent(resourceType, genCtx)
			}
			content = appendGroundingEvidence(resourceType, content, genCtx)
			if s.isJobCanceled(ctx, genCtx.StudentID, jobID) {
				return
			}
			resource := &resourcegenentity.GeneratedResource{
				JobID:         jobID,
				StudentID:     genCtx.StudentID,
				ResourceType:  resourceType,
				TypeLabel:     label,
				Title:         label + "：" + genCtx.KnowledgeName,
				Content:       content,
				Format:        resourceFormat(resourceType),
				KnowledgeID:   genCtx.KnowledgeID,
				KnowledgeName: genCtx.KnowledgeName,
				CourseTitle:   genCtx.CourseTitle,
				Reason:        buildResourceReason(resourceType, genCtx),
				Status:        resourcegenentity.ResourceStatusReady,
			}
			safety, modelReviewed := s.aiSafetyReport(ctx, *resource)
			if !modelReviewed {
				safety = buildSafetyReport(*resource)
			}
			if safety.Status == "blocked" {
				resource.Status = resourcegenentity.ResourceStatusBlocked
			}

			mu.Lock()
			defer mu.Unlock()
			if s.isJobCanceled(ctx, genCtx.StudentID, jobID) {
				return
			}
			if err := s.repo.CreateResource(ctx, resource); err != nil {
				return
			}
			if resource.Status != resourcegenentity.ResourceStatusBlocked {
				_ = s.repo.CreatePushItem(ctx, buildPushItem(resource, genCtx))
			}
			s.recordLearningEvent(ctx, resourcegenentity.LearningEvent{
				StudentID:     genCtx.StudentID,
				EventType:     "resource_generated",
				KnowledgeID:   resource.KnowledgeID,
				KnowledgeName: resource.KnowledgeName,
				ResourceID:    resource.ID,
				ResourceType:  resource.ResourceType,
				PayloadJSON: mustEventPayload(map[string]any{
					"title":         resource.Title,
					"typeLabel":     resource.TypeLabel,
					"reason":        resource.Reason,
					"courseTitle":   resource.CourseTitle,
					"safetyStatus":  safety.Status,
					"safetyScore":   safety.Score,
					"modelReviewed": modelReviewed,
				}),
			})
			completed++
			s.bumpJob(ctx, job, 35+int(float64(53)*float64(completed)/float64(total)), writerAgentName(resourceType))
		}(resourceType)
	}
	wg.Wait()
}

// appendGroundingEvidence adds auditable graph/curation references to text
// artifacts. Mermaid remains syntax-only so it can still render correctly.
func appendGroundingEvidence(resourceType string, content string, ctx generationContext) string {
	if resourceType == "mind_map" || strings.TrimSpace(content) == "" {
		return content
	}
	if strings.Contains(content, "## 依据与核验") {
		return content
	}
	lines := []string{"", "## 依据与核验"}
	if ctx.KnowledgeDetail != nil {
		detail := ctx.KnowledgeDetail
		lines = append(lines, "- 知识图谱节点："+firstNonEmpty(detail.CanonicalName, detail.Name, ctx.KnowledgeName)+"（ID："+firstNonEmpty(detail.RouteKnowledgeID, detail.ID, ctx.KnowledgeID)+"）")
		if source := firstNonEmpty(detail.CuratedSource, detail.SourcePath); source != "" {
			lines = append(lines, "- 课程资料来源："+source)
		}
		if len(detail.Prerequisites) > 0 {
			lines = append(lines, "- 图谱先修依据："+strings.Join(detail.Prerequisites, "、"))
		}
	} else {
		lines = append(lines, "- 知识点："+ctx.KnowledgeName+"（当前未取得图谱详情，事实性内容需人工复核）")
	}
	if len(ctx.Path) > 0 {
		lines = append(lines, "- 个性化路径依据："+pathSummary(ctx.Path))
	}
	lines = append(lines, "- 生成边界：未被上述课程资料或知识图谱支持的具体数字、论文、链接与结论，应在使用前再次核验。")
	return strings.TrimSpace(content) + "\n" + strings.Join(lines, "\n")
}

// writerAgentName maps a resource type to the multi-agent writer responsible for
// it, used to surface which agent is currently producing content.
func writerAgentName(resourceType string) string {
	switch resourceType {
	case "mind_map":
		return "MindMapAgent"
	case "practice_questions":
		return "QuestionAgent"
	case "coding_case":
		return "CaseAgent"
	case "reading_material":
		return "ReadingAgent"
	case "video_script":
		return "MultimodalAgent"
	case "generated_image":
		return "ImageGenerationAgent"
	case "generated_video":
		return "VideoGenerationAgent"
	default:
		return "ContentWriterAgent"
	}
}

func buildResourceContent(resourceType string, ctx generationContext) string {
	switch resourceType {
	case "mind_map":
		return buildMindMap(ctx)
	case "practice_questions":
		return buildPracticeQuestions(ctx)
	case "reading_material":
		return buildReadingMaterial(ctx)
	case "coding_case":
		return buildCodingCase(ctx)
	case "video_script":
		return buildVideoScript(ctx)
	default:
		return buildExplanationDoc(ctx)
	}
}

func buildExplanationDoc(ctx generationContext) string {
	summary := firstNonEmpty(detailText(ctx.KnowledgeDetail), "先明确概念，再结合例子和练习巩固。")
	return fmt.Sprintf(`# %s 个性化讲解

## 学习目标

围绕「%s」建立可迁移的理解，并服务于你的目标：%s。

## 概念解释

%s

## 个性化提示

- 知识基础：%s
- 资源偏好：%s
- 建议节奏：先读概念，再看导图，最后完成练习和实操案例。

## 下一步

完成本文档后，优先学习推荐路径中的：%s。
`, ctx.KnowledgeName, ctx.KnowledgeName, ctx.Target, summary, dim(ctx, "knowledge_foundation", "暂无画像记录"), dim(ctx, "resource_preference", "讲解文档、导图、练习题"), pathSummary(ctx.Path))
}

func buildMindMap(ctx generationContext) string {
	return fmt.Sprintf(`mindmap
  root((%s))
    核心概念
      定义
      关键术语
      典型误区
    先修基础
      %s
    学习资源
      讲解文档
      练习题
      实操案例
    个性化目标
      %s
`, ctx.KnowledgeName, prerequisiteSummary(ctx), ctx.Target)
}

func buildPracticeQuestions(ctx generationContext) string {
	return fmt.Sprintf(`# %s 个性化练习题

## 单选题

1. 学习「%s」时，最应该先确认的是哪一项？
   - A. 是否理解核心定义和先修基础
   - B. 是否可以跳过所有背景知识
   - C. 是否只记住名词即可
   - D. 是否完全不需要练习

答案：A

解析：当前画像显示「%s」，因此需要先确认概念和先修链。

## 简答题

请用自己的话说明「%s」和你的学习目标「%s」之间的关系。

## 反思题

完成练习后记录一个仍然不清楚的问题，系统后续会据此更新易错点画像。
`, ctx.KnowledgeName, ctx.KnowledgeName, dim(ctx, "knowledge_foundation", "基础信息不足"), ctx.KnowledgeName, ctx.Target)
}

func buildReadingMaterial(ctx generationContext) string {
	return fmt.Sprintf(`# %s 拓展阅读

## 阅读顺序

1. 先阅读课程资料中关于「%s」的定义和例题。
2. 再查看与它相关的先修知识：%s。
3. 最后结合推荐路径继续阅读：%s。

## 阅读重点

- 关注概念边界，避免把相近知识点混在一起。
- 每读完一段，用一句话写下它对「%s」的作用。
- 如果遇到陌生术语，先回到知识图谱查看前置链。

## 推荐原因

%s
`, ctx.KnowledgeName, ctx.KnowledgeName, prerequisiteSummary(ctx), pathSummary(ctx.Path), ctx.Target, buildResourceReason("reading_material", ctx))
}

func buildCodingCase(ctx generationContext) string {
	return fmt.Sprintf(`# %s 实操案例

## 案例目标

用一个最小实验把「%s」从概念转成可观察结果。

## 实验步骤

1. 准备一个小样例或伪数据。
2. 写出输入、处理过程和输出。
3. 标出每一步对应的知识点。
4. 运行或手工推演一次结果。

## Python 伪代码

`+"```python"+`
def demo():
    concept = "%s"
    goal = "%s"
    print(f"Use {concept} to support {goal}")

demo()
`+"```"+`

## 思考题

如果实验结果和预期不同，你会优先检查概念理解、数据输入还是代码实现？为什么？
`, ctx.KnowledgeName, ctx.KnowledgeName, ctx.KnowledgeName, ctx.Target)
}

func buildVideoScript(ctx generationContext) string {
	return fmt.Sprintf(`# %s 短视频讲解脚本

## 0-20 秒：问题引入

用一个真实学习场景说明为什么要学习「%s」。

## 20-60 秒：核心解释

用图解方式展示「定义 -> 先修 -> 例子 -> 练习」。

## 60-90 秒：个性化提醒

结合画像提示：%s

## 90-120 秒：行动建议

完成一题练习，并打开知识图谱查看下一步：%s。
`, ctx.KnowledgeName, ctx.KnowledgeName, dim(ctx, "error_preference", "先通过练习定位易错点"), pathSummary(ctx.Path))
}

func buildMediaPrompt(kind string, ctx generationContext, customPrompt string) string {
	var builder strings.Builder
	if kind == "video" {
		builder.WriteString("请调用真实文生视频模型生成一段高校课程学习短视频。")
		builder.WriteString("画面应突出知识点关系、学习步骤和可视化类比，节奏适合 6-8 秒课堂微课预览；避免水印、品牌标识和密集小字。")
	} else {
		builder.WriteString("请调用真实文生图模型生成一张高校课程学习配图。")
		builder.WriteString("画面应清晰表达核心概念、先修关系和学习路径，适合放入个性化学习资源卡片；避免密集文字和不可读标签。")
	}
	builder.WriteString("\n\n学习上下文：\n")
	builder.WriteString(resourceContextBlock(ctx))
	if trimmed := strings.TrimSpace(customPrompt); trimmed != "" {
		builder.WriteString("\n\n学生补充创意要求：")
		builder.WriteString(trimmed)
	}
	builder.WriteString("\n\n风格：现代高校 AI 课程教学视觉，干净、可信、信息层级清楚。")
	return builder.String()
}

func defaultMediaSize(kind string) string {
	if kind == "video" {
		return "1280x720"
	}
	return "1024x1024"
}

func mediaResourceType(kind string) string {
	if kind == "video" {
		return "generated_video"
	}
	return "generated_image"
}

func mediaTypeLabel(kind string) string {
	if kind == "video" {
		return "AI 生成短视频"
	}
	return "AI 生成配图"
}

func mediaAgentKey(kind string) string {
	if kind == "video" {
		return "video_generation_agent"
	}
	return "image_generation_agent"
}

func mediaAgentName(kind string) string {
	if kind == "video" {
		return "VideoGenerationAgent"
	}
	return "ImageGenerationAgent"
}

func mediaAgentRole(kind string) string {
	if kind == "video" {
		return "真实文生视频模型调用智能体"
	}
	return "真实文生图模型调用智能体"
}

func mediaStepSummary(kind string, media ai.MediaResponse) string {
	target := "媒体"
	if kind == "video" {
		target = "短视频"
	} else {
		target = "教学配图"
	}
	status := firstNonEmpty(media.Status, "ready")
	if media.URL != "" || media.B64JSON != "" {
		return fmt.Sprintf("已通过 %s / %s 生成%s，状态：%s。", firstNonEmpty(media.Provider, "媒体模型"), firstNonEmpty(media.Model, "未命名模型"), target, status)
	}
	return fmt.Sprintf("已向 %s / %s 提交%s生成任务，状态：%s，任务号：%s。", firstNonEmpty(media.Provider, "媒体模型"), firstNonEmpty(media.Model, "未命名模型"), target, status, firstNonEmpty(media.ProviderTaskID, "未返回"))
}

func mediaResourceReason(kind string, media ai.MediaResponse) string {
	return fmt.Sprintf("由 %s 调用真实%s模型 %s 生成，作为个性化多模态学习资源。",
		mediaAgentName(kind),
		map[bool]string{true: "文生视频", false: "文生图"}[kind == "video"],
		firstNonEmpty(media.Model, "未命名模型"),
	)
}

func mediaEventType(kind string) string {
	if kind == "video" {
		return "video_generated"
	}
	return "image_generated"
}

func marshalMediaContent(kind string, media ai.MediaResponse, prompt string) string {
	payload := map[string]any{
		"kind":           kind,
		"url":            media.URL,
		"b64Json":        media.B64JSON,
		"mimeType":       media.MimeType,
		"prompt":         prompt,
		"provider":       media.Provider,
		"model":          media.Model,
		"status":         firstNonEmpty(media.Status, "ready"),
		"providerTaskId": media.ProviderTaskID,
		"generatedAt":    time.Now().Format(time.RFC3339),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

type mediaContentPayload struct {
	Kind           string `json:"kind"`
	URL            string `json:"url"`
	B64JSON        string `json:"b64Json"`
	MimeType       string `json:"mimeType"`
	Prompt         string `json:"prompt"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	Status         string `json:"status"`
	ProviderTaskID string `json:"providerTaskId"`
	GeneratedAt    string `json:"generatedAt"`
}

func parseMediaContent(content string) mediaContentPayload {
	var payload mediaContentPayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &payload); err != nil {
		return mediaContentPayload{URL: strings.TrimSpace(content)}
	}
	return payload
}

func buildMediaStatus(resource resourcegenentity.GeneratedResource) resourcegendto.MediaStatusResponse {
	payload := parseMediaContent(resource.Content)
	kind := firstNonEmpty(payload.Kind, resource.Format)
	if kind == "" && resource.ResourceType == "generated_video" {
		kind = "video"
	} else if kind == "" {
		kind = "image"
	}
	ready := payload.URL != "" || payload.B64JSON != ""
	status := firstNonEmpty(payload.Status, map[bool]string{true: "ready", false: "pending"}[ready])
	hint := "媒体结果已可预览。"
	if !ready && payload.ProviderTaskID != "" {
		hint = "提供商已返回异步任务号，可在模型控制台或后续状态回调中继续跟踪。"
	} else if !ready {
		hint = "当前资源尚未包含可播放/可预览地址，建议稍后刷新或重新生成。"
	}
	return resourcegendto.MediaStatusResponse{
		ResourceID:     resource.ID,
		Kind:           kind,
		Status:         status,
		Ready:          ready,
		URL:            payload.URL,
		B64JSON:        payload.B64JSON,
		MimeType:       payload.MimeType,
		Prompt:         payload.Prompt,
		Provider:       payload.Provider,
		Model:          payload.Model,
		ProviderTaskID: payload.ProviderTaskID,
		Hint:           hint,
		CheckedAt:      time.Now().Format(time.RFC3339),
	}
}

func normalizeResourceTypes(input []string) []string {
	defaults := []string{"explanation_doc", "mind_map", "practice_questions", "reading_material", "coding_case"}
	allowed := map[string]bool{
		"explanation_doc":    true,
		"mind_map":           true,
		"practice_questions": true,
		"reading_material":   true,
		"coding_case":        true,
		"video_script":       true,
	}
	if len(input) == 0 {
		return defaults
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(input))
	for _, item := range input {
		key := strings.TrimSpace(item)
		if !allowed[key] {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	if len(result) == 0 {
		return defaults
	}
	sort.Strings(result)
	return result
}

func inferResourceTypes(resources []resourcegenentity.GeneratedResource) []string {
	if len(resources) == 0 {
		return normalizeResourceTypes(nil)
	}
	types := make([]string, 0, len(resources))
	for _, resource := range resources {
		switch resource.ResourceType {
		case "generated_image", "generated_video":
			continue
		default:
			types = append(types, resource.ResourceType)
		}
	}
	return normalizeResourceTypes(types)
}

func normalizeDifficulty(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "easy", "简单", "偏易":
		return "easy"
	case "hard", "困难", "偏难":
		return "hard"
	case "fit", "适中", "合适":
		return "fit"
	default:
		return "fit"
	}
}

func normalizeLearningStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pending", "未开始", "待学习":
		return "pending"
	case "in_progress", "learning", "学习中", "进行中":
		return "in_progress"
	case "completed", "done", "已完成", "完成":
		return "completed"
	case "needs_review", "review", "需复习", "需调整":
		return "needs_review"
	default:
		return ""
	}
}

func normalizeResourceSort(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "oldest", "title":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "latest"
	}
}

func buildNextAction(resource *resourcegenentity.GeneratedResource, request resourcegendto.SubmitFeedbackRequest) string {
	difficulty := normalizeDifficulty(request.Difficulty)
	switch {
	case request.Rating <= 2 || !request.Helpful:
		return "建议重新生成更基础的讲解文档和知识导图，并补充关键概念例子。"
	case difficulty == "hard":
		return "建议追加先修知识导图与低门槛练习题，先降低理解难度。"
	case request.Completed:
		return "建议进入下一知识点，并生成代码案例或拓展阅读巩固迁移能力。"
	default:
		return "建议继续完成配套练习，并记录新的疑问用于下一轮资源推荐。"
	}
}

func pushStatusFromFeedback(request resourcegendto.SubmitFeedbackRequest) string {
	difficulty := normalizeDifficulty(request.Difficulty)
	switch {
	case request.Completed:
		return "completed"
	case request.Rating <= 2 || !request.Helpful || difficulty == "hard":
		return "needs_review"
	default:
		return "in_progress"
	}
}

func buildFeedbackSummary(feedback []resourcegenentity.ResourceFeedback) resourcegendto.FeedbackSummaryView {
	summary := resourcegendto.FeedbackSummaryView{
		Difficulty: map[string]int{"easy": 0, "fit": 0, "hard": 0},
		Insight:    "暂无反馈记录，建议先完成一次资源学习并提交评分。",
	}
	if len(feedback) == 0 {
		return summary
	}

	totalRating := 0
	for _, item := range feedback {
		summary.Total++
		totalRating += item.Rating
		if item.Helpful {
			summary.HelpfulCount++
		}
		if item.Completed {
			summary.CompletedCount++
		}
		difficulty := normalizeDifficulty(item.Difficulty)
		summary.Difficulty[difficulty]++
	}
	avg := float64(totalRating) / float64(summary.Total)
	summary.AverageRating = float64(int(avg*10+0.5)) / 10

	switch {
	case summary.AverageRating < 3:
		summary.Insight = "近期资源评分偏低，下一轮应降低难度并增加解释型资源。"
	case summary.Difficulty["hard"] > summary.Difficulty["fit"]:
		summary.Insight = "近期反馈显示资源偏难，建议补充先修链和分步练习。"
	case summary.CompletedCount >= summary.Total && summary.Total > 0:
		summary.Insight = "近期资源完成度较好，可以推进到项目实操或下一知识点。"
	default:
		summary.Insight = "近期反馈整体稳定，建议保持讲解、练习和实操交替推进。"
	}
	return summary
}

func buildRecommendations(profile *profiledto.LearningProfileView, resources []resourcegenentity.GeneratedResource, summary resourcegendto.FeedbackSummaryView) []resourcegendto.ResourceRecommendationView {
	dims := dimensionsMap(profile)
	knowledgeName, knowledgeID, courseTitle := latestResourceTarget(resources)
	target := firstNonEmpty(dims["learning_goal"], "围绕当前知识点完成理解、练习和项目迁移")

	recommendations := []resourcegendto.ResourceRecommendationView{
		{
			ID:            "foundation-repair",
			Title:         "先修补强资源包",
			KnowledgeID:   knowledgeID,
			KnowledgeName: knowledgeName,
			CourseTitle:   courseTitle,
			Target:        target,
			ResourceTypes: []string{
				"explanation_doc",
				"mind_map",
				"practice_questions",
			},
			Reason:     foundationRecommendationReason(dims, summary),
			ActionText: "生成补强包",
			Priority:   95,
		},
		{
			ID:            "project-transfer",
			Title:         "项目迁移实操包",
			KnowledgeID:   knowledgeID,
			KnowledgeName: knowledgeName,
			CourseTitle:   courseTitle,
			Target:        target,
			ResourceTypes: []string{
				"coding_case",
				"reading_material",
			},
			Reason:     "结合学习目标与资源偏好，优先把知识点转化为可展示的实操案例。",
			ActionText: "生成实操包",
			Priority:   82,
		},
		{
			ID:            "review-loop",
			Title:         "多模态复盘资源包",
			KnowledgeID:   knowledgeID,
			KnowledgeName: knowledgeName,
			CourseTitle:   courseTitle,
			Target:        target,
			ResourceTypes: []string{
				"video_script",
				"practice_questions",
			},
			Reason:     reviewRecommendationReason(dims, summary),
			ActionText: "生成复盘包",
			Priority:   76,
		},
	}

	sort.SliceStable(recommendations, func(i, j int) bool {
		return recommendations[i].Priority > recommendations[j].Priority
	})
	return recommendations
}

func latestResourceTarget(resources []resourcegenentity.GeneratedResource) (string, string, string) {
	if len(resources) == 0 {
		return "智能体与工具调用", "", "人工智能导论"
	}
	latest := resources[0]
	return firstNonEmpty(latest.KnowledgeName, "智能体与工具调用"), latest.KnowledgeID, firstNonEmpty(latest.CourseTitle, "人工智能导论")
}

func foundationRecommendationReason(dims map[string]string, summary resourcegendto.FeedbackSummaryView) string {
	foundation := dims["knowledge_foundation"]
	switch {
	case strings.Contains(foundation, "不稳") || strings.Contains(foundation, "偏弱"):
		return "画像显示基础仍需补强，建议先用讲解文档、导图和练习题建立稳定先修链。"
	case summary.Difficulty["hard"] > 0:
		return "近期反馈存在偏难记录，建议追加低门槛讲解和分步练习降低理解成本。"
	default:
		return "用于在进入实操前快速校准概念、先修和易错点。"
	}
}

func reviewRecommendationReason(dims map[string]string, summary resourcegendto.FeedbackSummaryView) string {
	preference := dims["resource_preference"]
	switch {
	case strings.Contains(preference, "视频") || strings.Contains(preference, "导图"):
		return "画像显示偏好多模态材料，适合用短视频脚本和练习题做复盘。"
	case summary.AverageRating > 0 && summary.AverageRating < 3.5:
		return "近期评分仍有提升空间，建议用更短、更聚焦的复盘材料重新巩固。"
	default:
		return "用于把本轮学习内容压缩成可讲解、可自测的复盘材料。"
	}
}

func buildRecommendationSummary(profile *profiledto.LearningProfileView, resources []resourcegenentity.GeneratedResource, summary resourcegendto.FeedbackSummaryView) string {
	if profile == nil && len(resources) == 0 {
		return "建议先生成学习画像和第一批资源，系统会据此形成更精准的下一步推荐。"
	}
	if summary.Total == 0 {
		return "已根据当前画像和资源库给出下一步建议；提交学习反馈后会进一步调整推荐顺序。"
	}
	return summary.Insight
}

func latestFeedbackForResource(feedback []resourcegenentity.ResourceFeedback, resourceID int64) *resourcegenentity.ResourceFeedback {
	for index := range feedback {
		if feedback[index].ResourceID == resourceID {
			return &feedback[index]
		}
	}
	return nil
}

func buildQualityReport(resource resourcegenentity.GeneratedResource, feedback *resourcegenentity.ResourceFeedback) resourcegendto.ResourceQualityReportView {
	safety := buildSafetyReport(resource)
	content := strings.TrimSpace(resource.Content)
	contentLen := len([]rune(content))
	personalizationScore := 60
	if strings.TrimSpace(resource.Reason) != "" {
		personalizationScore += 25
	}
	if strings.TrimSpace(resource.KnowledgeName) != "" || strings.TrimSpace(resource.KnowledgeID) != "" {
		personalizationScore += 10
	}

	structureScore := 45
	switch resource.Format {
	case "mermaid":
		if strings.Contains(content, "mindmap") || strings.Contains(content, "graph") {
			structureScore = 92
		}
	case "image", "video":
		status := buildMediaStatus(resource)
		if status.Ready {
			structureScore = 94
		} else if status.ProviderTaskID != "" {
			structureScore = 76
		}
	default:
		switch {
		case contentLen >= 1200:
			structureScore = 92
		case contentLen >= 500:
			structureScore = 82
		case contentLen >= 160:
			structureScore = 68
		}
		if strings.Contains(content, "#") {
			structureScore += 5
		}
	}

	actionScore := 58
	switch resource.ResourceType {
	case "practice_questions":
		if containsAny(content, "答案", "解析", "易错") {
			actionScore = 90
		}
	case "coding_case":
		if strings.Contains(content, "```") {
			actionScore = 94
		}
	case "mind_map":
		actionScore = 86
	case "generated_image", "generated_video":
		if buildMediaStatus(resource).Ready {
			actionScore = 88
		}
	case "video_script":
		if containsAny(content, "镜头", "旁白", "时长") {
			actionScore = 90
		}
	default:
		if containsAny(content, "步骤", "练习", "建议", "总结") {
			actionScore = 84
		}
	}

	feedbackScore := 70
	if feedback != nil {
		feedbackScore = feedback.Rating * 18
		if feedback.Helpful {
			feedbackScore += 8
		}
		if feedback.Completed {
			feedbackScore += 6
		}
		if normalizeDifficulty(feedback.Difficulty) == "hard" {
			feedbackScore -= 8
		}
	}

	metrics := []resourcegendto.QualityMetricView{
		qualityMetric("personalization", "个性化依据", personalizationScore, "是否绑定知识点、课程目标和生成理由。"),
		qualityMetric("structure", "结构完整性", structureScore, "是否具备可阅读、可预览或可直接展示的结构。"),
		qualityMetric("actionability", "学习可执行性", actionScore, "是否包含练习、步骤、代码、分镜或可操作材料。"),
		qualityMetric("safety", "安全可信", safety.Score, "复用 SafetyReviewerAgent 的事实性、格式和敏感内容审查。"),
		qualityMetric("feedback", "学习反馈", feedbackScore, "结合学生评分、有帮助和完成状态判断实际学习效果。"),
	}

	score := 0
	weights := []int{22, 24, 22, 22, 10}
	for index, metric := range metrics {
		score += metric.Score * weights[index]
	}
	score = clampScore(score / 100)
	strengths, improvements := qualityNarratives(metrics, feedback)
	return resourcegendto.ResourceQualityReportView{
		ResourceID:   resource.ID,
		Title:        resource.Title,
		TypeLabel:    firstNonEmpty(resource.TypeLabel, resourceTypeLabel(resource.ResourceType)),
		Score:        score,
		Level:        qualityLevel(score),
		Metrics:      metrics,
		Strengths:    strengths,
		Improvements: improvements,
		GeneratedAt:  time.Now().Format(time.RFC3339),
	}
}

func qualityMetric(key string, label string, score int, reason string) resourcegendto.QualityMetricView {
	return resourcegendto.QualityMetricView{Key: key, Label: label, Score: clampScore(score), Reason: reason}
}

func clampScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func qualityLevel(score int) string {
	switch {
	case score >= 90:
		return "excellent"
	case score >= 78:
		return "good"
	case score >= 62:
		return "usable"
	default:
		return "needs_improvement"
	}
}

func qualityNarratives(metrics []resourcegendto.QualityMetricView, feedback *resourcegenentity.ResourceFeedback) ([]string, []string) {
	strengths := make([]string, 0, 3)
	improvements := make([]string, 0, 3)
	for _, metric := range metrics {
		if metric.Score >= 85 {
			strengths = append(strengths, metric.Label+"表现较好，可作为答辩展示证据。")
		} else if metric.Score < 70 {
			improvements = append(improvements, metric.Label+"仍需加强，建议重新生成或人工补充。")
		}
	}
	if feedback == nil {
		improvements = append(improvements, "尚缺少学生反馈，建议完成学习后提交评分以校准质量。")
	}
	if len(strengths) == 0 {
		strengths = append(strengths, "资源已进入生成库，可继续结合安全审查和反馈闭环优化。")
	}
	if len(improvements) == 0 {
		improvements = append(improvements, "当前质量稳定，后续可补充来源标注和课堂实测数据。")
	}
	return strengths, improvements
}

func comparisonReason(report resourcegendto.ResourceQualityReportView) string {
	switch report.Level {
	case "excellent":
		return "综合质量优秀，适合作为本轮学习或比赛展示的优先资源。"
	case "good":
		return "结构与可信度较好，适合继续学习并收集反馈。"
	case "usable":
		return "资源可用，但建议结合改进项补强后再作为核心材料。"
	default:
		return "质量分偏低，建议重新生成或人工修订后使用。"
	}
}

func buildComparisonSummary(items []resourcegendto.ResourceComparisonItemView) string {
	if len(items) == 0 {
		return "暂无可对比资源。"
	}
	winner := items[0]
	if len(items) == 1 {
		return "当前资源质量为 " + strconv.Itoa(winner.Quality.Score) + " 分。"
	}
	second := items[1]
	gap := winner.Quality.Score - second.Quality.Score
	if gap <= 3 {
		return fmt.Sprintf("资源质量接近，优先选择《%s》，同时可吸收第二名资源的补充内容。", winner.Resource.Title)
	}
	return fmt.Sprintf("建议优先使用《%s》，综合质量分 %d，比第二名高 %d 分。", winner.Resource.Title, winner.Quality.Score, gap)
}

func (s *Service) getJobParts(ctx context.Context, studentID int64, rawJobID string) (*resourcegenentity.GenerationJob, []resourcegenentity.AgentStep, []resourcegenentity.GeneratedResource, error) {
	jobID, err := strconv.ParseInt(strings.TrimSpace(rawJobID), 10, 64)
	if err != nil || jobID <= 0 {
		return nil, nil, nil, apperr.ErrBadRequest
	}
	job, steps, resources, err := s.repo.GetJob(ctx, studentID, jobID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, nil, nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, nil, nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	return job, steps, resources, nil
}

func (s *Service) responseForGeneratedResource(ctx context.Context, studentID int64, jobID int64) (*resourcegendto.GenerateResourcesResponse, error) {
	job, steps, resources, err := s.repo.GetJob(ctx, studentID, jobID)
	if errors.Is(err, resourcegenrepo.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	view := jobToView(job, steps, resources)
	return &resourcegendto.GenerateResourcesResponse{Job: view, Resources: view.Resources}, nil
}

func buildJobReviewSummary(job *resourcegenentity.GenerationJob, resources []resourcegenentity.GeneratedResource, feedback []resourcegenentity.ResourceFeedback) resourcegendto.JobReviewSummaryResponse {
	items := make([]resourcegendto.JobReviewResourceView, 0, len(resources))
	risks := make([]string, 0)
	suggestions := make([]string, 0)
	totalQuality := 0
	totalSafety := 0
	mediaReady := 0
	mediaPending := 0
	for _, resource := range resources {
		safety := buildSafetyReport(resource)
		quality := buildQualityReport(resource, latestFeedbackForResource(feedback, resource.ID))
		var media *resourcegendto.MediaStatusResponse
		if isMediaResourceEntity(resource) {
			status := buildMediaStatus(resource)
			media = &status
			if status.Ready {
				mediaReady++
			} else {
				mediaPending++
			}
		}
		totalQuality += quality.Score
		totalSafety += safety.Score
		risks = append(risks, prefixStrings(resource.Title, safety.Risks)...)
		suggestions = append(suggestions, prefixStrings(resource.Title, quality.Improvements)...)
		items = append(items, resourcegendto.JobReviewResourceView{
			Resource: resourceToView(resource),
			Safety:   safety,
			Quality:  quality,
			Media:    media,
		})
	}
	avgQuality := 0
	avgSafety := 0
	if len(resources) > 0 {
		avgQuality = totalQuality / len(resources)
		avgSafety = totalSafety / len(resources)
	}
	return resourcegendto.JobReviewSummaryResponse{
		JobID:             job.ID,
		Summary:           buildJobReviewSummaryText(job, len(resources), avgQuality, avgSafety, mediaReady, mediaPending),
		ResourceCount:     len(resources),
		AverageQuality:    avgQuality,
		AverageSafety:     avgSafety,
		MediaReadyCount:   mediaReady,
		MediaPendingCount: mediaPending,
		Risks:             limitReviewStrings(risks, 6),
		Suggestions:       limitReviewStrings(suggestions, 6),
		Resources:         items,
		GeneratedAt:       time.Now().Format(time.RFC3339),
	}
}

func buildJobReviewSummaryText(job *resourcegenentity.GenerationJob, count int, quality int, safety int, mediaReady int, mediaPending int) string {
	if count == 0 {
		return "当前任务尚未产出资源，建议先完成生成后再进行评审。"
	}
	base := fmt.Sprintf("任务 #%d 已产出 %d 份资源，平均质量 %d 分，平均安全 %d 分。", job.ID, count, quality, safety)
	if mediaReady+mediaPending > 0 {
		base += fmt.Sprintf(" 多模态资源中 %d 项已就绪、%d 项仍需跟踪。", mediaReady, mediaPending)
	}
	return base
}

func buildEvidencePackage(job *resourcegenentity.GenerationJob, steps []resourcegenentity.AgentStep, resources []resourcegenentity.GeneratedResource, feedback []resourcegenentity.ResourceFeedback) resourcegendto.JobEvidencePackageResponse {
	items := make([]resourcegendto.EvidencePackageResourceView, 0, len(resources))
	for _, resource := range resources {
		safety := buildSafetyReport(resource)
		quality := buildQualityReport(resource, latestFeedbackForResource(feedback, resource.ID))
		var media *resourcegendto.MediaStatusResponse
		if isMediaResourceEntity(resource) {
			status := buildMediaStatus(resource)
			media = &status
		}
		items = append(items, resourcegendto.EvidencePackageResourceView{
			Resource: resourceToView(resource),
			Safety:   safety,
			Quality:  quality,
			Media:    media,
		})
	}
	return resourcegendto.JobEvidencePackageResponse{
		JobID:       job.ID,
		FileName:    fmt.Sprintf("resource-evidence-package-%d.json", job.ID),
		Job:         jobToView(job, steps, resources),
		Steps:       stepsToViews(steps),
		Resources:   items,
		Checklist:   evidenceChecklist(resources),
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
}

func buildStudyChecklist(job *resourcegenentity.GenerationJob, resources []resourcegenentity.GeneratedResource) resourcegendto.StudyChecklistResponse {
	items := make([]resourcegendto.StudyChecklistItemView, 0, len(resources)+2)
	ordered := append([]resourcegenentity.GeneratedResource(nil), resources...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return checklistPriority(ordered[i].ResourceType) > checklistPriority(ordered[j].ResourceType)
	})
	index := 1
	for _, resource := range ordered {
		items = append(items, resourcegendto.StudyChecklistItemView{
			Index:      index,
			ResourceID: resource.ID,
			Title:      resource.Title,
			Action:     checklistAction(resource.ResourceType),
			Expected:   checklistExpected(resource.ResourceType),
			Status:     "pending",
		})
		index++
	}
	items = append(items, resourcegendto.StudyChecklistItemView{
		Index:    index,
		Title:    "提交学习反馈",
		Action:   "完成本轮资源学习后，为每个资源提交评分、难度和备注。",
		Expected: "系统据此刷新画像、调整推送流和下一步推荐。",
		Status:   "pending",
	})
	return resourcegendto.StudyChecklistResponse{
		JobID:       job.ID,
		Summary:     fmt.Sprintf("围绕“%s”整理了 %d 个学习动作，按理解、练习、迁移、复盘顺序推进。", firstNonEmpty(job.KnowledgeName, job.KnowledgeID, "当前知识点"), len(items)),
		Items:       items,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
}

func buildProvenance(resource *resourcegenentity.GeneratedResource, job *resourcegenentity.GenerationJob, steps []resourcegenentity.AgentStep, events []resourcegenentity.LearningEvent) resourcegendto.ResourceProvenanceResponse {
	filtered := make([]resourcegenentity.LearningEvent, 0)
	for _, event := range events {
		if event.ResourceID == resource.ID || (event.ResourceID == 0 && event.KnowledgeID != "" && event.KnowledgeID == resource.KnowledgeID) {
			filtered = append(filtered, event)
		}
		if len(filtered) >= 12 {
			break
		}
	}
	return resourcegendto.ResourceProvenanceResponse{
		Resource:    resourceToView(*resource),
		Job:         jobToView(job, steps, []resourcegenentity.GeneratedResource{*resource}),
		Steps:       stepsToViews(steps),
		Events:      learningEventsToViews(filtered),
		Summary:     fmt.Sprintf("资源由任务 #%d 生成，经过 %d 个智能体步骤，并关联 %d 条近期学习行为信号。", job.ID, len(steps), len(filtered)),
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
}

func isMediaResourceEntity(resource resourcegenentity.GeneratedResource) bool {
	return resource.Format == "image" || resource.Format == "video" || resource.ResourceType == "generated_image" || resource.ResourceType == "generated_video"
}

func prefixStrings(prefix string, values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		result = append(result, "《"+firstNonEmpty(prefix, "未命名资源")+"》"+value)
	}
	return result
}

func limitReviewStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func evidenceChecklist(resources []resourcegenentity.GeneratedResource) []string {
	checklist := []string{
		"确认资源正文、生成理由、质量评分和安全审查均已随包导出。",
		"展示或提交前由团队成员人工复核关键事实和引用来源。",
		"若包含外部材料、开源项目或第三方模型输出，应补充来源、协议和 AI 工具说明。",
	}
	if hasMediaResource(resources) {
		checklist = append(checklist, "对图片/视频资源核验媒体 URL、任务号和可预览状态。")
	}
	return checklist
}

func hasMediaResource(resources []resourcegenentity.GeneratedResource) bool {
	for _, resource := range resources {
		if isMediaResourceEntity(resource) {
			return true
		}
	}
	return false
}

func checklistPriority(resourceType string) int {
	switch resourceType {
	case "explanation_doc":
		return 90
	case "mind_map":
		return 85
	case "practice_questions":
		return 75
	case "coding_case":
		return 68
	case "reading_material":
		return 60
	case "video_script", "generated_image", "generated_video":
		return 55
	default:
		return 50
	}
}

func checklistAction(resourceType string) string {
	switch resourceType {
	case "explanation_doc":
		return "先阅读讲解文档，标记不理解的概念和公式。"
	case "mind_map":
		return "对照导图复述知识点、先修链和应用场景。"
	case "practice_questions":
		return "完成练习题并核对解析，把错题写入备注。"
	case "coding_case":
		return "运行或改写代码案例，记录可迁移到项目中的部分。"
	case "reading_material":
		return "阅读拓展材料，挑选 1 个方向继续深入。"
	case "video_script":
		return "按分镜脚本进行口头复述，检查是否能讲清核心流程。"
	case "generated_image":
		return "查看配图并用自己的话解释图中的概念关系。"
	case "generated_video":
		return "播放短视频并记录仍未理解的镜头或概念。"
	default:
		return "学习资源内容并记录关键收获。"
	}
}

func checklistExpected(resourceType string) string {
	switch resourceType {
	case "practice_questions":
		return "暴露易错点，形成下一轮难度调整依据。"
	case "coding_case":
		return "把知识迁移到项目实操，沉淀可展示代码片段。"
	case "generated_image", "generated_video", "video_script":
		return "形成多模态复盘材料，提升演示表达效果。"
	default:
		return "完成理解建模，为练习和项目迁移打基础。"
	}
}

func variantTarget(resource resourcegenentity.GeneratedResource, request resourcegendto.ResourceVariantRequest) string {
	target := strings.TrimSpace(request.Target)
	if target != "" {
		return target
	}
	mode := strings.TrimSpace(request.Mode)
	if mode == "" {
		mode = "simplify"
	}
	switch mode {
	case "deepen":
		return "在原资源基础上生成更深入的进阶版本，补充原理、边界条件和项目应用。"
	case "example":
		return "在原资源基础上生成更多贴近学生专业方向的例子和类比。"
	case "quiz":
		return "在原资源基础上生成可自测的练习题和解析，帮助发现易错点。"
	case "visual":
		return "在原资源基础上生成可视化图示，突出概念关系、流程和关键步骤。"
	default:
		return "在原资源基础上生成更容易理解的版本，降低术语密度并增加分步解释。"
	}
}

func variantResourceType(resourceType string, mode string) string {
	switch mode {
	case "quiz":
		return "practice_questions"
	case "visual":
		return "mind_map"
	case "example":
		if resourceType == "practice_questions" {
			return "explanation_doc"
		}
		return resourceType
	default:
		if resourceType == "generated_image" || resourceType == "generated_video" {
			return "explanation_doc"
		}
		return resourceType
	}
}

func buildSafetyReport(resource resourcegenentity.GeneratedResource) resourcegendto.SafetyReportView {
	content := strings.TrimSpace(resource.Content)
	isMedia := resource.Format == "image" || resource.Format == "video"
	isMindMap := resource.Format == "mermaid" || resource.ResourceType == "mind_map"
	hasGrounding := isMedia || isMindMap || strings.Contains(content, "## 依据与核验")
	structured := content != ""
	if resource.Format == "markdown" || resource.Format == "mermaid" {
		structured = strings.Contains(content, "#") || strings.Contains(content, "mindmap")
	}
	checks := []resourcegendto.SafetyCheckView{
		buildSafetyCheck("factuality", "事实性与防幻觉", !containsAny(content, "保证", "绝对正确", "100%", "唯一答案"), "未发现绝对化或不可验证承诺，建议保留人工复核。"),
		buildSafetyCheck("grounding", "来源可追溯", hasGrounding, "资源包含知识图谱、课程资料或生成模型依据。"),
		buildSafetyCheck("sensitive", "敏感与违规过滤", !containsAny(content, "违法", "攻击", "绕过", "作弊"), "未发现明显敏感违规表达。"),
		buildSafetyCheck("structure", "格式完整性", structured, "资源包含可展示内容，可直接在资源详情页预览。"),
		buildSafetyCheck("pedagogy", "教学适配性", strings.TrimSpace(resource.Reason) != "", "资源保留了生成理由，能够解释个性化依据。"),
		buildSafetyCheck("license", "开源与 AI 工具说明", !containsUnverifiedExternalLink(content), "未发现未标注的外部链接；引用外部材料时仍需确认协议。"),
	}

	risks := make([]string, 0, 3)
	suggestions := []string{
		"展示或提交前建议由团队成员进行一次人工事实核对。",
		"若引用外部教材、论文或开源项目，应在资源中补充来源与协议说明。",
	}
	score := 100
	for _, check := range checks {
		switch check.Status {
		case "blocked":
			score -= 35
			risks = append(risks, check.Name+"未通过："+check.Conclusion)
		case "warning":
			score -= 12
			risks = append(risks, check.Name+"需关注："+check.Conclusion)
		}
	}
	if resource.ResourceType == "coding_case" && !strings.Contains(content, "```") {
		score -= 10
		risks = append(risks, "代码案例未检测到代码块，建议补充可复制的示例。")
	}
	if score < 0 {
		score = 0
	}

	status := "passed"
	if len(risks) > 0 {
		status = "attention_required"
	}
	if score < 60 {
		status = "blocked"
	}

	return resourcegendto.SafetyReportView{
		ResourceID:   resource.ID,
		Title:        resource.Title,
		Status:       status,
		Score:        score,
		Checks:       checks,
		Risks:        risks,
		Suggestions:  suggestions,
		ReviewedAt:   time.Now().Format(time.RFC3339),
		ReviewerName: "SafetyReviewerAgent",
	}
}

func containsUnverifiedExternalLink(content string) bool {
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "http://") && !strings.Contains(lower, "https://") {
		return false
	}
	return !strings.Contains(content, "来源") && !strings.Contains(content, "参考") && !strings.Contains(content, "依据")
}

func buildSafetyCheck(key string, name string, passed bool, conclusion string) resourcegendto.SafetyCheckView {
	status := "passed"
	if !passed {
		status = "warning"
	}
	return resourcegendto.SafetyCheckView{
		Key:        key,
		Name:       name,
		Status:     status,
		Conclusion: conclusion,
	}
}

func buildJobMarkdown(job *resourcegenentity.GenerationJob, steps []resourcegenentity.AgentStep, resources []resourcegenentity.GeneratedResource) string {
	var builder strings.Builder
	builder.WriteString("# 个性化资源包：" + firstNonEmpty(job.KnowledgeName, "未命名知识点") + "\n\n")
	builder.WriteString("> 导出时间：" + time.Now().Format("2006-01-02 15:04:05") + "\n\n")
	builder.WriteString("## 任务信息\n\n")
	builder.WriteString("- 任务 ID：" + strconv.FormatInt(job.ID, 10) + "\n")
	builder.WriteString("- 课程：" + firstNonEmpty(job.CourseTitle, "未填写") + "\n")
	builder.WriteString("- 知识点：" + firstNonEmpty(job.KnowledgeName, job.KnowledgeID, "未填写") + "\n")
	builder.WriteString("- 生成目标：" + firstNonEmpty(job.Target, "未填写") + "\n")
	builder.WriteString("- 状态：" + job.Status + "\n\n")

	builder.WriteString("## 多智能体协作记录\n\n")
	for _, step := range steps {
		builder.WriteString("### " + step.AgentName + "\n\n")
		builder.WriteString("- 角色：" + step.Role + "\n")
		builder.WriteString("- 状态：" + step.Status + "\n")
		builder.WriteString("- 摘要：" + step.Summary + "\n\n")
	}

	builder.WriteString("## 资源目录\n\n")
	for index, resource := range resources {
		builder.WriteString(fmt.Sprintf("%d. %s（%s）\n", index+1, resource.Title, resource.TypeLabel))
	}
	builder.WriteString("\n")

	for _, resource := range resources {
		builder.WriteString("---\n\n")
		builder.WriteString("## " + resource.Title + "\n\n")
		builder.WriteString("- 类型：" + resource.TypeLabel + "\n")
		builder.WriteString("- 格式：" + resource.Format + "\n")
		builder.WriteString("- 推荐原因：" + resource.Reason + "\n\n")
		if resource.Format == "mermaid" {
			builder.WriteString("```mermaid\n")
			builder.WriteString(strings.TrimSpace(resource.Content))
			builder.WriteString("\n```\n\n")
		} else {
			builder.WriteString(strings.TrimSpace(resource.Content))
			builder.WriteString("\n\n")
		}
		report := buildSafetyReport(resource)
		builder.WriteString("### 安全审查摘要\n\n")
		builder.WriteString("- 审查智能体：" + report.ReviewerName + "\n")
		builder.WriteString("- 状态：" + report.Status + "\n")
		builder.WriteString("- 分数：" + strconv.Itoa(report.Score) + "\n")
		builder.WriteString("- 建议：" + strings.Join(report.Suggestions, "；") + "\n\n")
	}
	return builder.String()
}

func buildPresentationOutline(job *resourcegenentity.GenerationJob, steps []resourcegenentity.AgentStep, resources []resourcegenentity.GeneratedResource) resourcegendto.PresentationOutlineResponse {
	knowledgeName := firstNonEmpty(job.KnowledgeName, job.KnowledgeID, "个性化学习资源")
	title := "个性化资源智能体演示：" + knowledgeName
	subtitle := firstNonEmpty(job.CourseTitle, "人工智能课程") + " · " + firstNonEmpty(job.Target, "面向学生个性化学习")
	resourceLabels := resourceTypeNames(resources)
	agentNames := agentStepNames(steps)

	slides := []resourcegendto.PresentationSlideView{
		{
			Index: 1,
			Title: "赛题背景与核心痛点",
			Bullets: []string{
				"高校课程资源丰富但组织方式偏静态，学生难以快速匹配个人基础和目标。",
				"不同学生在知识基础、学习偏好、易错点上差异明显，统一资源包难以满足个性化学习。",
				"本系统围绕“画像识别、资源生成、反馈闭环、安全审查”构建可演示流程。",
			},
			SpeakerNotes: "用 30 秒说明为什么需要个性化资源生成，把痛点落到学生真实学习过程。",
			VisualHint:   "使用问题链或流程箭头：资源多 -> 不匹配 -> 需要智能体协同生成。",
		},
		{
			Index: 2,
			Title: "学生画像与生成目标",
			Bullets: []string{
				"当前生成任务围绕知识点：“" + knowledgeName + "”。",
				"课程场景：" + firstNonEmpty(job.CourseTitle, "未指定课程") + "。",
				"生成目标：" + firstNonEmpty(job.Target, "围绕当前知识点完成理解、练习和迁移") + "。",
			},
			SpeakerNotes: "展示系统如何把学生目标转化为资源生成上下文，突出个性化输入。",
			VisualHint:   "左侧展示学生画像维度，右侧展示资源生成目标。",
		},
		{
			Index: 3,
			Title: "多智能体协同流程",
			Bullets: []string{
				"本次任务调用智能体：" + strings.Join(agentNames, "、") + "。",
				"智能体按“画像分析 -> 课程规划 -> 资源生成 -> 安全审查”的顺序协作。",
				"每一步均保留可解释摘要，便于答辩时展示系统透明度。",
			},
			SpeakerNotes: "把多智能体不是口号，而是有任务步骤、有角色分工、有结果记录讲清楚。",
			VisualHint:   "使用横向流程图或泳道图展示 4 个智能体步骤。",
		},
		{
			Index: 4,
			Title: "生成资源成果",
			Bullets: []string{
				"本次共生成 " + strconv.Itoa(len(resources)) + " 类资源。",
				"资源类型包括：" + strings.Join(resourceLabels, "、") + "。",
				"资源内容覆盖讲解、结构化理解、练习巩固、拓展阅读和实操迁移。",
			},
			SpeakerNotes: "按资源卡片快速展示成果，重点讲不同资源如何服务不同学习环节。",
			VisualHint:   "使用卡片网格展示资源标题，配合一页截图或目录。",
		},
		{
			Index: 5,
			Title: "安全审查与学习反馈闭环",
			Bullets: []string{
				"SafetyReviewerAgent 对资源进行事实性、防幻觉、敏感过滤和格式完整性检查。",
				"学生可提交评分、难度、是否完成和备注，系统据此生成下一步推荐。",
				"导出材料自动附带安全审查摘要，便于提交和人工复核。",
			},
			SpeakerNotes: "强调比赛要求里的内容安全和应用效果，不只生成资源，还能持续优化。",
			VisualHint:   "用闭环图展示：生成 -> 学习 -> 反馈 -> 推荐 -> 再生成。",
		},
		{
			Index: 6,
			Title: "演示路径与创新价值",
			Bullets: []string{
				"演示路径：生成画像 -> 生成资源包 -> 查看智能体步骤 -> 安全审查 -> 提交反馈 -> 下一步推荐 -> 导出材料。",
				"创新点：将多智能体协作、知识图谱上下文和反馈闭环集中到一个学习工作台。",
				"实用价值：降低学生筛选学习资源成本，提升资源适配度和学习过程可追踪性。",
			},
			SpeakerNotes: "最后用一句话收束：系统不是单次生成工具，而是个性化学习资源闭环。",
			VisualHint:   "使用最终演示截图或系统架构图作为收尾。",
		},
	}

	return resourcegendto.PresentationOutlineResponse{
		JobID:       job.ID,
		Title:       title,
		Subtitle:    subtitle,
		Slides:      slides,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
}

func buildDemoScript(job *resourcegenentity.GenerationJob, steps []resourcegenentity.AgentStep, resources []resourcegenentity.GeneratedResource) resourcegendto.DemoScriptResponse {
	knowledgeName := firstNonEmpty(job.KnowledgeName, job.KnowledgeID, "个性化学习资源")
	resourceLabels := resourceTypeNames(resources)
	agentNames := agentStepNames(steps)
	title := "7 分钟演示脚本：" + knowledgeName

	scenes := []resourcegendto.DemoScriptSceneView{
		{
			Index:     1,
			TimeRange: "00:00-00:45",
			Title:     "赛题痛点与系统定位",
			Actions: []string{
				"打开资源中心或作品展示页。",
				"说明系统面向高等教育个性化资源生成场景。",
			},
			Narration: "传统课程资源多但不一定适合每个学生，本系统把学习画像、知识图谱、多智能体资源生成和反馈评估连成闭环。",
			ProofPoints: []string{
				"A3 要求的对话式画像、多资源生成和路径推送均有可操作入口。",
			},
		},
		{
			Index:     2,
			TimeRange: "00:45-01:40",
			Title:     "对话式画像与随学刷新",
			Actions: []string{
				"输入学习情况描述并生成画像。",
				"展示 6 个画像维度和画像更新时间线。",
			},
			Narration: "系统不是让学生填写长表单，而是从自然语言和后续学习行为中持续抽取画像维度。",
			ProofPoints: []string{
				"画像维度包含知识基础、目标、进度、认知风格、易错点和资源偏好。",
				"学习事件会推动画像刷新。",
			},
		},
		{
			Index:     3,
			TimeRange: "01:40-03:00",
			Title:     "多智能体协同生成资源",
			Actions: []string{
				"点击生成资源包。",
				"展示智能体步骤进度条和步骤摘要。",
			},
			Narration: "本次任务调用 " + strings.Join(agentNames, "、") + "，按照画像分析、课程规划、资源生成、安全审查的顺序协作。",
			ProofPoints: []string{
				"资源类型包括：" + strings.Join(resourceLabels, "、") + "。",
				"任务编号：" + strconv.FormatInt(job.ID, 10),
			},
		},
		{
			Index:     4,
			TimeRange: "03:00-04:10",
			Title:     "多模态资源与可信依据",
			Actions: []string{
				"打开一份资源详情页。",
				"展示正文、分镜预览、依据来源和安全审查。",
			},
			Narration: "资源详情页展示生成依据和 SafetyReviewerAgent 审查结果，避免资源看起来像无来源的纯模型输出。",
			ProofPoints: []string{
				"资源详情页包含知识点、课程、智能体、个性化依据和任务编号。",
			},
		},
		{
			Index:     5,
			TimeRange: "04:10-05:25",
			Title:     "资源推送与学习路径联动",
			Actions: []string{
				"展示资源中心右侧的路径资源推送。",
				"切换到仪表盘查看资源学习效果。",
			},
			Narration: "生成结果会进入资源推送流，系统按理解、练习、迁移和复盘阶段给学生安排下一步资源。",
			ProofPoints: []string{
				"资源推送项会随完成反馈变成已完成或需调整。",
			},
		},
		{
			Index:     6,
			TimeRange: "05:25-06:30",
			Title:     "反馈评估与闭环优化",
			Actions: []string{
				"提交资源评分、难度和完成状态。",
				"点击随学刷新，展示画像和推荐变化。",
			},
			Narration: "反馈不是只做记录，而是会影响资源推送状态、下一步推荐和画像维度。",
			ProofPoints: []string{
				"resource_feedbacks、learning_events、resource_push_items 三张表共同支撑闭环。",
			},
		},
		{
			Index:     7,
			TimeRange: "06:30-07:00",
			Title:     "工程可靠性与提交材料",
			Actions: []string{
				"展示 Markdown 导出、PPT 大纲和本演示脚本。",
				"说明测试和构建结果。",
			},
			Narration: "系统保留可运行源码、数据库迁移、构建产物和比赛材料，便于复现和答辩。",
			ProofPoints: []string{
				"go test ./... 通过。",
				"npm run build 与 npm run test:run 通过。",
			},
		},
	}

	return resourcegendto.DemoScriptResponse{
		JobID:    job.ID,
		Title:    title,
		Duration: "7 分钟以内",
		Scenes:   scenes,
		Checklist: []string{
			"提前登录学生账号并准备一条画像描述。",
			"确保已有一个生成完成的资源任务，避免现场等待。",
			"准备展示资源详情页中的依据来源和安全审查。",
			"提交一次反馈后展示资源推送状态和画像刷新。",
			"保留健康检查、测试和构建结果作为工程可靠性证据。",
		},
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
}

func resourceTypeNames(resources []resourcegenentity.GeneratedResource) []string {
	if len(resources) == 0 {
		return []string{"讲解文档", "思维导图", "练习题", "实操案例"}
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(resources))
	for _, resource := range resources {
		name := firstNonEmpty(resource.TypeLabel, resourceTypeLabel(resource.ResourceType))
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func agentStepNames(steps []resourcegenentity.AgentStep) []string {
	if len(steps) == 0 {
		return []string{"ProfileAnalystAgent", "CoursePlannerAgent", "ResourceWriterAgents", "SafetyReviewerAgent"}
	}
	result := make([]string, 0, len(steps))
	for _, step := range steps {
		result = append(result, firstNonEmpty(step.AgentName, step.AgentKey))
	}
	return result
}

func currentStudentID(ctx context.Context) (int64, error) {
	studentID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		return 0, apperr.ErrUnauthorized
	}
	return studentID, nil
}

func dimensionsMap(profile *profiledto.LearningProfileView) map[string]string {
	result := map[string]string{}
	if profile == nil {
		return result
	}
	for _, dimension := range profile.Dimensions {
		result[strings.TrimSpace(dimension.Key)] = strings.TrimSpace(dimension.Value)
	}
	return result
}

func dim(ctx generationContext, key string, fallback string) string {
	value := strings.TrimSpace(ctx.ProfileDims[key])
	if value == "" {
		return fallback
	}
	return value
}

func profileSummary(ctx generationContext) string {
	if ctx.Profile != nil && strings.TrimSpace(ctx.Profile.Summary) != "" {
		return strings.TrimSpace(ctx.Profile.Summary)
	}
	return "当前还没有完整画像，按目标和知识点生成基础资源包"
}

func detailText(detail *graphdto.KnowledgeDetailView) string {
	if detail == nil {
		return ""
	}
	return firstNonEmpty(detail.Summary, detail.Description, detail.RecommendReason)
}

func prerequisiteSummary(ctx generationContext) string {
	if ctx.KnowledgeDetail != nil && len(ctx.KnowledgeDetail.Prerequisites) > 0 {
		return strings.Join(limitStrings(ctx.KnowledgeDetail.Prerequisites, 3), "、")
	}
	return "基础概念、关键定义、相关练习"
}

func pathSummary(path []recommenddto.PathItemView) string {
	if len(path) == 0 {
		return "先完成当前知识点，再查看相邻节点"
	}
	titles := make([]string, 0, minInt(len(path), 3))
	for _, item := range path {
		title := firstNonEmpty(item.Title, item.KnowledgeID)
		if title != "" {
			titles = append(titles, title)
		}
		if len(titles) >= 3 {
			break
		}
	}
	if len(titles) == 0 {
		return "先完成当前知识点，再查看相邻节点"
	}
	return strings.Join(titles, " -> ")
}

func buildResourceReason(resourceType string, ctx generationContext) string {
	switch resourceType {
	case "mind_map":
		return "根据画像中的认知风格和知识基础，用结构图帮助建立全局关系。"
	case "practice_questions":
		return "根据易错点画像和掌握度反馈，用练习题校准学习效果。"
	case "reading_material":
		return "根据学习目标补充拓展材料，帮助从课程内容迁移到项目应用。"
	case "coding_case":
		return "根据资源偏好和项目目标，提供可动手的实操入口。"
	case "video_script":
		return "根据多模态学习需求，生成可用于演示或讲解的短视频脚本。"
	case "generated_image":
		return "根据当前知识点和学习画像调用真实文生图模型，生成教学配图。"
	case "generated_video":
		return "根据当前知识点和学习画像调用真实文生视频模型，生成微课短视频。"
	default:
		return "根据当前画像、知识图谱和推荐路径生成个性化讲解。"
	}
}

func buildPushItem(resource *resourcegenentity.GeneratedResource, ctx generationContext) *resourcegenentity.ResourcePushItem {
	return &resourcegenentity.ResourcePushItem{
		StudentID:     resource.StudentID,
		ResourceID:    resource.ID,
		JobID:         resource.JobID,
		KnowledgeID:   resource.KnowledgeID,
		KnowledgeName: resource.KnowledgeName,
		CourseTitle:   resource.CourseTitle,
		Stage:         resourceStage(resource.ResourceType),
		Reason:        buildPushReason(resource.ResourceType, firstNonEmpty(resource.Reason, buildResourceReason(resource.ResourceType, ctx))),
		Priority:      resourcePriority(resource.ResourceType),
		Status:        "pending",
	}
}

func resourceStage(resourceType string) string {
	switch resourceType {
	case "explanation_doc", "mind_map":
		return "understand"
	case "practice_questions":
		return "practice"
	case "coding_case":
		return "transfer"
	case "video_script", "generated_image", "generated_video":
		return "review"
	default:
		return "extend"
	}
}

func resourcePriority(resourceType string) int {
	switch resourceType {
	case "explanation_doc":
		return 96
	case "mind_map":
		return 92
	case "practice_questions":
		return 88
	case "coding_case":
		return 82
	case "video_script":
		return 76
	case "generated_image":
		return 74
	case "generated_video":
		return 73
	default:
		return 72
	}
}

func buildPushReason(resourceType string, reason string) string {
	switch resourceType {
	case "explanation_doc":
		return firstNonEmpty(reason, "先用讲解文档建立概念和先修基础。")
	case "mind_map":
		return firstNonEmpty(reason, "用导图把知识点、先修链和应用场景串起来。")
	case "practice_questions":
		return firstNonEmpty(reason, "通过练习题校准掌握度，并暴露易错点。")
	case "coding_case":
		return firstNonEmpty(reason, "把知识点迁移到可运行的项目实操。")
	case "video_script":
		return firstNonEmpty(reason, "用分镜和旁白形成多模态复盘材料。")
	case "generated_image":
		return firstNonEmpty(reason, "用真实生成配图辅助理解抽象概念。")
	case "generated_video":
		return firstNonEmpty(reason, "用真实生成短视频形成多模态复盘材料。")
	default:
		return firstNonEmpty(reason, "作为当前学习路径上的个性化补充资源。")
	}
}

func buildFeedSummary(items []resourcegendto.ResourcePushItemView, events []resourcegenentity.LearningEvent) string {
	if len(items) == 0 {
		return "还没有可推送资源。先生成一次资源包，系统会把讲解、导图、练习和案例绑定到学习路径。"
	}
	pending := 0
	for _, item := range items {
		if item.Status == "pending" {
			pending++
		}
	}
	if len(events) > 0 {
		return fmt.Sprintf("已根据最近 %d 条学习行为整理 %d 份路径资源，其中 %d 份建议优先学习。", len(events), len(items), pending)
	}
	return fmt.Sprintf("已整理 %d 份路径资源，其中 %d 份建议优先学习。提交反馈后推送顺序会继续调整。", len(items), pending)
}

func pushItemToView(item resourcegenentity.ResourcePushItem, resource resourcegenentity.GeneratedResource) resourcegendto.ResourcePushItemView {
	status := firstNonEmpty(item.Status, "pending")
	return resourcegendto.ResourcePushItemView{
		ID:            item.ID,
		KnowledgeID:   firstNonEmpty(item.KnowledgeID, resource.KnowledgeID),
		KnowledgeName: firstNonEmpty(item.KnowledgeName, resource.KnowledgeName),
		CourseTitle:   firstNonEmpty(item.CourseTitle, resource.CourseTitle),
		Stage:         firstNonEmpty(item.Stage, resourceStage(resource.ResourceType)),
		Reason:        firstNonEmpty(item.Reason, buildPushReason(resource.ResourceType, resource.Reason)),
		Priority:      item.Priority,
		Status:        status,
		ActionText:    pushActionText(resource.ResourceType, status),
		CreatedAt:     item.CreatedAt.Format(time.RFC3339),
		Resource:      resourceToView(resource),
	}
}

func pushActionText(resourceType string, status string) string {
	switch status {
	case "completed":
		return "已完成"
	case "needs_review":
		return "需调整"
	case "in_progress":
		return "继续学习"
	}
	switch resourceType {
	case "practice_questions":
		return "开始练习"
	case "coding_case":
		return "进入实操"
	case "video_script":
		return "查看分镜"
	case "generated_image":
		return "查看配图"
	case "generated_video":
		return "播放视频"
	default:
		return "开始学习"
	}
}

func resourceTypeLabel(resourceType string) string {
	switch resourceType {
	case "mind_map":
		return "知识点思维导图"
	case "practice_questions":
		return "个性化练习题"
	case "reading_material":
		return "拓展阅读材料"
	case "coding_case":
		return "代码类实操案例"
	case "video_script":
		return "多模态视频脚本"
	case "generated_image":
		return "AI 生成配图"
	case "generated_video":
		return "AI 生成短视频"
	default:
		return "专业课程讲解文档"
	}
}

func resourceFormat(resourceType string) string {
	switch resourceType {
	case "mind_map":
		return "mermaid"
	case "generated_image":
		return "image"
	case "generated_video":
		return "video"
	default:
		return "markdown"
	}
}

func jobToView(job *resourcegenentity.GenerationJob, steps []resourcegenentity.AgentStep, resources []resourcegenentity.GeneratedResource) resourcegendto.JobView {
	if job == nil {
		return resourcegendto.JobView{}
	}
	failureReason, retrySuggestion := classifyJobFailure(job)
	return resourcegendto.JobView{
		ID:              job.ID,
		StudentID:       job.StudentID,
		KnowledgeID:     job.KnowledgeID,
		KnowledgeName:   job.KnowledgeName,
		CourseTitle:     job.CourseTitle,
		Target:          job.Target,
		Status:          job.Status,
		Progress:        job.Progress,
		CurrentAgent:    job.CurrentAgent,
		ErrorMessage:    job.ErrorMessage,
		FailureReason:   failureReason,
		RetrySuggestion: retrySuggestion,
		CreatedAt:       job.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       job.UpdatedAt.Format(time.RFC3339),
		Steps:           stepsToViews(steps),
		Resources:       resourcesToViews(resources),
	}
}

func classifyJobFailure(job *resourcegenentity.GenerationJob) (string, string) {
	if job == nil {
		return "", ""
	}
	if job.Status == resourcegenentity.JobStatusCanceled {
		return "任务已手动停止", "确认生成参数后可点击“重试任务”重新发起。"
	}
	if job.Status != resourcegenentity.JobStatusFailed {
		return "", ""
	}
	message := strings.ToLower(strings.TrimSpace(job.ErrorMessage))
	switch {
	case message == "":
		return "任务异常结束", "建议重新生成；如果仍失败，请检查后端日志。"
	case containsAny(message, "api key", "apikey", "unauthorized", "401", "鉴权", "密钥"):
		return "模型鉴权失败", "检查 DEEPSEEK_API_KEY 或 OpenAI 兼容模型密钥后重试。"
	case containsAny(message, "not configured", "未配置", "disabled", "media model"):
		return "模型能力未配置", "补齐对应文本/图片/视频模型环境变量后重试。"
	case containsAny(message, "timeout", "deadline", "超时"):
		return "模型响应超时", "缩短生成目标、减少资源类型，或稍后重试。"
	case containsAny(message, "429", "rate limit", "too many"):
		return "模型请求限流", "等待上游额度恢复，或切换备用模型后重试。"
	case containsAny(message, "413", "too large", "payload", "图片", "image"):
		return "输入内容过大", "压缩图片或减少上传图片数量后重新生成。"
	case containsAny(message, "video", "视频"):
		return "视频生成失败", "检查视频模型端点和时长配置，或先生成图片/脚本资源。"
	default:
		return "上游模型或依赖异常", "建议重试；若持续失败，请检查模型服务、网络和 Docker 日志。"
	}
}

func stepsToViews(steps []resourcegenentity.AgentStep) []resourcegendto.AgentStepView {
	result := make([]resourcegendto.AgentStepView, 0, len(steps))
	for _, step := range steps {
		endedAt := ""
		if step.EndedAt != nil {
			endedAt = step.EndedAt.Format(time.RFC3339)
		}
		result = append(result, resourcegendto.AgentStepView{
			ID:        step.ID,
			AgentKey:  step.AgentKey,
			AgentName: step.AgentName,
			Role:      step.Role,
			Status:    step.Status,
			Summary:   step.Summary,
			StartedAt: step.StartedAt.Format(time.RFC3339),
			EndedAt:   endedAt,
		})
	}
	return result
}

func resourcesToViews(resources []resourcegenentity.GeneratedResource) []resourcegendto.GeneratedResourceView {
	result := make([]resourcegendto.GeneratedResourceView, 0, len(resources))
	for _, resource := range resources {
		result = append(result, resourceToView(resource))
	}
	return result
}

func resourcesToViewsWithLearningStatus(resources []resourcegenentity.GeneratedResource, statuses map[int64]string) []resourcegendto.GeneratedResourceView {
	result := make([]resourcegendto.GeneratedResourceView, 0, len(resources))
	for _, resource := range resources {
		status := firstNonEmpty(statuses[resource.ID], "pending")
		view := resourceToView(resource)
		view.LearningStatus = status
		view.ActionText = pushActionText(resource.ResourceType, status)
		result = append(result, view)
	}
	return result
}

func resourceIDs(resources []resourcegenentity.GeneratedResource) []int64 {
	result := make([]int64, 0, len(resources))
	for _, resource := range resources {
		if resource.ID > 0 {
			result = append(result, resource.ID)
		}
	}
	return result
}

func resourceToView(resource resourcegenentity.GeneratedResource) resourcegendto.GeneratedResourceView {
	return resourcegendto.GeneratedResourceView{
		ID:            resource.ID,
		JobID:         resource.JobID,
		ResourceType:  resource.ResourceType,
		TypeLabel:     resource.TypeLabel,
		Title:         resource.Title,
		Content:       resource.Content,
		Format:        resource.Format,
		KnowledgeID:   resource.KnowledgeID,
		KnowledgeName: resource.KnowledgeName,
		CourseTitle:   resource.CourseTitle,
		Reason:        resource.Reason,
		Status:        resource.Status,
		CreatedAt:     resource.CreatedAt.Format(time.RFC3339),
		Evidence:      resourceEvidence(resource),
	}
}

func resourceEvidence(resource resourcegenentity.GeneratedResource) []resourcegendto.ResourceEvidenceView {
	evidence := []resourcegendto.ResourceEvidenceView{
		{Label: "知识点", Value: firstNonEmpty(resource.KnowledgeName, resource.KnowledgeID, "未指定")},
		{Label: "课程", Value: firstNonEmpty(resource.CourseTitle, "人工智能专业课程")},
		{Label: "生成智能体", Value: writerAgentName(resource.ResourceType)},
	}
	if resource.Reason != "" {
		evidence = append(evidence, resourcegendto.ResourceEvidenceView{Label: "个性化依据", Value: resource.Reason})
	}
	if resource.JobID > 0 {
		evidence = append(evidence, resourcegendto.ResourceEvidenceView{Label: "任务编号", Value: strconv.FormatInt(resource.JobID, 10)})
	}
	evidence = append(evidence, resourcegendto.ResourceEvidenceView{Label: "安全机制", Value: "生成后由 SafetyReviewerAgent 进行事实性、格式和敏感内容检查。"})
	return evidence
}

func (s *Service) recordLearningEvent(ctx context.Context, event resourcegenentity.LearningEvent) {
	if event.StudentID <= 0 || strings.TrimSpace(event.EventType) == "" {
		return
	}
	event.EventType = normalizeEventType(event.EventType)
	if event.EventType == "" {
		return
	}
	if strings.TrimSpace(event.PayloadJSON) == "" {
		event.PayloadJSON = "{}"
	}
	_ = s.repo.CreateLearningEvent(ctx, &event)
}

func normalizeEventType(eventType string) string {
	allowed := map[string]bool{
		"resource_generation_started":  true,
		"resource_generated":           true,
		"resource_viewed":              true,
		"resource_feedback_submitted":  true,
		"resource_completed":           true,
		"practice_answered":            true,
		"note_saved":                   true,
		"profile_refreshed":            true,
		"image_generated":              true,
		"video_generated":              true,
		"resource_retry_started":       true,
		"resource_quality_checked":     true,
		"resource_compared":            true,
		"media_status_checked":         true,
		"job_review_generated":         true,
		"evidence_package_exported":    true,
		"study_checklist_generated":    true,
		"resource_variant_started":     true,
		"resource_provenance_viewed":   true,
		"resource_generation_canceled": true,
		"resource_status_updated":      true,
	}
	key := strings.ToLower(strings.TrimSpace(eventType))
	if allowed[key] {
		return key
	}
	return ""
}

func mustEventPayload(payload map[string]any) string {
	if len(payload) == 0 {
		return "{}"
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func learningEventsToViews(events []resourcegenentity.LearningEvent) []resourcegendto.LearningEventView {
	result := make([]resourcegendto.LearningEventView, 0, len(events))
	for _, event := range events {
		result = append(result, learningEventToView(event))
	}
	return result
}

func learningEventToView(event resourcegenentity.LearningEvent) resourcegendto.LearningEventView {
	return resourcegendto.LearningEventView{
		ID:            event.ID,
		EventType:     event.EventType,
		KnowledgeID:   event.KnowledgeID,
		KnowledgeName: event.KnowledgeName,
		ResourceID:    event.ResourceID,
		ResourceType:  event.ResourceType,
		Payload:       parseEventPayload(event.PayloadJSON),
		OccurredAt:    event.OccurredAt.Format(time.RFC3339),
	}
}

func parseEventPayload(payload string) map[string]any {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &parsed); err != nil {
		return nil
	}
	return parsed
}

func feedbackToView(feedback resourcegenentity.ResourceFeedback) resourcegendto.FeedbackView {
	return resourcegendto.FeedbackView{
		ID:         feedback.ID,
		ResourceID: feedback.ResourceID,
		JobID:      feedback.JobID,
		Rating:     feedback.Rating,
		Difficulty: feedback.Difficulty,
		Helpful:    feedback.Helpful,
		Completed:  feedback.Completed,
		Comment:    feedback.Comment,
		NextAction: feedback.NextAction,
		UpdatedAt:  feedback.UpdatedAt.Format(time.RFC3339),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func containsAny(value string, keywords ...string) bool {
	for _, keyword := range keywords {
		if keyword != "" && strings.Contains(value, keyword) {
			return true
		}
	}
	return false
}

func limitStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
