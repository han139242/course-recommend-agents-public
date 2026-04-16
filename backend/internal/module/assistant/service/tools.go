package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	authdto "github.com/han139242/course-recommend-agents/backend/internal/module/auth/dto"
	competencydto "github.com/han139242/course-recommend-agents/backend/internal/module/competency/dto"
	frontierdto "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/dto"
	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

func buildAgentTools() []map[string]any {
	return []map[string]any{
		{
			"type": "function",
			"function": map[string]any{
				"name":        "get_current_user_context",
				"description": "读取当前已登录用户的基本身份信息、目标方向和当前会话上下文。",
				"parameters": map[string]any{
					"type":                 "object",
					"properties":           map[string]any{},
					"additionalProperties": false,
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "get_dashboard_summary",
				"description": "读取当前用户首页摘要、推荐卡片和推荐路径结果。",
				"parameters": map[string]any{
					"type":                 "object",
					"properties":           map[string]any{},
					"additionalProperties": false,
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "get_competency_profile",
				"description": "读取当前用户能力画像、雷达维度、薄弱能力和成长洞察。",
				"parameters": map[string]any{
					"type":                 "object",
					"properties":           map[string]any{},
					"additionalProperties": false,
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "get_learning_progress",
				"description": "读取当前用户知识点掌握度、正确率、答题次数和最近答题时间。",
				"parameters": map[string]any{
					"type":                 "object",
					"properties":           map[string]any{},
					"additionalProperties": false,
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "get_recommended_path",
				"description": "读取当前用户的下一步学习推荐路径。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"target": map[string]any{
							"type":        "string",
							"description": "可选，覆盖默认目标方向。",
						},
						"scene": map[string]any{
							"type":        "string",
							"description": "可选，推荐场景，如 dashboard/detail/after_wrong_answer。",
						},
						"knowledgeId": map[string]any{
							"type":        "string",
							"description": "可选，当前知识点 ID，用于详情页个性化推荐。",
						},
						"limit": map[string]any{
							"type":        "integer",
							"description": "返回条数，1 到 10。",
						},
					},
					"additionalProperties": false,
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "get_knowledge_detail",
				"description": "读取指定知识点详情，并结合当前用户的掌握状态返回该知识点与用户的关系。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"knowledgeId": map[string]any{
							"type":        "string",
							"description": "知识点 ID 或 routeKnowledgeId。",
						},
					},
					"required":             []string{"knowledgeId"},
					"additionalProperties": false,
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "get_knowledge_graph",
				"description": "读取指定知识点的一跳知识图谱子图，用于分析关联知识、先修和扩展。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"root": map[string]any{
							"type":        "string",
							"description": "图谱根知识点。",
						},
						"limit": map[string]any{
							"type":        "integer",
							"description": "节点查询数量，1 到 20。",
						},
					},
					"required":             []string{"root"},
					"additionalProperties": false,
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "get_frontier_briefing",
				"description": "读取指定知识点的前沿摘要、关联主题和代表论文。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"knowledgeId": map[string]any{
							"type":        "string",
							"description": "知识点 ID。",
						},
						"knowledge": map[string]any{
							"type":        "string",
							"description": "知识点名称。",
						},
					},
					"additionalProperties": false,
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "query_learning_database",
				"description": "直接从学习数据库读取当前用户的最近答题、错题、薄弱知识点或答题统计。返回数据来源为 MySQL。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"queryType": map[string]any{
							"type": "string",
							"enum": []string{"recent_answers", "recent_wrong_answers", "weak_knowledge", "strong_knowledge", "answer_stats"},
						},
						"knowledgeId": map[string]any{
							"type":        "string",
							"description": "可选，按知识点过滤。",
						},
						"limit": map[string]any{
							"type":        "integer",
							"description": "可选，返回条数，1 到 20。",
						},
					},
					"required":             []string{"queryType"},
					"additionalProperties": false,
				},
			},
		},
	}
}

func (s *Service) executeTool(ctx context.Context, currentUser *authdto.AuthUserView, name string, rawArguments string) string {
	payload, err := s.runTool(ctx, currentUser, name, rawArguments)
	if err != nil {
		payload = map[string]any{
			"ok":          false,
			"tool":        name,
			"error":       err.Error(),
			"generatedAt": time.Now().Format(time.RFC3339),
		}
	}

	body, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return `{"ok":false,"error":"tool result marshal failed"}`
	}
	return string(body)
}

func (s *Service) runTool(ctx context.Context, currentUser *authdto.AuthUserView, name string, rawArguments string) (map[string]any, error) {
	switch strings.TrimSpace(name) {
	case "get_current_user_context":
		return s.toolGetCurrentUser(currentUser), nil
	case "get_dashboard_summary":
		return s.toolGetDashboardSummary(ctx, currentUser)
	case "get_competency_profile":
		return s.toolGetCompetencyProfile(ctx, currentUser)
	case "get_learning_progress":
		return s.toolGetLearningProgress(ctx)
	case "get_recommended_path":
		return s.toolGetRecommendedPath(ctx, rawArguments)
	case "get_knowledge_detail":
		return s.toolGetKnowledgeDetail(ctx, rawArguments)
	case "get_knowledge_graph":
		return s.toolGetKnowledgeGraph(ctx, rawArguments)
	case "get_frontier_briefing":
		return s.toolGetFrontierBriefing(ctx, rawArguments)
	case "query_learning_database":
		return s.toolQueryLearningDatabase(ctx, rawArguments)
	default:
		return nil, errors.New("unknown tool")
	}
}

func (s *Service) toolGetCurrentUser(currentUser *authdto.AuthUserView) map[string]any {
	return map[string]any{
		"ok":          true,
		"source":      "auth_service",
		"user":        currentUser,
		"generatedAt": time.Now().Format(time.RFC3339),
	}
}

func (s *Service) toolGetDashboardSummary(ctx context.Context, currentUser *authdto.AuthUserView) (map[string]any, error) {
	if s.dashboardService == nil {
		return nil, errors.New("dashboard service unavailable")
	}
	response, err := s.dashboardService.GetSummary(ctx, currentUserEntity(currentUser))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":              true,
		"source":          "dashboard_service",
		"summary":         response,
		"recommendedPath": response.RecommendedPath,
		"generatedAt":     time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Service) toolGetCompetencyProfile(ctx context.Context, currentUser *authdto.AuthUserView) (map[string]any, error) {
	if s.competencyService == nil {
		return nil, errors.New("competency service unavailable")
	}
	response, err := s.competencyService.GetProfile(ctx, currentUserProfile(currentUser))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"ok":            true,
		"source":        "competency_service",
		"profile":       response,
		"strongestAxes": pickRadarAxes(response.RadarAxes, false, 3),
		"weakestAxes":   pickRadarAxes(response.RadarAxes, true, 3),
		"generatedAt":   time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Service) toolGetLearningProgress(ctx context.Context) (map[string]any, error) {
	if s.learningService == nil {
		return nil, errors.New("learning service unavailable")
	}
	response, degraded, err := s.learningService.GetLearningProgress(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"ok":               true,
		"source":           learningProgressSource(degraded),
		"progress":         response,
		"weakKnowledge":    pickWeakKnowledge(response.List, 5),
		"strongKnowledge":  pickStrongKnowledge(response.List, 5),
		"learningSnapshot": summarizeProgress(response.List),
		"generatedAt":      time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Service) toolGetRecommendedPath(ctx context.Context, rawArguments string) (map[string]any, error) {
	if s.recommendService == nil {
		return nil, errors.New("recommend service unavailable")
	}

	type arguments struct {
		Target      string `json:"target"`
		Scene       string `json:"scene"`
		KnowledgeID string `json:"knowledgeId"`
		Limit       int    `json:"limit"`
	}
	var args arguments
	if err := decodeToolArguments(rawArguments, &args); err != nil {
		return nil, err
	}

	response, err := s.recommendService.GetPath(ctx, recommenddto.PathQuery{
		Target:      strings.TrimSpace(args.Target),
		KnowledgeID: strings.TrimSpace(args.KnowledgeID),
		Scene:       strings.TrimSpace(args.Scene),
		Limit:       clampToolLimit(args.Limit, 5, 10),
	})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"ok":          true,
		"source":      "recommend_service",
		"path":        response.Path,
		"generatedAt": time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Service) toolGetKnowledgeDetail(ctx context.Context, rawArguments string) (map[string]any, error) {
	if s.graphService == nil {
		return nil, errors.New("graph service unavailable")
	}

	type arguments struct {
		KnowledgeID string `json:"knowledgeId"`
	}
	var args arguments
	if err := decodeToolArguments(rawArguments, &args); err != nil {
		return nil, err
	}

	knowledgeID := strings.TrimSpace(args.KnowledgeID)
	response, err := s.graphService.GetKnowledgeDetail(ctx, knowledgeID)
	if err != nil {
		return nil, err
	}

	studentState, stateErr := s.lookupKnowledgeProgress(ctx, knowledgeID)
	if stateErr != nil {
		studentState = map[string]any{
			"knowledgeId": knowledgeID,
			"status":      "unknown",
			"message":     stateErr.Error(),
		}
	}

	return map[string]any{
		"ok":           true,
		"source":       "neo4j+learning_state",
		"detail":       response,
		"studentState": studentState,
		"generatedAt":  time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Service) toolGetKnowledgeGraph(ctx context.Context, rawArguments string) (map[string]any, error) {
	if s.graphService == nil {
		return nil, errors.New("graph service unavailable")
	}

	type arguments struct {
		Root  string `json:"root"`
		Limit int    `json:"limit"`
	}
	var args arguments
	if err := decodeToolArguments(rawArguments, &args); err != nil {
		return nil, err
	}

	response, err := s.graphService.GetSubgraph(ctx, graphdto.SubgraphQuery{
		Root:  strings.TrimSpace(args.Root),
		Limit: clampToolLimit(args.Limit, 12, 20),
	})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"ok":          true,
		"source":      "neo4j",
		"nodeCount":   len(response.Nodes),
		"edgeCount":   len(response.Edges),
		"nodes":       response.Nodes,
		"edges":       response.Edges,
		"generatedAt": time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Service) toolGetFrontierBriefing(ctx context.Context, rawArguments string) (map[string]any, error) {
	if s.frontierService == nil {
		return nil, errors.New("frontier service unavailable")
	}

	type arguments struct {
		KnowledgeID string `json:"knowledgeId"`
		Knowledge   string `json:"knowledge"`
	}
	var args arguments
	if err := decodeToolArguments(rawArguments, &args); err != nil {
		return nil, err
	}

	response, err := s.frontierService.GetBriefing(ctx, frontierdto.BriefingQuery{
		KnowledgeID: strings.TrimSpace(args.KnowledgeID),
		Knowledge:   strings.TrimSpace(args.Knowledge),
	})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"ok":          true,
		"source":      "frontier_service",
		"briefing":    response,
		"generatedAt": time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Service) toolQueryLearningDatabase(ctx context.Context, rawArguments string) (map[string]any, error) {
	if s.db == nil {
		return nil, errors.New("database unavailable")
	}

	type arguments struct {
		QueryType   string `json:"queryType"`
		KnowledgeID string `json:"knowledgeId"`
		Limit       int    `json:"limit"`
	}
	var args arguments
	if err := decodeToolArguments(rawArguments, &args); err != nil {
		return nil, err
	}

	studentID, ok := ctxmeta.CurrentUserID(ctx)
	if !ok {
		return nil, errors.New("current user not found")
	}

	limit := clampToolLimit(args.Limit, 5, 20)
	knowledgeID := strings.TrimSpace(args.KnowledgeID)
	switch strings.TrimSpace(args.QueryType) {
	case "recent_answers":
		list, err := s.queryRecentAnswers(ctx, studentID, knowledgeID, limit, false)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"ok":          true,
			"source":      "mysql",
			"queryType":   "recent_answers",
			"list":        list,
			"generatedAt": time.Now().Format(time.RFC3339),
		}, nil
	case "recent_wrong_answers":
		list, err := s.queryRecentAnswers(ctx, studentID, knowledgeID, limit, true)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"ok":          true,
			"source":      "mysql",
			"queryType":   "recent_wrong_answers",
			"list":        list,
			"generatedAt": time.Now().Format(time.RFC3339),
		}, nil
	case "weak_knowledge":
		list, err := s.queryKnowledgeRanking(ctx, studentID, limit, true)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"ok":          true,
			"source":      "mysql",
			"queryType":   "weak_knowledge",
			"list":        list,
			"generatedAt": time.Now().Format(time.RFC3339),
		}, nil
	case "strong_knowledge":
		list, err := s.queryKnowledgeRanking(ctx, studentID, limit, false)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"ok":          true,
			"source":      "mysql",
			"queryType":   "strong_knowledge",
			"list":        list,
			"generatedAt": time.Now().Format(time.RFC3339),
		}, nil
	case "answer_stats":
		stats, err := s.queryAnswerStats(ctx, studentID, knowledgeID)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"ok":          true,
			"source":      "mysql",
			"queryType":   "answer_stats",
			"stats":       stats,
			"generatedAt": time.Now().Format(time.RFC3339),
		}, nil
	default:
		return nil, errors.New("unsupported queryType")
	}
}

func decodeToolArguments(raw string, target any) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), target)
}

func clampToolLimit(input int, fallback int, max int) int {
	if fallback <= 0 {
		fallback = 5
	}
	if max <= 0 {
		max = 20
	}
	if input <= 0 {
		return fallback
	}
	if input > max {
		return max
	}
	return input
}

func learningProgressSource(degraded bool) string {
	if degraded {
		return "redis+mysql_fallback"
	}
	return "redis"
}

func pickWeakKnowledge(items []learningdto.LearningProgressItem, limit int) []learningdto.LearningProgressItem {
	sorted := append([]learningdto.LearningProgressItem{}, items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].MasteryScore == sorted[j].MasteryScore {
			return sorted[i].AnsweredCount > sorted[j].AnsweredCount
		}
		return sorted[i].MasteryScore < sorted[j].MasteryScore
	})
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

func pickStrongKnowledge(items []learningdto.LearningProgressItem, limit int) []learningdto.LearningProgressItem {
	sorted := append([]learningdto.LearningProgressItem{}, items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].MasteryScore == sorted[j].MasteryScore {
			return sorted[i].AnsweredCount > sorted[j].AnsweredCount
		}
		return sorted[i].MasteryScore > sorted[j].MasteryScore
	})
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

func summarizeProgress(items []learningdto.LearningProgressItem) map[string]any {
	weakCount := 0
	strongCount := 0
	for _, item := range items {
		if item.MasteryScore < 60 {
			weakCount++
		}
		if item.MasteryScore >= 80 {
			strongCount++
		}
	}

	return map[string]any{
		"totalKnowledge":       len(items),
		"weakKnowledgeCount":   weakCount,
		"strongKnowledgeCount": strongCount,
	}
}

func pickRadarAxes(items []competencydto.RadarAxis, weakest bool, limit int) []competencydto.RadarAxis {
	sorted := append([]competencydto.RadarAxis{}, items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Value == sorted[j].Value {
			return sorted[i].Name < sorted[j].Name
		}
		if weakest {
			return sorted[i].Value < sorted[j].Value
		}
		return sorted[i].Value > sorted[j].Value
	})
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

func (s *Service) lookupKnowledgeProgress(ctx context.Context, knowledgeID string) (map[string]any, error) {
	if s.learningService == nil {
		return nil, errors.New("learning service unavailable")
	}
	response, degraded, err := s.learningService.GetLearningProgress(ctx)
	if err != nil {
		return nil, err
	}
	normalized := strings.TrimSpace(knowledgeID)
	for _, item := range response.List {
		if strings.TrimSpace(item.KnowledgeID) == normalized {
			return map[string]any{
				"source":         learningProgressSource(degraded),
				"knowledgeId":    item.KnowledgeID,
				"canonicalName":  item.CanonicalName,
				"masteryScore":   item.MasteryScore,
				"correctRate":    item.CorrectRate,
				"answeredCount":  item.AnsweredCount,
				"lastAnsweredAt": item.LastAnsweredAt,
				"status":         item.Status,
			}, nil
		}
	}

	return map[string]any{
		"source":       learningProgressSource(degraded),
		"knowledgeId":  normalized,
		"status":       "not_started",
		"masteryScore": 0,
		"message":      "当前没有查到这个知识点的学习记录",
	}, nil
}
