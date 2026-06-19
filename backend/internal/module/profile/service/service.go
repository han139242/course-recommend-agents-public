package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	profiledto "github.com/han139242/course-recommend-agents/backend/internal/module/profile/dto"
	profileentity "github.com/han139242/course-recommend-agents/backend/internal/module/profile/entity"
	profilerepo "github.com/han139242/course-recommend-agents/backend/internal/module/profile/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

type Service struct {
	repo repository
}

type repository interface {
	GetCurrent(ctx context.Context, studentID int64) (*profileentity.LearningProfile, []profileentity.ProfileDimension, error)
	UpsertProfile(ctx context.Context, profile *profileentity.LearningProfile, dimensions []profileentity.ProfileDimension, versionPayload string) (*profileentity.LearningProfile, []profileentity.ProfileDimension, error)
}

func New(repo repository) *Service {
	return &Service{repo: repo}
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

	dimensions := inferProfileDimensions(message)
	summary := buildProfileSummary(dimensions)
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

func containsAny(input string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
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
