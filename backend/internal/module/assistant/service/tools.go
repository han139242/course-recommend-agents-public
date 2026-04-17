package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	assistantdto "github.com/han139242/course-recommend-agents/backend/internal/module/assistant/dto"
	authdto "github.com/han139242/course-recommend-agents/backend/internal/module/auth/dto"
	competencydto "github.com/han139242/course-recommend-agents/backend/internal/module/competency/dto"
	frontierdto "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/dto"
	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
)

func buildAgentTools(intent agentIntent, isPublic bool) []map[string]any {
	registry := buildToolRegistry()
	toolNames := buildAgentToolNames(intent, isPublic)
	result := make([]map[string]any, 0, len(toolNames))
	for _, name := range toolNames {
		definition, ok := registry[name]
		if !ok {
			continue
		}
		result = append(result, definition)
	}
	return result
}

func buildToolRegistry() map[string]map[string]any {
	return map[string]map[string]any{
		"get_current_user_context": buildFunctionTool("get_current_user_context", "读取当前已登录用户的基本身份信息、目标方向和当前会话上下文。", emptyToolParameters()),
		"get_dashboard_summary":    buildFunctionTool("get_dashboard_summary", "读取当前用户首页摘要、推荐卡片和推荐路径结果。", emptyToolParameters()),
		"get_competency_profile":   buildFunctionTool("get_competency_profile", "读取当前用户能力画像、雷达维度、薄弱能力和成长洞察。", emptyToolParameters()),
		"get_learning_progress":    buildFunctionTool("get_learning_progress", "读取当前用户知识点掌握度、正确率、答题次数和最近答题时间。", emptyToolParameters()),
		"get_recommended_path": buildFunctionTool("get_recommended_path", "读取当前用户的下一步学习推荐路径。", map[string]any{
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
		}),
		"get_knowledge_detail": buildFunctionTool("get_knowledge_detail", "读取指定知识点详情，并结合当前用户的掌握状态返回该知识点与用户的关系。", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"knowledgeId": map[string]any{
					"type":        "string",
					"description": "知识点 ID 或 routeKnowledgeId。",
				},
			},
			"required":             []string{"knowledgeId"},
			"additionalProperties": false,
		}),
		"get_knowledge_graph": buildFunctionTool("get_knowledge_graph", "读取指定知识点的一跳知识图谱子图，用于分析关联知识、先修和扩展。", map[string]any{
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
		}),
		"get_frontier_briefing": buildFunctionTool("get_frontier_briefing", "读取指定知识点的前沿摘要、关联主题和代表论文。", map[string]any{
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
		}),
		"query_learning_database": buildFunctionTool("query_learning_database", "直接从学习数据库读取当前用户的最近答题、错题、薄弱知识点或答题统计。返回数据来源为 MySQL。", map[string]any{
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
		}),
		"reason_knowledge_path": buildFunctionTool("reason_knowledge_path", "分析两个知识点之间的图谱路径关系，找到最短连接路径。", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"fromKnowledge": map[string]any{
					"type":        "string",
					"description": "起始知识点名称或 ID。",
				},
				"toKnowledge": map[string]any{
					"type":        "string",
					"description": "目标知识点名称或 ID。",
				},
			},
			"required":             []string{"fromKnowledge", "toKnowledge"},
			"additionalProperties": false,
		}),
		"check_student_prior_knowledge": buildFunctionTool("check_student_prior_knowledge", "查看指定知识点前置链上学生已经掌握的知识，帮助构造类比起点。", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"knowledgeId": map[string]any{
					"type":        "string",
					"description": "目标知识点 ID、routeKnowledgeId 或名称。",
				},
			},
			"required":             []string{"knowledgeId"},
			"additionalProperties": false,
		}),
		"focus_graph_knowledge": buildFunctionTool("focus_graph_knowledge", "让前端图谱聚焦到指定知识点，并在需要时轻微闪烁提示。", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"target": map[string]any{
					"type":        "string",
					"description": "知识点 routeKnowledgeId、名称或 canonicalName。",
				},
				"flash": map[string]any{
					"type":        "boolean",
					"description": "是否闪烁提示，默认 true。",
				},
			},
			"required":             []string{"target"},
			"additionalProperties": false,
		}),
		"expand_graph_knowledge": buildFunctionTool("expand_graph_knowledge", "让前端图谱定位并展开指定知识点的一跳邻居。", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"target": map[string]any{
					"type":        "string",
					"description": "知识点 routeKnowledgeId、名称或 canonicalName。",
				},
			},
			"required":             []string{"target"},
			"additionalProperties": false,
		}),
		"highlight_prerequisite_chain": buildFunctionTool("highlight_prerequisite_chain", "让前端高亮指定知识点的前置知识链。", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"target": map[string]any{
					"type":        "string",
					"description": "知识点 routeKnowledgeId、名称或 canonicalName。",
				},
			},
			"required":             []string{"target"},
			"additionalProperties": false,
		}),
		"open_knowledge_detail_page": buildFunctionTool("open_knowledge_detail_page", "让前端打开指定知识点的详情页。", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"target": map[string]any{
					"type":        "string",
					"description": "知识点 routeKnowledgeId、名称或 canonicalName。",
				},
			},
			"required":             []string{"target"},
			"additionalProperties": false,
		}),
		"filter_graph_course": buildFunctionTool("filter_graph_course", "让前端按课程过滤图谱，并尽量定位到该课程代表知识点。", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"courseTitle": map[string]any{
					"type":        "string",
					"description": "课程或章节标题。",
				},
			},
			"required":             []string{"courseTitle"},
			"additionalProperties": false,
		}),
		"fit_graph_view":    buildFunctionTool("fit_graph_view", "让前端将当前图谱适配到可视区域。", emptyToolParameters()),
		"reset_graph_scene": buildFunctionTool("reset_graph_scene", "让前端重置图谱场景、清空高亮并恢复总览。", emptyToolParameters()),
	}
}

func buildFunctionTool(name string, description string, parameters map[string]any) map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        name,
			"description": description,
			"parameters":  parameters,
		},
	}
}

func emptyToolParameters() map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
}

func buildAgentToolNames(intent agentIntent, isPublic bool) []string {
	publicGraphTools := []string{
		"get_knowledge_detail",
		"get_knowledge_graph",
		"get_frontier_briefing",
		"reason_knowledge_path",
	}
	authCoreTools := []string{
		"get_current_user_context",
		"get_learning_progress",
		"get_competency_profile",
		"get_dashboard_summary",
		"get_recommended_path",
	}
	uiTools := []string{
		"focus_graph_knowledge",
		"expand_graph_knowledge",
		"highlight_prerequisite_chain",
		"open_knowledge_detail_page",
		"filter_graph_course",
		"fit_graph_view",
		"reset_graph_scene",
	}

	var tools []string
	switch intent {
	case agentIntentPlanner:
		if isPublic {
			tools = append(tools, publicGraphTools...)
		} else {
			tools = append(tools, "get_current_user_context", "get_learning_progress", "get_competency_profile", "get_recommended_path", "get_knowledge_detail")
		}
	case agentIntentAssessor:
		if isPublic {
			tools = append(tools, publicGraphTools...)
		} else {
			tools = append(tools, "get_current_user_context", "get_learning_progress", "get_competency_profile", "query_learning_database", "get_dashboard_summary")
		}
	default:
		if isPublic {
			tools = append(tools, publicGraphTools...)
		} else {
			tools = append(tools, "get_current_user_context", "get_knowledge_detail", "get_knowledge_graph", "reason_knowledge_path", "get_frontier_briefing", "check_student_prior_knowledge")
		}
	}

	if !isPublic && intent != agentIntentAssessor {
		tools = append(tools, authCoreTools[:1]...)
	}
	tools = append(tools, uiTools...)
	return dedupeToolNames(tools)
}

func dedupeToolNames(items []string) []string {
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}

func (s *Service) executeTool(ctx context.Context, currentUser *authdto.AuthUserView, name string, rawArguments string) (string, []assistantdto.ClientAction) {
	payload, actions, err := s.runTool(ctx, currentUser, name, rawArguments)
	if err != nil {
		payload = map[string]any{
			"ok":          false,
			"tool":        name,
			"error":       err.Error(),
			"generatedAt": time.Now().Format(time.RFC3339),
		}
		actions = nil
	}

	body, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return `{"ok":false,"error":"tool result marshal failed"}`, nil
	}
	return string(body), actions
}

func (s *Service) runTool(ctx context.Context, currentUser *authdto.AuthUserView, name string, rawArguments string) (map[string]any, []assistantdto.ClientAction, error) {
	switch strings.TrimSpace(name) {
	case "get_current_user_context":
		return s.toolGetCurrentUser(currentUser), nil, nil
	case "get_dashboard_summary":
		payload, err := s.toolGetDashboardSummary(ctx, currentUser)
		return payload, nil, err
	case "get_competency_profile":
		payload, err := s.toolGetCompetencyProfile(ctx, currentUser)
		return payload, nil, err
	case "get_learning_progress":
		payload, err := s.toolGetLearningProgress(ctx)
		return payload, nil, err
	case "get_recommended_path":
		payload, err := s.toolGetRecommendedPath(ctx, rawArguments)
		return payload, nil, err
	case "get_knowledge_detail":
		payload, err := s.toolGetKnowledgeDetail(ctx, rawArguments)
		return payload, nil, err
	case "get_knowledge_graph":
		payload, err := s.toolGetKnowledgeGraph(ctx, rawArguments)
		return payload, nil, err
	case "get_frontier_briefing":
		payload, err := s.toolGetFrontierBriefing(ctx, rawArguments)
		return payload, nil, err
	case "reason_knowledge_path":
		payload, err := s.toolReasonKnowledgePath(ctx, rawArguments)
		return payload, nil, err
	case "check_student_prior_knowledge":
		payload, err := s.toolCheckStudentPriorKnowledge(ctx, rawArguments)
		return payload, nil, err
	case "query_learning_database":
		payload, err := s.toolQueryLearningDatabase(ctx, rawArguments)
		return payload, nil, err
	case "focus_graph_knowledge":
		return toolQueueKnowledgeAction("focus_knowledge", rawArguments, true)
	case "expand_graph_knowledge":
		return toolQueueKnowledgeAction("expand_knowledge", rawArguments, true)
	case "highlight_prerequisite_chain":
		return toolQueueKnowledgeAction("highlight_prerequisite_chain", rawArguments, true)
	case "open_knowledge_detail_page":
		return toolQueueKnowledgeAction("open_knowledge_detail", rawArguments, false)
	case "filter_graph_course":
		return toolQueueCourseFilterAction(rawArguments)
	case "fit_graph_view":
		return toolQueueSimpleAction("fit_graph_view")
	case "reset_graph_scene":
		return toolQueueSimpleAction("reset_graph_scene")
	default:
		return nil, nil, errors.New("unknown tool")
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

func (s *Service) toolReasonKnowledgePath(ctx context.Context, rawArguments string) (map[string]any, error) {
	if s.graphService == nil {
		return nil, errors.New("graph service unavailable")
	}

	type arguments struct {
		FromKnowledge    string `json:"fromKnowledge"`
		ToKnowledge      string `json:"toKnowledge"`
		FromKnowledgeAlt string `json:"from_knowledge"`
		ToKnowledgeAlt   string `json:"to_knowledge"`
	}
	var args arguments
	if err := decodeToolArguments(rawArguments, &args); err != nil {
		return nil, err
	}

	fromKnowledge := firstNonEmptyString(strings.TrimSpace(args.FromKnowledge), strings.TrimSpace(args.FromKnowledgeAlt))
	toKnowledge := firstNonEmptyString(strings.TrimSpace(args.ToKnowledge), strings.TrimSpace(args.ToKnowledgeAlt))
	if fromKnowledge == "" || toKnowledge == "" {
		return nil, errors.New("fromKnowledge and toKnowledge are required")
	}

	response, err := s.graphService.FindPath(ctx, graphdto.GraphPathQuery{
		From: fromKnowledge,
		To:   toKnowledge,
	})
	if err != nil {
		return nil, err
	}

	nodeLabels := mapGraphNodeLabels(response.Nodes)
	orderedLabels := make([]string, 0, len(response.OrderedNodeIDs))
	for _, orderedNodeID := range response.OrderedNodeIDs {
		label := strings.TrimSpace(nodeLabels[orderedNodeID])
		if label == "" {
			label = orderedNodeID
		}
		orderedLabels = append(orderedLabels, label)
	}

	summary := "当前没有找到一条稳定的图谱连接路径。"
	if response.Found && len(orderedLabels) > 0 {
		summary = strings.Join(orderedLabels, " -> ")
	}

	return map[string]any{
		"ok":             true,
		"source":         "neo4j",
		"found":          response.Found,
		"fromKnowledge":  fromKnowledge,
		"toKnowledge":    toKnowledge,
		"orderedNodeIds": response.OrderedNodeIDs,
		"orderedLabels":  orderedLabels,
		"summary":        summary,
		"nodes":          response.Nodes,
		"edges":          response.Edges,
		"generatedAt":    time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Service) toolCheckStudentPriorKnowledge(ctx context.Context, rawArguments string) (map[string]any, error) {
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
	if knowledgeID == "" {
		return nil, errors.New("knowledgeId is required")
	}

	response, err := s.graphService.GetPrerequisiteChain(ctx, graphdto.PrerequisiteChainQuery{
		Target: knowledgeID,
		Depth:  3,
	})
	if err != nil {
		return nil, err
	}

	nodeMap := make(map[string]graphdto.GraphNodeView, len(response.Nodes))
	for _, node := range response.Nodes {
		nodeMap[strings.TrimSpace(node.RouteKnowledgeID)] = node
		nodeMap[strings.TrimSpace(node.ID)] = node
	}

	mastered := make([]map[string]any, 0, len(response.OrderedNodeIDs))
	for index, orderedNodeID := range response.OrderedNodeIDs {
		if index == len(response.OrderedNodeIDs)-1 {
			break
		}
		node, ok := nodeMap[strings.TrimSpace(orderedNodeID)]
		if !ok {
			continue
		}
		progress, progressErr := s.lookupKnowledgeProgress(ctx, orderedNodeID)
		if progressErr != nil {
			continue
		}
		masteryScore, _ := progress["masteryScore"].(float64)
		if masteryScore < 60 {
			continue
		}
		mastered = append(mastered, map[string]any{
			"knowledgeId":   orderedNodeID,
			"name":          firstNonEmptyString(strings.TrimSpace(node.Name), strings.TrimSpace(node.CanonicalName), orderedNodeID),
			"chapterTitle":  strings.TrimSpace(node.ChapterTitle),
			"masteryScore":  masteryScore,
			"correctRate":   progress["correctRate"],
			"answeredCount": progress["answeredCount"],
			"status":        progress["status"],
		})
	}

	var anchor map[string]any
	if len(mastered) > 0 {
		anchor = mastered[len(mastered)-1]
	}

	return map[string]any{
		"ok":                    true,
		"source":                "neo4j+learning_progress",
		"knowledgeId":           knowledgeID,
		"orderedNodeIds":        response.OrderedNodeIDs,
		"masteredPrerequisites": mastered,
		"comparisonAnchor":      anchor,
		"generatedAt":           time.Now().Format(time.RFC3339),
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

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mapGraphNodeLabels(nodes []graphdto.GraphNodeView) map[string]string {
	labels := make(map[string]string, len(nodes)*2)
	for _, node := range nodes {
		label := firstNonEmptyString(strings.TrimSpace(node.Name), strings.TrimSpace(node.CanonicalName), strings.TrimSpace(node.RouteKnowledgeID), strings.TrimSpace(node.ID))
		for _, key := range []string{node.RouteKnowledgeID, node.ID} {
			trimmed := strings.TrimSpace(key)
			if trimmed == "" {
				continue
			}
			labels[trimmed] = label
		}
	}
	return labels
}

func toolQueueKnowledgeAction(actionType string, rawArguments string, defaultFlash bool) (map[string]any, []assistantdto.ClientAction, error) {
	type arguments struct {
		Target string `json:"target"`
		Flash  *bool  `json:"flash"`
	}

	var args arguments
	if err := decodeToolArguments(rawArguments, &args); err != nil {
		return nil, nil, err
	}

	target := strings.TrimSpace(args.Target)
	if target == "" {
		return nil, nil, errors.New("target is required")
	}

	action := assistantdto.ClientAction{
		Type:   actionType,
		Target: target,
		Flash:  defaultFlash,
	}
	if args.Flash != nil {
		action.Flash = *args.Flash
	}

	return map[string]any{
		"ok":          true,
		"source":      "frontend_action_queue",
		"queued":      true,
		"action":      action,
		"generatedAt": time.Now().Format(time.RFC3339),
	}, []assistantdto.ClientAction{action}, nil
}

func toolQueueCourseFilterAction(rawArguments string) (map[string]any, []assistantdto.ClientAction, error) {
	type arguments struct {
		CourseTitle string `json:"courseTitle"`
	}

	var args arguments
	if err := decodeToolArguments(rawArguments, &args); err != nil {
		return nil, nil, err
	}

	courseTitle := strings.TrimSpace(args.CourseTitle)
	if courseTitle == "" {
		return nil, nil, errors.New("courseTitle is required")
	}

	action := assistantdto.ClientAction{
		Type:        "filter_course",
		CourseTitle: courseTitle,
	}
	return map[string]any{
		"ok":          true,
		"source":      "frontend_action_queue",
		"queued":      true,
		"action":      action,
		"generatedAt": time.Now().Format(time.RFC3339),
	}, []assistantdto.ClientAction{action}, nil
}

func toolQueueSimpleAction(actionType string) (map[string]any, []assistantdto.ClientAction, error) {
	action := assistantdto.ClientAction{Type: actionType}
	return map[string]any{
		"ok":          true,
		"source":      "frontend_action_queue",
		"queued":      true,
		"action":      action,
		"generatedAt": time.Now().Format(time.RFC3339),
	}, []assistantdto.ClientAction{action}, nil
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
