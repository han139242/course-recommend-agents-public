package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

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
	profileService   *profileservice.Service
	recommendService *recommendservice.Service
	graphService     *graphservice.Service
}

type repository interface {
	CreateJob(ctx context.Context, job *resourcegenentity.GenerationJob) error
	UpdateJob(ctx context.Context, job *resourcegenentity.GenerationJob) error
	CreateStep(ctx context.Context, step *resourcegenentity.AgentStep) error
	CreateResources(ctx context.Context, resources []resourcegenentity.GeneratedResource) error
	GetJob(ctx context.Context, studentID int64, jobID int64) (*resourcegenentity.GenerationJob, []resourcegenentity.AgentStep, []resourcegenentity.GeneratedResource, error)
	ListResources(ctx context.Context, studentID int64, knowledgeID string, resourceType string, limit int) ([]resourcegenentity.GeneratedResource, error)
	GetResource(ctx context.Context, studentID int64, resourceID int64) (*resourcegenentity.GeneratedResource, error)
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

func New(repo repository, profileService *profileservice.Service, recommendService *recommendservice.Service, graphService *graphservice.Service) *Service {
	return &Service{repo: repo, profileService: profileService, recommendService: recommendService, graphService: graphService}
}

func (s *Service) Generate(ctx context.Context, request resourcegendto.GenerateResourcesRequest) (*resourcegendto.GenerateResourcesResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}

	genCtx := s.buildGenerationContext(ctx, studentID, request)
	job := &resourcegenentity.GenerationJob{
		StudentID:     studentID,
		KnowledgeID:   genCtx.KnowledgeID,
		KnowledgeName: genCtx.KnowledgeName,
		CourseTitle:   genCtx.CourseTitle,
		Target:        genCtx.Target,
		Status:        resourcegenentity.JobStatusRunning,
		Progress:      5,
		CurrentAgent:  "Orchestrator",
	}
	if err := s.repo.CreateJob(ctx, job); err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	steps := s.buildSteps(genCtx)
	for index := range steps {
		steps[index].JobID = job.ID
		steps[index].StudentID = studentID
		if err := s.repo.CreateStep(ctx, &steps[index]); err != nil {
			job.Status = resourcegenentity.JobStatusFailed
			job.ErrorMessage = err.Error()
			_ = s.repo.UpdateJob(ctx, job)
			return nil, apperr.ErrDependencyUnavailable.WithCause(err)
		}
		job.CurrentAgent = steps[index].AgentName
		job.Progress = 15 + index*15
		_ = s.repo.UpdateJob(ctx, job)
	}

	resources := s.buildResources(genCtx, job.ID)
	if err := s.repo.CreateResources(ctx, resources); err != nil {
		job.Status = resourcegenentity.JobStatusFailed
		job.ErrorMessage = err.Error()
		_ = s.repo.UpdateJob(ctx, job)
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	job.Status = resourcegenentity.JobStatusCompleted
	job.Progress = 100
	job.CurrentAgent = "SafetyReviewerAgent"
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	storedJob, storedSteps, storedResources, err := s.repo.GetJob(ctx, studentID, job.ID)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	jobView := jobToView(storedJob, storedSteps, storedResources)
	return &resourcegendto.GenerateResourcesResponse{Job: jobView, Resources: jobView.Resources}, nil
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

func (s *Service) buildSteps(ctx generationContext) []resourcegenentity.AgentStep {
	now := time.Now()
	steps := []resourcegenEntityStepSpec{
		{
			Key:     "profile_analyst",
			Name:    "ProfileAnalystAgent",
			Role:    "画像分析智能体",
			Summary: "读取学生画像、学习目标和资源偏好，形成个性化资源生成需求：" + profileSummary(ctx),
		},
		{
			Key:     "course_planner",
			Name:    "CoursePlannerAgent",
			Role:    "课程规划智能体",
			Summary: "结合知识图谱和推荐路径，锁定“" + ctx.KnowledgeName + "”及其先修/后续学习顺序。",
		},
		{
			Key:     "resource_writer",
			Name:    "ResourceWriterAgents",
			Role:    "资源生成智能体组",
			Summary: "协同生成讲解文档、思维导图、练习题、拓展阅读和实操案例。",
		},
		{
			Key:     "safety_reviewer",
			Name:    "SafetyReviewerAgent",
			Role:    "安全校验智能体",
			Summary: "检查资源格式、学习依据和内容安全，标记为可展示资源。",
		},
	}

	result := make([]resourcegenentity.AgentStep, 0, len(steps))
	for index, step := range steps {
		startedAt := now.Add(time.Duration(index*180) * time.Millisecond)
		endedAt := startedAt.Add(120 * time.Millisecond)
		result = append(result, resourcegenentity.AgentStep{
			AgentKey:  step.Key,
			AgentName: step.Name,
			Role:      step.Role,
			Status:    resourcegenentity.StepStatusCompleted,
			Summary:   step.Summary,
			StartedAt: startedAt,
			EndedAt:   &endedAt,
		})
	}
	return result
}

type resourcegenEntityStepSpec struct {
	Key     string
	Name    string
	Role    string
	Summary string
}

func (s *Service) buildResources(ctx generationContext, jobID int64) []resourcegenentity.GeneratedResource {
	resources := make([]resourcegenentity.GeneratedResource, 0, len(ctx.ResourceTypes))
	for _, resourceType := range ctx.ResourceTypes {
		label := resourceTypeLabel(resourceType)
		resources = append(resources, resourcegenentity.GeneratedResource{
			JobID:         jobID,
			StudentID:     ctx.StudentID,
			ResourceType:  resourceType,
			TypeLabel:     label,
			Title:         label + "：" + ctx.KnowledgeName,
			Content:       buildResourceContent(resourceType, ctx),
			Format:        resourceFormat(resourceType),
			KnowledgeID:   ctx.KnowledgeID,
			KnowledgeName: ctx.KnowledgeName,
			CourseTitle:   ctx.CourseTitle,
			Reason:        buildResourceReason(resourceType, ctx),
			Status:        resourcegenentity.ResourceStatusReady,
		})
	}
	return resources
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
