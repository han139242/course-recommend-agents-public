package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/ai"
	profiledto "github.com/han139242/course-recommend-agents/backend/internal/module/profile/dto"
	profileentity "github.com/han139242/course-recommend-agents/backend/internal/module/profile/entity"
	profilerepo "github.com/han139242/course-recommend-agents/backend/internal/module/profile/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

type Service struct {
	repo     repository
	aiClient *ai.Client
}

type repository interface {
	GetCurrent(ctx context.Context, studentID int64) (*profileentity.LearningProfile, []profileentity.ProfileDimension, error)
	UpsertProfile(ctx context.Context, profile *profileentity.LearningProfile, dimensions []profileentity.ProfileDimension, versionPayload string) (*profileentity.LearningProfile, []profileentity.ProfileDimension, error)
	ListVersions(ctx context.Context, studentID int64, limit int) ([]profileentity.ProfileVersion, error)
	ListLearningEvents(ctx context.Context, studentID int64, limit int) ([]profileentity.LearningEvent, error)
}

func New(repo repository, aiClient *ai.Client) *Service {
	return &Service{repo: repo, aiClient: aiClient}
}

func (s *Service) ExtractProfile(ctx context.Context, request profiledto.ExtractProfileRequest) (*profiledto.ProfileResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	message := strings.TrimSpace(request.Message)
	if message == "" {
		return nil, apperr.ErrBadRequest
	}

	dimensions, summary := s.inferProfile(ctx, message)
	return s.persist(ctx, studentID, summary, message, dimensions)
}

func (s *Service) GetCurrent(ctx context.Context) (*profiledto.ProfileResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	profile, dimensions, err := s.repo.GetCurrent(ctx, studentID)
	if errors.Is(err, profilerepo.ErrNotFound) {
		return &profiledto.ProfileResponse{Profile: nil, HasCompletedProfile: false}, nil
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	return &profiledto.ProfileResponse{Profile: profileToView(profile, dimensions), HasCompletedProfile: true}, nil
}

func (s *Service) RefreshProfile(ctx context.Context) (*profiledto.ProfileResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	profile, dimensions, err := s.repo.GetCurrent(ctx, studentID)
	if errors.Is(err, profilerepo.ErrNotFound) {
		return &profiledto.ProfileResponse{Profile: nil, HasCompletedProfile: false}, nil
	}
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	events, err := s.repo.ListLearningEvents(ctx, studentID, 40)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	if len(events) == 0 {
		return &profiledto.ProfileResponse{
			Profile:             profileToView(profile, dimensions),
			HasCompletedProfile: true,
			RefreshInsights:     []string{"暂无新的学习行为，画像保持不变。"},
		}, nil
	}

	nextDimensions, insights := refreshDimensionsFromEvents(dimensionsToViews(dimensions), events)
	summary := buildProfileSummary(nextDimensions)
	sourceText := strings.TrimSpace(profile.SourceText)
	if sourceText == "" {
		sourceText = "由学习行为刷新画像"
	}
	response, err := s.persist(ctx, studentID, summary, sourceText, nextDimensions)
	if err != nil {
		return nil, err
	}
	response.RefreshInsights = insights
	response.RecentSignals = learningEventsToViews(events, 8)
	return response, nil
}

func (s *Service) GetTimeline(ctx context.Context) (*profiledto.ProfileTimelineResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	versions, err := s.repo.ListVersions(ctx, studentID, 12)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	events, err := s.repo.ListLearningEvents(ctx, studentID, 12)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	return &profiledto.ProfileTimelineResponse{
		Items:        profileVersionsToTimeline(versions),
		RecentEvents: learningEventsToViews(events, 12),
	}, nil
}

func (s *Service) UpsertProfile(ctx context.Context, request profiledto.UpsertProfileRequest) (*profiledto.ProfileResponse, error) {
	studentID, err := currentStudentID(ctx)
	if err != nil {
		return nil, err
	}
	dimensions, err := normalizeDimensions(request.Dimensions)
	if err != nil {
		return nil, err
	}
	summary := strings.TrimSpace(request.Summary)
	if summary == "" {
		summary = buildProfileSummary(dimensions)
	}
	return s.persist(ctx, studentID, summary, strings.TrimSpace(request.SourceText), dimensions)
}

func (s *Service) persist(ctx context.Context, studentID int64, summary string, sourceText string, dimensions []profiledto.ProfileDimensionView) (*profiledto.ProfileResponse, error) {
	entities := make([]profileentity.ProfileDimension, 0, len(dimensions))
	for _, dimension := range dimensions {
		entities = append(entities, profileentity.ProfileDimension{
			Key:        strings.TrimSpace(dimension.Key),
			Name:       strings.TrimSpace(dimension.Name),
			Value:      strings.TrimSpace(dimension.Value),
			Confidence: clampConfidence(dimension.Confidence),
			Evidence:   strings.TrimSpace(dimension.Evidence),
		})
	}

	versionPayload := ""
	if payload, err := json.Marshal(profiledto.LearningProfileView{
		StudentID:  studentID,
		Summary:    summary,
		SourceText: sourceText,
		Dimensions: dimensions,
		UpdatedAt:  time.Now().Format(time.RFC3339),
	}); err == nil {
		versionPayload = string(payload)
	}

	profile, savedDimensions, err := s.repo.UpsertProfile(ctx, &profileentity.LearningProfile{
		StudentID:  studentID,
		Summary:    strings.TrimSpace(summary),
		SourceText: strings.TrimSpace(sourceText),
	}, entities, versionPayload)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	return &profiledto.ProfileResponse{Profile: profileToView(profile, savedDimensions), HasCompletedProfile: true}, nil
}

func currentStudentID(ctx context.Context) (int64, error) {
	studentID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		return 0, apperr.ErrUnauthorized
	}
	return studentID, nil
}

func normalizeDimensions(input []profiledto.ProfileDimensionView) ([]profiledto.ProfileDimensionView, error) {
	result := make([]profiledto.ProfileDimensionView, 0, len(input))
	seen := map[string]struct{}{}
	for _, item := range input {
		key := strings.TrimSpace(item.Key)
		name := strings.TrimSpace(item.Name)
		value := strings.TrimSpace(item.Value)
		if key == "" || name == "" || value == "" {
			return nil, apperr.ErrBadRequest
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		item.Key = key
		item.Name = name
		item.Value = value
		item.Confidence = clampConfidence(item.Confidence)
		item.Evidence = strings.TrimSpace(item.Evidence)
		result = append(result, item)
	}
	if len(result) < 6 {
		return nil, apperr.ErrBadRequest.WithCause(fmt.Errorf("profile requires at least 6 dimensions"))
	}
	return result, nil
}

func inferProfileDimensions(message string) []profiledto.ProfileDimensionView {
	learned := inferLearnedTopics(message)
	target := inferTargetDirection(message)
	goal := inferGoal(message, target)
	foundation := inferFoundation(message, learned)
	cognitiveStyle := inferCognitiveStyle(message)
	errorPattern := inferErrorPattern(message)
	resourcePreference := inferResourcePreference(message)
	pace := inferLearningPace(message)

	return []profiledto.ProfileDimensionView{
		{Key: "knowledge_foundation", Name: "知识基础", Value: foundation, Confidence: 0.78, Evidence: compactEvidence(message, learned)},
		{Key: "learning_goal", Name: "学习目标", Value: goal, Confidence: 0.82, Evidence: compactEvidence(message, []string{target})},
		{Key: "learning_progress", Name: "学习进度", Value: pace, Confidence: 0.66, Evidence: compactEvidence(message, inferGradeKeywords(message))},
		{Key: "cognitive_style", Name: "认知风格", Value: cognitiveStyle, Confidence: 0.62, Evidence: compactEvidence(message, []string{cognitiveStyle})},
		{Key: "error_preference", Name: "易错点偏好", Value: errorPattern, Confidence: 0.68, Evidence: compactEvidence(message, []string{"不稳", "薄弱", "公式", "代码", "概念"})},
		{Key: "resource_preference", Name: "资源偏好", Value: resourcePreference, Confidence: 0.7, Evidence: compactEvidence(message, []string{"视频", "导图", "代码", "题", "案例", "PPT"})},
	}
}

func inferLearnedTopics(message string) []string {
	candidates := []string{"Python", "高等数学", "线性代数", "概率论", "机器学习", "深度学习", "神经网络", "数据结构", "知识图谱", "自然语言处理", "大模型", "RAG", "智能体"}
	result := make([]string, 0, 6)
	lower := strings.ToLower(message)
	for _, candidate := range candidates {
		if strings.Contains(lower, strings.ToLower(candidate)) {
			result = append(result, candidate)
		}
	}
	return result
}

func inferTargetDirection(message string) string {
	candidates := []string{"智能体", "大模型应用", "RAG", "算法工程师", "机器学习", "深度学习", "计算机视觉", "自然语言处理", "知识图谱", "强化学习", "数据分析"}
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(message), strings.ToLower(candidate)) {
			return candidate
		}
	}
	if containsAny(message, "项目", "作品", "系统") {
		return "AI 项目实践"
	}
	return "人工智能课程学习"
}

func inferGoal(message string, target string) string {
	switch {
	case containsAny(message, "项目", "作品", "系统", "应用"):
		return "围绕" + target + "完成可展示的项目实践"
	case containsAny(message, "就业", "实习", "岗位", "求职"):
		return "面向" + target + "相关岗位补齐知识与项目能力"
	case containsAny(message, "考试", "期末", "考研"):
		return "围绕课程考核建立稳固知识结构"
	default:
		return "围绕" + target + "形成系统化学习路径"
	}
}

func inferFoundation(message string, learned []string) string {
	switch {
	case containsAny(message, "基础很好", "比较熟", "掌握不错", "熟悉"):
		return "基础较好，已具备继续拓展的先修条件"
	case containsAny(message, "不太稳", "薄弱", "不会", "零基础", "刚开始"):
		if len(learned) > 0 {
			return "已接触" + strings.Join(learned, "、") + "，但自述基础不稳，需要先补先修链"
		}
		return "基础偏弱，需要从先修知识和核心概念开始补齐"
	case len(learned) > 0:
		return "已学习" + strings.Join(learned, "、") + "，适合在已有基础上推进"
	default:
		return "基础信息不足，建议先通过诊断题确认知识起点"
	}
}

func inferCognitiveStyle(message string) string {
	switch {
	case containsAny(message, "项目", "实践", "代码", "案例", "做出来"):
		return "项目驱动型：更适合从案例、代码和可运行任务进入概念"
	case containsAny(message, "公式", "原理", "理论", "推导"):
		return "原理推导型：更适合先建立定义、公式和推理链"
	case containsAny(message, "图", "导图", "视频", "动画"):
		return "视觉结构型：更适合通过图解、导图和视频脚本理解"
	default:
		return "均衡型：适合按讲解、练习、案例交替推进"
	}
}

func inferErrorPattern(message string) string {
	switch {
	case containsAny(message, "公式", "数学", "线性代数", "概率"):
		return "容易卡在数学推导与符号含义，需要更多分步解释和例题"
	case containsAny(message, "代码", "编程", "实现", "报错"):
		return "容易卡在代码实现和调试，需要配套实操案例"
	case containsAny(message, "概念", "混", "分不清", "区别"):
		return "容易混淆相近概念，需要对比卡片和反例解释"
	case containsAny(message, "不太稳", "薄弱", "不会"):
		return "基础掌握不稳定，需要优先定位先修短板"
	default:
		return "暂无明确易错点，先通过练习题和学习行为持续识别"
	}
}

func inferResourcePreference(message string) string {
	preferences := make([]string, 0, 5)
	if containsAny(message, "文档", "讲义", "PPT") {
		preferences = append(preferences, "讲解文档")
	}
	if containsAny(message, "导图", "图解", "结构") {
		preferences = append(preferences, "知识导图")
	}
	if containsAny(message, "题", "练习", "测试") {
		preferences = append(preferences, "练习题")
	}
	if containsAny(message, "代码", "案例", "实操", "项目") {
		preferences = append(preferences, "实操案例")
	}
	if containsAny(message, "视频", "动画") {
		preferences = append(preferences, "视频/动画脚本")
	}
	if len(preferences) == 0 {
		preferences = []string{"讲解文档", "知识导图", "练习题", "实操案例"}
	}
	return strings.Join(preferences, "、")
}

func inferLearningPace(message string) string {
	switch {
	case containsAny(message, "大一", "一年级"):
		return "大一阶段，适合低门槛导入和基础补齐"
	case containsAny(message, "大二", "二年级"):
		return "大二阶段，适合从课程基础过渡到项目实践"
	case containsAny(message, "大三", "三年级"):
		return "大三阶段，适合结合方向课和项目成果推进"
	case containsAny(message, "大四", "四年级", "毕业"):
		return "高年级阶段，适合聚焦作品、实习和成果展示"
	case containsAny(message, "研究生", "研一", "研二", "研三"):
		return "研究生阶段，适合结合论文、前沿资料和工程实验"
	default:
		return "学习阶段未明确，需要根据后续行为动态更新节奏"
	}
}

func inferGradeKeywords(message string) []string {
	keywords := []string{"大一", "大二", "大三", "大四", "研究生", "研一", "研二", "研三"}
	result := make([]string, 0, 2)
	for _, keyword := range keywords {
		if strings.Contains(message, keyword) {
			result = append(result, keyword)
		}
	}
	return result
}

func buildProfileSummary(dimensions []profiledto.ProfileDimensionView) string {
	goal := dimensionValue(dimensions, "learning_goal")
	foundation := dimensionValue(dimensions, "knowledge_foundation")
	preference := dimensionValue(dimensions, "resource_preference")
	return strings.TrimSpace("画像摘要：" + goal + "；" + foundation + "；优先资源：" + preference + "。")
}

func dimensionValue(dimensions []profiledto.ProfileDimensionView, key string) string {
	for _, dimension := range dimensions {
		if dimension.Key == key {
			return strings.TrimSpace(dimension.Value)
		}
	}
	return ""
}

func compactEvidence(message string, candidates []string) string {
	for _, candidate := range candidates {
		if candidate != "" && strings.Contains(strings.ToLower(message), strings.ToLower(candidate)) {
			return "来自描述中的“" + candidate + "”"
		}
	}
	runes := []rune(strings.TrimSpace(message))
	if len(runes) > 36 {
		return "来自描述：“" + string(runes[:36]) + "...”"
	}
	if len(runes) == 0 {
		return "来自当前对话"
	}
	return "来自描述：“" + string(runes) + "”"
}

func refreshDimensionsFromEvents(current []profiledto.ProfileDimensionView, events []profileentity.LearningEvent) ([]profiledto.ProfileDimensionView, []string) {
	next := make([]profiledto.ProfileDimensionView, len(current))
	copy(next, current)
	byKey := map[string]int{}
	for index, dimension := range next {
		byKey[dimension.Key] = index
	}

	completed := 0
	hard := 0
	lowRating := 0
	practiceWrong := 0
	resourceTypeCounts := map[string]int{}
	knowledgeNames := make([]string, 0, 4)
	comments := make([]string, 0, 3)

	for _, event := range events {
		payload := parseEventPayload(event.PayloadJSON)
		if event.KnowledgeName != "" && len(knowledgeNames) < 4 && !containsString(knowledgeNames, event.KnowledgeName) {
			knowledgeNames = append(knowledgeNames, event.KnowledgeName)
		}
		if event.ResourceType != "" {
			resourceTypeCounts[event.ResourceType]++
		}
		switch event.EventType {
		case "resource_feedback_submitted":
			if boolPayload(payload, "completed") {
				completed++
			}
			if strings.EqualFold(stringPayload(payload, "difficulty"), "hard") {
				hard++
			}
			if rating := floatPayload(payload, "rating"); rating > 0 && rating <= 2 {
				lowRating++
			}
			if comment := stringPayload(payload, "comment"); comment != "" && len(comments) < 3 {
				comments = append(comments, comment)
			}
		case "resource_completed":
			completed++
		case "practice_answered":
			if correct, ok := payload["correct"].(bool); ok && !correct {
				practiceWrong++
			}
		}
	}

	insights := make([]string, 0, 4)
	if completed > 0 {
		value := fmt.Sprintf("最近完成 %d 份个性化资源，学习重点集中在：%s。", completed, firstNonEmpty(strings.Join(knowledgeNames, "、"), "当前路径知识点"))
		updateDimension(next, byKey, "learning_progress", value, 0.86, "来自资源完成和反馈记录")
		insights = append(insights, "学习进度已根据资源完成记录更新。")
	}
	if hard > 0 || lowRating > 0 || practiceWrong > 0 {
		value := "近期反馈显示仍需要先修拆解、易错提醒和分步练习。"
		if len(comments) > 0 {
			value = "近期反馈集中在：" + strings.Join(comments, "；")
		}
		updateDimension(next, byKey, "error_preference", value, 0.82, "来自偏难反馈、低评分或答题错误")
		insights = append(insights, "易错点偏好已根据偏难反馈和答题结果更新。")
	}
	if preference := preferredResourceTypes(resourceTypeCounts); preference != "" {
		updateDimension(next, byKey, "resource_preference", preference, 0.84, "来自最近浏览、生成和完成的资源类型")
		insights = append(insights, "资源偏好已根据近期使用频次更新。")
		if strings.Contains(preference, "代码") {
			updateDimension(next, byKey, "cognitive_style", "项目驱动型：更适合通过案例、代码和可运行任务进入概念。", 0.78, "来自代码案例资源使用记录")
		} else if strings.Contains(preference, "导图") || strings.Contains(preference, "视频") {
			updateDimension(next, byKey, "cognitive_style", "视觉结构型：更适合通过导图、分镜和图解理解。", 0.76, "来自导图或视频脚本资源使用记录")
		}
	}
	if len(insights) == 0 {
		insights = append(insights, "已读取近期学习行为，暂未发现需要显著调整的画像维度。")
	}
	return next, insights
}

func updateDimension(dimensions []profiledto.ProfileDimensionView, byKey map[string]int, key string, value string, confidence float64, evidence string) {
	index, ok := byKey[key]
	if !ok || strings.TrimSpace(value) == "" {
		return
	}
	dimensions[index].Value = strings.TrimSpace(value)
	dimensions[index].Confidence = clampConfidence(confidence)
	dimensions[index].Evidence = strings.TrimSpace(evidence)
}

func preferredResourceTypes(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}
	type pair struct {
		key   string
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for key, count := range counts {
		if count > 0 {
			pairs = append(pairs, pair{key: key, count: count})
		}
	}
	if len(pairs) == 0 {
		return ""
	}
	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].count > pairs[i].count {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	labels := make([]string, 0, minInt(len(pairs), 4))
	for _, item := range pairs {
		labels = append(labels, resourceTypeLabel(item.key))
		if len(labels) >= 4 {
			break
		}
	}
	return "近期更偏好：" + strings.Join(labels, "、")
}

func resourceTypeLabel(resourceType string) string {
	switch resourceType {
	case "mind_map":
		return "知识导图"
	case "practice_questions":
		return "练习题"
	case "reading_material":
		return "拓展阅读"
	case "coding_case":
		return "代码实操案例"
	case "video_script":
		return "视频/动画脚本"
	default:
		return "讲解文档"
	}
}

func containsAny(input string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func containsString(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func dimensionsToViews(dimensions []profileentity.ProfileDimension) []profiledto.ProfileDimensionView {
	result := make([]profiledto.ProfileDimensionView, 0, len(dimensions))
	for _, dimension := range dimensions {
		result = append(result, profiledto.ProfileDimensionView{
			Key:        dimension.Key,
			Name:       dimension.Name,
			Value:      dimension.Value,
			Confidence: clampConfidence(dimension.Confidence),
			Evidence:   dimension.Evidence,
		})
	}
	return result
}

func profileVersionsToTimeline(versions []profileentity.ProfileVersion) []profiledto.ProfileTimelineItemView {
	result := make([]profiledto.ProfileTimelineItemView, 0, len(versions))
	for index, version := range versions {
		var payload profiledto.LearningProfileView
		_ = json.Unmarshal([]byte(version.PayloadJSON), &payload)
		title := "画像版本"
		if index == 0 {
			title = "当前画像"
		}
		result = append(result, profiledto.ProfileTimelineItemView{
			ID:         version.ID,
			Title:      title,
			Summary:    firstNonEmpty(version.Summary, payload.Summary),
			Source:     firstNonEmpty(payload.SourceText, "学习行为刷新"),
			Dimensions: payload.Dimensions,
			CreatedAt:  version.CreatedAt.Format(time.RFC3339),
		})
	}
	return result
}

func learningEventsToViews(events []profileentity.LearningEvent, limit int) []profiledto.LearningEventView {
	if limit <= 0 || limit > len(events) {
		limit = len(events)
	}
	result := make([]profiledto.LearningEventView, 0, limit)
	for _, event := range events[:limit] {
		result = append(result, profiledto.LearningEventView{
			ID:            event.ID,
			EventType:     event.EventType,
			KnowledgeID:   event.KnowledgeID,
			KnowledgeName: event.KnowledgeName,
			ResourceID:    event.ResourceID,
			ResourceType:  event.ResourceType,
			Payload:       parseEventPayload(event.PayloadJSON),
			OccurredAt:    event.OccurredAt.Format(time.RFC3339),
		})
	}
	return result
}

func parseEventPayload(payload string) map[string]any {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &parsed); err != nil {
		return nil
	}
	return parsed
}

func stringPayload(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if value, ok := payload[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func boolPayload(payload map[string]any, key string) bool {
	if payload == nil {
		return false
	}
	value, ok := payload[key].(bool)
	return ok && value
}

func floatPayload(payload map[string]any, key string) float64 {
	if payload == nil {
		return 0
	}
	switch value := payload[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	default:
		return 0
	}
}

func profileToView(profile *profileentity.LearningProfile, dimensions []profileentity.ProfileDimension) *profiledto.LearningProfileView {
	if profile == nil {
		return nil
	}
	items := make([]profiledto.ProfileDimensionView, 0, len(dimensions))
	for _, dimension := range dimensions {
		items = append(items, profiledto.ProfileDimensionView{
			Key:        dimension.Key,
			Name:       dimension.Name,
			Value:      dimension.Value,
			Confidence: clampConfidence(dimension.Confidence),
			Evidence:   dimension.Evidence,
		})
	}
	return &profiledto.LearningProfileView{
		ID:         profile.ID,
		StudentID:  profile.StudentID,
		Summary:    profile.Summary,
		SourceText: profile.SourceText,
		Dimensions: items,
		UpdatedAt:  profile.UpdatedAt.Format(time.RFC3339),
	}
}

func clampConfidence(value float64) float64 {
	if value <= 0 {
		return 0.6
	}
	if value > 1 {
		return 1
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
