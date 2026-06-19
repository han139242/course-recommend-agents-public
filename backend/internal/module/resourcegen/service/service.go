package service

import (
	"context"
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
	GetJob(ctx context.Context, studentID int64, jobID int64) (*resourcegenentity.GenerationJob, []resourcegenentity.AgentStep, []resourcegenentity.GeneratedResource, error)
	ListResources(ctx context.Context, studentID int64, knowledgeID string, resourceType string, limit int) ([]resourcegenentity.GeneratedResource, error)
	GetResource(ctx context.Context, studentID int64, resourceID int64) (*resourcegenentity.GeneratedResource, error)
	UpsertFeedback(ctx context.Context, feedback *resourcegenentity.ResourceFeedback) (*resourcegenentity.ResourceFeedback, error)
	ListFeedback(ctx context.Context, studentID int64, limit int) ([]resourcegenentity.ResourceFeedback, error)
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
	s.bumpJob(ctx, job, 12, "ProfileAnalystAgent")

	s.advanceStep(ctx, stepByKey["profile_analyst"], summaries["profile_analyst"], resourcegenentity.StepStatusRunning)
	s.advanceStep(ctx, stepByKey["profile_analyst"], summaries["profile_analyst"], resourcegenentity.StepStatusCompleted)
	s.bumpJob(ctx, job, 22, "CoursePlannerAgent")

	s.advanceStep(ctx, stepByKey["course_planner"], summaries["course_planner"], resourcegenentity.StepStatusRunning)
	s.advanceStep(ctx, stepByKey["course_planner"], summaries["course_planner"], resourcegenentity.StepStatusCompleted)
	s.bumpJob(ctx, job, 32, "ResourceWriterAgents")

	s.advanceStep(ctx, stepByKey["resource_writer"], summaries["resource_writer"], resourcegenentity.StepStatusRunning)
	s.generateResourcesIncrementally(ctx, genCtx, jobID, job)
	s.advanceStep(ctx, stepByKey["resource_writer"], summaries["resource_writer"], resourcegenentity.StepStatusCompleted)

	s.bumpJob(ctx, job, 92, "SafetyReviewerAgent")
	s.advanceStep(ctx, stepByKey["safety_reviewer"], summaries["safety_reviewer"], resourcegenentity.StepStatusRunning)
	s.advanceStep(ctx, stepByKey["safety_reviewer"], summaries["safety_reviewer"], resourcegenentity.StepStatusCompleted)

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

func (s *Service) ListResources(ctx context.Context, query resourcegendto.ResourceQuery) (*resourcegendto.ListResourcesResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	resources, err := s.repo.ListResources(ctx, studentID, strings.TrimSpace(query.KnowledgeID), strings.TrimSpace(query.Type), query.Limit)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	list := resourcesToViews(resources)
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
	view := feedbackToView(*stored)
	return &view, nil
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

	resources, err := s.repo.ListResources(ctx, studentID, "", "", 12)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	feedback, err := s.repo.ListFeedback(ctx, studentID, 30)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	summary := buildFeedbackSummary(feedback)
	recommendations := buildRecommendations(profile, resources, summary)
	return &resourcegendto.ResourceRecommendationsResponse{
		Summary:         buildRecommendationSummary(profile, resources, summary),
		FeedbackSummary: summary,
		List:            recommendations,
	}, nil
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
	}
}

// stepSummaries returns the per-agent step summaries for a generation context.
// It is reused both when steps are first created (preliminary context) and when
// they are advanced with the fully resolved context.
func (s *Service) stepSummaries(genCtx generationContext) map[string]string {
	return map[string]string{
		"profile_analyst": "读取学生画像、学习目标和资源偏好，形成个性化资源生成需求：" + profileSummary(genCtx),
		"course_planner":  "结合知识图谱和推荐路径，锁定“" + genCtx.KnowledgeName + "”及其先修/后续学习顺序。",
		"resource_writer": s.resourceWriterSummary(genCtx),
		"safety_reviewer": s.safetyReviewerSummary(),
	}
}

// buildSteps creates the ordered multi-agent steps for a job in the pending
// state; runGeneration later advances them to running/completed as work proceeds.
func (s *Service) buildSteps(genCtx generationContext) []resourcegenentity.AgentStep {
	summaries := s.stepSummaries(genCtx)
	specs := []struct {
		Key  string
		Name string
		Role string
	}{
		{"profile_analyst", "ProfileAnalystAgent", "画像分析智能体"},
		{"course_planner", "CoursePlannerAgent", "课程规划智能体"},
		{"resource_writer", "ResourceWriterAgents", "资源生成智能体组"},
		{"safety_reviewer", "SafetyReviewerAgent", "安全校验智能体"},
	}

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
	job.Progress = progress
	if currentAgent != "" {
		job.CurrentAgent = currentAgent
	}
	_ = s.repo.UpdateJob(ctx, job)
}

// failJob marks a job as failed with a clamped error message.
func (s *Service) failJob(ctx context.Context, job *resourcegenentity.GenerationJob, message string) {
	job.Status = resourcegenentity.JobStatusFailed
	job.ErrorMessage = clampRunes(message, 900)
	_ = s.repo.UpdateJob(ctx, job)
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
			label := resourceTypeLabel(resourceType)
			content := s.aiResourceContent(ctx, resourceType, genCtx)
			if strings.TrimSpace(content) == "" {
				content = buildResourceContent(resourceType, genCtx)
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

			mu.Lock()
			defer mu.Unlock()
			if err := s.repo.CreateResource(ctx, resource); err != nil {
				return
			}
			completed++
			s.bumpJob(ctx, job, 35+int(float64(53)*float64(completed)/float64(total)), writerAgentName(resourceType))
		}(resourceType)
	}
	wg.Wait()
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

func buildSafetyReport(resource resourcegenentity.GeneratedResource) resourcegendto.SafetyReportView {
	content := strings.TrimSpace(resource.Content)
	checks := []resourcegendto.SafetyCheckView{
		buildSafetyCheck("factuality", "事实性与防幻觉", !containsAny(content, "保证", "绝对正确", "100%", "唯一答案"), "未发现绝对化或不可验证承诺，建议保留人工复核。"),
		buildSafetyCheck("sensitive", "敏感与违规过滤", !containsAny(content, "违法", "攻击", "绕过", "作弊"), "未发现明显敏感违规表达。"),
		buildSafetyCheck("structure", "格式完整性", content != "" && strings.Contains(content, "#"), "资源包含标题和结构化内容，可直接展示。"),
		buildSafetyCheck("pedagogy", "教学适配性", strings.TrimSpace(resource.Reason) != "", "资源保留了生成理由，能够解释个性化依据。"),
		buildSafetyCheck("license", "开源与 AI 工具说明", true, "当前资源为系统生成内容；若后续接入外部材料，需要补充来源与协议。"),
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
	default:
		return "根据当前画像、知识图谱和推荐路径生成个性化讲解。"
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
	default:
		return "专业课程讲解文档"
	}
}

func resourceFormat(resourceType string) string {
	if resourceType == "mind_map" {
		return "mermaid"
	}
	return "markdown"
}

func jobToView(job *resourcegenentity.GenerationJob, steps []resourcegenentity.AgentStep, resources []resourcegenentity.GeneratedResource) resourcegendto.JobView {
	if job == nil {
		return resourcegendto.JobView{}
	}
	return resourcegendto.JobView{
		ID:            job.ID,
		StudentID:     job.StudentID,
		KnowledgeID:   job.KnowledgeID,
		KnowledgeName: job.KnowledgeName,
		CourseTitle:   job.CourseTitle,
		Target:        job.Target,
		Status:        job.Status,
		Progress:      job.Progress,
		CurrentAgent:  job.CurrentAgent,
		ErrorMessage:  job.ErrorMessage,
		CreatedAt:     job.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     job.UpdatedAt.Format(time.RFC3339),
		Steps:         stepsToViews(steps),
		Resources:     resourcesToViews(resources),
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
	}
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
